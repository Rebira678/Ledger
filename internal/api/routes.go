package api

import (
	"net/http"
	"time"

	"github.com/abel-gezahegn/ledger/internal/auth"
	"github.com/abel-gezahegn/ledger/internal/httpx"
)

// Routes builds the full v1 mux with middleware applied per contract §1.1/§1.4.
// Health endpoints are registered by the caller on the root mux (they are
// path-unversioned liveness/readiness probes).
func (s *Server) Routes(tokens *auth.Tokenizer) http.Handler {
	if s.Now == nil {
		s.Now = time.Now // struct-literal constructors (main.go) may skip New()
	}
	mux := http.NewServeMux()

	// Auth endpoints (no auth, IP rate-limited).
	authMux := http.NewServeMux()
	authMux.HandleFunc("POST /v1/auth/register", s.handleRegister)
	authMux.HandleFunc("POST /v1/auth/login", s.handleLogin)
	authMux.HandleFunc("POST /v1/auth/refresh", s.handleRefresh)
	authChainHandler := httpx.Chain(http.Handler(authMux),
		httpx.RateLimit(httpx.KeyByIP, s.Cfg.RateUserPerMin),
	)
	mux.Handle("/v1/auth/", authChainHandler)

	// Standard authenticated endpoints (per-user rate limit). The limiter runs
	// BEFORE auth, so key on the raw bearer token (unique per session).
	stdChain := func(h http.HandlerFunc) http.Handler {
		return httpx.RateLimit(httpx.KeyByUser, s.Cfg.RateUserPerMin)(RequireAuth(tokens)(h))
	}

	mux.Handle("POST /v1/devices/pair", stdChain(s.handlePair))
	mux.Handle("GET /v1/transactions", stdChain(s.handleListTxns))
	mux.Handle("PATCH /v1/transactions/{id}/category", stdChain(s.handlePatchCategory))
	mux.Handle("GET /v1/agent/clarifications", stdChain(s.handleListClarifications))
	mux.Handle("POST /v1/agent/clarifications/{id}/respond", stdChain(s.handleRespondClarification))
	mux.Handle("GET /v1/reports/weekly", stdChain(s.handleListReports))
	mux.Handle("GET /v1/reports/weekly/{report_id}", stdChain(s.handleGetReport))
	mux.Handle("GET /v1/dashboard/summary", stdChain(s.handleDashboardSummary))

	// Ingestion: JWT + device key, per-device rate limit (contract §1.4).
	ingestChain := func(h http.HandlerFunc) http.Handler {
		return httpx.Chain(
			RequireDeviceAuth(tokens, s.Repo.Devices)(h),
			httpx.RateLimit(httpx.KeyByDevice, s.Cfg.RateIngestPerMin),
		)
	}
	mux.Handle("POST /v1/ingest/sms", ingestChain(s.handleIngestSMS))
	mux.Handle("POST /v1/ingest/statement", stdChain(s.handleStatementUpload))
	mux.Handle("GET /v1/ingest/statement/{upload_id}", stdChain(s.handleStatementStatus))

	return mux
}
