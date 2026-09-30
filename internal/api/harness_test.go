package api_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/rebira678/ledger/internal/agent"
	"github.com/rebira678/ledger/internal/api"
	"github.com/rebira678/ledger/internal/domain"
	"github.com/rebira678/ledger/internal/repo"
)

// containerDSN2 starts a real Postgres container (or reuses env DSN) — real
// database per the engineering bar, no mocks.
func containerDSN2(ctx context.Context) string {
	if dsn := os.Getenv("LEDGER_TEST_DATABASE_URL"); dsn != "" {
		return dsn
	}
	req := testcontainers.ContainerRequest{
		Image:        "postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_USER":     "ledger",
			"POSTGRES_PASSWORD": "ledger",
			"POSTGRES_DB":       "ledger_test",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp").WithStartupTimeout(90 * time.Second),
	}
	c, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req, Started: true,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "starting postgres container: %v\n", err)
		os.Exit(2)
	}
	host, err := c.Host(ctx)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	port, err := c.MappedPort(ctx, "5432/tcp")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	return fmt.Sprintf("postgres://ledger:ledger@%s:%s/ledger_test?sslmode=disable", host, port.Port())
}

func startContainer(ctx context.Context, _ any) (string, error) { return containerDSN2(ctx), nil }

func testcontainersRequest() any { return nil }

func dropAll(t *testing.T) {
	t.Helper()
	// Wait for Postgres to accept connections (container startup race).
	for i := 0; i < 60; i++ {
		if err := testDB.Ping(); err == nil {
			break
		}
		time.Sleep(250 * time.Millisecond)
	}
	if _, err := testDB.Exec(`
		DROP SCHEMA IF EXISTS public CASCADE;
		CREATE SCHEMA public;
	`); err != nil {
		t.Fatalf("failed to reset schema: %v", err)
	}
}

func applyMigrations(t *testing.T) {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			break
		}
		dir = filepath.Dir(dir)
	}
	entries, err := filepath.Glob(filepath.Join(dir, "migrations", "*.up.sql"))
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(entries)
	for _, p := range entries {
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := testDB.Exec(string(b)); err != nil {
			t.Fatalf("applying %s: %v", p, err)
		}
	}
}

// newWired returns the server with the orchestrator wired exactly as main.go does.
func newWired(srv *api.Server, r *repo.Repo, orch *agent.Orchestrator) *api.Server {
	srv.CategorizeAndQueue = func(req *http.Request, txn *domain.Transaction) {
		if err := orch.Categorize(req.Context(), txn.UserID, txn); err != nil {
			// Categorization failure must not fail ingestion; logged in main wiring.
			_ = err
		}
	}
	srv.QueueClarification = func(req *http.Request, txn *domain.Transaction) {
		if err := orch.QueueClarification(req.Context(), agent.Stores{
			Rules: r.Rules, Clarifs: r.Clarifs,
		}, txn.UserID, txn); err != nil {
			_ = err
		}
	}
	srv.ApplyCorrection = func(req *http.Request, userID, txnID, category string) bool {
		txn, err := r.Txns.GetTxn(req.Context(), userID, txnID)
		if err != nil {
			return false
		}
		created, err := orch.ApplyCorrection(req.Context(), r.Rules, userID, txn.Counterparty, category)
		if err != nil {
			return false
		}
		return created
	}
	return srv
}

var _ = sql.ErrNoRows
