---
title: RAG Recipes
description: Practical recipes for building retrieval-augmented generation pipelines with Beluga AI, covering splitting, embedding, retrieval, and indexing strategies.
sidebar:
  order: 0
---

Retrieval-Augmented Generation (RAG) connects LLMs to external knowledge by retrieving relevant documents at query time. The quality of a RAG system depends on every stage of the pipeline: how documents are split into chunks, how those chunks are embedded and stored, how retrieval combines multiple signals, and how results are filtered and ranked. These recipes address common challenges at each stage, using Beluga AI's composable RAG components.

Beluga AI's RAG pipeline follows the registry pattern (`Register()` + `New()` + `List()`) across all extensible components -- embedders, vector stores, retrievers, loaders, and splitters -- so you can swap implementations without changing application code. The default retrieval strategy uses hybrid search (vector similarity + BM25 keyword matching + Reciprocal Rank Fusion), which outperforms either approach alone across diverse query types.

## Parent Document Retrieval

**Problem:** Small chunks improve embedding precision, but the LLM needs larger context to generate good answers. You want to retrieve small chunks but return their parent documents.

There is a fundamental tension in chunk sizing for RAG: smaller chunks produce more precise embeddings that match specific queries, but they strip away the surrounding context that the LLM needs to generate accurate, grounded answers. Parent Document Retrieval (PDR) resolves this by decoupling the retrieval unit (small chunks) from the generation unit (larger parent documents).

**Solution:** Index small chunks with metadata pointing to their parent, then fetch parents at retrieval time. This two-level hierarchy gives you the best of both worlds: precision during search and richness during generation.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/rag/retriever"
	"github.com/lookatitude/beluga-ai/rag/splitter"
	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

func main() {
	ctx := context.Background()

	// Split documents into small child chunks and large parent chunks.
	childSplitter := splitter.NewRecursiveCharacter(
		splitter.WithChunkSize(200),
		splitter.WithChunkOverlap(20),
	)

	parentSplitter := splitter.NewRecursiveCharacter(
		splitter.WithChunkSize(1000),
		splitter.WithChunkOverlap(100),
	)

	doc := schema.Document{
		Content: "Long document content here...",
		Metadata: map[string]any{
			"source": "architecture.md",
		},
	}

	// Split into parent chunks.
	parents, err := parentSplitter.Split(ctx, []schema.Document{doc})
	if err != nil {
		slog.Error("parent split failed", "error", err)
		return
	}

	// For each parent, create child chunks with parent_id metadata.
	var children []schema.Document
	for i, parent := range parents {
		parentID := fmt.Sprintf("parent_%d", i)
		parent.Metadata["chunk_id"] = parentID

		childDocs, err := childSplitter.Split(ctx, []schema.Document{parent})
		if err != nil {
			slog.Error("child split failed", "error", err)
			continue
		}

		for j := range childDocs {
			childDocs[j].Metadata["parent_id"] = parentID
		}
		children = append(children, childDocs...)
	}

	// Index children in the vector store for precise matching.
	// At retrieval time, use the parent_id to fetch the larger context.
	_ = children
	_ = parents
	fmt.Printf("Created %d parent chunks with %d child chunks\n",
		len(parents), len(children))
}
```

The manual approach above shows the underlying mechanics. In practice, Beluga AI provides a built-in `ParentDocumentRetriever` that handles the parent-child relationship management, deduplication, and score propagation automatically:

**With the built-in `ParentDocumentRetriever`:**

```go
// Configure parent document retrieval.
ret := retriever.NewParentDocument(
	retriever.WithChildStore(childVectorStore),
	retriever.WithParentStore(parentStore),
	retriever.WithChildSplitter(childSplitter),
	retriever.WithParentSplitter(parentSplitter),
)

