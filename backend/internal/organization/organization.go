// Package organization owns the scope hierarchy (ADR-014):
// organization > company > branch > warehouse. It is deliberately minimal
// until Fase 03 (identity, sessions and permissions; see ADR-169).
package organization

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store writes the organization tables; no other module writes them.
type Store struct{ pool *pgxpool.Pool }

func NewStore(pool *pgxpool.Pool) *Store { return &Store{pool: pool} }

func (s *Store) insert(ctx context.Context, sql string, args ...any) (uuid.UUID, error) {
	id := uuid.New()
	if _, err := s.pool.Exec(ctx, sql, append([]any{id}, args...)...); err != nil {
		return uuid.Nil, fmt.Errorf("organization: %w", err)
	}
	return id, nil
}

func clean(name string) (string, error) {
	n := strings.TrimSpace(name)
	if n == "" {
		return "", fmt.Errorf("organization: name is required")
	}
	return n, nil
}

func (s *Store) CreateOrganization(ctx context.Context, name string) (uuid.UUID, error) {
	n, err := clean(name)
	if err != nil {
		return uuid.Nil, err
	}
	return s.insert(ctx, `INSERT INTO organizations (id, name) VALUES ($1, $2)`, n)
}

func (s *Store) CreateCompany(ctx context.Context, org uuid.UUID, name string) (uuid.UUID, error) {
	n, err := clean(name)
	if err != nil {
		return uuid.Nil, err
	}
	return s.insert(ctx, `INSERT INTO companies (id, organization_id, name) VALUES ($1, $2, $3)`, org, n)
}

func (s *Store) CreateBranch(ctx context.Context, company uuid.UUID, name string) (uuid.UUID, error) {
	n, err := clean(name)
	if err != nil {
		return uuid.Nil, err
	}
	return s.insert(ctx, `INSERT INTO branches (id, company_id, name) VALUES ($1, $2, $3)`, company, n)
}

// ListCompanyIDs returns every company, for system jobs such as the
// reservation sweeper.
func (s *Store) ListCompanyIDs(ctx context.Context) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `SELECT id FROM companies ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("organization: %w", err)
	}
	defer rows.Close()
	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("organization: %w", err)
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// CreateWarehouse fails if the branch belongs to another company (FK).
func (s *Store) CreateWarehouse(ctx context.Context, company, branch uuid.UUID, name string) (uuid.UUID, error) {
	n, err := clean(name)
	if err != nil {
		return uuid.Nil, err
	}
	return s.insert(ctx, `INSERT INTO warehouses (id, company_id, branch_id, name) VALUES ($1, $2, $3, $4)`, company, branch, n)
}
