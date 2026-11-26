// Package domain holds identity rules (BP-008, ADR-014): users belong to an
// organization and act in a company only through a membership. Authorize
// denies by default (SEC-01). Credentials and sessions (ADR-026) and roles by
// scope (ADR-024, BP-009) live elsewhere. It has no I/O.
package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

var (
	ErrInvalidArgument = errors.New("identity: invalid argument")
	ErrForbidden       = errors.New("identity: forbidden")
	ErrNotFound        = errors.New("identity: not found")
	ErrDuplicate       = errors.New("identity: duplicate")
)

// MaxLoginLen mirrors the identity_users.login check (migration 00005).
const MaxLoginLen = 64

// Status of a user. Users are disabled, never deleted.
type Status string

const (
	StatusActive   Status = "active"
	StatusDisabled Status = "disabled"
)

// User is a person or operator of one organization.
type User struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Login          string
	DisplayName    string
	Status         Status
	DisabledAt     *time.Time
}

// Membership lets a user act in one company of its organization.
type Membership struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	CompanyID      uuid.UUID
}

// Principal is an authorized identity acting in one company.
type Principal struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	CompanyID      uuid.UUID
}

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidArgument}, a...)...)
}

// NormalizeLogin trims, lowercases and rejects whitespace, control
// characters and logins longer than MaxLoginLen.
func NormalizeLogin(s string) (string, error) {
	l := strings.ToLower(strings.TrimSpace(s))
	if l == "" || len(l) > MaxLoginLen {
		return "", invalid("login must have 1..%d characters", MaxLoginLen)
	}
	for _, r := range l {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return "", invalid("login cannot contain spaces or control characters")
		}
	}
	return l, nil
}

// NewUser validates and builds an active user.
func NewUser(id, org uuid.UUID, login, displayName string) (User, error) {
	if id == uuid.Nil || org == uuid.Nil {
		return User{}, invalid("id and organization are required")
	}
	l, err := NormalizeLogin(login)
	if err != nil {
		return User{}, err
	}
	name := strings.TrimSpace(displayName)
	if name == "" {
		return User{}, invalid("display name is required")
	}
	return User{ID: id, OrganizationID: org, Login: l, DisplayName: name, Status: StatusActive}, nil
}

// Disable returns a disabled copy of the user.
func (u User) Disable(at time.Time) (User, error) {
	if u.Status != StatusActive {
		return u, invalid("user %s is not active", u.ID)
	}
	t := at.UTC()
	u.Status, u.DisabledAt = StatusDisabled, &t
	return u, nil
}

// NewMembership attaches an active user to a company. companyOrg is the
// organization the company belongs to; a different one is forbidden (SEC-01).
func NewMembership(u User, companyOrg, company uuid.UUID) (Membership, error) {
	if company == uuid.Nil || companyOrg == uuid.Nil {
		return Membership{}, invalid("company and its organization are required")
	}
	if companyOrg != u.OrganizationID {
		return Membership{}, fmt.Errorf("%w: company belongs to another organization", ErrForbidden)
	}
	if u.Status != StatusActive {
		return Membership{}, fmt.Errorf("%w: user %s is not active", ErrForbidden, u.ID)
	}
	return Membership{OrganizationID: u.OrganizationID, UserID: u.ID, CompanyID: company}, nil
}

// Authorize returns the principal of u acting in company, denying by
// default: the user must be active and hold a membership of its own
// organization for that exact company.
func Authorize(u User, memberships []Membership, company uuid.UUID) (Principal, error) {
	if u.Status != StatusActive {
		return Principal{}, fmt.Errorf("%w: user is not active", ErrForbidden)
	}
	for _, m := range memberships {
		if m.UserID == u.ID && m.OrganizationID == u.OrganizationID && m.CompanyID == company && company != uuid.Nil {
			return Principal{OrganizationID: u.OrganizationID, UserID: u.ID, CompanyID: company}, nil
		}
	}
	return Principal{}, fmt.Errorf("%w: no membership for this company", ErrForbidden)
}
