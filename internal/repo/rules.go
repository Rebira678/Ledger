package repo

import (
	"context"

	"github.com/rebira678/ledger/internal/domain"
)

// RuleRepo persists learned categorization rules (FR-4.2 / FR-5.2).
type RuleRepo struct{ db DB }

// UpsertRule inserts or updates a user-scoped categorization rule.
func (r *RuleRepo) UpsertRule(ctx context.Context, rule *domain.CategoryRule) error {
	if _, err := r.db.ExecContext(ctx,
		`INSERT INTO category_rules (id, user_id, match_type, match_value, category)
		 VALUES ($1,$2,$3,$4,$5)
		 ON CONFLICT (user_id, match_type, match_value)
		 DO UPDATE SET category = EXCLUDED.category`,
		rule.ID, rule.UserID, rule.MatchType, rule.MatchValue, rule.Category); err != nil {
		return wrap("upserting category rule", mapPgError(err))
	}
	return nil
}

// ListRules returns all rules for a user.
func (r *RuleRepo) ListRules(ctx context.Context, userID string) ([]*domain.CategoryRule, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, match_type, match_value, category, created_at
		 FROM category_rules WHERE user_id = $1 ORDER BY created_at`, userID)
	if err != nil {
		return nil, wrap("listing category rules", err)
	}
	defer rows.Close()
	var out []*domain.CategoryRule
	for rows.Next() {
		rule := &domain.CategoryRule{}
		if err := rows.Scan(&rule.ID, &rule.UserID, &rule.MatchType, &rule.MatchValue, &rule.Category, &rule.CreatedAt); err != nil {
			return nil, wrap("scanning category rule", err)
		}
		out = append(out, rule)
	}
	return out, rows.Err()
}
