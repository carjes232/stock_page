-- Create database objects for stocks app

CREATE TABLE IF NOT EXISTS stocks (
    id                UUID        DEFAULT gen_random_uuid() PRIMARY KEY,
    ticker            STRING      NOT NULL UNIQUE,
    company           STRING      NOT NULL,
    brokerage         STRING      NOT NULL,
    action            STRING      NOT NULL,
    rating_from       STRING      NOT NULL,
    rating_to         STRING      NOT NULL,
    target_from       DECIMAL     NULL,
    target_to         DECIMAL     NULL,
    last_rating_change_at TIMESTAMPTZ NULL,
    price_target_delta DECIMAL    GENERATED ALWAYS AS (COALESCE(target_to, 0) - COALESCE(target_from, 0)) STORED,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_stocks_ticker ON stocks (ticker);
CREATE INDEX IF NOT EXISTS idx_stocks_updated_at ON stocks (updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_stocks_price_target_delta ON stocks (price_target_delta DESC);

