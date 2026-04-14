# AI Debate Arena

AI Debate Arena is a distributed, event-driven Go backend where two AI agents with distinct personalities debate topics in real time.

## Architecture

```text
                         +------------------------------+
Client (HTTP/WebSocket)  |          API (Chi)          |
GraphQL Query/Mutation ->| gqlgen resolvers            |
Subscriptions <----------| Redis-backed subscription hub|
                         +---------------+--------------+
                                         |
                                         | Publish events (RabbitMQ topic)
                                         v
                          +--------------+---------------+
                          |    debate.events exchange    |
                          +----+-------------+-----------+
                               |             |
                debate.turn.requested   debate.ai.generate   debate.judge.evaluate
                               |             |               |
                               v             v               v
                        +------+--+    +-----+---+     +-----+-----+
                        |Debate   |    |AI Worker |     |Judge Worker|
                        |Worker   |    | (Groq)   |     |(Groq JSON) |
                        +---+-----+    +-----+----+     +-----+------+
                            |          +-----+----+           |
                            |                |                |
                            v                v                v
                     +------+----------------+----------------+------+
                     | PostgreSQL (debates/agents/messages/judgements)|
                     +------+----------------+----------------+------+
                            |                |
                            v                v
                     Redis state/cache   Redis pub/sub debate:{id}
```

## Tech Stack

- Go 1.22+
- Chi v5
- gqlgen (schema-first GraphQL)
- RabbitMQ (`amqp091-go`)
- PostgreSQL (`pgx/v5`) + goose migrations
- Redis (`go-redis/v9`)
- JWT (`golang-jwt/v5`)
- Viper + `.env`
- AI Provider: Groq
- Docker + docker-compose
- Tests: testify + gomock

## Project Layout

```text
debate-arena/
├── cmd/
│   ├── server/main.go
│   └── worker/main.go
├── internal/
│   ├── config/
│   ├── graph/
│   ├── models/
│   ├── db/
│   │   ├── migrations/
│   │   └── queries/
│   ├── queue/
│   ├── workers/
│   ├── ai/
│   ├── subscriptions/
│   └── middleware/
├── docker-compose.yml
├── Dockerfile
├── .env
└── README.md
```

## Prerequisites

- Docker Desktop (or Docker Engine + Compose)
- Go 1.22+ (if running locally without Docker)
- Optional tools:
  - `goose` for manual migration workflows
  - `wscat` for subscription testing

## Quick Start

1. Ensure `.env` is configured.
2. From project root, run:

```bash
docker-compose up --build
```

3. Open GraphQL playground:

```text
http://localhost:8080/
```

4. Health check:

```bash
curl http://localhost:8080/healthz
```

## Database Migrations

Migrations are in `internal/db/migrations` using goose SQL format.

If running locally with goose:

```bash
go install github.com/pressly/goose/v3/cmd/goose@latest
goose -dir internal/db/migrations postgres "$DATABASE_URL" up
```

## GraphQL API Examples

### Create Debate

```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{
    "query":"mutation($input: CreateDebateInput!){ createDebate(input:$input){ id topic status roundsTotal agents { id name provider personality turnOrder } } }",
    "variables":{
      "input":{
        "topic":"Should open-source models be mandatory for public AI systems?",
        "rounds":3,
        "agents":[
          {"name":"Logos","provider":"GROQ","personality":"logical and evidence-driven"},
          {"name":"Blaze","provider":"GROQ","personality":"aggressive and confrontational"}
        ]
      }
    }
  }'
```

### Start Debate

```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query":"mutation { startDebate(id:\"<DEBATE_ID>\") { id status } }"}'
```

### Query Messages

```bash
curl -X POST http://localhost:8080/query \
  -H "Content-Type: application/json" \
  -d '{"query":"query { messages(debateId:\"<DEBATE_ID>\") { id roundNumber agentId content createdAt } }"}'
```

### Subscription (wscat)

Install wscat:

```bash
npm i -g wscat
```

Connect:

```bash
wscat -c ws://localhost:8080/query
```

Then send GraphQL websocket messages following the GraphQL over WebSocket protocol; for example a `messageAdded` subscription filtered by `debateId`.

## Debate Flow

1. `createDebate` writes debate + agents to PostgreSQL with `pending` status.
2. `startDebate` publishes `turn.start` envelope to `debate.turn.requested`.
3. Debate worker resolves current turn state (Redis cache fallback to DB).
4. AI worker routes to provider, generates response, stores message, updates cache.
5. AI worker emits either next turn or judge event when rounds complete.
6. Judge worker evaluates transcript, stores judgement, publishes readiness event.
7. Redis pub/sub events are fanned out to GraphQL subscription clients.

## Environment Variables

| Variable | Required | Description |
|---|---|---|
| `DATABASE_URL` | yes | PostgreSQL DSN |
| `REDIS_URL` | yes | Redis URL (e.g. `redis://redis:6379/0`) |
| `RABBITMQ_URL` | yes | RabbitMQ AMQP URL |
| `GROQ_API_KEY` | yes | Groq API key |
| `JWT_SECRET` | yes | HMAC secret for JWT middleware |
| `PORT` | no | API port (default `8080`) |
| `LOG_LEVEL` | no | `debug`, `info`, `warn`, `error` |
| `MAX_ROUNDS` | no | Default max rounds (default `3`) |

## Running Tests

```bash
go test ./...
```

Included tests:

- Queue publish/consume round-trip (`internal/queue`)
- Debate worker turn sequencing (`internal/workers`)
- Groq retry handling on 429 (`internal/ai`)
- GraphQL integration test for create/start mutations (`internal/graph`)

## Provider Notes

- Runtime generation/judging is Groq-only.
- Exactly two agents are required per debate.
- A round completes when both agents have spoken once.

## Notes

- No ORM is used; all DB access is raw SQL through pgx.
- Dependencies are constructor-injected to avoid global state.
- Workers and subscription manager stop gracefully via context cancellation.
- Structured logging is done with `log/slog`.
- `gqlgen.yml` is committed and `go generate ./...` is supported.
