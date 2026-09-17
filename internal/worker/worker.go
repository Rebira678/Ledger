// Package worker runs the scheduled weekly report worker (FR-6.1): every week
// per the cron spec, generate reports for all users in parallel batches (5.4 scalability).
package worker

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/abel-gezahegn/ledger/internal/repo"
	"github.com/abel-gezahegn/ledger/internal/reports"
)

// WeeklyReportWorker schedules weekly report generation.
type WeeklyReportWorker struct {
	Repo *repo.Repo
	Gen  *reports.Generator
	Log  *slog.Logger
	// CronSpec is "minute hour dom month dow" (5-field); we support the
	// common "0 6 * * 1" shape — minute/hour/dow with * for dom/month.
	CronSpec  string
	Timezone  string
	BatchSize int
}

// Start blocks until ctx is cancelled, running the scheduler loop.
func (w *WeeklyReportWorker) Start(ctx context.Context) error {
	loc, err := time.LoadLocation(w.Timezone)
	if err != nil {
		loc = time.UTC
		w.Log.Warn("invalid LEDGER_REPORT_TIMEZONE, falling back to UTC", "tz", w.Timezone)
	}
	minute, hour, dow, perr := parseCron(w.CronSpec)
	if perr != nil {
		w.Log.Warn("invalid LEDGER_REPORT_CRON, falling back to Monday 06:00", "spec", w.CronSpec, "err", perr)
		minute, hour, dow = 0, 6, 1
	}
	w.Log.Info("weekly report worker started", "cron", w.CronSpec, "tz", w.Timezone)

	// Compute next run instantly rather than polling every minute.
	for {
		now := time.Now().In(loc)
		next := nextRun(now, minute, hour, dow)
		timer := time.NewTimer(next.Sub(now))
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			w.runOnce(ctx)
		}
	}
}

// runOnce generates reports for all users in parallel batches.
func (w *WeeklyReportWorker) runOnce(ctx context.Context) {
	start := time.Now()
	weekStart := domainWeekStart(start)
	users, err := w.listUserIDs(ctx)
	if err != nil {
		w.Log.Error("weekly reports: listing users failed", "err", err)
		return
	}
	batch := w.BatchSize
	if batch <= 0 {
		batch = 20
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, batch)
	for _, uid := range users {
		wg.Add(1)
		sem <- struct{}{}
		go func(userID string) {
			defer wg.Done()
			defer func() { <-sem }()
			if err := w.Gen.GenerateForUser(ctx, userID, weekStart); err != nil {
				w.Log.Error("weekly report failed", "user_id", userID, "err", err)
			}
		}(uid)
	}
	wg.Wait()
	w.Log.Info("weekly report run finished", "users", len(users), "took", time.Since(start))
}

func (w *WeeklyReportWorker) listUserIDs(ctx context.Context) ([]string, error) {
	rows, err := w.Repo.Users.ListUserIDs(ctx)
	if err != nil {
		return nil, err
	}
	return rows, nil
}

// nextRun returns the next matching minute/hour/dow occurrence strictly after now.
func nextRun(now time.Time, minute, hour, dow int) time.Time {
	cand := now.Truncate(time.Minute).Add(time.Minute)
	for i := 0; i < 7*24*60; i++ { // at most a week ahead
		if int(cand.Weekday()) == dow && cand.Hour() == hour && cand.Minute() == minute {
			return cand
		}
		cand = cand.Add(time.Minute)
	}
	return now.Add(time.Hour) // unreachable fallback
}

// parseCron parses the supported 5-field subset.
func parseCron(spec string) (minute, hour, dow int, err error) {
	var parts []string
	for _, p := range splitCron(spec) {
		parts = append(parts, p)
	}
	if len(parts) != 5 {
		return 0, 0, 0, errCronInvalid
	}
	minute, err = cronField(parts[0], 0, 59)
	if err != nil {
		return 0, 0, 0, err
	}
	hour, err = cronField(parts[1], 0, 23)
	if err != nil {
		return 0, 0, 0, err
	}
	if parts[2] != "*" || parts[3] != "*" {
		return 0, 0, 0, errCronInvalid
	}
	dow, err = cronField(parts[4], 0, 6)
	if err != nil {
		return 0, 0, 0, err
	}
	return minute, hour, dow, nil
}

func cronField(s string, lo, hi int) (int, error) {
	if s == "*" {
		return lo, nil
	}
	n := 0
	if s == "" {
		return 0, errCronInvalid
	}
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, errCronInvalid
		}
		n = n*10 + int(c-'0')
	}
	if n < lo || n > hi {
		return 0, errCronInvalid
	}
	return n, nil
}

func splitCron(s string) []string {
	var out []string
	cur := ""
	for _, c := range s {
		if c == ' ' || c == '\t' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
			continue
		}
		cur += string(c)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

type cronError string

func (e cronError) Error() string { return string(e) }

const errCronInvalid = cronError("invalid cron spec")

// domainWeekStart re-exports the Monday-00:00 week start from domain.
func domainWeekStart(t time.Time) time.Time {
	t = t.UTC()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	return time.Date(t.Year(), t.Month(), t.Day()-(weekday-1), 0, 0, 0, 0, time.UTC)
}

// MondayWeekStart is the exported form used by utility scripts.
func MondayWeekStart(t time.Time) time.Time { return domainWeekStart(t) }
