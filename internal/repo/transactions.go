package repo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/rebira678/ledger/internal/domain"
)

// TxnRepo persists transactions. EVERY method takes userID — tenant isolation
// is enforced at this layer, not only in handlers (engineering bar).
type TxnRepo struct{ db DB }

// Ping verifies database connectivity for the readiness endpoint.
func (r *TxnRepo) Ping(ctx context.Context) error {
	row := r.db.QueryRowContext(ctx, `SELECT 1`)
	var one int
	if err := row.Scan(&one); err != nil {
		return wrap("pinging database", err)
	}
	return nil
}

// InsertTxn inserts a transaction within an optional transaction.
func (r *TxnRepo) InsertTxn(ctx context.Context, db DB, t *domain.Transaction) error {
	exec := r.db
	if db != nil {
		exec = db
	}
	row := exec.QueryRowContext(ctx,
		`INSERT INTO transactions
		 (id, user_id, device_id, amount, currency, direction, counterparty,
		  category, category_confidence, category_source, source, occurred_at)
		 VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,$7,$8,$9,$10,$11,$12)
		 RETURNING created_at`,
		t.ID, t.UserID, t.DeviceID, t.Amount.String(), t.Currency, string(t.Direction),
		t.Counterparty, t.Category, t.CategoryConfidence, t.CategorySource,
		string(t.Source), t.OcurredAt)
	if err := row.Scan(&t.CreatedAt); err != nil {
		return wrap("inserting transaction", mapPgError(err))
	}
	return nil
}

// TxnListFilter carries tenant-scoped list parameters.
type TxnListFilter struct {
	UserID   string
	From, To string // ISO dates, optional
	Category string
	Limit    int
	Cursor   string
}

// ListTxns returns a page of transactions plus the next cursor (or "" when done).
func (r *TxnRepo) ListTxns(ctx context.Context, f TxnListFilter) ([]*domain.Transaction, string, error) {
	where := []string{"user_id = $1"}
	args := []any{f.UserID}
	n := 2
	if f.From != "" {
		where = append(where, fmt.Sprintf("occurred_at >= $%d::timestamptz", n))
		n++
		args = append(args, f.From)
	}
	if f.To != "" {
		where = append(where, fmt.Sprintf("occurred_at < $%d::timestamptz", n))
		n++
		args = append(args, f.To)
	}
	if f.Category != "" {
		where = append(where, fmt.Sprintf("category = $%d", n))
		n++
		args = append(args, f.Category)
	}
	if f.Cursor != "" {
		where = append(where, fmt.Sprintf("(occurred_at, id) < ($%d::timestamptz, $%d)", n, n+1))
		n += 2
		args = append(args, cursorTime(f.Cursor), f.Cursor)
	}
	q := `SELECT id, user_id, coalesce(device_id,''), amount::text, currency, direction,
	             coalesce(counterparty,''), coalesce(category,''), category_confidence,
	             coalesce(category_source,''), source, occurred_at, created_at
	      FROM transactions
	      WHERE ` + strings.Join(where, " AND ") + `
	      ORDER BY occurred_at DESC, id DESC
	      LIMIT $` + fmt.Sprint(n)
	args = append(args, f.Limit+1)
	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, "", wrap("listing transactions", err)
	}
	defer rows.Close()

	var out []*domain.Transaction
	for rows.Next() {
		t := &domain.Transaction{}
		var amount string
		var conf sql.NullFloat64
		var src sql.NullString
		if err := rows.Scan(&t.ID, &t.UserID, &t.DeviceID, &amount, &t.Currency,
			&t.Direction, &t.Counterparty, &t.Category, &conf, &src, &t.Source,
			&t.OcurredAt, &t.CreatedAt); err != nil {
			return nil, "", wrap("scanning transaction row", err)
		}
		amt, perr := domain.ParseMoney(amount)
		if perr != nil {
			return nil, "", wrap("parsing amount from db", perr)
		}
		t.Amount = amt
		t.CategoryConfidence = conf.Float64
		t.CategoryHasConfidence = conf.Valid
		t.CategorySource = src.String
		out = append(out, t)
	}
	if err := rows.Err(); err != nil {
		return nil, "", wrap("iterating transactions", err)
	}
	next := ""
	if len(out) > f.Limit {
		last := out[f.Limit-1]
		next = encodeCursor(last.OcurredAt, last.ID)
		out = out[:f.Limit]
	}
	return out, next, nil
}

// GetTxn fetches one transaction, scoped by owner.
func (r *TxnRepo) GetTxn(ctx context.Context, userID, txnID string) (*domain.Transaction, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, coalesce(device_id,''), amount::text, currency, direction,
		        coalesce(counterparty,''), coalesce(category,''), category_confidence,
		        coalesce(category_source,''), source, occurred_at, created_at
		 FROM transactions WHERE id = $1 AND user_id = $2`, txnID, userID)
	t := &domain.Transaction{}
	var amount string
	var conf sql.NullFloat64
	var src sql.NullString
	if err := row.Scan(&t.ID, &t.UserID, &t.DeviceID, &amount, &t.Currency, &t.Direction,
		&t.Counterparty, &t.Category, &conf, &src, &t.Source, &t.OcurredAt, &t.CreatedAt); err != nil {
		return nil, notFound("getting transaction", err)
	}
	amt, err := domain.ParseMoney(amount)
	if err != nil {
		return nil, wrap("parsing amount from db", err)
	}
	t.Amount = amt
	t.CategoryConfidence = conf.Float64
	t.CategoryHasConfidence = conf.Valid
	t.CategorySource = src.String
	return t, nil
}

// UpdateCategory sets the category on a tenant-scoped transaction.
func (r *TxnRepo) UpdateCategory(ctx context.Context, userID, txnID, category, source string) error {
	cmd, err := r.db.ExecContext(ctx,
		`UPDATE transactions SET category = $1, category_source = $2, category_confidence = 1.0
		 WHERE id = $3 AND user_id = $4`,
		category, source, txnID, userID)
	if err != nil {
		return wrap("updating transaction category", err)
	}
	if n, _ := cmd.RowsAffected(); n == 0 {
		return fmt.Errorf("updating transaction category: %w", domain.ErrNotFound)
	}
	return nil
}

// cursorTime decodes the timestamp portion of a cursor; invalid cursors
// yield zero time and are rejected by the handler validation.
func cursorTime(c string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, c)
	if err != nil {
		return time.Time{}
	}
	return t
}

// encodeCursor builds an opaque next-page cursor from the last row.
func encodeCursor(t time.Time, id string) string {
	return t.UTC().Format(time.RFC3339Nano) + "|" + id
}
