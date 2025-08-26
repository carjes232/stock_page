-- Create table for user portfolios
CREATE TABLE IF NOT EXISTS portfolio (
    id UUID DEFAULT gen_random_uuid() PRIMARY KEY,
    user_id UUID NOT NULL,
    ticker STRING NOT NULL,
    position DECIMAL NOT NULL,
    average_price DECIMAL NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, ticker)
);

CREATE INDEX IF NOT EXISTS idx_portfolio_user_id_ticker ON portfolio (user_id, ticker);
