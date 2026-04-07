package cache

import (
	"context"
	"math"
	"sync"
	"testing"
	"time"
)

// mockEmbedder returns deterministic vectors based on input text.
// It maps known strings to fixed vectors; unknown strings get a default vector.
type mockEmbedder struct {
	vectors map[string][]float32
	dims    int
}

func newMockEmbedder() *mockEmbedder {
	return &mockEmbedder{
		dims: 3,
		vectors: map[string][]float32{
			"hello":     {1, 0, 0},
			"hi":        {0.95, 0.05, 0},   // very similar to "hello"
			"greetings": {0.90, 0.10, 0.05}, // similar to "hello"
			"goodbye":   {0, 0, 1},          // orthogonal to "hello"
			"farewell":  {0.05, 0, 0.95},    // similar to "goodbye"
			"weather":   {0, 1, 0},          // orthogonal to both
			"foo":       {0.5, 0.5, 0.5},
			"bar":       {-0.5, -0.5, -0.5},
		},
	}
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	result := make([][]float32, len(texts))
	for i, t := range texts {
		v, err := m.EmbedSingle(ctx, t)
		if err != nil {
			return nil, err
		}
		result[i] = v
	}
	return result, nil
}

func (m *mockEmbedder) EmbedSingle(_ context.Context, text string) ([]float32, error) {
	if v, ok := m.vectors[text]; ok {
		cp := make([]float32, len(v))
		copy(cp, v)
		return cp, nil
	}
	// Default: a unique-ish vector
	return []float32{0.33, 0.33, 0.33}, nil
}

func (m *mockEmbedder) Dimensions() int {
	return m.dims
}

// --- Cosine Similarity Tests ---

func TestCosineSimilarity_Identical(t *testing.T) {
	a := []float32{1, 2, 3}
	got := cosineSimilarity(a, a)
	if math.Abs(got-1.0) > 1e-6 {
		t.Errorf("identical vectors: got %v, want 1.0", got)
	}
}

func TestCosineSimilarity_UnitVectors(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	got := cosineSimilarity(a, b)
	if math.Abs(got) > 1e-6 {
		t.Errorf("orthogonal unit vectors: got %v, want 0.0", got)
	}
}

func TestCosineSimilarity_ZeroVector(t *testing.T) {
	a := []float32{0, 0, 0}
	b := []float32{1, 2, 3}
	if got := cosineSimilarity(a, b); got != 0 {
		t.Errorf("zero vector a: got %v, want 0", got)
	}
	if got := cosineSimilarity(b, a); got != 0 {
		t.Errorf("zero vector b: got %v, want 0", got)
	}
}

func TestCosineSimilarity_MismatchedDims(t *testing.T) {
	a := []float32{1, 2}
	b := []float32{1, 2, 3}
	if got := cosineSimilarity(a, b); got != 0 {
		t.Errorf("mismatched dims: got %v, want 0", got)
	}
}

func TestCosineSimilarity_Empty(t *testing.T) {
	if got := cosineSimilarity(nil, nil); got != 0 {
		t.Errorf("nil vectors: got %v, want 0", got)
	}
	if got := cosineSimilarity([]float32{}, []float32{}); got != 0 {
		t.Errorf("empty vectors: got %v, want 0", got)
	}
}

func TestCosineSimilarity_Antiparallel(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{-1, 0, 0}
	got := cosineSimilarity(a, b)
	if math.Abs(got-(-1.0)) > 1e-6 {
		t.Errorf("antiparallel: got %v, want -1.0", got)
	}
}

// --- SemanticCache Set/Get Tests ---

func TestSemanticCache_SetGetHit(t *testing.T) {
	emb := newMockEmbedder()
	sc := NewSemanticCache(emb, WithThreshold(0.85))
	ctx := context.Background()

	if err := sc.Set(ctx, "hello", "world", 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// "hi" is very similar to "hello" — should be a hit.
	val, ok, err := sc.Get(ctx, "hi")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("expected cache hit for similar key")
	}
	if val != "world" {
		t.Errorf("got %v, want %q", val, "world")
	}
}

