// Package auth validates OIDC-issued JWT access tokens (RS256) against a JWKS endpoint.
package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	keyMaxAge       = 10 * time.Minute // refresh the key set at least this often
	refreshThrottle = 30 * time.Second // never hit the JWKS endpoint more often than this
)

type Config struct {
	Issuer   string
	Audience string
	JWKSURL  string
}

type Verifier struct {
	cfg    Config
	client *http.Client

	mu          sync.RWMutex
	keys        map[string]*rsa.PublicKey
	fetchedAt   time.Time
	lastAttempt time.Time
}

func NewVerifier(cfg Config) *Verifier {
	return &Verifier{
		cfg:    cfg,
		client: &http.Client{Timeout: 5 * time.Second},
		keys:   map[string]*rsa.PublicKey{},
	}
}

// Verify checks signature (RS256 only), issuer, audience and expiry. It returns the roles/scopes
// carried by the token; the caller decides what those roles allow.
func (v *Verifier) Verify(ctx context.Context, raw string) ([]string, error) {
	claims := jwt.MapClaims{}
	_, err := jwt.ParseWithClaims(raw, claims,
		func(t *jwt.Token) (any, error) {
			kid, _ := t.Header["kid"].(string)
			if kid == "" {
				return nil, errors.New("token has no kid header")
			}
			return v.key(ctx, kid)
		},
		jwt.WithValidMethods([]string{"RS256"}), // blocks alg=none and HS256 key-confusion attacks
		jwt.WithIssuer(v.cfg.Issuer),
		jwt.WithAudience(v.cfg.Audience),
		jwt.WithExpirationRequired(),
		jwt.WithLeeway(30*time.Second),
	)
	if err != nil {
		return nil, err
	}
	return rolesFrom(claims), nil
}

func (v *Verifier) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	v.mu.RLock()
	k, ok := v.keys[kid]
	fresh := time.Since(v.fetchedAt) < keyMaxAge
	v.mu.RUnlock()
	if ok && fresh {
		return k, nil
	}

	if err := v.refresh(ctx); err != nil && !ok {
		return nil, err // nothing cached to fall back on
	}

	v.mu.RLock()
	defer v.mu.RUnlock()
	k, ok = v.keys[kid]
	if !ok {
		return nil, errors.New("unknown signing key")
	}
	return k, nil
}

// refresh re-fetches the JWKS, throttled. On failure the old keys are kept (served stale).
func (v *Verifier) refresh(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if len(v.keys) > 0 && time.Since(v.lastAttempt) < refreshThrottle {
		return nil
	}
	v.lastAttempt = time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.cfg.JWKSURL, nil)
	if err != nil {
		return err
	}
	resp, err := v.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch jwks: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch jwks: status %d", resp.StatusCode)
	}

	var set struct {
		Keys []struct {
			Kty string `json:"kty"`
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&set); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}

	keys := map[string]*rsa.PublicKey{}
	for _, jk := range set.Keys {
		if jk.Kty != "RSA" || jk.Kid == "" {
			continue
		}
		nb, err1 := base64.RawURLEncoding.DecodeString(jk.N)
		eb, err2 := base64.RawURLEncoding.DecodeString(jk.E)
		if err1 != nil || err2 != nil || len(nb) == 0 || len(eb) == 0 {
			continue
		}
		e := new(big.Int).SetBytes(eb)
		if !e.IsInt64() {
			continue
		}
		keys[jk.Kid] = &rsa.PublicKey{N: new(big.Int).SetBytes(nb), E: int(e.Int64())}
	}
	if len(keys) == 0 {
		return errors.New("jwks contained no usable RSA keys")
	}
	v.keys = keys
	v.fetchedAt = time.Now()
	return nil
}

// rolesFrom collects a "roles" array (Entra ID app roles, Keycloak) and space-separated
// "scope" / "scp" strings (OAuth scopes) into one list.
func rolesFrom(claims jwt.MapClaims) []string {
	var out []string
	if arr, ok := claims["roles"].([]any); ok {
		for _, r := range arr {
			if s, ok := r.(string); ok {
				out = append(out, s)
			}
		}
	}
	for _, name := range []string{"scope", "scp"} {
		if s, ok := claims[name].(string); ok {
			out = append(out, strings.Fields(s)...)
		}
	}
	return out
}
