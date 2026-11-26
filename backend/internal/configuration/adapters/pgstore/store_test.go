package pgstore_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/configuration/adapters/pgstore"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/configuration/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/configuration/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/organization"
	perm "github.com/gamijoam/Balanzapro3.0/backend/internal/permissions/domain"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/pgtest"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

type allow struct{}

func (allow) Authorize(context.Context, perm.Request) error { return nil }

func TestConfigurationPersistenceAndIsolation(t *testing.T) {
	ctx := context.Background()
	pool := pgtest.New(t)
	o := organization.NewStore(pool)
	org, err := o.CreateOrganization(ctx, "A")
	require.NoError(t, err)
	other, err := o.CreateOrganization(ctx, "B")
	require.NoError(t, err)
	co, err := o.CreateCompany(ctx, org, "A1")
	require.NoError(t, err)
	foreign, err := o.CreateCompany(ctx, other, "B1")
	require.NoError(t, err)
	branch, err := o.CreateBranch(ctx, co, "Sucursal")
	require.NoError(t, err)
	actor := uuid.New()
	_, err = pool.Exec(ctx, "INSERT INTO identity_users(id,organization_id,login,display_name) VALUES($1,$2,'synthetic','Synthetic')", actor, org)
	require.NoError(t, err)
	levels := []domain.Level{domain.LevelOrganization, domain.LevelCompany, domain.LevelBranch}
	reg, err := domain.NewRegistry(
		domain.Key{Name: "ui.precision", Type: domain.TypeDecimal, Default: qty.MustParse("0"), Levels: levels},
		domain.Key{Name: "ui.count", Type: domain.TypeInteger, Default: int64(0), Levels: levels},
	)
	require.NoError(t, err)
	store := pgstore.New(pool, reg)
	svc := application.NewService(reg, store, allow{}, time.Now)
	cmd := application.SetCommand{OrganizationID: org, ActorUserID: actor, Key: "ui.precision", Scope: domain.Scope{Level: domain.LevelOrganization, ID: org}, Value: qty.MustParse("1.123456"), Reason: "synthetic", CorrelationID: uuid.New()}
	require.NoError(t, svc.Set(ctx, cmd))
	cmd.Value = qty.MustParse("2.123456")
	require.NoError(t, svc.Set(ctx, cmd))
	chain := domain.Chain{OrganizationID: org, CompanyID: co, BranchID: branch}
	v, err := svc.Resolve(ctx, cmd.Key, chain)
	require.NoError(t, err)
	assert.Equal(t, "2.123456", v.Value.(qty.Q).String())
	assert.Equal(t, 2, v.Version)
	cmd.CompanyID = co
	cmd.BranchID = branch
	cmd.Scope = domain.Scope{Level: domain.LevelBranch, ID: branch}
	cmd.Value = qty.MustParse("3.123456")
	require.NoError(t, svc.Set(ctx, cmd))
	v, err = svc.Resolve(ctx, cmd.Key, chain)
	require.NoError(t, err)
	assert.Equal(t, "3.123456", v.Value.(qty.Q).String())
	cmd.Key = "ui.count"
	cmd.Value = int64(9007199254740993)
	require.NoError(t, svc.Set(ctx, cmd))
	v, err = svc.Resolve(ctx, cmd.Key, chain)
	require.NoError(t, err)
	assert.Equal(t, int64(9007199254740993), v.Value)
	_, err = svc.Resolve(ctx, cmd.Key, domain.Chain{OrganizationID: org, CompanyID: foreign})
	assert.ErrorIs(t, err, application.ErrNotFound)
	_, err = svc.Resolve(ctx, cmd.Key, domain.Chain{OrganizationID: other, CompanyID: foreign, BranchID: branch})
	assert.ErrorIs(t, err, application.ErrNotFound)
	_, err = svc.Resolve(ctx, cmd.Key, domain.Chain{OrganizationID: org, CompanyID: co, TerminalID: uuid.New()})
	assert.ErrorIs(t, err, application.ErrNotFound)
	var n int
	require.NoError(t, pool.QueryRow(ctx, "SELECT count(*) FROM audit_events WHERE organization_id=$1", org).Scan(&n))
	assert.Equal(t, 4, n)
	sentinel := errors.New("rollback test")
	e := domain.Entry{Key: cmd.Key, Scope: cmd.Scope, Version: 2, Value: int64(1), ActorUserID: actor, Reason: "test", CreatedAt: time.Now()}
	require.ErrorIs(t, store.InTx(ctx, func(tx application.Tx) error {
		if err := tx.InsertEntry(ctx, org, e); err != nil {
			return err
		}
		return sentinel
	}), sentinel)
	v, err = svc.Resolve(ctx, cmd.Key, chain)
	require.NoError(t, err)
	assert.Equal(t, 1, v.Version)
	// A rejected audit insert must roll back the configuration insert.
	require.Error(t, store.InTx(ctx, func(tx application.Tx) error {
		if err := tx.InsertEntry(ctx, org, e); err != nil {
			return err
		}
		return tx.AppendAudit(ctx, application.AuditEvent{OrganizationID: org, CompanyID: co, ActorUserID: actor, CorrelationID: uuid.New()})
	}))
	v, err = svc.Resolve(ctx, cmd.Key, chain)
	require.NoError(t, err)
	assert.Equal(t, 1, v.Version)
	e.ActorUserID = uuid.New()
	assert.ErrorIs(t, store.InTx(ctx, func(tx application.Tx) error { return tx.InsertEntry(ctx, org, e) }), application.ErrNotFound)
	e.ActorUserID = actor
	e.Version = 1
	assert.ErrorIs(t, store.InTx(ctx, func(tx application.Tx) error { return tx.InsertEntry(ctx, org, e) }), application.ErrVersionConflict)
}

