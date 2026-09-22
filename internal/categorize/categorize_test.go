package categorize

import (
	"context"
	"testing"

	"github.com/rebira678/ledger/internal/domain"
)

type fakeRules struct{ rules []*domain.CategoryRule }

func (f *fakeRules) ListRules(_ context.Context, _ string) ([]*domain.CategoryRule, error) {
	return f.rules, nil
}

func TestUnit_Categorize_Tiers(t *testing.T) {
	engine := NewEngine(&fakeRules{rules: []*domain.CategoryRule{
		{MatchType: "counterparty", MatchValue: "landlord abebe", Category: "Rent"},
		{MatchType: "keyword", MatchValue: "brewery", Category: "Dining"},
	}})

	cases := []struct {
		name         string
		counterparty string
		wantCat      string
		wantConf     float64
		wantSource   string
	}{
		{"learned counterparty rule wins", "Landlord Abebe", "Rent", ConfLearnedRule, "user"},
		{"learned keyword rule", "Addis Brewery Co", "Dining", ConfLearnedRule, "user"},
		{"system keyword airtime", "ETHIO TELECOM AIRTIME", "Airtime", ConfKeyword, "system"},
		{"system keyword transport", "RIDE Taxi 1234", "Transport", ConfKeyword, "system"},
		{"default low confidence", "Random Kiosk", DefaultCategory, ConfDefault, "system"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := engine.Categorize(context.Background(), "usr_x", tc.counterparty)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Category != tc.wantCat || got.Confidence != tc.wantConf || got.Source != tc.wantSource {
				t.Fatalf("got %+v, want cat=%s conf=%.2f src=%s", got, tc.wantCat, tc.wantConf, tc.wantSource)
			}
		})
	}
}

func TestUnit_Categorize_LowConfidenceTriggersClarifyThreshold(t *testing.T) {
	engine := NewEngine(&fakeRules{})
	res, err := engine.Categorize(context.Background(), "usr_x", "unknown shop")
	if err != nil {
		t.Fatal(err)
	}
	if res.Confidence >= ClarifyBelow {
		t.Fatalf("default confidence %.2f should sit below clarify threshold %.2f", res.Confidence, ClarifyBelow)
	}
}
