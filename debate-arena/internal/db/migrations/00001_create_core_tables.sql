-- +goose Up
-- +goose StatementBegin
CREATE TYPE debate_status AS ENUM ('pending', 'active', 'completed');
CREATE TYPE agent_provider AS ENUM ('google', 'groq', 'huggingface');

CREATE TABLE debates (
  id UUID PRIMARY KEY,
  topic TEXT NOT NULL,
  status debate_status NOT NULL DEFAULT 'pending',
  rounds_total INT NOT NULL CHECK (rounds_total > 0),
  rounds_done INT NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE agents (
  id UUID PRIMARY KEY,
  debate_id UUID NOT NULL REFERENCES debates(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  provider agent_provider NOT NULL,
  personality TEXT NOT NULL,
  turn_order INT NOT NULL,
  UNIQUE (debate_id, turn_order)
);

CREATE TABLE messages (
  id UUID PRIMARY KEY,
  debate_id UUID NOT NULL REFERENCES debates(id) ON DELETE CASCADE,
  agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
  round_number INT NOT NULL CHECK (round_number > 0),
  content TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE judgements (
  id UUID PRIMARY KEY,
  debate_id UUID NOT NULL UNIQUE REFERENCES debates(id) ON DELETE CASCADE,
  winner_agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
  strengths JSONB NOT NULL,
  weaknesses JSONB NOT NULL,
  verdict TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_debates_status ON debates(status);
CREATE INDEX idx_messages_debate_round ON messages(debate_id, round_number);
CREATE INDEX idx_messages_created_at_desc ON messages(created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_messages_created_at_desc;
DROP INDEX IF EXISTS idx_messages_debate_round;
DROP INDEX IF EXISTS idx_debates_status;

DROP TABLE IF EXISTS judgements;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS debates;

DROP TYPE IF EXISTS agent_provider;
DROP TYPE IF EXISTS debate_status;
-- +goose StatementEnd
