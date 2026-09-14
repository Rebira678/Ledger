// Package repo implements the repository layer: the ONLY place in the codebase
// that talks SQL. Every tenant-scoped query is scoped by user_id at this layer
// (multi-tenant isolation enforced below the handler, per the engineering bar).
package repo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"github.com/abel-gezahegn/ledger/internal/domain"
)

// ErrNotFound is re-exported for handler mapping.
var ErrNotFound = domain.ErrNotFound

// DB is the common interface the repos need (pgx stdlib connection/tx).
type DB interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// mapPgError converts Postgres constraint violations into domain errors.
func mapPgError(err error) error {
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch pgErr.Code {
		case "23505": // unique_violation
			return fmt.Errorf("%w: %s", domain.ErrAlreadyExists, pgErr.ConstraintName)
		case "23503": // foreign_key_violation
			return fmt.Errorf("%w: %s", domain.ErrConflict, pgErr.ConstraintName)
		}
	}
	return err
}

func wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", op, err)
}

// Repo aggregates all repositories over one DB handle.
type Repo struct {
	Users    *UserRepo
	Devices  *DeviceRepo
	Tokens   *TokenRepo
	Txns     *TxnRepo
	Messages *MessageRepo
	Uploads  *UploadRepo
	Rules    *RuleRepo
	Clarifs  *ClarifRepo
	Reports  *ReportRepo
}

// New builds the aggregate repository.
func New(db DB) *Repo {
	return &Repo{
		Users:    &UserRepo{db: db},
		Devices:  &DeviceRepo{db: db},
		Tokens:   &TokenRepo{db: db},
		Txns:     &TxnRepo{db: db},
		Messages: &MessageRepo{db: db},
		Uploads:  &UploadRepo{db: db},
		Rules:    &RuleRepo{db: db},
		Clarifs:  &ClarifRepo{db: db},
		Reports:  &ReportRepo{db: db},
	}
}

// notFound converts sql.ErrNoRows to domain.ErrNotFound with context.
func notFound(op string, err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return fmt.Errorf("%s: %w", op, domain.ErrNotFound)
	}
	return err
}
