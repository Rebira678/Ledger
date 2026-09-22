// Package httpx holds HTTP plumbing: middleware, the API error envelope,
// rate limiting, and small request helpers shared by all handlers.
package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/rebira678/ledger/internal/domain"
	"github.com/rebira678/ledger/internal/loggerx"
)

// Envelope writes the API-contract error envelope.
func Envelope(w http.ResponseWriter, status int, code, message, field string) {
	e := struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Field   string `json:"field,omitempty"`
		} `json:"error"`
	}{}
	e.Error.Code = code
	e.Error.Message = message
	e.Error.Field = field
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(e)
}

// APIError maps handler errors onto the contract envelope.
type APIError struct {
	Status  int
	Code    string
	Message string
	Field   string
}

func (e *APIError) Error() string { return e.Code + ": " + e.Message }

// NewAPIErrorf builds an APIError with the contract code.
func NewAPIErrorf(status int, code, field, format string, args ...any) *APIError {
	return &APIError{Status: status, Code: code, Message: sprintf(format, args...), Field: field}
}

func sprintf(format string, args ...any) string {
	if len(args) == 0 {
		return format
	}
	return fmt.Sprintf(format, args...)
}

// MapDomainError converts repo/domain errors to API errors per contract §9.
func MapDomainError(err error) *APIError {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		return &APIError{Status: http.StatusNotFound, Code: "NOT_FOUND", Message: "resource does not exist or is not owned by the caller"}
	case errors.Is(err, domain.ErrAlreadyExists):
		return &APIError{Status: http.StatusConflict, Code: "DUPLICATE", Message: "idempotent resource already exists"}
	case errors.Is(err, domain.ErrConflict):
		return &APIError{Status: http.StatusConflict, Code: "DUPLICATE", Message: "conflicting resource state"}
	default:
		return &APIError{Status: http.StatusInternalServerError, Code: "INTERNAL", Message: "internal server error"}
	}
}

// WriteError maps any error to the contract envelope (5xx details logged upstream).
func WriteError(w http.ResponseWriter, err error) {
	var apiErr *APIError
	if errors.As(err, &apiErr) {
		Envelope(w, apiErr.Status, apiErr.Code, apiErr.Message, apiErr.Field)
		return
	}
	Envelope(w, http.StatusInternalServerError, "INTERNAL", "internal server error", "")
}

// DecodeJSON strictly decodes a JSON body into v, rejecting unknown fields.
func DecodeJSON(r *http.Request, v any) error {
	dec := json.NewDecoder(http.MaxBytesReader(nil, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return NewAPIErrorf(http.StatusUnprocessableEntity, "VALIDATION_ERROR", "", "malformed JSON body: %v", err)
	}
	return nil
}

//---- Middleware -------------------------------------------------------------

type ctxKeyMiddleware int

// StatusWriter captures response codes for logging/rate-limit headers.
type StatusWriter struct {
	http.ResponseWriter
	Status int
}

func (sw *StatusWriter) WriteHeader(code int) {
	if sw.Status == 0 {
		sw.Status = code
	}
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *StatusWriter) Write(b []byte) (int, error) {
	if sw.Status == 0 {
		sw.Status = http.StatusOK
	}
	return sw.ResponseWriter.Write(b)
}

// Correlate assigns a correlation ID and exposes it via X-Request-ID.
func Correlate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = loggerx.NewCorrelationID()
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r.WithContext(loggerx.WithCorrelationID(r.Context(), id)))
	})
}

// Logger is the minimal logging surface httpx needs (satisfied by loggerx.Adapter).
type Logger interface {
	Error(ctx context.Context, msg string, args ...any)
	Info(ctx context.Context, msg string, args ...any)
}

// Recover converts panics into 500s — never a bare panic on a request path.
func Recover(log Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					log.Error(r.Context(), "panic recovered", "panic", rec, "stack", string(debug.Stack()))
					Envelope(w, http.StatusInternalServerError, "INTERNAL", "internal server error", "")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestLogger emits one structured line per request.
func RequestLogger(log Logger) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			sw := &StatusWriter{ResponseWriter: w}
			next.ServeHTTP(sw, r)
			log.Info(r.Context(), "http request", "method", r.Method, "path", r.URL.Path, "status", sw.Status, "duration", time.Since(start))
		})
	}
}

// Middleware wraps an http.Handler.
type Middleware func(http.Handler) http.Handler

// Chain applies middleware in order (first listed runs first).
func Chain(h http.Handler, mw ...Middleware) http.Handler {
	for i := len(mw) - 1; i >= 0; i-- {
		h = mw[i](h)
	}
	return h
}

//---- Rate limiting ----------------------------------------------------------

type bucket struct {
	tokens   float64
	last     time.Time
	capacity float64
	refill   float64 // tokens per second
}

// RateLimiter is an in-process token bucket keyed by string (D-11).
type RateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*bucket
}

// NewRateLimiter builds a limiter.
func NewRateLimiter() *RateLimiter {
	return &RateLimiter{buckets: map[string]*bucket{}}
}

// Allow consumes one token for key at rpm requests/minute, returning the
// retry-after seconds when denied.
func (rl *RateLimiter) Allow(key string, rpm int) (ok bool, retryAfter time.Duration) {
	capacity := float64(rpm)
	refill := capacity / 60.0
	now := time.Now()
	rl.mu.Lock()
	defer rl.mu.Unlock()
	b, exists := rl.buckets[key]
	if !exists {
		rl.buckets[key] = &bucket{tokens: capacity - 1, last: now, capacity: capacity, refill: refill}
		return true, 0
	}
	elapsed := now.Sub(b.last).Seconds()
	b.tokens += elapsed * b.refill
	if b.tokens > b.capacity {
		b.tokens = b.capacity
	}
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	needed := 1 - b.tokens
	return false, time.Duration(needed / b.refill * float64(time.Second)).Round(time.Second)
}

// RateLimit returns middleware enforcing per-key limits with 429+Retry-After.
func RateLimit(keyFn func(*http.Request) string, rpm int) Middleware {
	limiter := NewRateLimiter()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := keyFn(r)
			if key != "" {
				ok, retry := limiter.Allow(key, rpm)
				if !ok {
					w.Header().Set("Retry-After", fmtInt(int(retry.Seconds()+1)))
					Envelope(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many requests; see Retry-After header", "")
					return
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func fmtInt(n int) string {
	return fmt.Sprintf("%d", n)
}

// clientIP extracts the request IP for anonymous limiting fallback.
func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// KeyByIP rate-limits unauthenticated endpoints by source IP.
func KeyByIP(r *http.Request) string { return clientIP(r) }

// KeyByUser limits by bearer subject.
func KeyByUser(r *http.Request) string {
	if auth := r.Header.Get("Authorization"); strings.HasPrefix(auth, "Bearer ") {
		return "u:" + strings.TrimPrefix(auth, "Bearer ")
	}
	return ""
}

// KeyByDevice limits by device API key header.
func KeyByDevice(r *http.Request) string {
	if k := r.Header.Get("X-Device-Key"); k != "" {
		return "d:" + k
	}
	return KeyByIP(r)
}
