package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/modules/domain"
)

// ADR-023 acceptance at the domain level. Synthetic manifests only.

func catalog(t *testing.T) domain.Catalog {
	t.Helper()
	c, err := domain.NewCatalog(append(domain.BaseManifests(),
		domain.Manifest{ID: "inventory", Version: "1.0.0", Permissions: []string{"inventory.adjust"}},
		domain.Manifest{ID: "sales", Version: "1.0.0", Requires: []string{"inventory"}},
		domain.Manifest{ID: "loyalty", Version: "0.1.0", Requires: []string{"sales"}},
	)...)
	require.NoError(t, err)
	return c
}

var all = domain.Entitlements{"inventory": true, "sales": true, "loyalty": true}

func TestCatalogRejectsInvalidManifests(t *testing.T) {
	base := domain.BaseManifests()
	cases := map[string][]domain.Manifest{
		"malformed id":         {{ID: "Sales", Version: "1"}},
		"one-letter id":        {{ID: "s", Version: "1"}},
		"missing version":      {{ID: "sales"}},
		"duplicate id":         {{ID: "sales", Version: "1"}, {ID: "sales", Version: "2"}},
		"unknown dependency":   {{ID: "sales", Version: "1", Requires: []string{"ghost"}}},
		"self dependency":      {{ID: "sales", Version: "1", Requires: []string{"sales"}}},
		"cycle":                {{ID: "aa", Version: "1", Requires: []string{"bb"}}, {ID: "bb", Version: "1", Requires: []string{"aa"}}},
		"bad permission code":  {{ID: "sales", Version: "1", Permissions: []string{"Sales"}}},
		"optional marked base": {{ID: "sales", Version: "1", Base: true}},
	}
	for name, ms := range cases {
		_, err := domain.NewCatalog(append(append([]domain.Manifest{}, base...), ms...)...)
		assert.ErrorIs(t, err, domain.ErrInvalidManifest, name)
	}
	unmarked := append([]domain.Manifest{}, base...)
	unmarked[0].Base = false
	_, err := domain.NewCatalog(unmarked...)
	assert.ErrorIs(t, err, domain.ErrInvalidManifest, "a base module must be marked base")
	onOptional := append([]domain.Manifest{{ID: "sales", Version: "1"}}, base...)
	onOptional[1].Requires = []string{"sales"}
	_, err = domain.NewCatalog(onOptional...)
	assert.ErrorIs(t, err, domain.ErrInvalidManifest, "a base module cannot require an optional one")
	_, err = domain.NewCatalog(domain.Manifest{ID: "sales", Version: "1"})
	assert.ErrorIs(t, err, domain.ErrInvalidManifest, "every base module must be declared")
}

func TestActivateRequiresLicenseAndActiveDependencies(t *testing.T) {
	c := catalog(t)
	st := domain.State{}

	_, err := c.Activate(st, "ghost", all)
	assert.ErrorIs(t, err, domain.ErrUnknownModule)
	_, err = c.Activate(st, "inventory", domain.Entitlements{"sales": true})
	assert.ErrorIs(t, err, domain.ErrNotLicensed)
	_, err = c.Activate(st, "sales", all)
	assert.ErrorIs(t, err, domain.ErrDependencyInactive, "inventory is not active yet")

	st, err = c.Activate(st, "inventory", all)
	require.NoError(t, err)
	st, err = c.Activate(st, "sales", all)
	require.NoError(t, err)
	assert.True(t, st.IsActive("sales"))

	again, err := c.Activate(st, "sales", all)
	require.NoError(t, err, "activating an active module is a no-op")
	assert.Equal(t, st, again)

	// Base modules are always active and need no entitlement.
	assert.True(t, domain.State{}.IsActive("identity"))
	_, err = c.Activate(domain.State{}, "identity", nil)
	assert.NoError(t, err)
}

func TestActivateDoesNotMutateInput(t *testing.T) {
	c := catalog(t)
	st := domain.State{}
	_, err := c.Activate(st, "inventory", all)
	require.NoError(t, err)
	assert.False(t, st.IsActive("inventory"))
}

func TestDeactivateProtectsBaseAndDependencies(t *testing.T) {
	c := catalog(t)
	st := domain.State{"inventory": true, "sales": true}

	_, err := c.Deactivate(st, "permissions")
	assert.ErrorIs(t, err, domain.ErrBaseModule)
	_, err = c.Deactivate(st, "inventory")
	assert.ErrorIs(t, err, domain.ErrDependencyInUse, "sales needs inventory")
	_, err = c.Deactivate(st, "ghost")
	assert.ErrorIs(t, err, domain.ErrUnknownModule)

	st, err = c.Deactivate(st, "sales")
	require.NoError(t, err)
	assert.False(t, st.IsActive("sales"))
	st, err = c.Deactivate(st, "inventory")
	require.NoError(t, err, "no active dependent remains")
	assert.False(t, st.IsActive("inventory"))
	_, err = c.Deactivate(st, "inventory")
	assert.NoError(t, err, "deactivating an inactive module is a no-op")
}

func TestRequireActiveBlocksNewOperations(t *testing.T) {
	c := catalog(t)
	assert.ErrorIs(t, c.RequireActive(domain.State{}, "sales"), domain.ErrModuleInactive)
	assert.NoError(t, c.RequireActive(domain.State{"inventory": true}, "inventory"))
	assert.NoError(t, c.RequireActive(domain.State{}, "audit"), "base modules are always active")
	assert.ErrorIs(t, c.RequireActive(domain.State{"ghost": true}, "ghost"), domain.ErrUnknownModule)
}
