package api_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/rebira678/ledger/internal/agent"
	"github.com/rebira678/ledger/internal/agent/llm"
	"github.com/rebira678/ledger/internal/api"
	"github.com/rebira678/ledger/internal/auth"
	"github.com/rebira678/ledger/internal/categorize"
	"github.com/rebira678/ledger/internal/loggerx"
	"github.com/rebira678/ledger/internal/parsers"
	"github.com/rebira678/ledger/internal/repo"
)

var testDB *sql.DB

func TestMain(m *testing.M) {
	code := 1
	func() {
		dsn := os.Getenv("LEDGER_TEST_DATABASE_URL")
		if dsn == "" {
			dsn = containerDSN(context.Background())
		}
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		testDB = db
		defer db.Close()
		code = m.Run()
	}()
	os.Exit(code)
}

func containerDSN(ctx context.Context) string {
	req := testcontainersRequest()
	c, err := startContainer(ctx, req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	return c
}

// newTestServer boots a full stack (real DB, real routes) per test.
func newTestServer(t *testing.T) (*httptest.Server, *api.Server) {
	t.Helper()
	dropAll(t)
	applyMigrations(t)

	tok := auth.NewTokenizer([]byte("test-signing-key-0123456789abcdef"), 15*time.Minute, time.Hour)
	r := repo.New(testDB)
	registry := parsers.NewRegistry(parsers.CBEParser{}, parsers.TelebirrParser{})
	engine := categorize.NewEngine(r.Rules)
	orch := agent.New(engine, llm.New("", "", "", time.Second))

	srv := api.New(r, tok, registry, api.Config{
		MaxStatementBytes: 10 << 20, RateIngestPerMin: 120, RateUserPerMin: 60, RefreshTTLSeconds: 3600,
	})
	srv = newWired(srv, r, orch)
	ts := httptest.NewServer(srv.Routes(tok))
	t.Cleanup(ts.Close)
	return ts, srv
}

//---- HTTP helpers -------------------------------------------------------------

func postJSON(t *testing.T, url string, headers map[string]string, body any) (*http.Response, map[string]any) {
	t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

func getJSON(t *testing.T, url string, headers map[string]string) (*http.Response, map[string]any) {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

func errorEnvelope(t *testing.T, body map[string]any) map[string]any {
	t.Helper()
	e, ok := body["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error envelope, got %v", body)
	}
	return e
}

// TestIntegration_RegisterLoginPairIngest walks the full happy path.
func TestIntegration_RegisterLoginPairIngest(t *testing.T) {
	ts, _ := newTestServer(t)

	// Register.
	resp, reg := postJSON(t, ts.URL+"/v1/auth/register", nil, map[string]string{
		"email": "abel@example.com", "password": "s3curepass!",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("register status %d: %v", resp.StatusCode, reg)
	}
	userID, _ := reg["user_id"].(string)
	if userID == "" {
		t.Fatalf("no user_id in %v", reg)
	}

	// Duplicate email → 400.
	if resp, _ = postJSON(t, ts.URL+"/v1/auth/register", nil, map[string]string{
		"email": "abel@example.com", "password": "s3curepass!",
	}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("duplicate email should 400, got %d", resp.StatusCode)
	}

	// Weak password → 400.
	if resp, _ = postJSON(t, ts.URL+"/v1/auth/register", nil, map[string]string{
		"email": "x@example.com", "password": "short",
	}); resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("weak password should 400, got %d", resp.StatusCode)
	}

	// Login.
	resp, login := postJSON(t, ts.URL+"/v1/auth/login", nil, map[string]string{
		"email": "abel@example.com", "password": "s3curepass!",
	})
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("login status %d: %v", resp.StatusCode, login)
	}
	access, _ := login["access_token"].(string)
	if access == "" {
		t.Fatal("no access token")
	}

	// Bad login → 401.
	if resp, _ = postJSON(t, ts.URL+"/v1/auth/login", nil, map[string]string{
		"email": "abel@example.com", "password": "wrongpass!",
	}); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad login should 401, got %d", resp.StatusCode)
	}

	// Pair device.
	authHdr := map[string]string{"Authorization": "Bearer " + access}
	resp, pair := postJSON(t, ts.URL+"/v1/devices/pair", authHdr, map[string]string{
		"device_model": "Pixel 6a", "os_version": "14",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("pair status %d: %v", resp.StatusCode, pair)
	}
	deviceKey, _ := pair["device_api_key"].(string)
	if deviceKey == "" {
		t.Fatal("no device key")
	}

	// Ingest without device key → 401 (contract: JWT + X-Device-Key).
	smsBody := map[string]string{
		"sender_id": "CBE", "body": "CBE: You have received 500.00 ETB from ALMAZ TESFAYE (Acc 1) on 14/09/2026 07:41. Ref: TXN12345678. Balance: 4,820.50 ETB",
		"received_at": "2026-09-14T07:41:00Z", "client_message_id": "c8f1e2a0",
	}
	if resp, _ = postJSON(t, ts.URL+"/v1/ingest/sms", authHdr, smsBody); resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("ingest without device key should 401, got %d", resp.StatusCode)
	}

	// Ingest with both → 201 parsed.
	ingestHdr := map[string]string{"Authorization": "Bearer " + access, "X-Device-Key": deviceKey}
	resp, ing := postJSON(t, ts.URL+"/v1/ingest/sms", ingestHdr, smsBody)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("ingest status %d: %v", resp.StatusCode, ing)
	}
	txnID, _ := ing["transaction_id"].(string)
	if ing["status"] != "parsed" || txnID == "" {
		t.Fatalf("unexpected ingest response %v", ing)
	}

	// Duplicate send → 409 DUPLICATE (idempotent no-op).
	resp, dup := postJSON(t, ts.URL+"/v1/ingest/sms", ingestHdr, smsBody)
	if resp.StatusCode != http.StatusConflict {
		t.Fatalf("duplicate should 409, got %d: %v", resp.StatusCode, dup)
	}
	if e := errorEnvelope(t, dup); e["code"] != "DUPLICATE" {
		t.Fatalf("expected DUPLICATE code, got %v", e)
	}
	var n int
	_ = testDB.QueryRow(`SELECT count(*) FROM raw_messages WHERE client_message_id='c8f1e2a0'`).Scan(&n)
	if n != 1 {
		t.Fatalf("expected 1 raw message, got %d", n)
	}

	// Malformed payload → 422.
	resp, bad := postJSON(t, ts.URL+"/v1/ingest/sms", ingestHdr, map[string]string{
		"sender_id": "CBE", "body": "x", "received_at": "not-a-time", "client_message_id": "z1",
	})
	if resp.StatusCode != http.StatusUnprocessableEntity {
		t.Fatalf("malformed should 422, got %d: %v", resp.StatusCode, bad)
	}

	// Unmatched sender → 202 queued_for_review (FR-3.3).
	resp, queued := postJSON(t, ts.URL+"/v1/ingest/sms", ingestHdr, map[string]string{
		"sender_id": "MY-BANK-XYZ", "body": "totally unknown format",
		"received_at": "2026-09-14T08:00:00Z", "client_message_id": "u1",
	})
	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("unmatched should 202, got %d: %v", resp.StatusCode, queued)
	}
	if queued["status"] != "queued_for_review" {
		t.Fatalf("expected queued_for_review, got %v", queued)
	}

	// Transactions list shows the parsed one.
	resp, list := getJSON(t, ts.URL+"/v1/transactions", authHdr)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("list status %d", resp.StatusCode)
	}
	items, _ := list["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 transaction, got %d: %v", len(items), list)
	}

	// Dashboard summary responds.
	if resp, sum := getJSON(t, ts.URL+"/v1/dashboard/summary", authHdr); resp.StatusCode != http.StatusOK {
		t.Fatalf("summary status %d: %v", resp.StatusCode, sum)
	}
}

