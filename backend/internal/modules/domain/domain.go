// Package domain holds module activation rules (ADR-023): modules are
// compiled in and described by manifests; each company activates the
// optional ones it is licensed for. Base modules are always active.
// Deactivation hides functions and blocks new operations but keeps data.
// It has no I/O.
package domain

import (
	"errors"
	"fmt"
	"maps"
	"regexp"
	"strings"
)

var (
	ErrInvalidArgument    = errors.New("modules: invalid argument")
	ErrInvalidManifest    = errors.New("modules: invalid manifest")
	ErrUnknownModule      = errors.New("modules: unknown module")
	ErrNotLicensed        = errors.New("modules: not licensed")
	ErrDependencyInactive = errors.New("modules: dependency inactive")
	ErrDependencyInUse    = errors.New("modules: dependency in use")
	ErrBaseModule         = errors.New("modules: base module cannot be deactivated")
	ErrModuleInactive     = errors.New("modules: module inactive")
)

// PermissionManage is required, at company scope, to change activation.
const PermissionManage = "modules.manage"

var (
	// moduleID mirrors the company_modules.module_id check (migration 00006).
	moduleID       = regexp.MustCompile(`^[a-z][a-z0-9_]{1,62}$`)
	permissionCode = regexp.MustCompile(`^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+$`)
)

// baseIDs are always active (ADR-023 §5).
var baseIDs = []string{"organization", "identity", "permissions", "configuration", "audit"}

// Manifest describes one compiled-in module.
type Manifest struct {
	ID          string
	Version     string
	Base        bool
	Requires    []string // module IDs that must be active first
	Permissions []string // permission codes it declares (ADR-024)
	// Typed configuration keys (ADR-027) join the manifest with the
	// configuration domain.
}

// BaseManifests returns the manifests of the base modules.
func BaseManifests() []Manifest {
	out := make([]Manifest, 0, len(baseIDs))
	for _, id := range baseIDs {
		m := Manifest{ID: id, Version: "1.0.0", Base: true}
		if id == "configuration" {
			m.Permissions = []string{PermissionManage}
		}
		out = append(out, m)
	}
	return out
}

// Entitlements are the module IDs a company's license includes (ADR-147).
type Entitlements map[string]bool

// State is a company's activation of optional modules (true = active).
type State map[string]bool

func isBase(id string) bool {
	for _, b := range baseIDs {
		if b == id {
			return true
		}
	}
	return false
}

// IsActive reports whether id is active; base modules always are.
func (s State) IsActive(id string) bool { return isBase(id) || s[id] }

// Catalog is the validated set of manifests.
type Catalog struct{ byID map[string]Manifest }

func invalid(format string, a ...any) error {
	return fmt.Errorf("%w: "+format, append([]any{ErrInvalidManifest}, a...)...)
}

// NewCatalog validates ids, versions, permission codes and dependencies:
// they must exist, form no cycle, and a base module may only depend on base
// modules. Every base module must be declared, as base.
func NewCatalog(ms ...Manifest) (Catalog, error) {
	c := Catalog{byID: map[string]Manifest{}}
	for _, m := range ms {
		switch {
		case !moduleID.MatchString(m.ID):
			return Catalog{}, invalid("module id %q", m.ID)
		case strings.TrimSpace(m.Version) == "":
			return Catalog{}, invalid("module %s has no version", m.ID)
		case m.Base != isBase(m.ID):
			return Catalog{}, invalid("module %s: only %v are base", m.ID, baseIDs)
		}
		if _, dup := c.byID[m.ID]; dup {
			return Catalog{}, invalid("duplicate module %s", m.ID)
		}
		for _, p := range m.Permissions {
			if !permissionCode.MatchString(p) {
				return Catalog{}, invalid("module %s permission %q", m.ID, p)
			}
		}
		c.byID[m.ID] = m
	}
	for _, id := range baseIDs {
		if _, ok := c.byID[id]; !ok {
			return Catalog{}, invalid("base module %s must be declared", id)
		}
	}
	for _, m := range c.byID {
		for _, d := range m.Requires {
			dep, ok := c.byID[d]
			if !ok {
				return Catalog{}, invalid("module %s requires unknown %s", m.ID, d)
			}
			if m.Base && !dep.Base {
				return Catalog{}, invalid("base module %s cannot require optional %s", m.ID, d)
			}
		}
	}
	return c, c.checkAcyclic()
}

func (c Catalog) checkAcyclic() error {
	const visiting, done = 1, 2
	mark := map[string]int{}
	var visit func(id string) error
	visit = func(id string) error {
		switch mark[id] {
		case visiting:
			return invalid("dependency cycle through %s", id)
		case done:
			return nil
		}
		mark[id] = visiting
		for _, d := range c.byID[id].Requires {
			if err := visit(d); err != nil {
				return err
			}
		}
		mark[id] = done
		return nil
	}
	for id := range c.byID {
		if err := visit(id); err != nil {
			return err
		}
	}
	return nil
}

func (c Catalog) get(id string) (Manifest, error) {
	m, ok := c.byID[id]
	if !ok {
		return Manifest{}, fmt.Errorf("%w: %q", ErrUnknownModule, id)
	}
	return m, nil
}

// Activate returns a new state with id active. The license must include it
// and its dependencies must be active; activating an active module changes
// nothing. The caller checks the actor's permission (ADR-023 §3).
func (c Catalog) Activate(s State, id string, e Entitlements) (State, error) {
	m, err := c.get(id)
	if err != nil {
		return s, err
	}
	if s.IsActive(id) {
		return s, nil
	}
	if !e[id] {
		return s, fmt.Errorf("%w: %s", ErrNotLicensed, id)
	}
	for _, d := range m.Requires {
		if !s.IsActive(d) {
			return s, fmt.Errorf("%w: %s requires %s", ErrDependencyInactive, id, d)
		}
	}
	out := maps.Clone(s)
	if out == nil {
		out = State{}
	}
	out[id] = true
	return out, nil
}

// Deactivate returns a new state with id inactive. Base modules and modules
// an active module requires cannot be deactivated.
func (c Catalog) Deactivate(s State, id string) (State, error) {
	m, err := c.get(id)
	if err != nil {
		return s, err
	}
	if m.Base {
		return s, fmt.Errorf("%w: %s", ErrBaseModule, id)
	}
	if !s.IsActive(id) {
		return s, nil
	}
	for _, other := range c.byID {
		if !s.IsActive(other.ID) {
			continue
		}
		for _, d := range other.Requires {
			if d == id {
				return s, fmt.Errorf("%w: %s requires %s", ErrDependencyInUse, other.ID, id)
			}
		}
	}
	out := maps.Clone(s)
	out[id] = false
	return out, nil
}

// RequireActive guards new operations of a module.
func (c Catalog) RequireActive(s State, id string) error {
	if _, err := c.get(id); err != nil {
		return err
	}
	if !s.IsActive(id) {
		return fmt.Errorf("%w: %s", ErrModuleInactive, id)
	}
	return nil
}
