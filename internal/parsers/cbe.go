package parsers

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/rebira678/ledger/internal/domain"
)

// CBEParser parses Commercial Bank of Ethiopia SMS notifications.
//
// Fixture formats (see fixtures/ and D-12 — representative, to be validated
// against real captured samples):
//
//	CBE: You have received 500.00 ETB from ALMAZ TESFAYE (Acc 1000123456789)
//	on 14/09/2026 07:41. Ref: TXN12345678. Balance: 4,820.50 ETB
//
//	CBE: You have paid 3200.00 ETB to A A REAL ESTATE using CBE Birr
//	on 14/09/2026 18:30. Ref: PAY98765432. Balance: 1,620.50 ETB
//
//	CBE: Transfer of 1,000.00 ETB from your account to BEKELE HAILE
//	(1000234567890) on 12/09/2026 12:05. Ref: TRN55443322. Balance: 3,000.00 ETB
type CBEParser struct{}

// Compile-time interface check.
var _ Parser = (*CBEParser)(nil)

// SenderIDs returns the CBE sender identifiers.
func (CBEParser) SenderIDs() []string { return []string{"CBE", "CBE-BIRR", "127"} }

// senderRe tolerates a leading "CBE:" prefix inside the body.
var cbePrefixRe = regexp.MustCompile(`(?i)^\s*CBE\s*[:\-]\s*`)

var (
	cbeAmountRe  = regexp.MustCompile(`(?i)(?:received|paid|transfer(?: of)?)\s+(?:ETB\s*)?([0-9][0-9,]*(?:\.[0-9]{2})?)\s*(?:ETB|Birr)?`)
	cbeFromRe    = regexp.MustCompile(`(?i)from\s+(?:your account\s+)?(?:to\s+)?([A-Z][A-Za-z.'\- ]+?)(?:\s*\(|\s+using\s+|\s+on\s+|\.|$)`)
	cbeToRe      = regexp.MustCompile(`(?i)(?:paid\s+(?:[0-9][0-9,]*(?:\.[0-9]{2})?)\s*(?:ETB|Birr)?\s+to|to\s+)([A-Z][A-Za-z.'\- ]+?)(?:\s*\(|\s+using\s+|\s+on\s+|\.|$)`)
	cbeDateRe    = regexp.MustCompile(`(\d{1,2})/(\d{1,2})/(\d{4})\s+(\d{1,2}):(\d{2})`)
	cbeRefRe     = regexp.MustCompile(`(?i)Ref(?:erence)?\s*[:#]?\s*([A-Z0-9]+)`)
	cbeBalanceRe = regexp.MustCompile(`(?i)balance\s*[:#]?\s*(?:ETB\s*)?([0-9][0-9,]*(?:\.[0-9]{2})?)`)
)

// Match reports whether the message looks like a CBE notification.
func (CBEParser) Match(msg domain.RawMessage) bool {
	b := msg.Body
	if cbePrefixRe.MatchString(b) {
		return true
	}
	lower := strings.ToLower(b)
	return strings.Contains(lower, "cbe") && cbeDateRe.MatchString(b) && cbeAmountRe.MatchString(b)
}

// Parse extracts the transaction (FR-3.2: amount, direction, counterparty,
// timestamp, reference; balance is parsed and carried in Reference context
// when present).
func (p CBEParser) Parse(msg domain.RawMessage) (domain.ParsedTransaction, error) {
	body := cbePrefixRe.ReplaceAllString(msg.Body, "")

	amountStr, err := firstMatch(cbeAmountRe, body)
	if err != nil {
		return domain.ParsedTransaction{}, fmt.Errorf("parsing CBE sms: %w: %v", ErrParse, err)
	}
	amount, err := parseAmount(amountStr)
	if err != nil {
		return domain.ParsedTransaction{}, fmt.Errorf("parsing CBE sms: %w: %v", ErrParse, err)
	}

	// Direction: credit for "received", debit for paid/transfers.
	lower := strings.ToLower(body)
	direction := domain.DirectionDebit
	if strings.Contains(lower, "received") || strings.Contains(lower, "deposited") {
		direction = domain.DirectionCredit
	}

	counterparty := ""
	if direction == domain.DirectionCredit {
		if m := cbeFromRe.FindStringSubmatch(body); m != nil {
			counterparty = cleanName(m[1])
		}
	} else if m := cbeToRe.FindStringSubmatch(body); m != nil {
		counterparty = cleanName(m[1])
	}

	occurredAt, err := parseCBETimestamp(cbeDateRe.FindStringSubmatch(body))
	if err != nil {
		return domain.ParsedTransaction{}, fmt.Errorf("parsing CBE sms: %w: %v", ErrParse, err)
	}

	ref := ""
	if m := cbeRefRe.FindStringSubmatch(body); m != nil {
		ref = m[1]
	}
	if ref == "" {
		return domain.ParsedTransaction{}, fmt.Errorf("parsing CBE sms: %w: missing reference", ErrParse)
	}

	return domain.ParsedTransaction{
		Amount: amount, Currency: "ETB", Direction: direction,
		Counterparty: counterparty, OccurredAt: occurredAt, Reference: ref,
	}, nil
}

func parseCBETimestamp(m []string) (time.Time, error) {
	if m == nil {
		return time.Time{}, fmt.Errorf("missing timestamp")
	}
	day, _ := strconv.Atoi(m[1])
	month, _ := strconv.Atoi(m[2])
	year, _ := strconv.Atoi(m[3])
	hour, _ := strconv.Atoi(m[4])
	minute, _ := strconv.Atoi(m[5])
	// CBE SMS timestamps are local Ethiopian time (UTC+3).
	loc := time.FixedZone("EAT", 3*3600)
	return time.Date(year, time.Month(month), day, hour, minute, 0, 0, loc), nil
}

// parseAmount normalizes "4,820.50" into exact Money.
func parseAmount(s string) (domain.Money, error) {
	s = strings.ReplaceAll(s, ",", "")
	if s == "" {
		return domain.Money{}, fmt.Errorf("empty amount")
	}
	parts := strings.SplitN(s, ".", 2)
	units, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return domain.Money{}, fmt.Errorf("invalid amount %q: %v", s, err)
	}
	var cents int64
	if len(parts) == 2 {
		if len(parts[1]) != 2 {
			return domain.Money{}, fmt.Errorf("invalid cents in %q", s)
		}
		cents, err = strconv.ParseInt(parts[1], 10, 64)
		if err != nil {
			return domain.Money{}, fmt.Errorf("invalid cents in %q: %v", s, err)
		}
	}
	return domain.Money{Units: units, Cents: cents}, nil
}

func firstMatch(re *regexp.Regexp, body string) (string, error) {
	m := re.FindStringSubmatch(body)
	if m == nil {
		return "", fmt.Errorf("pattern %q not found", re.String())
	}
	return m[1], nil
}

// cleanName trims and collapses whitespace in extracted names.
func cleanName(s string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(s)), " ")
}
