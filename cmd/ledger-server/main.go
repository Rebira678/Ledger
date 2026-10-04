// Command ledger-server runs the Ledger HTTP API + dashboard.
package main

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/rebira678/ledger/internal/agent"
	"github.com/rebira678/ledger/internal/agent/llm"
	"github.com/rebira678/ledger/internal/api"
	"github.com/rebira678/ledger/internal/auth"
	"github.com/rebira678/ledger/internal/categorize"
	"github.com/rebira678/ledger/internal/config"
	"github.com/rebira678/ledger/internal/domain"
	"github.com/rebira678/ledger/internal/httpx"
	"github.com/rebira678/ledger/internal/loggerx"
	"github.com/rebira678/ledger/internal/mail"
	"github.com/rebira678/ledger/internal/parsers"
	"github.com/rebira678/ledger/internal/repo"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "startup failed: %v\n", err)
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

	// Mailer
	mailer := mail.NewSMTPSender(mail.Config{
		Host:     cfg.SMTPHost,
		Port:     cfg.SMTPPort,
		Username: cfg.SMTPUser,
		Password: cfg.SMTPPass,
		From:     cfg.SMTPFrom,
	})

	srv := &api.Server{
		Repo: r, Tokens: tokenizer, Parser: registry, Mailer: mailer,
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
	srv.ParseReceipt = func(ctx context.Context, mimeType string, imageBytes []byte) (*domain.Transaction, error) {
		importBase64 := base64.StdEncoding.EncodeToString(imageBytes)
		data, err := llmClient.ParseReceipt(ctx, importBase64, mimeType)
		if err != nil {
			return nil, err
		}

		occurredAt := time.Now()
		if t, err := time.Parse("2006-01-02", data.Date); err == nil {
			occurredAt = t
		}

		fmt.Printf("DEBUG LLM PARSED: %+v\n", data)

		money, _ := domain.ParseMoney(fmt.Sprintf("%.2f", data.Amount))
		if money.Units <= 0 && money.Cents <= 0 {
			return nil, errors.New("could not extract a valid amount greater than zero from the receipt image")
		}

		dir := domain.DirectionDebit
		if strings.ToLower(strings.TrimSpace(data.Direction)) == "credit" {
			dir = domain.DirectionCredit
		}

		currency := strings.TrimSpace(data.Currency)
		if currency == "" {
			currency = "ETB"
		}

		counterparty := strings.TrimSpace(data.Merchant)
		if counterparty == "" {
			counterparty = "Unknown"
		}

		return &domain.Transaction{
			Amount:       money,
			Currency:     currency,
			Direction:    dir,
			Counterparty: counterparty,
			Source:       domain.SourceStatement,
			OcurredAt:    occurredAt,
		}, nil
	}
	srv.ParseSMSFallback = func(ctx context.Context, senderID, body string) (*domain.ParsedTransaction, error) {
		return llmClient.ParseSMS(ctx, senderID, body)
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
