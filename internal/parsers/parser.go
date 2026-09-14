// Package parsers implements the bank/telecom SMS parsing engine.
// Each format is an isolated plugin conforming to Parser (FR-3); the Registry
// selects a parser by sender ID and message shape (FR-3.1) and routes unmatched
// messages to a review queue via ErrUnmatchedFormat (FR-3.3).
package parsers

import (
	"errors"
	"time"

	"github.com/abel-gezahegn/ledger/internal/domain"
)

// ErrUnmatchedFormat signals that no parser matched — routes to review queue.
var ErrUnmatchedFormat = errors.New("no parser matched sender/format")

// ErrParse is a parse failure within a matched parser (malformed content).
var ErrParse = errors.New("parsing sms body")

// Parser is the plugin interface (FR-3). Add a new bank format by implementing
// this and registering it — no existing code changes (open/closed).
type Parser interface {
	// SenderIDs returns the alphanumeric sender IDs this parser claims,
	// e.g. "CBE", "TELEBIRR", "8329".
	SenderIDs() []string
	// Match reports whether this parser can attempt the message.
	Match(msg domain.RawMessage) bool
	// Parse extracts the normalized transaction. Returns a wrapped ErrParse
	// with context on malformed content.
	Parse(msg domain.RawMessage) (domain.ParsedTransaction, error)
}

// Registry holds the registered parsers, ordering shape-matchers first.
type Registry struct {
	parsers  []Parser
	bySender map[string]Parser
}

// NewRegistry builds an empty registry.
func NewRegistry(ps ...Parser) *Registry {
	r := &Registry{bySender: map[string]Parser{}}
	for _, p := range ps {
		r.Register(p)
	}
	return r
}

// Register adds a parser; sender IDs are claimed case-insensitively.
func (r *Registry) Register(p Parser) {
	r.parsers = append(r.parsers, p)
	for _, id := range p.SenderIDs() {
		r.bySender[normalizeSender(id)] = p
	}
}

func normalizeSender(s string) string {
	out := make([]rune, 0, len(s))
	for _, ch := range s {
		if ch == ' ' || ch == '-' {
			continue
		}
		if ch >= 'A' && ch <= 'Z' {
			ch += 32
		}
		out = append(out, ch)
	}
	return string(out)
}

// ParseSender picks a parser by sender ID, falling back to shape matching,
// and parses the body. Errors are wrapped with the parser context per the
// engineering bar ("parsing CBE sms: ...").
func (r *Registry) ParseSender(senderID, body string, receivedAt time.Time) (domain.ParsedTransaction, error) {
	msg := domain.RawMessage{SenderID: senderID, Body: body, ReceivedAt: receivedAt}

	if p, ok := r.bySender[normalizeSender(senderID)]; ok {
		pt, err := p.Parse(msg)
		if err != nil {
			return domain.ParsedTransaction{}, wrapParse(senderID, err)
		}
		return pt, nil
	}
	// Fall back to shape matching across parsers (registration order).
	for _, p := range r.parsers {
		if p.Match(msg) {
			pt, err := p.Parse(msg)
			if err != nil {
				return domain.ParsedTransaction{}, wrapParse(senderID, err)
			}
			return pt, nil
		}
	}
	return domain.ParsedTransaction{}, ErrUnmatchedFormat
}

func wrapParse(sender string, err error) error {
	return err // parsers already wrap with their bank context
}
