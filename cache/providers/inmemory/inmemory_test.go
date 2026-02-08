package inmemory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/lookatitude/beluga-ai/cache"
)

func newTestCache(ttl time.Duration, maxSize int) *InMemoryCache {
	return New(cache.Config{TTL: ttl, MaxSize: maxSize})
}

func TestInMemoryCache_SetAndGet(t *testing.T) {
	c := newTestCache(time.Minute, 100)
	ctx := context.Background()

	if err := c.Set(ctx, "key1", "value1", 0); err != nil {
		t.Fatalf("Set() error = %v", err)
	}

	val, ok, err := c.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if val != "value1" {
		t.Errorf("Get() = %v, want %q", val, "value1")
	}
}

func TestInMemoryCache_GetMissing(t *testing.T) {
	c := newTestCache(time.Minute, 100)
	ctx := context.Background()

	val, ok, err := c.Get(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if ok {
		t.Error("Get() ok = true, want false for missing key")
	}
	if val != nil {
		t.Errorf("Get() = %v, want nil", val)
	}
}

func TestInMemoryCache_SetOverwrite(t *testing.T) {
	c := newTestCache(time.Minute, 100)
	ctx := context.Background()

	_ = c.Set(ctx, "key", "v1", 0)
	_ = c.Set(ctx, "key", "v2", 0)

	val, ok, err := c.Get(ctx, "key")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if !ok {
		t.Fatal("Get() ok = false, want true")
	}
	if val != "v2" {
		t.Errorf("Get() = %v, want %q", val, "v2")
	}
}

func TestInMemoryCache_Delete(t *testing.T) {
	c := newTestCache(time.Minute, 100)
	ctx := context.Background()

	_ = c.Set(ctx, "key", "value", 0)
	if err := c.Delete(ctx, "key"); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, ok, _ := c.Get(ctx, "key")
	if ok {
		t.Error("Get() ok = true after Delete(), want false")
	}
}

func TestInMemoryCache_DeleteNonexistent(t *testing.T) {
	c := newTestCache(time.Minute, 100)
	ctx := context.Background()

	err := c.Delete(ctx, "nonexistent")
	if err != nil {
		t.Errorf("Delete() of nonexistent key error = %v, want nil", err)
	}
}

func TestInMemoryCache_Clear(t *testing.T) {
	c := newTestCache(time.Minute, 100)
	ctx := context.Background()

	_ = c.Set(ctx, "a", 1, 0)
	_ = c.Set(ctx, "b", 2, 0)
	_ = c.Set(ctx, "c", 3, 0)

	if err := c.Clear(ctx); err != nil {
		t.Fatalf("Clear() error = %v", err)
	}

	if c.Len() != 0 {
		t.Errorf("Len() = %d after Clear(), want 0", c.Len())
	}

	_, ok, _ := c.Get(ctx, "a")
	if ok {
		t.Error("Get(a) ok = true after Clear(), want false")
	}
}

func TestInMemoryCache_TTL_Expiration(t *testing.T) {
	c := newTestCache(time.Minute, 100)
	ctx := context.Background()

	// Use injectable now function to simulate time.
	currentTime := time.Now()
	c.now = func() time.Time { return currentTime }

	_ = c.Set(ctx, "key", "value", 100*time.Millisecond)

	// Before expiry.
	val, ok, _ := c.Get(ctx, "key")
	if !ok {
		t.Fatal("Get() ok = false, want true before expiry")
	}
	if val != "value" {
		t.Errorf("Get() = %v, want %q", val, "value")
	}

	// Advance time past TTL.
	currentTime = currentTime.Add(200 * time.Millisecond)

	_, ok, _ = c.Get(ctx, "key")
	if ok {
		t.Error("Get() ok = true after TTL expired, want false")
	}
}

func TestInMemoryCache_DefaultTTL(t *testing.T) {
	c := newTestCache(50*time.Millisecond, 100)
	ctx := context.Background()

	currentTime := time.Now()
	c.now = func() time.Time { return currentTime }

	// TTL=0 should use default (50ms).
	_ = c.Set(ctx, "key", "value", 0)

	// Before default TTL.
	_, ok, _ := c.Get(ctx, "key")
	if !ok {
		t.Fatal("Get() ok = false before default TTL")
	}

	// Advance past default TTL.
	currentTime = currentTime.Add(100 * time.Millisecond)

	_, ok, _ = c.Get(ctx, "key")
	if ok {
		t.Error("Get() ok = true after default TTL expired")
	}
}

func TestInMemoryCache_NegativeTTL_NoExpiration(t *testing.T) {
	c := newTestCache(50*time.Millisecond, 100)
	ctx := context.Background()

	currentTime := time.Now()
	c.now = func() time.Time { return currentTime }

	// Negative TTL = never expires.
	_ = c.Set(ctx, "key", "value", -1)

	// Advance way past default TTL.
	currentTime = currentTime.Add(10 * time.Second)

	val, ok, _ := c.Get(ctx, "key")
	if !ok {
		t.Fatal("Get() ok = false, want true for non-expiring entry")
	}
	if val != "value" {
		t.Errorf("Get() = %v, want %q", val, "value")
	}
}

func TestInMemoryCache_LRU_Eviction(t *testing.T) {
	c := newTestCache(time.Minute, 3)
	ctx := context.Background()

	_ = c.Set(ctx, "a", 1, 0)
	_ = c.Set(ctx, "b", 2, 0)
	_ = c.Set(ctx, "c", 3, 0)

	// Cache is full (3 items). Adding one more evicts LRU ("a").
	_ = c.Set(ctx, "d", 4, 0)

	if c.Len() != 3 {
		t.Errorf("Len() = %d, want 3", c.Len())
	}

	_, ok, _ := c.Get(ctx, "a")
	if ok {
		t.Error("Get(a) ok = true, want false (should be evicted as LRU)")
	}

	// b, c, d should still exist.
	for _, key := range []string{"b", "c", "d"} {
		_, ok, _ := c.Get(ctx, key)
		if !ok {
			t.Errorf("Get(%q) ok = false, want true", key)
		}
	}
}

func TestInMemoryCache_LRU_AccessPromotes(t *testing.T) {
	c := newTestCache(time.Minute, 3)
	ctx := context.Background()

	_ = c.Set(ctx, "a", 1, 0)
	_ = c.Set(ctx, "b", 2, 0)
	_ = c.Set(ctx, "c", 3, 0)

	// Access "a" to promote it from LRU to MRU.
	_, _, _ = c.Get(ctx, "a")

	// Add new item → "b" should be evicted (now LRU).
	_ = c.Set(ctx, "d", 4, 0)

	_, ok, _ := c.Get(ctx, "b")
	if ok {
		t.Error("Get(b) ok = true, want false (should be evicted as LRU)")
	}

	// a, c, d should exist.
	_, ok, _ = c.Get(ctx, "a")
	if !ok {
		t.Error("Get(a) ok = false, want true (was promoted by access)")
	}
}

func TestInMemoryCache_MaxSize_Zero_Unlimited(t *testing.T) {
	c := newTestCache(time.Minute, 0)
	ctx := context.Background()

	for i := 0; i < 1000; i++ {
		_ = c.Set(ctx, fmt.Sprintf("key-%d", i), i, 0)
	}

	if c.Len() != 1000 {
		t.Errorf("Len() = %d, want 1000 (unlimited)", c.Len())
	}
}

func TestInMemoryCache_Len(t *testing.T) {
	c := newTestCache(time.Minute, 100)
	ctx := context.Background()

	if c.Len() != 0 {
		t.Errorf("Len() = %d on empty cache, want 0", c.Len())
	}

	_ = c.Set(ctx, "a", 1, 0)
	_ = c.Set(ctx, "b", 2, 0)

	if c.Len() != 2 {
		t.Errorf("Len() = %d, want 2", c.Len())
	}

	_ = c.Delete(ctx, "a")

	if c.Len() != 1 {
		t.Errorf("Len() = %d after delete, want 1", c.Len())
	}
}

func TestInMemoryCache_DifferentValueTypes(t *testing.T) {
	c := newTestCache(time.Minute, 100)
	ctx := context.Background()

	tests := []struct {
		name  string
		key   string
		value any
	}{
		{"string", "str", "hello"},
		{"int", "num", 42},
		{"float", "flt", 3.14},
		{"bool", "bln", true},
		{"slice", "slc", []int{1, 2, 3}},
		{"map", "mp", map[string]int{"a": 1}},
		{"nil", "nil", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := c.Set(ctx, tt.key, tt.value, 0); err != nil {
				t.Fatalf("Set() error = %v", err)
			}
			val, ok, err := c.Get(ctx, tt.key)
			if err != nil {
				t.Fatalf("Get() error = %v", err)
			}
			if !ok {
				t.Fatal("Get() ok = false, want true")
			}
			// For nil, check directly.
			if tt.value == nil {
				if val != nil {
					t.Errorf("Get() = %v, want nil", val)
				}
				return
			}
			// For others, use fmt comparison.
			if fmt.Sprintf("%v", val) != fmt.Sprintf("%v", tt.value) {
				t.Errorf("Get() = %v, want %v", val, tt.value)
			}
		})
	}
}