func TestSemanticCache_SetGetMiss(t *testing.T) {
	emb := newMockEmbedder()
	sc := NewSemanticCache(emb, WithThreshold(0.85))
	ctx := context.Background()

	if err := sc.Set(ctx, "hello", "world", 0); err != nil {
		t.Fatalf("Set: %v", err)
	}

	// "goodbye" is orthogonal to "hello" — should miss.
	val, ok, err := sc.Get(ctx, "goodbye")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ok {
		t.Errorf("expected cache miss for dissimilar key, got val=%v", val)
	}
}

func TestSemanticCache_ExactKeyUpdate(t *testing.T) {
	emb := newMockEmbedder()
	sc := NewSemanticCache(emb, WithThreshold(0.99))
	ctx := context.Background()

	_ = sc.Set(ctx, "hello", "first", 0)
	_ = sc.Set(ctx, "hello", "second", 0)

	val, ok, err := sc.Get(ctx, "hello")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("expected hit")
	}
	if val != "second" {
		t.Errorf("got %v, want %q", val, "second")
	}

	// Should have only 1 entry, not 2.
	sc.mu.RLock()
	n := len(sc.entries)
	sc.mu.RUnlock()
	if n != 1 {
		t.Errorf("expected 1 entry after update, got %d", n)
	}
}

func TestSemanticCache_TTLExpiration(t *testing.T) {
	emb := newMockEmbedder()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sc := NewSemanticCache(emb, WithThreshold(0.99))
	sc.now = func() time.Time { return now }
	ctx := context.Background()

	_ = sc.Set(ctx, "hello", "value", 5*time.Second)

	// Still within TTL.
	val, ok, err := sc.Get(ctx, "hello")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok || val != "value" {
		t.Fatalf("expected hit before expiry, ok=%v val=%v", ok, val)
	}

	// Advance past TTL.
	now = now.Add(10 * time.Second)
	_, ok, err = sc.Get(ctx, "hello")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ok {
		t.Error("expected miss after TTL expiration")
	}
}

func TestSemanticCache_DefaultTTL(t *testing.T) {
	emb := newMockEmbedder()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sc := NewSemanticCache(emb, WithThreshold(0.99), WithDefaultTTL(3*time.Second))
	sc.now = func() time.Time { return now }
	ctx := context.Background()

	// TTL=0 should use defaultTTL.
	_ = sc.Set(ctx, "hello", "val", 0)

	now = now.Add(5 * time.Second)
	_, ok, err := sc.Get(ctx, "hello")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if ok {
		t.Error("expected miss after default TTL")
	}
}

func TestSemanticCache_NegativeTTL_NoExpiration(t *testing.T) {
	emb := newMockEmbedder()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sc := NewSemanticCache(emb, WithThreshold(0.99), WithDefaultTTL(1*time.Second))
	sc.now = func() time.Time { return now }
	ctx := context.Background()

	_ = sc.Set(ctx, "hello", "forever", -1)

	now = now.Add(24 * time.Hour)
	val, ok, err := sc.Get(ctx, "hello")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok || val != "forever" {
		t.Errorf("negative TTL entry should not expire, ok=%v val=%v", ok, val)
	}
}

func TestSemanticCache_MaxEntries(t *testing.T) {
	emb := newMockEmbedder()
	sc := NewSemanticCache(emb, WithThreshold(0.99), WithMaxEntries(2))
	ctx := context.Background()

	_ = sc.Set(ctx, "hello", "v1", 0)
	_ = sc.Set(ctx, "goodbye", "v2", 0)
	_ = sc.Set(ctx, "weather", "v3", 0) // should evict "hello"

	// "hello" should be evicted (oldest).
	_, ok, _ := sc.Get(ctx, "hello")
	if ok {
		t.Error("expected 'hello' to be evicted")
	}

	// "goodbye" and "weather" should still be present.
	val, ok, _ := sc.Get(ctx, "goodbye")
	if !ok || val != "v2" {
		t.Errorf("goodbye: ok=%v val=%v", ok, val)
	}
	val, ok, _ = sc.Get(ctx, "weather")
	if !ok || val != "v3" {
		t.Errorf("weather: ok=%v val=%v", ok, val)
	}
}

