package workers

import (
	"context"

	"debate-arena/internal/models"
	"debate-arena/internal/queue"
)

// DebateStore defines DB operations required by workers.
type DebateStore interface {
	GetDebateByID(ctx context.Context, id string) (models.Debate, error)
	ListRecentMessages(ctx context.Context, debateID string, n int) ([]models.Message, error)
	CreateMessage(ctx context.Context, debateID, agentID string, round int, content string) (models.Message, error)
	IncrementRoundsDone(ctx context.Context, debateID string) error
	UpdateDebateStatus(ctx context.Context, debateID string, status models.DebateStatus) error
	ListMessages(ctx context.Context, debateID string, round *int) ([]models.Message, error)
	SaveJudgement(ctx context.Context, j models.Judgement) error
}

// EventPublisher defines message publishing capabilities.
type EventPublisher interface {
	Publish(ctx context.Context, routingKey string, msg queue.Envelope) error
}
