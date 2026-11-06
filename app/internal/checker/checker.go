// Package checker runs periodic HTTP health checks against registered targets.
package checker

import (
	"context"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"cloudops-platform/internal/config"
	"cloudops-platform/internal/metrics"
	"cloudops-platform/internal/netguard"
	"cloudops-platform/internal/store"
)

// Store is the subset of the data layer the checker needs (small interface = easy to fake in tests).
type Store interface {
	ListTargets(ctx context.Context) ([]store.Target, error)
	InsertResult(ctx context.Context, r store.Result) error
	Prune(ctx context.Context, before time.Time) (int64, error)
}

type Checker struct {
	store     Store
	metrics   *metrics.Metrics
	log       *slog.Logger
	client    *http.Client
	interval  time.Duration
	workers   int
	retention time.Duration
}

func New(st Store, m *metrics.Metrics, log *slog.Logger, cfg config.Config) *Checker {
	// Unless private targets are explicitly allowed, refuse to connect to internal addresses.
	// Enforced on the resolved IP at dial time, so redirects and DNS rebinding cannot bypass it.
	dialer := &net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}
	if !cfg.AllowPrivateTargets {
		dialer.Control = netguard.Control
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.DialContext = dialer.DialContext

	return &Checker{
		store:     st,
		metrics:   m,
		log:       log,
		client:    &http.Client{Timeout: cfg.CheckTimeout, Transport: transport},
		interval:  cfg.CheckInterval,
		workers:   cfg.Workers,
		retention: time.Duration(cfg.RetentionDays) * 24 * time.Hour,
	}
}

// Run blocks until ctx is cancelled.
func (c *Checker) Run(ctx context.Context) {
	tick := time.NewTicker(c.interval)
	defer tick.Stop()
	prune := time.NewTicker(time.Hour)
	defer prune.Stop()

	c.runOnce(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			c.runOnce(ctx)
		case <-prune.C:
			n, err := c.store.Prune(ctx, time.Now().Add(-c.retention))
			if err != nil {
				c.log.Error("prune failed", "err", err)
			} else if n > 0 {
				c.log.Info("pruned old results", "rows", n)
			}
		}
	}
}

func (c *Checker) runOnce(ctx context.Context) {
	targets, err := c.store.ListTargets(ctx)
	if err != nil {
		c.log.Error("list targets failed", "err", err)
		return
	}
	sem := make(chan struct{}, c.workers)
	var wg sync.WaitGroup
	for _, t := range targets {
		wg.Add(1)
		sem <- struct{}{}
		go func(t store.Target) {
			defer wg.Done()
			defer func() { <-sem }()
			c.check(ctx, t)
		}(t)
	}
	wg.Wait()
}

func (c *Checker) check(ctx context.Context, t store.Target) {
	start := time.Now()
	res := store.Result{TargetID: t.ID, CheckedAt: start.UTC()}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.URL, nil)
	if err != nil {
		res.Error = err.Error()
	} else {
		req.Header.Set("User-Agent", "sentinel-checker/1.0")
		resp, err := c.client.Do(req)
		if err != nil {
			res.Error = err.Error()
		} else {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
			resp.Body.Close()
			res.StatusCode = resp.StatusCode
			res.Success = resp.StatusCode == t.ExpectedStatus
			if !res.Success {
				res.Error = "unexpected status code"
			}
		}
	}
	elapsed := time.Since(start)
	res.LatencyMs = float64(elapsed.Microseconds()) / 1000.0

	label := "failure"
	up := 0.0
	if res.Success {
		label = "success"
		up = 1.0
	}
	c.metrics.CheckTotal.WithLabelValues(t.Name, label).Inc()
	c.metrics.CheckDuration.WithLabelValues(t.Name).Observe(elapsed.Seconds())
	c.metrics.TargetUp.WithLabelValues(t.Name).Set(up)

	if err := c.store.InsertResult(ctx, res); err != nil {
		c.log.Error("store result failed", "target", t.Name, "err", err)
	}
}
