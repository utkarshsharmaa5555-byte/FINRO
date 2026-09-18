package app

import (
	"context"
	_ "embed"
	"errors"
	"os"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed schema.sql
var schemaSQL string

var (
	poolOnce sync.Once
	pool     *pgxpool.Pool
	poolErr  error
)

// DB returns a process-wide pool; serverless instances reuse it across warm invocations.
func DB() (*pgxpool.Pool, error) {
	poolOnce.Do(func() {
		url := os.Getenv("DATABASE_URL")
		if url == "" {
			poolErr = errors.New("DATABASE_URL not set")
			return
		}
		cfg, err := pgxpool.ParseConfig(url)
		if err != nil {
			poolErr = err
			return
		}
		cfg.MaxConns = 4
		pool, poolErr = pgxpool.NewWithConfig(context.Background(), cfg)
	})
	return pool, poolErr
}

func Migrate(ctx context.Context) error {
	db, err := DB()
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, schemaSQL)
	return err
}
