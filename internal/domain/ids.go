// Package domain holds the core Ledger domain types shared across layers.
package domain

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// NewID generates a prefixed identifier, e.g. "txn_1a2b3c4d5e6f7a8b".
// The random suffix is 128 bits.
func NewID(prefix string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generating %s id: %w", prefix, err)
	}
	return prefix + "_" + hex.EncodeToString(b[:]), nil
}

// MustNewID is for tests and non-request-path usage only.
func MustNewID(prefix string) string {
	id, err := NewID(prefix)
	if err != nil {
		panic(err) // test-only helper
	}
	return id
}

// WeekStart returns the Monday 00:00 UTC of the week containing t.
func WeekStart(t time.Time) time.Time {
	t = t.UTC()
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	daysSinceMonday := weekday - 1
	start := time.Date(t.Year(), t.Month(), t.Day()-daysSinceMonday, 0, 0, 0, 0, time.UTC)
	return start
}
