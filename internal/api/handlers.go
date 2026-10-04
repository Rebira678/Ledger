package api

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/rebira678/ledger/internal/auth"
	"github.com/rebira678/ledger/internal/domain"
	"github.com/rebira678/ledger/internal/httpx"
	"github.com/rebira678/ledger/internal/mail"
	"github.com/rebira678/ledger/internal/parsers"
	"github.com/rebira678/ledger/internal/repo"
)

// Server wires dependencies to the v1 handlers.
type Server struct {
	Repo   *repo.Repo
	Tokens *auth.Tokenizer
	Parser *parsers.Registry
	Cfg    Config
	Env    string           // "development" | "production" — controls secure cookies
	Now    func() time.Time // injectable clock for tests
	Mailer mail.Sender      // interface for sending emails

	// CategorizeAndQueue runs the categorization engine before persistence
	// (Phase 4); wired at startup, nil-safe in tests.
	CategorizeAndQueue func(r *http.Request, txn *domain.Transaction)
	// QueueClarification queues a clarifying question after the transaction is
	// persisted (Phase 6 / FR-5.1); nil-safe in tests.
	QueueClarification func(r *http.Request, txn *domain.Transaction)
	// ApplyCorrection persists a user category correction into the learning
	// loop; returns whether a new rule was created.
	ApplyCorrection func(r *http.Request, userID, txnID, category string) bool

	// ParseReceipt extracts a transaction from an image using the LLM.
	ParseReceipt func(ctx context.Context, mimeType string, imageBytes []byte) (*domain.Transaction, error)
	// ParseSMSFallback uses LLM to parse an unrecognized SMS format.
	ParseSMSFallback func(ctx context.Context, senderID, body string) (*domain.ParsedTransaction, error)
}

// Config carries the handler-level configuration subset.
type Config struct {
	MaxStatementBytes int64
	RateIngestPerMin  int
	RateUserPerMin    int
	RefreshTTLSeconds int64
}

// New builds the API server.
func New(r *repo.Repo, t *auth.Tokenizer, p *parsers.Registry, m mail.Sender, cfg Config) *Server {
	return &Server{Repo: r, Tokens: t, Parser: p, Mailer: m, Cfg: cfg, Now: time.Now}
}

//---- Auth handlers -----------------------------------------------------------

type registerReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

var emailRe = regexp.MustCompile(`^[^@\s]+@[^@\s]+\.[^@\s]+$`)

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	if !emailRe.MatchString(req.Email) {
		httpx.Envelope(w, http.StatusBadRequest, "VALIDATION_ERROR", "email must be a valid address", "email")
		return
	}
	if len(req.Password) < 8 {
		httpx.Envelope(w, http.StatusBadRequest, "VALIDATION_ERROR", "password must be at least 8 characters", "password")
		return
	}
	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	id, err := domain.NewID("usr")
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	user, err := s.Repo.Users.CreateUser(r.Context(), id, req.Email, hash)
	if err != nil {
		if errors.Is(err, domain.ErrAlreadyExists) {
			httpx.Envelope(w, http.StatusBadRequest, "VALIDATION_ERROR", "email already registered", "email")
			return
		}
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"user_id":    user.ID,
		"email":      user.Email,
		"created_at": user.CreatedAt.UTC().Format(time.RFC3339),
	})
}

type loginReq struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	user, hash, err := s.Repo.Users.GetByEmail(r.Context(), strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	if user == nil || !auth.CheckPassword(hash, req.Password) {
		httpx.Envelope(w, http.StatusUnauthorized, "UNAUTHENTICATED", "invalid credentials", "")
		return
	}
	s.issueTokens(w, r, user.ID, http.StatusOK)
}