func TestStoredValuesRejectNullAndWrongTypes(t *testing.T) {
	levels := []domain.Level{domain.LevelOrganization}
	reg, err := domain.NewRegistry(
		domain.Key{Name: "ui.flag", Type: domain.TypeBool, Default: false, Levels: levels},
		domain.Key{Name: "ui.count", Type: domain.TypeInteger, Default: int64(0), Levels: levels},
		domain.Key{Name: "ui.text", Type: domain.TypeString, Default: "", Levels: levels},
		domain.Key{Name: "ui.choice", Type: domain.TypeEnum, Default: "a", Options: []string{"a", "b"}, Levels: levels},
		domain.Key{Name: "ui.dec", Type: domain.TypeDecimal, Default: qty.MustParse("0"), Levels: levels})
	require.NoError(t, err)
	for _, k := range []string{"ui.flag", "ui.count", "ui.text", "ui.choice", "ui.dec"} {
		_, err = reg.DecodeValue(k, []byte("null"))
		assert.ErrorIs(t, err, domain.ErrInvalidValue)
	}
	for _, pair := range [][2]string{{"ui.flag", "1"}, {"ui.count", "1.5"}, {"ui.count", "9223372036854775808"}, {"ui.text", "true"}, {"ui.choice", "\"c\""}, {"ui.dec", "1.5"}} {
		_, err = reg.DecodeValue(pair[0], []byte(pair[1]))
		assert.ErrorIs(t, err, domain.ErrInvalidValue)
	}
	for _, tc := range []struct {
		key, raw string
		want     any
	}{{"ui.flag", "true", true}, {"ui.count", "9007199254740993", int64(9007199254740993)}, {"ui.text", "\"texto\"", "texto"}, {"ui.choice", "\"b\"", "b"}, {"ui.dec", "\"1.123456\"", qty.MustParse("1.123456")}} {
		got, err := reg.DecodeValue(tc.key, []byte(tc.raw))
		require.NoError(t, err)
		assert.Equal(t, tc.want, got)
	}
}
