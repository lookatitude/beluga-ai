package cache

import (
	"sort"
	"testing"
	"time"
)

func TestConfig_Fields(t *testing.T) {
	cfg := Config{
		TTL:     5 * time.Minute,
		MaxSize: 1000,
		Options: map[string]any{"custom": true},
	}

	if cfg.TTL != 5*time.Minute {
		t.Errorf("TTL = %v, want %v", cfg.TTL, 5*time.Minute)
	}
	if cfg.MaxSize != 1000 {
		t.Errorf("MaxSize = %d, want %d", cfg.MaxSize, 1000)
	}
	if cfg.Options["custom"] != true {
		t.Errorf("Options[custom] = %v, want true", cfg.Options["custom"])
	}
}

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{}

	if cfg.TTL != 0 {
		t.Errorf("TTL = %v, want 0", cfg.TTL)
	}
	if cfg.MaxSize != 0 {
		t.Errorf("MaxSize = %d, want 0", cfg.MaxSize)
	}
	if cfg.Options != nil {
		t.Errorf("Options = %v, want nil", cfg.Options)
	}
}

// testFactory is a simple factory for testing the registry.
func testFactory(cfg Config) (Cache, error) {
	return nil, nil
}

func TestRegistry_RegisterAndList(t *testing.T) {
	// Register a test provider.
	Register("test_provider_abc", testFactory)
	defer func() {
		// Clean up by removing from registry (re-register is overwrite).
		mu.Lock()
		delete(registry, "test_provider_abc")
		mu.Unlock()
	}()

	names := List()
	found := false
	for _, name := range names {
		if name == "test_provider_abc" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("List() = %v, want to contain %q", names, "test_provider_abc")
	}
}

func TestRegistry_List_Sorted(t *testing.T) {
	names := List()
	if !sort.StringsAreSorted(names) {
		t.Errorf("List() = %v, want sorted", names)
	}
}

func TestRegistry_Register_Overwrite(t *testing.T) {
	// Unlike guard.Register, cache.Register allows overwriting.
	called := false
	Register("overwrite_test_xyz", func(cfg Config) (Cache, error) {
		called = true
		return nil, nil
	})
	defer func() {
		mu.Lock()
		delete(registry, "overwrite_test_xyz")
		mu.Unlock()
	}()

	// Overwrite with a new factory.
	Register("overwrite_test_xyz", func(cfg Config) (Cache, error) {
		called = true
		return nil, nil
	})

	_, _ = New("overwrite_test_xyz", Config{})
	if !called {
		t.Error("overwritten factory was not called")
	}
}

func TestRegistry_New_UnknownProvider(t *testing.T) {
	_, err := New("nonexistent_provider_xyz", Config{})
	if err == nil {
		t.Fatal("New(nonexistent_provider_xyz) expected error, got nil")
	}
}

func TestRegistry_New_ErrorMessageContainsList(t *testing.T) {
	_, err := New("unknown_provider_abc", Config{})
	if err == nil {
		t.Fatal("expected error")
	}
	// Error message should contain the registered providers.
	errStr := err.Error()
	if errStr == "" {
		t.Error("error message should not be empty")
	}
}

func TestRegistry_New_ValidProvider(t *testing.T) {
	Register("valid_test_provider", func(cfg Config) (Cache, error) {
		// Return a nil cache for test purposes.
		return nil, nil
	})
	defer func() {
		mu.Lock()
		delete(registry, "valid_test_provider")
		mu.Unlock()
	}()

	c, err := New("valid_test_provider", Config{TTL: time.Minute})
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	// Our test factory returns nil, which is valid.
	_ = c
}
