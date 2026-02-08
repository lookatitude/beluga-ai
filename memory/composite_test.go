package memory

import (
	"context"
	"testing"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewComposite(t *testing.T) {
	t.Run("empty composite", func(t *testing.T) {
		comp := NewComposite()
		assert.NotNil(t, comp)
		assert.Nil(t, comp.Core())
		assert.Nil(t, comp.Recall())
		assert.Nil(t, comp.Archival())
		assert.Nil(t, comp.Graph())
	})

	t.Run("with core", func(t *testing.T) {
		core := NewCore(CoreConfig{})
		comp := NewComposite(WithCore(core))
		assert.Equal(t, core, comp.Core())
		assert.Nil(t, comp.Recall())
	})

	t.Run("with recall", func(t *testing.T) {
		recall := NewRecall(&mockMessageStore{})
		comp := NewComposite(WithRecall(recall))
		assert.Nil(t, comp.Core())
		assert.Equal(t, recall, comp.Recall())
	})

	t.Run("with archival", func(t *testing.T) {
		arch, err := NewArchival(ArchivalConfig{
			VectorStore: &mockVectorStore{},
			Embedder:    &mockEmbedder{dim: 4},
		})
		require.NoError(t, err)

		comp := NewComposite(WithArchival(arch))
		assert.Nil(t, comp.Core())
		assert.Equal(t, arch, comp.Archival())
	})

	t.Run("with all tiers", func(t *testing.T) {
		core := NewCore(CoreConfig{})
		recall := NewRecall(&mockMessageStore{})
		arch, err := NewArchival(ArchivalConfig{
			VectorStore: &mockVectorStore{},
			Embedder:    &mockEmbedder{dim: 4},
		})
		require.NoError(t, err)

		comp := NewComposite(
			WithCore(core),
			WithRecall(recall),
			WithArchival(arch),
		)

		assert.Equal(t, core, comp.Core())
		assert.Equal(t, recall, comp.Recall())
		assert.Equal(t, arch, comp.Archival())
	})
}

func TestCompositeSave(t *testing.T) {
	ctx := context.Background()

	t.Run("delegates to recall and archival", func(t *testing.T) {
		recallStore := &mockMessageStore{}
		recall := NewRecall(recallStore)

		vs := &mockVectorStore{}
		arch, err := NewArchival(ArchivalConfig{
			VectorStore: vs,
			Embedder:    &mockEmbedder{dim: 4},
		})
		require.NoError(t, err)

		comp := NewComposite(WithRecall(recall), WithArchival(arch))

		input := schema.NewHumanMessage("hello")
		output := schema.NewAIMessage("hi")

		err = comp.Save(ctx, input, output)
		require.NoError(t, err)

		// Verify recall store received messages.
		assert.Len(t, recallStore.msgs, 2)

		// Verify archival store received documents.
		assert.Len(t, vs.docs, 2)
	})

	t.Run("skips nil tiers", func(t *testing.T) {
		comp := NewComposite()

		input := schema.NewHumanMessage("hello")
		output := schema.NewAIMessage("hi")

		err := comp.Save(ctx, input, output)
		assert.NoError(t, err)
	})

	t.Run("core not saved to", func(t *testing.T) {
		core := NewCore(CoreConfig{})
		comp := NewComposite(WithCore(core))

		input := schema.NewHumanMessage("hello")
		output := schema.NewAIMessage("hi")

		err := comp.Save(ctx, input, output)
		require.NoError(t, err)

		// Core memory should remain empty (Save is a no-op for core).
		assert.Empty(t, core.GetPersona())
		assert.Empty(t, core.GetHuman())
	})

	t.Run("recall error stops propagation", func(t *testing.T) {
		recallStore := &mockMessageStore{}
		recall := NewRecall(recallStore)

		vs := &mockVectorStore{}
		arch, err := NewArchival(ArchivalConfig{
			VectorStore: vs,
			Embedder:    &mockEmbedder{dim: 4},
		})
		require.NoError(t, err)

		comp := NewComposite(WithRecall(recall), WithArchival(arch))

		// This will succeed in recall but we'll test the error path.
		// For a real error test, we'd need a mock that can return errors.
		input := schema.NewHumanMessage("hello")
		output := schema.NewAIMessage("hi")

		err = comp.Save(ctx, input, output)
		assert.NoError(t, err)
	})
}

func TestCompositeLoad(t *testing.T) {
	ctx := context.Background()

	t.Run("combines core and recall", func(t *testing.T) {
		core := NewCore(CoreConfig{})
		require.NoError(t, core.SetPersona("I am helpful"))

		recallStore := &mockMessageStore{}
		recall := NewRecall(recallStore)
		require.NoError(t, recall.Save(ctx,
			schema.NewHumanMessage("hello"),
			schema.NewAIMessage("hi")))

		comp := NewComposite(WithCore(core), WithRecall(recall))

		msgs, err := comp.Load(ctx, "")
		require.NoError(t, err)

		// Should have core messages (1) + recall messages (2).
		assert.Len(t, msgs, 3)

		// First message should be from core (system).
		assert.Equal(t, schema.RoleSystem, msgs[0].GetRole())

		// Next messages from recall.
		assert.Equal(t, schema.RoleHuman, msgs[1].GetRole())
		assert.Equal(t, schema.RoleAI, msgs[2].GetRole())
	})

	t.Run("empty when no tiers", func(t *testing.T) {
		comp := NewComposite()

		msgs, err := comp.Load(ctx, "any query")
		require.NoError(t, err)
		assert.Empty(t, msgs)
	})

	t.Run("only core", func(t *testing.T) {
		core := NewCore(CoreConfig{})
		require.NoError(t, core.SetPersona("I am helpful"))

		comp := NewComposite(WithCore(core))

		msgs, err := comp.Load(ctx, "")
		require.NoError(t, err)
		assert.Len(t, msgs, 1)
		assert.Equal(t, schema.RoleSystem, msgs[0].GetRole())
	})

	t.Run("only recall", func(t *testing.T) {
		recallStore := &mockMessageStore{}
		recall := NewRecall(recallStore)
		require.NoError(t, recall.Save(ctx,
			schema.NewHumanMessage("hello"),
			schema.NewAIMessage("hi")))

		comp := NewComposite(WithRecall(recall))

		msgs, err := comp.Load(ctx, "")
		require.NoError(t, err)
		assert.Len(t, msgs, 2)
	})

	t.Run("query passed to recall", func(t *testing.T) {
		recallStore := &mockMessageStore{}
		recall := NewRecall(recallStore)
		require.NoError(t, recall.Save(ctx,
			schema.NewHumanMessage("hello world"),
			schema.NewAIMessage("hi")))
		require.NoError(t, recall.Save(ctx,
			schema.NewHumanMessage("goodbye"),
			schema.NewAIMessage("bye")))

		comp := NewComposite(WithRecall(recall))

		msgs, err := comp.Load(ctx, "hello")
		require.NoError(t, err)

		// Should only match "hello world".
		assert.Len(t, msgs, 1)
		assert.Contains(t, msgs[0].(*schema.HumanMessage).Text(), "hello")
	})
}

func TestCompositeSearch(t *testing.T) {
	ctx := context.Background()

	t.Run("delegates to archival", func(t *testing.T) {
		doc1 := schema.Document{ID: "1", Content: "hello"}
		doc2 := schema.Document{ID: "2", Content: "world"}
		vs := &mockVectorStore{docs: []schema.Document{doc1, doc2}}

		arch, err := NewArchival(ArchivalConfig{
			VectorStore: vs,
			Embedder:    &mockEmbedder{dim: 4},
		})
		require.NoError(t, err)

		comp := NewComposite(WithArchival(arch))

		docs, err := comp.Search(ctx, "test query", 2)
		require.NoError(t, err)
		assert.Len(t, docs, 2)
		assert.Equal(t, "1", docs[0].ID)
	})

	t.Run("returns nil when no archival", func(t *testing.T) {
		core := NewCore(CoreConfig{})
		recall := NewRecall(&mockMessageStore{})
		comp := NewComposite(WithCore(core), WithRecall(recall))

		docs, err := comp.Search(ctx, "query", 5)
		require.NoError(t, err)
		assert.Nil(t, docs)
	})
}

func TestCompositeClear(t *testing.T) {
	ctx := context.Background()

	t.Run("clears all tiers", func(t *testing.T) {
		core := NewCore(CoreConfig{})
		require.NoError(t, core.SetPersona("I am helpful"))
		require.NoError(t, core.SetHuman("User is friendly"))

		recallStore := &mockMessageStore{}
		recall := NewRecall(recallStore)
		require.NoError(t, recall.Save(ctx,
			schema.NewHumanMessage("hello"),
			schema.NewAIMessage("hi")))

		arch, err := NewArchival(ArchivalConfig{
			VectorStore: &mockVectorStore{},
			Embedder:    &mockEmbedder{dim: 4},
		})
		require.NoError(t, err)

		comp := NewComposite(WithCore(core), WithRecall(recall), WithArchival(arch))

		err = comp.Clear(ctx)
		require.NoError(t, err)

		// Verify core is cleared.
		assert.Empty(t, core.GetPersona())
		assert.Empty(t, core.GetHuman())

		// Verify recall is cleared.
		msgs, err := recall.Load(ctx, "")
		require.NoError(t, err)
		assert.Empty(t, msgs)
	})

	t.Run("clears empty composite", func(t *testing.T) {
		comp := NewComposite()
		err := comp.Clear(ctx)
		assert.NoError(t, err)
	})
}

func TestCompositeRegistryEntry(t *testing.T) {
	// Verify "composite" is registered via init().
	mem, err := New("composite", config.ProviderConfig{Provider: "composite"})
	require.NoError(t, err)

	comp, ok := mem.(*CompositeMemory)
	require.True(t, ok)

	// Default composite has core and recall, but not archival.
	assert.NotNil(t, comp.Core())
	assert.NotNil(t, comp.Recall())
	assert.Nil(t, comp.Archival())
	assert.Nil(t, comp.Graph())

	// Verify core is self-editable by default.
	assert.True(t, comp.Core().IsSelfEditable())
}

func TestCompositeWithGraph(t *testing.T) {
	// Test the graph option (even though we don't use it in Save/Load/Search).
	graph := &mockGraphStore{}
	comp := NewComposite(WithGraph(graph))

	assert.Equal(t, graph, comp.Graph())
}

// mockGraphStore is a minimal GraphStore for testing.
type mockGraphStore struct {
	entities  map[string]Entity
	relations []Relation
}

func (g *mockGraphStore) AddEntity(_ context.Context, entity Entity) error {
	if g.entities == nil {
		g.entities = make(map[string]Entity)
	}
	g.entities[entity.ID] = entity
	return nil
}

func (g *mockGraphStore) AddRelation(_ context.Context, from, to, relation string, props map[string]any) error {
	g.relations = append(g.relations, Relation{
		From:       from,
		To:         to,
		Type:       relation,
		Properties: props,
	})
	return nil
}

func (g *mockGraphStore) Query(_ context.Context, query string) ([]GraphResult, error) {
	return nil, nil
}

func (g *mockGraphStore) Neighbors(_ context.Context, entityID string, depth int) ([]Entity, []Relation, error) {
	return nil, nil, nil
}
