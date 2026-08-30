package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all environment-based configuration for the API server.
type Config struct {
	DatabaseURL    string
	APIPort        string
	JWTSecret      string
	JWTExpiryHours int
	LogLevel       string
	CORSOrigin     string
	AdminEmails    []string
	Environment    string
	CookieSecure   bool
	CookieSameSite string
	MetricsEnabled bool
	MetricsToken   string
}

// IngestConfig holds configuration for the ingestion CLI.
type IngestConfig struct {
	DatabaseURL      string
	USAJobsAPIKey    string
	USAJobsUserAgent string
	LogLevel         string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	loadDotEnv()

	cfg := &Config{
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		APIPort:        envOrDefault("API_PORT", "8080"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		JWTExpiryHours: envIntOrDefault("JWT_EXPIRY_HOURS", 24),
		LogLevel:       envOrDefault("LOG_LEVEL", "info"),
		CORSOrigin:     envOrDefault("CORS_ORIGIN", "http://localhost:5173"),
		AdminEmails:    parseEmailList(os.Getenv("CAREEROS_ADMIN_EMAILS")),
		Environment:    envOrDefault("CAREEROS_ENV", "development"),
		CookieSecure:   envBoolOrDefault("COOKIE_SECURE", os.Getenv("CAREEROS_ENV") == "production"),
		CookieSameSite: envOrDefault("COOKIE_SAMESITE", defaultCookieSameSite(os.Getenv("CAREEROS_ENV"))),
		MetricsEnabled: envBoolOrDefault("METRICS_ENABLED", os.Getenv("CAREEROS_ENV") != "production"),
		MetricsToken:   os.Getenv("METRICS_TOKEN"),
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadIngest reads configuration for the ingestion command.
func LoadIngest() (*IngestConfig, error) {
	loadDotEnv()

	cfg := &IngestConfig{
		DatabaseURL:      os.Getenv("DATABASE_URL"),
		USAJobsAPIKey:    os.Getenv("USAJOBS_API_KEY"),
		USAJobsUserAgent: os.Getenv("USAJOBS_USER_AGENT"),
		LogLevel:         envOrDefault("LOG_LEVEL", "info"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.USAJobsAPIKey == "" {
		return nil, fmt.Errorf("USAJOBS_API_KEY is required")
	}
	if cfg.USAJobsUserAgent == "" {
		return nil, fmt.Errorf("USAJOBS_USER_AGENT is required (use your contact email)")
	}

	return cfg, nil
}

// Validate checks that required configuration is present and valid.
func (c *Config) Validate() error {
	if c.DatabaseURL == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if c.JWTSecret == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if len(c.JWTSecret) < 16 {
		return fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}
	if c.JWTExpiryHours <= 0 {
		return fmt.Errorf("JWT_EXPIRY_HOURS must be positive")
	}

	switch strings.ToLower(c.LogLevel) {
	case "debug", "info", "warn", "error":
	default:
		return fmt.Errorf("LOG_LEVEL must be one of: debug, info, warn, error")
	}

	if c.Environment == "production" {
		if c.CORSOrigin == "*" {
			return fmt.Errorf("CORS_ORIGIN cannot be * in production")
		}
		if c.JWTSecret == "change-me-to-a-random-secret-in-production" {
			return fmt.Errorf("JWT_SECRET must be changed from the example value in production")
		}
	}

	return nil
}

func (c *Config) IsProduction() bool {
	return strings.EqualFold(c.Environment, "production")
}

func defaultCookieSameSite(env string) string {
	if strings.EqualFold(env, "production") {
		return "None"
	}
	return "Lax"
}

func envBoolOrDefault(key string, defaultVal bool) bool {
	v := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
	if v == "" {
		return defaultVal
	}
	return v == "1" || v == "true" || v == "yes"
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func envIntOrDefault(key string, defaultVal int) int {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return defaultVal
	}
	return n
}

func parseEmailList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func (c *Config) AdminEmailSet() map[string]struct{} {
	set := make(map[string]struct{}, len(c.AdminEmails))
	for _, email := range c.AdminEmails {
		set[email] = struct{}{}
	}
	return set
}
