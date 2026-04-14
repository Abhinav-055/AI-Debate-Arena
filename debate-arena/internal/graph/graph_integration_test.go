package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"debate-arena/internal/graph/generated"
	"debate-arena/internal/models"
	"debate-arena/internal/queue"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/stretchr/testify/require"
)

type graphStoreStub struct {
	debate models.Debate
}

func (s *graphStoreStub) CreateDebate(ctx context.Context, input models.CreateDebateInput) (models.Debate, error) {
	s.debate = models.Debate{ID: "debate-1", Topic: input.Topic, Status: models.DebateStatusPending, RoundsTotal: input.Rounds}
	for i, a := range input.Agents {
		s.debate.Agents = append(s.debate.Agents, models.Agent{ID: "agent-" + a.Name, DebateID: "debate-1", Name: a.Name, Provider: a.Provider, Personality: a.Personality, TurnOrder: i})
	}
	return s.debate, nil
}
func (s *graphStoreStub) GetDebateByID(ctx context.Context, id string) (models.Debate, error) {
	return s.debate, nil
}
func (s *graphStoreStub) ListDebates(ctx context.Context, status *models.DebateStatus, limit, offset int) ([]models.Debate, error) {
	return []models.Debate{s.debate}, nil
}
func (s *graphStoreStub) ListMessages(ctx context.Context, debateID string, round *int) ([]models.Message, error) {
	return nil, nil
}
func (s *graphStoreStub) GetJudgement(ctx context.Context, debateID string) (models.Judgement, error) {
	return models.Judgement{}, nil
}

type graphPublisherStub struct {
	published int
}

func (p *graphPublisherStub) Publish(ctx context.Context, routingKey string, msg queue.Envelope) error {
	p.published++
	return nil
}

func TestCreateDebateThenStartDebate(t *testing.T) {
	store := &graphStoreStub{}
	pub := &graphPublisherStub{}
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{Resolvers: &Resolver{
		Store:     store,
		Publisher: pub,
		Logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
	}}))

	createBody := map[string]any{
		"query": `mutation($input: CreateDebateInput!){ createDebate(input:$input){ id topic status } }`,
		"variables": map[string]any{
			"input": map[string]any{
				"topic":  "AI in healthcare",
				"rounds": 2,
				"agents": []map[string]any{{"name": "Alpha", "provider": "GROQ", "personality": "logical"}, {"name": "Beta", "provider": "GROQ", "personality": "aggressive"}},
			},
		},
	}
	createRaw, _ := json.Marshal(createBody)
	createReq := httptest.NewRequest(http.MethodPost, "/query", bytes.NewReader(createRaw))
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	srv.ServeHTTP(createRec, createReq)
	require.Equal(t, http.StatusOK, createRec.Code)
	require.Contains(t, createRec.Body.String(), "createDebate")

	startBody := map[string]any{
		"query": `mutation { startDebate(id:"debate-1"){ id } }`,
	}
	startRaw, _ := json.Marshal(startBody)
	startReq := httptest.NewRequest(http.MethodPost, "/query", bytes.NewReader(startRaw))
	startReq.Header.Set("Content-Type", "application/json")
	startRec := httptest.NewRecorder()
	srv.ServeHTTP(startRec, startReq)
	require.Equal(t, http.StatusOK, startRec.Code)
	require.GreaterOrEqual(t, pub.published, 1)
}
