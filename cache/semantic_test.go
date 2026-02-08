package cache

import (
	"context"
	"testing"
	"time"
)

// mockCache is a simple in-memory cache for testing SemanticCache.
type mockCache struct {
	data map[string]any
}

func newMockCache() *mockCache {
	return &mockCache{data: make(map[string]any)}
}

func (m *mockCache) Get(_ context.Context, key string) (any, bool, error) {
	v, ok := m.data[key]
	return v, ok, nil
}

func (m *mockCache) Set(_ context.Context, key string, value any, ttl time.Duration) error {
	m.data[key] = value
	return nil
}

func (m *mockCache) Delete(_ context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockCache) Clear(_ context.Context) error {
	m.data = make(map[string]any)
	return nil
}

func TestNewSemanticCache(t *testing.T) {
	mc := newMockCache()
	sc := NewSemanticCache(mc, 0.9)

	if sc == nil {
		t.Fatal("NewSemanticCache() returned nil")
	}
}

func TestNewSemanticCache_ThresholdClamping(t *testing.T) {
	mc := newMockCache()

	tests := []struct {
		name      string
		threshold float64
		want      float64
	}{
		{"normal", 0.9, 0.9},
		{"zero", 0.0, 0.0},
		{"one", 1.0, 1.0},
		{"negative_clamped_to_zero", -0.5, 0.0},
		{"above_one_clamped", 1.5, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := NewSemanticCache(mc, tt.threshold)
			if sc.threshold != tt.want {
				t.Errorf("threshold = %v, want %v", sc.threshold, tt.want)
			}
		})
	}
}

func TestSemanticCache_SetAndGetSemantic(t *testing.T) {
	mc := newMockCache()
	sc := NewSemanticCache(mc, 0.9)
	ctx := context.Background()

	embedding := []float32{0.1, 0.2, 0.3}

	// Store a value.
	if err := sc.SetSemantic(ctx, embedding, "result1"); err != nil {
		t.Fatalf("SetSemantic() error = %v", err)
	}

	// Retrieve with the same embedding.
	val, ok, err := sc.GetSemantic(ctx, embedding, 0)
	if err != nil {
		t.Fatalf("GetSemantic() error = %v", err)
	}
	if !ok {
		t.Fatal("GetSemantic() ok = false, want true")
	}
	if val != "result1" {
		t.Errorf("GetSemantic() = %v, want %q", val, "result1")
	}
}

func TestSemanticCache_GetSemantic_Miss(t *testing.T) {
	mc := newMockCache()
	sc := NewSemanticCache(mc, 0.9)
	ctx := context.Background()

	embedding := []float32{0.1, 0.2, 0.3}

	val, ok, err := sc.GetSemantic(ctx, embedding, 0)
	if err != nil {
		t.Fatalf("GetSemantic() error = %v", err)
	}
	if ok {
		t.Error("GetSemantic() ok = true, want false for missing embedding")
	}
	if val != nil {
		t.Errorf("GetSemantic() = %v, want nil", val)
	}
}

func TestSemanticCache_DifferentEmbeddings(t *testing.T) {
	mc := newMockCache()
	sc := NewSemanticCache(mc, 0.9)
	ctx := context.Background()

	emb1 := []float32{0.1, 0.2, 0.3}
	emb2 := []float32{0.4, 0.5, 0.6}

	_ = sc.SetSemantic(ctx, emb1, "value1")
	_ = sc.SetSemantic(ctx, emb2, "value2")

	// Each embedding should return its own value.
	val1, ok1, _ := sc.GetSemantic(ctx, emb1, 0)
	val2, ok2, _ := sc.GetSemantic(ctx, emb2, 0)

	if !ok1 || val1 != "value1" {
		t.Errorf("emb1: ok=%v, val=%v, want ok=true, val=value1", ok1, val1)
	}
	if !ok2 || val2 != "value2" {
		t.Errorf("emb2: ok=%v, val=%v, want ok=true, val=value2", ok2, val2)
	}
}

func TestSemanticCache_GetSemantic_ThresholdOverride(t *testing.T) {
	mc := newMockCache()
	sc := NewSemanticCache(mc, 0.9)
	ctx := context.Background()

	embedding := []float32{1.0, 2.0}
	_ = sc.SetSemantic(ctx, embedding, "data")

	// With threshold override > 0, the override is used (but in placeholder impl,
	// it's still an exact hash match).
	val, ok, err := sc.GetSemantic(ctx, embedding, 0.5)
	if err != nil {
		t.Fatalf("GetSemantic() error = %v", err)
	}
	if !ok {
		t.Fatal("GetSemantic() ok = false, want true")
	}
	if val != "data" {
		t.Errorf("GetSemantic() = %v, want %q", val, "data")
	}
}

func TestSemanticCache_GetSemantic_ZeroThresholdUsesDefault(t *testing.T) {
	mc := newMockCache()
	sc := NewSemanticCache(mc, 0.95)
	ctx := context.Background()

	embedding := []float32{1.0, 2.0, 3.0}
	_ = sc.SetSemantic(ctx, embedding, "cached")

	// threshold=0 should use the default (0.95).
	val, ok, err := sc.GetSemantic(ctx, embedding, 0)
	if err != nil {
		t.Fatalf("GetSemantic() error = %v", err)
	}
	if !ok {
		t.Fatal("GetSemantic() ok = false, want true")
	}
	if val != "cached" {
		t.Errorf("GetSemantic() = %v, want %q", val, "cached")
	}
}

