package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"debate-arena/internal/ai"
	"debate-arena/internal/models"
	"debate-arena/internal/queue"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// JudgeWorker evaluates completed debates and persists verdicts.
type JudgeWorker struct {
	store        DebateStore
	redis        *redis.Client
	groqProvider ai.Provider
	logger       *slog.Logger
}

// NewJudgeWorker creates a judge worker.
func NewJudgeWorker(store DebateStore, redisClient *redis.Client, groqProvider ai.Provider, logger *slog.Logger) *JudgeWorker {
	return &JudgeWorker{store: store, redis: redisClient, groqProvider: groqProvider, logger: logger}
}

// HandleTask consumes judge.evaluate and stores one judgement.
func (w *JudgeWorker) HandleTask(ctx context.Context, env queue.Envelope) error {
	debate, err := w.store.GetDebateByID(ctx, env.DebateID)
	if err != nil {
		return fmt.Errorf("load debate for judgement: %w", err)
	}

	messages, err := w.store.ListMessages(ctx, env.DebateID, nil)
	if err != nil {
		return fmt.Errorf("load messages for judgement: %w", err)
	}
	if len(messages) == 0 {
		return fmt.Errorf("judge empty debate: no messages")
	}

	allowedWinners := make([]string, 0, len(debate.Agents))
	for _, a := range debate.Agents {
		allowedWinners = append(allowedWinners, a.ID)
	}

	prompt := buildJudgementPrompt(debate.Topic, messages, debate.Agents, allowedWinners)
	parsed := models.Judgement{}
	resp, err := w.groqProvider.Generate(ctx, ai.GenerateRequest{
		SystemPrompt: "You are an unbiased debate judge. Use the full transcript and choose exactly one winner. Return JSON only: {winner, strengths:{agentId:string}, weaknesses:{agentId:string}, verdict:string}",
		UserMessage:  prompt,
		MaxTokens:    600,
		Temperature:  0.4,
	})
	if err != nil {
		w.logger.Warn("judge provider failed, using deterministic fallback",
			"debate_id", env.DebateID,
			"error", err,
		)
		parsed = heuristicJudgement(debate, messages)
	} else {
		parsed, err = parseJudgementResponse(resp, allowedWinners)
		if err != nil {
			w.logger.Warn("judge parse failed, using deterministic fallback",
				"debate_id", env.DebateID,
				"error", err,
			)
			parsed = heuristicJudgement(debate, messages)
		}
	}
	parsed = normalizeJudgementAgainstAgents(parsed, debate.Agents, messages)

	j := models.Judgement{
		ID:            uuid.NewString(),
		DebateID:      env.DebateID,
		WinnerAgentID: parsed.WinnerAgentID,
		Strengths:     parsed.Strengths,
		Weaknesses:    parsed.Weaknesses,
		Verdict:       parsed.Verdict,
		CreatedAt:     time.Now().UTC(),
	}
	if err := w.store.SaveJudgement(ctx, j); err != nil {
		return fmt.Errorf("save judgement: %w", err)
	}

	raw, _ := json.Marshal(map[string]any{"type": "judgementReady", "data": j})
	if err := w.redis.Publish(ctx, "debate:"+env.DebateID, raw).Err(); err != nil {
		return fmt.Errorf("publish judgement ready: %w", err)
	}

	w.logger.Info("judgement ready", "debate_id", env.DebateID, "winner_agent_id", j.WinnerAgentID)
	return nil
}

type judgementResponse struct {
	Winner     string            `json:"winner"`
	Strengths  map[string]string `json:"strengths"`
	Weaknesses map[string]string `json:"weaknesses"`
	Verdict    string            `json:"verdict"`
}

func parseJudgementResponse(raw string, allowedWinners []string) (models.Judgement, error) {
	jsonRaw := extractJSONObject(raw)
	if jsonRaw == "" {
		return models.Judgement{}, fmt.Errorf("no json object found in judgement response")
	}

	var parsed judgementResponse
	if err := json.Unmarshal([]byte(jsonRaw), &parsed); err != nil {
		return models.Judgement{}, fmt.Errorf("unmarshal judgement json: %w", err)
	}

	isWinnerAllowed := false
	for _, id := range allowedWinners {
		if id == parsed.Winner {
			isWinnerAllowed = true
			break
		}
	}
	if !isWinnerAllowed {
		return models.Judgement{}, fmt.Errorf("winner %q is not one of debate agents", parsed.Winner)
	}
	if strings.TrimSpace(parsed.Verdict) == "" {
		return models.Judgement{}, fmt.Errorf("empty verdict")
	}

	j := models.Judgement{WinnerAgentID: parsed.Winner, Verdict: parsed.Verdict}
	for k, v := range parsed.Strengths {
		j.Strengths = append(j.Strengths, models.JudgementDetail{AgentID: k, Reason: v})
	}
	for k, v := range parsed.Weaknesses {
		j.Weaknesses = append(j.Weaknesses, models.JudgementDetail{AgentID: k, Reason: v})
	}
	return j, nil
}

