package parsers

import (
	"errors"
	"testing"
	"time"

	"github.com/rebira678/ledger/internal/domain"
)

// NOTE (D-12): REPRESENTATIVE synthetic Telebirr fixtures — real redacted
// samples required for genuine accuracy validation.

func TestUnit_TelebirrParser_Parse(t *testing.T) {
	loc := time.FixedZone("EAT", 3*3600)
	p := TelebirrParser{}

	cases := []struct {
		name    string
		body    string
		want    domain.ParsedTransaction
		wantErr bool
	}{
		{
			name: "wallet credit",
			body: "Telebirr: You have received 500.00 ETB from ALMAZ TESFAYE (Ref: P1234567890) on 14/09/2026 07:41. Balance: 1,250.00 ETB",
			want: domain.ParsedTransaction{
				Amount: domain.Money{Units: 500}, Currency: "ETB",
				Direction: domain.DirectionCredit, Counterparty: "ALMAZ TESFAYE",
				OccurredAt: time.Date(2026, 9, 14, 7, 41, 0, 0, loc), Reference: "P1234567890",
			},
		},
		{
			name: "wallet payment",
			body: "Telebirr: You have paid 45.00 ETB to SAFARICOM DATA (Ref: T987654321) on 14/09/2026 08:15. Balance: 1,205.00 ETB",
			want: domain.ParsedTransaction{
				Amount: domain.Money{Units: 45, Cents: 0}, Currency: "ETB",
				Direction: domain.DirectionDebit, Counterparty: "SAFARICOM DATA",
				OccurredAt: time.Date(2026, 9, 14, 8, 15, 0, 0, loc), Reference: "T987654321",
			},
		},
		{
			name: "wallet send",
			body: "Telebirr: You have sent 250.00 ETB to BEKELE HAILE (Ref: T5555444433) on 13/09/2026 21:02. Balance: 955.00 ETB",
			want: domain.ParsedTransaction{
				Amount: domain.Money{Units: 250}, Currency: "ETB",
				Direction: domain.DirectionDebit, Counterparty: "BEKELE HAILE",
				OccurredAt: time.Date(2026, 9, 13, 21, 2, 0, 0, loc), Reference: "T5555444433",
			},
		},
		{
			name:    "missing ref",
			body:    "Telebirr: You have received 10.00 ETB from A B on 14/09/2026 07:41.",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			msg := domain.RawMessage{SenderID: "TELEBIRR", Body: tc.body}
			got, err := p.Parse(msg)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got %+v", got)
				}
				if !errors.Is(err, ErrParse) {
					t.Fatalf("error should wrap ErrParse, got %v", err)
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

func TestUnit_Registry_RoutesBySenderAndShape(t *testing.T) {
	reg := NewRegistry(CBEParser{}, TelebirrParser{})

	// By sender ID.
	pt, err := reg.ParseSender("CBE", "CBE: You have received 5.00 ETB from A B on 01/01/2026 09:00. Ref: X1. Balance: 5.00 ETB", time.Now())
	if err != nil || pt.Reference != "X1" {
		t.Fatalf("ParseSender by sender id: got %+v, err %v", pt, err)
	}

	// By shape when sender unknown.
	pt, err = reg.ParseSender("UNKNOWN-77", "Telebirr: You have received 7.00 ETB from C D (Ref: P2) on 02/01/2026 09:00. Balance: 7.00 ETB", time.Now())
	if err != nil || pt.Reference != "P2" {
		t.Fatalf("ParseSender by shape: got %+v, err %v", pt, err)
	}

	// Unmatched → ErrUnmatchedFormat for the review queue (FR-3.3).
	if _, err := reg.ParseSender("UNKNOWN", "hello are you there", time.Now()); !errors.Is(err, ErrUnmatchedFormat) {
		t.Fatalf("expected ErrUnmatchedFormat, got %v", err)
	}
}
