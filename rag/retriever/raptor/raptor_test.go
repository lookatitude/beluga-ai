package raptor

import (
	"context"
	"fmt"
	"iter"
	"math"
	"testing"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/rag/retriever"
	"github.com/lookatitude/beluga-ai/schema"
)

// --- Mock Embedder ---

type mockEmbedder struct {
	embedFn    func(ctx context.Context, texts []string) ([][]float32, error)
	dimensions int
}

func (m *mockEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	if m.embedFn != nil {
		return m.embedFn(ctx, texts)
	}
	result := make([][]float32, len(texts))
	for i := range texts {
		result[i] = make([]float32, m.dimensions)
		// Simple deterministic embedding: hash-like.
		for j := 0; j < m.dimensions; j++ {
			if j < len(texts[i]) {
				result[i][j] = float32(texts[i][j]) / 255.0
			}
		}
	}
	return result, nil
}

func (m *mockEmbedder) EmbedSingle(ctx context.Context, text string) ([]float32, error) {
	results, err := m.Embed(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	return results[0], nil
}

func (m *mockEmbedder) Dimensions() int { return m.dimensions }

// --- Mock Summarizer ---

type mockSummarizer struct {
	summarizeFn func(ctx context.Context, texts []string) (string, error)
	calls       int
}

func (m *mockSummarizer) Summarize(ctx context.Context, texts []string) (string, error) {
	m.calls++
	if m.summarizeFn != nil {
		return m.summarizeFn(ctx, texts)
	}
	return fmt.Sprintf("Summary of %d texts", len(texts)), nil
}

// --- Mock Clusterer ---

type mockClusterer struct {
	clusterFn func(ctx context.Context, embeddings [][]float32) ([][]int, error)
}

func (m *mockClusterer) Cluster(ctx context.Context, embeddings [][]float32) ([][]int, error) {
	if m.clusterFn != nil {
		return m.clusterFn(ctx, embeddings)
	}
	// Default: one cluster with all indices.
	indices := make([]int, len(embeddings))
	for i := range indices {
		indices[i] = i
	}
	return [][]int{indices}, nil
}

// --- Mock ChatModel ---

type mockChatModel struct {
	generateFn func(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error)
}

func (m *mockChatModel) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	if m.generateFn != nil {
		return m.generateFn(ctx, msgs, opts...)
	}
	return schema.NewAIMessage("mock summary"), nil
}

func (m *mockChatModel) Stream(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(yield func(schema.StreamChunk, error) bool) {}
}

func (m *mockChatModel) BindTools(_ []schema.ToolDefinition) llm.ChatModel { return m }
func (m *mockChatModel) ModelID() string                                   { return "mock" }

// Compile-time check.
var _ llm.ChatModel = (*mockChatModel)(nil)

// =============================================================================
// Clustering Tests
// =============================================================================

