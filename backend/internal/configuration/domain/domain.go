// Package domain holds typed configuration by scope (ADR-027). Modules
// declare keys with a type, default, validation and the levels where they
// may be set; the most specific allowed level wins. Security and fiscal
// keys (Sensitive) are only set at administrative levels. Every change is a
// new version; history is never edited. It has no I/O.
package domain

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/gamijoam/Balanzapro3.0/backend/internal/platform/qty"
)

var (
	ErrInvalidArgument = errors.New("configuration: invalid argument")
	ErrInvalidKey      = errors.New("configuration: invalid key")
	ErrUnknownKey      = errors.New("configuration: unknown key")
	ErrInvalidValue    = errors.New("configuration: invalid value")
	ErrLevelNotAllowed = errors.New("configuration: level not allowed")
)

// PermissionManage is required to change configuration (ADR-027 §5).
const PermissionManage = "configuration.manage"

// keyName mirrors the config_entries.key check (migration 00006).
var keyName = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)

// MaxKeyLen mirrors the config_entries.key length check.
const MaxKeyLen = 128

// Level of a configuration scope, from general to specific.
type Level string

const (
	LevelDefault      Level = "default" // only as a Resolved source
	LevelGlobal       Level = "global"
	LevelOrganization Level = "organization"
	LevelCompany      Level = "company"
	LevelBranch       Level = "branch"
	LevelTerminal     Level = "terminal"
	LevelUser         Level = "user"
)

// order lists the settable levels from general to specific.
var order = []Level{LevelGlobal, LevelOrganization, LevelCompany, LevelBranch, LevelTerminal, LevelUser}

// administrative levels are the only ones sensitive keys may use.
func administrative(l Level) bool {
	return l == LevelGlobal || l == LevelOrganization || l == LevelCompany
}

// Type of a key's value. Values are bool, int64, qty.Q or string; floats
// are never used.
type Type string

const (
	TypeBool    Type = "bool"
	TypeInteger Type = "integer"
	TypeDecimal Type = "decimal"
	TypeString  Type = "string"
	TypeEnum    Type = "enum"
)

// Key is a typed setting declared by a module.
type Key struct {
	Name      string
	Type      Type
	Default   any
	Levels    []Level
	Sensitive bool     // security or fiscal: administrative levels only
	Min, Max  *qty.Q   // integer and decimal bounds, inclusive
	Options   []string // enum values
	MaxLen    int      // string length in characters; 0 means 1024
}

// Scope is where a value is set; the global scope has no ID.
type Scope struct {
	Level Level
	ID    uuid.UUID
}

// Chain is the context a value is resolved for; unknown parts are uuid.Nil.
type Chain struct {
	OrganizationID uuid.UUID
	CompanyID      uuid.UUID
	BranchID       uuid.UUID
	TerminalID     uuid.UUID
	UserID         uuid.UUID
}

// Scopes returns the chain's scopes from general to specific.
func (c Chain) Scopes() []Scope {
	out := []Scope{{Level: LevelGlobal}}
	for _, s := range []Scope{{LevelOrganization, c.OrganizationID}, {LevelCompany, c.CompanyID},
		{LevelBranch, c.BranchID}, {LevelTerminal, c.TerminalID}, {LevelUser, c.UserID}} {
		if s.ID != uuid.Nil {
			out = append(out, s)
		}
	}
	return out
}

// Entry is one immutable version of a value at a scope.
type Entry struct {
	Key         string
	Scope       Scope
	Version     int
	Value       any
	ActorUserID uuid.UUID
	Reason      string
	CreatedAt   time.Time
}

// Resolved is the effective value, where it came from and its version, so
// operations can record what they applied.
type Resolved struct {
	Value   any
	Source  Level
	Version int
}

// Registry is the validated set of keys.
type Registry struct{ keys map[string]Key }

func invalidKey(k Key, format string, a ...any) error {
	return fmt.Errorf("%w: %s: "+format, append([]any{ErrInvalidKey, k.Name}, a...)...)
}

// NewRegistry validates each key: name, type, levels, bounds, options and
// that the default is itself valid.
func NewRegistry(ks ...Key) (Registry, error) {
	r := Registry{keys: map[string]Key{}}
	for _, k := range ks {
		if !keyName.MatchString(k.Name) || len(k.Name) > MaxKeyLen {
			return Registry{}, invalidKey(k, "malformed name")
		}
		if _, dup := r.keys[k.Name]; dup {
			return Registry{}, invalidKey(k, "duplicate")
		}
		if len(k.Levels) == 0 {
			return Registry{}, invalidKey(k, "no levels")
		}
		for _, l := range k.Levels {
			if !slices.Contains(order, l) {
				return Registry{}, invalidKey(k, "unknown level %q", l)
			}
			if k.Sensitive && !administrative(l) {
				return Registry{}, invalidKey(k, "security or fiscal keys cannot be set at %s", l)
			}
		}
		switch k.Type {
		case TypeBool, TypeDecimal, TypeString:
		case TypeInteger:
			for _, b := range []*qty.Q{k.Min, k.Max} {
				if b != nil && b.Scale() != 0 {
					return Registry{}, invalidKey(k, "integer bounds must be whole")
				}
			}
		case TypeEnum:
			if len(k.Options) == 0 {
				return Registry{}, invalidKey(k, "enum without options")
			}
		default:
			return Registry{}, invalidKey(k, "unknown type %q", k.Type)
		}
		if _, err := check(k, k.Default); err != nil {
			return Registry{}, invalidKey(k, "default: %v", err)
		}
		r.keys[k.Name] = k
	}
	return r, nil
}

