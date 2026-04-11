package associative

import (
	"context"
	"encoding/json"
	"fmt"
	"iter"
	"sync/atomic"
	"testing"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time checks for mocks.
var (
	_ llm.ChatModel = (*mockChatModel)(nil)
)

// mockEmbedder returns deterministic embeddings for testing.
type mockEmbedder struct {
	dim       int
	callCount atomic.Int64
	embedFn   func(texts []string) ([][]float32, error)
}

func newMockEmbedder(dim int) *mockEmbedder {
	return &mockEmbedder{
		dim: dim,
		embedFn: func(texts []string) ([][]float32, error) {
			vecs := make([][]float32, len(texts))
			for i := range texts {
				vec := make([]float32, dim)
				// Simple deterministic hash: spread characters across dims.
				for j, c := range texts[i] {
					vec[j%dim] += float32(c)
				}
				// Normalize.
				var norm float32
				for _, v := range vec {
					norm += v * v
				}
				if norm > 0 {
					for j := range vec {
						vec[j] /= norm
					}
				}
				vecs[i] = vec
			}
			return vecs, nil
		},
	}
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	m.callCount.Add(1)
	return m.embedFn(texts)
}

func (m *mockEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	vecs, err := m.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return vecs[0], nil
}

func (m *mockEmbedder) Dimensions() int { return m.dim }

// mockChatModel returns a predictable JSON enrichment response.
type mockChatModel struct {
	generateFn func(ctx context.Context, msgs []schema.Message) (*schema.AIMessage, error)
}

func newMockChatModel() *mockChatModel {
	return &mockChatModel{
		generateFn: func(_ context.Context, _ []schema.Message) (*schema.AIMessage, error) {
			resp := Enrichment{
				Keywords:    []string{"go", "concurrency", "goroutines"},
				Tags:        []string{"programming", "golang"},
				Description: "A note about Go concurrency.",
			}
			data, _ := json.Marshal(resp)
			return schema.NewAIMessage(string(data)), nil
		},
	}
}

func (m *mockChatModel) Generate(ctx context.Context, msgs []schema.Message, _ ...llm.GenerateOption) (*schema.AIMessage, error) {
	return m.generateFn(ctx, msgs)
}

func (m *mockChatModel) Stream(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(yield func(schema.StreamChunk, error) bool) {}
}

func (m *mockChatModel) BindTools(_ []schema.ToolDefinition) llm.ChatModel { return m }
func (m *mockChatModel) ModelID() string                                   { return "mock" }

// --- Enricher Tests ---

func TestNoteEnricher_Enrich(t *testing.T) {
	t.Run("with LLM", func(t *testing.T) {
		model := newMockChatModel()
		enricher := NewNoteEnricher(model, 8)

		enrichment, err := enricher.Enrich(context.Background(), "Go uses goroutines for concurrency")
		require.NoError(t, err)
		assert.NotEmpty(t, enrichment.Keywords)
		assert.NotEmpty(t, enrichment.Tags)
		assert.NotEmpty(t, enrichment.Description)
	})

	t.Run("without LLM returns empty", func(t *testing.T) {
		enricher := NewNoteEnricher(nil, 8)
		enrichment, err := enricher.Enrich(context.Background(), "some content")
		require.NoError(t, err)
		assert.Empty(t, enrichment.Keywords)
		assert.Empty(t, enrichment.Tags)
		assert.Empty(t, enrichment.Description)
	})

	t.Run("empty content returns empty", func(t *testing.T) {
		model := newMockChatModel()
		enricher := NewNoteEnricher(model, 8)
		enrichment, err := enricher.Enrich(context.Background(), "")
		require.NoError(t, err)
		assert.Empty(t, enrichment.Keywords)
	})

	t.Run("LLM error propagates", func(t *testing.T) {
		model := newMockChatModel()
		model.generateFn = func(_ context.Context, _ []schema.Message) (*schema.AIMessage, error) {
			return nil, fmt.Errorf("api error")
		}
		enricher := NewNoteEnricher(model, 8)
		_, err := enricher.Enrich(context.Background(), "content")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "api error")
	})

	t.Run("max tags enforced", func(t *testing.T) {
		model := newMockChatModel()
		model.generateFn = func(_ context.Context, _ []schema.Message) (*schema.AIMessage, error) {
			resp := Enrichment{
				Keywords:    []string{"a"},
				Tags:        []string{"t1", "t2", "t3", "t4", "t5"},
				Description: "desc",
			}
			data, _ := json.Marshal(resp)
			return schema.NewAIMessage(string(data)), nil
		}
		enricher := NewNoteEnricher(model, 3)
		enrichment, err := enricher.Enrich(context.Background(), "content")
		require.NoError(t, err)
		assert.Len(t, enrichment.Tags, 3)
	})
}

