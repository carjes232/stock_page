-- User watchlist for fundamentals/quotes coverage

CREATE TABLE IF NOT EXISTS watchlist (
    ticker     STRING       PRIMARY KEY,
    notes      STRING       NULL,
    added_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Helpful index when joining/merging
CREATE INDEX IF NOT EXISTS idx_watchlist_added ON watchlist (added_at DESC);