func (r Registry) key(name string) (Key, error) {
	k, ok := r.keys[name]
	if !ok {
		return Key{}, fmt.Errorf("%w: %q", ErrUnknownKey, name)
	}
	return k, nil
}

func inRange(k Key, v qty.Q) bool {
	return (k.Min == nil || v.Cmp(*k.Min) >= 0) && (k.Max == nil || v.Cmp(*k.Max) <= 0)
}

func check(k Key, v any) (any, error) {
	bad := fmt.Errorf("%w: %s expects %s", ErrInvalidValue, k.Name, k.Type)
	switch k.Type {
	case TypeBool:
		if b, ok := v.(bool); ok {
			return b, nil
		}
	case TypeInteger:
		if n, ok := v.(int64); ok {
			if !inRange(k, qty.FromInt(n)) {
				return nil, fmt.Errorf("%w: %s out of range", ErrInvalidValue, k.Name)
			}
			return n, nil
		}
	case TypeDecimal:
		if d, ok := v.(qty.Q); ok {
			if !d.Fits() || !inRange(k, d) {
				return nil, fmt.Errorf("%w: %s out of range", ErrInvalidValue, k.Name)
			}
			return d, nil
		}
	case TypeString:
		max := k.MaxLen
		if max <= 0 {
			max = 1024
		}
		if s, ok := v.(string); ok && utf8.ValidString(s) && utf8.RuneCountInString(s) <= max {
			return s, nil
		}
	case TypeEnum:
		if s, ok := v.(string); ok && slices.Contains(k.Options, s) {
			return s, nil
		}
	}
	return nil, bad
}

// Validate checks a value against the key's type and validation.
func (r Registry) Validate(key string, v any) (any, error) {
	k, err := r.key(key)
	if err != nil {
		return nil, err
	}
	return check(k, v)
}

// NewChange builds the next version (current+1) of key at scope. The caller
// checks the actor's permission for the scope.
func (r Registry) NewChange(key string, sc Scope, v any, actor uuid.UUID, reason string,
	current int, at time.Time) (Entry, error) {
	k, err := r.key(key)
	if err != nil {
		return Entry{}, err
	}
	switch {
	case strings.TrimSpace(reason) == "" || actor == uuid.Nil || current < 0:
		return Entry{}, fmt.Errorf("%w: actor and reason are required", ErrInvalidArgument)
	case (sc.Level == LevelGlobal) != (sc.ID == uuid.Nil):
		return Entry{}, fmt.Errorf("%w: only the global scope has no id", ErrInvalidArgument)
	case !slices.Contains(k.Levels, sc.Level):
		return Entry{}, fmt.Errorf("%w: %s at %s", ErrLevelNotAllowed, key, sc.Level)
	}
	val, err := check(k, v)
	if err != nil {
		return Entry{}, err
	}
	return Entry{Key: key, Scope: sc, Version: current + 1, Value: val, ActorUserID: actor,
		Reason: strings.TrimSpace(reason), CreatedAt: at.UTC()}, nil
}

// Resolve returns the latest version at the most specific level of the
// chain the key allows, or the default. Entries for other scopes or
// forbidden levels are ignored.
func (r Registry) Resolve(key string, c Chain, entries []Entry) (Resolved, error) {
	k, err := r.key(key)
	if err != nil {
		return Resolved{}, err
	}
	scopes := c.Scopes()
	for i := len(scopes) - 1; i >= 0; i-- {
		sc := scopes[i]
		if !slices.Contains(k.Levels, sc.Level) {
			continue
		}
		var best *Entry
		for j := range entries {
			e := &entries[j]
			if e.Key == key && e.Scope == sc && (best == nil || e.Version > best.Version) {
				best = e
			}
		}
		if best != nil {
			v, err := check(k, best.Value)
			if err != nil {
				return Resolved{}, err
			}
			return Resolved{Value: v, Source: sc.Level, Version: best.Version}, nil
		}
	}
	return Resolved{Value: k.Default, Source: LevelDefault}, nil
}
