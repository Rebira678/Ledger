# Ledger — SMS-Native Personal Finance Agent

Ledger reads the bank/mobile-money SMS that already exists on a user's Android phone,
parses it into transactions, categorizes it, answers ambiguities with clarifying
questions, and produces a weekly narrative spending report — no manual data entry.

> **Status:** portfolio project built from the SRS/PRD/API-Contract v1.0 in-repo.
> See [DECISIONS.md](DECISIONS.md) for every open design decision and its rationale.

## Architecture

```text
                         ┌──────────────────────────────┐
                         │        Android client        │
                         │  SMS BroadcastReceiver ─►    │
                         │  sender-ID allow-list ─►     │
                         │  WorkManager offline queue   │
                         └──────────────┬───────────────┘
                                        │ HTTPS (JWT + X-Device-Key)
                                        ▼
┌──────────────┐   HTTPS   ┌───────────────────────────────────────────┐
│   Web        │◄──────────│              Go backend                   │
│  dashboard   │           │  ingestion → parser plugins → categorizer │
│ (Go templates│           │  → agent orchestrator (LLM narrative)     │
│  + SVG charts│           │  → weekly report worker                   │
└──────────────┘           └───────────────┬───────────────────────────┘
                                           │ SQL (pgx, repo layer only)
                                           ▼
                                    ┌──────────────┐
                                    │  PostgreSQL  │
                                    └──────────────┘
```

Subsystems:

| Component | Location | Role |
|---|---|---|
| Go backend | `cmd/ledger-server`, `cmd/ledger-worker`, `internal/…` | ingestion, parsing, categorization, agent, reports |
| Parser plugins | `internal/parsers` | CBE / Telebirr SMS formats; open/closed plugin interface |
| Android client | `android/` | SMS capture, local filtering, offline queue (Kotlin, minSdk 26) |
| Dashboard | `internal/web`, `internal/api/dashboard.go` | server-rendered UI + SVG charts (D-09) |
| Migrations | `migrations/` | versioned `*.up.sql` / `*.down.sql` (embedded in the binary) |

## Quick start (docker compose)

```bash
cp .env.example .env          # then edit LEDGER_JWT_SIGNING_KEY!
docker compose up --build     # postgres + server on :8080
```

## Local development (no Docker for the app)

```bash
docker compose up -d postgres
cp .env.example .env
go run ./cmd/ledger-server     # migrations run automatically
```

## Running the tests

```bash
make test            # unit + integration (integration uses a real Postgres
                     #  via testcontainers — requires Docker running)
make test-unit       # unit tests only
make lint            # gofmt + go vet
```

## Scripted demo walkthrough

```bash
./scripts/demo-walkthrough.sh
```

Boots a throwaway Postgres container, starts the server, and walks the whole
flow live: register → login → device pairing → CBE/Telebirr SMS ingestion →
idempotent duplicate (409) → review queue (202) → clarifying question →
learning loop → dashboard summary. Run it to verify the deployment end-to-end.

## API

The full REST contract lives in `Ledger_API_Contract.pdf` (v1). Highlights:

- `POST /v1/auth/register` · `POST /v1/auth/login` · `POST /v1/auth/refresh`
- `POST /v1/devices/pair` (issues per-device API key)
- `POST /v1/ingest/sms` (idempotent via `client_message_id`) · `POST /v1/ingest/statement`
- `GET /v1/transactions` · `PATCH /v1/transactions/{id}/category`
- `GET /v1/agent/clarifications` · `POST /v1/agent/clarifications/{id}/respond`
- `GET /v1/reports/weekly[/{id}]` · `GET /v1/dashboard/summary`
- `GET /healthz` (liveness) · `GET /readyz` (readiness)

Errors use the contract envelope: `{"error":{"code","message","field"}}`.

## Parser extensibility

New bank/telecom formats are added as isolated plugin modules — no changes to
existing parsers or the ingestion API:

```go
// internal/parsers/parser.go
type Parser interface {
    SenderIDs() []string          // e.g. "CBE", "8329"
    Match(msg domain.RawMessage) bool
    Parse(msg domain.RawMessage) (domain.ParsedTransaction, error)
}
```

Unmatched messages are persisted to a review queue (`needs_review`), never silently
dropped (FR-3.3).

## Configuration

All configuration is environment-based; see the documented `.env.example`.
No secrets are hardcoded anywhere in the repo.

## What still needs a human

See [DECISIONS.md → Known Gaps](DECISIONS.md#known-gaps--require-human-action-before-end-to-end-validation):
a live LLM API key, real (redacted) bank SMS samples, physical-device SMS testing,
APK release signing, and production TLS.
