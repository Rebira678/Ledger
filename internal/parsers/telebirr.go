package parsers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rebira678/ledger/internal/domain"
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
	tbFromRe    = regexp.MustCompile(`(?i)from\s+([A-Z0-9][A-Za-z0-9.'\-& ]+?)(?:\s+to|\s*\(Ref)`)
	tbToRe      = regexp.MustCompile(`(?i)(?:paid|sent)\s+(?:(?:ETB\s*)?[0-9][0-9,]*(?:\.[0-9]{2})?\s*(?:ETB|Birr)?\s+(?:to|for)|\bto)\s+([A-Z0-9][A-Za-z0-9.'\-& ]+?)(?:\s+using|\s+on|\s*\(Ref|\.|$)`)
	tbDateRe    = regexp.MustCompile(`(?i)(?:on\s+)?(\d{1,4})[-/](\d{1,2})[-/](\d{1,4})(?:\s+|T)(\d{1,2}):(\d{2})(?::\d{2})?`)
	tbRefRe     = regexp.MustCompile(`(?i)(?:transaction\s*number|transaction\s*ID|Ref)\s*[:#]?\s*(?:is\s*)?([A-Z0-9]+)`)
	tbBalanceRe = regexp.MustCompile(`(?i)balance\s*[:#]?\s*(?:(?:is\s*)?ETB\s*)?([0-9][0-9,]*(?:\.[0-9]{2})?)`)
)

// Match reports whether the message looks like a Telebirr notification.
func (TelebirrParser) Match(msg domain.RawMessage) bool {
	b := msg.Body
	if telebirrPrefixRe.MatchString(b) {
		return true
	}
	lower := strings.ToLower(b)
	return (strings.Contains(lower, "telebirr") || strings.Contains(lower, "ethio telecom")) && tbDateRe.MatchString(b) && tbAmountRe.MatchString(b)
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

	occurredAt := msg.ReceivedAt
	m := tbDateRe.FindStringSubmatch(body)
	if m != nil {
		if t, err := parseTelebirrTimestamp(m); err == nil {
			occurredAt = t
		}
	}

	ref := ""
	if rm := tbRefRe.FindStringSubmatch(body); rm != nil {
		ref = rm[1]
	}
	if ref == "" {
		return domain.ParsedTransaction{}, fmt.Errorf("parsing Telebirr sms: %w: missing reference", ErrParse)
	}

	var balance *domain.Money
	if rm := tbBalanceRe.FindStringSubmatch(body); rm != nil {
		if bal, err := parseAmount(rm[1]); err == nil {
			balance = &bal
		}
	}

	return domain.ParsedTransaction{
		Amount: amount, Currency: "ETB", Direction: direction,
		Counterparty: counterparty, OccurredAt: occurredAt, Reference: ref,
		Balance: balance,
	}, nil
}

func parseTelebirrTimestamp(m []string) (time.Time, error) {
	if m == nil {
		return time.Time{}, fmt.Errorf("missing timestamp")
	}
	p1, _ := strconv.Atoi(m[1])
	p2, _ := strconv.Atoi(m[2])
	p3, _ := strconv.Atoi(m[3])
	
	var year, month, day int
	if len(m[1]) == 4 { // YYYY-MM-DD
		year, month, day = p1, p2, p3
	} else if len(m[3]) == 4 || len(m[3]) == 2 { // DD/MM/YYYY
		day, month, year = p1, p2, p3
	} else {
		return time.Time{}, fmt.Errorf("invalid date format")
	}
	if year < 100 {
		year += 2000
	}
	hour, _ := strconv.Atoi(m[4])
	minute, _ := strconv.Atoi(m[5])
	loc := time.FixedZone("EAT", 3*3600)
	return time.Date(year, time.Month(month), day, hour, minute, 0, 0, loc), nil
}