func TestSemanticCache_GetSemantic_NegativeThresholdUsesDefault(t *testing.T) {
	mc := newMockCache()
	sc := NewSemanticCache(mc, 0.8)
	ctx := context.Background()

	embedding := []float32{0.5}
	_ = sc.SetSemantic(ctx, embedding, "neg_test")

	// threshold=-1 should use default.
	val, ok, err := sc.GetSemantic(ctx, embedding, -1)
	if err != nil {
		t.Fatalf("GetSemantic() error = %v", err)
	}
	if !ok {
		t.Fatal("GetSemantic() ok = false, want true")
	}
	if val != "neg_test" {
		t.Errorf("GetSemantic() = %v, want %q", val, "neg_test")
	}
}

func TestSemanticCache_Cache(t *testing.T) {
	mc := newMockCache()
	sc := NewSemanticCache(mc, 0.9)

	underlying := sc.Cache()
	if underlying != mc {
		t.Error("Cache() did not return the underlying cache")
	}
}

func TestSemanticCache_SetSemantic_OverwritesSameEmbedding(t *testing.T) {
	mc := newMockCache()
	sc := NewSemanticCache(mc, 0.9)
	ctx := context.Background()

	embedding := []float32{1.0, 2.0}

	_ = sc.SetSemantic(ctx, embedding, "first")
	_ = sc.SetSemantic(ctx, embedding, "second")

	val, ok, err := sc.GetSemantic(ctx, embedding, 0)
	if err != nil {
		t.Fatalf("GetSemantic() error = %v", err)
	}
	if !ok {
		t.Fatal("GetSemantic() ok = false, want true")
	}
	if val != "second" {
		t.Errorf("GetSemantic() = %v, want %q (should be overwritten)", val, "second")
	}
}

func TestSemanticCache_EmptyEmbedding(t *testing.T) {
	mc := newMockCache()
	sc := NewSemanticCache(mc, 0.9)
	ctx := context.Background()

	embedding := []float32{}

	_ = sc.SetSemantic(ctx, embedding, "empty_emb")
	val, ok, err := sc.GetSemantic(ctx, embedding, 0)
	if err != nil {
		t.Fatalf("GetSemantic() error = %v", err)
	}
	if !ok {
		t.Fatal("GetSemantic() ok = false, want true")
	}
	if val != "empty_emb" {
		t.Errorf("GetSemantic() = %v, want %q", val, "empty_emb")
	}
}

func TestSemanticCache_DifferentValueTypes(t *testing.T) {
	mc := newMockCache()
	sc := NewSemanticCache(mc, 0.9)
	ctx := context.Background()

	tests := []struct {
		name      string
		embedding []float32
		value     any
	}{
		{"string", []float32{1.0}, "hello"},
		{"int", []float32{2.0}, 42},
		{"float", []float32{3.0}, 3.14},
		{"nil", []float32{5.0}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := sc.SetSemantic(ctx, tt.embedding, tt.value); err != nil {
				t.Fatalf("SetSemantic() error = %v", err)
			}
			val, ok, err := sc.GetSemantic(ctx, tt.embedding, 0)
			if err != nil {
				t.Fatalf("GetSemantic() error = %v", err)
			}
			if !ok {
				t.Fatal("GetSemantic() ok = false, want true")
			}
			if tt.value == nil {
				if val != nil {
					t.Errorf("GetSemantic() = %v, want nil", val)
				}
				return
			}
			if val != tt.value {
				t.Errorf("GetSemantic() = %v, want %v", val, tt.value)
			}
		})
	}
}

func TestEmbeddingKey_Deterministic(t *testing.T) {
	emb := []float32{0.1, 0.2, 0.3}

	key1 := embeddingKey(emb)
	key2 := embeddingKey(emb)

	if key1 != key2 {
		t.Errorf("embeddingKey not deterministic: %q != %q", key1, key2)
	}
}

func TestEmbeddingKey_DifferentEmbeddings(t *testing.T) {
	emb1 := []float32{0.1, 0.2, 0.3}
	emb2 := []float32{0.3, 0.2, 0.1}

	key1 := embeddingKey(emb1)
	key2 := embeddingKey(emb2)

	if key1 == key2 {
		t.Error("different embeddings should produce different keys")
	}
}

func TestEmbeddingKey_Prefix(t *testing.T) {
	emb := []float32{1.0}
	key := embeddingKey(emb)

	if len(key) < 4 || key[:4] != "sem:" {
		t.Errorf("embeddingKey() = %q, want prefix 'sem:'", key)
	}
}

func TestEmbeddingKey_Empty(t *testing.T) {
	emb := []float32{}
	key := embeddingKey(emb)

	// Even empty embedding should produce a valid key with the prefix.
	if len(key) < 4 || key[:4] != "sem:" {
		t.Errorf("embeddingKey([]) = %q, want prefix 'sem:'", key)
	}
}
