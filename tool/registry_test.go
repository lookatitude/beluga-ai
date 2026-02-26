package tool

import (
	"testing"
)

func TestRegistry_AddAndGet(t *testing.T) {
	reg := NewRegistry()

	t1 := &mockTool{name: "search", description: "Search"}
	t2 := &mockTool{name: "calc", description: "Calculate"}

	if err := reg.Add(t1); err != nil {
		t.Fatalf("Add(search) error: %v", err)
	}
	if err := reg.Add(t2); err != nil {
		t.Fatalf("Add(calc) error: %v", err)
	}

	got, err := reg.Get("search")
	if err != nil {
		t.Fatalf("Get(search) error: %v", err)
	}
	if got.Name() != "search" {
		t.Errorf("got name %q, want %q", got.Name(), "search")
	}
}

func TestRegistry_Add_Duplicate(t *testing.T) {
	reg := NewRegistry()

	t1 := &mockTool{name: "search", description: "Search v1"}
	t2 := &mockTool{name: "search", description: "Search v2"}

	if err := reg.Add(t1); err != nil {
		t.Fatalf("first Add error: %v", err)
	}
	if reg.Add(t2) == nil {
		t.Fatal("expected error on duplicate Add")
	}
}

func TestRegistry_Get_NotFound(t *testing.T) {
	reg := NewRegistry()
	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent tool")
	}
}

func TestRegistry_List(t *testing.T) {
	reg := NewRegistry()

	_ = reg.Add(&mockTool{name: "zebra"})
	_ = reg.Add(&mockTool{name: "alpha"})
	_ = reg.Add(&mockTool{name: "middle"})

	names := reg.List()
	want := []string{"alpha", "middle", "zebra"}
	if len(names) != len(want) {
		t.Fatalf("List() len = %d, want %d", len(names), len(want))
	}
	for i, name := range names {
		if name != want[i] {
			t.Errorf("List()[%d] = %q, want %q", i, name, want[i])
		}
	}
}

func TestRegistry_List_Empty(t *testing.T) {
	reg := NewRegistry()
	names := reg.List()
	if len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}
}

func TestRegistry_Remove(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Add(&mockTool{name: "search"})

	if err := reg.Remove("search"); err != nil {
		t.Fatalf("Remove error: %v", err)
	}

	_, err := reg.Get("search")
	if err == nil {
		t.Fatal("expected error after removal")
	}

	names := reg.List()
	if len(names) != 0 {
		t.Errorf("expected empty list after removal, got %v", names)
	}
}

func TestRegistry_Remove_NotFound(t *testing.T) {
	reg := NewRegistry()
	if reg.Remove("nonexistent") == nil {
		t.Fatal("expected error removing non-existent tool")
	}
}

func TestRegistry_All(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Add(&mockTool{name: "beta"})
	_ = reg.Add(&mockTool{name: "alpha"})

	tools := reg.All()
	if len(tools) != 2 {
		t.Fatalf("All() len = %d, want 2", len(tools))
	}
	// Should be sorted.
	if tools[0].Name() != "alpha" {
		t.Errorf("All()[0] = %q, want %q", tools[0].Name(), "alpha")
	}
	if tools[1].Name() != "beta" {
		t.Errorf("All()[1] = %q, want %q", tools[1].Name(), "beta")
	}
}

func TestRegistry_Definitions(t *testing.T) {
	reg := NewRegistry()
	_ = reg.Add(&mockTool{
		name:        "search",
		description: "Search the web",
		inputSchema: map[string]any{"type": "object"},
	})

	defs := reg.Definitions()
	if len(defs) != 1 {
		t.Fatalf("Definitions() len = %d, want 1", len(defs))
	}
	if defs[0]["name"] != "search" {
		t.Errorf("name = %v, want %q", defs[0]["name"], "search")
	}
	if defs[0]["description"] != "Search the web" {
		t.Errorf("description = %v, want %q", defs[0]["description"], "Search the web")
	}
}
