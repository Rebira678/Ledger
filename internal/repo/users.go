package repo

import (
	"context"
	"database/sql"
	"errors"

	"github.com/rebira678/ledger/internal/domain"
)

// UserRepo persists users.
type UserRepo struct{ db DB }

const userCols = `id, email, password_hash, display_name, avatar_url, created_at`

func scanUser(row *sql.Row) (*domain.User, string, error) {
	var (
		u    domain.User
		hash string
	)
	if err := row.Scan(&u.ID, &u.Email, &hash, &u.DisplayName, &u.AvatarURL, &u.CreatedAt); err != nil {
		return nil, "", err
	}
	return &u, hash, nil
}

// CreateUser inserts a new user with a bcrypt hash.
func (r *UserRepo) CreateUser(ctx context.Context, id, email, passwordHash string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)
		 RETURNING `+userCols,
		id, email, passwordHash)
	u, _, err := scanUser(row)
	if err != nil {
		return nil, wrap("creating user", mapPgError(err))
	}
	return u, nil
}

// UpdateUserProfile updates a user's display name and avatar URL.
func (r *UserRepo) UpdateUserProfile(ctx context.Context, id, displayName, avatarURL string) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET display_name = $1, avatar_url = $2 WHERE id = $3`,
		displayName, avatarURL, id)
	if err != nil {
		return wrap("updating user profile", mapPgError(err))
	}
	return nil
}

// GetByEmail returns the user and password hash for credential verification.
func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*domain.User, string, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT `+userCols+` FROM users WHERE email = $1`, email)
	u, hash, err := scanUser(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, "", nil // caller decides 401 semantics; avoids user-enumeration signal
		}
		return nil, "", wrap("getting user by email", err)
	}
	return u, hash, nil
}

// GetByID returns a user by id.
func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, email, display_name, avatar_url, created_at FROM users WHERE id = $1`, id)
	var u domain.User
	err := row.Scan(&u.ID, &u.Email, &u.DisplayName, &u.AvatarURL, &u.CreatedAt)
	if err != nil {
		return nil, notFound("getting user by id", err)
	}
	return &u, nil
}

// ListUserIDs returns all user ids — used only by the report worker (no tenant scope by design).
func (r *UserRepo) ListUserIDs(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id FROM users`)
	if err != nil {
		return nil, wrap("listing user ids", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, wrap("scanning user id", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

// DeviceRepo persists paired devices.
type DeviceRepo struct{ db DB }

// CreateDevice registers a paired device with the SHA-256 hash of its API key.
func (r *DeviceRepo) CreateDevice(ctx context.Context, d *domain.Device) (*domain.Device, error) {
	row := r.db.QueryRowContext(ctx,
		`INSERT INTO devices (id, user_id, device_model, os_version, api_key_hash)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING id, user_id, device_model, os_version, api_key_hash, created_at`,
		d.ID, d.UserID, d.DeviceModel, d.OSVersion, d.APIKeyHash)
	var out domain.Device
	err := row.Scan(&out.ID, &out.UserID, &out.DeviceModel, &out.OSVersion, &out.APIKeyHash, &out.CreatedAt)
	if err != nil {
		return nil, wrap("creating device", mapPgError(err))
	}
	return &out, nil
}

// GetUserByAPIKeyHash resolves a device + its owner from a device API key hash.
func (r *DeviceRepo) GetUserByAPIKeyHash(ctx context.Context, apiKeyHash string) (*domain.Device, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, device_model, os_version, api_key_hash, created_at
		 FROM devices WHERE api_key_hash = $1`, apiKeyHash)
	var d domain.Device
	err := row.Scan(&d.ID, &d.UserID, &d.DeviceModel, &d.OSVersion, &d.APIKeyHash, &d.CreatedAt)
	if err != nil {
		return nil, notFound("getting device by api key", err)
	}
	return &d, nil
}

// ListUserDevices returns all devices paired by a user.
func (r *DeviceRepo) ListUserDevices(ctx context.Context, userID string) ([]*domain.Device, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, device_model, os_version, api_key_hash, created_at
		 FROM devices WHERE user_id = $1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, wrap("listing devices", err)
	}
	defer rows.Close()
	var out []*domain.Device
	for rows.Next() {
		var d domain.Device
		if err := rows.Scan(&d.ID, &d.UserID, &d.DeviceModel, &d.OSVersion, &d.APIKeyHash, &d.CreatedAt); err != nil {
			return nil, wrap("scanning device", err)
		}
		out = append(out, &d)
	}
	return out, rows.Err()
}

var _ = errors.Is
