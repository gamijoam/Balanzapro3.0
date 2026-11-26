// Package domain holds the permission rules (ADR-024) and supervisor
// authorizations (ADR-025). It has no I/O; the Authorizer port of ADR-010
// is built on Decide.
package domain

import (
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

var (
	ErrInvalidArgument      = errors.New("permissions: invalid argument")
	ErrForbidden            = errors.New("permissions: forbidden")
	ErrLimitExceeded        = errors.New("permissions: limit exceeded")
	ErrAuthorizationExpired = errors.New("permissions: authorization expired")
	ErrAuthorizationUsed    = errors.New("permissions: authorization already used")
)

var (
	roleCode       = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
	permissionCode = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
)

// MaxAuthorizationTTL bounds supervisor authorizations (ADR-025: ~2 min,
// configurable within safe limits).
const MaxAuthorizationTTL = 5 * time.Minute

// Permission is a code declared by a module, e.g. "sales.discount".
type Permission string

// Grant gives a permission, optionally capped by an exact limit (nil means
// unlimited).
type Grant struct {
	Permission Permission
	Limit      *qty.Q
}

// Role is a named set of grants of one organization.
type Role struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Code           string
	Grants         map[Permission]*qty.Q
}

// NewRole validates codes, rejects duplicate permissions and negative limits.
func NewRole(id, org uuid.UUID, code string, grants []Grant) (Role, error) {
	if id == uuid.Nil || org == uuid.Nil {
		return Role{}, fmt.Errorf("%w: id and organization are required", ErrInvalidArgument)
	}
	if !roleCode.MatchString(code) {
		return Role{}, fmt.Errorf("%w: role code %q", ErrInvalidArgument, code)
	}
	r := Role{ID: id, OrganizationID: org, Code: code, Grants: map[Permission]*qty.Q{}}
	for _, g := range grants {
		if !permissionCode.MatchString(string(g.Permission)) {
			return Role{}, fmt.Errorf("%w: permission %q", ErrInvalidArgument, g.Permission)
		}
		if _, dup := r.Grants[g.Permission]; dup {
			return Role{}, fmt.Errorf("%w: duplicate permission %q", ErrInvalidArgument, g.Permission)
		}
		if g.Limit != nil && g.Limit.IsNegative() {
			return Role{}, fmt.Errorf("%w: negative limit for %q", ErrInvalidArgument, g.Permission)
		}
		r.Grants[g.Permission] = g.Limit
	}
	return r, nil
}

// ScopeLevel of an assignment within a company (ADR-014).
type ScopeLevel string

const (
	LevelCompany   ScopeLevel = "company"
	LevelBranch    ScopeLevel = "branch"
	LevelWarehouse ScopeLevel = "warehouse"
)

// ScopeRef is where an assignment applies.
type ScopeRef struct {
	Level ScopeLevel
	ID    uuid.UUID
}

// ScopePath is the target of an action: its company and, when it has one,
// branch and warehouse.
type ScopePath struct {
	CompanyID   uuid.UUID
	BranchID    uuid.UUID
	WarehouseID uuid.UUID
}

// covers reports whether an assignment at ref applies to the path:
// assignments inherit downwards only (ADR-024).
func (p ScopePath) covers(ref ScopeRef) bool {
	switch ref.Level {
	case LevelCompany:
		return ref.ID == p.CompanyID
	case LevelBranch:
		return p.BranchID != uuid.Nil && ref.ID == p.BranchID
	case LevelWarehouse:
		return p.WarehouseID != uuid.Nil && ref.ID == p.WarehouseID
	}
	return false
}

// Assignment gives a role to a user in one scope of one company.
type Assignment struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	RoleID         uuid.UUID
	CompanyID      uuid.UUID
	Scope          ScopeRef
}

// Request asks whether a user may perform a permission on a target,
// optionally for an exact amount checked against limits.
type Request struct {
	OrganizationID uuid.UUID
	UserID         uuid.UUID
	Permission     Permission
	Target         ScopePath
	Amount         *qty.Q
}

