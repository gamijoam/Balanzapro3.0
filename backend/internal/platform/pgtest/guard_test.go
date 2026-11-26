package pgtest

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Corrective review: tests must never run against a non-disposable database.
func TestCheckURLAcceptsOnlyLocalTestDatabases(t *testing.T) {
	for _, ok := range []string{
		"postgres://bp@127.0.0.1:55432/bp_test?sslmode=disable",
		"postgres://bp:pw@localhost:5432/test?sslmode=disable",
		"postgres://bp@[::1]:5432/inventory_test?sslmode=disable",
		"host=/var/run/postgresql dbname=bp_test user=bp",
	} {
		assert.NoError(t, checkURL(ok, ""), ok)
	}
	for _, bad := range []string{
		"postgres://bp@127.0.0.1:5432/balanzapro?sslmode=disable",                  // not a *_test name
		"postgres://bp@127.0.0.1:5432/bp_test_prod?sslmode=disable",                // suffix must be _test
		"postgres://bp@db.example.com:5432/bp_test?sslmode=disable",                // remote host
		"postgres://bp@10.0.0.5:5432/bp_test?sslmode=disable",                      // non-loopback IP
		"postgres://bp@127.0.0.1:5432/bp_test?sslmode=disable&host=db.example.com", // multi-host
		"not a url %%%",
	} {
		assert.Error(t, checkURL(bad, ""), bad)
	}
	same := "postgres://bp@127.0.0.1:5432/bp_test?sslmode=disable"
	assert.Error(t, checkURL(same, "postgres://other@127.0.0.1:5432/bp_test"), "must differ from BP_DATABASE_URL")
}

// A database that already holds a deployed schema (goose table in public)
// is refused even if its name ends in _test.
func TestCheckDatabaseRefusesDeployedSchema(t *testing.T) {
	url := os.Getenv("BP_TEST_DATABASE_URL")
	if url == "" {
		if os.Getenv("BP_REQUIRE_DB") == "1" {
			t.Fatal("BP_TEST_DATABASE_URL is required")
		}
		t.Skip("set BP_TEST_DATABASE_URL")
	}
	ctx := context.Background()
	admin, err := pgx.Connect(ctx, url)
	require.NoError(t, err)
	defer admin.Close(ctx)
	require.NoError(t, checkDatabase(ctx, admin), "the disposable test database itself is accepted")

	_, _ = admin.Exec(ctx, `DROP DATABASE IF EXISTS bp_guard_probe_test`)
	_, err = admin.Exec(ctx, `CREATE DATABASE bp_guard_probe_test`)
	require.NoError(t, err)
	defer func() { _, _ = admin.Exec(ctx, `DROP DATABASE IF EXISTS bp_guard_probe_test`) }()

	cfg, err := pgx.ParseConfig(url)
	require.NoError(t, err)
	cfg.Database = "bp_guard_probe_test"
	probe, err := pgx.ConnectConfig(ctx, cfg)
	require.NoError(t, err)
	defer probe.Close(ctx) // runs before the deferred DROP DATABASE above
	_, err = probe.Exec(ctx, `CREATE TABLE public.goose_db_version (id int)`)
	require.NoError(t, err)
	assert.Error(t, checkDatabase(ctx, probe), "a migrated public schema looks like a real deployment")
}

// Second independent review: equivalent loopback spellings are the same
// server, and an unparsable BP_DATABASE_URL fails closed.
func TestCheckURLComparesNormalizedProductionURL(t *testing.T) {
	test := "postgres://bp@127.0.0.1:5432/app_test?sslmode=disable"
	assert.Error(t, checkURL(test, "postgres://bp@localhost:5432/app_test"))
	assert.Error(t, checkURL(test, "postgres://bp@[::1]:5432/app_test"))
	assert.Error(t, checkURL(test, "%%% not parsable"), "fail closed")
	assert.NoError(t, checkURL(test, "postgres://bp@localhost:5432/app"))
	assert.NoError(t, checkURL(test, "postgres://bp@localhost:5433/app_test"))
}
