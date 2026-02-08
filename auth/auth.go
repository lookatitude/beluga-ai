// Package auth provides capability-based authorization for the Beluga AI
// framework. It implements RBAC, ABAC, and composite policy patterns with a
// default-deny security model. Every authorization check is explicit — if no
// policy grants access, the request is denied.
//
// Policies are registered via the standard Registry pattern and composed using
// CompositePolicy with configurable combination modes (allow-if-any,
// allow-if-all, deny-if-any).
//
// Usage:
//
//	rbac := auth.NewRBACPolicy("main")
//	rbac.AddRole(auth.Role{Name: "admin", Permissions: []auth.Permission{auth.PermToolExec}})
//	rbac.AssignRole("alice", "admin")
//	allowed, err := rbac.Authorize(ctx, "alice", auth.PermToolExec, "calculator")
package auth

import (
	"context"
	"fmt"
	"sort"
	"sync"
)

// Policy determines whether a subject is authorized to perform a given
// permission on a resource. Implementations must be safe for concurrent use.
type Policy interface {
	// Name returns a unique identifier for this policy.
	Name() string

	// Authorize checks whether subject is allowed to perform permission on
	// resource. Returns (false, nil) for a clean deny. Returns (false, err)
	// when the decision cannot be made due to an error.
	Authorize(ctx context.Context, subject string, permission Permission, resource string) (bool, error)
}

// Permission represents an action that can be authorized.
type Permission string

const (
	// PermToolExec authorizes executing tools.
	PermToolExec Permission = "tool:execute"
	// PermMemoryRead authorizes reading from memory stores.
	PermMemoryRead Permission = "memory:read"
	// PermMemoryWrite authorizes writing to memory stores.
	PermMemoryWrite Permission = "memory:write"
	// PermAgentDelegate authorizes delegating work to other agents.
	PermAgentDelegate Permission = "agent:delegate"
	// PermExternalAPI authorizes calling external APIs.
	PermExternalAPI Permission = "api:external"
)

// Capability represents a fine-grained system capability.
type Capability string

const (
	// CapFileRead grants read access to files.
	CapFileRead Capability = "file:read"
	// CapFileWrite grants write access to files.
	CapFileWrite Capability = "file:write"
	// CapCodeExec grants code execution ability.
	CapCodeExec Capability = "code:exec"
	// CapNetworkAccess grants network access.
	CapNetworkAccess Capability = "network:access"
)

// RiskLevel classifies the risk of an operation for confidence-based routing.
type RiskLevel string

const (
	// RiskReadOnly indicates a read-only operation with minimal risk.
	RiskReadOnly RiskLevel = "read_only"
	// RiskDataModification indicates an operation that modifies data.
	RiskDataModification RiskLevel = "data_modification"
	// RiskIrreversible indicates an operation that cannot be undone.
	RiskIrreversible RiskLevel = "irreversible"
)

// Config carries arbitrary configuration for policy factories.
type Config struct {
	// Extra holds provider-specific configuration.
	Extra map[string]any
}

// Factory creates a Policy from a configuration. Factories are stored in the
// package-level registry and invoked by New.
type Factory func(cfg Config) (Policy, error)

// registry holds the named policy factories.
var (
	registryMu sync.RWMutex
	registry   = make(map[string]Factory)
)

// Register adds a named policy factory to the global registry. It is safe to
// call from init functions. Register panics if name is empty or already
// registered.
func Register(name string, f Factory) {
	if name == "" {
		panic("auth: Register called with empty name")
	}
	if f == nil {
		panic("auth: Register called with nil factory for " + name)
	}

	registryMu.Lock()
	defer registryMu.Unlock()

	if _, dup := registry[name]; dup {
		panic("auth: Register called twice for " + name)
	}
	registry[name] = f
}

// New creates a Policy by looking up the named factory in the registry and
// invoking it with cfg. Returns an error if the name is not registered.
func New(name string, cfg Config) (Policy, error) {
	registryMu.RLock()
	f, ok := registry[name]
	registryMu.RUnlock()

	if !ok {
		return nil, fmt.Errorf("auth: unknown policy %q", name)
	}
	return f(cfg)
}

// List returns the sorted names of all registered policy factories.
func List() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
