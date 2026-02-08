package auth

import (
	"context"
	"fmt"
	"sync"
)

// Role groups a set of permissions under a name. Roles are assigned to subjects
// via RBACPolicy.AssignRole.
type Role struct {
	// Name uniquely identifies this role.
	Name string

	// Permissions lists the actions this role grants.
	Permissions []Permission
}

// RBACPolicy implements role-based access control. Subjects are assigned one or
// more roles, and authorization checks whether any assigned role contains the
// requested permission.
//
// RBACPolicy is safe for concurrent use.
type RBACPolicy struct {
	name string

	mu          sync.RWMutex
	roles       map[string]*Role    // roleName -> Role
	assignments map[string][]string // subject -> []roleName
}

// NewRBACPolicy creates a new RBAC policy with the given name.
func NewRBACPolicy(name string) *RBACPolicy {
	return &RBACPolicy{
		name:        name,
		roles:       make(map[string]*Role),
		assignments: make(map[string][]string),
	}
}

// Name returns the policy name.
func (p *RBACPolicy) Name() string { return p.name }

// AddRole registers a role. Returns an error if a role with the same name
// already exists.
func (p *RBACPolicy) AddRole(role Role) error {
	if role.Name == "" {
		return fmt.Errorf("auth/rbac: role name must not be empty")
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.roles[role.Name]; exists {
		return fmt.Errorf("auth/rbac: role %q already exists", role.Name)
	}
	r := role // copy
	p.roles[role.Name] = &r
	return nil
}

// AssignRole assigns a named role to a subject. Returns an error if the role
// does not exist or is already assigned.
func (p *RBACPolicy) AssignRole(subject, roleName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.roles[roleName]; !exists {
		return fmt.Errorf("auth/rbac: role %q does not exist", roleName)
	}

	for _, r := range p.assignments[subject] {
		if r == roleName {
			return fmt.Errorf("auth/rbac: role %q already assigned to %q", roleName, subject)
		}
	}

	p.assignments[subject] = append(p.assignments[subject], roleName)
	return nil
}

// RemoveRole removes a role assignment from a subject. Returns an error if the
// role is not assigned to the subject.
func (p *RBACPolicy) RemoveRole(subject, roleName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	roles := p.assignments[subject]
	for i, r := range roles {
		if r == roleName {
			p.assignments[subject] = append(roles[:i], roles[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("auth/rbac: role %q not assigned to %q", roleName, subject)
}

// Authorize checks whether subject has permission on resource. It iterates
// through all roles assigned to the subject and returns true if any role
// contains the requested permission. Default deny: returns false if no role
// grants the permission.
func (p *RBACPolicy) Authorize(ctx context.Context, subject string, permission Permission, resource string) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	roleNames := p.assignments[subject]
	for _, roleName := range roleNames {
		role, ok := p.roles[roleName]
		if !ok {
			continue
		}
		for _, perm := range role.Permissions {
			if perm == permission {
				return true, nil
			}
		}
	}

	// Default deny.
	return false, nil
}

// Ensure RBACPolicy implements Policy at compile time.
var _ Policy = (*RBACPolicy)(nil)
