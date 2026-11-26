// Package postgres connects to PostgreSQL (the single authority, ADR-003)
// and applies the embedded, versioned migrations with goose.
package postgres

import (
	"context"
	"embed"
	"fmt"
	"io/fs"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Connect opens a pool and verifies the connection.
func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("postgres: parse config: %w", err)
	}
	return ConnectConfig(ctx, cfg)
}

// ConnectConfig opens a pool from a parsed config and pings it.
func ConnectConfig(ctx context.Context, cfg *pgxpool.Config) (*pgxpool.Pool, error) {
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("postgres: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("postgres: ping: %w", err)
	}
	return pool, nil
}

func provider(pool *pgxpool.Pool) (*goose.Provider, func(), error) {
	db := stdlib.OpenDBFromPool(pool)
	sub, err := fs.Sub(migrations, "migrations")
	if err != nil {
		db.Close()
		return nil, nil, err
	}
	p, err := goose.NewProvider(goose.DialectPostgres, db, sub)
	if err != nil {
		db.Close()
		return nil, nil, fmt.Errorf("postgres: migrations: %w", err)
	}
	return p, func() { db.Close() }, nil
}

// Migrate applies every pending migration.
func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	p, done, err := provider(pool)
	if err != nil {
		return err
	}
	defer done()
	if _, err := p.Up(ctx); err != nil {
		return fmt.Errorf("postgres: migrate up: %w", err)
	}
	return nil
}

// RollbackTo runs the Down sections until the schema is at version.
// Operational recovery only; never run against production without a
// verified backup (docs/deployment/operations.md).
func RollbackTo(ctx context.Context, pool *pgxpool.Pool, version int64) error {
	p, done, err := provider(pool)
	if err != nil {
		return err
	}
	defer done()
	if _, err := p.DownTo(ctx, version); err != nil {
		return fmt.Errorf("postgres: migrate down to %d: %w", version, err)
	}
	return nil
}
