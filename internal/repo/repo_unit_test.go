package repo_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/rebira678/ledger/internal/repo"
)

// mockTxDB implements repo.DB and CustomTxStarter for unit testing transactions.
type mockTxDB struct {
	commitCalled   bool
	rollbackCalled bool
	failCommit     bool
	failBegin      bool
}

func (m *mockTxDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}

func (m *mockTxDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return nil
}

func (m *mockTxDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, nil
}

func (m *mockTxDB) BeginCustomTx(ctx context.Context) (repo.DB, func() error, func() error, error) {
	if m.failBegin {
		return nil, nil, nil, errors.New("begin failed")
	}
	commit := func() error {
		m.commitCalled = true
		if m.failCommit {
			return errors.New("commit failed")
		}
		return nil
	}
	rollback := func() error {
		m.rollbackCalled = true
		return nil
	}
	return m, commit, rollback, nil
}

func TestUnit_WithTx_CommitOnSuccess(t *testing.T) {
	mock := &mockTxDB{}
	r := repo.New(mock)

	executed := false
	err := r.WithTx(context.Background(), func(txRepo *repo.Repo) error {
		executed = true
		return nil
	})

	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if !executed {
		t.Fatal("expected fn to execute")
	}
	if !mock.commitCalled {
		t.Fatal("expected transaction Commit to be called on success")
	}
	if mock.rollbackCalled {
		t.Fatal("expected Rollback NOT to be called on success")
	}
}

func TestUnit_WithTx_RollbackOnError(t *testing.T) {
	mock := &mockTxDB{}
	r := repo.New(mock)

	targetErr := errors.New("something went wrong")
	err := r.WithTx(context.Background(), func(txRepo *repo.Repo) error {
		return targetErr
	})

	if !errors.Is(err, targetErr) {
		t.Fatalf("expected targetErr, got: %v", err)
	}
	if !mock.rollbackCalled {
		t.Fatal("expected transaction Rollback to be called on error")
	}
	if mock.commitCalled {
		t.Fatal("expected Commit NOT to be called on error")
	}
}

func TestUnit_WithTx_RollbackOnPanic(t *testing.T) {
	mock := &mockTxDB{}
	r := repo.New(mock)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic to propagate")
		}
		if !mock.rollbackCalled {
			t.Fatal("expected Rollback to be called when panicking")
		}
		if mock.commitCalled {
			t.Fatal("expected Commit NOT to be called when panicking")
		}
	}()

	_ = r.WithTx(context.Background(), func(txRepo *repo.Repo) error {
		panic("database connection severed")
	})
}

func TestUnit_WithTx_FallbackWhenNoTxSupport(t *testing.T) {
	// Simple struct implementing only repo.DB (no TxBeginner or BeginCustomTx)
	simple := &simpleDB{}
	r := repo.New(simple)

	executed := false
	err := r.WithTx(context.Background(), func(txRepo *repo.Repo) error {
		executed = true
		return nil
	})

	if err != nil {
		t.Fatalf("expected fallback to succeed, got %v", err)
	}
	if !executed {
		t.Fatal("expected fn to be executed in fallback mode")
	}
}

type simpleDB struct{}

func (s *simpleDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return nil, nil
}
func (s *simpleDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return nil
}
func (s *simpleDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return nil, nil
}
