// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr            string
	DatabaseURL         string
	CheckInterval       time.Duration
	CheckTimeout        time.Duration
	Workers             int
	RetentionDays       int
	AllowPrivateTargets bool
	APITokens           []string // static tokens with full (read + write) access
	APIReadTokens       []string // static tokens with read-only access
	AllowNoAuth         bool     // local development only

	// Optional per-user auth: validate JWTs issued by an OIDC provider (Entra ID, Auth0, Keycloak, ...).
	OIDCIssuer       string
	OIDCAudience     string
	OIDCJWKSURL      string
	OIDCReadRole     string // optional: role/scope needed for read endpoints (empty = any valid token)
	OIDCWriteRole    string // optional: role/scope needed for write endpoints (empty = any valid token)
}

func Load() Config {
	return Config{
		HTTPAddr:            env("HTTP_ADDR", ":8080"),
		DatabaseURL:         env("DATABASE_URL", "postgres://sentinel:sentinel@localhost:5432/sentinel?sslmode=disable"),
		CheckInterval:       envDuration("CHECK_INTERVAL", 30*time.Second),
		CheckTimeout:        envDuration("CHECK_TIMEOUT", 10*time.Second),
		Workers:             envInt("CHECK_WORKERS", 10),
		RetentionDays:       envInt("RETENTION_DAYS", 7),
		AllowPrivateTargets: envBool("ALLOW_PRIVATE_TARGETS", false),
		APITokens:           envList("API_TOKENS"),
		APIReadTokens:       envList("API_READ_TOKENS"),
		AllowNoAuth:         envBool("ALLOW_NO_AUTH", false),

		OIDCIssuer:       env("OIDC_ISSUER", ""),
		OIDCAudience:     env("OIDC_AUDIENCE", ""),
		OIDCJWKSURL:      env("OIDC_JWKS_URL", ""),
		OIDCReadRole:     env("OIDC_READ_ROLE", ""),
		OIDCWriteRole:    env("OIDC_WRITE_ROLE", ""),
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return def
}

func envBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

func envDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return def
}

// envList reads a comma-separated list, trimming spaces and dropping empty items.
func envList(key string) []string {
	var out []string
	for _, p := range strings.Split(os.Getenv(key), ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
