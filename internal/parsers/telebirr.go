package parsers

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/abel-gezahegn/ledger/internal/domain"
)

// TelebirrParser parses Ethio Telecom Telebirr wallet SMS notifications.
//
// Fixture formats (see fixtures/ and D-12 — representative, to be validated
// against real captured samples):
//
//	Telebirr: You have received 500.00 ETB from ALMAZ TESFAYE (Ref: P1234567890)
//	on 14/09/2026 07:41. Balance: 1,250.00 ETB
//
//	Telebirr: You have paid 45.00 ETB to SAFARICOM DATA (Ref: T987654321)
//	on 14/09/2026 08:15. Balance: 1,205.00 ETB
//
//	Telebirr: You have sent 250.00 ETB to BEKELE HAILE (Ref: T5555444433)
//	on 13/09/2026 21:02. Balance: 955.00 ETB
type TelebirrParser struct{}

// Compile-time interface check.
var _ Parser = (*TelebirrParser)(nil)

// SenderIDs returns the Telebirr sender identifiers.
func (TelebirrParser) SenderIDs() []string { return []string{"TELEBIRR", "127-99", "ETHIO-TELECOM"} }

var telebirrPrefixRe = regexp.MustCompile(`(?i)^\s*Telebirr\s*[:\-]\s*`)

var (
	tbAmountRe  = regexp.MustCompile(`(?i)(?:received|paid|sent)\s+(?:ETB\s*)?([0-9][0-9,]*(?:\.[0-9]{2})?)\s*(?:ETB|Birr)?`)
	tbFromRe    = regexp.MustCompile(`(?i)from\s+([A-Z][A-Za-z.'\- ]+?)\s*\(Ref`)
	tbToRe      = regexp.MustCompile(`(?i)(?:paid|sent)\s+(?:ETB\s*)?[0-9][0-9,]*(?:\.[0-9]{2})?\s*(?:ETB|Birr)?\s+to\s+([A-Z][A-Za-z.'\- ]+?)\s*\(Ref`)
	tbDateRe    = regexp.MustCompile(`(\d{1,2})/(\d{1,2})/(\d{4})\s+(\d{1,2}):(\d{2})`)
	tbRefRe     = regexp.MustCompile(`(?i)Ref\s*[:#]?\s*([A-Z0-9]+)`)
	tbBalanceRe = regexp.MustCompile(`(?i)balance\s*[:#]?\s*(?:ETB\s*)?([0-9][0-9,]*(?:\.[0-9]{2})?)`)
)

// Match reports whether the message looks like a Telebirr notification.
func (TelebirrParser) Match(msg domain.RawMessage) bool {
	b := msg.Body
	if telebirrPrefixRe.MatchString(b) {
		return true
	}
	lower := strings.ToLower(b)
	return strings.Contains(lower, "telebirr") && tbDateRe.MatchString(b) && tbAmountRe.MatchString(b)
}

// Parse extracts the normalized transaction from a Telebirr SMS.
func (p TelebirrParser) Parse(msg domain.RawMessage) (domain.ParsedTransaction, error) {
	body := telebirrPrefixRe.ReplaceAllString(msg.Body, "")

	amountStr, err := firstMatch(tbAmountRe, body)
	if err != nil {
		return domain.ParsedTransaction{}, fmt.Errorf("parsing Telebirr sms: %w: %v", ErrParse, err)
	}
	amount, err := parseAmount(amountStr)
	if err != nil {
		return domain.ParsedTransaction{}, fmt.Errorf("parsing Telebirr sms: %w: %v", ErrParse, err)
	}

	lower := strings.ToLower(body)
	direction := domain.DirectionDebit // "paid"/"sent"
	if strings.Contains(lower, "received") {
		direction = domain.DirectionCredit
	}

	counterparty := ""
	if direction == domain.DirectionCredit {
		if m := tbFromRe.FindStringSubmatch(body); m != nil {
			counterparty = cleanName(m[1])
		}
	} else if m := tbToRe.FindStringSubmatch(body); m != nil {
		counterparty = cleanName(m[1])
	}

	m := tbDateRe.FindStringSubmatch(body)
	if m == nil {
		return domain.ParsedTransaction{}, fmt.Errorf("parsing Telebirr sms: %w: missing timestamp", ErrParse)
	}
	occurredAt, err := parseCBETimestamp(m) // same DD/MM/YYYY HH:MM layout, EAT
	if err != nil {
		return domain.ParsedTransaction{}, fmt.Errorf("parsing Telebirr sms: %w: %v", ErrParse, err)
	}

	ref := ""
	if rm := tbRefRe.FindStringSubmatch(body); rm != nil {
		ref = rm[1]
	}
	if ref == "" {
		return domain.ParsedTransaction{}, fmt.Errorf("parsing Telebirr sms: %w: missing reference", ErrParse)
	}

	return domain.ParsedTransaction{
		Amount: amount, Currency: "ETB", Direction: direction,
		Counterparty: counterparty, OccurredAt: occurredAt, Reference: ref,
	}, nil
}
