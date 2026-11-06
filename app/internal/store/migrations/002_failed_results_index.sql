-- Speeds up "recent failures" lookups; partial index stays small because most checks succeed.
CREATE INDEX IF NOT EXISTS idx_results_failed ON check_results (target_id, checked_at DESC) WHERE NOT success;
