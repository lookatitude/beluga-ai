---
title: RAG Recipes
description: Practical recipes for building retrieval-augmented generation pipelines.
sidebar:
  order: 0
---

## Parent Document Retrieval

**Problem:** Small chunks improve embedding precision, but the LLM needs larger context to generate good answers. You want to retrieve small chunks but return their parent documents.

**Solution:** Index small chunks with metadata pointing to their parent, then fetch parents at retrieval time.

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

**Solution:** Use a reranker in the retriever pipeline to refine results after initial retrieval.

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

**Solution:** Use goroutines with a semaphore to load documents in parallel.

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

**Solution:** Wrap loaders with error handling that skips bad documents and logs failures.

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

**Solution:** Chunk documents into batches and embed them with controlled concurrency.

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

**Solution:** Use the vector store's metadata filter when querying.

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

**Solution:** Use `splitter.NewCode` which understands language-specific boundaries.

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

**Solution:** Use sentence-aware splitting that respects natural boundaries.

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

**Solution:** Track indexed document IDs and support incremental reindexing.

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

**Solution:** Use the built-in hybrid retriever which combines vector similarity with BM25 scoring via Reciprocal Rank Fusion.

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
