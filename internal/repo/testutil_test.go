package repo_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// testDB is a real Postgres handle shared by integration tests in this package.
var testDB *sql.DB

func TestMain(m *testing.M) {
	code := 1
	func() {
		ctx := context.Background()
		dsn := startPostgres(ctx)
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			fmt.Fprintf(os.Stderr, "open test db: %v\n", err)
			os.Exit(2)
		}
		testDB = db
		defer db.Close()
		code = m.Run()
	}()
	os.Exit(code)
}

// startPostgres returns a DSN: either an external test database from
// LEDGER_TEST_DATABASE_URL (CI) or a throwaway container via testcontainers.
func startPostgres(ctx context.Context) string {
	if dsn := os.Getenv("LEDGER_TEST_DATABASE_URL"); dsn != "" {
		return dsn
	}
	// testcontainers requires Docker; both paths are REAL Postgres (D-19).
	dsn, err := postgresContainer(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "starting postgres container: %v\n", err)
		os.Exit(2)
	}
	return dsn
}

// freshDB gives each test a clean schema: drop all tables, re-apply migrations.
func freshDB(t *testing.T) {
	t.Helper()
	dropAllTables(t)
	applyMigrations(t)
}

func dropAllTables(t *testing.T) {
	t.Helper()
	rows, err := testDB.Query(`SELECT tablename FROM pg_tables WHERE schemaname='public'`)
	if err != nil {
		t.Fatalf("listing tables: %v", err)
	}
	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("scanning table: %v", err)
		}
		tables = append(tables, name)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		t.Fatalf("iterating tables: %v", err)
	}
	for _, tb := range tables {
		if _, err := testDB.Exec(`DROP TABLE IF EXISTS ` + tb + ` CASCADE`); err != nil {
			t.Fatalf("dropping %s: %v", tb, err)
		}
	}
}

// applyMigrations runs every migrations/*.up.sql against the test database.
func applyMigrations(t *testing.T) {
	t.Helper()
	root := findRepoRoot(t)
	entries, err := filepath.Glob(filepath.Join(root, "migrations", "*.up.sql"))
	if err != nil {
		t.Fatalf("glob migrations: %v", err)
	}
	sort.Strings(entries)
	if len(entries) == 0 {
		t.Fatal("no migrations found")
	}
	for _, path := range entries {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		if _, err := testDB.Exec(string(b)); err != nil {
			t.Fatalf("applying %s: %v", path, err)
		}
	}
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("repo root not found")
	return ""
}

// seed helpers ---------------------------------------------------------------

func seedUser(t *testing.T, email string) string {
	t.Helper()
	id := fmt.Sprintf("usr_%s", strings.ReplaceAll(strings.ToLower(email), "@", "_"))
	_, err := testDB.Exec(`INSERT INTO users (id, email, password_hash) VALUES ($1,$2,'x')`,
		id, email)
	if err != nil {
		t.Fatalf("seeding user: %v", err)
	}
	return id
}

func seedTxn(t *testing.T, userID, counterparty, category string, amountCents int64, occurredAt time.Time) string {
	t.Helper()
	id := fmt.Sprintf("txn_%d", time.Now().UnixNano())
	units := amountCents / 100
	cents := amountCents % 100
	_, err := testDB.Exec(`INSERT INTO transactions
	 (id, user_id, amount, currency, direction, counterparty, category, category_confidence, category_source, source, occurred_at)
	 VALUES ($1,$2,$3,'ETB','debit',$4,$5,0.9,'system','sms',$6)`,
		id, userID, fmt.Sprintf("%d.%02d", units, cents), counterparty, category, occurredAt)
	if err != nil {
		t.Fatalf("seeding txn: %v", err)
	}
	return id
}
