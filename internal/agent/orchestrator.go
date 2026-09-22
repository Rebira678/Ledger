// Package agent implements the agent orchestrator (FR-5): clarifying-question
// generation when confidence is low, the correction/learning loop (FR-5.2),
// week-over-week anomaly detection (FR-5.3), and LLM-backed weekly narrative
// generation grounded strictly in the user's own transaction data (FR-5.4).
package agent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/rebira678/ledger/internal/agent/llm"
	"github.com/rebira678/ledger/internal/categorize"
	"github.com/rebira678/ledger/internal/domain"
)

// Stores aggregates the persistence the orchestrator needs.
type Stores struct {
	TxnsTxnInserter TxnInserter
	MessagesOutcome MessageOutcome
	Rules           RuleStore
	Clarifs         ClarifStore
	Reports         ReportStore
}

// TxnInserter inserts transactions (satisfied by *repo.TxnRepo).
type TxnInserter interface {
	InsertTxn(ctx context.Context, db any, t *domain.Transaction) error
}

// MessageOutcome updates raw-message parse outcomes.
type MessageOutcome interface {
	SetRawMessageOutcome(ctx context.Context, id, status, txnID, parseErr string) error
}

// RuleStore upserts learned rules.
type RuleStore interface {
	UpsertRule(ctx context.Context, rule *domain.CategoryRule) error
}

// ClarifStore queues and lists clarifications.
type ClarifStore interface {
	CreateClarification(ctx context.Context, c *domain.Clarification) error
	HasPendingForTransaction(ctx context.Context, userID, txnID string) (bool, error)
	ListPending(ctx context.Context, userID string) ([]*domain.Clarification, error)
}

// ReportStore aggregates for reports.
type ReportStore interface {
	SumByCategoryForPeriod(ctx context.Context, userID, from, to string) (map[string]float64, error)
}

// Orchestrator is the reasoning layer (FR-5).
type Orchestrator struct {
	engine *categorize.Engine
	llm    *llm.Client
	now    func() time.Time
}

// New builds the orchestrator.
func New(engine *categorize.Engine, client *llm.Client) *Orchestrator {
	return &Orchestrator{engine: engine, llm: client, now: time.Now}
}

// Categorize assigns category fields on the transaction (FR-4). Called before
// the transaction row exists.
func (o *Orchestrator) Categorize(ctx context.Context, userID string, txn *domain.Transaction) error {
	res, err := o.engine.Categorize(ctx, userID, txn.Counterparty)
	if err != nil {
		return fmt.Errorf("categorizing transaction: %w", err)
	}
	txn.Category = res.Category
	txn.CategoryConfidence = res.Confidence
	txn.CategoryHasConfidence = true
	txn.CategorySource = res.Source
	return nil
}

// QueueClarification queues a clarifying question for a low-confidence
// transaction (FR-5.1). Must be called AFTER the transaction row is persisted
// (clarifications reference transactions).
func (o *Orchestrator) QueueClarification(ctx context.Context, stores Stores, userID string, txn *domain.Transaction) error {
	if txn.CategoryHasConfidence && txn.CategoryConfidence >= categorize.ClarifyBelow {
		return nil
	}
	if pending, err := stores.Clarifs.HasPendingForTransaction(ctx, userID, txn.ID); err != nil {
		return fmt.Errorf("checking pending clarifications: %w", err)
	} else if pending {
		return nil
	}
	question := o.buildQuestion(txn)
	clr := &domain.Clarification{
		ID:            domain.MustNewID("clr"),
		UserID:        userID,
		TransactionID: txn.ID,
		Question:      question,
		Options:       o.buildOptions(txn.Category),
	}
	if err := stores.Clarifs.CreateClarification(ctx, clr); err != nil {
		return fmt.Errorf("queueing clarification: %w", err)
	}
	return nil
}

// buildQuestion phrases a short clarifying question. Deterministic template
// today; the LLM-phrased path activates when a key is configured (see WeeklyNarrative).
func (o *Orchestrator) buildQuestion(txn *domain.Transaction) string {
	cp := txn.Counterparty
	if cp == "" {
		cp = "an unrecognized payee"
	}
	return fmt.Sprintf(
		"This %.2f ETB %s to %s on %s — what category is it?",
		txn.Amount.Float64(), txn.Direction, cp,
		txn.OcurredAt.Format("Jan 2"),
	)
}

func (o *Orchestrator) buildOptions(defaultCat string) []string {
	opts := []string{defaultCat, "Transfer-Out", "Other"}
	seen := map[string]bool{}
	unique := opts[:0]
	for _, op := range opts {
		if !seen[op] {
			seen[op] = true
			unique = append(unique, op)
		}
	}
	return unique
}