func TestKMeansClusterer_Cluster(t *testing.T) {
	tests := []struct {
		name       string
		embeddings [][]float32
		k          int
		seed       uint64
		wantErr    bool
		checkFn    func(t *testing.T, clusters [][]int)
	}{
		{
			name:    "empty embeddings",
			k:       2,
			wantErr: true,
		},
		{
			name:       "single embedding",
			embeddings: [][]float32{{1.0, 2.0}},
			k:          2,
			checkFn: func(t *testing.T, clusters [][]int) {
				if len(clusters) != 1 {
					t.Errorf("got %d clusters, want 1", len(clusters))
				}
				if len(clusters[0]) != 1 || clusters[0][0] != 0 {
					t.Errorf("expected cluster with index 0, got %v", clusters[0])
				}
			},
		},
		{
			name: "two distinct groups",
			embeddings: [][]float32{
				{0.0, 0.0}, {0.1, 0.1}, {0.05, 0.05},
				{10.0, 10.0}, {10.1, 10.1}, {9.95, 9.95},
			},
			k:    2,
			seed: 42,
			checkFn: func(t *testing.T, clusters [][]int) {
				if len(clusters) != 2 {
					t.Errorf("got %d clusters, want 2", len(clusters))
					return
				}
				// Each cluster should have 3 elements.
				for i, c := range clusters {
					if len(c) != 3 {
						t.Errorf("cluster %d has %d elements, want 3", i, len(c))
					}
				}
				// Verify all indices are covered.
				seen := make(map[int]bool)
				for _, c := range clusters {
					for _, idx := range c {
						seen[idx] = true
					}
				}
				if len(seen) != 6 {
					t.Errorf("expected 6 unique indices, got %d", len(seen))
				}
			},
		},
		{
			name: "auto K estimation",
			embeddings: func() [][]float32 {
				e := make([][]float32, 20)
				for i := range e {
					e[i] = []float32{float32(i), float32(i * 2)}
				}
				return e
			}(),
			k:    0, // auto
			seed: 42,
			checkFn: func(t *testing.T, clusters [][]int) {
				if len(clusters) < 2 {
					t.Errorf("expected at least 2 clusters, got %d", len(clusters))
				}
				total := 0
				for _, c := range clusters {
					total += len(c)
				}
				if total != 20 {
					t.Errorf("total elements = %d, want 20", total)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &KMeansClusterer{K: tt.k, Seed: tt.seed}
			clusters, err := c.Cluster(context.Background(), tt.embeddings)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Cluster() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if tt.checkFn != nil {
				tt.checkFn(t, clusters)
			}
		})
	}
}

func TestKMeansClusterer_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	embeddings := make([][]float32, 100)
	for i := range embeddings {
		embeddings[i] = []float32{float32(i), float32(i)}
	}

	c := &KMeansClusterer{K: 5}
	_, err := c.Cluster(ctx, embeddings)
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

// =============================================================================
// Summarizer Tests
// =============================================================================

func TestLLMSummarizer_Summarize(t *testing.T) {
	tests := []struct {
		name      string
		texts     []string
		modelResp string
		modelErr  error
		wantErr   bool
		want      string
	}{
		{
			name:      "happy path",
			texts:     []string{"text1", "text2"},
			modelResp: "combined summary",
			want:      "combined summary",
		},
		{
			name:    "empty texts",
			texts:   []string{},
			wantErr: true,
		},
		{
			name:     "model error",
			texts:    []string{"text1"},
			modelErr: fmt.Errorf("api error"),
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			model := &mockChatModel{
				generateFn: func(_ context.Context, _ []schema.Message, _ ...llm.GenerateOption) (*schema.AIMessage, error) {
					if tt.modelErr != nil {
						return nil, tt.modelErr
					}
					return schema.NewAIMessage(tt.modelResp), nil
				},
			}
			s := NewLLMSummarizer(model)
			got, err := s.Summarize(context.Background(), tt.texts)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Summarize() error = %v, wantErr = %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Summarize() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestLLMSummarizer_CustomPrompt(t *testing.T) {
	var receivedPrompt string
	model := &mockChatModel{
		generateFn: func(_ context.Context, msgs []schema.Message, _ ...llm.GenerateOption) (*schema.AIMessage, error) {
			if hm, ok := msgs[0].(*schema.HumanMessage); ok {
				receivedPrompt = hm.Text()
			}
			return schema.NewAIMessage("result"), nil
		},
	}
	s := NewLLMSummarizer(model, WithSummaryPrompt("Custom: %s"))
	_, err := s.Summarize(context.Background(), []string{"hello"})
	if err != nil {
		t.Fatal(err)
	}
	if receivedPrompt != "Custom: hello" {
		t.Errorf("prompt = %q, want %q", receivedPrompt, "Custom: hello")
	}
}

// =============================================================================
// Tree Builder Tests
// =============================================================================

func TestTreeBuilder_Build(t *testing.T) {
	embedder := &mockEmbedder{dimensions: 4}
	summarizer := &mockSummarizer{}

	// Use a clusterer that puts docs into pairs.
	clusterer := &mockClusterer{
		clusterFn: func(_ context.Context, embeddings [][]float32) ([][]int, error) {
			var clusters [][]int
			for i := 0; i < len(embeddings); i += 2 {
				if i+1 < len(embeddings) {
					clusters = append(clusters, []int{i, i + 1})
				}
			}
			return clusters, nil
		},
	}

	builder := NewTreeBuilder(
		WithClusterer(clusterer),
		WithSummarizer(summarizer),
		WithEmbedder(embedder),
		WithMaxLevels(2),
		WithMinClusterSize(2),
	)

	docs := []schema.Document{
		{ID: "d1", Content: "doc one"},
		{ID: "d2", Content: "doc two"},
		{ID: "d3", Content: "doc three"},
		{ID: "d4", Content: "doc four"},
	}

	tree, err := builder.Build(context.Background(), docs)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Should have 4 leaves + 2 L1 summaries + 1 L2 summary = 7 nodes.
	if len(tree.Nodes) != 7 {
		t.Errorf("tree has %d nodes, want 7", len(tree.Nodes))
	}

	// Verify leaf nodes.
	for _, id := range []string{"d1", "d2", "d3", "d4"} {
		node, ok := tree.Nodes[id]
		if !ok {
			t.Errorf("missing leaf node %q", id)
			continue
		}
		if node.Level != 0 {
			t.Errorf("leaf %q level = %d, want 0", id, node.Level)
		}
	}

	// Verify summary nodes exist at level 1.
	l1 := tree.NodesAtLevel(1)
	if len(l1) != 2 {
		t.Errorf("level 1 has %d nodes, want 2", len(l1))
	}

	// Verify level 2.
	l2 := tree.NodesAtLevel(2)
	if len(l2) != 1 {
		t.Errorf("level 2 has %d nodes, want 1", len(l2))
	}

	if tree.MaxLevel != 2 {
		t.Errorf("MaxLevel = %d, want 2", tree.MaxLevel)
	}

	// Summarizer should have been called 3 times (2 at L1, 1 at L2).
	if summarizer.calls != 3 {
		t.Errorf("summarizer called %d times, want 3", summarizer.calls)
	}
}

func TestTreeBuilder_Build_EmptyDocs(t *testing.T) {
	builder := NewTreeBuilder(
		WithEmbedder(&mockEmbedder{dimensions: 4}),
		WithSummarizer(&mockSummarizer{}),
	)
	_, err := builder.Build(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error for empty docs")
	}
}

func TestTreeBuilder_Build_MissingEmbedder(t *testing.T) {
	builder := NewTreeBuilder(
		WithSummarizer(&mockSummarizer{}),
	)
	_, err := builder.Build(context.Background(), []schema.Document{{Content: "test"}})
	if err == nil {
		t.Fatal("expected error for missing embedder")
	}
}

func TestTreeBuilder_Build_MissingSummarizer(t *testing.T) {
	builder := NewTreeBuilder(
		WithEmbedder(&mockEmbedder{dimensions: 4}),
	)
	_, err := builder.Build(context.Background(), []schema.Document{{Content: "test"}})
	if err == nil {
		t.Fatal("expected error for missing summarizer")
	}
}

func TestTreeBuilder_Build_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	builder := NewTreeBuilder(
		WithEmbedder(&mockEmbedder{dimensions: 4}),
		WithSummarizer(&mockSummarizer{}),
	)

	_, err := builder.Build(ctx, []schema.Document{{Content: "test"}})
	if err == nil {
		t.Fatal("expected error from cancelled context")
	}
}

func TestTreeBuilder_Build_PreEmbeddedDocs(t *testing.T) {
	embedCalls := 0
	embedder := &mockEmbedder{
		dimensions: 2,
		embedFn: func(_ context.Context, texts []string) ([][]float32, error) {
			embedCalls++
			result := make([][]float32, len(texts))
			for i := range texts {
				result[i] = []float32{0.5, 0.5}
			}
			return result, nil
		},
	}

	// Use a mock clusterer that groups both docs into one cluster.
	clusterer := &mockClusterer{
		clusterFn: func(_ context.Context, embeddings [][]float32) ([][]int, error) {
			indices := make([]int, len(embeddings))
			for i := range indices {
				indices[i] = i
			}
			return [][]int{indices}, nil
		},
	}

	builder := NewTreeBuilder(
		WithEmbedder(embedder),
		WithSummarizer(&mockSummarizer{}),
		WithClusterer(clusterer),
		WithMaxLevels(1),
		WithMinClusterSize(2),
	)

	// Docs already have embeddings.
	docs := []schema.Document{
		{ID: "d1", Content: "one", Embedding: []float32{1.0, 0.0}},
		{ID: "d2", Content: "two", Embedding: []float32{0.0, 1.0}},
	}

	tree, err := builder.Build(context.Background(), docs)
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	// Embedder should only be called for the summary, not for leaves.
	// 1 call for summary embedding.
	if embedCalls != 1 {
		t.Errorf("embed calls = %d, want 1 (summary only)", embedCalls)
	}

	// 2 leaves + 1 summary = 3 nodes.
	if len(tree.Nodes) != 3 {
		t.Errorf("expected 3 nodes (2 leaves + 1 summary), got %d", len(tree.Nodes))
	}
}

// =============================================================================
// Retriever Tests
// =============================================================================

func TestRAPTORRetriever_Retrieve(t *testing.T) {
	// Build a simple tree with known embeddings for testing similarity.
	tree := &Tree{
		Nodes: map[string]*TreeNode{
			"leaf-1": {
				ID: "leaf-1", Level: 0, Content: "specific detail about cats",
				Embedding: []float32{1.0, 0.0, 0.0},
				Metadata:  map[string]any{"topic": "cats"},
			},
			"leaf-2": {
				ID: "leaf-2", Level: 0, Content: "specific detail about dogs",
				Embedding: []float32{0.0, 1.0, 0.0},
				Metadata:  map[string]any{"topic": "dogs"},
			},
			"summary-1": {
				ID: "summary-1", Level: 1, Content: "summary about pets",
				Embedding: []float32{0.5, 0.5, 0.0},
				Children:  []string{"leaf-1", "leaf-2"},
				Metadata:  map[string]any{"raptor_level": 1},
			},
		},
		MaxLevel: 1,
	}

	embedder := &mockEmbedder{
		dimensions: 3,
		embedFn: func(_ context.Context, texts []string) ([][]float32, error) {
			// Query "cats" is close to leaf-1.
			return [][]float32{{0.9, 0.1, 0.0}}, nil
		},
	}

	r := NewRAPTORRetriever(
		WithTree(tree),
		WithRetrieverEmbedder(embedder),
		WithRaptorTopK(2),
	)

	docs, err := r.Retrieve(context.Background(), "cats")
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if len(docs) != 2 {
		t.Fatalf("got %d docs, want 2", len(docs))
	}

	// First result should be leaf-1 (closest to query).
	if docs[0].ID != "leaf-1" {
		t.Errorf("first result ID = %q, want %q", docs[0].ID, "leaf-1")
	}

	// Verify scores are in descending order.
	if docs[0].Score < docs[1].Score {
		t.Errorf("scores not in descending order: %f < %f", docs[0].Score, docs[1].Score)
	}

	// Verify metadata includes raptor_level.
	if level, ok := docs[0].Metadata["raptor_level"]; !ok || level != 0 {
		t.Errorf("expected raptor_level=0, got %v", docs[0].Metadata["raptor_level"])
	}
}

func TestRAPTORRetriever_Retrieve_WithThreshold(t *testing.T) {
	tree := &Tree{
		Nodes: map[string]*TreeNode{
			"n1": {ID: "n1", Level: 0, Content: "close", Embedding: []float32{1.0, 0.0}},
			"n2": {ID: "n2", Level: 0, Content: "far", Embedding: []float32{0.0, 1.0}},
		},
	}

	embedder := &mockEmbedder{
		dimensions: 2,
		embedFn: func(_ context.Context, _ []string) ([][]float32, error) {
			return [][]float32{{1.0, 0.0}}, nil
		},
	}

	r := NewRAPTORRetriever(
		WithTree(tree),
		WithRetrieverEmbedder(embedder),
		WithRaptorTopK(10),
	)

	// High threshold should filter out the far document.
	docs, err := r.Retrieve(context.Background(), "query", retriever.WithThreshold(0.5))
	if err != nil {
		t.Fatalf("Retrieve() error = %v", err)
	}

	if len(docs) != 1 {
		t.Fatalf("got %d docs, want 1 (threshold filtered)", len(docs))
	}
	if docs[0].ID != "n1" {
		t.Errorf("expected n1, got %q", docs[0].ID)
	}
}

func TestRAPTORRetriever_Retrieve_EmptyTree(t *testing.T) {
	r := NewRAPTORRetriever(
		WithRetrieverEmbedder(&mockEmbedder{dimensions: 2}),
	)
	_, err := r.Retrieve(context.Background(), "query")
	if err == nil {
		t.Fatal("expected error for empty tree")
	}
}

func TestRAPTORRetriever_Retrieve_MissingEmbedder(t *testing.T) {
	r := NewRAPTORRetriever(
		WithTree(&Tree{Nodes: map[string]*TreeNode{"n": {ID: "n"}}}),
	)
	_, err := r.Retrieve(context.Background(), "query")
	if err == nil {
		t.Fatal("expected error for missing embedder")
	}
}

func TestRAPTORRetriever_Retrieve_TopKFromOption(t *testing.T) {
	tree := &Tree{
		Nodes: map[string]*TreeNode{
			"n1": {ID: "n1", Level: 0, Content: "a", Embedding: []float32{1.0, 0.0}},
			"n2": {ID: "n2", Level: 0, Content: "b", Embedding: []float32{0.9, 0.1}},
			"n3": {ID: "n3", Level: 0, Content: "c", Embedding: []float32{0.8, 0.2}},
		},
	}

	embedder := &mockEmbedder{
		dimensions: 2,
		embedFn: func(_ context.Context, _ []string) ([][]float32, error) {
			return [][]float32{{1.0, 0.0}}, nil
		},
	}

	r := NewRAPTORRetriever(
		WithTree(tree),
		WithRetrieverEmbedder(embedder),
		WithRaptorTopK(10),
	)

	// Override topK via retriever option.
	docs, err := r.Retrieve(context.Background(), "q", retriever.WithTopK(1))
	if err != nil {
		t.Fatal(err)
	}
	if len(docs) != 1 {
		t.Errorf("got %d docs, want 1", len(docs))
	}
}

func TestRAPTORRetriever_Hooks(t *testing.T) {
	var beforeCalled, afterCalled bool

	tree := &Tree{
		Nodes: map[string]*TreeNode{
			"n1": {ID: "n1", Level: 0, Content: "test", Embedding: []float32{1.0}},
		},
	}

	embedder := &mockEmbedder{
		dimensions: 1,
		embedFn: func(_ context.Context, _ []string) ([][]float32, error) {
			return [][]float32{{1.0}}, nil
		},
	}

	r := NewRAPTORRetriever(
		WithTree(tree),
		WithRetrieverEmbedder(embedder),
		WithRaptorHooks(retriever.Hooks{
			BeforeRetrieve: func(_ context.Context, _ string) error {
				beforeCalled = true
				return nil
			},
			AfterRetrieve: func(_ context.Context, _ []schema.Document, _ error) {
				afterCalled = true
			},
		}),
	)

	_, err := r.Retrieve(context.Background(), "query")
	if err != nil {
		t.Fatal(err)
	}

	if !beforeCalled {
		t.Error("BeforeRetrieve hook was not called")
	}
	if !afterCalled {
		t.Error("AfterRetrieve hook was not called")
	}
}

// =============================================================================
// Cosine Similarity Tests
// =============================================================================

func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name string
		a, b []float32
		want float64
	}{
		{
			name: "identical vectors",
			a:    []float32{1, 0, 0},
			b:    []float32{1, 0, 0},
			want: 1.0,
		},
		{
			name: "orthogonal vectors",
			a:    []float32{1, 0},
			b:    []float32{0, 1},
			want: 0.0,
		},
		{
			name: "opposite vectors",
			a:    []float32{1, 0},
			b:    []float32{-1, 0},
			want: -1.0,
		},
		{
			name: "zero vector",
			a:    []float32{0, 0},
			b:    []float32{1, 0},
			want: 0.0,
		},
		{
			name: "empty vectors",
			a:    []float32{},
			b:    []float32{},
			want: 0.0,
		},
		{
			name: "mismatched lengths",
			a:    []float32{1},
			b:    []float32{1, 2},
			want: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cosineSimilarity(tt.a, tt.b)
			if math.Abs(got-tt.want) > 1e-9 {
				t.Errorf("cosineSimilarity() = %f, want %f", got, tt.want)
			}
		})
	}
}

