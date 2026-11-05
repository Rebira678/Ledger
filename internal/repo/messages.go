package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/rebira678/ledger/internal/domain"
)

// MessageRepo persists raw ingested SMS messages (idempotency + reprocessing).
type MessageRepo struct{ db DB }

// InsertRawMessageResult reports whether a new row was created.
type InsertRawMessageResult struct {
	Created        bool
	ID             string
	ExistingStatus string
	ExistingTxnID  string
}

// InsertRawMessage inserts the raw message; if the (device_id, client_message_id)
// pair already exists it returns Created=false along with the existing id, status,
// and transaction_id — allowing callers to distinguish between completed duplicates
// (409) and retries of failed/incomplete messages that should be reprocessed.
func (r *MessageRepo) InsertRawMessage(ctx context.Context, m *domain.RawMessage) (InsertRawMessageResult, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO raw_messages
		 (id, user_id, device_id, client_message_id, sender_id, body, received_at, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		 ON CONFLICT (device_id, client_message_id) DO NOTHING
		 RETURNING id`,
		m.ID, m.UserID, m.DeviceID, m.ClientMessageID, m.SenderID, m.Body, m.ReceivedAt, m.Status)
	var id string
	err := row.Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		// Conflict: fetch existing id, status, and transaction_id for idempotency/retry handling.
		var existingID, status string
		var txnID sql.NullString
		qerr := r.db.QueryRowContext(ctx,
			`SELECT id, status, COALESCE(transaction_id, '') FROM raw_messages WHERE device_id = $1 AND client_message_id = $2`,
			m.DeviceID, m.ClientMessageID).Scan(&existingID, &status, &txnID)
		if qerr != nil {
			return InsertRawMessageResult{}, wrap("looking up duplicate raw message", qerr)
		}
		return InsertRawMessageResult{
			Created:        false,
			ID:             existingID,
			ExistingStatus: status,
			ExistingTxnID:  txnID.String,
		}, nil
	}
	if err != nil {
		return InsertRawMessageResult{}, wrap("inserting raw message", mapPgError(err))
	}
	return InsertRawMessageResult{Created: true, ID: id}, nil
}

// SetRawMessageOutcome links a raw message to its parsed transaction or failure.
func (r *MessageRepo) SetRawMessageOutcome(ctx context.Context, id, status, txnID, parseErr string) error {
	if _, err := r.db.ExecContext(ctx,
		`UPDATE raw_messages SET status = $2, transaction_id = NULLIF($3,''), parse_error = NULLIF($4,'')
		 WHERE id = $1`, id, status, txnID, parseErr); err != nil {
		return wrap("updating raw message outcome", err)
	}
	return nil
}

// CountRecentByDevice counts messages ingested by a device since a time — used by rate limiting tests.
func (r *MessageRepo) CountRecentByDevice(ctx context.Context, deviceID string, since time.Time) (int, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT count(*) FROM raw_messages WHERE device_id = $1 AND created_at >= $2`,
		deviceID, since)
	var n int
	if err := row.Scan(&n); err != nil {
		return 0, wrap("counting recent messages", err)
	}
	return n, nil
}
