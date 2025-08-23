package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	BackendPort    int
	DBURL          string
	APIBase        string
	APIToken       string
	IngestInterval time.Duration
	IngestOnStart  bool
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

	// IMPORTANT: The CockroachDB service in docker-compose listens on 26259 inside the container.
	// Always default to 26259 to avoid connecting to the RPC port (26257) by mistake.
	dbURL := getenv("DB_URL", "postgresql://root@db:26259/stocks?sslmode=disable")
	apiBase := getenv("API_BASE", "")
	apiToken := getenv("API_TOKEN", "")

	intervalStr := getenv("INGEST_INTERVAL", "15m")
	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid INGEST_INTERVAL: %w", err)
	}

	ingestOnStart := getenv("INGEST_ON_START", "true") == "true"

	return &Config{
		BackendPort:    port,
		DBURL:          dbURL,
		APIBase:        apiBase,
		APIToken:       apiToken,
		IngestInterval: interval,
		IngestOnStart:  ingestOnStart,
	}, nil
}