// =============================================================================
// Tree Type Tests
// =============================================================================

func TestTree_AllNodes(t *testing.T) {
	tree := &Tree{
		Nodes: map[string]*TreeNode{
			"a": {ID: "a", Level: 0},
			"b": {ID: "b", Level: 0},
			"c": {ID: "c", Level: 1},
		},
	}

	nodes := tree.AllNodes()
	if len(nodes) != 3 {
		t.Errorf("AllNodes() returned %d nodes, want 3", len(nodes))
	}
}

func TestTree_NodesAtLevel(t *testing.T) {
	tree := &Tree{
		Nodes: map[string]*TreeNode{
			"a": {ID: "a", Level: 0},
			"b": {ID: "b", Level: 0},
			"c": {ID: "c", Level: 1},
			"d": {ID: "d", Level: 2},
		},
	}

	l0 := tree.NodesAtLevel(0)
	if len(l0) != 2 {
		t.Errorf("NodesAtLevel(0) = %d, want 2", len(l0))
	}

	l1 := tree.NodesAtLevel(1)
	if len(l1) != 1 {
		t.Errorf("NodesAtLevel(1) = %d, want 1", len(l1))
	}

	l3 := tree.NodesAtLevel(3)
	if len(l3) != 0 {
		t.Errorf("NodesAtLevel(3) = %d, want 0", len(l3))
	}
}

// =============================================================================
// Registry Test
// =============================================================================

func TestRAPTORRegistered(t *testing.T) {
	names := retriever.List()
	found := false
	for _, n := range names {
		if n == "raptor" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("raptor not found in retriever registry, registered: %v", names)
	}
}

// =============================================================================
// Compile-time checks
// =============================================================================

var _ Clusterer = (*KMeansClusterer)(nil)
var _ Summarizer = (*LLMSummarizer)(nil)
var _ retriever.Retriever = (*RAPTORRetriever)(nil)
