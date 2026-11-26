// Package pgtest gives integration tests a real PostgreSQL with a fresh,
// migrated schema per test. It needs BP_TEST_DATABASE_URL; without it the
// test is skipped, unless BP_REQUIRE_DB=1 (CI), where it fails instead.
package pgtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/postgres"
)

// New returns a pool bound to a new schema with all migrations applied.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("BP_TEST_DATABASE_URL")
	if url == "" {
		if os.Getenv("BP_REQUIRE_DB") == "1" {
			t.Fatal("BP_REQUIRE_DB=1 but BP_TEST_DATABASE_URL is not set")
		}
		t.Skip("integration test: set BP_TEST_DATABASE_URL to run it")
	}
	ctx := context.Background()

	b := make([]byte, 6)
	if _, err := rand.Read(b); err != nil {
		t.Fatal(err)
	}
	schema := "t_" + hex.EncodeToString(b)

	admin, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatalf("pgtest: connect: %v", err)
	}
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatalf("pgtest: create schema: %v", err)
	}
	_ = admin.Close(ctx)

	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 32
	pool, err := postgres.ConnectConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := postgres.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		c, err := pgx.Connect(context.Background(), url)
		if err != nil {
			return
		}
		_, _ = c.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
		_ = c.Close(context.Background())
	})
	return pool
}
