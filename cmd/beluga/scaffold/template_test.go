package scaffold

import (
	"errors"
	"io/fs"
	"sync"
	"testing"
	"testing/fstest"

	"github.com/lookatitude/beluga-ai/v2/core"
)

// TestNewRegistry_Empty asserts the zero-created registry returns an empty,
// non-nil slice from Names (nil-safe for the --template help text).
func TestNewRegistry_Empty(t *testing.T) {
	r := NewRegistry()
	names := r.Names()
	if names == nil {
		t.Fatalf("Names on empty registry must not be nil")
	}
	if len(names) != 0 {
		t.Errorf("Names on empty registry: want empty slice, got %v", names)
	}
	if _, ok := r.Get("basic"); ok {
		t.Errorf("Get('basic') on empty registry: want ok=false, got ok=true")
	}
}

// TestRegister_DuplicateRejected asserts that re-registering the same name
// returns an ErrInvalidInput — the registry is append-only.
func TestRegister_DuplicateRejected(t *testing.T) {
	r := NewRegistry()
	testFS := fstest.MapFS{"main.go": {Data: []byte("package foo")}}

	if err := r.Register("sample", testFS); err != nil {
		t.Fatalf("first Register: unexpected error: %v", err)
	}
	err := r.Register("sample", testFS)
	if err == nil {
		t.Fatalf("second Register: expected duplicate error, got nil")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrInvalidInput {
		t.Errorf("second Register: want ErrInvalidInput, got %v", err)
	}
}

// TestRegister_EmptyNameRejected asserts the registry refuses empty names.
func TestRegister_EmptyNameRejected(t *testing.T) {
	r := NewRegistry()
	testFS := fstest.MapFS{"main.go": {Data: []byte("package foo")}}
	err := r.Register("", testFS)
	if err == nil {
		t.Fatalf("Register('', fs): expected error, got nil")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrInvalidInput {
		t.Errorf("Register('', fs): want ErrInvalidInput, got %v", err)
	}
}

// TestRegister_NilFSRejected asserts the registry refuses nil filesystems.
func TestRegister_NilFSRejected(t *testing.T) {
	r := NewRegistry()
	if err := r.Register("nilfs", nil); err == nil {
		t.Fatalf("Register('nilfs', nil): expected error, got nil")
	}
}

// TestRegistry_GetAndNames asserts Get resolves registered entries and
// Names returns a sorted slice of keys.
func TestRegistry_GetAndNames(t *testing.T) {
	r := NewRegistry()
	fsA := fstest.MapFS{"a.txt": {Data: []byte("a")}}
	fsB := fstest.MapFS{"b.txt": {Data: []byte("b")}}
	// Register in reverse alphabetical order to prove Names sorts.
	if err := r.Register("bravo", fsB); err != nil {
		t.Fatalf("Register bravo: %v", err)
	}
	if err := r.Register("alpha", fsA); err != nil {
		t.Fatalf("Register alpha: %v", err)
	}

	got, ok := r.Get("alpha")
	if !ok {
		t.Fatalf("Get('alpha'): want ok=true")
	}
	// Round-trip: verify the FS handed back is the one we registered.
	if _, err := fs.Stat(got, "a.txt"); err != nil {
		t.Errorf("expected a.txt in registered FS, got err %v", err)
	}

	names := r.Names()
	if len(names) != 2 || names[0] != "alpha" || names[1] != "bravo" {
		t.Errorf("Names: want [alpha bravo], got %v", names)
	}
}

// TestRegistry_Concurrent exercises the RWMutex: 8 goroutines concurrently
// calling Register (for distinct names) and Get. The race detector must
// remain clean.
func TestRegistry_Concurrent(t *testing.T) {
	r := NewRegistry()
	testFS := fstest.MapFS{"main.go": {Data: []byte("package foo")}}

	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		name := string(rune('a' + i))
		go func() {
			defer wg.Done()
			_ = r.Register(name, testFS)
			_, _ = r.Get(name)
			_ = r.Names()
		}()
	}
	wg.Wait()

	if got := len(r.Names()); got != 8 {
		t.Errorf("after concurrent Register: want 8 entries, got %d", got)
	}
}
