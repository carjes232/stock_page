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
	fredClient := marketdata.NewFredClient()
	recommender.SetCorporateBondYieldProvider(fredClient)
    // We now rely on the Python Fundamentals API to populate quotes_cache and
    // fundamentals in the database. The Go service reads from cache and does
    // not call external price or fundamentals providers directly.
	// Enable quote cache and top-K enrichment
	recommender.EnableQuoteCache(cfg.QuotesTTL)
    recommender.EnableGrahamValuation(cfg.FundamentalsTTL)
    recommender.SetTopK(cfg.PriceTopK)


    // Optionally run ingestion on start
	if cfg.IngestOnStart {
		go func() {
			if err := ing.RunOnce(context.Background()); err != nil {
				sugar.Warnf("initial ingest error: %v", err)
			}
		}()
	}

    // Start ingestion cron
    cronStop := make(chan struct{})
    go ingest.StartCron(ing, cfg.IngestInterval, sugar, cronStop)

    // Start daily warm cron to prefetch quotes and fundamentals for top-K
    warmStop := make(chan struct{})
    go func() {
        t := time.NewTicker(cfg.PriceWarmInterval)
        defer t.Stop()
        for {
            // Do an initial warm immediately at startup
            ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
            _ = recommender.WarmCachesForTopK(ctx)
            cancel()
            break
        }
        for {
            select {
            case <-t.C:
                ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
                if err := recommender.WarmCachesForTopK(ctx); err != nil {
                    sugar.Warnf("warm caches error: %v", err)
                }
                cancel()
            case <-warmStop:
                sugar.Infof("warm cron stopped")
                return
            }
        }
    }()

	// HTTP router
    router := api.NewRouter(pool, ing, recommender, sugar, cfg.FundamentalsAPIBase)

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
        close(warmStop)
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
