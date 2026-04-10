package raptor

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/rag/embedding"
	"github.com/lookatitude/beluga-ai/schema"
)

// TreeBuilder constructs a RAPTOR tree from a set of documents by repeatedly
// clustering, summarizing, and embedding nodes.
type TreeBuilder struct {
	clusterer  Clusterer
	summarizer Summarizer
	embedder   embedding.Embedder
	config     TreeConfig
}

// BuilderOption configures a TreeBuilder.
type BuilderOption func(*TreeBuilder)

// WithClusterer sets the clustering strategy for the tree builder.
func WithClusterer(c Clusterer) BuilderOption {
	return func(b *TreeBuilder) {
		b.clusterer = c
	}
}

// WithSummarizer sets the summarization strategy for the tree builder.
func WithSummarizer(s Summarizer) BuilderOption {
	return func(b *TreeBuilder) {
		b.summarizer = s
	}
}

// WithEmbedder sets the embedding model for the tree builder.
func WithEmbedder(e embedding.Embedder) BuilderOption {
	return func(b *TreeBuilder) {
		b.embedder = e
	}
}

// WithMaxLevels sets the maximum number of summary levels above the leaf
// level. Default: 3.
func WithMaxLevels(n int) BuilderOption {
	return func(b *TreeBuilder) {
		b.config.MaxLevels = n
	}
}

// WithMinClusterSize sets the minimum number of nodes required to form a
// cluster. Default: 2.
func WithMinClusterSize(n int) BuilderOption {
	return func(b *TreeBuilder) {
		b.config.MinClusterSize = n
	}
}

// NewTreeBuilder creates a TreeBuilder with the given options.
func NewTreeBuilder(opts ...BuilderOption) *TreeBuilder {
	b := &TreeBuilder{
		clusterer: &KMeansClusterer{},
		config:    DefaultTreeConfig(),
	}
	for _, o := range opts {
		o(b)
	}
	return b
}

// Build constructs a RAPTOR tree from the provided documents. It creates leaf
// nodes from each document, then iteratively clusters, summarizes, and embeds
// nodes to build higher abstraction levels.
func (b *TreeBuilder) Build(ctx context.Context, docs []schema.Document) (*Tree, error) {
	if len(docs) == 0 {
		return nil, fmt.Errorf("raptor: build: no documents provided")
	}
	if b.embedder == nil {
		return nil, fmt.Errorf("raptor: build: embedder is required")
	}
	if b.summarizer == nil {
		return nil, fmt.Errorf("raptor: build: summarizer is required")
	}

	tree := &Tree{
		Nodes: make(map[string]*TreeNode),
	}

	// Create leaf nodes (level 0).
	currentLevel := make([]*TreeNode, 0, len(docs))
	for i, doc := range docs {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		id := doc.ID
		if id == "" {
			id = fmt.Sprintf("leaf-%d", i)
		}

		emb := doc.Embedding
		if len(emb) == 0 {
			var err error
			emb, err = b.embedder.EmbedSingle(ctx, doc.Content)
			if err != nil {
				return nil, fmt.Errorf("raptor: build: embed leaf %d: %w", i, err)
			}
		}

		if _, exists := tree.Nodes[id]; exists {
			return nil, fmt.Errorf("raptor: build: duplicate document ID %q at index %d", id, i)
		}

		node := &TreeNode{
			ID:        id,
			Level:     0,
			Content:   doc.Content,
			Embedding: emb,
			Metadata:  doc.Metadata,
		}
		tree.Nodes[id] = node
		currentLevel = append(currentLevel, node)
	}

	// Build summary levels.
	for level := 1; level <= b.config.MaxLevels; level++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if len(currentLevel) < b.config.MinClusterSize {
			break
		}

		// Extract embeddings for clustering.
		embeddings := make([][]float32, len(currentLevel))
		for i, n := range currentLevel {
			embeddings[i] = n.Embedding
		}

		clusters, err := b.clusterer.Cluster(ctx, embeddings)
		if err != nil {
			return nil, fmt.Errorf("raptor: build: cluster level %d: %w", level, err)
		}

		nextLevel := make([]*TreeNode, 0, len(clusters))
		for ci, indices := range clusters {
			if len(indices) < b.config.MinClusterSize {
				continue
			}

			// Gather texts from cluster members.
			texts := make([]string, len(indices))
			childIDs := make([]string, len(indices))
			for j, idx := range indices {
				texts[j] = currentLevel[idx].Content
				childIDs[j] = currentLevel[idx].ID
			}

			summary, err := b.summarizer.Summarize(ctx, texts)
			if err != nil {
				return nil, fmt.Errorf("raptor: build: summarize cluster %d at level %d: %w", ci, level, err)
			}

			emb, err := b.embedder.EmbedSingle(ctx, summary)
			if err != nil {
				return nil, fmt.Errorf("raptor: build: embed summary cluster %d at level %d: %w", ci, level, err)
			}

			nodeID := fmt.Sprintf("summary-L%d-C%d", level, ci)
			node := &TreeNode{
				ID:        nodeID,
				Level:     level,
				Content:   summary,
				Embedding: emb,
				Children:  childIDs,
				Metadata: map[string]any{
					"raptor_level":   level,
					"raptor_cluster": ci,
				},
			}

			// Set parent references on children.
			for _, idx := range indices {
				currentLevel[idx].ParentID = nodeID
			}

			tree.Nodes[nodeID] = node
			nextLevel = append(nextLevel, node)
		}

		if len(nextLevel) == 0 {
			break
		}

		tree.MaxLevel = level
		currentLevel = nextLevel
	}

	return tree, nil
}
