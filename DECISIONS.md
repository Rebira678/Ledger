# Ledger — Engineering Decisions

This file records every decision that was left open by the SRS, PRD, or API Contract,
with rationale. It is updated at the end of each build phase.
Format: D-NN | Phase | Decision | Rationale | Consequences / follow-ups.

---

- **D-01 | Phase 0 | HTTP routing: Go 1.22+ standard-library `net/http` ServeMux** (method+wildcard patterns), no third-party router.
  *Rationale:* zero dependencies, supports `GET /v1/transactions/{id}` patterns natively, trivially auditable for a security-sensitive codebase; the API surface (~15 routes) doesn't justify chi/echo.
  *Consequences:* no middleware ecosystem — a small internal middleware chain is implemented in `internal/httpx`.

- **D-02 | Phase 0 | Data access: `pgx` v5 stdlib mode (`database/sql` driver) with explicit, hand-written parameterized SQL in a repository layer.**
  *Rationale:* sqlc adds a codegen step to the build; hand-written SQL in one layer is easy to review for the multi-tenant `user_id` scoping requirement, and pgx gives native `numeric`/`timestamptz` handling.
  *Consequences:* every query must be reviewed for parameterization and tenant scoping; enforced by repo-layer tests.

- **D-03 | Phase 0 | Migrations: `golang-migrate/migrate` v4** as a Go library, embedded via `source/iofs` from `migrations/*.up.sql` / `*.down.sql`, run automatically at server startup.
  *Rationale:* versioned, reversible, committed to the repo as required; library embedding means a single `docker compose up` brings up a fully migrated stack.

- **D-04 | Phase 0 | Password hashing: golang.org/x/crypto/bcrypt** (cost 12). Refresh tokens are 256-bit random values stored hashed (SHA-256) server-side and returned in plaintext once to the client.

- **D-05 | Phase 0 | Money: `numeric(14,2)` in Postgres, scanned as `pgtype.Numeric`/float64-free decimal strings in Go.** Amounts are always positive; `direction` carries the sign semantics per the API contract.
  *Consequences:* JSON responses emit `500.00`-style numbers; no float arithmetic is used for money beyond display rounding from the decimal value.

- **D-06 | Phase 0 | Logging: `log/slog` (stdlib structured logging)** with request-scoped correlation IDs propagated via `context.Context`. No third-party logger.

- **D-07 | Phase 0 | LLM provider: OpenAI-compatible Chat Completions API**, configured by `LEDGER_LLM_BASE_URL` / `LEDGER_LLM_API_KEY` / `LEDGER_LLM_MODEL`.
  *Rationale:* the prompt requires calling the *real provider API*; an OpenAI-compatible client covers OpenAI and any compatible endpoint. The integration point is complete and real; a live key is required to validate end-to-end (see Known-Gaps).
  *Consequences:* with no key configured the system degrades explicitly (report generation records `llm_skipped` status) — it never fabricates a narrative.

- **D-08 | Phase 0 | IDs: prefixed, time-ordered identifiers** (`txn_`, `usr_`, `dev_`, `up_`, `clr_`, `rpt_`) generated server-side, matching the API contract examples.

- **D-09 | Phase 0 | Dashboard: server-rendered Go `html/template`** instead of a React SPA.
  *Rationale:* the full production bar (tests, CI, isolation, idempotency) is deliverable faster and more verifiably with server-rendered templates in the same module; zero JS build pipeline, session auth is simple, and the dashboard is read-mostly.
  *Consequences:* charts are rendered as inline SVG from server data (no JS chart lib).

- **D-10 | Phase 2 | Idempotency: unique index on `(device_id, client_message_id)`** at the database layer, checked with `ON CONFLICT DO NOTHING` + `SELECT` on re-submit → `409 DUPLICATE` per the API contract, transactionally safe under concurrent retries.

- **D-11 | Phase 2 | Rate limiting: in-process token bucket** keyed by device key (ingestion, 120/min) or user (standard, 60/min), returning `429` + `Retry-After` per contract. Single-instance assumption recorded; a Redis-backed limiter is the documented scale-out path.

- **D-12 | Phase 3 | Parser accuracy caveat:** CBE/Telebirr parser fixtures are written from the publicly documented formats of these banks and *representative synthetic samples clearly labeled as such*; real (redacted) captured SMS samples from each bank are required to validate accuracy end-to-end (per the master prompt: never fabricate samples and label them real).

- **D-13 | Phase 4 | Categorization: deterministic rule engine** (counterparty match → learned user rules → keyword rules → default), each emitting a confidence score. Clarifying questions are queued when confidence < 0.60.

- **D-14 | Phase 5 | Android `minSdk 26`, `targetSdk 34`, Kotlin, `WorkManager` for queue/retry, BroadcastReceiver holding *no* network logic — it enqueues and exits.**

- **D-15 | Phase 6 | Anomaly detection: week-over-week category spend change > ±35% AND > ETB 200 absolute** → flagged into the weekly report narrative inputs.

- **D-16 | Phase 7 | Weekly report worker: in-process scheduler** (ticker + cron-style spec from env) rather than a separate binary/deployment; batched, parallel per-user report generation.

- **D-17 | Phase 7 | Email delivery of weekly reports is deferred** (PRD open question resolved: dashboard-only for v1); the report worker exposes the hook point.

- **D-18 | Phase 8 | Review queue for unmatched SMS is engineer-only for v1** (PRD open question resolved): unmatched messages are persisted with `status='needs_review'` and visible via server logs/SQL, not the user dashboard.

- **D-19 | Phase 0 | Testing: `testcontainers-go` spins up a real PostgreSQL** for repository and API integration tests — the master prompt requires real-database integration tests, not mocks.

- **D-20 | Phase 0 | Session auth for the dashboard uses the same JWT access tokens as the API** (cookie-based for browser flows), avoiding a parallel auth system.

- **D-21 | Phase 2 | Clarification options are stored as JSONB**, not `text[]`, for portable scanning through `database/sql`.
- **D-22 | Phase 2 | Clarification queueing runs AFTER the transaction row is persisted** (split into `Categorize` → insert → `QueueClarification`), because clarifications hold an FK to transactions.
- **D-23 | Phase 2 | Rate-limit keys**: authenticated endpoints key on the raw bearer token (limiter runs before auth resolves identity); ingestion keys on the device key. Per-user identity keying is the documented scale-out refinement.
- **D-24 | Phase 8 | Dashboard charts are server-generated inline SVG** (no JS chart library), matching the server-rendered choice (D-09).

## Known Gaps — require human action before end-to-end validation

1. **LLM API key** (`LEDGER_LLM_API_KEY`): the narrative/clarification LLM integration is implemented against the real OpenAI-compatible API but needs a live key to validate end-to-end.
2. **Real captured SMS samples** for CBE, Telebirr (and any additional bank): required to confirm parser accuracy against real on-device formats (D-12).
3. **Physical Android device**: SMS permission prompts, receiver timing, and battery-optimization behavior require real-device verification.
4. **APK signing key**: release signing requires a human-generated keystore; debug-signed build is produced for testing.
5. **Production TLS/domain**: served by reverse proxy in production; not provisioned here.
