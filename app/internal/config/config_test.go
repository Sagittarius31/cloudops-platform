package config

import (
	"reflect"
	"testing"
	"time"
)

func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"HTTP_ADDR", "DATABASE_URL", "CHECK_INTERVAL", "CHECK_TIMEOUT", "CHECK_WORKERS", "RETENTION_DAYS",
		"ALLOW_PRIVATE_TARGETS", "API_TOKENS", "API_READ_TOKENS", "ALLOW_NO_AUTH",
		"OIDC_ISSUER", "OIDC_AUDIENCE", "OIDC_JWKS_URL", "OIDC_READ_ROLE", "OIDC_WRITE_ROLE",
	} {
		t.Setenv(k, "") // empty is treated as unset
	}
}

func TestLoadDefaultsAreSecure(t *testing.T) {
	clearEnv(t)
	c := Load()
	if c.HTTPAddr != ":8080" || c.CheckInterval != 30*time.Second || c.Workers != 10 || c.RetentionDays != 7 {
		t.Fatalf("unexpected defaults: %+v", c)
	}
	if c.AllowPrivateTargets {
		t.Error("private targets must be disallowed by default (SSRF protection)")
	}
	if c.AllowNoAuth {
		t.Error("running without auth must be opt-in, never the default")
	}
	if len(c.APITokens) != 0 || len(c.APIReadTokens) != 0 {
		t.Error("no tokens should be configured by default")
	}
}

func TestEnvList(t *testing.T) {
	t.Setenv("API_TOKENS", " a , ,b,, ")
	got := envList("API_TOKENS")
	if want := []string{"a", "b"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestInvalidValuesFallBackToDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("CHECK_INTERVAL", "nonsense")
	t.Setenv("CHECK_WORKERS", "-3")
	t.Setenv("ALLOW_PRIVATE_TARGETS", "maybe")
	c := Load()
	if c.CheckInterval != 30*time.Second {
		t.Errorf("bad duration should fall back, got %v", c.CheckInterval)
	}
	if c.Workers != 10 {
		t.Errorf("non-positive workers should fall back, got %d", c.Workers)
	}
	if c.AllowPrivateTargets {
		t.Error("unparseable bool must fall back to the safe default (false)")
	}
}

func TestValidOverrides(t *testing.T) {
	clearEnv(t)
	t.Setenv("CHECK_INTERVAL", "5s")
	t.Setenv("CHECK_WORKERS", "3")
	t.Setenv("ALLOW_PRIVATE_TARGETS", "true")
	t.Setenv("OIDC_WRITE_ROLE", "sentinel.write")
	c := Load()
	if c.CheckInterval != 5*time.Second || c.Workers != 3 || !c.AllowPrivateTargets || c.OIDCWriteRole != "sentinel.write" {
		t.Fatalf("overrides not applied: %+v", c)
	}
}
