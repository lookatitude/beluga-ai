package inmemory_test

import (
	"testing"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/memory"
	_ "github.com/lookatitude/beluga-ai/v2/memory/stores/inmemory"
)

// TestInmemoryRegistered verifies the blank import side-effect: the "inmemory"
// name is present in memory.List() and memory.New("inmemory", ...) returns a
// non-nil Memory without error.
func TestInmemoryRegistered(t *testing.T) {
	providers := memory.List()

	var found bool
	for _, p := range providers {
		if p == "inmemory" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("inmemory not in memory.List(): %v", providers)
	}

	mem, err := memory.New("inmemory", config.ProviderConfig{})
	if err != nil {
		t.Fatalf("memory.New(\"inmemory\"): %v", err)
	}
	if mem == nil {
		t.Fatal("memory.New(\"inmemory\") returned nil Memory")
	}
}
