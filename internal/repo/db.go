// db.go — Postgres connectivity + embedded versioned migrations.
package repo

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/abel-gezahegn/ledger/internal/domain"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// OpenDB opens the pgx stdlib pool with sane defaults.
func OpenDB(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)
	pingCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("pinging database: %w", err)
	}
	return db, nil
}

// Migrate applies all up-migrations from the embedded FS (D-03).
func Migrate(databaseURL string) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("loading embedded migrations: %w", err)
	}
	m, err := migrate.NewWithSourceInstance("iofs", src, "pgx5://"+stripScheme(databaseURL))
	if err != nil {
		return fmt.Errorf("creating migrator: %w", err)
	}
	defer m.Close()
	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("applying migrations: %w", err)
	}
	return nil
}

// stripScheme converts postgres://… into the host/path form golang-migrate's
// pgx5 driver expects. (migrate's registered name is "pgx5".)
func stripScheme(databaseURL string) string {
	for _, p := range []string{"postgres://", "postgresql://"} {
		if len(databaseURL) >= len(p) && databaseURL[:len(p)] == p {
			return databaseURL[len(p):]
		}
	}
	return databaseURL
}

var _ = domain.ErrNotFound
