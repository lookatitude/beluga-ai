package raptor

import (
	"context"
	"math"
	"sort"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/rag/embedding"
	"github.com/lookatitude/beluga-ai/v2/rag/retriever"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

func init() {
	// RAPTOR requires a pre-built *Tree and an Embedder, neither of which can
	// be sourced from a generic ProviderConfig. Register a factory that
	// returns a descriptive error directing callers to NewRAPTORRetriever.
	retriever.Register("raptor", func(cfg config.ProviderConfig) (retriever.Retriever, error) {
		return nil, core.Errorf(core.ErrInvalidInput, "raptor: use NewRAPTORRetriever directly; tree and embedder cannot be provided via ProviderConfig")
	})
}

// RAPTORRetriever implements retriever.Retriever using collapsed tree search.
// It flattens all tree nodes across all levels and performs cosine similarity
// against the query embedding to find the most relevant nodes.
type RAPTORRetriever struct { //nolint:revive // stutter accepted: renaming this exported type is a breaking API change
	tree     *Tree
	embedder embedding.Embedder
	topK     int
	hooks    retriever.Hooks
}

// Compile-time interface check.
var _ retriever.Retriever = (*RAPTORRetriever)(nil)

// RAPTOROption configures a RAPTORRetriever.
type RAPTOROption func(*RAPTORRetriever) //nolint:revive // stutter accepted: renaming this exported type is a breaking API change

// WithTree sets the pre-built RAPTOR tree for retrieval.
func WithTree(t *Tree) RAPTOROption {
	return func(r *RAPTORRetriever) {
		r.tree = t
	}
}

// WithRetrieverEmbedder sets the embedder used to embed queries at retrieval
// time.
func WithRetrieverEmbedder(e embedding.Embedder) RAPTOROption {
	return func(r *RAPTORRetriever) {
		r.embedder = e
	}
}

// WithRaptorTopK sets the default number of results to return.
func WithRaptorTopK(k int) RAPTOROption {
	return func(r *RAPTORRetriever) {
		r.topK = k
	}
}

// WithRaptorHooks sets hooks on the RAPTORRetriever.
func WithRaptorHooks(h retriever.Hooks) RAPTOROption {
	return func(r *RAPTORRetriever) {
		r.hooks = h
	}
}

// NewRAPTORRetriever creates a new RAPTORRetriever with the given options.
func NewRAPTORRetriever(opts ...RAPTOROption) *RAPTORRetriever {
	r := &RAPTORRetriever{
		topK: 10,
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// Retrieve performs collapsed tree search: it embeds the query, computes
// cosine similarity against all nodes in the tree, and returns the top-K
// most similar nodes as documents.
func (r *RAPTORRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
	cfg := retriever.DefaultConfig()
	cfg.TopK = r.topK
	for _, o := range opts {
		o(&cfg)
	}

	if r.hooks.BeforeRetrieve != nil {
		if err := r.hooks.BeforeRetrieve(ctx, query); err != nil {
			return nil, err
		}
	}

	if r.tree == nil || len(r.tree.Nodes) == 0 {
		return nil, r.fail(ctx, core.Errorf(core.ErrInvalidInput, "raptor: retrieve: tree is empty or not set"))
	}

	if r.embedder == nil {
		return nil, r.fail(ctx, core.Errorf(core.ErrInvalidInput, "raptor: retrieve: embedder is required"))
	}

	queryEmb, err := r.embedder.EmbedSingle(ctx, query)
	if err != nil {
		return nil, r.fail(ctx, core.Errorf(core.ErrProviderDown, "raptor: retrieve: embed query: %w", err))
	}

	docs := r.collapsedSearch(queryEmb, cfg.Threshold, cfg.TopK)

	if r.hooks.AfterRetrieve != nil {
		r.hooks.AfterRetrieve(ctx, docs, nil)
	}

	return docs, nil
}

// fail reports err to the AfterRetrieve hook (if set) and returns it.
func (r *RAPTORRetriever) fail(ctx context.Context, err error) error {
	if r.hooks.AfterRetrieve != nil {
		r.hooks.AfterRetrieve(ctx, nil, err)
	}
	return err
}

// scoredNode pairs a tree node with its similarity to the query.
type scoredNode struct {
	node  *TreeNode
	score float64
}

// collapsedSearch scores every node across all tree levels against queryEmb,
// drops nodes below threshold (when threshold > 0), and returns the topK most
// similar nodes as documents in descending score order.
func (r *RAPTORRetriever) collapsedSearch(queryEmb []float32, threshold float64, topK int) []schema.Document {
	allNodes := r.tree.AllNodes()
	candidates := make([]scoredNode, 0, len(allNodes))
	for _, node := range allNodes {
		sim := cosineSimilarity(queryEmb, node.Embedding)
		if threshold > 0 && sim < threshold {
			continue
		}
		candidates = append(candidates, scoredNode{node: node, score: sim})
	}

	// Sort by descending score.
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if topK > len(candidates) {
		topK = len(candidates)
	}

	docs := make([]schema.Document, topK)
	for i := 0; i < topK; i++ {
		s := candidates[i]
		metadata := make(map[string]any)
		for k, v := range s.node.Metadata {
			metadata[k] = v
		}
		metadata["raptor_level"] = s.node.Level
		metadata["raptor_node_id"] = s.node.ID

		docs[i] = schema.Document{
			ID:       s.node.ID,
			Content:  s.node.Content,
			Score:    s.score,
			Metadata: metadata,
		}
	}
	return docs
}

// cosineSimilarity computes the cosine similarity between two vectors.
// Returns 0 if either vector has zero magnitude.
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, normA, normB float64
	for i := range a {
		ai, bi := float64(a[i]), float64(b[i])
		dot += ai * bi
		normA += ai * ai
		normB += bi * bi
	}

	denom := math.Sqrt(normA) * math.Sqrt(normB)
	if denom == 0 {
		return 0
	}

	return dot / denom
}
