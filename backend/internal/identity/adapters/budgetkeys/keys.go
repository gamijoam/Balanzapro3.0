// Package budgetkeys derives opaque budget identifiers on the server.
// It does not determine whether a source address or identity is trustworthy.
package budgetkeys

import (
	"crypto/hmac"
	"crypto/sha256"
	"net/netip"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/application"
	"github.com/gamijoam/Balanzapro3.0/backend/internal/identity/domain"
	"github.com/google/uuid"
)

// Deriver owns a copy of a dedicated server secret. Provision a random
// 32-byte secret outside the database; use the same one across replicas.
// Rotating it changes all budget keys and requires a coordinated policy.
type Deriver struct{ secret [32]byte }

func New(secret []byte) (*Deriver, error) {
	if len(secret) != 32 {
		return nil, domain.ErrInvalidArgument
	}
	d := &Deriver{}
	copy(d.secret[:], secret)
	if d.secret == ([32]byte{}) {
		return nil, domain.ErrInvalidArgument
	}
	return d, nil
}

func (d *Deriver) Identity(org, user uuid.UUID, kind string) (application.AttemptKey, error) {
	if user == uuid.Nil {
		return application.AttemptKey{}, domain.ErrInvalidArgument
	}
	return d.derive(org, kind, 1, user[:])
}

// Origin takes an already trusted network address, not a client header.
// IPv4-mapped addresses normalize to IPv4. This primitive uses exact IPs;
// subnet aggregation and proxy trust are separate policy decisions.
func (d *Deriver) Origin(org uuid.UUID, addr netip.Addr, kind string) (application.AttemptKey, error) {
	addr = addr.Unmap()
	if !addr.IsValid() || addr.Zone() != "" || addr.IsUnspecified() || addr.IsMulticast() {
		return application.AttemptKey{}, domain.ErrInvalidArgument
	}
	return d.derive(org, kind, 2, addr.AsSlice())
}

func (d *Deriver) Organization(org uuid.UUID, kind string) (application.AttemptKey, error) {
	return d.derive(org, kind, 3, nil)
}

func (d *Deriver) derive(org uuid.UUID, kind string, scope byte, subject []byte) (application.AttemptKey, error) {
	if d == nil || d.secret == ([32]byte{}) || org == uuid.Nil {
		return application.AttemptKey{}, domain.ErrInvalidArgument
	}
	var k byte
	switch kind {
	case "password":
		k = 1
	case "pin":
		k = 2
	case "totp":
		k = 3
	default:
		return application.AttemptKey{}, domain.ErrInvalidArgument
	}
	mac := hmac.New(sha256.New, d.secret[:])
	mac.Write([]byte("balanzapro/identity/attempt-budget/v1"))
	mac.Write(org[:])
	mac.Write([]byte{k, scope, byte(len(subject))})
	mac.Write(subject)
	var digest [32]byte
	copy(digest[:], mac.Sum(nil))
	return application.AttemptKey{OrganizationID: org, Kind: kind, Digest: digest}, nil
}
