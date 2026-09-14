package repo

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/abel-gezahegn/ledger/internal/domain"
)

// TokenRepo persists opaque refresh tokens (stored hashed).
type TokenRepo struct{ db DB }

// CreateRefreshToken stores a hashed refresh token.
func (r *TokenRepo) CreateRefreshToken(ctx context.Context, tokenHash, userID string, expiresAt time.Time) error {
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (token_hash, user_id, expires_at) VALUES ($1, $2, $3)`,
		tokenHash, userID, expiresAt); err != nil {
		return wrap("creating refresh token", mapPgError(err))
	}
	return nil
}

// GetRefreshToken returns the token row if present and unexpired.
func (r *TokenRepo) GetRefreshToken(ctx context.Context, tokenHash string) (userID string, expiresAt time.Time, revoked bool, err error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT user_id, expires_at, revoked_at FROM refresh_tokens WHERE token_hash = $1`, tokenHash)
	var revokedAt sql.NullTime
	if err = row.Scan(&userID, &expiresAt, &revokedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", time.Time{}, false, domain.ErrNotFound
		}
		return "", time.Time{}, false, wrap("getting refresh token", err)
	}
	return userID, expiresAt, revokedAt.Valid, nil
}

// RevokeRefreshToken marks a token revoked (rotation).
func (r *TokenRepo) RevokeRefreshToken(ctx context.Context, tokenHash string) error {
	if _, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = now() WHERE token_hash = $1`, tokenHash); err != nil {
		return wrap("revoking refresh token", err)
	}
	return nil
}