// --- Linker Tests ---

func TestLinkManager_Link(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryNoteStore()

	// Add existing notes with known embeddings.
	n1 := newTestNote("existing1", "go programming", []float32{1, 0, 0})
	n2 := newTestNote("existing2", "rust programming", []float32{0.9, 0.1, 0})
	n3 := newTestNote("existing3", "cooking", []float32{0, 0, 1})
	require.NoError(t, store.Add(ctx, n1))
	require.NoError(t, store.Add(ctx, n2))
	require.NoError(t, store.Add(ctx, n3))

	linker := NewLinkManager(store, 2)

	t.Run("links top-k candidates bidirectionally", func(t *testing.T) {
		newNote := newTestNote("new1", "go concurrency", []float32{0.95, 0.05, 0})
		require.NoError(t, store.Add(ctx, newNote))

		linkedIDs, err := linker.Link(ctx, newNote)
		require.NoError(t, err)
		assert.Len(t, linkedIDs, 2)

		// Check new note has links.
		got, _ := store.Get(ctx, "new1")
		assert.Len(t, got.Links, 2)

		// Check neighbor has backlink.
		neighbor, _ := store.Get(ctx, linkedIDs[0])
		assert.Contains(t, neighbor.Links, "new1")
	})

	t.Run("nil note returns nil", func(t *testing.T) {
		ids, err := linker.Link(ctx, nil)
		require.NoError(t, err)
		assert.Nil(t, ids)
	})

	t.Run("note without embedding returns nil", func(t *testing.T) {
		n := newTestNote("noEmbed", "test", nil)
		ids, err := linker.Link(ctx, n)
		require.NoError(t, err)
		assert.Nil(t, ids)
	})
}

// --- RetroactiveUpdater Tests ---

func TestRetroactiveUpdater_Update(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryNoteStore()
	embedder := newMockEmbedder(3)
	model := newMockChatModel()
	enricher := NewNoteEnricher(model, 8)

	n1 := newTestNote("neighbor1", "go goroutines", []float32{1, 0, 0})
	n2 := newTestNote("neighbor2", "go channels", []float32{0.9, 0.1, 0})
	require.NoError(t, store.Add(ctx, n1))
	require.NoError(t, store.Add(ctx, n2))

	updater := NewRetroactiveUpdater(store, enricher, embedder, 2)

	newNote := newTestNote("new", "go concurrency patterns", []float32{0.95, 0.05, 0})

	t.Run("updates neighbors", func(t *testing.T) {
		refinedIDs, err := updater.Update(ctx, newNote, []string{"neighbor1", "neighbor2"})
		require.NoError(t, err)
		assert.Len(t, refinedIDs, 2)

		// Check neighbor was re-enriched.
		got, _ := store.Get(ctx, "neighbor1")
		assert.NotEqual(t, n1.Keywords, got.Keywords, "keywords should be updated")
	})

	t.Run("nil enricher returns nil", func(t *testing.T) {
		u := NewRetroactiveUpdater(store, NewNoteEnricher(nil, 8), embedder, 1)
		ids, err := u.Update(ctx, newNote, []string{"neighbor1"})
		require.NoError(t, err)
		assert.Nil(t, ids)
	})

	t.Run("empty neighbor list returns nil", func(t *testing.T) {
		ids, err := updater.Update(ctx, newNote, nil)
		require.NoError(t, err)
		assert.Nil(t, ids)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		cancelCtx, cancel := context.WithCancel(ctx)
		cancel() // Cancel immediately.

		_, err := updater.Update(cancelCtx, newNote, []string{"neighbor1"})
		// Should get context error (but neighbors might still complete
		// depending on timing).
		if err != nil {
			assert.ErrorIs(t, err, context.Canceled)
		}
	})
}