// issueTokens mints an access token, rotates a refresh token, and writes the response.
func (s *Server) issueTokens(w http.ResponseWriter, r *http.Request, userID string, status int) {
	access, expiresIn, err := s.Tokens.NewAccessToken(userID)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	refresh, hash, err := auth.NewOpaqueToken()
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	expires := s.Now().Add(time.Duration(s.Cfg.RefreshTTLSeconds) * time.Second)
	if err := s.Repo.Tokens.CreateRefreshToken(r.Context(), hash, userID, expires); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	writeJSON(w, status, map[string]any{
		"access_token":  access,
		"refresh_token": refresh,
		"expires_in":    expiresIn,
	})
}

type refreshReq struct {
	RefreshToken string `json:"refresh_token"`
}

func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.RefreshToken == "" {
		httpx.Envelope(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "refresh_token is required", "refresh_token")
		return
	}
	hash := auth.HashToken(req.RefreshToken)
	userID, expiresAt, revoked, err := s.Repo.Tokens.GetRefreshToken(r.Context(), hash)
	if err != nil || revoked || s.Now().After(expiresAt) {
		httpx.Envelope(w, http.StatusUnauthorized, "UNAUTHENTICATED", "refresh token invalid or expired", "")
		return
	}
	// Rotate: revoke old, issue new pair.
	if err := s.Repo.Tokens.RevokeRefreshToken(r.Context(), hash); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	access, expiresIn, err := s.Tokens.NewAccessToken(userID)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	refresh, newHash, err := auth.NewOpaqueToken()
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	if err := s.Repo.Tokens.CreateRefreshToken(r.Context(), newHash, userID, s.Now().Add(time.Duration(s.Cfg.RefreshTTLSeconds)*time.Second)); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"access_token":  access,
		"refresh_token": refresh,
		"expires_in":    expiresIn,
	})
}

type forgotPasswordReq struct {
	Email string `json:"email"`
}

func (s *Server) handleForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req forgotPasswordReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	if email == "" {
		httpx.Envelope(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "email is required", "email")
		return
	}

	user, _, err := s.Repo.Users.GetByEmail(r.Context(), email)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	// Always return 200 to avoid email enumeration.
	if user != nil {
		rawCode, tokenHash, err := auth.NewResetCode()
		if err != nil {
			httpx.WriteError(w, httpx.MapDomainError(err))
			return
		}
		expiresAt := s.Now().Add(1 * time.Hour)
		if err := s.Repo.Users.CreatePasswordReset(r.Context(), tokenHash, user.ID, expiresAt); err != nil {
			httpx.WriteError(w, httpx.MapDomainError(err))
			return
		}

		if s.Mailer != nil {
			go func(email, code string) {
				if err := s.Mailer.SendPasswordReset(email, code); err != nil {
					// Don't fail the request if email fails, but log it (in a real app)
					fmt.Printf("ERROR sending email: %v\n", err)
				}
			}(user.Email, rawCode)
		}

		// Keep header for local dev convenience
		w.Header().Set("X-Debug-Reset-Token", rawCode)
	}
	w.WriteHeader(http.StatusOK)
}

type resetPasswordReq struct {
	Token    string `json:"token"`
	Password string `json:"password"`
}

func (s *Server) handleResetPassword(w http.ResponseWriter, r *http.Request) {
	var req resetPasswordReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Token == "" {
		httpx.Envelope(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "token is required", "token")
		return
	}
	if len(req.Password) < 8 {
		httpx.Envelope(w, http.StatusBadRequest, "VALIDATION_ERROR", "password must be at least 8 characters", "password")
		return
	}

	tokenHash := auth.HashToken(req.Token)
	userID, expiresAt, err := s.Repo.Users.GetPasswordReset(r.Context(), tokenHash)
	if err != nil || s.Now().After(expiresAt) {
		httpx.Envelope(w, http.StatusUnauthorized, "UNAUTHENTICATED", "invalid or expired reset token", "")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}

	if err := s.Repo.Users.UpdatePassword(r.Context(), userID, hash); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}

	// Consume token
	_ = s.Repo.Users.DeletePasswordReset(r.Context(), tokenHash)

	w.WriteHeader(http.StatusOK)
}

