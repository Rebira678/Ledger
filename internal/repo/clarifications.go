package repo

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"

	"github.com/rebira678/ledger/internal/domain"
)

// ClarifRepo persists agent clarifying questions (FR-5.1/5.2).
type ClarifRepo struct{ db DB }

// CreateClarification queues a new question.
func (r *ClarifRepo) CreateClarification(ctx context.Context, c *domain.Clarification) error {
	optsJSON, err := json.Marshal(c.Options)
	if err != nil {
		return wrap("marshalling clarification options", err)
	}
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO clarifications (id, user_id, transaction_id, question, options)
		 VALUES ($1,$2,$3,$4,$5)`,
		c.ID, c.UserID, c.TransactionID, c.Question, optsJSON); err != nil {
		return wrap("creating clarification", mapPgError(err))
	}
	return nil
}

// ListPending returns pending clarifications for a user.
func (r *ClarifRepo) ListPending(ctx context.Context, userID string) ([]*domain.Clarification, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, transaction_id, question, options, status,
		        coalesce(answer_category,''), answered_at, created_at
		 FROM clarifications WHERE user_id = $1 AND status = 'pending'
		 ORDER BY created_at`, userID)
	if err != nil {
		return nil, wrap("listing pending clarifications", err)
	}
	defer rows.Close()
	var out []*domain.Clarification
	for rows.Next() {
		c := &domain.Clarification{}
		var answered sql.NullTime
		var optsJSON []byte
		if err := rows.Scan(&c.ID, &c.UserID, &c.TransactionID, &c.Question, &optsJSON, &c.Status, &c.AnswerCategory, &answered, &c.CreatedAt); err != nil {
			return nil, wrap("scanning clarification", err)
		}
		if err := json.Unmarshal(optsJSON, &c.Options); err != nil {
			return nil, wrap("unmarshalling clarification options", err)
		}
		if answered.Valid {
			c.AnsweredAt = answered.Time
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ResolveClarification applies the user's answer, returning whether the row existed.
func (r *ClarifRepo) ResolveClarification(ctx context.Context, userID, clarifID, answer string) (bool, error) {
	cmd, err := r.db.ExecContext(ctx,
		`UPDATE clarifications
		 SET status = 'resolved', answer_category = $3, answered_at = now()
		 WHERE id = $2 AND user_id = $1 AND status = 'pending'`,
		userID, clarifID, answer)
	if err != nil {
		return false, wrap("resolving clarification", err)
	}
	n, _ := cmd.RowsAffected()
	return n > 0, nil
}

// HasPendingForTransaction prevents duplicate questions for the same txn.
func (r *ClarifRepo) HasPendingForTransaction(ctx context.Context, userID, txnID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx,
		`SELECT EXISTS(SELECT 1 FROM clarifications WHERE user_id = $1 AND transaction_id = $2 AND status = 'pending')`,
		userID, txnID).Scan(&exists)
	if err != nil {
		return false, wrap("checking pending clarification", err)
	}
	return exists, nil
}

var _ = errors.Is
