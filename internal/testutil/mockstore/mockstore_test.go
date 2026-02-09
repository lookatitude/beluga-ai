package mockstore

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name     string
		opts     []Option
		wantDocs int
		wantErr  error
	}{
		{
			name:     "default configuration",
			wantDocs: 0,
		},
		{
			name: "with documents",
			opts: []Option{WithDocuments([]schema.Document{
				{ID: "1", Content: "test"},
			})},
			wantDocs: 1,
		},
		{
			name:    "with error",
			opts:    []Option{WithError(errors.New("test error"))},
			wantErr: errors.New("test error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.opts...)
			require.NotNil(t, m)
			assert.Len(t, m.Documents(), tt.wantDocs)
			if tt.wantErr != nil {
				assert.NotNil(t, m.err)
			}
		})
	}
}

func TestMockVectorStore_Add(t *testing.T) {
	tests := []struct {
		name       string
		opts       []Option
		docs       []schema.Document
		embeddings [][]float32
		wantErr    bool
		wantStored int
	}{
		{
			name: "add single document",
			docs: []schema.Document{{ID: "1", Content: "hello"}},
			embeddings: [][]float32{{1.0, 2.0}},
			wantStored: 1,
		},
		{
			name: "add multiple documents",
			docs: []schema.Document{
				{ID: "1", Content: "hello"},
				{ID: "2", Content: "world"},
			},
			embeddings: [][]float32{{1.0, 2.0}, {3.0, 4.0}},
			wantStored: 2,
		},
		{
			name:       "error path",
			opts:       []Option{WithError(errors.New("add failed"))},
			docs:       []schema.Document{{ID: "1", Content: "test"}},
			embeddings: [][]float32{{1.0}},
			wantErr:    true,
		},
		{
			name:       "empty documents",
			docs:       []schema.Document{},
			embeddings: [][]float32{},
			wantStored: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.opts...)
			ctx := context.Background()

			err := m.Add(ctx, tt.docs, tt.embeddings)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, 1, m.AddCalls())
			assert.Equal(t, tt.docs, m.LastDocs())
			assert.Len(t, m.Documents(), tt.wantStored)
		})
	}
}

func TestMockVectorStore_Search(t *testing.T) {
	docs := []schema.Document{
		{ID: "1", Content: "first"},
		{ID: "2", Content: "second"},
		{ID: "3", Content: "third"},
	}

	tests := []struct {
		name      string
		opts      []Option
		query     []float32
		k         int
		wantCount int
		wantErr   bool
	}{
		{
			name:      "search with results",
			opts:      []Option{WithDocuments(docs)},
			query:     []float32{1.0, 2.0},
			k:         2,
			wantCount: 2,
		},
		{
			name:      "search all documents",
			opts:      []Option{WithDocuments(docs)},
			query:     []float32{1.0, 2.0},
			k:         10,
			wantCount: 3,
		},
		{
			name:      "search with k=1",
			opts:      []Option{WithDocuments(docs)},
			query:     []float32{1.0, 2.0},
			k:         1,
			wantCount: 1,
		},
		{
			name:      "search empty store",
			query:     []float32{1.0, 2.0},
			k:         5,
			wantCount: 0,
		},
		{
			name:    "error path",
			opts:    []Option{WithError(errors.New("search failed"))},
			query:   []float32{1.0, 2.0},
			k:       5,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.opts...)
			ctx := context.Background()

			result, err := m.Search(ctx, tt.query, tt.k)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, result, tt.wantCount)
			assert.Equal(t, 1, m.SearchCalls())
			assert.Equal(t, tt.query, m.LastQuery())
		})
	}
}

func TestMockVectorStore_Delete(t *testing.T) {
	docs := []schema.Document{
		{ID: "1", Content: "first"},
		{ID: "2", Content: "second"},
		{ID: "3", Content: "third"},
	}

	tests := []struct {
		name           string
		opts           []Option
		ids            []string
		wantErr        bool
		wantRemaining  int
	}{
		{
			name:          "delete single document",
			opts:          []Option{WithDocuments(docs)},
			ids:           []string{"2"},
			wantRemaining: 2,
		},
		{
			name:          "delete multiple documents",
			opts:          []Option{WithDocuments(docs)},
			ids:           []string{"1", "3"},
			wantRemaining: 1,
		},
		{
			name:          "delete non-existent ID",
			opts:          []Option{WithDocuments(docs)},
			ids:           []string{"999"},
			wantRemaining: 3,
		},
		{
			name:          "delete all documents",
			opts:          []Option{WithDocuments(docs)},
			ids:           []string{"1", "2", "3"},
			wantRemaining: 0,
		},
		{
			name:    "error path",
			opts:    []Option{WithError(errors.New("delete failed"))},
			ids:     []string{"1"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := New(tt.opts...)
			ctx := context.Background()

			err := m.Delete(ctx, tt.ids)

			if tt.wantErr {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, 1, m.DeleteCalls())
			assert.Equal(t, tt.ids, m.LastIDs())
			assert.Len(t, m.Documents(), tt.wantRemaining)
		})
	}
}

func TestMockVectorStore_WithAddFunc(t *testing.T) {
	called := false
	customFn := func(ctx context.Context, docs []schema.Document, embeddings [][]float32) error {
		called = true
		assert.Len(t, docs, 1)
		return nil
	}

	m := New(WithAddFunc(customFn))
	err := m.Add(context.Background(), []schema.Document{{ID: "1"}}, [][]float32{{1.0}})

	require.NoError(t, err)
	assert.True(t, called)
}