func TestSemanticCache_Delete(t *testing.T) {
	emb := newMockEmbedder()
	sc := NewSemanticCache(emb, WithThreshold(0.99))
	ctx := context.Background()

	_ = sc.Set(ctx, "hello", "val", 0)
	if err := sc.Delete(ctx, "hello"); err != nil {
		t.Fatalf("Delete: %v", err)
	}

	_, ok, _ := sc.Get(ctx, "hello")
	if ok {
		t.Error("expected miss after Delete")
	}

	// Deleting non-existent key is a no-op.
	if err := sc.Delete(ctx, "nonexistent"); err != nil {
		t.Fatalf("Delete non-existent: %v", err)
	}
}

func TestSemanticCache_Clear(t *testing.T) {
	emb := newMockEmbedder()
	sc := NewSemanticCache(emb, WithThreshold(0.99))
	ctx := context.Background()

	_ = sc.Set(ctx, "hello", "v1", 0)
	_ = sc.Set(ctx, "goodbye", "v2", 0)

	if err := sc.Clear(ctx); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	sc.mu.RLock()
	n := len(sc.entries)
	sc.mu.RUnlock()
	if n != 0 {
		t.Errorf("expected 0 entries after Clear, got %d", n)
	}
}

// --- GetByEmbedding / SetByEmbedding Tests ---

func TestSemanticCache_GetSetByEmbedding(t *testing.T) {
	emb := newMockEmbedder()
	sc := NewSemanticCache(emb, WithThreshold(0.85))
	ctx := context.Background()

	vec := []float32{1, 0, 0}
	if err := sc.SetByEmbedding(ctx, "custom", vec, "data", 0); err != nil {
		t.Fatalf("SetByEmbedding: %v", err)
	}

	// Exact same vector should hit.
	val, ok, err := sc.GetByEmbedding(ctx, vec)
	if err != nil {
		t.Fatalf("GetByEmbedding: %v", err)
	}
	if !ok || val != "data" {
		t.Errorf("expected hit, ok=%v val=%v", ok, val)
	}

	// Similar vector should also hit.
	similar := []float32{0.98, 0.02, 0}
	val, ok, err = sc.GetByEmbedding(ctx, similar)
	if err != nil {
		t.Fatalf("GetByEmbedding: %v", err)
	}
	if !ok || val != "data" {
		t.Errorf("expected hit for similar vector, ok=%v val=%v", ok, val)
	}

	// Orthogonal vector should miss.
	orthogonal := []float32{0, 1, 0}
	_, ok, err = sc.GetByEmbedding(ctx, orthogonal)
	if err != nil {
		t.Fatalf("GetByEmbedding: %v", err)
	}
	if ok {
		t.Error("expected miss for orthogonal vector")
	}
}

// --- Concurrent Access Test ---

func TestSemanticCache_ConcurrentAccess(t *testing.T) {
	emb := newMockEmbedder()
	sc := NewSemanticCache(emb, WithThreshold(0.99), WithMaxEntries(100))
	ctx := context.Background()

	var wg sync.WaitGroup
	keys := []string{"hello", "goodbye", "weather", "foo"}

	// Concurrent writers.
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := keys[idx%len(keys)]
			_ = sc.Set(ctx, key, idx, 0)
		}(i)
	}

	// Concurrent readers.
	for i := 0; i < 20; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := keys[idx%len(keys)]
			_, _, _ = sc.Get(ctx, key)
		}(i)
	}

	wg.Wait()
}

// --- Registry Integration Test ---

