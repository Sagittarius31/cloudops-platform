// Package api implements the REST API.
package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"time"

	"cloudops-platform/internal/metrics"
	"cloudops-platform/internal/store"
)

type Store interface {
	Ping(ctx context.Context) error
	CreateTarget(ctx context.Context, name, url string, expected int) (store.Target, error)
	ListTargets(ctx context.Context) ([]store.Target, error)
	GetTarget(ctx context.Context, id int64) (store.Target, error)
	DeleteTarget(ctx context.Context, id int64) error
	ListResults(ctx context.Context, targetID int64, limit int) ([]store.Result, error)
	Summary(ctx context.Context, targetID int64, window time.Duration) (store.Summary, error)
}

// TokenVerifier validates a bearer token that is not one of the static tokens (for example an
// OIDC JWT). It returns the roles/scopes the token carries, or an error if it is not valid.
type TokenVerifier interface {
	Verify(ctx context.Context, raw string) ([]string, error)
}

// AuthOptions configures authentication and authorization for /api/v1/*.
type AuthOptions struct {
	Tokens     []string      // static tokens with read + write access
	ReadTokens []string      // static tokens with read-only access
	Verifier   TokenVerifier // optional per-user JWT validation
	ReadRole   string        // role needed to read (empty = any valid JWT)
	WriteRole  string        // role needed to write (empty = any valid JWT)
}

type Server struct {
	store        Store
	metrics      *metrics.Metrics
	log          *slog.Logger
	allowPrivate bool
	tokenHashes  [][32]byte // SHA-256 of full-access static tokens
	readHashes   [][32]byte // SHA-256 of read-only static tokens
	verifier     TokenVerifier
	readRole     string
	writeRole    string
}

// New builds the API server. With no tokens and no verifier, /api/v1/* is NOT authenticated
// (main refuses to start in that state unless ALLOW_NO_AUTH=true).
func New(st Store, m *metrics.Metrics, log *slog.Logger, allowPrivate bool, a AuthOptions) *Server {
	s := &Server{
		store: st, metrics: m, log: log, allowPrivate: allowPrivate,
		verifier: a.Verifier, readRole: a.ReadRole, writeRole: a.WriteRole,
	}
	for _, t := range a.Tokens {
		s.tokenHashes = append(s.tokenHashes, sha256.Sum256([]byte(t)))
	}
	for _, t := range a.ReadTokens {
		s.readHashes = append(s.readHashes, sha256.Sum256([]byte(t)))
	}
	return s
}

// Routes uses Go 1.22 method+path patterns.
func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()
	// public: probes only
	open := func(pattern, name string, h http.HandlerFunc) {
		mux.Handle(pattern, s.instrument(name, h))
	}
	// protected: requires a valid bearer token; write=true needs write permission
	protected := func(pattern, name string, write bool, h http.HandlerFunc) {
		mux.Handle(pattern, s.instrument(name, s.requireAuth(write, h)))
	}
	open("GET /healthz", "healthz", s.healthz)
	open("GET /readyz", "readyz", s.readyz)
	protected("POST /api/v1/targets", "create_target", true, s.createTarget)
	protected("GET /api/v1/targets", "list_targets", false, s.listTargets)
	protected("GET /api/v1/targets/{id}", "get_target", false, s.getTarget)
	protected("DELETE /api/v1/targets/{id}", "delete_target", true, s.deleteTarget)
	protected("GET /api/v1/targets/{id}/results", "list_results", false, s.listResults)
	protected("GET /api/v1/targets/{id}/summary", "summary", false, s.summary)
	mux.Handle("GET /metrics", s.metrics.Handler())
	return s.recoverer(s.logging(mux))
}

func (s *Server) healthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) readyz(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()
	if err := s.store.Ping(ctx); err != nil {
		writeError(w, http.StatusServiceUnavailable, "database not ready")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

type createTargetRequest struct {
	Name           string `json:"name"`
	URL            string `json:"url"`
	ExpectedStatus int    `json:"expected_status"`
}

func (s *Server) createTarget(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	var req createTargetRequest
	if err := dec.Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}
	if req.ExpectedStatus == 0 {
		req.ExpectedStatus = http.StatusOK
	}
	if err := validateTarget(req.Name, req.URL, req.ExpectedStatus, s.allowPrivate); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	t, err := s.store.CreateTarget(r.Context(), req.Name, req.URL, req.ExpectedStatus)
	if errors.Is(err, store.ErrConflict) {
		writeError(w, http.StatusConflict, "a target with this name already exists")
		return
	}
	if err != nil {
		s.log.Error("create target", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusCreated, t)
}

func (s *Server) listTargets(w http.ResponseWriter, r *http.Request) {
	ts, err := s.store.ListTargets(r.Context())
	if err != nil {
		s.log.Error("list targets", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, ts)
}

func (s *Server) getTarget(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	t, err := s.store.GetTarget(r.Context(), id)
	if s.handleStoreErr(w, err) {
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) deleteTarget(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	t, err := s.store.GetTarget(r.Context(), id)
	if s.handleStoreErr(w, err) {
		return
	}
	if err := s.store.DeleteTarget(r.Context(), id); s.handleStoreErr(w, err) {
		return
	}
	// Drop the per-target series so deleted targets don't linger in /metrics.
	s.metrics.TargetUp.DeleteLabelValues(t.Name)
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listResults(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			writeError(w, http.StatusBadRequest, "limit must be between 1 and 500")
			return
		}
		limit = n
	}
	res, err := s.store.ListResults(r.Context(), id, limit)
	if err != nil {
		s.log.Error("list results", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r)
	if !ok {
		return
	}
	window := 24 * time.Hour
	if v := r.URL.Query().Get("window"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil || d <= 0 || d > 720*time.Hour {
			writeError(w, http.StatusBadRequest, "window must be a duration like 1h or 24h (max 720h)")
			return
		}
		window = d
	}
	sum, err := s.store.Summary(r.Context(), id, window)
	if err != nil {
		s.log.Error("summary", "err", err)
		writeError(w, http.StatusInternalServerError, "internal error")
		return
	}
	writeJSON(w, http.StatusOK, sum)
}

// handleStoreErr writes an error response and returns true if err != nil.
func (s *Server) handleStoreErr(w http.ResponseWriter, err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, store.ErrNotFound) {
		writeError(w, http.StatusNotFound, "not found")
		return true
	}
	s.log.Error("store error", "err", err)
	writeError(w, http.StatusInternalServerError, "internal error")
	return true
}

func parseID(w http.ResponseWriter, r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid id")
		return 0, false
	}
	return id, true
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}
