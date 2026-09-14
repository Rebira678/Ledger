// Package auth provides JWT access tokens, bcrypt password hashing, opaque
// refresh tokens, and per-device API key issuance/verification.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Claims carried by access tokens.
type Claims struct {
	UserID string `json:"uid"`
	jwt.RegisteredClaims
}

// Tokenizer issues and verifies HMAC-SHA256 JWTs.
type Tokenizer struct {
	key        []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewTokenizer builds a Tokenizer.
func NewTokenizer(signingKey []byte, accessTTL, refreshTTL time.Duration) *Tokenizer {
	return &Tokenizer{key: signingKey, accessTTL: accessTTL, refreshTTL: refreshTTL}
}

// NewAccessToken mints a signed JWT for the user.
func (t *Tokenizer) NewAccessToken(userID string) (token string, expiresIn int64, err error) {
	now := time.Now()
	claims := Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.accessTTL)),
			Issuer:    "ledger",
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString(t.key)
	if err != nil {
		return "", 0, fmt.Errorf("signing access token: %w", err)
	}
	return signed, int64(t.accessTTL.Seconds()), nil
}

// VerifyAccessToken validates a JWT and returns the user id.
func (t *Tokenizer) VerifyAccessToken(token string) (string, error) {
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(*jwt.Token) (any, error) {
		return t.key, nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return "", fmt.Errorf("verifying access token: %w", err)
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || !parsed.Valid || claims.UserID == "" {
		return "", errors.New("invalid access token claims")
	}
	return claims.UserID, nil
}

// HashPassword hashes with bcrypt cost 12 (D-04).
func HashPassword(password string) (string, error) {
	if len(password) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	if err != nil {
		return "", fmt.Errorf("hashing password: %w", err)
	}
	return string(h), nil
}

// CheckPassword verifies a password against its bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// NewOpaqueToken returns a 256-bit random hex token and its SHA-256 hash.
func NewOpaqueToken() (token string, hash string, err error) {
	var b [32]byte
	if _, err = rand.Read(b[:]); err != nil {
		return "", "", fmt.Errorf("generating token: %w", err)
	}
	token = hex.EncodeToString(b[:])
	sum := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(sum[:]), nil
}

// HashToken computes the SHA-256 hex hash used for refresh tokens and device keys.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// NewDeviceAPIKey returns a device key (plaintext shown once) plus its hash.
func NewDeviceAPIKey() (key string, hash string, err error) {
	var b [32]byte
	if _, err = rand.Read(b[:]); err != nil {
		return "", "", fmt.Errorf("generating device key: %w", err)
	}
	key = "dk_live_" + hex.EncodeToString(b[:])
	return key, HashToken(key), nil
}
