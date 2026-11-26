// Package pgtest gives integration tests a real PostgreSQL with a fresh,
// migrated schema per test. It needs BP_TEST_DATABASE_URL; without it the
// test is skipped, unless BP_REQUIRE_DB=1 (CI), where it fails instead.
//
// Guards against touching a real database (corrective review of PR #1):
//   - only usable from `go test` (testing.Testing);
//   - the URL must point to a loopback host or a local Unix socket, to a
//     database whose name is "test" or ends in "_test", and must not be the
//     application's BP_DATABASE_URL;
//   - the database must not hold a deployed schema (goose table in public);
//   - every test works in its own random schema t_<hex>, dropped afterwards.
package pgtest

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/postgres"
)

// New returns a pool bound to a new schema with all migrations applied.
func New(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if !testing.Testing() {
		panic("pgtest: only for use in tests")
	}
	url := os.Getenv("BP_TEST_DATABASE_URL")
	if url == "" {
		if os.Getenv("BP_REQUIRE_DB") == "1" {
			t.Fatal("BP_REQUIRE_DB=1 but BP_TEST_DATABASE_URL is not set")
		}
		t.Skip("integration test: set BP_TEST_DATABASE_URL to run it")
	}
	if err := checkURL(url, os.Getenv("BP_DATABASE_URL")); err != nil {
		t.Fatalf("pgtest: refusing BP_TEST_DATABASE_URL: %v", err)
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
	if err := checkDatabase(ctx, admin); err != nil {
		_ = admin.Close(ctx)
		t.Fatalf("pgtest: refusing database: %v", err)
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

// checkURL accepts only local, test-named databases distinct from prodURL.
func checkURL(url, prodURL string) error {
	cfg, err := pgx.ParseConfig(url)
	if err != nil {
		return fmt.Errorf("unparsable URL: %w", err)
	}
	if len(cfg.Fallbacks) > 0 {
		return errors.New("multiple hosts are not allowed")
	}
	if !isLocalHost(cfg.Host) {
		return fmt.Errorf("host %q is not loopback or a local socket", cfg.Host)
	}
	if !isTestName(cfg.Database) {
		return fmt.Errorf("database %q must be named \"test\" or end in \"_test\"", cfg.Database)
	}
	if prodURL != "" {
		p, err := pgx.ParseConfig(prodURL)
		if err != nil {
			return errors.New("BP_DATABASE_URL is set but unparsable; refusing to guess whether it is the test database")
		}
		if sameServer(p.Host, cfg.Host) && p.Port == cfg.Port && p.Database == cfg.Database {
			return errors.New("BP_TEST_DATABASE_URL points to the same database as BP_DATABASE_URL")
		}
	}
	return nil
}

// sameServer treats every loopback spelling as one host.
func sameServer(a, b string) bool {
	norm := func(h string) string {
		if h == "localhost" {
			return "loopback"
		}
		if ip := net.ParseIP(h); ip != nil && ip.IsLoopback() {
			return "loopback"
		}
		return h
	}
	return norm(a) == norm(b)
}

func isLocalHost(h string) bool {
	if strings.HasPrefix(h, "/") || h == "localhost" {
		return true
	}
	ip := net.ParseIP(h)
	return ip != nil && ip.IsLoopback()
}

func isTestName(db string) bool { return db == "test" || strings.HasSuffix(db, "_test") }

// checkDatabase verifies the connected database is disposable: test name
// and no deployed schema in public.
func checkDatabase(ctx context.Context, c *pgx.Conn) error {
	var name string
	var deployed bool
	err := c.QueryRow(ctx, `SELECT current_database(), to_regclass('public.goose_db_version') IS NOT NULL`).Scan(&name, &deployed)
	if err != nil {
		return err
	}
	if !isTestName(name) {
		return fmt.Errorf("connected database %q is not a test database", name)
	}
	if deployed {
		return fmt.Errorf("database %q has migrations applied in public (looks like a real deployment); use an empty test database", name)
	}
	return nil
}