// Retrieve returns parent documents, not child chunks.
docs, err := ret.Retrieve(ctx, "query about architecture", 5)
```

---

## Reranking with Cohere

**Problem:** Initial vector search returns many candidates, but the ordering isn't optimal. You want to rerank results with a cross-encoder for better relevance.

Vector search uses bi-encoders that embed queries and documents independently, then compare via cosine similarity. This is fast but loses nuance because the query and document are never considered together. Cross-encoder rerankers like Cohere's Rerank API score each query-document pair jointly, capturing subtle relevance signals that bi-encoders miss. The tradeoff is latency: cross-encoders are slower, so they work best as a second stage applied to a smaller candidate set.

**Solution:** Use a reranker in the retriever pipeline to refine results after initial retrieval. Retrieve more candidates than needed, then rerank to select the top results.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"sort"

	"github.com/lookatitude/beluga-ai/schema"
)

// Reranker scores and reorders documents by relevance to a query.
type Reranker interface {
	Rerank(ctx context.Context, query string, docs []schema.Document, topK int) ([]schema.Document, error)
}

// CohereReranker calls the Cohere Rerank API.
type CohereReranker struct {
	apiKey string
	model  string
}

func NewCohereReranker(apiKey, model string) *CohereReranker {
	return &CohereReranker{apiKey: apiKey, model: model}
}

func (r *CohereReranker) Rerank(ctx context.Context, query string, docs []schema.Document, topK int) ([]schema.Document, error) {
	// In production, this calls the Cohere API.
	// Simplified example: score by content length match (placeholder).
	type scored struct {
		doc   schema.Document
		score float64
	}

	results := make([]scored, len(docs))
	for i, doc := range docs {
		results[i] = scored{doc: doc, score: float64(len(doc.Content))}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].score > results[j].score
	})

	reranked := make([]schema.Document, 0, topK)
	for i := 0; i < topK && i < len(results); i++ {
		results[i].doc.Metadata["rerank_score"] = results[i].score
		reranked = append(reranked, results[i].doc)
	}

	return reranked, nil
}

func main() {
	ctx := context.Background()

	reranker := NewCohereReranker("cohere-api-key", "rerank-v3.5")

	// Assume initial retrieval returned 20 candidates.
	candidates := make([]schema.Document, 20)
	for i := range candidates {
		candidates[i] = schema.Document{
			Content:  fmt.Sprintf("Document %d content about Go programming", i),
			Metadata: map[string]any{"id": i},
		}
	}

	// Rerank to get the top 5 most relevant.
	reranked, err := reranker.Rerank(ctx, "Go concurrency patterns", candidates, 5)
	if err != nil {
		slog.Error("reranking failed", "error", err)
		return
	}

	for i, doc := range reranked {
		fmt.Printf("#%d (score: %.2f): %s\n", i+1,
			doc.Metadata["rerank_score"], doc.Content)
	}
}
```

---

## Parallel Document Loading

**Problem:** Loading documents from multiple files or directories is slow when done sequentially.

Document loading is I/O-bound: the CPU spends most of its time waiting for disk reads. Sequential loading wastes this idle time, especially when loading hundreds or thousands of files. Go's goroutines make concurrent I/O straightforward, but unbounded parallelism can exhaust file descriptors or overwhelm the OS scheduler.

**Solution:** Use goroutines with a semaphore to load documents in parallel. The semaphore (buffered channel) caps the number of concurrent file reads, balancing throughput against resource usage.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/lookatitude/beluga-ai/rag/loader"
	"github.com/lookatitude/beluga-ai/schema"
)

// ParallelLoader loads documents from multiple sources concurrently.
type ParallelLoader struct {
	loader      loader.DocumentLoader
	concurrency int
}

func NewParallelLoader(l loader.DocumentLoader, concurrency int) *ParallelLoader {
	return &ParallelLoader{loader: l, concurrency: concurrency}
}

