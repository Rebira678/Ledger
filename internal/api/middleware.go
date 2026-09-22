package api

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/rebira678/ledger/internal/auth"
	"github.com/rebira678/ledger/internal/domain"
	"github.com/rebira678/ledger/internal/httpx"
)

// bearerToken extracts "Authorization: Bearer <token>".
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(h, "Bearer ")
}

func errorWriter(w http.ResponseWriter, err error) {
	httpx.WriteError(w, err)
}

// RequireAuth validates the JWT and stores Identity in context (contract §1.1).
func RequireAuth(tokens *auth.Tokenizer) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				errorWriter(w, httpx.NewAPIErrorf(http.StatusUnauthorized, "UNAUTHENTICATED", "", "missing bearer token"))
				return
			}
			userID, err := tokens.VerifyAccessToken(token)
			if err != nil {
				errorWriter(w, httpx.NewAPIErrorf(http.StatusUnauthorized, "UNAUTHENTICATED", "", "missing, invalid, or expired token"))
				return
			}
			ctx := WithIdentity(r.Context(), Identity{UserID: userID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// deviceAuthenticator resolves device API keys.
type deviceAuthenticator interface {
	GetUserByAPIKeyHash(ctx context.Context, hash string) (*domain.Device, error)
}

// RequireDeviceAuth enforces JWT + X-Device-Key for ingestion endpoints. A
// stolen JWT alone must not be able to inject transactions (contract §1.1).
func RequireDeviceAuth(tokens *auth.Tokenizer, devices deviceAuthenticator) httpx.Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := bearerToken(r)
			if token == "" {
				errorWriter(w, httpx.NewAPIErrorf(http.StatusUnauthorized, "UNAUTHENTICATED", "", "missing bearer token"))
				return
			}
			userID, err := tokens.VerifyAccessToken(token)
			if err != nil {
				errorWriter(w, httpx.NewAPIErrorf(http.StatusUnauthorized, "UNAUTHENTICATED", "", "missing, invalid, or expired token"))
				return
			}
			key := r.Header.Get("X-Device-Key")
			if key == "" {
				errorWriter(w, httpx.NewAPIErrorf(http.StatusUnauthorized, "UNAUTHENTICATED", "", "missing X-Device-Key header"))
				return
			}
			device, err := devices.GetUserByAPIKeyHash(r.Context(), auth.HashToken(key))
			if err != nil {
				if errors.Is(err, domain.ErrNotFound) {
					errorWriter(w, httpx.NewAPIErrorf(http.StatusUnauthorized, "UNAUTHENTICATED", "", "invalid device key"))
					return
				}
				errorWriter(w, httpx.MapDomainError(err))
				return
			}
			// The device key must belong to the JWT's user: prevents injecting
			// under another user's device identity.
			if device.UserID != userID {
				errorWriter(w, httpx.NewAPIErrorf(http.StatusForbidden, "FORBIDDEN", "", "device key does not belong to authenticated user"))
				return
			}
			ctx := WithIdentity(r.Context(), Identity{UserID: userID, DeviceID: device.ID})
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
