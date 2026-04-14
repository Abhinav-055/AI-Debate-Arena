package workers

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"testing"
	"time"

	"debate-arena/internal/models"
	"debate-arena/internal/queue"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type storeStub struct {
	debate models.Debate
}

func (s *storeStub) GetDebateByID(ctx context.Context, id string) (models.Debate, error) {
	return s.debate, nil
}
func (s *storeStub) ListRecentMessages(ctx context.Context, debateID string, n int) ([]models.Message, error) {
	return []models.Message{}, nil
}
func (s *storeStub) CreateMessage(ctx context.Context, debateID, agentID string, round int, content string) (models.Message, error) {
	return models.Message{}, nil
}
func (s *storeStub) IncrementRoundsDone(ctx context.Context, debateID string) error { return nil }
func (s *storeStub) UpdateDebateStatus(ctx context.Context, debateID string, status models.DebateStatus) error {
	return nil
}
func (s *storeStub) ListMessages(ctx context.Context, debateID string, round *int) ([]models.Message, error) {
	return nil, nil
}
func (s *storeStub) SaveJudgement(ctx context.Context, j models.Judgement) error { return nil }

type publisherStub struct {
	routing string
	env     queue.Envelope
}

func (p *publisherStub) Publish(ctx context.Context, routingKey string, msg queue.Envelope) error {
	p.routing = routingKey
	p.env = msg
	return nil
}

func TestDebateWorkerTurnSequencing(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	store := &storeStub{debate: models.Debate{
		ID:     "debate-1",
		Topic:  "AI safety",
		Status: models.DebateStatusPending,
		Agents: []models.Agent{
			{ID: "agent-a", TurnOrder: 0},
			{ID: "agent-b", TurnOrder: 1},
		},
	}}
	pub := &publisherStub{}
	w := NewDebateWorker(store, rdb, pub, slog.New(slog.NewTextHandler(io.Discard, nil)))

	payload, _ := json.Marshal(TurnStartPayload{Round: 1, AgentIndex: 0})
	err := w.HandleTurn(context.Background(), queue.Envelope{
		EventType:  "turn.start",
		DebateID:   "debate-1",
		Payload:    payload,
		RetryCount: 0,
		Timestamp:  time.Now().UTC(),
	})
	require.NoError(t, err)
	require.Equal(t, queue.RoutingDebateAIGenerate, pub.routing)

	var task map[string]any
	require.NoError(t, json.Unmarshal(pub.env.Payload, &task))
	require.Equal(t, "agent-a", task["agent_id"])
	require.EqualValues(t, float64(1), task["round"])
}