func TestMockVectorStore_WithSearchFunc(t *testing.T) {
	called := false
	customFn := func(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) {
		called = true
		return []schema.Document{{ID: "custom"}}, nil
	}

	m := New(WithSearchFunc(customFn))
	result, err := m.Search(context.Background(), []float32{1.0}, 5)

	require.NoError(t, err)
	assert.True(t, called)
	assert.Equal(t, []schema.Document{{ID: "custom"}}, result)
}

func TestMockVectorStore_WithDeleteFunc(t *testing.T) {
	called := false
	customFn := func(ctx context.Context, ids []string) error {
		called = true
		assert.Equal(t, []string{"1"}, ids)
		return nil
	}

	m := New(WithDeleteFunc(customFn))
	err := m.Delete(context.Background(), []string{"1"})

	require.NoError(t, err)
	assert.True(t, called)
}

func TestMockVectorStore_SetDocuments(t *testing.T) {
	m := New()
	ctx := context.Background()

	// Initial search returns empty
	result1, err := m.Search(ctx, []float32{1.0}, 5)
	require.NoError(t, err)
	assert.Empty(t, result1)

	// Set documents
	newDocs := []schema.Document{{ID: "1", Content: "test"}}
	m.SetDocuments(newDocs)
	result2, err := m.Search(ctx, []float32{1.0}, 5)
	require.NoError(t, err)
	assert.Equal(t, newDocs, result2)
}

func TestMockVectorStore_SetError(t *testing.T) {
	m := New()
	ctx := context.Background()

	// Initial calls succeed
	err := m.Add(ctx, []schema.Document{{ID: "1"}}, [][]float32{{1.0}})
	require.NoError(t, err)

	// Set error
	testErr := errors.New("new error")
	m.SetError(testErr)
	err = m.Add(ctx, []schema.Document{{ID: "2"}}, [][]float32{{2.0}})
	require.Error(t, err)
	assert.Equal(t, testErr, err)
}

func TestMockVectorStore_Reset(t *testing.T) {
	m := New(
		WithDocuments([]schema.Document{{ID: "1"}}),
		WithError(errors.New("test")),
	)

	// Make some calls
	_, _ = m.Search(context.Background(), []float32{1.0}, 5)
	_ = m.Add(context.Background(), []schema.Document{{ID: "2"}}, [][]float32{{2.0}})
	_ = m.Delete(context.Background(), []string{"1"})

	assert.Equal(t, 1, m.AddCalls())
	assert.Equal(t, 1, m.SearchCalls())
	assert.Equal(t, 1, m.DeleteCalls())

	// Reset
	m.Reset()
	assert.Equal(t, 0, m.AddCalls())
	assert.Equal(t, 0, m.SearchCalls())
	assert.Equal(t, 0, m.DeleteCalls())
	assert.Empty(t, m.Documents())
	assert.Empty(t, m.LastDocs())
	assert.Empty(t, m.LastQuery())
	assert.Empty(t, m.LastIDs())
	assert.Nil(t, m.err)
}

func TestMockVectorStore_Concurrency(t *testing.T) {
	m := New()
	ctx := context.Background()

	// Multiple goroutines calling methods concurrently
	const goroutines = 10
	done := make(chan bool, goroutines*3)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			_ = m.Add(ctx, []schema.Document{{ID: "test"}}, [][]float32{{1.0}})
			done <- true
		}(i)
		go func(id int) {
			_, _ = m.Search(ctx, []float32{1.0}, 5)
			done <- true
		}(i)
		go func(id int) {
			_ = m.Delete(ctx, []string{"test"})
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines*3; i++ {
		<-done
	}

	assert.Equal(t, goroutines, m.AddCalls())
	assert.Equal(t, goroutines, m.SearchCalls())
	assert.Equal(t, goroutines, m.DeleteCalls())
}

func TestMockVectorStore_AddThenSearch(t *testing.T) {
	m := New()
	ctx := context.Background()

	// Add documents
	docs := []schema.Document{
		{ID: "1", Content: "first"},
		{ID: "2", Content: "second"},
	}
	err := m.Add(ctx, docs, [][]float32{{1.0}, {2.0}})
	require.NoError(t, err)

	// Search returns added documents
	results, err := m.Search(ctx, []float32{1.0}, 10)
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestMockVectorStore_AddThenDelete(t *testing.T) {
	m := New()
	ctx := context.Background()

	// Add documents
	docs := []schema.Document{
		{ID: "1", Content: "first"},
		{ID: "2", Content: "second"},
		{ID: "3", Content: "third"},
	}
	err := m.Add(ctx, docs, [][]float32{{1.0}, {2.0}, {3.0}})
	require.NoError(t, err)
	assert.Len(t, m.Documents(), 3)

	// Delete one
	err = m.Delete(ctx, []string{"2"})
	require.NoError(t, err)
	assert.Len(t, m.Documents(), 2)

	// Verify remaining documents
	remaining := m.Documents()
	ids := make([]string, len(remaining))
	for i, doc := range remaining {
		ids[i] = doc.ID
	}
	assert.Contains(t, ids, "1")
	assert.Contains(t, ids, "3")
	assert.NotContains(t, ids, "2")
}
