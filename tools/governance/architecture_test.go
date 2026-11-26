package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestForbidden(t *testing.T) {
	for _, c := range []struct {
		path, imp string
		bad       bool
	}{
		{"internal/inventory/domain/a.go", "github.com/jackc/pgx/v5", true},
		{"internal/inventory/application/a.go", module + "internal/inventory/adapters/pgstore", true},
		{"internal/inventory/adapters/pgstore/a.go", "github.com/jackc/pgx/v5", false},
		{"internal/organization/organization.go", "github.com/jackc/pgx/v5/pgxpool", false},
		{"internal/organization/new.go", "database/sql", true},
		{"internal/finance/application/a.go", module + "internal/inventory/adapters/pgstore", true},
	} {
		if forbidden(c.path, c.imp) != c.bad {
			t.Errorf("unexpected rule result: %v", c)
		}
	}
}
func TestFloatAndCycle(t *testing.T) {
	root := t.TempDir()
	for path, source := range map[string]string{
		"internal/a/domain/a.go": "package a; import _ \"" + module + "internal/b/domain\"; var amount float64",
		"internal/b/domain/b.go": "package b; import _ \"" + module + "internal/a/domain\"",
	} {
		f := filepath.Join(root, path)
		if err := os.MkdirAll(filepath.Dir(f), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(f, []byte(source), 0600); err != nil {
			t.Fatal(err)
		}
	}
	v, err := inspect(root)
	if err != nil || len(v) != 2 {
		t.Fatalf("expected float and cycle: %v %v", v, err)
	}
}
