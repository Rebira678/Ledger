package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/rebira678/ledger/internal/domain"
)

// ReportRepo persists weekly reports (FR-6).
type ReportRepo struct{ db DB }

// ReportInput carries the generated report content for persistence.
type ReportInput struct {
	ID          string
	UserID      string
	PeriodStart string // YYYY-MM-DD
	PeriodEnd   string
	Totals      map[string]float64
	Wow         map[string]string
	Narrative   string
	Status      string // completed | llm_skipped | failed
}

// UpsertReport stores or replaces the report for (user, period).
func (r *ReportRepo) UpsertReport(ctx context.Context, in ReportInput) error {
	totalsJSON, err := json.Marshal(in.Totals)
	if err != nil {
		return fmt.Errorf("marshalling report totals: %w", err)
	}
	wowJSON, err := json.Marshal(in.Wow)
	if err != nil {
		return fmt.Errorf("marshalling week-over-week: %w", err)
	}
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO weekly_reports
		 (id, user_id, period_start, period_end, totals_by_category, week_over_week, narrative, generation_status, generated_at)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8, now())
		 ON CONFLICT (user_id, period_start) DO UPDATE SET
		   totals_by_category = EXCLUDED.totals_by_category,
		   week_over_week = EXCLUDED.week_over_week,
		   narrative = EXCLUDED.narrative,
		   generation_status = EXCLUDED.generation_status,
		   generated_at = now()`,
		in.ID, in.UserID, in.PeriodStart, in.PeriodEnd, totalsJSON, wowJSON, in.Narrative, in.Status); err != nil {
		return wrap("upserting weekly report", mapPgError(err))
	}
	return nil
}

// reportCols is the shared projection for report queries.
const reportCols = `id, user_id, period_start, period_end, totals_by_category, week_over_week,
 coalesce(narrative,''), generation_status, generated_at, created_at`

type reportRow struct {
	ID          string
	UserID      string
	Start       string
	End         string
	Totals      []byte
	Wow         []byte
	Narrative   string
	Status      string
	GeneratedAt sql.NullTime
	CreatedAt   sql.NullTime
}

func scanReport(sc interface{ Scan(...any) error }) (*domain.WeeklyReport, error) {
	var row reportRow
	if err := sc.Scan(&row.ID, &row.UserID, &row.Start, &row.End, &row.Totals, &row.Wow, &row.Narrative, &row.Status, &row.GeneratedAt, &row.CreatedAt); err != nil {
		return nil, err
	}
	out := &domain.WeeklyReport{
		ID:               row.ID,
		UserID:           row.UserID,
		Narrative:        row.Narrative,
		GenerationStatus: row.Status,
	}
	if row.GeneratedAt.Valid {
		out.GeneratedAt = row.GeneratedAt.Time
	}
	if row.CreatedAt.Valid {
		out.CreatedAt = row.CreatedAt.Time
	}
	if err := json.Unmarshal(row.Totals, &out.TotalsByCategory); err != nil {
		return nil, fmt.Errorf("unmarshalling report totals: %w", err)
	}
	if err := json.Unmarshal(row.Wow, &out.WeekOverWeek); err != nil {
		return nil, fmt.Errorf("unmarshalling week-over-week: %w", err)
	}
	return out, nil
}

// ListReports returns the user's reports, newest first.
func (r *ReportRepo) ListReports(ctx context.Context, userID string, limit int) ([]*domain.WeeklyReport, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT `+reportCols+` FROM weekly_reports WHERE user_id = $1
		 ORDER BY period_start DESC LIMIT $2`, userID, limit)
	if err != nil {
		return nil, wrap("listing weekly reports", err)
	}
	defer rows.Close()
	var out []*domain.WeeklyReport
	for rows.Next() {
		rep, err := scanReport(rows)
		if err != nil {
			return nil, wrap("scanning weekly report", err)
		}
		out = append(out, rep)
	}
	return out, rows.Err()
}

// GetReport returns one report scoped by owner.
func (r *ReportRepo) GetReport(ctx context.Context, userID, reportID string) (*domain.WeeklyReport, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+reportCols+` FROM weekly_reports WHERE id = $1 AND user_id = $2`, reportID, userID)
	rep, err := scanReport(row)
	if err != nil {
		return nil, notFound("getting weekly report", err)
	}
	return rep, nil
}

// SumByCategoryForPeriod returns per-category totals of debit transactions in
// [from, to) for a user — the grounding data for reports and anomaly detection.
func (r *ReportRepo) SumByCategoryForPeriod(ctx context.Context, userID, from, to string) (map[string]float64, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT coalesce(category,'Uncategorized') AS cat, sum(amount) AS total
		 FROM transactions
		 WHERE user_id = $1 AND direction = 'debit' AND occurred_at >= $2::timestamptz AND occurred_at < $3::timestamptz
		 GROUP BY cat ORDER BY total DESC`, userID, from, to)
	if err != nil {
		return nil, wrap("summing transactions by category", err)
	}
	defer rows.Close()
	out := map[string]float64{}
	for rows.Next() {
		var cat string
		var total float64
		if err := rows.Scan(&cat, &total); err != nil {
			return nil, wrap("scanning category sum", err)
		}
		out[cat] = total
	}
	return out, rows.Err()
}

// EstimateBalance returns (credits - debits) across all history for a user.
func (r *ReportRepo) EstimateBalance(ctx context.Context, userID string) (float64, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT coalesce(sum(CASE direction WHEN 'credit' THEN amount ELSE -amount END), 0)
		 FROM transactions WHERE user_id = $1`, userID)
	var bal float64
	if err := row.Scan(&bal); err != nil {
		return 0, wrap("estimating balance", err)
	}
	return bal, nil
}

var _ = errors.Is