type pairReq struct {
	DeviceModel string `json:"device_model"`
	OSVersion   string `json:"os_version"`
}

func (s *Server) handlePair(w http.ResponseWriter, r *http.Request) {
	var req pairReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.DeviceModel == "" || req.OSVersion == "" {
		httpx.Envelope(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "device_model and os_version are required", "")
		return
	}
	userID := MustUserID(r.Context())
	devID, err := domain.NewID("dev")
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	key, keyHash, err := auth.NewDeviceAPIKey()
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	device, err := s.Repo.Devices.CreateDevice(r.Context(), &domain.Device{
		ID: devID, UserID: userID, DeviceModel: req.DeviceModel, OSVersion: req.OSVersion, APIKeyHash: keyHash,
	})
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"device_id":      device.ID,
		"device_api_key": key, // shown exactly once; only the hash is stored
	})
}

//---- Ingestion ----------------------------------------------------------------

type ingestSMSReq struct {
	SenderID        string `json:"sender_id"`
	Body            string `json:"body"`
	ReceivedAt      string `json:"received_at"`
	ClientMessageID string `json:"client_message_id"`
}

func (s *Server) handleIngestSMS(w http.ResponseWriter, r *http.Request) {
	var req ingestSMSReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	// Input validation at the boundary (engineering bar).
	if req.SenderID == "" || req.Body == "" || req.ClientMessageID == "" {
		httpx.Envelope(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "sender_id, body, and client_message_id are required", "")
		return
	}
	receivedAt, err := time.Parse(time.RFC3339, req.ReceivedAt)
	if err != nil {
		httpx.Envelope(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "received_at must be ISO 8601 (RFC 3339)", "received_at")
		return
	}
	ident, _ := IdentityFrom(r.Context())

	raw := &domain.RawMessage{
		ID: domain.MustNewID("msg"), UserID: ident.UserID, DeviceID: ident.DeviceID,
		ClientMessageID: req.ClientMessageID, SenderID: req.SenderID, Body: req.Body,
		ReceivedAt: receivedAt, Status: "parsed",
	}
	res, err := s.Repo.Messages.InsertRawMessage(r.Context(), raw)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	if !res.Created {
		// Idempotent no-op: duplicate client_message_id (contract §3).
		httpx.Envelope(w, http.StatusConflict, "DUPLICATE", "client_message_id already ingested", "client_message_id")
		return
	}

	parsed, perr := s.Parser.ParseSender(req.SenderID, req.Body, receivedAt)
	if perr != nil {
		if errors.Is(perr, parsers.ErrUnmatchedFormat) && s.ParseSMSFallback != nil {
			aiParsed, aiErr := s.ParseSMSFallback(r.Context(), req.SenderID, req.Body)
			if aiErr == nil && aiParsed != nil {
				parsed = *aiParsed
				perr = nil
				parsed.OccurredAt = receivedAt // LLM might not know exact date
			}
		}

		if perr != nil {
			// Route unmatched messages to the review queue, never fail silently (FR-3.3).
			if errors.Is(perr, parsers.ErrUnmatchedFormat) {
				if qerr := s.Repo.Messages.SetRawMessageOutcome(r.Context(), raw.ID, "queued_for_review", "", "no parser matched sender/format"); qerr != nil {
					httpx.WriteError(w, httpx.MapDomainError(qerr))
					return
				}
				writeJSON(w, http.StatusAccepted, map[string]any{
					"transaction_id": nil, "status": "queued_for_review", "confidence": nil,
				})
				return
			}
			if serr := s.Repo.Messages.SetRawMessageOutcome(r.Context(), raw.ID, "failed", "", perr.Error()); serr != nil {
				httpx.WriteError(w, httpx.MapDomainError(serr))
				return
			}
			httpx.WriteError(w, httpx.NewAPIErrorf(http.StatusUnprocessableEntity, "VALIDATION_ERROR", "body", "parsing sms: %v", perr))
			return
		}
	}

	txn := &domain.Transaction{
		ID: domain.MustNewID("txn"), UserID: ident.UserID, DeviceID: ident.DeviceID,
		Amount: parsed.Amount, Currency: parsed.Currency, Direction: parsed.Direction,
		Counterparty: parsed.Counterparty, Source: domain.SourceSMS, OcurredAt: parsed.OccurredAt,
		Balance: parsed.Balance,
	}
	// Categorization (FR-4) before persisting; nil-safe in tests.
	if s.CategorizeAndQueue != nil {
		s.CategorizeAndQueue(r, txn)
	}

	if err := s.Repo.Txns.InsertTxn(r.Context(), nil, txn); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	// Clarification queueing (FR-5.1) after the row exists.
	if s.QueueClarification != nil {
		s.QueueClarification(r, txn)
	}
	if err := s.Repo.Messages.SetRawMessageOutcome(r.Context(), raw.ID, "parsed", txn.ID, ""); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"transaction_id": txn.ID,
		"status":         "parsed",
		"confidence":     txn.CategoryConfidence,
	})
}

