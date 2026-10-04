// Package config loads all Ledger configuration from environment variables.
// No hardcoded secrets: everything comes from the environment (see .env.example).
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the fully-parsed application configuration.
type Config struct {
	DatabaseURL string
	HTTPAddr    string
	Env         string // "development" | "production"

	JWTSigningKey   []byte
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration

	LLMAPIKey  string
	LLMBaseURL string
	LLMModel   string
	LLMTimeout time.Duration

	MaxStatementBytes int64

	RateLimitIngestPerMin int
	RateLimitUserPerMin   int

	ReportCron     string
	ReportTimezone string

	CORSOrigins []string

	TemplatesDir string

	SMTPHost string
	SMTPPort string
	SMTPUser string
	SMTPPass string
	SMTPFrom string
}

func getEnv(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

func getDuration(key string, def time.Duration) (time.Duration, error) {
	v := getEnv(key, "")
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("config: parsing %s=%q: %w", key, v, err)
	}
	return d, nil
}

func getInt(key string, def int) (int, error) {
	v := getEnv(key, "")
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("config: parsing %s=%q: %w", key, v, err)
	}
	return n, nil
}

// Load reads configuration from the environment and validates it.
func Load() (*Config, error) {
	cfg := &Config{
		DatabaseURL: getEnv("LEDGER_DATABASE_URL", "postgres://ledger:ledger@localhost:5432/ledger?sslmode=disable"),
		HTTPAddr:    getEnv("LEDGER_HTTP_ADDR", ":8080"),
		Env:         getEnv("LEDGER_ENV", "development"),

		LLMAPIKey:  os.Getenv("LEDGER_LLM_API_KEY"),
		LLMBaseURL: getEnv("LEDGER_LLM_BASE_URL", "https://api.openai.com/v1"),
		LLMModel:   getEnv("LEDGER_LLM_MODEL", "gpt-4o-mini"),

		ReportCron:     getEnv("LEDGER_REPORT_CRON", "0 6 * * 1"),
		ReportTimezone: getEnv("LEDGER_REPORT_TIMEZONE", "Africa/Addis_Ababa"),
		TemplatesDir:   getEnv("LEDGER_TEMPLATES_DIR", "web/templates"),

		SMTPHost: getEnv("LEDGER_SMTP_HOST", ""),
		SMTPPort: getEnv("LEDGER_SMTP_PORT", "587"),
		SMTPUser: getEnv("LEDGER_SMTP_USER", ""),
		SMTPPass: getEnv("LEDGER_SMTP_PASS", ""),
		SMTPFrom: getEnv("LEDGER_SMTP_FROM", "noreply@ledger.local"),
	}

	var err error
	if cfg.JWTSigningKey = []byte(os.Getenv("LEDGER_JWT_SIGNING_KEY")); len(cfg.JWTSigningKey) == 0 {
		return nil, fmt.Errorf("config: LEDGER_JWT_SIGNING_KEY is required (see .env.example)")
	}
	if cfg.AccessTokenTTL, err = getDuration("LEDGER_ACCESS_TOKEN_TTL", 15*time.Minute); err != nil {
		return nil, err
	}
	if cfg.RefreshTokenTTL, err = getDuration("LEDGER_REFRESH_TOKEN_TTL", 720*time.Hour); err != nil {
		return nil, err
	}
	if cfg.LLMTimeout, err = getDuration("LEDGER_LLM_TIMEOUT", 30*time.Second); err != nil {
		return nil, err
	}
	maxBytes, err := getInt("LEDGER_MAX_STATEMENT_BYTES", 10*1024*1024)
	if err != nil {
		return nil, err
	}
	cfg.MaxStatementBytes = int64(maxBytes)
	if cfg.RateLimitIngestPerMin, err = getInt("LEDGER_RATE_LIMIT_INGEST_PER_MIN", 120); err != nil {
		return nil, err
	}
	if cfg.RateLimitUserPerMin, err = getInt("LEDGER_RATE_LIMIT_USER_PER_MIN", 60); err != nil {
		return nil, err
	}
	for _, o := range strings.Split(getEnv("LEDGER_CORS_ORIGINS", ""), ",") {
		if o = strings.TrimSpace(o); o != "" {
			cfg.CORSOrigins = append(cfg.CORSOrigins, o)
		}
	}
	if cfg.Env != "development" && cfg.Env != "production" {
		return nil, fmt.Errorf("config: LEDGER_ENV must be \"development\" or \"production\", got %q", cfg.Env)
	}
	if len(cfg.JWTSigningKey) < 32 && cfg.Env == "production" {
		return nil, fmt.Errorf("config: LEDGER_JWT_SIGNING_KEY must be >= 32 bytes in production")
	}
	return cfg, nil
}
