package auth

import (
	"testing"
)

func TestPermissionConstants(t *testing.T) {
	perms := []Permission{
		PermToolExec,
		PermMemoryRead,
		PermMemoryWrite,
		PermAgentDelegate,
		PermExternalAPI,
	}
	seen := make(map[Permission]bool)
	for _, p := range perms {
		if p == "" {
			t.Error("permission constant must not be empty")
		}
		if seen[p] {
			t.Errorf("duplicate permission constant: %s", p)
		}
		seen[p] = true
	}
}

func TestCapabilityConstants(t *testing.T) {
	caps := []Capability{
		CapFileRead,
		CapFileWrite,
		CapCodeExec,
		CapNetworkAccess,
	}
	seen := make(map[Capability]bool)
	for _, c := range caps {
		if c == "" {
			t.Error("capability constant must not be empty")
		}
		if seen[c] {
			t.Errorf("duplicate capability constant: %s", c)
		}
		seen[c] = true
	}
}

func TestRiskLevelConstants(t *testing.T) {
	levels := []RiskLevel{
		RiskReadOnly,
		RiskDataModification,
		RiskIrreversible,
	}
	seen := make(map[RiskLevel]bool)
	for _, l := range levels {
		if l == "" {
			t.Error("risk level constant must not be empty")
		}
		if seen[l] {
			t.Errorf("duplicate risk level constant: %s", l)
		}
		seen[l] = true
	}
}

func TestRegisterAndNew(t *testing.T) {
	// Clean up after test.
	origRegistry := registry
	defer func() {
		registry = origRegistry
	}()
	registry = make(map[string]Factory)

	Register("test-policy", func(cfg Config) (Policy, error) {
		return NewRBACPolicy("from-factory"), nil
	})

	p, err := New("test-policy", Config{})
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if p.Name() != "from-factory" {
		t.Errorf("expected name 'from-factory', got %q", p.Name())
	}
}

func TestNewUnknown(t *testing.T) {
	origRegistry := registry
	defer func() {
		registry = origRegistry
	}()
	registry = make(map[string]Factory)

	_, err := New("nonexistent", Config{})
	if err == nil {
		t.Fatal("expected error for unknown policy")
	}
}

func TestRegisterPanicsEmptyName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for empty name")
		}
	}()
	Register("", func(cfg Config) (Policy, error) { return nil, nil })
}

func TestRegisterPanicsNilFactory(t *testing.T) {
	origRegistry := registry
	defer func() {
		registry = origRegistry
	}()
	registry = make(map[string]Factory)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for nil factory")
		}
	}()
	Register("nil-factory", nil)
}

func TestRegisterPanicsDuplicate(t *testing.T) {
	origRegistry := registry
	defer func() {
		registry = origRegistry
	}()
	registry = make(map[string]Factory)

	f := func(cfg Config) (Policy, error) { return nil, nil }
	Register("dup", f)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic for duplicate registration")
		}
	}()
	Register("dup", f)
}

func TestList(t *testing.T) {
	origRegistry := registry
	defer func() {
		registry = origRegistry
	}()
	registry = make(map[string]Factory)

	f := func(cfg Config) (Policy, error) { return nil, nil }
	Register("charlie", f)
	Register("alice", f)
	Register("bob", f)

	names := List()
	if len(names) != 3 {
		t.Fatalf("expected 3 names, got %d", len(names))
	}
	if names[0] != "alice" || names[1] != "bob" || names[2] != "charlie" {
		t.Errorf("expected sorted names, got %v", names)
	}
}