//---- Transactions -------------------------------------------------------------

func (s *Server) handleListTxns(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit := clampLimit(q.Get("limit"))
	f := repo.TxnListFilter{
		UserID:   MustUserID(r.Context()),
		From:     q.Get("from"),
		To:       q.Get("to"),
		Category: q.Get("category"),
		Limit:    limit,
		Cursor:   q.Get("cursor"),
	}
	items, next, err := s.Repo.Txns.ListTxns(r.Context(), f)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	type item struct {
		TransactionID string   `json:"transaction_id"`
		Amount        float64  `json:"amount"`
		Currency      string   `json:"currency"`
		Direction     string   `json:"direction"`
		Counterparty  string   `json:"counterparty"`
		Category      string   `json:"category"`
		Confidence    *float64 `json:"confidence"`
		Source        string   `json:"source"`
		OccurredAt    string   `json:"occurred_at"`
	}
	out := make([]item, 0, len(items))
	for _, t := range items {
		var conf *float64
		if t.CategoryHasConfidence {
			c := t.CategoryConfidence
			conf = &c
		}
		out = append(out, item{
			TransactionID: t.ID, Amount: t.Amount.Float64(), Currency: t.Currency,
			Direction: string(t.Direction), Counterparty: t.Counterparty, Category: t.Category,
			Confidence: conf, Source: string(t.Source), OccurredAt: t.OcurredAt.UTC().Format(time.RFC3339),
		})
	}
	var nextPtr *string
	if next != "" {
		nextPtr = &next
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out, "next_cursor": nextPtr})
}

type patchCategoryReq struct {
	Category string `json:"category"`
}

func (s *Server) handlePatchCategory(w http.ResponseWriter, r *http.Request) {
	txnID := r.PathValue("id")
	var req patchCategoryReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Category == "" {
		httpx.Envelope(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "category is required", "category")
		return
	}
	userID := MustUserID(r.Context())
	// Ownership check + update, both tenant-scoped.
	if _, err := s.Repo.Txns.GetTxn(r.Context(), userID, txnID); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	ruleCreated := false
	if s.ApplyCorrection != nil {
		ruleCreated = s.ApplyCorrection(r, userID, txnID, req.Category)
	}
	if err := s.Repo.Txns.UpdateCategory(r.Context(), userID, txnID, req.Category, "user"); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"transaction_id": txnID,
		"category":       req.Category,
		"rule_created":   ruleCreated,
	})
}

//---- Statement upload -----------------------------------------------------------

