package colbert

import (
	"context"
	"sync"
	"testing"
)

func TestInMemoryIndex_Add(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name      string
		id        string
		tokenVecs [][]float32
		wantErr   bool
	}{
		{
			name:      "valid add",
			id:        "doc1",
			tokenVecs: [][]float32{{1, 0, 0}, {0, 1, 0}},
		},
		{
			name:      "empty token vecs slice",
			id:        "doc2",
			tokenVecs: [][]float32{},
		},
		{
			name:    "empty id",
			id:      "",
			wantErr: true,
		},
		{
			name:    "nil token vecs",
			id:      "doc3",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			idx := NewInMemoryIndex()
			err := idx.Add(ctx, tt.id, tt.tokenVecs)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if idx.Len() != 1 {
				t.Errorf("Len() = %d, want 1", idx.Len())
			}
		})
	}
}

func TestInMemoryIndex_AddReplace(t *testing.T) {
	ctx := context.Background()
	idx := NewInMemoryIndex()

	if err := idx.Add(ctx, "doc1", [][]float32{{1, 0}}); err != nil {
		t.Fatal(err)
	}
	if err := idx.Add(ctx, "doc1", [][]float32{{0, 1}}); err != nil {
		t.Fatal(err)
	}
	if idx.Len() != 1 {
		t.Errorf("Len() = %d after replace, want 1", idx.Len())
	}
}

func TestInMemoryIndex_AddDeepCopy(t *testing.T) {
	ctx := context.Background()
	idx := NewInMemoryIndex()

	vecs := [][]float32{{1, 2, 3}}
	if err := idx.Add(ctx, "doc1", vecs); err != nil {
		t.Fatal(err)
	}

	// Mutate the original — should not affect stored data.
	vecs[0][0] = 999

	results, err := idx.Search(ctx, [][]float32{{1, 2, 3}}, 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	// Score should reflect the original {1,2,3} not the mutated {999,2,3}.
	resultsOriginal, _ := idx.Search(ctx, [][]float32{{999, 2, 3}}, 1)
	if results[0].Score <= resultsOriginal[0].Score {
		t.Error("deep copy failed: mutation affected stored data")
	}
}

func TestInMemoryIndex_Search(t *testing.T) {
	ctx := context.Background()
	idx := NewInMemoryIndex()

	// Add three documents with distinct token embeddings.
	_ = idx.Add(ctx, "docA", [][]float32{{1, 0, 0}})
	_ = idx.Add(ctx, "docB", [][]float32{{0, 1, 0}})
	_ = idx.Add(ctx, "docC", [][]float32{{0, 0, 1}})

	tests := []struct {
		name     string
		query    [][]float32
		k        int
		wantIDs  []string
		wantLen  int
		wantZero bool
	}{
		{
			name:    "exact match returns top doc",
			query:   [][]float32{{1, 0, 0}},
			k:       1,
			wantIDs: []string{"docA"},
			wantLen: 1,
		},
		{
			name:    "top 2",
			query:   [][]float32{{1, 0, 0}},
			k:       2,
			wantLen: 2,
		},
		{
			name:    "k larger than index",
			query:   [][]float32{{1, 0, 0}},
			k:       10,
			wantLen: 3,
		},
		{
			name:     "k zero returns nil",
			query:    [][]float32{{1, 0, 0}},
			k:        0,
			wantZero: true,
		},
		{
			name:     "k negative returns nil",
			query:    [][]float32{{1, 0, 0}},
			k:        -1,
			wantZero: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			results, err := idx.Search(ctx, tt.query, tt.k)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantZero {
				if results != nil {
					t.Errorf("expected nil results, got %v", results)
				}
				return
			}
			if len(results) != tt.wantLen {
				t.Fatalf("got %d results, want %d", len(results), tt.wantLen)
			}
			for i, wantID := range tt.wantIDs {
				if results[i].ID != wantID {
					t.Errorf("result[%d].ID = %q, want %q", i, results[i].ID, wantID)
				}
			}
		})
	}
}

func TestInMemoryIndex_SearchOrdering(t *testing.T) {
	ctx := context.Background()
	idx := NewInMemoryIndex()

	_ = idx.Add(ctx, "best", [][]float32{{1, 0, 0}})
	_ = idx.Add(ctx, "mid", [][]float32{{0.7, 0.7, 0}})
	_ = idx.Add(ctx, "worst", [][]float32{{0, 1, 0}})

	results, err := idx.Search(ctx, [][]float32{{1, 0, 0}}, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 3 {
		t.Fatalf("got %d results, want 3", len(results))
	}
	// Scores should be in descending order.
	for i := 1; i < len(results); i++ {
		if results[i].Score > results[i-1].Score {
			t.Errorf("results not in descending order: [%d].Score=%f > [%d].Score=%f",
				i, results[i].Score, i-1, results[i-1].Score)
		}
	}
	if results[0].ID != "best" {
		t.Errorf("best match ID = %q, want %q", results[0].ID, "best")
	}
}

func TestInMemoryIndex_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	idx := NewInMemoryIndex()

	// Add should fail with cancelled context.
	if err := idx.Add(ctx, "doc1", [][]float32{{1, 0}}); err == nil {
		t.Error("expected error from cancelled context on Add")
	}

	// Search should fail with cancelled context.
	if _, err := idx.Search(ctx, [][]float32{{1, 0}}, 1); err == nil {
		t.Error("expected error from cancelled context on Search")
	}
}

func TestInMemoryIndex_ConcurrentAccess(t *testing.T) {
	ctx := context.Background()
	idx := NewInMemoryIndex()

	var wg sync.WaitGroup
	// Concurrent writes.
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			id := string(rune('A' + i%26))
			_ = idx.Add(ctx, id, [][]float32{{float32(i), 0, 0}})
		}(i)
	}
	// Concurrent reads.
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = idx.Search(ctx, [][]float32{{1, 0, 0}}, 5)
		}()
	}
	wg.Wait()
}
