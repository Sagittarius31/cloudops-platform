package api

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"cloudops-platform/internal/metrics"
)

// fakeVerifier maps a raw token to the roles it "carries"; unknown tokens are invalid.
type fakeVerifier map[string][]string

func (f fakeVerifier) Verify(_ context.Context, raw string) ([]string, error) {
	roles, ok := f[raw]
	if !ok {
		return nil, errors.New("invalid")
	}
	return roles, nil
}

func newTestServer(a AuthOptions) *Server {
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	return New(nil, metrics.New(), log, true, a)
}

func TestRequireAuth(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	full := "full-token-0123456789"
	ro := "read-token-0123456789"
	v := fakeVerifier{
		"jwt-reader": {"sentinel.read"},
		"jwt-writer": {"sentinel.write"},
		"jwt-none":   {},
	}
	static := AuthOptions{Tokens: []string{full}, ReadTokens: []string{ro}}
	withRoles := AuthOptions{Verifier: v, ReadRole: "sentinel.read", WriteRole: "sentinel.write"}
	anyJWT := AuthOptions{Verifier: v}

	tests := []struct {
		name   string
		opts   AuthOptions
		write  bool
		header string
		want   int
	}{
		{"no header", static, false, "", http.StatusUnauthorized},
		{"wrong scheme", static, false, "Basic " + full, http.StatusUnauthorized},
		{"empty bearer", static, false, "Bearer ", http.StatusUnauthorized},
		{"wrong token", static, false, "Bearer nope", http.StatusUnauthorized},
		{"full token reads", static, false, "Bearer " + full, http.StatusOK},
		{"full token writes", static, true, "Bearer " + full, http.StatusOK},
		{"read token reads", static, false, "Bearer " + ro, http.StatusOK},
		{"read token cannot write", static, true, "Bearer " + ro, http.StatusForbidden},

		{"jwt reader reads", withRoles, false, "Bearer jwt-reader", http.StatusOK},
		{"jwt reader cannot write", withRoles, true, "Bearer jwt-reader", http.StatusForbidden},
		{"jwt writer writes", withRoles, true, "Bearer jwt-writer", http.StatusOK},
		{"jwt writer can also read", withRoles, false, "Bearer jwt-writer", http.StatusOK},
		{"jwt without roles forbidden to read", withRoles, false, "Bearer jwt-none", http.StatusForbidden},
		{"invalid jwt is 401 not 403", withRoles, false, "Bearer garbage", http.StatusUnauthorized},

		{"any valid jwt reads when no roles set", anyJWT, false, "Bearer jwt-none", http.StatusOK},
		{"any valid jwt writes when no roles set", anyJWT, true, "Bearer jwt-none", http.StatusOK},

		{"auth disabled passes through", AuthOptions{}, true, "", http.StatusOK},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := newTestServer(tc.opts).requireAuth(tc.write, ok)
			req := httptest.NewRequest(http.MethodGet, "/api/v1/targets", nil)
			if tc.header != "" {
				req.Header.Set("Authorization", tc.header)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tc.want {
				t.Fatalf("got %d want %d", rec.Code, tc.want)
			}
		})
	}
}
