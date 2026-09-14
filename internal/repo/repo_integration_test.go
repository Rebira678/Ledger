package repo_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/abel-gezahegn/ledger/internal/domain"
	"github.com/abel-gezahegn/ledger/internal/repo"
)

// TestIntegration_Ingestion_Idempotency submits the same message twice and
// asserts exactly one transaction-bearing raw message row exists (FR-2.3,
// contract §3 409 semantics; required by the engineering bar).
func TestIntegration_Ingestion_Idempotency(t *testing.T) {
	freshDB(t)
	ctx := context.Background()
	r := repo.New(testDB)

	userID := seedUser(t, "abel@example.com")
	_, err := testDB.Exec(`INSERT INTO devices (id, user_id, device_model, os_version, api_key_hash)
		VALUES ('dev_1', $1, 'Pixel 6a', '14', 'hash1')`, userID)
	if err != nil {
		t.Fatal(err)
	}

	msg := &domain.RawMessage{
		ID: "msg_1", UserID: userID, DeviceID: "dev_1",
		ClientMessageID: "c8f1e2a0", SenderID: "CBE",
		Body:       "CBE: You have received 500.00 ETB from ALMAZ on 14/09/2026 07:41. Ref: TX123.",
		ReceivedAt: time.Now().UTC(), Status: "parsed",
	}

	first, err := r.Messages.InsertRawMessage(ctx, msg)
	if err != nil {
		t.Fatalf("first insert: %v", err)
	}
	if !first.Created {
		t.Fatal("first insert should be Created=true")
	}

	// Retry with the SAME device + client_message_id: must be an idempotent no-op.
	second, err := r.Messages.InsertRawMessage(ctx, msg)
	if err != nil {
		t.Fatalf("second insert: %v", err)
	}
	if second.Created {
		t.Fatal("duplicate insert must not be created")
	}
	if second.ID != first.ID {
		t.Fatalf("duplicate should return existing id %s, got %s", first.ID, second.ID)
	}

	var n int
	if err := testDB.QueryRow(`SELECT count(*) FROM raw_messages WHERE client_message_id='c8f1e2a0'`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("expected exactly 1 raw message row, got %d", n)
	}
}

// TestIntegration_TenantIsolation_AtQueryLayer proves user A cannot read or
// modify user B's data even when A's requests hit the repository directly
// (multi-tenant isolation enforced at the query layer; engineering bar).
func TestIntegration_TenantIsolation_AtQueryLayer(t *testing.T) {
	freshDB(t)
	ctx := context.Background()
	r := repo.New(testDB)

	userA := seedUser(t, "a@example.com")
	userB := seedUser(t, "b@example.com")
	txnB := seedTxn(t, userB, "Secret Merchant", "Groceries", 99900, time.Now().Add(-time.Hour))

	// A cannot READ B's transaction.
	if _, err := r.Txns.GetTxn(ctx, userA, txnB); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-user read must return ErrNotFound, got %v", err)
	}

	// A cannot MODIFY B's transaction.
	if err := r.Txns.UpdateCategory(ctx, userA, txnB, "Rent", "user"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-user update must return ErrNotFound, got %v", err)
	}

	// A cannot see B's rows in a listing.
	items, _, err := r.Txns.ListTxns(ctx, repo.TxnListFilter{UserID: userA, Limit: 50})
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if it.UserID != userA {
			t.Fatalf("user A listing leaked user %s row %s", it.UserID, it.ID)
		}
	}

	// B still sees their own row.
	got, err := r.Txns.GetTxn(ctx, userB, txnB)
	if err != nil || got.ID != txnB {
		t.Fatalf("owner read failed: %v", err)
	}

	// A cannot resolve B's clarification.
	ok, err := r.Clarifs.ResolveClarification(ctx, userA, "clr_B", "Rent")
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("cross-user clarification resolution must fail")
	}

	// A cannot read B's report.
	if _, err := r.Reports.GetReport(ctx, userA, "rpt_B"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatalf("cross-user report read must return ErrNotFound, got %v", err)
	}
}

// TestIntegration_TransactionLifecycle covers insert → list → patch → aggregates.
func TestIntegration_TransactionLifecycle(t *testing.T) {
	freshDB(t)
	ctx := context.Background()
	r := repo.New(testDB)

	userID := seedUser(t, "life@example.com")
	txn := &domain.Transaction{
		ID: "txn_life1", UserID: userID, DeviceID: "",
		Amount: domain.Money{Units: 3200}, Currency: "ETB",
		Direction: domain.DirectionDebit, Counterparty: "A A REAL ESTATE",
		Category: "Rent", CategoryConfidence: 0.7, CategoryHasConfidence: true,
		CategorySource: "system", Source: domain.SourceSMS,
		OcurredAt: time.Now().Add(-24 * time.Hour).UTC(),
	}
	if err := r.Txns.InsertTxn(ctx, nil, txn); err != nil {
		t.Fatalf("insert: %v", err)
	}

	items, next, err := r.Txns.ListTxns(ctx, repo.TxnListFilter{UserID: userID, Category: "Rent", Limit: 10})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(items) != 1 || next != "" {
		t.Fatalf("want 1 item no cursor, got %d items cursor %q", len(items), next)
	}

	if err := r.Txns.UpdateCategory(ctx, userID, "txn_life1", "Housing", "user"); err != nil {
		t.Fatalf("patch: %v", err)
	}
	got, _ := r.Txns.GetTxn(ctx, userID, "txn_life1")
	if got.Category != "Housing" || got.CategorySource != "user" {
		t.Fatalf("patch not applied: %+v", got)
	}

	totals, err := r.Reports.SumByCategoryForPeriod(ctx, userID, "2026-01-01T00:00:00Z", "2027-01-01T00:00:00Z")
	if err != nil {
		t.Fatal(err)
	}
	if totals["Housing"] != 3200 {
		t.Fatalf("aggregate Housing = %v, want 3200", totals["Housing"])
	}
}

// TestIntegration_StatementUploads covers upload creation + tenant-scoped status.
func TestIntegration_StatementUploads(t *testing.T) {
	freshDB(t)
	ctx := context.Background()
	r := repo.New(testDB)

	user := seedUser(t, "up@example.com")
	other := seedUser(t, "other@example.com")

	up, err := r.Uploads.CreateUpload(ctx, &domain.StatementUpload{
		ID: "up_1", UserID: user, MimeType: "text/csv", SizeBytes: 12, Status: "processing",
	}, []byte("col1,col2\n1,2\n"))
	if err != nil {
		t.Fatalf("create upload: %v", err)
	}
	if up.ID != "up_1" {
		t.Fatalf("unexpected id %s", up.ID)
	}

	if _, err := r.Uploads.GetUpload(ctx, other, "up_1"); !errors.Is(err, domain.ErrNotFound) {
		t.Fatal("cross-user upload read must be ErrNotFound")
	}
	got, err := r.Uploads.GetUpload(ctx, user, "up_1")
	if err != nil || got.Status != "processing" {
		t.Fatalf("owner read: %v %+v", err, got)
	}

	if err := r.Uploads.SetUploadStatus(ctx, "up_1", "completed", "", 42, 3); err != nil {
		t.Fatal(err)
	}
	got, _ = r.Uploads.GetUpload(ctx, user, "up_1")
	if got.TxCreated != 42 || got.TxFlagged != 3 {
		t.Fatalf("status update not persisted: %+v", got)
	}
}
