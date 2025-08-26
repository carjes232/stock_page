package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	BackendPort       int
	DBURL             string
	APIBase           string
	APIToken          string
	IngestInterval    time.Duration
	IngestOnStart     bool
	FMPAPIKey         string
	QuotesTTL         time.Duration
	PriceTopK         int
	FundamentalsTTL   time.Duration
	PriceWarmInterval time.Duration
	// If true, do not wire the built-in Graham valuation provider (EPS/growth).
	// This lets external tools populate the fundamentals table instead.
	DisableGrahamProvider bool
	// Optional Python Fundamentals API base URL (inside Compose: http://fundamentals-api:9000)
	FundamentalsAPIBase string
	// Intervals for external updaters (used by docs/compose only)
	PriceUpdateInterval        time.Duration
	FundamentalsUpdateInterval time.Duration
	GeminiAPIKey               string
	GeminiModelID              string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() (*Config, error) {
	portStr := getenv("BACKEND_PORT", "8080")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("invalid BACKEND_PORT: %w", err)
	}

	dbURL := getenv("DB_URL", "postgresql://root@db:26259/stocks?sslmode=disable")
	apiBase := getenv("API_BASE", "")
	apiToken := getenv("API_TOKEN", "")
	fmpAPIKey := getenv("FMP_API_KEY", "")

	// Quote cache TTL for price enrichment (default 10m)
	quotesTTLStr := getenv("QUOTES_TTL", "10m")
	quotesTTL, err := time.ParseDuration(quotesTTLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid QUOTES_TTL: %w", err)
	}

	// Price enrichment top-K (fetch quotes only for these before final sort)
	topKStr := getenv("PRICE_TOPK", "20")
	topK, err := strconv.Atoi(topKStr)
	if err != nil || topK < 1 {
		topK = 20
	}

	// Fundamentals cache TTL (default 30d)
	fundTTLStr := getenv("FUNDAMENTALS_TTL", "720h") // 30d
	fundTTL, err := time.ParseDuration(fundTTLStr)
	if err != nil {
		return nil, fmt.Errorf("invalid FUNDAMENTALS_TTL: %w", err)
	}

	// How often to proactively warm top-K quotes/fundamentals (default daily)
	warmStr := getenv("PRICE_WARM_INTERVAL", "24h")
	warmEvery, err := time.ParseDuration(warmStr)
	if err != nil {
		return nil, fmt.Errorf("invalid PRICE_WARM_INTERVAL: %w", err)
	}

	intervalStr := getenv("INGEST_INTERVAL", "15m")
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid INGEST_INTERVAL: %w", err)
	}

	ingestOnStart := getenv("INGEST_ON_START", "true") == "true"
	disableGraham := getenv("DISABLE_GRAHAM_PROVIDER", "false") == "true"

	fundamentalsAPIBase := getenv("FUNDAMENTALS_API_BASE", "")

	// External updater intervals (used by scheduler service; keep here for clarity)
	priceUpdStr := getenv("PRICE_UPDATE_INTERVAL", "24h")
	priceUpdEvery, err := time.ParseDuration(priceUpdStr)
	if err != nil {
		return nil, fmt.Errorf("invalid PRICE_UPDATE_INTERVAL: %w", err)
	}
	fundUpdStr := getenv("FUNDAMENTALS_UPDATE_INTERVAL", "720h") // 30d
	fundUpdEvery, err := time.ParseDuration(fundUpdStr)
	if err != nil {
		return nil, fmt.Errorf("invalid FUNDAMENTALS_UPDATE_INTERVAL: %w", err)
	}

	geminiAPIKey := getenv("GEMINI_API_KEY", "")
	geminiModelID := getenv("GEMINI_MODEL_ID", "gemini-2.5-flash-lite")

	return &Config{
		BackendPort:                port,
		DBURL:                      dbURL,
		APIBase:                    apiBase,
		APIToken:                   apiToken,
		IngestInterval:             interval,
		IngestOnStart:              ingestOnStart,
		FMPAPIKey:                  fmpAPIKey,
		QuotesTTL:                  quotesTTL,
		PriceTopK:                  topK,
		FundamentalsTTL:            fundTTL,
		PriceWarmInterval:          warmEvery,
		DisableGrahamProvider:      disableGraham,
		FundamentalsAPIBase:        fundamentalsAPIBase,
		PriceUpdateInterval:        priceUpdEvery,
		FundamentalsUpdateInterval: fundUpdEvery,
		GeminiAPIKey:               geminiAPIKey,
		GeminiModelID:              geminiModelID,
	}, nil
}
