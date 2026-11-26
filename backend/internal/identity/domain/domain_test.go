package domain_test

import (
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/domain"
)

var orgA, orgB = uuid.New(), uuid.New()

func TestNormalizeLoginMatchesSchemaRules(t *testing.T) {
	got, err := domain.NormalizeLogin("  Ana.Perez  ")
	require.NoError(t, err)
	assert.Equal(t, "ana.perez", got)
	for _, bad := range []string{"", "   ", "ana perez", "ana\tperez", "ana\x00", strings.Repeat("a", 65)} {
		_, err := domain.NormalizeLogin(bad)
		assert.ErrorIs(t, err, domain.ErrInvalidArgument, "%q", bad)
	}
	_, err = domain.NormalizeLogin(strings.Repeat("a", 64))
	assert.NoError(t, err)
}

func TestNewUser(t *testing.T) {
	u, err := domain.NewUser(uuid.New(), orgA, " Ana ", " Ana Pérez ")
	require.NoError(t, err)
	assert.Equal(t, "ana", u.Login)
	assert.Equal(t, "Ana Pérez", u.DisplayName)
	assert.Equal(t, domain.StatusActive, u.Status)
	assert.Nil(t, u.DisabledAt)

	_, err = domain.NewUser(uuid.Nil, orgA, "ana", "Ana")
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	_, err = domain.NewUser(uuid.New(), uuid.Nil, "ana", "Ana")
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	_, err = domain.NewUser(uuid.New(), orgA, "ana", "  ")
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
}

func TestDisableUser(t *testing.T) {
	u, err := domain.NewUser(uuid.New(), orgA, "ana", "Ana")
	require.NoError(t, err)
	at := time.Date(2026, 11, 26, 10, 0, 0, 0, time.UTC)
	d, err := u.Disable(at)
	require.NoError(t, err)
	assert.Equal(t, domain.StatusDisabled, d.Status)
	require.NotNil(t, d.DisabledAt)
	assert.Equal(t, at, *d.DisabledAt)
	assert.Equal(t, domain.StatusActive, u.Status, "Disable returns a copy")
	_, err = d.Disable(at)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument, "already disabled")
}

func TestMembershipMustStayInTheUsersOrganization(t *testing.T) {
	u, err := domain.NewUser(uuid.New(), orgA, "ana", "Ana")
	require.NoError(t, err)
	company := uuid.New()
	m, err := domain.NewMembership(u, orgA, company)
	require.NoError(t, err)
	assert.Equal(t, domain.Membership{OrganizationID: orgA, UserID: u.ID, CompanyID: company}, m)

	_, err = domain.NewMembership(u, orgB, uuid.New())
	assert.ErrorIs(t, err, domain.ErrForbidden, "SEC-01: company of another organization")
	_, err = domain.NewMembership(u, orgA, uuid.Nil)
	assert.ErrorIs(t, err, domain.ErrInvalidArgument)
	d, _ := u.Disable(time.Now())
	_, err = domain.NewMembership(d, orgA, company)
	assert.ErrorIs(t, err, domain.ErrForbidden, "disabled users get no new memberships")
}

// SEC-01: a principal exists only for an active user who is a member of the
// requested company in its own organization; everything else is denied.
func TestAuthorizeDeniesByDefault(t *testing.T) {
	u, err := domain.NewUser(uuid.New(), orgA, "ana", "Ana")
	require.NoError(t, err)
	c1, c2 := uuid.New(), uuid.New()
	m1, err := domain.NewMembership(u, orgA, c1)
	require.NoError(t, err)

	p, err := domain.Authorize(u, []domain.Membership{m1}, c1)
	require.NoError(t, err)
	assert.Equal(t, domain.Principal{OrganizationID: orgA, UserID: u.ID, CompanyID: c1}, p)

	_, err = domain.Authorize(u, []domain.Membership{m1}, c2)
	assert.ErrorIs(t, err, domain.ErrForbidden, "not a member")
	_, err = domain.Authorize(u, nil, c1)
	assert.ErrorIs(t, err, domain.ErrForbidden, "no memberships")

	foreign := domain.Membership{OrganizationID: orgB, UserID: u.ID, CompanyID: c2}
	_, err = domain.Authorize(u, []domain.Membership{foreign}, c2)
	assert.ErrorIs(t, err, domain.ErrForbidden, "membership from another organization is ignored")
	other := domain.Membership{OrganizationID: orgA, UserID: uuid.New(), CompanyID: c1}
	_, err = domain.Authorize(u, []domain.Membership{other}, c1)
	assert.ErrorIs(t, err, domain.ErrForbidden, "membership of another user is ignored")

	d, _ := u.Disable(time.Now())
	_, err = domain.Authorize(d, []domain.Membership{m1}, c1)
	assert.ErrorIs(t, err, domain.ErrForbidden, "disabled user")
}
