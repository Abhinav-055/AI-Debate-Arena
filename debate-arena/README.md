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
- Node.js and npm (for running the frontend)
- Go 1.25+ (for running migrations or the backend locally without Docker)
- Optional tools:
  - `wscat` for subscription testing

## Quick Start (Windows PowerShell)

1. Create a `.env` file in the project root with these settings. Replace the Groq key and JWT secret with your own values:

```env
DATABASE_URL=postgres://postgres:postgres@postgres:5432/debate_arena?sslmode=disable
REDIS_URL=redis://redis:6379/0
RABBITMQ_URL=amqp://guest:guest@rabbitmq:5672/
GROQ_API_KEY=your_groq_api_key
JWT_SECRET=replace_with_a_long_random_secret
FRONTEND_ORIGIN=http://localhost:5173
POSTGRES_PORT=5433
PORT=8080
LOG_LEVEL=info
MAX_ROUNDS=3
```

2. Start PostgreSQL from the project root:

```powershell
docker compose up -d postgres
```

3. Apply the database schema before starting the API. This uses the host-mapped PostgreSQL port; substitute your `POSTGRES_PORT` if you changed it:

```powershell
go run github.com/pressly/goose/v3/cmd/goose@latest -dir internal/db/migrations postgres "postgres://postgres:postgres@localhost:5433/debate_arena?sslmode=disable" up
```

4. Build and start the API, worker, and remaining dependencies:

```powershell
docker compose up --build -d
```

5. Check service status and API health:

```powershell
docker compose ps
Invoke-RestMethod -Uri http://localhost:8080/healthz
```

6. Open the GraphQL playground at `http://localhost:8080/`.

7. In a separate terminal, start the frontend as described below.

## Frontend Development

The frontend is a Vite/React app. Keep the backend stack running while using it; the app connects to the GraphQL API at `http://localhost:8080/graphql` and uses `ws://localhost:8080/graphql` for subscriptions.

From the project root, run in a separate PowerShell terminal:

```powershell
Set-Location frontend
npm ci
npm run dev
```

Open the frontend at:

```text
http://localhost:5173/
```

To create a production build, run `npm run build` from the `frontend` directory.

After creating a debate, open it and select **Start Debate** to enqueue its first turn. The worker currently uses Groq's `openai/gpt-oss-20b` model. To watch service logs, run `docker compose logs -f api worker` from the project root. If you change `GROQ_API_KEY` in `.env`, recreate the worker so it loads the new value:

```powershell
docker compose up -d --force-recreate worker
```

Stop the containers without deleting the database volume:

```powershell
docker compose down
```

## Database Migrations

Migrations are in `internal/db/migrations` using goose SQL format. Quick Start runs Goose with `go run`, so a separate Goose installation is not needed. To check migration status after PostgreSQL is running:

```powershell
go run github.com/pressly/goose/v3/cmd/goose@latest -dir internal/db/migrations postgres "postgres://postgres:postgres@localhost:5433/debate_arena?sslmode=disable" status
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
| `FRONTEND_ORIGIN` | no | Allowed browser origin for GraphQL WebSockets (default `http://localhost:5173`) |
| `POSTGRES_PORT` | no | Host port mapped to PostgreSQL (default `5433`; container port remains `5432`) |
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
