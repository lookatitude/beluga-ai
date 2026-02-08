package guard

import (
	"sort"
	"testing"
)

func TestRegistry_List_ContainsBuiltins(t *testing.T) {
	names := List()

	expected := []string{"content_filter", "pii_redactor", "prompt_injection_detector", "spotlighting"}
	for _, want := range expected {
		found := false
		for _, name := range names {
			if name == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("List() = %v, want to contain %q", names, want)
		}
	}
}

func TestRegistry_List_Sorted(t *testing.T) {
	names := List()
	if !sort.StringsAreSorted(names) {
		t.Errorf("List() = %v, want sorted", names)
	}
}

func TestRegistry_New_ContentFilter(t *testing.T) {
	g, err := New("content_filter", nil)
	if err != nil {
		t.Fatalf("New(content_filter) error = %v", err)
	}
	if g == nil {
		t.Fatal("New(content_filter) returned nil")
	}
	if g.Name() != "content_filter" {
		t.Errorf("Name() = %q, want %q", g.Name(), "content_filter")
	}
}

func TestRegistry_New_Spotlighting(t *testing.T) {
	g, err := New("spotlighting", map[string]any{"delimiter": "---"})
	if err != nil {
		t.Fatalf("New(spotlighting) error = %v", err)
	}
	if g == nil {
		t.Fatal("New(spotlighting) returned nil")
	}
	if g.Name() != "spotlighting" {
		t.Errorf("Name() = %q, want %q", g.Name(), "spotlighting")
	}
}

func TestRegistry_New_SpotlightingDefaultDelimiter(t *testing.T) {
	g, err := New("spotlighting", map[string]any{})
	if err != nil {
		t.Fatalf("New(spotlighting) error = %v", err)
	}
	if g == nil {
		t.Fatal("New(spotlighting) returned nil")
	}
}

func TestRegistry_New_UnknownGuard(t *testing.T) {
	_, err := New("nonexistent_guard_xyz", nil)
	if err == nil {
		t.Fatal("New(nonexistent_guard_xyz) expected error, got nil")
	}
}

func TestRegistry_Register_PanicOnEmptyName(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Register with empty name should panic")
		}
	}()
	Register("", func(cfg map[string]any) (Guard, error) {
		return nil, nil
	})
}

func TestRegistry_Register_PanicOnNilFactory(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Register with nil factory should panic")
		}
	}()
	Register("nil_factory_test", nil)
}

func TestRegistry_Register_PanicOnDuplicate(t *testing.T) {
	// content_filter is already registered via init(), so re-registering should panic.
	defer func() {
		if r := recover(); r == nil {
			t.Error("Register duplicate name should panic")
		}
	}()
	Register("content_filter", func(cfg map[string]any) (Guard, error) {
		return nil, nil
	})
}

func TestRegistry_New_NilConfig(t *testing.T) {
	// content_filter factory ignores cfg, so nil is safe.
	g, err := New("content_filter", nil)
	if err != nil {
		t.Fatalf("New(content_filter, nil) error = %v", err)
	}
	if g == nil {
		t.Fatal("New(content_filter, nil) returned nil")
	}
}