// LoadAll loads documents from all paths concurrently with bounded parallelism.
func (pl *ParallelLoader) LoadAll(ctx context.Context, paths []string) ([]schema.Document, error) {
	sem := make(chan struct{}, pl.concurrency)
	var mu sync.Mutex
	var allDocs []schema.Document
	var firstErr error

	var wg sync.WaitGroup
	for _, p := range paths {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire.
			defer func() { <-sem }() // Release.

			docs, err := pl.loader.Load(ctx, path)
			mu.Lock()
			defer mu.Unlock()

			if err != nil {
				slog.Error("load failed", "path", path, "error", err)
				if firstErr == nil {
					firstErr = err
				}
				return
			}
			allDocs = append(allDocs, docs...)
		}(p)
	}

	wg.Wait()
	return allDocs, firstErr
}

func main() {
	ctx := context.Background()

	// Load up to 10 files simultaneously.
	pl := NewParallelLoader(loader.NewText(), 10)

	paths := []string{
		"/data/docs/readme.md",
		"/data/docs/architecture.md",
		"/data/docs/api-reference.md",
	}

	docs, err := pl.LoadAll(ctx, paths)
	if err != nil {
		slog.Warn("some documents failed to load", "error", err)
	}

	fmt.Printf("Loaded %d documents\n", len(docs))
}
```

---

## Handle Corrupt Documents Gracefully

**Problem:** Some documents in the pipeline are corrupt or in unexpected formats, causing the entire ingestion to fail.

In production, document collections are rarely clean. Files may be truncated, encoded incorrectly, or contain binary data mixed with text. A single corrupt file should not halt an entire ingestion pipeline that would otherwise succeed for hundreds of valid documents. Error isolation at the document level keeps the pipeline running while providing visibility into which files failed and why.

**Solution:** Wrap loaders with error handling that skips bad documents and logs failures. Returning an empty slice instead of an error for corrupt files allows the pipeline to continue processing remaining documents.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/rag/loader"
	"github.com/lookatitude/beluga-ai/schema"
)

// ResilientLoader wraps a DocumentLoader and skips corrupt files.
type ResilientLoader struct {
	inner loader.DocumentLoader
}

func NewResilientLoader(inner loader.DocumentLoader) *ResilientLoader {
	return &ResilientLoader{inner: inner}
}

func (r *ResilientLoader) Load(ctx context.Context, source string) ([]schema.Document, error) {
	docs, err := r.inner.Load(ctx, source)
	if err != nil {
		slog.Warn("skipping corrupt document",
			"source", source,
			"error", err,
		)
		// Return empty slice instead of error — pipeline continues.
		return nil, nil
	}

	// Validate each document.
	var valid []schema.Document
	for _, doc := range docs {
		if doc.Content == "" {
			slog.Warn("skipping empty document", "source", source)
			continue
		}
		valid = append(valid, doc)
	}

	return valid, nil
}

func main() {
	ctx := context.Background()

	resilient := NewResilientLoader(loader.NewText())

	paths := []string{
		"/data/good-file.md",
		"/data/corrupt-file.bin",
		"/data/another-good.md",
	}

	var allDocs []schema.Document
	for _, path := range paths {
		docs, err := resilient.Load(ctx, path)
		if err != nil {
			continue
		}
		allDocs = append(allDocs, docs...)
	}

	fmt.Printf("Successfully loaded %d documents (skipped corrupt ones)\n", len(allDocs))
}
```

---

## Batch Embedding Optimization

**Problem:** Embedding documents one at a time is slow. You need to batch embeddings for throughput while respecting API rate limits.

Embedding API calls have significant per-request overhead (network latency, connection setup, authentication). Sending one document per request means that overhead dominates total time. Batching amortizes this overhead across many documents. Most embedding providers support batch operations with limits (e.g., OpenAI allows up to 2048 inputs per batch), so batching also aligns with provider-optimal usage patterns.

**Solution:** Chunk documents into batches and embed them with controlled concurrency. This reduces total API calls from N to N/batch_size while staying within provider rate limits.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/rag/embedding"
)

