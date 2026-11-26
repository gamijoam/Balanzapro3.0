package domain_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/configuration/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

// ADR-027 acceptance at the domain level. Synthetic keys only.

var at = time.Date(2026, 11, 26, 12, 0, 0, 0, time.UTC)

func q(s string) *qty.Q { v := qty.MustParse(s); return &v }

var (
	allLevels = []domain.Level{domain.LevelGlobal, domain.LevelOrganization, domain.LevelCompany,
		domain.LevelBranch, domain.LevelTerminal, domain.LevelUser}
	pageSize = domain.Key{Name: "ui.page_size", Type: domain.TypeInteger, Default: int64(20),
		Levels: allLevels, Min: q("5"), Max: q("200")}
	maxDiscount = domain.Key{Name: "sales.max_discount", Type: domain.TypeDecimal, Default: qty.MustParse("0"),
		Levels: []domain.Level{domain.LevelCompany, domain.LevelBranch}, Min: q("0"), Max: q("100")}
	pinLength = domain.Key{Name: "security.pin_length", Type: domain.TypeInteger, Default: int64(6), Sensitive: true,
		Levels: []domain.Level{domain.LevelGlobal, domain.LevelOrganization, domain.LevelCompany}, Min: q("4"), Max: q("12")}
	theme = domain.Key{Name: "ui.theme", Type: domain.TypeEnum, Default: "light", Options: []string{"light", "dark"},
		Levels: allLevels}
)

func registry(t *testing.T) domain.Registry {
	t.Helper()
	r, err := domain.NewRegistry(pageSize, maxDiscount, pinLength, theme)
	require.NoError(t, err)
	return r
}

func TestRegistryRejectsInvalidKeys(t *testing.T) {
	bad := map[string]domain.Key{
		"malformed name":           {Name: "PageSize", Type: domain.TypeString, Default: "x", Levels: allLevels},
		"no levels":                {Name: "ui.x", Type: domain.TypeString, Default: "x"},
		"unknown level":            {Name: "ui.x", Type: domain.TypeString, Default: "x", Levels: []domain.Level{"warehouse"}},
		"invalid default":          {Name: "ui.x", Type: domain.TypeInteger, Default: "20", Levels: allLevels},
		"default out of range":     {Name: "ui.x", Type: domain.TypeInteger, Default: int64(1), Levels: allLevels, Min: q("5")},
		"enum without options":     {Name: "ui.x", Type: domain.TypeEnum, Default: "a", Levels: allLevels},
		"sensitive at terminal":    {Name: "security.x", Type: domain.TypeBool, Default: true, Sensitive: true, Levels: []domain.Level{domain.LevelCompany, domain.LevelTerminal}},
		"sensitive at user":        {Name: "fiscal.x", Type: domain.TypeBool, Default: true, Sensitive: true, Levels: []domain.Level{domain.LevelUser}},
		"unknown type":             {Name: "ui.x", Type: "float", Default: "1", Levels: allLevels},
		"fractional integer bound": {Name: "ui.x", Type: domain.TypeInteger, Default: int64(5), Levels: allLevels, Min: q("1.5")},
	}
	for name, k := range bad {
		_, err := domain.NewRegistry(k)
		assert.ErrorIs(t, err, domain.ErrInvalidKey, name)
	}
	_, err := domain.NewRegistry(pageSize, pageSize)
	assert.ErrorIs(t, err, domain.ErrInvalidKey, "duplicate key")
}

func TestValidateChecksTypeAndRange(t *testing.T) {
	r := registry(t)
	ok := map[string]any{"ui.page_size": int64(50), "sales.max_discount": qty.MustParse("12.5"), "ui.theme": "dark"}
	for k, v := range ok {
		_, err := r.Validate(k, v)
		assert.NoError(t, err, k)
	}
	bad := []struct {
		key string
		v   any
	}{
		{"ui.page_size", "50"}, {"ui.page_size", int64(4)}, {"ui.page_size", int64(201)},
		{"sales.max_discount", qty.MustParse("100.000001")}, {"sales.max_discount", int64(5)},
		{"ui.theme", "blue"}, {"ui.theme", nil},
	}
	for _, c := range bad {
		_, err := r.Validate(c.key, c.v)
		assert.ErrorIs(t, err, domain.ErrInvalidValue, "%s=%v", c.key, c.v)
	}
	_, err := r.Validate("ghost.key", int64(1))
	assert.ErrorIs(t, err, domain.ErrUnknownKey)
}

