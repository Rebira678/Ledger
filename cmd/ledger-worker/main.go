// Command ledger-worker runs the scheduled weekly report worker.
package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/abel-gezahegn/ledger/internal/agent"
	"github.com/abel-gezahegn/ledger/internal/agent/llm"
	"github.com/abel-gezahegn/ledger/internal/categorize"
	"github.com/abel-gezahegn/ledger/internal/config"
	"github.com/abel-gezahegn/ledger/internal/loggerx"
	"github.com/abel-gezahegn/ledger/internal/repo"
	"github.com/abel-gezahegn/ledger/internal/reports"
	"github.com/abel-gezahegn/ledger/internal/worker"
)

func main() {
	if err := run(); err != nil && err != context.Canceled {
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
	r := repo.New(db)

	llmClient := llm.New(cfg.LLMAPIKey, cfg.LLMBaseURL, cfg.LLMModel, cfg.LLMTimeout)
	engine := categorize.NewEngine(r.Rules)
	orch := agent.New(engine, llmClient)
	gen := reports.New(r, orch, log)

	w := &worker.WeeklyReportWorker{
		Repo: r, Gen: gen, Log: log,
		CronSpec: cfg.ReportCron, Timezone: cfg.ReportTimezone,
	}
	log.Info("ledger worker starting", "cron", cfg.ReportCron, "tz", cfg.ReportTimezone)
	return w.Start(ctx)
}
