// Package reports generates the weekly report for each user (FR-6):
// category breakdown, week-over-week comparison, and the LLM narrative
// grounded strictly in the user's transaction aggregates (FR-5.4).
package reports

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/rebira678/ledger/internal/agent"
	"github.com/rebira678/ledger/internal/agent/llm"
	"github.com/rebira678/ledger/internal/repo"
)

// Generator produces weekly reports.
type Generator struct {
	Repo *repo.Repo
	Orch *agent.Orchestrator
	Log  *slog.Logger
	Now  func() time.Time
}

// New builds a Generator.
func New(r *repo.Repo, orch *agent.Orchestrator, log *slog.Logger) *Generator {
	return &Generator{Repo: r, Orch: orch, Log: log, Now: time.Now}
}

// GenerateForUser builds and persists the report for [weekStart, weekStart+7d).
func (g *Generator) GenerateForUser(ctx context.Context, userID string, weekStart time.Time) error {
	weekEnd := weekStart.AddDate(0, 0, 7)
	from := weekStart.Format("2006-01-02T15:04:05Z")
	to := weekEnd.Format("2006-01-02T15:04:05Z")
	fromStr := weekStart.Format("2006-01-02")
	toStr := weekEnd.AddDate(0, 0, -1).Format("2006-01-02")

	totals, err := g.Repo.Reports.SumByCategoryForPeriod(ctx, userID, from, to)
	if err != nil {
		return fmt.Errorf("summing week totals: %w", err)
	}
	lastTotals, err := g.Repo.Reports.SumByCategoryForPeriod(ctx, userID,
		weekStart.AddDate(0, 0, -7).Format("2006-01-02T15:04:05Z"), from)
	if err != nil {
		return fmt.Errorf("summing previous week: %w", err)
	}

	// Week-over-week % change per category (FR-6.2).
	wow := map[string]float64{}
	for c, v := range totals {
		if last, ok := lastTotals[c]; ok && last > 0 {
			wow[c] = (v - last) / last * 100
		}
	}
	anomalies, err := g.Orch.DetectAnomalies(ctx, g.Repo.Reports, userID, weekStart)
	if err != nil {
		return fmt.Errorf("detecting anomalies: %w", err)
	}

	// LLM narrative — grounded strictly in this user's aggregates (FR-5.4).
	status := "completed"
	narrative := ""
	narr, err := g.Orch.WeeklyNarrative(ctx, agent.NarrativeInput{
		PeriodStart: fromStr, PeriodEnd: toStr,
		Totals: totals, WeekOverWeek: wow, Anomalies: anomalies,
	})
	switch {
	case err == nil:
		narrative = narr
	case err == llm.ErrNotConfigured || (err != nil && llm.ErrNotConfigured.Error() == err.Error()):
		// Explicit degradation (D-07): record llm_skipped, never fabricate.
		status = "llm_skipped"
		g.Log.InfoContext(ctx, "weekly report: llm not configured, narrative skipped",
			"user_id", userID)
	default:
		status = "failed"
		g.Log.ErrorContext(ctx, "weekly report: narrative generation failed",
			"user_id", userID, "err", err)
	}

	reportID := "rpt_" + fromStr + "_" + userID[:min(8, len(userID))]
	err = g.Repo.Reports.UpsertReport(ctx, repo.ReportInput{
		ID: reportID, UserID: userID,
		PeriodStart: fromStr, PeriodEnd: toStr,
		Totals:    totals,
		Wow:       wowToString(wow),
		Narrative: narrative,
		Status:    status,
	})
	if err != nil {
		return fmt.Errorf("persisting weekly report: %w", err)
	}
	g.Log.InfoContext(ctx, "weekly report generated",
		"user_id", userID, "period", fromStr, "status", status, "anomalies", len(anomalies))
	return nil
}

// wowToString converts numeric changes to contract-style "+35%" strings.
func wowToString(wow map[string]float64) map[string]string {
	out := make(map[string]string, len(wow))
	for c, v := range wow {
		sign := "+"
		if v < 0 {
			sign = ""
		}
		out[c] = fmt.Sprintf("%s%.0f%%", sign, v)
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