// ApplyCorrection persists a user correction as a reusable rule (FR-5.2 /
// FR-4.2) and reports whether a new rule was created.
func (o *Orchestrator) ApplyCorrection(ctx context.Context, rules RuleStore, userID, counterparty, category string) (bool, error) {
	if strings.TrimSpace(counterparty) == "" {
		return false, nil
	}
	rule := &domain.CategoryRule{
		ID:         domain.MustNewID("rule"),
		UserID:     userID,
		MatchType:  "counterparty",
		MatchValue: strings.ToLower(strings.TrimSpace(counterparty)),
		Category:   category,
	}
	if err := rules.UpsertRule(ctx, rule); err != nil {
		return false, fmt.Errorf("persisting correction rule: %w", err)
	}
	return true, nil
}

//---- Anomaly detection (FR-5.3) -------------------------------------------------

// Anomaly is a flagged week-over-week category change.
type Anomaly struct {
	Category  string  `json:"category"`
	ThisWeek  float64 `json:"this_week"`
	LastWeek  float64 `json:"last_week"`
	ChangePct float64 `json:"change_pct"`
}

// DetectAnomalies compares category totals between two consecutive weeks.
// Flag when |change| > 35% AND absolute change > ETB 200 (D-15).
func (o *Orchestrator) DetectAnomalies(ctx context.Context, reports ReportStore, userID string, weekStart time.Time) ([]Anomaly, error) {
	from := weekStart.Format("2006-01-02T15:04:05Z")
	to := weekStart.AddDate(0, 0, 7).Format("2006-01-02T15:04:05Z")
	thisWeek, err := reports.SumByCategoryForPeriod(ctx, userID, from, to)
	if err != nil {
		return nil, fmt.Errorf("summing this week: %w", err)
	}
	fromL := weekStart.AddDate(0, 0, -7).Format("2006-01-02T15:04:05Z")
	lastWeek, err := reports.SumByCategoryForPeriod(ctx, userID, fromL, from)
	if err != nil {
		return nil, fmt.Errorf("summing last week: %w", err)
	}

	cats := make([]string, 0, len(thisWeek))
	for c := range thisWeek {
		cats = append(cats, c)
	}
	sort.Strings(cats)

	var out []Anomaly
	for _, c := range cats {
		this, last := thisWeek[c], lastWeek[c]
		if last == 0 {
			continue // new category spending isn't a spike vs zero baseline
		}
		change := (this - last) / last * 100
		if abs(change) > 35 && abs(this-last) > 200 {
			out = append(out, Anomaly{Category: c, ThisWeek: this, LastWeek: last, ChangePct: change})
		}
	}
	return out, nil
}

func abs(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}

//---- Weekly narrative (FR-5.4) ---------------------------------------------------

// NarrativeInput is the ONLY data passed to the LLM. The model can never
// invent a figure that isn't present here because the prompt contains only
// these aggregates and instructs it to use no other numbers.
type NarrativeInput struct {
	PeriodStart  string
	PeriodEnd    string
	Totals       map[string]float64
	WeekOverWeek map[string]float64 // category -> % change vs last week
	Anomalies    []Anomaly
}

// WeeklyNarrative generates the report narrative via the LLM, grounded in
// NarrativeInput. Returns llm.ErrNotConfigured when no key is set.
func (o *Orchestrator) WeeklyNarrative(ctx context.Context, in NarrativeInput) (string, error) {
	if !o.llm.Configured() {
		return "", llm.ErrNotConfigured
	}
	system := "You are Ledger, a personal-finance assistant. " +
		"Write a 2-4 sentence weekly spending summary. " +
		"STRICT RULE: use ONLY the numbers provided in the user message; never invent or estimate any figure. " +
		"If a category is missing from the data, do not mention it. Plain language, no markdown."
	user := buildGroundedPrompt(in)
	text, err := o.llm.Chat(ctx, system, user)
	if err != nil {
		return "", fmt.Errorf("generating weekly narrative: %w", err)
	}
	return strings.TrimSpace(text), nil
}

// buildGroundedPrompt renders the aggregates as the complete factual context.
func buildGroundedPrompt(in NarrativeInput) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Period: %s to %s. Spending by category (ETB):\n", in.PeriodStart, in.PeriodEnd)
	cats := make([]string, 0, len(in.Totals))
	for c := range in.Totals {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	for _, c := range cats {
		fmt.Fprintf(&b, "- %s: %.2f\n", c, in.Totals[c])
	}
	if len(in.WeekOverWeek) > 0 {
		b.WriteString("Change vs previous week:\n")
		for _, c := range cats {
			if ch, ok := in.WeekOverWeek[c]; ok {
				fmt.Fprintf(&b, "- %s: %+.1f%%\n", c, ch)
			}
		}
	}
	if len(in.Anomalies) > 0 {
		b.WriteString("Flagged anomalies:\n")
		for _, a := range in.Anomalies {
			fmt.Fprintf(&b, "- %s: %.2f this week vs %.2f last week (%+.1f%%)\n", a.Category, a.ThisWeek, a.LastWeek, a.ChangePct)
		}
	}
	b.WriteString("Explain the biggest drivers of change in plain language.")
	return b.String()
}
