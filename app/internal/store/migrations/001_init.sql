CREATE TABLE IF NOT EXISTS targets (
    id              BIGSERIAL PRIMARY KEY,
    name            TEXT NOT NULL UNIQUE,
    url             TEXT NOT NULL,
    expected_status INTEGER NOT NULL DEFAULT 200,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS check_results (
    id          BIGSERIAL PRIMARY KEY,
    target_id   BIGINT NOT NULL REFERENCES targets(id) ON DELETE CASCADE,
    checked_at  TIMESTAMPTZ NOT NULL,
    status_code INTEGER NOT NULL DEFAULT 0,
    latency_ms  DOUBLE PRECISION NOT NULL DEFAULT 0,
    success     BOOLEAN NOT NULL,
    error       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_results_target_time ON check_results (target_id, checked_at DESC);
CREATE INDEX IF NOT EXISTS idx_results_time ON check_results (checked_at);