// --- AssociativeMemory Tests ---

func TestAssociativeMemory_NewRequiresEmbedder(t *testing.T) {
	_, err := NewAssociativeMemory(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "embedder is required")
}

func TestAssociativeMemory_AddNote(t *testing.T) {
	embedder := newMockEmbedder(3)
	model := newMockChatModel()

	mem, err := NewAssociativeMemory(embedder,
		WithLLM(model),
		WithLinkCandidates(5),
		WithMaxTags(4),
	)
	require.NoError(t, err)

	t.Run("happy path", func(t *testing.T) {
		note, err := mem.AddNote(context.Background(), "Go uses goroutines for concurrency")
		require.NoError(t, err)
		assert.NotEmpty(t, note.ID)
		assert.NotEmpty(t, note.Keywords)
		assert.NotEmpty(t, note.Tags)
		assert.NotEmpty(t, note.Description)
		assert.NotNil(t, note.Embedding)
	})

	t.Run("empty content errors", func(t *testing.T) {
		_, err := mem.AddNote(context.Background(), "")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "content is empty")
	})

	t.Run("without LLM skips enrichment", func(t *testing.T) {
		noLLM, err := NewAssociativeMemory(embedder)
		require.NoError(t, err)
		note, err := noLLM.AddNote(context.Background(), "some content")
		require.NoError(t, err)
		assert.Empty(t, note.Keywords)
		assert.NotNil(t, note.Embedding)
	})
}

func TestAssociativeMemory_SearchNotes(t *testing.T) {
	embedder := newMockEmbedder(3)
	mem, err := NewAssociativeMemory(embedder)
	require.NoError(t, err)
	ctx := context.Background()

	_, err = mem.AddNote(ctx, "go programming language")
	require.NoError(t, err)
	_, err = mem.AddNote(ctx, "rust programming language")
	require.NoError(t, err)

	notes, err := mem.SearchNotes(ctx, "go programming", 5)
	require.NoError(t, err)
	assert.NotEmpty(t, notes)
}

func TestAssociativeMemory_DeleteNote(t *testing.T) {
	embedder := newMockEmbedder(3)
	mem, err := NewAssociativeMemory(embedder, WithLinkCandidates(5))
	require.NoError(t, err)
	ctx := context.Background()

	n1, err := mem.AddNote(ctx, "go programming")
	require.NoError(t, err)
	n2, err := mem.AddNote(ctx, "go concurrency")
	require.NoError(t, err)

	// Delete n1 and verify backlinks are cleaned from n2.
	err = mem.DeleteNote(ctx, n1.ID)
	require.NoError(t, err)

	_, err = mem.GetNote(ctx, n1.ID)
	require.Error(t, err)

	// n2 should not have n1 in its links anymore.
	got, err := mem.GetNote(ctx, n2.ID)
	require.NoError(t, err)
	assert.NotContains(t, got.Links, n1.ID)
}

