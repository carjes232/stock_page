-- Cache for latest quotes to minimize external API usage

CREATE TABLE IF NOT EXISTS quotes_cache (
    symbol      STRING      PRIMARY KEY,
    price       DECIMAL     NOT NULL,
    as_of       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_quotes_cache_asof ON quotes_cache (as_of DESC);

