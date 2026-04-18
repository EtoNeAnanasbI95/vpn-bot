# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What This Is

A Telegram bot for managing VPN access and payments, written in Go. It integrates with a **3x-ui panel** for VPN client management and provides admin controls for users, connections, and payments.

## Common Commands

```bash
make build      # Build binary to ./bin/bot
make run-dev    # Run with DEBUG=true
make run-prod   # Run in production mode
make tidy       # go mod tidy
make lint       # golangci-lint

# Docker
docker-compose up           # Run via docker-compose
docker build -t vpn-bot .   # Build Docker image
```

Entry point: `cmd/bot/main.go`

## Architecture

Layered clean architecture: **Domain → Use Cases → Handlers → Router**

```
Telegram Webhook (POST /webhook)
        │
        ▼
    Router  ←── Auth Middleware
        │
        ├── Handlers (internal/bot/handler/)
        │       │
        │       ▼
        │   Use Cases (internal/usecase/)
        │       │             │
        │       ▼             ▼
        │   SQLite DB     3x-ui Client
        │
        └── Session Store (in-memory, per user FSM state)

Scheduler (cron) ──► Payment reminders (12:00, 20:00)
                 └──► Block overdue connections (10:00)
```

**Key packages:**

| Package | Role |
|---|---|
| `internal/bot` | Router, server, handlers, sessions, keyboard builders, callback parsing |
| `internal/usecase` | Business logic for users, connections, payments, guides |
| `internal/repository/sqlite` | SQLite-backed data access; also owns schema creation and migrations |
| `internal/domain` | Plain structs: `User`, `Connection`, `Payment`, `ConnectionPayment` |
| `internal/xui` | HTTP client for 3x-ui panel REST API |
| `internal/scheduler` | Cron jobs using `robfig/cron` |
| `internal/config` | Env var loading; `cfg.IsAdmin(id)` for auth |
| `internal/guide` | Serves platform PDF guides from `assets/guides/` |

**Session management:** Multi-step flows (broadcasts, connection creation) use an in-memory FSM store keyed by Telegram user ID (`internal/bot/session/memory_store.go`).

**Callback data:** Structured callback payloads are encoded/decoded in `internal/bot/callback/parse.go`.

## Database

SQLite at `DB_PATH` (default `./data/bot.db`). Schema and migrations are applied automatically on startup via `sqlite.Migrate(db)` in `internal/repository/sqlite/db.go`.

- Named migrations are tracked in `_applied_migrations` table (idempotent re-runs safe).
- SQL migration files live in `migrations/`.
- DB uses WAL mode, foreign keys enabled, single connection (`SetMaxOpenConns(1)`).

Main tables: `users`, `connections`, `connection_payments`, `payments`, `admin_profiles`.

## Configuration

All config is loaded from environment variables (`.env` file auto-loaded if present). See `.env.example` for the full list. Required variables:

| Variable | Purpose |
|---|---|
| `TELEGRAM_TOKEN` | Telegram bot token |
| `ADMIN_IDS` | Comma-separated Telegram IDs with admin access |

Notable optional variables: `XUI_BASE_URL`, `XUI_USERNAME`, `XUI_PASSWORD`, `XUI_SERVER_ADDR`, `XUI_INBOUND_ID` (all required for VPN connection features), `WEBHOOK_URL` / `LISTEN_ADDR` (for webhook mode), `REMINDER_TZ` (default `Europe/Moscow`), `DEBUG`.

## Assets

Platform setup guides (PDF files) go in `assets/guides/` — named `ios.pdf`, `android.pdf`, `windows.pdf`, `macos.pdf`, `linux.pdf`. See `assets/guides/README.md` and `internal/guide/fs_provider.go` for platform registration.

## CI / Deployment

GitHub Actions workflow (`.github/workflows/build.yml`) builds a Docker image on push to `main` and exports it as an artifact. The `docker-compose.yml` connects to an external `web-server` network and mounts `.env`, `./data`, and `./assets`.
