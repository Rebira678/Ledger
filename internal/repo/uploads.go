package repo

import (
	"context"

	"github.com/abel-gezahegn/ledger/internal/domain"
)

// UploadRepo persists statement uploads (FR-7).
type UploadRepo struct{ db DB }

// CreateUpload stores an upload record including retained file bytes (FR-2.4-style reprocessing).
func (r *UploadRepo) CreateUpload(ctx context.Context, u *domain.StatementUpload, fileBytes []byte) (*domain.StatementUpload, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO statement_uploads
		 (id, user_id, bank_hint, mime_type, file_bytes, size_bytes, status)
		 VALUES ($1,$2,$3,$4,$5,$6,$7)
		 RETURNING created_at`,
		u.ID, u.UserID, u.BankHint, u.MimeType, fileBytes, u.SizeBytes, u.Status)
	if err := row.Scan(&u.CreatedAt); err != nil {
		return nil, wrap("creating upload", mapPgError(err))
	}
	return u, nil
}

// GetUpload returns an upload scoped to the owning user (isolation-critical).
func (r *UploadRepo) GetUpload(ctx context.Context, userID, uploadID string) (*domain.StatementUpload, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, coalesce(bank_hint,''), mime_type, size_bytes, status,
		        coalesce(error,''), transactions_created, transactions_flagged, created_at
		 FROM statement_uploads WHERE id = $1 AND user_id = $2`, uploadID, userID)
	var u domain.StatementUpload
	err := row.Scan(&u.ID, &u.UserID, &u.BankHint, &u.MimeType, &u.SizeBytes, &u.Status,
		&u.Error, &u.TxCreated, &u.TxFlagged, &u.CreatedAt)
	if err != nil {
		return nil, notFound("getting upload", err)
	}
	return &u, nil
}

// GetUploadBytes loads retained file bytes for processing/reprocessing.
func (r *UploadRepo) GetUploadBytes(ctx context.Context, uploadID string) ([]byte, error) {
	var b []byte
	if err := r.db.QueryRowContext(ctx,
		`SELECT file_bytes FROM statement_uploads WHERE id = $1`, uploadID).Scan(&b); err != nil {
		return nil, notFound("getting upload bytes", err)
	}
	return b, nil
}

// SetUploadStatus transitions an upload's processing status.
func (r *UploadRepo) SetUploadStatus(ctx context.Context, uploadID, status, errText string, created, flagged int) error {
	if _, err := r.db.ExecContext(ctx,
		`UPDATE statement_uploads
		 SET status = $2, error = NULLIF($3,''),
		     transactions_created = $4, transactions_flagged = $5
		 WHERE id = $1`, uploadID, status, errText, created, flagged); err != nil {
		return wrap("updating upload status", err)
	}
	return nil
}
