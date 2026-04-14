package models

import "time"

// DebateStatus represents the lifecycle state of a debate.
type DebateStatus string

const (
	// DebateStatusPending marks a debate waiting to start.
	DebateStatusPending DebateStatus = "pending"
	// DebateStatusActive marks a running debate.
	DebateStatusActive DebateStatus = "active"
	// DebateStatusCompleted marks a finished debate.
	DebateStatusCompleted DebateStatus = "completed"
)

// AgentProvider identifies the AI backend used by an agent.
type AgentProvider string

const (
	// AgentProviderGroq maps to Groq.
	AgentProviderGroq AgentProvider = "groq"
)

// Debate is the root aggregate for a debate session.
type Debate struct {
	ID          string
	Topic       string
	Status      DebateStatus
	RoundsTotal int
	RoundsDone  int
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Agents      []Agent
}

// Agent describes one participant in a debate.
type Agent struct {
	ID          string
	DebateID    string
	Name        string
	Provider    AgentProvider
	Personality string
	TurnOrder   int
}

// Message stores one generated turn in the debate.
type Message struct {
	ID          string
	DebateID    string
	AgentID     string
	RoundNumber int
	Content     string
	CreatedAt   time.Time
}

// JudgementDetail represents per-agent reasoning for strengths/weaknesses.
type JudgementDetail struct {
	AgentID string `json:"agentId"`
	Reason  string `json:"reason"`
}

// Judgement stores final post-debate evaluation.
type Judgement struct {
	ID            string
	DebateID      string
	WinnerAgentID string
	Strengths     []JudgementDetail
	Weaknesses    []JudgementDetail
	Verdict       string
	CreatedAt     time.Time
}

// DebateState stores the active orchestration state in Redis.
type DebateState struct {
	Status            DebateStatus `json:"status"`
	CurrentRound      int          `json:"current_round"`
	CurrentAgentIndex int          `json:"current_agent_index"`
}

// CreateDebateInput captures the mutation payload for a new debate.
type CreateDebateInput struct {
	Topic  string
	Rounds int
	Agents []CreateDebateAgentInput
}

// CreateDebateAgentInput captures one agent declaration at creation time.
type CreateDebateAgentInput struct {
	Name        string
	Provider    AgentProvider
	Personality string
}
