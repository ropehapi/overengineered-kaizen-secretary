# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
# Run locally
go run cmd/worker/main.go

# Build binary
go build -o bin/worker ./cmd/worker

# Run binary
./bin/worker

# Docker
docker compose up -d
```

No test suite exists in this project.

## Architecture

Go worker process that runs scheduled cron jobs indefinitely. No HTTP server — it blocks with `select{}` after starting the scheduler.

**Startup sequence** (`cmd/worker/main.go`):
1. Initialize structured JSON logger (`internal/logger/`)
2. Load `.env` via `godotenv`
3. Set timezone to `America/Sao_Paulo`
4. Create `robfig/cron` scheduler with second-level precision (`cron.WithSeconds()`)
5. Register routine functions with cron expressions via `c.AddFunc`
6. Start scheduler and block forever

## Adding a new routine

1. Create a new file in `internal/routines/` with a `func MyRoutine()` signature
2. Register it in `cmd/worker/main.go`: `c.AddFunc("0 0 9 * * 1", routines.MyRoutine)`

Cron format is 6-field (seconds first): `"Second Minute Hour DayOfMonth Month DayOfWeek"`

## Routine conventions

- Functions must be `func()` with no parameters (cron requirement)
- Read config from `os.Getenv`
- Use `slog` (via `internal/logger/`) for structured JSON logging — not `fmt.Println`
- Routines call the `messaging-officer` API via plain `net/http` POST with JSON payload
- Add random delays between sends to avoid rate limiting (see `mensalidadesEscoteiro.go`)

## Environment variables

| Variable | Description |
|---|---|
| `MESSAGING_OFFICER_HOST` | Base URL of messaging-officer API (e.g. `http://localhost`) |
| `MESSAGING_OFFICER_PORT` | Port of messaging-officer API (e.g. `3000`) |
| `MESSAGING_OFFICER_API_KEY` | API authentication key |
| `MESSAGING_OFFICER_SESSION_ID` | WhatsApp session identifier |

Copy `.env.example` to `.env` to get started.

## Ecosystem context

This worker is part of the **manda-pra-mim** ecosystem, connected via the external Docker network `manda-pra-mim`:
- **messaging-officer** — WhatsApp REST API (Baileys-based, port 3000). All routines send messages through this service.
- **kaizen-wpp-scheduler-backend** — Go REST API for schedule management (port 8080)
- **kaizen-wpp-scheduler-frontend** — React UI for the above
