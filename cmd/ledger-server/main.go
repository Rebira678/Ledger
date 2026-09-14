// Command ledger-server runs the Ledger HTTP API + dashboard.
package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abel-gezahegn/ledger/internal/agent"
	"github.com/abel-gezahegn/ledger/internal/agent/llm"
	"github.com/abel-gezahegn/ledger/internal/api"
	"github.com/abel-gezahegn/ledger/internal/auth"
	"github.com/abel-gezahegn/ledger/internal/categorize"
	"github.com/abel-gezahegn/ledger/internal/config"
	"github.com/abel-gezahegn/ledger/internal/domain"
	"github.com/abel-gezahegn/ledger/internal/httpx"
	"github.com/abel-gezahegn/ledger/internal/loggerx"
	"github.com/abel-gezahegn/ledger/internal/parsers"
	"github.com/abel-gezahegn/ledger/internal/repo"
)

func main() {
	if err := run(); err != nil {
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := loggerx.New(cfg.Env)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// Database + embedded migrations (D-02, D-03).
	db, err := repo.OpenDB(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database open failed", "err", err)
		return err
	}
	defer db.Close()
	if err := repo.Migrate(cfg.DatabaseURL); err != nil {
		log.Error("migrations failed", "err", err)
		return err
	}
	log.Info("database ready", "migrations", "applied")

	r := repo.New(db)

	// Auth.
	tokenizer := auth.NewTokenizer(cfg.JWTSigningKey, cfg.AccessTokenTTL, cfg.RefreshTokenTTL)

	// Parsers (FR-3) — register all format plugins here.
	registry := parsers.NewRegistry(parsers.CBEParser{}, parsers.TelebirrParser{})

	// Agent (FR-5) + categorization (FR-4).
	llmClient := llm.New(cfg.LLMAPIKey, cfg.LLMBaseURL, cfg.LLMModel, cfg.LLMTimeout)
	engine := categorize.NewEngine(r.Rules)
	orch := agent.New(engine, llmClient)

	srv := &api.Server{
		Repo: r, Tokens: tokenizer, Parser: registry,
		Env: cfg.Env,
		Cfg: api.Config{
			MaxStatementBytes: cfg.MaxStatementBytes,
			RateIngestPerMin:  cfg.RateLimitIngestPerMin,
			RateUserPerMin:    cfg.RateLimitUserPerMin,
			RefreshTTLSeconds: int64(cfg.RefreshTokenTTL.Seconds()),
		},
	}
	// Wire the orchestrator into ingestion + correction endpoints (FR-4/FR-5).
	srv.CategorizeAndQueue = func(req *http.Request, txn *domain.Transaction) {
		if err := orch.Categorize(req.Context(), txn.UserID, txn); err != nil {
			loggerx.FromContext(req.Context(), log).Error("categorization failed", "err", err)
		}
	}
	srv.QueueClarification = func(req *http.Request, txn *domain.Transaction) {
		if err := orch.QueueClarification(req.Context(), agent.Stores{
			Rules: r.Rules, Clarifs: r.Clarifs,
		}, txn.UserID, txn); err != nil {
			loggerx.FromContext(req.Context(), log).Error("clarification queueing failed", "err", err)
		}
	}
	srv.ApplyCorrection = func(req *http.Request, userID, txnID, category string) bool {
		txn, err := r.Txns.GetTxn(req.Context(), userID, txnID)
		if err != nil {
			loggerx.FromContext(req.Context(), log).Error("correction: txn lookup failed", "err", err)
			return false
		}
		created, err := orch.ApplyCorrection(req.Context(), r.Rules, userID, txn.Counterparty, category)
		if err != nil {
			loggerx.FromContext(req.Context(), log).Error("correction rule persist failed", "err", err)
			return false
		}
		return created
	}

	// API + dashboard on the same mux with shared middleware.
	rootMux := http.NewServeMux()
	rootMux.HandleFunc("GET /healthz", srv.HandleLivez)
	rootMux.HandleFunc("GET /readyz", srv.HandleReadyz)
	rootMux.Handle("/v1/", srv.Routes(tokenizer))
	rootMux.Handle("/", srv.DashboardRoutes())

	root := httpx.Chain(rootMux,
		httpx.Correlate,
		httpx.RequestLogger(loggerx.NewAdapter(log)),
		httpx.Recover(loggerx.NewAdapter(log)),
	)

	httpServer := &http.Server{
		Addr: cfg.HTTPAddr, Handler: root,
		ReadHeaderTimeout: 5 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Info("ledger server listening", "addr", cfg.HTTPAddr, "env", cfg.Env)
		errCh <- httpServer.ListenAndServe()
	}()

	// Graceful shutdown (engineering bar).
	select {
	case <-ctx.Done():
		log.Info("shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Error("graceful shutdown failed", "err", err)
		}
		return nil
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("server failed", "err", err)
			return err
		}
		return nil
	}
}
