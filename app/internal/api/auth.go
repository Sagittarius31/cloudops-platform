package api

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"strings"
)

type authResult int

const (
	authDenied    authResult = iota // 401: no/invalid credentials
	authOK                          // allowed
	authForbidden                   // 403: valid credentials, not allowed to do this
)

// requireAuth enforces authentication and read/write authorization.
//
//	static full token  -> read + write
//	static read token  -> read only (403 on write)
//	OIDC JWT           -> read needs ReadRole (or WriteRole), write needs WriteRole; empty role = any valid JWT
//
// With nothing configured it is a pass-through (guarded at startup in main).
func (s *Server) requireAuth(write bool, next http.Handler) http.Handler {
	if len(s.tokenHashes) == 0 && len(s.readHashes) == 0 && s.verifier == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, ok := bearerToken(r)
		if !ok {
			unauthorized(w)
			return
		}
		switch s.authorize(r.Context(), raw, write) {
		case authOK:
			next.ServeHTTP(w, r)
		case authForbidden:
			forbidden(w)
		default:
			unauthorized(w)
		}
	})
}

func (s *Server) authorize(ctx context.Context, raw string, write bool) authResult {
	given := sha256.Sum256([]byte(raw))
	if matchesAny(given, s.tokenHashes) {
		return authOK
	}
	if matchesAny(given, s.readHashes) {
		if write {
			return authForbidden
		}
		return authOK
	}
	if s.verifier == nil {
		return authDenied
	}
	roles, err := s.verifier.Verify(ctx, raw)
	if err != nil {
		s.log.Debug("jwt rejected", "err", err)
		return authDenied
	}
	if write {
		if s.writeRole == "" || hasRole(roles, s.writeRole) {
			return authOK
		}
		return authForbidden
	}
	if s.readRole == "" || hasRole(roles, s.readRole) || (s.writeRole != "" && hasRole(roles, s.writeRole)) {
		return authOK
	}
	return authForbidden
}

// matchesAny compares SHA-256 digests in constant time and checks every entry (no early exit).
func matchesAny(given [32]byte, set [][32]byte) bool {
	ok := 0
	for _, want := range set {
		ok |= subtle.ConstantTimeCompare(given[:], want[:])
	}
	return ok == 1
}

func hasRole(roles []string, want string) bool {
	for _, r := range roles {
		if r == want {
			return true
		}
	}
	return false
}

func bearerToken(r *http.Request) (string, bool) {
	const prefix = "Bearer "
	h := r.Header.Get("Authorization")
	if !strings.HasPrefix(h, prefix) {
		return "", false
	}
	tok := strings.TrimSpace(h[len(prefix):])
	return tok, tok != ""
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Bearer realm="sentinel"`)
	writeError(w, http.StatusUnauthorized, "missing or invalid bearer token")
}

func forbidden(w http.ResponseWriter) {
	writeError(w, http.StatusForbidden, "insufficient permissions for this operation")
}
