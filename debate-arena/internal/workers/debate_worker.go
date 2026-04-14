package workers

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"debate-arena/internal/models"
	"debate-arena/internal/queue"

	"github.com/redis/go-redis/v9"
)

// DebateWorker orchestrates turn sequencing and AI task dispatch.
type DebateWorker struct {
	store     DebateStore
	redis     *redis.Client
	publisher EventPublisher
	logger    *slog.Logger
}

// NewDebateWorker creates a debate worker.
func NewDebateWorker(store DebateStore, redisClient *redis.Client, publisher EventPublisher, logger *slog.Logger) *DebateWorker {
	return &DebateWorker{store: store, redis: redisClient, publisher: publisher, logger: logger}
}

// TurnStartPayload carries turn progression fields.
type TurnStartPayload struct {
	Round      int `json:"round"`
	AgentIndex int `json:"agent_index"`
}

// HandleTurn consumes turn events and emits AI generation tasks.
func (w *DebateWorker) HandleTurn(ctx context.Context, env queue.Envelope) error {
	if env.RetryCount >= 3 {
		return fmt.Errorf("turn event exceeded retries")
	}

	var payload TurnStartPayload
	if err := json.Unmarshal(env.Payload, &payload); err != nil {
		return fmt.Errorf("decode turn payload: %w", err)
	}

	debate, err := w.store.GetDebateByID(ctx, env.DebateID)
	if err != nil {
		return fmt.Errorf("load debate: %w", err)
	}
	if len(debate.Agents) == 0 {
		return fmt.Errorf("load debate agents: no agents configured")
	}
	if debate.Status == models.DebateStatusCompleted {
		return nil
	}
	if debate.RoundsTotal > 0 && payload.Round > debate.RoundsTotal {
		w.logger.Info("ignoring turn beyond configured rounds",
			"debate_id", debate.ID,
			"round", payload.Round,
			"rounds_total", debate.RoundsTotal,
		)
		return nil
	}

	state, err := w.loadState(ctx, debate, payload)
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	agent := debate.Agents[state.CurrentAgentIndex]
	contextMessages, err := w.store.ListRecentMessages(ctx, debate.ID, 4)
	if err != nil {
		return fmt.Errorf("load recent messages: %w", err)
	}

	task := map[string]any{
		"agent_id": agent.ID,
		"round":    state.CurrentRound,
		"context":  contextMessages,
	}
	taskRaw, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal ai task payload: %w", err)
	}

	if err := w.publisher.Publish(ctx, queue.RoutingDebateAIGenerate, queue.Envelope{
		EventType:  "ai.generate",
		DebateID:   debate.ID,
		Payload:    taskRaw,
		RetryCount: 0,
		Timestamp:  time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("publish ai task: %w", err)
	}

	return nil
}

func (w *DebateWorker) loadState(ctx context.Context, debate models.Debate, payload TurnStartPayload) (models.DebateState, error) {
	cacheKey := "debate:state:" + debate.ID
	state := models.DebateState{}

	// Turn event payload is authoritative for sequencing.
	if payload.Round > 0 {
		state = models.DebateState{
			Status:            debate.Status,
			CurrentRound:      payload.Round,
			CurrentAgentIndex: payload.AgentIndex,
		}
	} else {
		cached, err := w.redis.Get(ctx, cacheKey).Result()
		if err == nil {
			if uErr := json.Unmarshal([]byte(cached), &state); uErr != nil {
				state = models.DebateState{}
			}
		}
		if state.CurrentRound <= 0 {
			state.CurrentRound = max(1, debate.RoundsDone+1)
		}
	}

	if len(debate.Agents) == 0 {
		return models.DebateState{}, fmt.Errorf("invalid debate state: no agents")
	}
	if state.CurrentAgentIndex < 0 || state.CurrentAgentIndex >= len(debate.Agents) {
		state.CurrentAgentIndex = 0
	}
	if state.Status == "" {
		state.Status = debate.Status
	}

	raw, err := json.Marshal(state)
	if err != nil {
		return models.DebateState{}, fmt.Errorf("marshal debate state: %w", err)
	}
	if err := w.redis.Set(ctx, cacheKey, raw, 24*time.Hour).Err(); err != nil {
		return models.DebateState{}, fmt.Errorf("set debate state: %w", err)
	}
	return state, nil
}
