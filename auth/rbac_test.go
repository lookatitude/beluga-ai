package auth

import (
	"context"
	"testing"
)

func TestRBACPolicy_Name(t *testing.T) {
	p := NewRBACPolicy("test-rbac")
	if p.Name() != "test-rbac" {
		t.Errorf("expected name 'test-rbac', got %q", p.Name())
	}
}

func TestRBACPolicy_AddRole(t *testing.T) {
	p := NewRBACPolicy("rbac")

	err := p.AddRole(Role{Name: "admin", Permissions: []Permission{PermToolExec, PermMemoryRead}})
	if err != nil {
		t.Fatalf("AddRole failed: %v", err)
	}

	// Duplicate should fail.
	err = p.AddRole(Role{Name: "admin", Permissions: []Permission{PermToolExec}})
	if err == nil {
		t.Fatal("expected error for duplicate role")
	}
}

func TestRBACPolicy_AddRoleEmptyName(t *testing.T) {
	p := NewRBACPolicy("rbac")
	err := p.AddRole(Role{Name: "", Permissions: []Permission{PermToolExec}})
	if err == nil {
		t.Fatal("expected error for empty role name")
	}
}

func TestRBACPolicy_AssignRole(t *testing.T) {
	p := NewRBACPolicy("rbac")
	_ = p.AddRole(Role{Name: "admin", Permissions: []Permission{PermToolExec}})

	err := p.AssignRole("alice", "admin")
	if err != nil {
		t.Fatalf("AssignRole failed: %v", err)
	}

	// Assign nonexistent role.
	err = p.AssignRole("alice", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent role")
	}

	// Duplicate assignment.
	err = p.AssignRole("alice", "admin")
	if err == nil {
		t.Fatal("expected error for duplicate assignment")
	}
}

func TestRBACPolicy_RemoveRole(t *testing.T) {
	p := NewRBACPolicy("rbac")
	_ = p.AddRole(Role{Name: "admin", Permissions: []Permission{PermToolExec}})
	_ = p.AssignRole("alice", "admin")

	err := p.RemoveRole("alice", "admin")
	if err != nil {
		t.Fatalf("RemoveRole failed: %v", err)
	}

	// Remove again should fail.
	err = p.RemoveRole("alice", "admin")
	if err == nil {
		t.Fatal("expected error removing unassigned role")
	}
}

func TestRBACPolicy_AuthorizeAllowed(t *testing.T) {
	ctx := context.Background()
	p := NewRBACPolicy("rbac")
	_ = p.AddRole(Role{Name: "admin", Permissions: []Permission{PermToolExec, PermMemoryRead}})
	_ = p.AssignRole("alice", "admin")

	allowed, err := p.Authorize(ctx, "alice", PermToolExec, "calculator")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("expected alice to be allowed PermToolExec")
	}

	allowed, err = p.Authorize(ctx, "alice", PermMemoryRead, "history")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if !allowed {
		t.Error("expected alice to be allowed PermMemoryRead")
	}
}

func TestRBACPolicy_AuthorizeDenied(t *testing.T) {
	ctx := context.Background()
	p := NewRBACPolicy("rbac")
	_ = p.AddRole(Role{Name: "reader", Permissions: []Permission{PermMemoryRead}})
	_ = p.AssignRole("bob", "reader")

	allowed, err := p.Authorize(ctx, "bob", PermToolExec, "calculator")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected bob to be denied PermToolExec")
	}
}

func TestRBACPolicy_AuthorizeDefaultDeny(t *testing.T) {
	ctx := context.Background()
	p := NewRBACPolicy("rbac")

	// No roles assigned — default deny.
	allowed, err := p.Authorize(ctx, "unknown", PermToolExec, "anything")
	if err != nil {
		t.Fatalf("Authorize error: %v", err)
	}
	if allowed {
		t.Error("expected default deny for unknown subject")
	}
}

func TestRBACPolicy_MultipleRoles(t *testing.T) {
	ctx := context.Background()
	p := NewRBACPolicy("rbac")
	_ = p.AddRole(Role{Name: "reader", Permissions: []Permission{PermMemoryRead}})
	_ = p.AddRole(Role{Name: "writer", Permissions: []Permission{PermMemoryWrite}})
	_ = p.AssignRole("charlie", "reader")
	_ = p.AssignRole("charlie", "writer")

	tests := []struct {
		perm    Permission
		allowed bool
	}{
		{PermMemoryRead, true},
		{PermMemoryWrite, true},
		{PermToolExec, false},
	}

	for _, tt := range tests {
		allowed, err := p.Authorize(ctx, "charlie", tt.perm, "resource")
		if err != nil {
			t.Fatalf("Authorize error for %s: %v", tt.perm, err)
		}
		if allowed != tt.allowed {
			t.Errorf("Authorize(%s) = %v, want %v", tt.perm, allowed, tt.allowed)
		}
	}
}

func TestRBACPolicy_RemoveRoleDeniesAccess(t *testing.T) {
	ctx := context.Background()
	p := NewRBACPolicy("rbac")
	_ = p.AddRole(Role{Name: "admin", Permissions: []Permission{PermToolExec}})
	_ = p.AssignRole("alice", "admin")

	// Verify allowed before removal.
	allowed, _ := p.Authorize(ctx, "alice", PermToolExec, "tool")
	if !allowed {
		t.Fatal("expected allowed before removal")
	}

	_ = p.RemoveRole("alice", "admin")

	// Verify denied after removal.
	allowed, _ = p.Authorize(ctx, "alice", PermToolExec, "tool")
	if allowed {
		t.Error("expected denied after role removal")
	}
}