// Decide denies by default. It allows when an assignment of the same user
// and organization, in the target's company and covering the target, gives
// a role of that organization granting the permission within its limit.
// ErrLimitExceeded means the permission exists but no applicable grant
// covers the amount (a supervisor may authorize it, ADR-025).
func Decide(r Request, assignments []Assignment, roles map[uuid.UUID]Role) error {
	limited := false
	for _, a := range assignments {
		if a.UserID != r.UserID || a.OrganizationID != r.OrganizationID ||
			a.CompanyID != r.Target.CompanyID || r.Target.CompanyID == uuid.Nil || !r.Target.covers(a.Scope) {
			continue
		}
		role, ok := roles[a.RoleID]
		if !ok || role.OrganizationID != r.OrganizationID {
			continue
		}
		limit, granted := role.Grants[r.Permission]
		switch {
		case !granted:
			continue
		case limit == nil:
			return nil
		case r.Amount != nil && r.Amount.Cmp(*limit) <= 0:
			return nil
		default:
			limited = true
		}
	}
	if limited {
		return fmt.Errorf("%w: %s", ErrLimitExceeded, r.Permission)
	}
	return fmt.Errorf("%w: %s", ErrForbidden, r.Permission)
}

// Authorization is a supervisor's approval of one concrete action (ADR-025).
type Authorization struct {
	ID           uuid.UUID
	SupervisorID uuid.UUID
	Request      Request
	EntityRef    string
	TerminalID   uuid.UUID
	SessionID    uuid.UUID
	ExpiresAt    time.Time
	UsedAt       *time.Time
}

// NewAuthorization binds an approval to the request, entity, terminal and
// session; the caller must have checked the supervisor's own permission.
func NewAuthorization(id, supervisor uuid.UUID, r Request, entity string, terminal, session uuid.UUID,
	now time.Time, ttl time.Duration) (Authorization, error) {
	switch {
	case id == uuid.Nil || supervisor == uuid.Nil || terminal == uuid.Nil || session == uuid.Nil || entity == "":
		return Authorization{}, fmt.Errorf("%w: incomplete authorization", ErrInvalidArgument)
	case ttl <= 0 || ttl > MaxAuthorizationTTL:
		return Authorization{}, fmt.Errorf("%w: ttl must be in (0, %s]", ErrInvalidArgument, MaxAuthorizationTTL)
	case supervisor == r.UserID:
		return Authorization{}, fmt.Errorf("%w: a user cannot authorize themselves", ErrForbidden)
	}
	return Authorization{ID: id, SupervisorID: supervisor, Request: r, EntityRef: entity,
		TerminalID: terminal, SessionID: session, ExpiresAt: now.Add(ttl).UTC()}, nil
}

// Redemption is the action that wants to use an authorization.
type Redemption struct {
	Request    Request
	EntityRef  string
	TerminalID uuid.UUID
	SessionID  uuid.UUID
}

func sameAmount(a, b *qty.Q) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return a.Cmp(*b) == 0
}

// Redeem uses the authorization once, for exactly the authorized action,
// before it expires (SEC-02). It returns the used copy.
func (a Authorization) Redeem(r Redemption, now time.Time) (Authorization, error) {
	if a.UsedAt != nil {
		return a, ErrAuthorizationUsed
	}
	if !now.Before(a.ExpiresAt) {
		return a, ErrAuthorizationExpired
	}
	q := a.Request
	if r.Request.OrganizationID != q.OrganizationID || r.Request.UserID != q.UserID ||
		r.Request.Permission != q.Permission || r.Request.Target != q.Target ||
		!sameAmount(r.Request.Amount, q.Amount) || r.EntityRef != a.EntityRef ||
		r.TerminalID != a.TerminalID || r.SessionID != a.SessionID {
		return a, fmt.Errorf("%w: authorization does not match this action", ErrForbidden)
	}
	used := now.UTC()
	a.UsedAt = &used
	return a, nil
}