// TestIntegration_TenantIsolationViaAPI proves user A's token cannot touch
// user B's transaction through the API (engineering bar: explicit test).
func TestIntegration_TenantIsolationViaAPI(t *testing.T) {
	ts, _ := newTestServer(t)

	registerAndLogin := func(email string) (string, string) {
		postJSON(t, ts.URL+"/v1/auth/register", nil, map[string]string{"email": email, "password": "s3curepass!"})
		_, login := postJSON(t, ts.URL+"/v1/auth/login", nil, map[string]string{"email": email, "password": "s3curepass!"})
		access, _ := login["access_token"].(string)
		_, pair := postJSON(t, ts.URL+"/v1/devices/pair", map[string]string{"Authorization": "Bearer " + access},
			map[string]string{"device_model": "M", "os_version": "14"})
		key, _ := pair["device_api_key"].(string)
		return access, key
	}

	accessA, keyA := registerAndLogin("a@example.com")
	accessB, keyB := registerAndLogin("b@example.com")

	hdrA := map[string]string{"Authorization": "Bearer " + accessA, "X-Device-Key": keyA}
	hdrB := map[string]string{"Authorization": "Bearer " + accessB, "X-Device-Key": keyB}

	// B ingests a transaction.
	body := map[string]string{
		"sender_id": "CBE", "body": "CBE: You have paid 100.00 ETB to MERCHANT X on 14/09/2026 10:00. Ref: TXB1.",
		"received_at": "2026-09-14T10:00:00Z", "client_message_id": "b-msg-1",
	}
	resp, ing := postJSON(t, ts.URL+"/v1/ingest/sms", hdrB, body)
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("B ingest failed: %d %v", resp.StatusCode, ing)
	}
	txnB, _ := ing["transaction_id"].(string)

	// A cannot patch B's transaction.
	req, _ := http.NewRequest(http.MethodPatch, ts.URL+"/v1/transactions/"+txnB+"/category",
		bytes.NewReader([]byte(`{"category":"Rent"}`)))
	req.Header.Set("Authorization", "Bearer "+accessA)
	respA, _ := http.DefaultClient.Do(req)
	if respA.StatusCode != http.StatusNotFound {
		t.Fatalf("A patching B's txn must 404, got %d", respA.StatusCode)
	}
	respA.Body.Close()

	// A's list must not include B's txn.
	_, list := getJSON(t, ts.URL+"/v1/transactions", hdrA)
	items, _ := list["items"].([]any)
	if len(items) != 0 {
		t.Fatalf("A's listing leaked B's data: %v", list)
	}

	// B still sees it.
	_, listB := getJSON(t, ts.URL+"/v1/transactions", hdrB)
	itemsB, _ := listB["items"].([]any)
	if len(itemsB) != 1 {
		t.Fatalf("B lost sight of own txn: %v", listB)
	}
}

