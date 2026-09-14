package api

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/abel-gezahegn/ledger/internal/auth"
	"github.com/abel-gezahegn/ledger/internal/domain"
	"github.com/abel-gezahegn/ledger/internal/httpx"
	"github.com/abel-gezahegn/ledger/internal/repo"
	"github.com/abel-gezahegn/ledger/internal/web"
)

const sessionCookie = "ledger_session"

// DashboardRoutes serves the server-rendered UI (D-09) under /app and /login.
func (s *Server) DashboardRoutes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", s.handleHomeRedirect)
	mux.HandleFunc("GET /login", s.handleLoginPage)
	mux.HandleFunc("POST /login", s.handleLoginSubmit)
	mux.HandleFunc("GET /register", s.handleRegisterPage)
	mux.HandleFunc("POST /register", s.handleRegisterSubmit)
	mux.HandleFunc("GET /logout", s.handleLogout)

	// Authenticated pages.
	authed := func(h http.HandlerFunc) http.Handler {
		return s.requireSession(h)
	}
	mux.Handle("GET /app", authed(s.handleAppHome))
	mux.Handle("GET /app/transactions", authed(s.handleAppTransactions))
	mux.Handle("POST /app/transactions/{id}/category", authed(s.handleAppPatchCategory))
	mux.Handle("GET /app/clarifications", authed(s.handleAppClarifications))
	mux.Handle("POST /app/clarifications/{id}/respond", authed(s.handleAppRespond))
	mux.Handle("GET /app/reports", authed(s.handleAppReports))
	mux.Handle("GET /app/reports/{report_id}", authed(s.handleAppReport))
	return mux
}

//---- session helpers -----------------------------------------------------------

func (s *Server) requireSession(next http.HandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		c, err := r.Cookie(sessionCookie)
		if err != nil || c.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		userID, err := s.Tokens.VerifyAccessToken(c.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		ctx := WithIdentity(r.Context(), Identity{UserID: userID})
		next(w, r.WithContext(ctx))
	})
}

func (s *Server) setSession(w http.ResponseWriter, userID string) error {
	token, _, err := s.Tokens.NewAccessToken(userID)
	if err != nil {
		return fmt.Errorf("minting session token: %w", err)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   s.secureCookies(),
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(s.Cfg.RefreshTTLSeconds),
	})
	return nil
}

func (s *Server) secureCookies() bool { return s.Env == "production" }

//---- auth pages ----------------------------------------------------------------

func (s *Server) handleHomeRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

func (s *Server) handleLoginPage(w http.ResponseWriter, r *http.Request) {
	web.Render(w, "login.html", web.PageData{Title: "Sign in"})
}

func (s *Server) handleRegisterPage(w http.ResponseWriter, r *http.Request) {
	web.Render(w, "register.html", web.PageData{Title: "Create account"})
}

func (s *Server) handleLoginSubmit(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")
	user, hash, err := s.Repo.Users.GetByEmail(r.Context(), email)
	if err != nil || user == nil || !auth.CheckPassword(hash, password) {
		web.Render(w, "login.html", web.PageData{Title: "Sign in", ErrorMsg: "Invalid email or password."})
		return
	}
	if err := s.setSession(w, user.ID); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

func (s *Server) handleRegisterSubmit(w http.ResponseWriter, r *http.Request) {
	email := strings.ToLower(strings.TrimSpace(r.FormValue("email")))
	password := r.FormValue("password")
	if !emailRe.MatchString(email) || len(password) < 8 {
		web.Render(w, "register.html", web.PageData{
			Title: "Create account", ErrorMsg: "Enter a valid email and a password of at least 8 characters.",
		})
		return
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		web.Render(w, "register.html", web.PageData{Title: "Create account", ErrorMsg: "Could not create account."})
		return
	}
	id, err := domain.NewID("usr")
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	user, err := s.Repo.Users.CreateUser(r.Context(), id, email, hash)
	if err != nil {
		if errIsAlreadyExists(err) {
			web.Render(w, "register.html", web.PageData{Title: "Create account", ErrorMsg: "That email is already registered."})
			return
		}
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	if err := s.setSession(w, user.ID); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	http.Redirect(w, r, "/app", http.StatusSeeOther)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: sessionCookie, Value: "", Path: "/", MaxAge: -1})
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

//---- app pages -------------------------------------------------------------------

func (s *Server) handleAppHome(w http.ResponseWriter, r *http.Request) {
	userID := MustUserID(r.Context())
	ctx := r.Context()

	bal, err := s.Repo.Reports.EstimateBalance(ctx, userID)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	now := s.Now()
	weekStart := domain.WeekStart(now)
	from := weekStart.Format("2006-01-02T15:04:05Z")
	to := weekStart.AddDate(0, 0, 7).Format("2006-01-02T15:04:05Z")
	totals, err := s.Repo.Reports.SumByCategoryForPeriod(ctx, userID, from, to)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	pending, err := s.Repo.Clarifs.ListPending(ctx, userID)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	svg := web.SpendByCategorySVG(totals)
	web.Render(w, "home.html", web.PageData{
		Title:       "Dashboard",
		Balance:     fmt.Sprintf("%.2f", bal),
		WeekSpend:   fmt.Sprintf("%.2f", sumVals(totals)),
		TopCategory: topVal(totals),
		PendingCnt:  len(pending),
		SVGChart:    svg,
	})
}

func (s *Server) handleAppTransactions(w http.ResponseWriter, r *http.Request) {
	userID := MustUserID(r.Context())
	items, _, err := s.Repo.Txns.ListTxns(r.Context(), repoFilter(userID, r.URL.Query().Get("cursor")))
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	rows := make([]web.TxnRow, 0, len(items))
	for _, t := range items {
		rows = append(rows, web.TxnRow{
			ID: t.ID, Amount: t.Amount.String(), Direction: string(t.Direction),
			Counterparty: t.Counterparty, Category: t.Category,
			Source: string(t.Source), OccurredAt: t.OcurredAt.Format("2006-01-02 15:04"),
		})
	}
	web.Render(w, "transactions.html", web.PageData{Title: "Transactions", Txns: rows})
}

func (s *Server) handleAppPatchCategory(w http.ResponseWriter, r *http.Request) {
	userID := MustUserID(r.Context())
	txnID := r.PathValue("id")
	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		httpx.Envelope(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "category is required", "category")
		return
	}
	if _, err := s.Repo.Txns.GetTxn(r.Context(), userID, txnID); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	if s.ApplyCorrection != nil {
		s.ApplyCorrection(r, userID, txnID, category)
	}
	if err := s.Repo.Txns.UpdateCategory(r.Context(), userID, txnID, category, "user"); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	http.Redirect(w, r, "/app/transactions", http.StatusSeeOther)
}

