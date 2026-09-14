package domain

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// Direction of a transaction. Amount is always positive; direction carries meaning.
type Direction string

const (
	DirectionDebit  Direction = "debit"
	DirectionCredit Direction = "credit"
)

func (d Direction) Valid() bool { return d == DirectionDebit || d == DirectionCredit }

// Source of a transaction.
type Source string

const (
	SourceSMS       Source = "sms"
	SourceStatement Source = "statement"
)

// Money is an exact decimal amount, e.g. 500.00, always positive.
type Money struct {
	Units int64 // whole currency units
	Cents int64 // fractional cents (0-99); sign always positive
}

// ParseMoney parses "500.00" into exact cents.
func ParseMoney(s string) (Money, error) {
	var m Money
	if _, err := fmt.Sscanf(s, "%d.%02d", &m.Units, &m.Cents); err != nil {
		// try integer-only
		var units int64
		if _, err2 := fmt.Sscanf(s, "%d", &units); err2 != nil {
			return Money{}, fmt.Errorf("parsing money %q: %w", s, err)
		}
		return Money{Units: units, Cents: 0}, nil
	}
	if m.Cents < 0 || m.Cents > 99 {
		return Money{}, fmt.Errorf("parsing money %q: invalid cents", s)
	}
	return m, nil
}

// String renders money as "500.00".
func (m Money) String() string {
	return fmt.Sprintf("%d.%02d", m.Units, m.Cents)
}

// Float64 returns the amount for JSON display purposes (display only, D-05).
func (m Money) Float64() float64 {
	return float64(m.Units) + float64(m.Cents)/100
}

// Value implements driver.Valuer so Money can be written to numeric columns.
func (m Money) Value() (driver.Value, error) {
	return m.String(), nil
}

// User is a registered account.
type User struct {
	ID        string
	Email     string
	CreatedAt time.Time
}

// Device is a paired Android device with its API key hash.
type Device struct {
	ID          string
	UserID      string
	DeviceModel string
	OSVersion   string
	APIKeyHash  string
	CreatedAt   time.Time
}

// Transaction is the core resource.
type Transaction struct {
	ID                    string
	UserID                string
	DeviceID              string
	Amount                Money
	Currency              string
	Direction             Direction
	Counterparty          string
	Category              string
	CategoryConfidence    float64
	CategoryHasConfidence bool
	CategorySource        string // "system" | "user" | ""
	Source                Source
	OcurredAt             time.Time
	CreatedAt             time.Time
}

// RawMessage is a retained ingested SMS with its parse outcome (FR-2.4).
type RawMessage struct {
	ID              string
	UserID          string
	DeviceID        string
	ClientMessageID string
	SenderID        string
	Body            string
	ReceivedAt      time.Time
	Status          string // parsed | queued_for_review | failed
	TransactionID   string
	ParseError      string
	CreatedAt       time.Time
}

// ParsedTransaction is the normalized parser output.
type ParsedTransaction struct {
	Amount       Money
	Currency     string
	Direction    Direction
	Counterparty string
	OccurredAt   time.Time
	Reference    string
}

// StatementUpload is an uploaded PDF/CSV statement (FR-7).
type StatementUpload struct {
	ID        string
	UserID    string
	BankHint  string
	MimeType  string
	SizeBytes int64
	Status    string // processing | completed | failed
	Error     string
	TxCreated int
	TxFlagged int
	CreatedAt time.Time
	// FileBytes loaded on demand for processing, never in list views.
	FileBytes []byte `db:"-"`
}

// Clarification is a queued clarifying question (FR-5.1).
type Clarification struct {
	ID             string
	UserID         string
	TransactionID  string
	Question       string
	Options        []string
	Status         string // pending | resolved
	AnswerCategory string
	AnsweredAt     time.Time
	CreatedAt      time.Time
}

// WeeklyReport is a generated weekly summary (FR-6).
type WeeklyReport struct {
	ID               string
	UserID           string
	PeriodStart      time.Time
	PeriodEnd        time.Time
	TotalsByCategory map[string]Money
	WeekOverWeek     map[string]string
	Narrative        string
	GenerationStatus string // pending | llm_skipped | completed | failed
	GeneratedAt      time.Time
	CreatedAt        time.Time
}

// CategoryRule is a learned categorization rule (FR-4.2).
type CategoryRule struct {
	ID         string
	UserID     string
	MatchType  string // counterparty | keyword
	MatchValue string
	Category   string
	CreatedAt  time.Time
}