func TestSemanticCache_Registry(t *testing.T) {
	emb := newMockEmbedder()

	c, err := New("semantic", Config{
		TTL:     10 * time.Second,
		MaxSize: 50,
		Options: map[string]any{
			"embedder":  emb,
			"threshold": 0.9,
		},
	})
	if err != nil {
		t.Fatalf("New(\"semantic\"): %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil cache")
	}

	sc, ok := c.(*SemanticCache)
	if !ok {
		t.Fatal("expected *SemanticCache from registry")
	}
	if sc.threshold != 0.9 {
		t.Errorf("threshold = %v, want 0.9", sc.threshold)
	}
	if sc.maxEntries != 50 {
		t.Errorf("maxEntries = %v, want 50", sc.maxEntries)
	}
	if sc.defaultTTL != 10*time.Second {
		t.Errorf("defaultTTL = %v, want 10s", sc.defaultTTL)
	}
}

func TestSemanticCache_Registry_MissingEmbedder(t *testing.T) {
	_, err := New("semantic", Config{
		Options: map[string]any{},
	})
	if err == nil {
		t.Fatal("expected error for missing embedder")
	}
}

func TestSemanticCache_Registry_InvalidEmbedder(t *testing.T) {
	_, err := New("semantic", Config{
		Options: map[string]any{
			"embedder": "not-an-embedder",
		},
	})
	if err == nil {
		t.Fatal("expected error for invalid embedder type")
	}
}

func TestSemanticCache_BestMatch(t *testing.T) {
	emb := newMockEmbedder()
	sc := NewSemanticCache(emb, WithThreshold(0.5))
	ctx := context.Background()

	// Store two entries with different embeddings.
	_ = sc.Set(ctx, "hello", "hello-val", 0)
	_ = sc.Set(ctx, "goodbye", "goodbye-val", 0)

	// "farewell" is similar to "goodbye" but not "hello" — should return "goodbye-val".
	val, ok, err := sc.Get(ctx, "farewell")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok {
		t.Fatal("expected hit")
	}
	if val != "goodbye-val" {
		t.Errorf("got %v, want %q (best match should be goodbye)", val, "goodbye-val")
	}
}

func TestSemanticCache_DefaultThreshold(t *testing.T) {
	emb := newMockEmbedder()
	sc := NewSemanticCache(emb) // no options — default threshold 0.85
	if sc.threshold != 0.85 {
		t.Errorf("default threshold = %v, want 0.85", sc.threshold)
	}
}

// --- Input Validation Tests ---

func TestWithThreshold_Clamping(t *testing.T) {
	tests := []struct {
		name  string
		input float64
		want  float64
	}{
		{"zero clamped to 0.01", 0, 0.01},
		{"negative clamped to 0.01", -1, 0.01},
		{"above 1 clamped to 1.0", 2.0, 1.0},
		{"valid value unchanged", 0.5, 0.5},
		{"lower bound exact", 0.01, 0.01},
		{"upper bound exact", 1.0, 1.0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			emb := newMockEmbedder()
			sc := NewSemanticCache(emb, WithThreshold(tt.input))
			if sc.threshold != tt.want {
				t.Errorf("WithThreshold(%v): got %v, want %v", tt.input, sc.threshold, tt.want)
			}
		})
	}
}

func TestWithMaxEntries_NegativeClamped(t *testing.T) {
	emb := newMockEmbedder()
	sc := NewSemanticCache(emb, WithMaxEntries(-5))
	if sc.maxEntries != 0 {
		t.Errorf("WithMaxEntries(-5): got %d, want 0 (unlimited)", sc.maxEntries)
	}
}

func TestSemanticCache_EmbeddingDimensionValidation(t *testing.T) {
	emb := newMockEmbedder()
	sc := NewSemanticCache(emb, WithMaxDimensions(4))
	ctx := context.Background()

	// Within limit — should succeed.
	err := sc.SetByEmbedding(ctx, "ok", []float32{1, 2, 3, 4}, "val", 0)
	if err != nil {
		t.Fatalf("expected no error for 4-dim vector, got: %v", err)
	}

	// Exceeds limit — should fail.
	err = sc.SetByEmbedding(ctx, "too-big", []float32{1, 2, 3, 4, 5}, "val", 0)
	if err == nil {
		t.Fatal("expected error for 5-dim vector with maxDimensions=4")
	}
}

