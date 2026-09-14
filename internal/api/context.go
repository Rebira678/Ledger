package api

import (
	"context"
)

// Identity is the authenticated principal attached to a request context.
type Identity struct {
	UserID   string
	DeviceID string // non-empty for device-key-authenticated ingestion requests
}

type ctxKey int

const identityKey ctxKey = 1

// WithIdentity stores the identity on the context.
func WithIdentity(ctx context.Context, id Identity) context.Context {
	return context.WithValue(ctx, identityKey, id)
}

// IdentityFrom retrieves the identity, if any.
func IdentityFrom(ctx context.Context) (Identity, bool) {
	id, ok := ctx.Value(identityKey).(Identity)
	return id, ok
}

// MustUserID returns the user id or "" — handlers check authn before.
func MustUserID(ctx context.Context) string {
	id, _ := IdentityFrom(ctx)
	return id.UserID
}