func (s *Server) handleStatementUpload(w http.ResponseWriter, r *http.Request) {
	userID := MustUserID(r.Context())
	r.Body = http.MaxBytesReader(w, r.Body, s.Cfg.MaxStatementBytes+1<<20)
	if err := r.ParseMultipartForm(s.Cfg.MaxStatementBytes); err != nil {
		httpx.Envelope(w, http.StatusRequestEntityTooLarge, "UNSUPPORTED_FORMAT", "file exceeds size limit (10MB)", "")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Envelope(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "multipart field 'file' is required", "file")
		return
	}
	defer file.Close()
	bankHint := r.FormValue("bank_hint")

	mimeType := header.Header.Get("Content-Type")
	switch {
	case strings.HasPrefix(mimeType, "application/pdf"),
		strings.HasPrefix(mimeType, "text/csv"),
		strings.HasPrefix(mimeType, "application/csv"),
		strings.HasSuffix(strings.ToLower(header.Filename), ".csv"),
		strings.HasSuffix(strings.ToLower(header.Filename), ".pdf"),
		strings.HasPrefix(mimeType, "image/jpeg"),
		strings.HasPrefix(mimeType, "image/png"):
	default:
		httpx.Envelope(w, http.StatusUnsupportedMediaType, "UNSUPPORTED_FORMAT", "uploaded file type not supported; accepts pdf, csv, jpeg, png", "")
		return
	}

	buf, err := io.ReadAll(file)
	if err != nil {
		httpx.Envelope(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "reading uploaded file", "file")
		return
	}
	n := len(buf)
	if int64(n) > s.Cfg.MaxStatementBytes {
		httpx.Envelope(w, http.StatusRequestEntityTooLarge, "UNSUPPORTED_FORMAT", "file exceeds size limit (10MB)", "")
		return
	}
	uploadID, err := domain.NewID("up")
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}

	up, err := s.Repo.Uploads.CreateUpload(r.Context(), &domain.StatementUpload{
		ID: uploadID, UserID: userID, BankHint: bankHint, MimeType: mimeType,
		SizeBytes: int64(n), Status: "processing",
	}, buf[:n])
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}

	// Synchronously process image and PDF receipts
	if s.ParseReceipt != nil && (strings.HasPrefix(mimeType, "image/") || mimeType == "application/pdf") {
		ident, ok := IdentityFrom(r.Context())
		txn, err := s.ParseReceipt(r.Context(), mimeType, buf[:n])
		if err == nil && txn != nil {
			txn.ID = domain.MustNewID("txn")
			txn.UserID = userID
			if ok {
				txn.DeviceID = ident.DeviceID
			}

			if s.CategorizeAndQueue != nil {
				s.CategorizeAndQueue(r, txn)
			}
			if err := s.Repo.Txns.InsertTxn(r.Context(), nil, txn); err == nil {
				s.Repo.Uploads.SetUploadStatus(r.Context(), uploadID, "completed", "", 1, 0)
				if s.QueueClarification != nil {
					s.QueueClarification(r, txn)
				}
			} else {
				s.Repo.Uploads.SetUploadStatus(r.Context(), uploadID, "failed", err.Error(), 0, 0)
				httpx.Envelope(w, http.StatusUnprocessableEntity, "PARSE_ERROR", "failed to insert transaction: "+err.Error(), "")
				return
			}
		} else {
			errMsg := "parse failed"
			if err != nil {
				errMsg = err.Error()
			}
			s.Repo.Uploads.SetUploadStatus(r.Context(), uploadID, "failed", errMsg, 0, 0)
			httpx.Envelope(w, http.StatusUnprocessableEntity, "PARSE_ERROR", "failed to parse receipt: "+errMsg, "")
			return
		}
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"upload_id": up.ID, "status": up.Status,
	})
}

func (s *Server) handleStatementStatus(w http.ResponseWriter, r *http.Request) {
	userID := MustUserID(r.Context())
	up, err := s.Repo.Uploads.GetUpload(r.Context(), userID, r.PathValue("upload_id"))
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"upload_id":            up.ID,
		"status":               up.Status,
		"transactions_created": up.TxCreated,
		"transactions_flagged": up.TxFlagged,
	})
}

//---- Agent: clarifications -----------------------------------------------------

