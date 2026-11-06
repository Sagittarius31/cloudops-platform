package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	testIssuer   = "https://issuer.example"
	testAudience = "sentinel-api"
)

func newKeyAndJWKS(t *testing.T) (*rsa.PrivateKey, *httptest.Server) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	body := map[string]any{"keys": []map[string]string{{
		"kty": "RSA",
		"kid": "k1",
		"n":   base64.RawURLEncoding.EncodeToString(priv.N.Bytes()),
		"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(priv.E)).Bytes()),
	}}}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(body)
	}))
	t.Cleanup(srv.Close)
	return priv, srv
}

func sign(t *testing.T, priv *rsa.PrivateKey, claims jwt.MapClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = "k1"
	s, err := tok.SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func goodClaims() jwt.MapClaims {
	return jwt.MapClaims{
		"iss":   testIssuer,
		"aud":   testAudience,
		"exp":   time.Now().Add(time.Hour).Unix(),
		"roles": []string{"sentinel.user"},
	}
}

func TestVerify(t *testing.T) {
	priv, jwks := newKeyAndJWKS(t)
	other, _ := rsa.GenerateKey(rand.Reader, 2048)

	mk := func(mod func(jwt.MapClaims)) jwt.MapClaims {
		c := goodClaims()
		if mod != nil {
			mod(c)
		}
		return c
	}

	tests := []struct {
		name     string
		token    func() string
		wantFail bool
	}{
		{"valid", func() string { return sign(t, priv, mk(nil)) }, false},
		{"wrong audience", func() string { return sign(t, priv, mk(func(c jwt.MapClaims) { c["aud"] = "other" })) }, true},
		{"wrong issuer", func() string { return sign(t, priv, mk(func(c jwt.MapClaims) { c["iss"] = "https://evil" })) }, true},
		{"expired", func() string {
			return sign(t, priv, mk(func(c jwt.MapClaims) { c["exp"] = time.Now().Add(-time.Hour).Unix() }))
		}, true},
		{"no exp", func() string { return sign(t, priv, mk(func(c jwt.MapClaims) { delete(c, "exp") })) }, true},
		{"signed by a different key", func() string { return sign(t, other, mk(nil)) }, true},
		{"HS256 key-confusion attempt", func() string {
			tok := jwt.NewWithClaims(jwt.SigningMethodHS256, mk(nil))
			tok.Header["kid"] = "k1"
			s, _ := tok.SignedString([]byte("secret"))
			return s
		}, true},
		{"garbage", func() string { return "not.a.jwt" }, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := NewVerifier(Config{Issuer: testIssuer, Audience: testAudience, JWKSURL: jwks.URL})
			_, err := v.Verify(context.Background(), tc.token())
			if (err != nil) != tc.wantFail {
				t.Fatalf("err=%v wantFail=%v", err, tc.wantFail)
			}
		})
	}
}

func TestVerifyReturnsRolesAndScopes(t *testing.T) {
	priv, jwks := newKeyAndJWKS(t)
	c := goodClaims()
	c["roles"] = []string{"sentinel.read", "sentinel.write"}
	c["scp"] = "openid profile"
	c["scope"] = "extra"

	v := NewVerifier(Config{Issuer: testIssuer, Audience: testAudience, JWKSURL: jwks.URL})
	got, err := v.Verify(context.Background(), sign(t, priv, c))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"extra", "openid", "profile", "sentinel.read", "sentinel.write"}
	sort.Strings(got)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("roles = %v, want %v", got, want)
	}
}
