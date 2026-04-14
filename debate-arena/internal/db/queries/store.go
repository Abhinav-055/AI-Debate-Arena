package queries

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"debate-arena/internal/models"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store exposes raw SQL operations for the debate arena.
type Store struct {
	pool *pgxpool.Pool
}

// NewStore creates a new query store.
func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// CreateDebate inserts one debate and its agents in a transaction.
func (s *Store) CreateDebate(ctx context.Context, input models.CreateDebateInput) (models.Debate, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return models.Debate{}, fmt.Errorf("begin create debate tx: %w", err)
	}
	defer tx.Rollback(ctx)

	now := time.Now().UTC()
	debate := models.Debate{
		ID:          uuid.NewString(),
		Topic:       input.Topic,
		Status:      models.DebateStatusPending,
		RoundsTotal: input.Rounds,
		RoundsDone:  0,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO debates (id, topic, status, rounds_total, rounds_done, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`, debate.ID, debate.Topic, debate.Status, debate.RoundsTotal, debate.RoundsDone, debate.CreatedAt, debate.UpdatedAt); err != nil {
		return models.Debate{}, fmt.Errorf("insert debate: %w", err)
	}

	debate.Agents = make([]models.Agent, 0, len(input.Agents))
	for i, a := range input.Agents {
		agent := models.Agent{
			ID:          uuid.NewString(),
			DebateID:    debate.ID,
			Name:        a.Name,
			Provider:    a.Provider,
			Personality: a.Personality,
			TurnOrder:   i,
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO agents (id, debate_id, name, provider, personality, turn_order)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, agent.ID, agent.DebateID, agent.Name, agent.Provider, agent.Personality, agent.TurnOrder); err != nil {
			return models.Debate{}, fmt.Errorf("insert agent: %w", err)
		}

		debate.Agents = append(debate.Agents, agent)
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Debate{}, fmt.Errorf("commit create debate tx: %w", err)
	}

	return debate, nil
}

// GetDebateByID returns one debate with its agents.
func (s *Store) GetDebateByID(ctx context.Context, id string) (models.Debate, error) {
	var d models.Debate
	if err := s.pool.QueryRow(ctx, `
		SELECT id, topic, status, rounds_total, rounds_done, created_at, updated_at
		FROM debates WHERE id = $1
	`, id).Scan(&d.ID, &d.Topic, &d.Status, &d.RoundsTotal, &d.RoundsDone, &d.CreatedAt, &d.UpdatedAt); err != nil {
		return models.Debate{}, fmt.Errorf("select debate by id: %w", err)
	}
	agents, err := s.GetAgentsByDebateID(ctx, id)
	if err != nil {
		return models.Debate{}, fmt.Errorf("load debate agents: %w", err)
	}
	d.Agents = agents
	return d, nil
}

// ListDebates returns debates filtered by status.
func (s *Store) ListDebates(ctx context.Context, status *models.DebateStatus, limit, offset int) ([]models.Debate, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	var rows pgx.Rows
	var err error
	if status != nil {
		rows, err = s.pool.Query(ctx, `
			SELECT id, topic, status, rounds_total, rounds_done, created_at, updated_at
			FROM debates WHERE status = $1
			ORDER BY created_at DESC LIMIT $2 OFFSET $3
		`, *status, limit, offset)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT id, topic, status, rounds_total, rounds_done, created_at, updated_at
			FROM debates
			ORDER BY created_at DESC LIMIT $1 OFFSET $2
		`, limit, offset)
	}
	if err != nil {
		return nil, fmt.Errorf("query debates: %w", err)
	}
	defer rows.Close()

	out := make([]models.Debate, 0)
	for rows.Next() {
		var d models.Debate
		if err := rows.Scan(&d.ID, &d.Topic, &d.Status, &d.RoundsTotal, &d.RoundsDone, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan debate row: %w", err)
		}
		d.Agents, err = s.GetAgentsByDebateID(ctx, d.ID)
		if err != nil {
			return nil, fmt.Errorf("load agents for debate: %w", err)
		}
		out = append(out, d)
	}

	return out, nil
}

