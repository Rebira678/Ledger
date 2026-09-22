// Package categorize implements the rule-based categorization engine (FR-4):
// learned user rules → counterparty match → keyword rules → default, each with
// a confidence score that drives the clarifying-question logic (FR-5.1).
package categorize

import (
	"context"
	"strings"

	"github.com/rebira678/ledger/internal/domain"
)

// DefaultCategory is assigned when nothing matches.
const DefaultCategory = "Other"

// Confidence threshold below which the agent queues a clarifying question.
const ClarifyBelow = 0.60

// Confidence scores per match tier (deterministic, explainable; D-13).
const (
	ConfLearnedRule  = 0.95
	ConfCounterparty = 0.80
	ConfKeyword      = 0.70
	ConfDefault      = 0.30
)

// KeywordRule maps a lowercase keyword to a category.
type KeywordRule struct {
	Keyword  string
	Category string
}

// DefaultKeywordRules are the system-wide keyword rules (FR-4.1).
func DefaultKeywordRules() []KeywordRule {
	return []KeywordRule{
		{"airtime", "Airtime"}, {"telecom", "Airtime"}, {"data pack", "Airtime"},
		{"safaricom", "Airtime"}, {"ethio telecom", "Airtime"},
		{"taxi", "Transport"}, {"ride", "Transport"}, {"fuel", "Transport"},
		{"transport", "Transport"}, {"anbessa", "Transport"},
		{"supermarket", "Groceries"}, {"market", "Groceries"}, {"grocery", "Groceries"},
		{"restaurant", "Dining"}, {"cafe", "Dining"}, {"hotel", "Dining"},
		{"rent", "Rent"}, {"electric", "Utilities"}, {"water", "Utilities"},
		{"school", "Education"}, {"university", "Education"}, {"tuition", "Education"},
		{"hospital", "Health"}, {"pharmacy", "Health"}, {"clinic", "Health"},
		{"salary", "Salary"}, {"payroll", "Salary"},
		{"transfer", "Transfer-Out"}, {"send", "Transfer-Out"},
	}
}

// Engine categorizes transactions using learned rules then system keywords.
type Engine struct {
	rules RuleSource
}

// RuleSource supplies learned rules (satisfied by *repo.RuleRepo).
type RuleSource interface {
	ListRules(ctx context.Context, userID string) ([]*domain.CategoryRule, error)
}

// NewEngine builds the categorization engine.
func NewEngine(rules RuleSource) *Engine {
	return &Engine{rules: rules}
}

// Result carries the category and its confidence.
type Result struct {
	Category   string
	Confidence float64
	Source     string // "system" | "user"
}

// Categorize assigns a category to a transaction for a user (FR-4.1/4.3).
func (e *Engine) Categorize(ctx context.Context, userID string, counterparty string) (Result, error) {
	cp := strings.ToLower(strings.TrimSpace(counterparty))

	// 1. Learned rules (FR-4.2): counterparty-exact first, then keyword.
	if e.rules != nil {
		rules, err := e.rules.ListRules(ctx, userID)
		if err != nil {
			return Result{}, err
		}
		for _, r := range rules {
			mv := strings.ToLower(r.MatchValue)
			if r.MatchType == "counterparty" && mv != "" && mv == cp {
				return Result{Category: r.Category, Confidence: ConfLearnedRule, Source: "user"}, nil
			}
		}
		for _, r := range rules {
			mv := strings.ToLower(r.MatchValue)
			if r.MatchType == "keyword" && mv != "" && strings.Contains(cp, mv) {
				return Result{Category: r.Category, Confidence: ConfLearnedRule, Source: "user"}, nil
			}
		}
	}

	// 2. System keyword rules.
	for _, kr := range DefaultKeywordRules() {
		if strings.Contains(cp, kr.Keyword) {
			return Result{Category: kr.Category, Confidence: ConfKeyword, Source: "system"}, nil
		}
	}

	// 3. Default — low confidence, may trigger a clarifying question.
	return Result{Category: DefaultCategory, Confidence: ConfDefault, Source: "system"}, nil
}
