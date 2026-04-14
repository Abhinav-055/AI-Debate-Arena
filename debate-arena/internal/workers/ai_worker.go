package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"

	"debate-arena/internal/ai"
	"debate-arena/internal/models"
	"debate-arena/internal/queue"

	"github.com/redis/go-redis/v9"
)

// AIWorker generates turn content using provider-specific clients.
type AIWorker struct {
	store     DebateStore
	redis     *redis.Client
	publisher EventPublisher
	providers map[models.AgentProvider]ai.Provider
	logger    *slog.Logger
}

// NewAIWorker creates an AI worker.
func NewAIWorker(store DebateStore, redisClient *redis.Client, publisher EventPublisher, providers map[models.AgentProvider]ai.Provider, logger *slog.Logger) *AIWorker {
	return &AIWorker{store: store, redis: redisClient, publisher: publisher, providers: providers, logger: logger}
}

// AITaskPayload describes one generation request.
type AITaskPayload struct {
	AgentID string           `json:"agent_id"`
	Round   int              `json:"round"`
	Context []models.Message `json:"context"`
}

// HandleTask consumes ai.generate and emits next turn or judge event.
func (w *AIWorker) HandleTask(ctx context.Context, env queue.Envelope) error {
	if env.RetryCount >= 3 {
		return fmt.Errorf("ai task exceeded retries")
	}

	var payload AITaskPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("decode ai payload: %w", err)
	}

	debate, err := w.store.GetDebateByID(ctx, env.DebateID)
	if err != nil {
		return fmt.Errorf("load debate: %w", err)
	}
	if debate.Status == models.DebateStatusCompleted {
		return nil
	}
	if debate.RoundsTotal > 0 && payload.Round > debate.RoundsTotal {
		w.logger.Info("ignoring ai task beyond configured rounds",
			"debate_id", debate.ID,
			"round", payload.Round,
			"rounds_total", debate.RoundsTotal,
		)
		return nil
	}

	var currentAgent models.Agent
	for _, a := range debate.Agents {
		if a.ID == payload.AgentID {
			currentAgent = a
			break
		}
	}
	if currentAgent.ID == "" {
		return fmt.Errorf("resolve current agent: not found")
	}

	provider, ok := w.providers[currentAgent.Provider]
	if !ok {
		return fmt.Errorf("resolve provider: provider %s not registered", currentAgent.Provider)
	}

	stance := stanceFromPersonality(currentAgent.Personality)
	stanceInstruction := "Argue your assigned side of the topic"
	if stance != "" {
		stanceInstruction = fmt.Sprintf("You must argue %s the topic", stance)
	}
	systemPrompt := fmt.Sprintf("You are %s. Your personality: %s. Debate topic: %s. %s. Argue your position in 150 words or less. Be direct. Do not mention round numbers, round labels, or any reference to which round this is in your response.", currentAgent.Name, currentAgent.Personality, debate.Topic, stanceInstruction)
	agentNames := make(map[string]string, len(debate.Agents))
	for _, a := range debate.Agents {
		agentNames[a.ID] = a.Name
	}
	userPrompt := buildContextPrompt(payload.Context, agentNames)

	text, err := provider.Generate(ctx, ai.GenerateRequest{
		SystemPrompt: systemPrompt,
		UserMessage:  userPrompt,
		MaxTokens:    220,
		Temperature:  temperatureForProvider(currentAgent.Provider),
	})
	if err != nil {
		if env.RetryCount < 2 {
			return fmt.Errorf("generate response: %w", err)
		}
		w.logger.Error("provider generation failed after max retries, using fallback message",
			"debate_id", debate.ID,
			"agent_id", currentAgent.ID,
			"provider", currentAgent.Provider,
			"error", err,
		)
		text = fmt.Sprintf("%s: I could not reach my provider for this turn, so this is a fallback statement to keep the debate moving.", currentAgent.Name)
	}
	text = sanitizeGeneratedResponse(text)

	msg, err := w.store.CreateMessage(ctx, debate.ID, currentAgent.ID, payload.Round, text)
	if err != nil {
		return fmt.Errorf("save generated message: %w", err)
	}

	if err := w.updateMessageCache(ctx, debate.ID, msg); err != nil {
		return fmt.Errorf("update message cache: %w", err)
	}

	msgRaw, _ := json.Marshal(map[string]any{"type": "messageAdded", "data": msg})
	if err := w.redis.Publish(ctx, "debate:"+debate.ID, msgRaw).Err(); err != nil {
		return fmt.Errorf("publish redis messageAdded: %w", err)
	}

	if err := w.advance(ctx, debate, payload); err != nil {
		return fmt.Errorf("advance debate: %w", err)
	}
	return nil
}

