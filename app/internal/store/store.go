// Package store is the PostgreSQL data layer.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/lib/pq"
)

var (
	ErrNotFound = errors.New("not found")
	ErrConflict = errors.New("already exists")
)

type Target struct {
	ID             int64     `json:"id"`
	Name           string    `json:"name"`
	URL            string    `json:"url"`
	ExpectedStatus int       `json:"expected_status"`
	CreatedAt      time.Time `json:"created_at"`
}

type Result struct {
	ID         int64     `json:"id"`
	TargetID   int64     `json:"target_id"`
	CheckedAt  time.Time `json:"checked_at"`
	StatusCode int       `json:"status_code"`
	LatencyMs  float64   `json:"latency_ms"`
	Success    bool      `json:"success"`
	Error      string    `json:"error,omitempty"`
}

type Summary struct {
	TargetID      int64   `json:"target_id"`
	Window        string  `json:"window"`
	Checks        int64   `json:"checks"`
	Successes     int64   `json:"successes"`
	UptimePercent float64 `json:"uptime_percent"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
}

type Store struct {
	db *sql.DB
}

// Open connects (retrying while the database starts up) and applies the schema.
func Open(ctx context.Context, dsn string) (*Store, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	var lastErr error
	for i := 0; i < 15; i++ {
		pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		lastErr = db.PingContext(pctx)
		cancel()
		if lastErr == nil {
			break
		}
		select {
		case <-ctx.Done():
			db.Close()
			return nil, ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	if lastErr != nil {
		db.Close()
		return nil, fmt.Errorf("database not reachable: %w", lastErr)
	}
	if err := migrate(ctx, db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error                   { return s.db.Close() }
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

func (s *Store) CreateTarget(ctx context.Context, name, url string, expected int) (Target, error) {
	var t Target
	err := s.db.QueryRowContext(ctx,
		`INSERT INTO targets (name, url, expected_status) VALUES ($1, $2, $3)
		 RETURNING id, name, url, expected_status, created_at`,
		name, url, expected,
	).Scan(&t.ID, &t.Name, &t.URL, &t.ExpectedStatus, &t.CreatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return Target{}, ErrConflict
		}
		return Target{}, err
	}
	return t, nil
}

func (s *Store) ListTargets(ctx context.Context) ([]Target, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, name, url, expected_status, created_at FROM targets ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Target{}
	for rows.Next() {
		var t Target
		if err := rows.Scan(&t.ID, &t.Name, &t.URL, &t.ExpectedStatus, &t.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (s *Store) GetTarget(ctx context.Context, id int64) (Target, error) {
	var t Target
	err := s.db.QueryRowContext(ctx,
		`SELECT id, name, url, expected_status, created_at FROM targets WHERE id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.URL, &t.ExpectedStatus, &t.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Target{}, ErrNotFound
	}
	return t, err
}

func (s *Store) DeleteTarget(ctx context.Context, id int64) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM targets WHERE id = $1`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) InsertResult(ctx context.Context, r Result) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO check_results (target_id, checked_at, status_code, latency_ms, success, error)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		r.TargetID, r.CheckedAt, r.StatusCode, r.LatencyMs, r.Success, r.Error)
	return err
}

func (s *Store) ListResults(ctx context.Context, targetID int64, limit int) ([]Result, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, target_id, checked_at, status_code, latency_ms, success, error
		 FROM check_results WHERE target_id = $1 ORDER BY checked_at DESC LIMIT $2`,
		targetID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Result{}
	for rows.Next() {
		var r Result
		if err := rows.Scan(&r.ID, &r.TargetID, &r.CheckedAt, &r.StatusCode, &r.LatencyMs, &r.Success, &r.Error); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (s *Store) Summary(ctx context.Context, targetID int64, window time.Duration) (Summary, error) {
	sum := Summary{TargetID: targetID, Window: window.String()}
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*),
		        COALESCE(SUM(CASE WHEN success THEN 1 ELSE 0 END), 0),
		        COALESCE(AVG(latency_ms), 0)
		 FROM check_results WHERE target_id = $1 AND checked_at >= $2`,
		targetID, time.Now().Add(-window),
	).Scan(&sum.Checks, &sum.Successes, &sum.AvgLatencyMs)
	if err != nil {
		return sum, err
	}
	if sum.Checks > 0 {
		sum.UptimePercent = float64(sum.Successes) / float64(sum.Checks) * 100
	}
	return sum, nil
}

// Prune deletes results older than the cutoff and returns how many rows were removed.
func (s *Store) Prune(ctx context.Context, before time.Time) (int64, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM check_results WHERE checked_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