func TestSemanticCache_DefaultMaxDimensions(t *testing.T) {
	emb := newMockEmbedder()
	sc := NewSemanticCache(emb)
	if sc.maxDimensions != 8192 {
		t.Errorf("default maxDimensions = %d, want 8192", sc.maxDimensions)
	}
}

func TestSemanticCache_SetEvictsExpiredEntries(t *testing.T) {
	emb := newMockEmbedder()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sc := NewSemanticCache(emb, WithThreshold(0.99))
	sc.now = func() time.Time { return now }
	ctx := context.Background()

	// Add entries with short TTL.
	_ = sc.Set(ctx, "hello", "v1", 2*time.Second)
	_ = sc.Set(ctx, "goodbye", "v2", 2*time.Second)
	// Add one that won't expire.
	_ = sc.Set(ctx, "weather", "v3", -1)

	sc.mu.RLock()
	before := len(sc.entries)
	sc.mu.RUnlock()
	if before != 3 {
		t.Fatalf("expected 3 entries before expiry, got %d", before)
	}

	// Advance past TTL so hello and goodbye expire.
	now = now.Add(5 * time.Second)

	// A new Set (write operation) should sweep expired entries.
	_ = sc.Set(ctx, "foo", "v4", -1)

	sc.mu.RLock()
	after := len(sc.entries)
	sc.mu.RUnlock()
	// Should have "weather" + "foo" = 2 entries. "hello" and "goodbye" evicted.
	if after != 2 {
		t.Errorf("expected 2 entries after write-triggered cleanup, got %d", after)
	}
}

func TestSemanticCache_Prune(t *testing.T) {
	emb := newMockEmbedder()
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	sc := NewSemanticCache(emb, WithThreshold(0.99))
	sc.now = func() time.Time { return now }
	ctx := context.Background()

	_ = sc.Set(ctx, "hello", "v1", 2*time.Second)
	_ = sc.Set(ctx, "goodbye", "v2", 2*time.Second)
	_ = sc.Set(ctx, "weather", "v3", -1)

	// Advance past TTL.
	now = now.Add(5 * time.Second)

	if err := sc.Prune(ctx); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	sc.mu.RLock()
	n := len(sc.entries)
	sc.mu.RUnlock()
	if n != 1 {
		t.Errorf("expected 1 entry after Prune, got %d", n)
	}

	// The surviving entry should be "weather".
	val, ok, err := sc.Get(ctx, "weather")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if !ok || val != "v3" {
		t.Errorf("expected weather entry to survive, ok=%v val=%v", ok, val)
	}
}

func TestSemanticCache_PruneNoExpired(t *testing.T) {
	emb := newMockEmbedder()
	sc := NewSemanticCache(emb, WithThreshold(0.99))
	ctx := context.Background()

	_ = sc.Set(ctx, "hello", "v1", -1)
	_ = sc.Set(ctx, "goodbye", "v2", -1)

	if err := sc.Prune(ctx); err != nil {
		t.Fatalf("Prune: %v", err)
	}

	sc.mu.RLock()
	n := len(sc.entries)
	sc.mu.RUnlock()
	if n != 2 {
		t.Errorf("expected 2 entries after Prune with no expired, got %d", n)
	}
}

func TestSemanticCache_EmbeddingDefensiveCopy(t *testing.T) {
	emb := newMockEmbedder()
	sc := NewSemanticCache(emb, WithThreshold(0.99))
	ctx := context.Background()

	vec := []float32{1, 0, 0}
	if err := sc.SetByEmbedding(ctx, "test", vec, "data", 0); err != nil {
		t.Fatalf("SetByEmbedding: %v", err)
	}

	// Mutate the original slice after storing.
	vec[0] = 0
	vec[1] = 1

	// The cache should still return a hit for {1, 0, 0}, not the mutated vector.
	lookup := []float32{1, 0, 0}
	val, ok, err := sc.GetByEmbedding(ctx, lookup)
	if err != nil {
		t.Fatalf("GetByEmbedding: %v", err)
	}
	if !ok || val != "data" {
		t.Errorf("expected cache hit after mutating original embedding, ok=%v val=%v", ok, val)
	}
}
