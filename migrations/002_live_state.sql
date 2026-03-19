-- Single table that stores the latest snapshot for each F1 live timing topic.
-- Using JSONB gives us flexibility since each topic has a different shape,
-- and we can query inside the JSON with Postgres operators when needed.

CREATE TABLE IF NOT EXISTS live_state (
    topic       TEXT PRIMARY KEY,
    data        JSONB NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_live_state_updated ON live_state(updated_at DESC);
