package pgstore_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// Corrective review: the database rejects a base unit outside the SKU's
// dimension even if a writer bypasses the domain (defence in depth).
func TestDatabaseRejectsUnitOutsideDimension(t *testing.T) {
	e := setup(t)
	_, err := e.pool.Exec(e.ctx, `
		INSERT INTO inventory_skus (id, company_id, code, name, quantity_kind, base_unit, precision)
		VALUES ($1, $2, 'BAD', 'bad', 'weight', 'l', 3)`, uuid.New(), e.sc.CompanyID)
	assert.Error(t, err)
	_, err = e.pool.Exec(e.ctx, `
		INSERT INTO inventory_skus (id, company_id, code, name, quantity_kind, base_unit, precision)
		VALUES ($1, $2, 'OK', 'ok', 'volume', 'ml', 0)`, uuid.New(), e.sc.CompanyID)
	assert.NoError(t, err)
}
