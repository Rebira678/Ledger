// Package loggerx provides structured logging (log/slog) with request-scoped
// correlation IDs propagated through context.Context.
package loggerx

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
)

type ctxKey int

const correlationIDKey ctxKey = 1

// New builds the process-wide logger. In production it emits JSON at info
// level; in development, friendlier text at debug level.
func New(env string) *slog.Logger {
	if env == "production" {
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

// NewCorrelationID returns a fresh 16-hex-char request correlation ID.
func NewCorrelationID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		// crypto/rand failure is unrecoverable; fall back to a zero id rather
		// than panicking on a request path.
		return "0000000000000000"
	}
	return hex.EncodeToString(b[:])
}

// WithCorrelationID attaches a correlation ID to the context.
func WithCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// CorrelationID extracts the correlation ID from the context, if present.
func CorrelationID(ctx context.Context) (string, bool) {
	id, ok := ctx.Value(correlationIDKey).(string)
	return id, ok
}

// FromContext returns a logger enriched with the correlation ID from ctx.
func FromContext(ctx context.Context, base *slog.Logger) *slog.Logger {
	if id, ok := CorrelationID(ctx); ok {
		return base.With("correlation_id", id)
	}
	return base
}
