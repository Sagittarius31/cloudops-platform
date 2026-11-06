package checker

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"cloudops-platform/internal/config"
	"cloudops-platform/internal/metrics"
	"cloudops-platform/internal/store"
)

type fakeStore struct {
	mu      sync.Mutex
	targets []store.Target
	results []store.Result
}

func (f *fakeStore) ListTargets(context.Context) ([]store.Target, error) { return f.targets, nil }
func (f *fakeStore) InsertResult(_ context.Context, r store.Result) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.results = append(f.results, r)
	return nil
}
func (f *fakeStore) Prune(context.Context, time.Time) (int64, error) { return 0, nil }

func TestRunOnceRecordsSuccessAndFailure(t *testing.T) {
	ok := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ok.Close()
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()

	fs := &fakeStore{targets: []store.Target{
		{ID: 1, Name: "ok", URL: ok.URL, ExpectedStatus: 200},
		{ID: 2, Name: "bad", URL: bad.URL, ExpectedStatus: 200},
	}}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	c := New(fs, metrics.New(), log, config.Config{
		CheckInterval: time.Minute, CheckTimeout: 2 * time.Second, Workers: 2, RetentionDays: 1,
		AllowPrivateTargets: true, // the httptest servers listen on loopback
	})

	c.runOnce(context.Background())

	if len(fs.results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(fs.results))
	}
	for _, r := range fs.results {
		switch r.TargetID {
		case 1:
			if !r.Success {
				t.Errorf("target 1 should succeed: %+v", r)
			}
		case 2:
			if r.Success || r.StatusCode != 500 {
				t.Errorf("target 2 should fail with 500: %+v", r)
			}
		}
	}
}

func TestPrivateTargetsBlockedByDefault(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	fs := &fakeStore{targets: []store.Target{{ID: 1, Name: "internal", URL: srv.URL, ExpectedStatus: 200}}}
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	c := New(fs, metrics.New(), log, config.Config{
		CheckInterval: time.Minute, CheckTimeout: 2 * time.Second, Workers: 1, RetentionDays: 1,
		// AllowPrivateTargets left false: loopback must be refused at connection time
	})

	c.runOnce(context.Background())

	if len(fs.results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(fs.results))
	}
	if fs.results[0].Success {
		t.Fatal("a loopback target must not be reachable when private targets are disallowed")
	}
}