// registerLoginPair is a test helper: full onboarding for one user.
func registerLoginPair(t *testing.T, tsURL, email string) map[string]string {
	t.Helper()
	postJSON(t, tsURL+"/v1/auth/register", nil, map[string]string{"email": email, "password": "s3curepass!"})
	_, login := postJSON(t, tsURL+"/v1/auth/login", nil, map[string]string{"email": email, "password": "s3curepass!"})
	access, _ := login["access_token"].(string)
	_, pair := postJSON(t, tsURL+"/v1/devices/pair", map[string]string{"Authorization": "Bearer " + access},
		map[string]string{"device_model": "Pixel 6a", "os_version": "14"})
	key, _ := pair["device_api_key"].(string)
	return map[string]string{"Authorization": "Bearer " + access, "X-Device-Key": key}
}

// TestIntegration_ClaimCorrectionLoop exercises clarifications + learning loop.
func TestIntegration_ClaimCorrectionLoop(t *testing.T) {
	ts, _ := newTestServer(t)
	hdr := registerLoginPair(t, ts.URL, "c@example.com")

	// Ingest a low-confidence (uncategorized counterparty) transaction.
	resp, ing := postJSON(t, ts.URL+"/v1/ingest/sms", hdr, map[string]string{
		"sender_id": "CBE", "body": "CBE: You have paid 800.00 ETB to YILMA TESFAYE on 14/09/2026 08:00. Ref: TXC1.",
		"received_at": "2026-09-14T08:00:00Z", "client_message_id": "c-msg-1",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("ingest status %d: %v", resp.StatusCode, ing)
	}
	txnID, _ := ing["transaction_id"].(string)
	if conf, ok := ing["confidence"].(float64); ok && conf >= 0.60 {
		t.Fatalf("expected low confidence for unknown counterparty, got %v", ing)
	}

	// A pending clarification should exist.
	resp, clars := getJSON(t, ts.URL+"/v1/agent/clarifications", hdr)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("clarifications status %d", resp.StatusCode)
	}
	items, _ := clars["items"].([]any)
	if len(items) != 1 {
		t.Fatalf("expected 1 pending clarification, got %v", clars)
	}
	c0, _ := items[0].(map[string]any)
	clrID, _ := c0["clarification_id"].(string)

	// Answer it.
	resp, answered := postJSON(t, ts.URL+"/v1/agent/clarifications/"+clrID+"/respond", hdr,
		map[string]string{"answer": "Rent"})
	if resp.StatusCode != http.StatusOK || answered["status"] != "resolved" {
		t.Fatalf("respond failed: %d %v", resp.StatusCode, answered)
	}

	// Transaction category updated by the answer.
	_, list := getJSON(t, ts.URL+"/v1/transactions", hdr)
	items, _ = list["items"].([]any)
	var cat string
	for _, it := range items {
		m, _ := it.(map[string]any)
		if m["transaction_id"] == txnID {
			cat, _ = m["category"].(string)
		}
	}
	if cat != "Rent" {
		t.Fatalf("category should be Rent after clarification, got %q", cat)
	}

	// Re-answering must 404 (already resolved).
	resp, _ = postJSON(t, ts.URL+"/v1/agent/clarifications/"+clrID+"/respond", hdr, map[string]string{"answer": "Other"})
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("resolved clarification re-answer should 404, got %d", resp.StatusCode)
	}

	// A second similar transaction now auto-categorizes via the learned rule.
	resp, ing2 := postJSON(t, ts.URL+"/v1/ingest/sms", hdr, map[string]string{
		"sender_id": "CBE", "body": "CBE: You have paid 800.00 ETB to YILMA TESFAYE on 15/09/2026 08:00. Ref: TXC2.",
		"received_at": "2026-09-15T08:00:00Z", "client_message_id": "c-msg-2",
	})
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("second ingest failed: %d %v", resp.StatusCode, ing2)
	}
	if ing2["status"] != "parsed" {
		t.Fatalf("second ingest status %v", ing2)
	}
	if conf, ok := ing2["confidence"].(float64); ok && conf < 0.60 {
		t.Fatalf("learned rule should raise confidence above threshold, got %v", ing2)
	}
}