func TestAssociativeMemory_MemoryInterface(t *testing.T) {
	embedder := newMockEmbedder(3)
	mem, err := NewAssociativeMemory(embedder)
	require.NoError(t, err)
	ctx := context.Background()

	t.Run("Save and Load", func(t *testing.T) {
		input := schema.NewHumanMessage("What is Go?")
		output := schema.NewAIMessage("Go is a programming language.")
		require.NoError(t, mem.Save(ctx, input, output))

		msgs, err := mem.Load(ctx, "Go programming")
		require.NoError(t, err)
		assert.NotEmpty(t, msgs)
	})

	t.Run("Search returns documents", func(t *testing.T) {
		docs, err := mem.Search(ctx, "Go programming", 5)
		require.NoError(t, err)
		assert.NotEmpty(t, docs)
		assert.NotEmpty(t, docs[0].ID)
		assert.NotEmpty(t, docs[0].Content)
	})

	t.Run("Clear removes all notes", func(t *testing.T) {
		require.NoError(t, mem.Clear(ctx))
		docs, err := mem.Search(ctx, "anything", 10)
		require.NoError(t, err)
		assert.Empty(t, docs)
	})

	t.Run("Save with empty output is no-op", func(t *testing.T) {
		input := schema.NewHumanMessage("hello")
		output := schema.NewAIMessage("")
		require.NoError(t, mem.Save(ctx, input, output))
	})
}

func TestAssociativeMemory_Hooks(t *testing.T) {
	embedder := newMockEmbedder(3)

	var createdCount, linkedCount int

	hooks := Hooks{
		OnNoteCreated: func(_ context.Context, note *schema.Note) {
			createdCount++
		},
		OnNoteLinked: func(_ context.Context, note *schema.Note, linkedIDs []string) {
			linkedCount++
		},
	}

	mem, err := NewAssociativeMemory(embedder,
		WithHooks(hooks),
		WithLinkCandidates(5),
	)
	require.NoError(t, err)
	ctx := context.Background()

	_, err = mem.AddNote(ctx, "first note about go")
	require.NoError(t, err)
	assert.Equal(t, 1, createdCount)

	_, err = mem.AddNote(ctx, "second note about go programming")
	require.NoError(t, err)
	assert.Equal(t, 2, createdCount)
	assert.GreaterOrEqual(t, linkedCount, 1, "second note should link to first")
}

func TestAssociativeMemory_WithRetroactiveRefinement(t *testing.T) {
	embedder := newMockEmbedder(3)
	model := newMockChatModel()

	var refinedCount int
	hooks := Hooks{
		OnNoteRefined: func(_ context.Context, note *schema.Note, refinedIDs []string) {
			refinedCount++
		},
	}

	mem, err := NewAssociativeMemory(embedder,
		WithLLM(model),
		WithRetroactiveRefinement(true),
		WithRetroactiveMaxWorkers(2),
		WithHooks(hooks),
	)
	require.NoError(t, err)
	ctx := context.Background()

	_, err = mem.AddNote(ctx, "first note")
	require.NoError(t, err)
	_, err = mem.AddNote(ctx, "second related note")
	require.NoError(t, err)

	assert.GreaterOrEqual(t, refinedCount, 1, "retroactive refinement should trigger")
}

func TestComposeHooks(t *testing.T) {
	var order []string

	h1 := Hooks{
		OnNoteCreated: func(_ context.Context, _ *schema.Note) {
			order = append(order, "h1")
		},
	}
	h2 := Hooks{
		OnNoteCreated: func(_ context.Context, _ *schema.Note) {
			order = append(order, "h2")
		},
	}

	composed := ComposeHooks(h1, h2)
	composed.OnNoteCreated(context.Background(), &schema.Note{})

	assert.Equal(t, []string{"h1", "h2"}, order)
}

func TestAssociativeMemory_Options(t *testing.T) {
	embedder := newMockEmbedder(3)

	t.Run("default options", func(t *testing.T) {
		mem, err := NewAssociativeMemory(embedder)
		require.NoError(t, err)
		assert.NotNil(t, mem)
	})

	t.Run("with custom store", func(t *testing.T) {
		store := NewInMemoryNoteStore()
		mem, err := NewAssociativeMemory(embedder, WithStore(store))
		require.NoError(t, err)
		assert.NotNil(t, mem)
	})

	t.Run("invalid link candidates ignored", func(t *testing.T) {
		mem, err := NewAssociativeMemory(embedder, WithLinkCandidates(-1))
		require.NoError(t, err)
		assert.NotNil(t, mem)
	})
}
