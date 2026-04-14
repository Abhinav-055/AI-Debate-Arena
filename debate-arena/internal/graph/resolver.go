package graph

//go:generate go run github.com/99designs/gqlgen generate

import (
	"log/slog"

	"context"
	"debate-arena/internal/models"
	"debate-arena/internal/queue"
	"debate-arena/internal/subscriptions"
)

// Store defines resolver data-access operations.
type Store interface {
	CreateDebate(ctx context.Context, input models.CreateDebateInput) (models.Debate, error)
	GetDebateByID(ctx context.Context, id string) (models.Debate, error)
	ListDebates(ctx context.Context, status *models.DebateStatus, limit, offset int) ([]models.Debate, error)
	ListMessages(ctx context.Context, debateID string, round *int) ([]models.Message, error)
	GetJudgement(ctx context.Context, debateID string) (models.Judgement, error)
}

// Publisher defines queue publishing needed by resolvers.
type Publisher interface {
	Publish(ctx context.Context, routingKey string, msg queue.Envelope) error
}

// Resolver wires dependencies used by GraphQL resolvers.
type Resolver struct {
	Store         Store
	Publisher     Publisher
	Subscriptions *subscriptions.Manager
	Logger        *slog.Logger
}