func TestNewChangeEnforcesLevelsAndVersions(t *testing.T) {
	r := registry(t)
	actor, company, terminal := uuid.New(), uuid.New(), uuid.New()

	e, err := r.NewChange("security.pin_length", domain.Scope{Level: domain.LevelCompany, ID: company},
		int64(8), actor, "política", 0, at)
	require.NoError(t, err)
	assert.Equal(t, 1, e.Version)
	assert.Equal(t, int64(8), e.Value)

	_, err = r.NewChange("security.pin_length", domain.Scope{Level: domain.LevelTerminal, ID: terminal},
		int64(4), actor, "preferencia", 0, at)
	assert.ErrorIs(t, err, domain.ErrLevelNotAllowed, "security keys are never set per terminal")
	_, err = r.NewChange("sales.max_discount", domain.Scope{Level: domain.LevelUser, ID: actor},
		qty.MustParse("5"), actor, "x", 0, at)
	assert.ErrorIs(t, err, domain.ErrLevelNotAllowed, "a level the key does not allow")
	_, err = r.NewChange("ui.page_size", domain.Scope{Level: domain.LevelCompany, ID: company}, int64(10), actor, " ", 0, at)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument, "reason is required")
	_, err = r.NewChange("ui.page_size", domain.Scope{Level: domain.LevelCompany}, int64(10), actor, "x", 0, at)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument, "non-global scopes need an id")
	_, err = r.NewChange("ui.page_size", domain.Scope{Level: domain.LevelGlobal, ID: company}, int64(10), actor, "x", 0, at)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument, "the global scope has no id")

	next, err := r.NewChange("ui.page_size", domain.Scope{Level: domain.LevelCompany, ID: company}, int64(10), actor, "x", 3, at)
	require.NoError(t, err)
	assert.Equal(t, 4, next.Version, "a change is a new version after the current one")
}

func TestResolvePicksMostSpecificAllowedLevel(t *testing.T) {
	r := registry(t)
	org, comp, otherComp, branch, term, user := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	chain := domain.Chain{OrganizationID: org, CompanyID: comp, BranchID: branch, TerminalID: term, UserID: user}
	entry := func(key string, lvl domain.Level, id uuid.UUID, v any, ver int) domain.Entry {
		return domain.Entry{Key: key, Scope: domain.Scope{Level: lvl, ID: id}, Value: v, Version: ver}
	}

	got, err := r.Resolve("ui.page_size", chain, nil)
	require.NoError(t, err)
	assert.Equal(t, domain.Resolved{Value: int64(20), Source: domain.LevelDefault}, got, "default when nothing is set")

	entries := []domain.Entry{
		entry("ui.page_size", domain.LevelGlobal, uuid.Nil, int64(30), 1),
		entry("ui.page_size", domain.LevelCompany, comp, int64(40), 1),
		entry("ui.page_size", domain.LevelCompany, comp, int64(45), 2),
		entry("ui.page_size", domain.LevelCompany, otherComp, int64(99), 7),
		entry("ui.page_size", domain.LevelTerminal, uuid.New(), int64(77), 1),
	}
	got, err = r.Resolve("ui.page_size", chain, entries)
	require.NoError(t, err)
	assert.Equal(t, domain.Resolved{Value: int64(45), Source: domain.LevelCompany, Version: 2}, got,
		"latest version of the most specific level in the chain; other companies and terminals are ignored (SEC-01)")

	entries = append(entries, entry("ui.page_size", domain.LevelUser, user, int64(10), 1))
	got, _ = r.Resolve("ui.page_size", chain, entries)
	assert.Equal(t, domain.LevelUser, got.Source)

	// A stored row at a level the key forbids never wins (defense in depth).
	pin := []domain.Entry{
		entry("security.pin_length", domain.LevelOrganization, org, int64(8), 1),
		entry("security.pin_length", domain.LevelUser, user, int64(4), 1),
	}
	got, err = r.Resolve("security.pin_length", chain, pin)
	require.NoError(t, err)
	assert.Equal(t, domain.Resolved{Value: int64(8), Source: domain.LevelOrganization, Version: 1}, got)

	_, err = r.Resolve("ghost.key", chain, nil)
	assert.ErrorIs(t, err, domain.ErrUnknownKey)
}