func (s *Server) handleListClarifications(w http.ResponseWriter, r *http.Request) {
	userID := MustUserID(r.Context())
	items, err := s.Repo.Clarifs.ListPending(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	type item struct {
		ClarificationID string   `json:"clarification_id"`
		TransactionID   string   `json:"transaction_id"`
		Question        string   `json:"question"`
		Options         []string `json:"options"`
	}
	out := make([]item, 0, len(items))
	for _, c := range items {
		out = append(out, item{ClarificationID: c.ID, TransactionID: c.TransactionID, Question: c.Question, Options: c.Options})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

type respondReq struct {
	Answer string `json:"answer"`
}

func (s *Server) handleRespondClarification(w http.ResponseWriter, r *http.Request) {
	var req respondReq
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.WriteError(w, err)
		return
	}
	if req.Answer == "" {
		httpx.Envelope(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "answer is required", "answer")
		return
	}
	userID := MustUserID(r.Context())
	clarifID := r.PathValue("id")
	// Verify ownership and pending status.
	pending, err := s.Repo.Clarifs.ListPending(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	var target *domain.Clarification
	for _, c := range pending {
		if c.ID == clarifID {
			target = c
			break
		}
	}
	if target == nil {
		httpx.Envelope(w, http.StatusNotFound, "NOT_FOUND", "clarification not found or already resolved", "")
		return
	}
	ok, err := s.Repo.Clarifs.ResolveClarification(r.Context(), userID, clarifID, req.Answer)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	if !ok {
		httpx.Envelope(w, http.StatusNotFound, "NOT_FOUND", "clarification not found or already resolved", "")
		return
	}
	if err := s.Repo.Txns.UpdateCategory(r.Context(), userID, target.TransactionID, req.Answer, "user"); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	if s.ApplyCorrection != nil {
		s.ApplyCorrection(r, userID, target.TransactionID, req.Answer)
	}
	writeJSON(w, http.StatusOK, map[string]any{"clarification_id": clarifID, "status": "resolved"})
}

//---- Reports -------------------------------------------------------------------

func (s *Server) handleListReports(w http.ResponseWriter, r *http.Request) {
	userID := MustUserID(r.Context())
	items, err := s.Repo.Reports.ListReports(r.Context(), userID, clampLimit(r.URL.Query().Get("limit")))
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	type item struct {
		ReportID    string `json:"report_id"`
		PeriodStart string `json:"period_start"`
		PeriodEnd   string `json:"period_end"`
		GeneratedAt string `json:"generated_at"`
	}
	out := make([]item, 0, len(items))
	for _, rep := range items {
		out = append(out, item{
			ReportID:    rep.ID,
			PeriodStart: rep.PeriodStart.Format("2006-01-02"),
			PeriodEnd:   rep.PeriodEnd.Format("2006-01-02"),
			GeneratedAt: rep.GeneratedAt.UTC().Format(time.RFC3339),
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": out})
}

func (s *Server) handleGetReport(w http.ResponseWriter, r *http.Request) {
	rep, err := s.Repo.Reports.GetReport(r.Context(), MustUserID(r.Context()), r.PathValue("report_id"))
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	totals := map[string]float64{}
	for k, m := range rep.TotalsByCategory {
		totals[k] = m.Float64()
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"report_id":          rep.ID,
		"totals_by_category": totals,
		"week_over_week":     rep.WeekOverWeek,
		"narrative":          rep.Narrative,
	})
}

//---- Dashboard summary -----------------------------------------------------------

func (s *Server) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	userID := MustUserID(r.Context())
	ctx := r.Context()
	bal, err := s.Repo.Reports.EstimateBalance(ctx, userID)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "this_week"
	}

	now := s.Now()
	var from, to time.Time
	var prevFrom, prevTo time.Time

	switch period {
	case "last_week":
		to = domain.WeekStart(now)
		from = to.AddDate(0, 0, -7)
		prevTo = from
		prevFrom = prevTo.AddDate(0, 0, -7)
	case "this_month":
		from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		to = from.AddDate(0, 1, 0)
		prevTo = from
		prevFrom = prevTo.AddDate(0, -1, 0)
	case "last_month":
		to = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		from = to.AddDate(0, -1, 0)
		prevTo = from
		prevFrom = prevTo.AddDate(0, -1, 0)
	default: // "this_week"
		from = domain.WeekStart(now)
		to = from.AddDate(0, 0, 7)
		prevTo = from
		prevFrom = prevTo.AddDate(0, 0, -7)
	}

	fromStr := from.Format("2006-01-02T15:04:05Z")
	toStr := to.Format("2006-01-02T15:04:05Z")
	prevFromStr := prevFrom.Format("2006-01-02T15:04:05Z")
	prevToStr := prevTo.Format("2006-01-02T15:04:05Z")

	totals, err := s.Repo.Reports.SumByCategoryForPeriod(ctx, userID, fromStr, toStr)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	inc, exp, err := s.Repo.Reports.IncomeAndExpenseForPeriod(ctx, userID, fromStr, toStr)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}

	prevTotals, err := s.Repo.Reports.SumByCategoryForPeriod(ctx, userID, prevFromStr, prevToStr)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}

	topCat, topAmt := "", 0.0
	var thisPeriodTotal float64
	for c, v := range totals {
		thisPeriodTotal += v
		if v > topAmt {
			topCat, topAmt = c, v
		}
	}
	var lastPeriodTotal float64
	for _, v := range prevTotals {
		lastPeriodTotal += v
	}

	// Real WOW / MOM % change.
	var popPct *float64
	if lastPeriodTotal > 0 {
		pct := (thisPeriodTotal - lastPeriodTotal) / lastPeriodTotal * 100
		popPct = &pct
	}

	pending, err := s.Repo.Clarifs.ListPending(ctx, userID)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"estimated_balance":      bal,
		"currency":               "ETB",
		"period_income":          inc,
		"period_spend":           exp,
		"prev_period_spend":      lastPeriodTotal,
		"period_over_period_pct": popPct,
		"top_category":           topCat,
		"pending_clarifications": len(pending),
		"category_totals":        totals,
	})
}