// TestIntegration_StatementUpload covers the fallback path (FR-7).
func TestIntegration_StatementUpload(t *testing.T) {
	ts, _ := newTestServer(t)
	hdr := registerLoginPair(t, ts.URL, "s@example.com")

	// Unsupported type → 415.
	resp, _ := postMultipart(t, ts.URL+"/v1/ingest/statement", hdr, "notes.txt", []byte("hello"))
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Fatalf("txt upload should 415, got %d", resp.StatusCode)
	}

	// CSV accepted → 202/201 processing.
	csv := []byte("date,amount,direction,counterparty\n2026-09-01,100.00,debit,SHOP X\n")
	resp, up := postMultipart(t, ts.URL+"/v1/ingest/statement", hdr, "statement.csv", csv)
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Fatalf("csv upload status %d: %v", resp.StatusCode, up)
	}
	uploadID, _ := up["upload_id"].(string)
	if uploadID == "" {
		t.Fatal("no upload id")
	}

	// Status endpoint (tenant-scoped) → 200.
	if resp, st := getJSON(t, ts.URL+"/v1/ingest/statement/"+uploadID, hdr); resp.StatusCode != http.StatusOK {
		t.Fatalf("status endpoint %d: %v", resp.StatusCode, st)
	}

	// Unknown upload → 404.
	if resp, _ = getJSON(t, ts.URL+"/v1/ingest/statement/up_nope", hdr); resp.StatusCode != http.StatusNotFound {
		t.Fatalf("unknown upload should 404, got %d", resp.StatusCode)
	}
}

// postMultipart uploads a file field with bank_hint.
func postMultipart(t *testing.T, url string, headers map[string]string, filename string, content []byte) (*http.Response, map[string]any) {
	t.Helper()
	var buf bytes.Buffer
	boundary := "----ledgerboundary123"
	fmt.Fprintf(&buf, "--%s\r\n", boundary)
	fmt.Fprintf(&buf, "Content-Disposition: form-data; name=\"file\"; filename=%q\r\n", filename)
	fmt.Fprintf(&buf, "Content-Type: %s\r\n\r\n", contentTypeFor(filename))
	buf.Write(content)
	fmt.Fprintf(&buf, "\r\n--%s--\r\n", boundary)
	req, err := http.NewRequest(http.MethodPost, url, &buf)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "multipart/form-data; boundary="+boundary)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&out)
	return resp, out
}

func contentTypeFor(name string) string {
	switch filepath.Ext(name) {
	case ".csv":
		return "text/csv"
	case ".pdf":
		return "application/pdf"
	default:
		return "text/plain"
	}
}

var _ = loggerx.New // keep import symmetry with main wiring