// BatchEmbed processes texts in batches for optimal throughput.
func BatchEmbed(ctx context.Context, embedder embedding.Embedder, texts []string, batchSize int) ([][]float64, error) {
	var allEmbeddings [][]float64

	for i := 0; i < len(texts); i += batchSize {
		end := i + batchSize
		if end > len(texts) {
			end = len(texts)
		}

		batch := texts[i:end]
		embeddings, err := embedder.EmbedDocuments(ctx, batch)
		if err != nil {
			return nil, fmt.Errorf("batch %d-%d failed: %w", i, end, err)
		}

		allEmbeddings = append(allEmbeddings, embeddings...)

		slog.Info("embedded batch",
			"batch_start", i,
			"batch_end", end,
			"total", len(texts),
		)
	}

	return allEmbeddings, nil
}

func main() {
	ctx := context.Background()

	embedder, err := embedding.New("openai", embedding.ProviderConfig{
		APIKey: "your-api-key",
		Model:  "text-embedding-3-small",
	})
	if err != nil {
		slog.Error("embedder creation failed", "error", err)
		return
	}

	// 1000 documents, processed in batches of 100.
	texts := make([]string, 1000)
	for i := range texts {
		texts[i] = fmt.Sprintf("Document %d content about various topics", i)
	}

	embeddings, err := BatchEmbed(ctx, embedder, texts, 100)
	if err != nil {
		slog.Error("batch embedding failed", "error", err)
		return
	}

	fmt.Printf("Embedded %d documents (dimension: %d)\n",
		len(embeddings), len(embeddings[0]))
}
```

---

## Advanced Metadata Filtering in Vector Stores

**Problem:** You need to filter vector search results by metadata (e.g., date range, category, access level) in addition to semantic similarity.

Semantic similarity alone is not enough for production RAG. A query about "deployment patterns" should not return restricted security documents even if they are semantically similar. Metadata filtering applies hard constraints (access control, date ranges, categories) before or alongside vector similarity, ensuring results are both relevant and appropriate. Most vector stores support native metadata filtering, which is far more efficient than post-retrieval filtering because it reduces the candidate set before computing similarity scores.

**Solution:** Use the vector store's metadata filter when querying. Beluga AI's vector store interface accepts filter options via the `WithFilter()` functional option pattern.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

func main() {
	ctx := context.Background()

	store, err := vectorstore.New("pgvector", vectorstore.ProviderConfig{
		ConnectionString: "postgres://user:pass@localhost:5432/vectors",
	})
	if err != nil {
		slog.Error("store creation failed", "error", err)
		return
	}

	// Add documents with rich metadata.
	docs := []schema.Document{
		{
			Content: "Kubernetes deployment best practices",
			Metadata: map[string]any{
				"category": "infrastructure",
				"date":     "2025-01-15",
				"access":   "public",
			},
		},
		{
			Content: "Internal security audit results",
			Metadata: map[string]any{
				"category": "security",
				"date":     "2025-03-01",
				"access":   "restricted",
			},
		},
	}

	err = store.AddDocuments(ctx, docs)
	if err != nil {
		slog.Error("add failed", "error", err)
		return
	}

	// Search with metadata filters — only public infrastructure docs.
	results, err := store.SimilaritySearch(ctx, "deployment patterns", 10,
		vectorstore.WithFilter(map[string]any{
			"category": "infrastructure",
			"access":   "public",
		}),
	)
	if err != nil {
		slog.Error("search failed", "error", err)
		return
	}

	for _, doc := range results {
		fmt.Printf("[%s] %s\n", doc.Metadata["category"], doc.Content)
	}
}
```

---

## Code-Aware Text Splitting

**Problem:** Generic text splitters break code at arbitrary points, splitting functions mid-definition.

When indexing code for RAG, the default character-based splitting produces chunks that start or end mid-function, making them nearly useless for retrieval. An embedding of half a function captures incomplete semantics. Code-aware splitting uses language-specific boundaries (function declarations, class definitions, method signatures) as split points, producing chunks that represent complete logical units.

