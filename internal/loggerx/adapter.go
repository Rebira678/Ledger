package loggerx

import (
	"context"
	"log/slog"
)

// Adapter exposes a context-aware slog surface for middleware packages.
type Adapter struct{ base *slog.Logger }

// NewAdapter wraps a base logger.
func NewAdapter(base *slog.Logger) *Adapter { return &Adapter{base: base} }

func (a *Adapter) enrich(ctx context.Context, args []any) []any {
	if id, ok := CorrelationID(ctx); ok {
		return append([]any{"correlation_id", id}, args...)
	}
	return args
}

// Info logs at info level with correlation ID.
func (a *Adapter) Info(ctx context.Context, msg string, args ...any) {
	a.base.InfoContext(ctx, msg, a.enrich(ctx, args)...)
}

// Error logs at error level with correlation ID.
func (a *Adapter) Error(ctx context.Context, msg string, args ...any) {
	a.base.ErrorContext(ctx, msg, a.enrich(ctx, args)...)
}
