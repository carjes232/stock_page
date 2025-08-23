package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	"stockchallenge/backend/internal/api"
	"stockchallenge/backend/internal/config"
	"stockchallenge/backend/internal/db"
	"stockchallenge/backend/internal/ingest"
	"stockchallenge/backend/internal/marketdata"
	"stockchallenge/backend/internal/rec"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

func main() {
	_ = godotenv.Load()

	logger, _ := zap.NewProduction()
	defer logger.Sync()
	sugar := logger.Sugar()

	// Load config
	cfg, err := config.Load()
	if err != nil {
		sugar.Fatalf("config load error: %v", err)
	}

	// DB connect
	pool, err := db.Connect(context.Background(), cfg.DBURL)
	if err != nil {
		sugar.Fatalf("db connect error: %v", err)
	}
	defer pool.Close()

	// Ensure database exists and run migrations
	if err := db.EnsureDatabase(context.Background(), pool, "stocks"); err != nil {
		sugar.Fatalf("ensure db error: %v", err)
	}
	// Wire exec function so RunMigrations can execute SQL files using this pool
	db.WireExec(context.Background(), pool)
	if err := db.RunMigrations(); err != nil {
		sugar.Fatalf("migrations error: %v", err)
	}

	// Services
	ing := ingest.NewService(cfg.APIBase, cfg.APIToken, pool, sugar)
	recommender := rec.NewService(pool)
	// Wire optional price provider if configured
	if cfg.FMPAPIKey != "" {
		pp := marketdata.NewFMPClient(cfg.FMPAPIKey)
		recommender.SetPriceProvider(pp)
	}
	// Enable quote cache and top-K enrichment
	recommender.EnableQuoteCache(cfg.QuotesTTL)
	recommender.SetTopK(cfg.PriceTopK)

	// Optionally run ingestion on start
	if cfg.IngestOnStart {
		go func() {
			if err := ing.RunOnce(context.Background()); err != nil {
				sugar.Warnf("initial ingest error: %v", err)
			}
		}()
	}

	// Start cron
	cronStop := make(chan struct{})
	go ingest.StartCron(ing, cfg.IngestInterval, sugar, cronStop)

	// HTTP router
	router := api.NewRouter(pool, ing, recommender, sugar)

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.BackendPort),
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	// Graceful shutdown
	idleConnsClosed := make(chan struct{})
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		<-c
		close(cronStop)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = srv.Shutdown(ctx)
		close(idleConnsClosed)
	}()

	sugar.Infof("backend listening on :%d", cfg.BackendPort)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		sugar.Fatalf("server error: %v", err)
	}

	<-idleConnsClosed
}
