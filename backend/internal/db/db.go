package db

import (
	"context"
	"embed"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

var execFunc func(string) error

type DBTX interface {
	Exec(context.Context, string, ...interface{}) (pgconn.CommandTag, error)
	Query(context.Context, string, ...interface{}) (pgx.Rows, error)
	QueryRow(context.Context, string, ...interface{}) pgx.Row
	Begin(context.Context) (pgx.Tx, error)
	CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error)
}

func Connect(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, err
	}
	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		return nil, err
	}
	return pool, nil
}

func EnsureDatabase(ctx context.Context, pool *pgxpool.Pool, name string) error {
	_, err := pool.Exec(ctx, "CREATE DATABASE IF NOT EXISTS "+name)
	return err
}

func RunMigrations() error {
	if execFunc == nil {
		return fmt.Errorf("must call WireExec first")
	}
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(f.Name(), ".sql") {
			content, err := migrations.ReadFile("migrations/" + f.Name())
			if err != nil {
				return err
			}
			if err := execFunc(string(content)); err != nil {
				return err
			}
		}
	}
	return nil
}

func SetExecFunc(f func(string) error) {
	execFunc = f
}

func WireExec(ctx context.Context, pool *pgxpool.Pool) {
	SetExecFunc(func(sql string) error {
		_, err := pool.Exec(ctx, sql)
		if err != nil {
			return fmt.Errorf("exec migration: %w", err)
		}
		return nil
	})
}