// GetAgentsByDebateID returns all agents in turn order.
func (s *Store) GetAgentsByDebateID(ctx context.Context, debateID string) ([]models.Agent, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, debate_id, name, provider, personality, turn_order
		FROM agents WHERE debate_id = $1
		ORDER BY turn_order ASC
	`, debateID)
	if err != nil {
		return nil, fmt.Errorf("query agents by debate id: %w", err)
	}
	defer rows.Close()

	out := make([]models.Agent, 0)
	for rows.Next() {
		var a models.Agent
		if err := rows.Scan(&a.ID, &a.DebateID, &a.Name, &a.Provider, &a.Personality, &a.TurnOrder); err != nil {
			return nil, fmt.Errorf("scan agent row: %w", err)
		}
		out = append(out, a)
	}
	return out, nil
}

// ListMessages returns messages for a debate, optionally filtered by round.
func (s *Store) ListMessages(ctx context.Context, debateID string, round *int) ([]models.Message, error) {
	var rows pgx.Rows
	var err error
	if round != nil {
		rows, err = s.pool.Query(ctx, `
			SELECT id, debate_id, agent_id, round_number, content, created_at
			FROM messages WHERE debate_id = $1 AND round_number = $2
			ORDER BY created_at ASC
		`, debateID, *round)
	} else {
		rows, err = s.pool.Query(ctx, `
			SELECT id, debate_id, agent_id, round_number, content, created_at
			FROM messages WHERE debate_id = $1
			ORDER BY round_number ASC, created_at ASC
		`, debateID)
	}
	if err != nil {
		return nil, fmt.Errorf("query messages: %w", err)
	}
	defer rows.Close()

	out := make([]models.Message, 0)
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.DebateID, &m.AgentID, &m.RoundNumber, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan message row: %w", err)
		}
		out = append(out, m)
	}
	return out, nil
}

// ListRecentMessages returns the latest N messages in ascending order.
func (s *Store) ListRecentMessages(ctx context.Context, debateID string, n int) ([]models.Message, error) {
	if n <= 0 {
		n = 4
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, debate_id, agent_id, round_number, content, created_at
		FROM (
			SELECT id, debate_id, agent_id, round_number, content, created_at
			FROM messages WHERE debate_id = $1
			ORDER BY created_at DESC LIMIT $2
		) t ORDER BY created_at ASC
	`, debateID, n)
	if err != nil {
		return nil, fmt.Errorf("query recent messages: %w", err)
	}
	defer rows.Close()

	out := make([]models.Message, 0)
	for rows.Next() {
		var m models.Message
		if err := rows.Scan(&m.ID, &m.DebateID, &m.AgentID, &m.RoundNumber, &m.Content, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan recent message row: %w", err)
		}
		out = append(out, m)
	}
	return out, nil
}

// CreateMessage persists one generated agent response.
func (s *Store) CreateMessage(ctx context.Context, debateID, agentID string, round int, content string) (models.Message, error) {
	m := models.Message{
		ID:          uuid.NewString(),
		DebateID:    debateID,
		AgentID:     agentID,
		RoundNumber: round,
		Content:     content,
		CreatedAt:   time.Now().UTC(),
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO messages (id, debate_id, agent_id, round_number, content, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, m.ID, m.DebateID, m.AgentID, m.RoundNumber, m.Content, m.CreatedAt); err != nil {
		return models.Message{}, fmt.Errorf("insert message: %w", err)
	}
	return m, nil
}

// UpdateDebateStatus updates the debate status field.
func (s *Store) UpdateDebateStatus(ctx context.Context, debateID string, status models.DebateStatus) error {
	if _, err := s.pool.Exec(ctx, `
		UPDATE debates SET status = $2, updated_at = NOW() WHERE id = $1
	`, debateID, status); err != nil {
		return fmt.Errorf("update debate status: %w", err)
	}
	return nil
}

// IncrementRoundsDone increments rounds_done by one.
func (s *Store) IncrementRoundsDone(ctx context.Context, debateID string) error {
	if _, err := s.pool.Exec(ctx, `
		UPDATE debates SET rounds_done = rounds_done + 1, updated_at = NOW() WHERE id = $1
	`, debateID); err != nil {
		return fmt.Errorf("increment rounds_done: %w", err)
	}
	return nil
}

// SaveJudgement stores final judgement data for a debate.
func (s *Store) SaveJudgement(ctx context.Context, j models.Judgement) error {
	strengths, err := json.Marshal(j.Strengths)
	if err != nil {
		return fmt.Errorf("marshal strengths: %w", err)
	}
	weaknesses, err := json.Marshal(j.Weaknesses)
	if err != nil {
		return fmt.Errorf("marshal weaknesses: %w", err)
	}

	if _, err := s.pool.Exec(ctx, `
		INSERT INTO judgements (id, debate_id, winner_agent_id, strengths, weaknesses, verdict, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (debate_id) DO UPDATE
		SET winner_agent_id = EXCLUDED.winner_agent_id,
		    strengths = EXCLUDED.strengths,
		    weaknesses = EXCLUDED.weaknesses,
		    verdict = EXCLUDED.verdict,
		    created_at = EXCLUDED.created_at
	`, j.ID, j.DebateID, j.WinnerAgentID, strengths, weaknesses, j.Verdict, j.CreatedAt); err != nil {
		return fmt.Errorf("insert judgement: %w", err)
	}
	return nil
}

// GetJudgement returns the judgement associated with a debate.
func (s *Store) GetJudgement(ctx context.Context, debateID string) (models.Judgement, error) {
	var j models.Judgement
	var strengthsRaw []byte
	var weaknessesRaw []byte
	if err := s.pool.QueryRow(ctx, `
		SELECT id, debate_id, winner_agent_id, strengths, weaknesses, verdict, created_at
		FROM judgements WHERE debate_id = $1
	`, debateID).Scan(&j.ID, &j.DebateID, &j.WinnerAgentID, &strengthsRaw, &weaknessesRaw, &j.Verdict, &j.CreatedAt); err != nil {
		return models.Judgement{}, fmt.Errorf("select judgement: %w", err)
	}
	if err := json.Unmarshal(strengthsRaw, &j.Strengths); err != nil {
		return models.Judgement{}, fmt.Errorf("unmarshal strengths: %w", err)
	}
	if err := json.Unmarshal(weaknessesRaw, &j.Weaknesses); err != nil {
		return models.Judgement{}, fmt.Errorf("unmarshal weaknesses: %w", err)
	}
	return j, nil
}