func TestInMemoryCache_Registry(t *testing.T) {
	// Verify the cache is registered via init().
	names := cache.List()
	found := false
	for _, name := range names {
		if name == "inmemory" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("cache.List() = %v, want to contain %q", names, "inmemory")
	}

	// Create via registry.
	c, err := cache.New("inmemory", cache.Config{TTL: time.Minute, MaxSize: 10})
	if err != nil {
		t.Fatalf("cache.New() error = %v", err)
	}
	if c == nil {
		t.Fatal("cache.New() returned nil")
	}
}

func TestInMemoryCache_SetUpdatePromotesToFront(t *testing.T) {
	c := newTestCache(time.Minute, 3)
	ctx := context.Background()

	_ = c.Set(ctx, "a", 1, 0)
	_ = c.Set(ctx, "b", 2, 0)
	_ = c.Set(ctx, "c", 3, 0)

	// Update "a" (promotes to front).
	_ = c.Set(ctx, "a", 10, 0)

	// Add "d" → "b" should be evicted (LRU).
	_ = c.Set(ctx, "d", 4, 0)

	_, ok, _ := c.Get(ctx, "b")
	if ok {
		t.Error("Get(b) ok = true, want false (evicted after a was promoted)")
	}

	val, ok, _ := c.Get(ctx, "a")
	if !ok {
		t.Fatal("Get(a) ok = false, want true")
	}
	if val != 10 {
		t.Errorf("Get(a) = %v, want 10", val)
	}
}