func (w *AIWorker) advance(ctx context.Context, debate models.Debate, payload AITaskPayload) error {
	nextAgent := 0
	isLastAgentInRound := false
	for idx, a := range debate.Agents {
		if a.ID == payload.AgentID {
			nextAgent = idx + 1
			if nextAgent >= len(debate.Agents) {
				nextAgent = 0
				isLastAgentInRound = true
			}
			break
		}
	}

	nextRound := payload.Round
	if isLastAgentInRound {
		nextRound = min(payload.Round+1, debate.RoundsTotal)
		if debate.RoundsDone < debate.RoundsTotal {
			if err := w.store.IncrementRoundsDone(ctx, debate.ID); err != nil {
				return fmt.Errorf("increment rounds done: %w", err)
			}
		}
	}

	if payload.Round >= debate.RoundsTotal && isLastAgentInRound {
		if err := w.store.UpdateDebateStatus(ctx, debate.ID, models.DebateStatusCompleted); err != nil {
			return fmt.Errorf("complete debate status: %w", err)
		}
		if err := w.setDebateState(ctx, debate.ID, models.DebateState{
			Status:            models.DebateStatusCompleted,
			CurrentRound:      debate.RoundsTotal,
			CurrentAgentIndex: 0,
		}); err != nil {
			return fmt.Errorf("set completed debate state: %w", err)
		}
		judgePayload, _ := json.Marshal(map[string]any{"debate_id": debate.ID})
		if err := w.publisher.Publish(ctx, queue.RoutingDebateJudgeEvaluate, queue.Envelope{
			EventType:  "judge.evaluate",
			DebateID:   debate.ID,
			Payload:    judgePayload,
			RetryCount: 0,
			Timestamp:  time.Now().UTC(),
		}); err != nil {
			return fmt.Errorf("publish judge event: %w", err)
		}
		statusRaw, _ := json.Marshal(map[string]any{"type": "debateStatusChanged", "data": map[string]any{"id": debate.ID, "status": models.DebateStatusCompleted}})
		if err := w.redis.Publish(ctx, "debate:"+debate.ID, statusRaw).Err(); err != nil {
			return fmt.Errorf("publish redis status change: %w", err)
		}
		return nil
	}

	nextPayload, _ := json.Marshal(TurnStartPayload{Round: nextRound, AgentIndex: nextAgent})
	if err := w.publisher.Publish(ctx, queue.RoutingDebateTurnRequested, queue.Envelope{
		EventType:  "turn.start",
		DebateID:   debate.ID,
		Payload:    nextPayload,
		RetryCount: 0,
		Timestamp:  time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("publish next turn: %w", err)
	}

	if err := w.setDebateState(ctx, debate.ID, models.DebateState{
		Status:            models.DebateStatusActive,
		CurrentRound:      nextRound,
		CurrentAgentIndex: nextAgent,
	}); err != nil {
		return fmt.Errorf("set active debate state: %w", err)
	}

	if err := w.store.UpdateDebateStatus(ctx, debate.ID, models.DebateStatusActive); err != nil {
		return fmt.Errorf("update active status: %w", err)
	}
	return nil
}

func (w *AIWorker) setDebateState(ctx context.Context, debateID string, state models.DebateState) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("marshal debate state: %w", err)
	}
	if err := w.redis.Set(ctx, "debate:state:"+debateID, raw, 24*time.Hour).Err(); err != nil {
		return fmt.Errorf("redis set debate state: %w", err)
	}
	return nil
}

func (w *AIWorker) updateMessageCache(ctx context.Context, debateID string, msg models.Message) error {
	key := "debate:messages:" + debateID
	current, _ := w.redis.Get(ctx, key).Result()
	items := make([]models.Message, 0, 20)
	if current != "" {
		_ = json.Unmarshal([]byte(current), &items)
	}
	items = append(items, msg)
	if len(items) > 20 {
		items = items[len(items)-20:]
	}
	raw, err := json.Marshal(items)
	if err != nil {
		return fmt.Errorf("marshal message cache: %w", err)
	}
	if err := w.redis.Set(ctx, key, raw, time.Hour).Err(); err != nil {
		return fmt.Errorf("set message cache: %w", err)
	}
	return nil
}

func buildContextPrompt(messages []models.Message, agentNames map[string]string) string {
	if len(messages) == 0 {
		return "Open the debate with your strongest argument."
	}
	var b strings.Builder
	b.WriteString("Debate context:\n")
	for _, m := range messages {
		name := agentNames[m.AgentID]
		if name == "" {
			name = "Opponent"
		}
		b.WriteString(fmt.Sprintf("%s: %s\n", name, m.Content))
	}
	b.WriteString("Respond with your next argument. Do not mention rounds.")
	return b.String()
}

var leadingRoundPattern = regexp.MustCompile(`(?i)^\s*(?:\[?\s*round\s*\d+\s*\]?\s*[:\-]?\s*)+`)
var leadingSpeakerPattern = regexp.MustCompile(`^\s*[A-Za-z][A-Za-z0-9_'\- ]{0,40}:\s*`)
var leadingBracketNarrationPattern = regexp.MustCompile(`^\s*\[[^\]\n]{0,140}\]\s*`)
var inlineRoundPattern = regexp.MustCompile(`(?i)\[\s*round\s*\d+\s*\]|\bround\s*\d+\s*:`)

func sanitizeGeneratedResponse(text string) string {
	clean := strings.TrimSpace(text)
	clean = inlineRoundPattern.ReplaceAllString(clean, "")
	for i := 0; i < 4; i++ {
		next := leadingRoundPattern.ReplaceAllString(clean, "")
		next = leadingSpeakerPattern.ReplaceAllString(next, "")
		next = leadingBracketNarrationPattern.ReplaceAllString(next, "")
		next = strings.TrimSpace(next)
		if next == clean {
			break
		}
		clean = next
	}
	return strings.TrimSpace(clean)
}

func temperatureForProvider(provider models.AgentProvider) float32 {
	switch provider {
	case models.AgentProviderGroq:
		return 0.8
	default:
		return 0.8
	}
}

func stanceFromPersonality(personality string) string {
	upper := strings.ToUpper(personality)
	switch {
	case strings.Contains(upper, "STANCE: FOR") || strings.Contains(upper, "[FOR]"):
		return "FOR"
	case strings.Contains(upper, "STANCE: AGAINST") || strings.Contains(upper, "[AGAINST]"):
		return "AGAINST"
	default:
		return ""
	}
}
