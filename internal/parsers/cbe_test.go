package parsers

import (
	"errors"
	"testing"
	"time"

	"github.com/abel-gezahegn/ledger/internal/domain"
)

// NOTE (D-12): these fixtures are REPRESENTATIVE synthetic samples written from
// the documented CBE SMS layout. They are clearly labeled as fixtures, not real
// captured messages; real redacted samples are required to validate accuracy.

func TestUnit_CBEParser_Parse(t *testing.T) {
	loc := time.FixedZone("EAT", 3*3600)
	p := CBEParser{}

	cases := []struct {
		name    string
		body    string
		want    domain.ParsedTransaction
		wantErr bool
	}{
		{
			name: "credit from person",
			body: "CBE: You have received 500.00 ETB from ALMAZ TESFAYE (Acc 1000123456789) on 14/09/2026 07:41. Ref: TXN12345678. Balance: 4,820.50 ETB",
			want: domain.ParsedTransaction{
				Amount: domain.Money{Units: 500, Cents: 0}, Currency: "ETB",
				Direction: domain.DirectionCredit, Counterparty: "ALMAZ TESFAYE",
				OccurredAt: time.Date(2026, 9, 14, 7, 41, 0, 0, loc), Reference: "TXN12345678",
			},
		},
		{
			name: "debit payment to merchant",
			body: "CBE: You have paid 3,200.00 ETB to A A REAL ESTATE using CBE Birr on 14/09/2026 18:30. Ref: PAY98765432. Balance: 1,620.50 ETB",
			want: domain.ParsedTransaction{
				Amount: domain.Money{Units: 3200, Cents: 0}, Currency: "ETB",
				Direction: domain.DirectionDebit, Counterparty: "A A REAL ESTATE",
				OccurredAt: time.Date(2026, 9, 14, 18, 30, 0, 0, loc), Reference: "PAY98765432",
			},
		},
		{
			name: "transfer out to person",
			body: "CBE: Transfer of 1,000.00 ETB from your account to BEKELE HAILE (1000234567890) on 12/09/2026 12:05. Ref: TRN55443322. Balance: 3,000.00 ETB",
			want: domain.ParsedTransaction{
				Amount: domain.Money{Units: 1000, Cents: 0}, Currency: "ETB",
				Direction: domain.DirectionDebit, Counterparty: "BEKELE HAILE",
				OccurredAt: time.Date(2026, 9, 12, 12, 5, 0, 0, loc), Reference: "TRN55443322",
			},
		},
		{
			name:    "missing reference",
			body:    "CBE: You have received 100.00 ETB from ABEBE BEKELE on 14/09/2026 07:41. Balance: 50.00 ETB",
			wantErr: true,
		},
		{
			name:    "garbage body",
			body:    "CBE: nothing usable here",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			msg := domain.RawMessage{SenderID: "CBE", Body: tc.body}
			got, err := p.Parse(msg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				if !errors.Is(err, ErrParse) {
					t.Fatalf("error should wrap ErrParse, got %v", err)
				}
				if got, want := err.Error()[:16], "parsing CBE sms:"; got != want {
					t.Fatalf("error should carry bank context: got %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.Amount != tc.want.Amount {
				t.Errorf("amount = %s, want %s", got.Amount, tc.want.Amount)
			}
			if got.Direction != tc.want.Direction {
				t.Errorf("direction = %s, want %s", got.Direction, tc.want.Direction)
			}
			if got.Counterparty != tc.want.Counterparty {
				t.Errorf("counterparty = %q, want %q", got.Counterparty, tc.want.Counterparty)
			}
			if !got.OccurredAt.Equal(tc.want.OccurredAt) {
				t.Errorf("occurred_at = %v, want %v", got.OccurredAt, tc.want.OccurredAt)
			}
			if got.Reference != tc.want.Reference {
				t.Errorf("reference = %q, want %q", got.Reference, tc.want.Reference)
			}
		})
	}
}

func TestUnit_CBEParser_Match(t *testing.T) {
	p := CBEParser{}
	cases := []struct {
		body string
		want bool
	}{
		{"CBE: You have received 10.00 ETB from A B on 01/01/2026 10:00. Ref: X12345678.", true},
		{"random personal sms", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := p.Match(domain.RawMessage{Body: tc.body}); got != tc.want {
			t.Errorf("Match(%q) = %v, want %v", tc.body, got, tc.want)
		}
	}
}