**Solution:** Use `splitter.NewCode` which understands language-specific boundaries. It respects function and class definitions, keeping related code together within each chunk.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/rag/splitter"
	"github.com/lookatitude/beluga-ai/schema"
)

func main() {
	ctx := context.Background()

	// Code-aware splitter respects function and class boundaries.
	codeSplitter := splitter.NewCode(
		splitter.WithLanguage("go"),
		splitter.WithChunkSize(500),
		splitter.WithChunkOverlap(50),
	)

	goCode := schema.Document{
		Content: `package main

import "fmt"

func hello() {
	fmt.Println("Hello, World!")
}

func goodbye() {
	fmt.Println("Goodbye!")
}

type Server struct {
	port int
	host string
}

func (s *Server) Start() error {
	fmt.Printf("Starting on %s:%d\n", s.host, s.port)
	return nil
}`,
		Metadata: map[string]any{
			"source":   "main.go",
			"language": "go",
		},
	}

	chunks, err := codeSplitter.Split(ctx, []schema.Document{goCode})
	if err != nil {
		slog.Error("split failed", "error", err)
		return
	}

	for i, chunk := range chunks {
		fmt.Printf("--- Chunk %d (%d chars) ---\n%s\n\n",
			i, len(chunk.Content), chunk.Content)
	}
}
```

---

## Sentence-Boundary Splitting

**Problem:** Splitting text at fixed character counts can break mid-sentence, degrading retrieval quality.

Each chunk in a RAG system becomes an independent unit for embedding and retrieval. When a chunk starts or ends mid-sentence, the embedding captures incomplete thoughts, and the retrieved context presented to the LLM is harder to interpret. Sentence-aware splitting ensures each chunk contains complete sentences, producing more coherent embeddings and more useful retrieval results.

**Solution:** Use sentence-aware splitting that respects natural boundaries. Sentences are grouped together until the chunk size limit is reached, then a new chunk begins at the next sentence boundary.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/rag/splitter"
	"github.com/lookatitude/beluga-ai/schema"
)

func main() {
	ctx := context.Background()

	// Sentence-boundary-aware splitting.
	sentenceSplitter := splitter.NewSentence(
		splitter.WithChunkSize(300),
		splitter.WithChunkOverlap(30),
	)

	doc := schema.Document{
		Content: "Go is a statically typed language. It was designed at Google. " +
			"The language emphasizes simplicity and readability. Go has built-in " +
			"concurrency support through goroutines and channels. The standard " +
			"library is comprehensive and well-designed. Many cloud-native tools " +
			"are written in Go, including Docker and Kubernetes.",
	}

	chunks, err := sentenceSplitter.Split(ctx, []schema.Document{doc})
	if err != nil {
		slog.Error("split failed", "error", err)
		return
	}

	for i, chunk := range chunks {
		fmt.Printf("Chunk %d: %q\n", i, chunk.Content)
	}
}
```

---

## Reindexing with Status Tracking

**Problem:** You need to reindex a large document collection and track progress, handling failures without restarting from scratch.

Reindexing a vector store with thousands or millions of documents is a long-running operation that can take minutes to hours. Without progress tracking, operators have no visibility into whether the system is working, how far along it is, or which documents failed. Batch-level failure tracking allows retrying only the failed subset rather than restarting from the beginning, which is critical for large collections where transient errors (network timeouts, rate limits) are expected.

**Solution:** Track indexed document IDs and support incremental reindexing. Process documents in batches, recording successes and failures independently so failed batches can be retried.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/lookatitude/beluga-ai/rag/vectorstore"
	"github.com/lookatitude/beluga-ai/schema"
)

// IndexStatus tracks the state of the reindexing operation.
type IndexStatus struct {
	mu         sync.Mutex
	Total      int
	Indexed    int
	Failed     int
	FailedIDs  []string
	StartTime  time.Time
}