func (s *Server) handleAppClarifications(w http.ResponseWriter, r *http.Request) {
	userID := MustUserID(r.Context())
	items, err := s.Repo.Clarifs.ListPending(r.Context(), userID)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	rows := make([]web.ClarifRow, 0, len(items))
	for _, c := range items {
		rows = append(rows, web.ClarifRow{ID: c.ID, TransactionID: c.TransactionID, Question: c.Question, Options: c.Options})
	}
	web.Render(w, "clarifications.html", web.PageData{Title: "Clarifications", Clarifs: rows})
}

func (s *Server) handleAppRespond(w http.ResponseWriter, r *http.Request) {
	userID := MustUserID(r.Context())
	clarifID := r.PathValue("id")
	answer := strings.TrimSpace(r.FormValue("answer"))
	if answer == "" {
		httpx.Envelope(w, http.StatusUnprocessableEntity, "VALIDATION_ERROR", "answer is required", "answer")
		return
	}
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
	if _, err := s.Repo.Clarifs.ResolveClarification(r.Context(), userID, clarifID, answer); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	if err := s.Repo.Txns.UpdateCategory(r.Context(), userID, target.TransactionID, answer, "user"); err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	if s.ApplyCorrection != nil {
		s.ApplyCorrection(r, userID, target.TransactionID, answer)
	}
	http.Redirect(w, r, "/app/clarifications", http.StatusSeeOther)
}

func (s *Server) handleAppReports(w http.ResponseWriter, r *http.Request) {
	items, err := s.Repo.Reports.ListReports(r.Context(), MustUserID(r.Context()), 25)
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	rows := make([]web.ReportRow, 0, len(items))
	for _, rep := range items {
		rows = append(rows, web.ReportRow{
			ID: rep.ID, Period: fmt.Sprintf("%s – %s", rep.PeriodStart.Format("2006-01-02"), rep.PeriodEnd.Format("2006-01-02")),
			Status: rep.GenerationStatus,
		})
	}
	web.Render(w, "reports.html", web.PageData{Title: "Weekly reports", Reports: rows})
}

func (s *Server) handleAppReport(w http.ResponseWriter, r *http.Request) {
	rep, err := s.Repo.Reports.GetReport(r.Context(), MustUserID(r.Context()), r.PathValue("report_id"))
	if err != nil {
		httpx.WriteError(w, httpx.MapDomainError(err))
		return
	}
	totals := map[string]float64{}
	for k, m := range rep.TotalsByCategory {
		totals[k] = m.Float64()
	}
	web.Render(w, "report.html", web.PageData{
		Title:     "Weekly report",
		Narrative: rep.Narrative,
		SVGChart:  web.SpendByCategorySVG(totals),
		WoWRaw:    rep.WeekOverWeek,
		ReportMeta: fmt.Sprintf("%s – %s (%s)",
			rep.PeriodStart.Format("2006-01-02"), rep.PeriodEnd.Format("2006-01-02"), rep.GenerationStatus),
	})
}

//---- small helpers ---------------------------------------------------------------

func repoFilter(userID, cursor string) repo.TxnListFilter {
	return repo.TxnListFilter{UserID: userID, Limit: 25, Cursor: cursor}
}

func sumVals(m map[string]float64) float64 {
	s := 0.0
	for _, v := range m {
		s += v
	}
	return s
}

func topVal(m map[string]float64) string {
	top, best := "", 0.0
	for k, v := range m {
		if v > best {
			top, best = k, v
		}
	}
	return top
}

func errIsAlreadyExists(err error) bool {
	return errors.Is(err, domain.ErrAlreadyExists)
}