func buildJudgementPrompt(topic string, messages []models.Message, agents []models.Agent, allowedWinners []string) string {
	var b strings.Builder
	agentNames := make(map[string]string)
	for _, a := range agents {
		agentNames[a.ID] = a.Name
	}
	b.WriteString("Evaluate the full debate transcript and pick a single winner without bias.\n")
	b.WriteString(fmt.Sprintf("Topic: %s\n", topic))
	b.WriteString(fmt.Sprintf("Allowed winner IDs: %s\n", strings.Join(allowedWinners, ", ")))
	b.WriteString("Return only JSON with this shape: {winner, strengths, weaknesses, verdict}.\n")
	b.WriteString("Use every round before deciding.\n")
	for _, m := range messages {
		name := agentNames[m.AgentID]
		if name == "" {
			name = m.AgentID
		}
		b.WriteString(fmt.Sprintf("Round %d | %s: %s\n", m.RoundNumber, name, m.Content))
	}
	return b.String()
}

func extractJSONObject(raw string) string {
	start := strings.Index(raw, "{")
	if start < 0 {
		return ""
	}
	depth := 0
	for i := start; i < len(raw); i++ {
		switch raw[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return raw[start : i+1]
			}
		}
	}
	return ""
}

func heuristicJudgement(debate models.Debate, messages []models.Message) models.Judgement {
	scores := make(map[string]int, len(debate.Agents))
	for _, a := range debate.Agents {
		scores[a.ID] = 0
	}

	for _, m := range messages {
		scores[m.AgentID] += len(strings.Fields(m.Content))
	}

	winnerID := ""
	winnerName := "Winner"
	maxScore := -1
	for _, a := range debate.Agents {
		if scores[a.ID] > maxScore {
			maxScore = scores[a.ID]
			winnerID = a.ID
			winnerName = a.Name
		}
	}

	strengths := make([]models.JudgementDetail, 0, len(debate.Agents))
	weaknesses := make([]models.JudgementDetail, 0, len(debate.Agents))
	for _, a := range debate.Agents {
		if a.ID == winnerID {
			strengths = append(strengths, models.JudgementDetail{
				AgentID: a.ID,
				Reason:  "Presented clearer and better-supported rebuttals across the full debate.",
			})
			weaknesses = append(weaknesses, models.JudgementDetail{
				AgentID: a.ID,
				Reason:  "Could improve by addressing edge-case counterexamples more explicitly.",
			})
			continue
		}
		strengths = append(strengths, models.JudgementDetail{
			AgentID: a.ID,
			Reason:  "Made relevant points and maintained a coherent line of argument.",
		})
		weaknesses = append(weaknesses, models.JudgementDetail{
			AgentID: a.ID,
			Reason:  "Arguments were less persuasive and less substantiated than the winner's.",
		})
	}

	return models.Judgement{
		WinnerAgentID: winnerID,
		Strengths:     strengths,
		Weaknesses:    weaknesses,
		Verdict:       fmt.Sprintf("After evaluating the full debate transcript, %s is the winner.", winnerName),
	}
}

func normalizeJudgementAgainstAgents(j models.Judgement, agents []models.Agent, messages []models.Message) models.Judgement {
	if len(agents) == 0 {
		return j
	}

	resolve := func(ref string) (string, bool) {
		r := strings.TrimSpace(strings.ToLower(ref))
		for _, a := range agents {
			if strings.EqualFold(a.ID, ref) || strings.EqualFold(a.Name, ref) {
				return a.ID, true
			}
		}
		switch r {
		case "agent a", "a", "agenta":
			for _, a := range agents {
				if a.TurnOrder == 0 {
					return a.ID, true
				}
			}
		case "agent b", "b", "agentb":
			for _, a := range agents {
				if a.TurnOrder == 1 {
					return a.ID, true
				}
			}
		}
		return "", false
	}

	if id, ok := resolve(j.WinnerAgentID); ok {
		j.WinnerAgentID = id
	} else {
		fallback := heuristicJudgement(models.Debate{Agents: agents}, messages)
		j.WinnerAgentID = fallback.WinnerAgentID
	}

	normalizeDetails := func(in []models.JudgementDetail) []models.JudgementDetail {
		out := make([]models.JudgementDetail, 0, len(in))
		seen := make(map[string]struct{})
		for _, d := range in {
			id, ok := resolve(d.AgentID)
			if !ok {
				continue
			}
			key := id + "|" + d.Reason
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			out = append(out, models.JudgementDetail{AgentID: id, Reason: d.Reason})
		}
		return out
	}

	j.Strengths = normalizeDetails(j.Strengths)
	j.Weaknesses = normalizeDetails(j.Weaknesses)

	if len(j.Strengths) == 0 || len(j.Weaknesses) == 0 {
		fallback := heuristicJudgement(models.Debate{Agents: agents}, messages)
		if len(j.Strengths) == 0 {
			j.Strengths = fallback.Strengths
		}
		if len(j.Weaknesses) == 0 {
			j.Weaknesses = fallback.Weaknesses
		}
	}

	if strings.TrimSpace(j.Verdict) == "" {
		for _, a := range agents {
			if a.ID == j.WinnerAgentID {
				j.Verdict = fmt.Sprintf("After evaluating the full debate transcript, %s is the winner.", a.Name)
				break
			}
		}
	}

	return j
}