func (s *IndexStatus) RecordSuccess() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Indexed++
}

func (s *IndexStatus) RecordFailure(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Failed++
	s.FailedIDs = append(s.FailedIDs, id)
}

func (s *IndexStatus) Progress() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	elapsed := time.Since(s.StartTime)
	pct := float64(s.Indexed+s.Failed) / float64(s.Total) * 100
	return fmt.Sprintf("%.1f%% (%d/%d indexed, %d failed, %v elapsed)",
		pct, s.Indexed, s.Total, s.Failed, elapsed.Round(time.Second))
}

// Reindex processes all documents with status tracking and failure recovery.
func Reindex(ctx context.Context, store vectorstore.VectorStore, docs []schema.Document, batchSize int) *IndexStatus {
	status := &IndexStatus{
		Total:     len(docs),
		StartTime: time.Now(),
	}

	for i := 0; i < len(docs); i += batchSize {
		end := i + batchSize
		if end > len(docs) {
			end = len(docs)
		}

		batch := docs[i:end]
		err := store.AddDocuments(ctx, batch)
		if err != nil {
			// Record individual failures.
			for _, doc := range batch {
				id, _ := doc.Metadata["id"].(string)
				status.RecordFailure(id)
			}
			slog.Error("batch failed", "start", i, "end", end, "error", err)
			continue
		}

		for range batch {
			status.RecordSuccess()
		}

		slog.Info("reindex progress", "status", status.Progress())
	}

	return status
}

func main() {
	ctx := context.Background()

	store, err := vectorstore.New("pgvector", vectorstore.ProviderConfig{
		ConnectionString: "postgres://user:pass@localhost:5432/vectors",
	})
	if err != nil {
		slog.Error("store creation failed", "error", err)
		return
	}

	// Create documents to reindex.
	docs := make([]schema.Document, 500)
	for i := range docs {
		docs[i] = schema.Document{
			Content:  fmt.Sprintf("Document %d content", i),
			Metadata: map[string]any{"id": fmt.Sprintf("doc_%d", i)},
		}
	}

	status := Reindex(ctx, store, docs, 50)
	fmt.Printf("Final: %s\n", status.Progress())

	if len(status.FailedIDs) > 0 {
		fmt.Printf("Failed documents: %v\n", status.FailedIDs)
		fmt.Println("Retry failed documents with: Reindex(ctx, store, failedDocs, 10)")
	}
}
```

---

## Hybrid Search with RRF Fusion

**Problem:** Neither pure vector search nor keyword search alone gives the best results. You want to combine both approaches.

Vector search excels at finding semantically similar content but can miss documents with exact keyword matches that a user expects. Keyword search (BM25) finds exact matches but misses semantically related content phrased differently. Reciprocal Rank Fusion (RRF) combines the ranked results from both approaches by scoring documents based on their rank position in each result list, weighted by a constant k (default 60). This hybrid approach consistently outperforms either method alone across diverse query types, which is why Beluga AI uses it as the default retrieval strategy.

**Solution:** Use the built-in hybrid retriever which combines vector similarity with BM25 scoring via Reciprocal Rank Fusion. The `WithFusionK()` option controls how much weight is given to rank positions.

```go
package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/rag/retriever"
)

func main() {
	ctx := context.Background()

	// Hybrid retriever combines vector and keyword search.
	hybrid := retriever.NewHybrid(
		retriever.WithVectorRetriever(vectorRetriever),
		retriever.WithKeywordRetriever(bm25Retriever),
		retriever.WithFusionK(60), // RRF constant (default 60).
	)

	docs, err := hybrid.Retrieve(ctx, "Go error handling best practices", 10)
	if err != nil {
		slog.Error("hybrid search failed", "error", err)
		return
	}

	for i, doc := range docs {
		fmt.Printf("#%d: %s\n", i+1, doc.Content[:80])
	}
}
```