//---- Health ----------------------------------------------------------------------

// HandleLivez is the unversioned liveness probe.
func (s *Server) HandleLivez(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

// HandleReadyz is the unversioned readiness probe (checks DB reachability).
func (s *Server) HandleReadyz(w http.ResponseWriter, r *http.Request) {
	if err := s.Repo.Txns.Ping(r.Context()); err != nil {
		httpx.Envelope(w, http.StatusServiceUnavailable, "INTERNAL", "database unreachable", "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

//---- helpers ---------------------------------------------------------------------

func clampLimit(raw string) int {
	const def, max = 25, 100
	if raw == "" {
		return def
	}
	n := 0
	for _, c := range raw {
		if c < '0' || c > '9' {
			return def
		}
		n = n*10 + int(c-'0')
		if n > max {
			return max
		}
	}
	if n == 0 {
		return def
	}
	return n
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// constantTimeEquals is kept for future token comparisons needing subtlety.
var _ = subtle.ConstantTimeCompare

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	userID := MustUserID(r.Context())

	user, err := s.Repo.Users.GetByID(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}

	devices, err := s.Repo.Devices.ListUserDevices(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"user":    user,
		"devices": devices,
	})
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := MustUserID(r.Context())
	var req struct {
		DisplayName string `json:"display_name"`
		AvatarURL   string `json:"avatar_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.WriteError(w, httpx.NewAPIErrorf(http.StatusBadRequest, "BAD_REQUEST", "", "invalid json"))
		return
	}

	// Just merge with existing since we don't want to blank out one if only updating the other
	user, err := s.Repo.Users.GetByID(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}

	if req.DisplayName == "" {
		req.DisplayName = user.DisplayName
	}
	if req.AvatarURL == "" {
		req.AvatarURL = user.AvatarURL
	}

	if err := s.Repo.Users.UpdateUserProfile(r.Context(), userID, req.DisplayName, req.AvatarURL); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
