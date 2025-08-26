-- Optional fundamentals storage for EPS history, estimates, and macro yields

CREATE TABLE IF NOT EXISTS eps_points (
    id               UUID         DEFAULT gen_random_uuid() PRIMARY KEY,
    ticker           STRING       NOT NULL,
    period_date      DATE         NOT NULL,
    eps_reported     DECIMAL      NULL,
    eps_basic        DECIMAL      NULL,
    eps_diluted      DECIMAL      NULL,
    is_estimate      BOOL         NOT NULL DEFAULT false,
    surprise_percent DECIMAL      NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT now(),
    UNIQUE (ticker, period_date, is_estimate)
);

CREATE INDEX IF NOT EXISTS idx_eps_points_ticker_date ON eps_points (ticker, period_date DESC);

-- Aggregated per-ticker fundamentals (precomputed if desired)
CREATE TABLE IF NOT EXISTS fundamentals (
    ticker           STRING       PRIMARY KEY,
    eps_avg          DECIMAL      NULL,
    growth_estimate  DECIMAL      NULL,
    surprise_sum     DECIMAL      NULL,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT now()
);

-- Macro series cache (e.g., AAA corporate bond yield)
CREATE TABLE IF NOT EXISTS macro_series (
    series     STRING       PRIMARY KEY,
    as_of      TIMESTAMPTZ  NOT NULL,
    value      DECIMAL      NOT NULL,
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT now()
);

