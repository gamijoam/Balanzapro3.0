package budgetkeys_test

import (
	"net/netip"
	"testing"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/adapters/budgetkeys"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBudgetKeySeparation(t *testing.T) {
	secret := [32]byte{1}
	d, err := budgetkeys.New(secret[:])
	require.NoError(t, err)
	org, user, other := uuid.New(), uuid.New(), uuid.New()
	first, err := d.Identity(org, user, "password")
	require.NoError(t, err)
	again, err := d.Identity(org, user, "password")
	require.NoError(t, err)
	assert.Equal(t, first, again)
	assert.Equal(t, org, first.OrganizationID)
	assert.Equal(t, "password", first.Kind)
	pin, err := d.Identity(org, user, "pin")
	require.NoError(t, err)
	assert.NotEqual(t, first.Digest, pin.Digest)
	foreign, err := d.Identity(other, user, "password")
	require.NoError(t, err)
	assert.NotEqual(t, first.Digest, foreign.Digest)
	second, err := d.Identity(org, other, "password")
	require.NoError(t, err)
	assert.NotEqual(t, first.Digest, second.Digest)
	global, err := d.Organization(org, "password")
	require.NoError(t, err)
	assert.NotEqual(t, first.Digest, global.Digest)
	origin, err := d.Origin(org, netip.MustParseAddr("192.0.2.1"), "password")
	require.NoError(t, err)
	mapped, err := d.Origin(org, netip.MustParseAddr("::ffff:192.0.2.1"), "password")
	require.NoError(t, err)
	assert.Equal(t, origin, mapped)
	ipv6, err := d.Origin(org, netip.MustParseAddr("2001:db8::1"), "password")
	require.NoError(t, err)
	expanded, err := d.Origin(org, netip.MustParseAddr("2001:0db8:0:0:0:0:0:1"), "password")
	require.NoError(t, err)
	assert.Equal(t, ipv6, expanded)
	assert.NotEqual(t, origin.Digest, ipv6.Digest)
	assert.NotEqual(t, first.Digest, origin.Digest)
	secret[0] = 2
	again, err = d.Identity(org, user, "password")
	require.NoError(t, err)
	assert.Equal(t, first, again)
	rotated, err := budgetkeys.New(secret[:])
	require.NoError(t, err)
	different, err := rotated.Identity(org, user, "password")
	require.NoError(t, err)
	assert.NotEqual(t, first.Digest, different.Digest)
}
func TestInvalidBudgetKeys(t *testing.T) {
	_, err := budgetkeys.New(nil)
	require.Error(t, err)
	_, err = budgetkeys.New(make([]byte, 32))
	require.Error(t, err)
	s := [32]byte{1}
	d, err := budgetkeys.New(s[:])
	require.NoError(t, err)
	org, user := uuid.New(), uuid.New()
	_, err = d.Identity(org, uuid.Nil, "password")
	require.Error(t, err)
	_, err = d.Identity(uuid.Nil, user, "password")
	require.Error(t, err)
	_, err = d.Identity(org, user, "unknown")
	require.Error(t, err)
	for _, a := range []netip.Addr{{}, netip.MustParseAddr("0.0.0.0"), netip.MustParseAddr("::"), netip.MustParseAddr("ff02::1"), netip.MustParseAddr("fe80::1%eth0")} {
		_, err = d.Origin(org, a, "password")
		require.Error(t, err)
	}
	_, err = d.Organization(uuid.Nil, "password")
	require.Error(t, err)
}
