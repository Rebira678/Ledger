# Ledger — SMS-Native Personal Finance Agent

<div align="center">

Turn bank and mobile-money SMS into structured spending insights, automated categorization, and narrative financial reports — zero manual bookkeeping required.

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![Android](https://img.shields.io/badge/Android-Kotlin_minSdk_26-3DDC84?style=flat&logo=android)](https://developer.android.com)
[![PostgreSQL](https://img.shields.io/badge/Database-PostgreSQL_16-4169E1?style=flat&logo=postgresql)](https://www.postgresql.org)
[![Testing](https://img.shields.io/badge/Integration_Tests-Testcontainers-FF6B6B?style=flat&logo=docker)](https://testcontainers.com)
[![Architecture](https://img.shields.io/badge/Spec-SRS%20%2F%20PRD%20%2F%20v1.0-blueviolet)](Ledger_API_Contract.pdf)

[About](#about) • [Key Features](#key-features) • [Architecture](#system-architecture) • [Quick Start](#quick-start) • [API Contract](#api-surface) • [Extensibility](#parser-extensibility) • [Design Decisions](#design-decisions) • [Commit Message](#suggested-commit-message)

</div>

---

## About

> **GitHub Repository About Field:**  
> SMS-native personal finance agent. Automatically captures bank & mobile-money SMS on Android, parses transactions, resolves ambiguities via LLM, and generates weekly spending reports.

In emerging markets and digital economies, financial activity primarily flows through automated banking and mobile money notifications (such as CBE and Telebirr). **Ledger** turns an Android device into a passive financial ingestion engine. It intercepts transactional notifications on-device, pushes them to a Go backend via an authenticated, offline-resilient queue, resolves data ambiguities through an LLM agent feedback loop, and produces weekly narrative spending summaries.

> 📌 **Engineering Status:** Built strictly against an in-repo PRD, Software Requirements Specification (SRS), and versioned REST contract. For full architectural trade-offs, refer to [DECISIONS.md](DECISIONS.md).

---

## Key Features

- **Zero-Friction Ingestion:** Android `BroadcastReceiver` captures financial notifications locally using a strict sender-ID allowlist.
- **Offline Resilience:** Guaranteed at-least-once delivery backed by Android `WorkManager` and idempotent backend ingestion (`client_message_id`).
- **Open/Closed Parser Pipeline:** Isolated parser plugins handle regional bank and telecom SMS templates without touching ingestion logic.
- **Human-in-the-Loop Clarifications:** Surfaced prompts clarify ambiguous transfers or unknown merchants instead of hallucinating categories.
- **Device-Bound Security:** Direct client attestation via `X-Device-Key` hardware headers and rotating JWT sessions.
- **Zero-JS Web Console:** Lightweight server-rendered Go templates with embedded, zero-dependency SVG charts.

---

## System Architecture

```text
                        ┌──────────────────────────────────────┐
                        │            Android Client            │
                        │  SMS BroadcastReceiver (Allowlist)   │
                        │       WorkManager Offline Queue      │
                        └──────────────────┬───────────────────┘
                                           │ HTTPS (JWT + X-Device-Key)
                                           ▼
┌──────────────────┐    HTTPS     ┌────────────────────────────────────────────────────┐
│   Web Console    │◄─────────────│                     Go Backend                     │
│  (Go Templates   │              │  • Ingestion & Deduplication (idempotency keys)    │
│   + SVG Charts)  │              │  • Parser Plugin Pipeline (CBE, Telebirr)          │
└──────────────────┘              │  • Categorization Engine & LLM Orchestrator        │
                                  │  • Weekly Narrative Report Worker                  │
                                  └────────────────────────┬───────────────────────────┘
                                                           │ SQL (pgx pool)
                                                           ▼
                                                ┌─────────────────────┐
                                                │  PostgreSQL Store   │
                                                │ (Versioned Schemas) │
                                                └─────────────────────┘
```

### Subsystems

| Subsystem | Location | Tech Stack | Responsibility |
|---|---|---|---|
| **Backend & Worker** | `cmd/ledger-server`, `cmd/ledger-worker`, `internal/` | Go, `pgx`, `chi` / stdlib | Ingestion endpoints, deduplication, categorization pipeline, weekly cron worker |
| **Parser Plugins** | `internal/parsers/` | Go Interfaces, Regex | Isolated format matching for financial providers (CBE, Telebirr) |
| **Android Client** | `android/` | Kotlin, Jetpack WorkManager | Background SMS capture, sender allowlisting, device key attestation, backoff sync |
| **Web Dashboard** | `internal/web/`, `internal/api/` | Go Templates, SVG | Server-rendered analytical dashboards and transaction views |
| **Database Migrations** | `migrations/` | Raw SQL | Deterministic `*.up.sql` / `*.down.sql` migrations embedded directly into the binary |

---

## Quick Start

### Prerequisites
- [Docker & Docker Compose](https://www.docker.com/)
- [Go 1.22+](https://go.dev/) (for native host development)

### Run with Docker Compose

```bash
# 1. Clone repository and prepare environment
git clone [https://github.com/your-username/ledger.git](https://github.com/your-username/ledger.git)
cd ledger
cp .env.example .env

# Set a secure signing key for LEDGER_JWT_SIGNING_KEY in .env

# 2. Build and launch services
docker compose up --build
```
The server will bind to `http://localhost:8080`.

### Local Native Development

```bash
# Start standalone PostgreSQL container
docker compose up -d postgres

# Run backend (embedded migrations run automatically on boot)
cp .env.example .env
go run ./cmd/ledger-server
```

---

## Testing & Verification

Ledger uses [Testcontainers-Go](https://golang.testcontainers.org/) to run integration tests against ephemeral PostgreSQL instances:

```bash
# Run unit and containerized integration tests
make test

# Run isolated unit tests
make test-unit

# Run static analysis and formatting
make lint
```

### End-to-End Walkthrough Script

Verify the entire lifecycle (registration → device pairing → ingestion → 409 duplicate defense → review queue → clarifying question → learning loop → dashboard summary) in one command:

```bash
./scripts/demo-walkthrough.sh
```

---

## API Surface

All responses conform to the standard error/data envelope specification:
```json
{
  "data": {},
  "error": { "code": "VALIDATION_FAILED", "message": "...", "field": "sender_id" }
}
```

### Primary Endpoints

| Category | Method | Path | Description |
|---|---|---|---|
| **Auth & Security** | `POST` | `/v1/auth/register` | Create user account |
| | `POST` | `/v1/auth/login` | Acquire short-lived JWT + refresh token |
| | `POST` | `/v1/devices/pair` | Register Android client & issue `X-Device-Key` |
| **Ingestion** | `POST` | `/v1/ingest/sms` | Idempotent bulk/single SMS ingestion |
| | `POST` | `/v1/ingest/statement` | Historical statement bulk ingestion |
| **Transactions** | `GET` | `/v1/transactions` | Query and filter parsed ledger records |
| | `PATCH`| `/v1/transactions/{id}/category` | Manual category override |
| **Agent Loop** | `GET` | `/v1/agent/clarifications` | Fetch ambiguity queue |
| | `POST` | `/v1/agent/clarifications/{id}/respond`| Supply context to resolve category ambiguities |
| **Analytics** | `GET` | `/v1/reports/weekly` | Retrieve structured LLM narrative report |
| | `GET` | `/v1/dashboard/summary` | Dashboard metric aggregate payload |
| **Health** | `GET` | `/healthz`, `/readyz` | Liveness and database connectivity probes |

*The full formal specification lives in [`Ledger_API_Contract.pdf`](Ledger_API_Contract.pdf).*

---

## Parser Extensibility

The core pipeline follows the **Open/Closed Principle**. New financial institutions, SMS schemas, or regional formats can be integrated by implementing the `Parser` interface:

```go
// internal/parsers/parser.go
type Parser interface {
    // SenderIDs returns the SMS originator headers (e.g., "CBE", "Telebirr", "8329")
    SenderIDs() []string

    // Match validates whether the message body matches this specific bank format
    Match(msg domain.RawMessage) bool

    // Parse extracts raw text into a normalized, domain-level transaction entity
    Parse(msg domain.RawMessage) (domain.ParsedTransaction, error)
}
```

> **Reliability Guarantee:** Messages from authorized financial senders that cannot be deterministically parsed are persisted to a dead-letter `needs_review` state (FR-3.3) for review and parser updates — no data is dropped.

---

## Configuration

All configuration is environment-based; see the documented `.env.example`. No secrets or credentials are hardcoded anywhere in the codebase.

---

## Design Decisions

All architectural trade-offs, invariant models, and implementation boundaries are logged in [`DECISIONS.md`](DECISIONS.md):

- **D-01:** Server-side vs. on-device parsing trade-offs.
- **D-04:** Device pairing and key authorization scheme.
- **D-09:** Minimal zero-dependency web frontend using server-rendered templates and SVG graphics.
- **D-14:** Idempotency enforcement strategy across lossy mobile network boundaries.

---

## Known Gaps & Manual Prerequisites

Before taking this deployment to production:
- [ ] Supply a production LLM API key in `.env` for narrative report synthesis and edge-case classification.
- [ ] Ingest additional production SMS samples across varying telecom notification templates.
- [ ] Configure physical device keystore signing for release `.apk` builds.
- [ ] Terminate TLS via reverse proxy (Traefik/Nginx) for secure production ingestion.

---

