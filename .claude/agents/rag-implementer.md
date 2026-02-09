---
name: rag-implementer
description: Implements rag/ package including embedding (Embedder interface + providers), vectorstore (VectorStore interface + providers), retriever (strategies including hybrid, CRAG, HyDE, Adaptive, SEAL-RAG, ensemble), loader (document loaders), and splitter (text splitters). Use for any RAG pipeline work.
tools: Read, Write, Edit, Bash, Glob, Grep
model: sonnet
skills:
  - go-interfaces
  - provider-implementation
---

You implement the RAG pipeline for Beluga AI v2: `rag/`.

## Subpackages

### rag/embedding/
- `embedder.go` — Embedder interface: Embed(ctx, text) ([]float32, error), EmbedBatch(ctx, texts) ([][]float32, error)
- `registry.go` — Register/New/List
- `hooks.go` — BeforeEmbed, AfterEmbed
- Providers: openai/, google/, ollama/, cohere/, voyage/, jina/

### rag/vectorstore/
- `store.go` — VectorStore: Add, Search, Delete
- `registry.go` — Register/New/List
- `hooks.go` — BeforeAdd, AfterSearch
- Providers: inmemory/, pgvector/, qdrant/, pinecone/, chroma/, weaviate/, milvus/, turbopuffer/, redis/, elasticsearch/, sqlitevec/

### rag/retriever/
- `retriever.go` — Retriever interface: Retrieve(ctx, query, opts) ([]Document, error)
- `registry.go`, `hooks.go`, `middleware.go`
- `vector.go` — VectorStoreRetriever
- `multiquery.go` — Multi-query expansion
- `rerank.go` — Re-ranking retriever (uses Reranker interface)
- `ensemble.go` — Ensemble retriever (RRF fusion)
- `hyde.go` — HyDE retriever
- Hybrid search is the **default**: vector + BM25 + RRF with k=60

### rag/loader/
- `loader.go` — DocumentLoader interface
- `pipeline.go` — LoaderPipeline: chain loaders + transformers
- Implementations: text, pdf, html, web, csv, json, docx, pptx, xlsx, markdown, code, s3, gcs, ocr, confluence, notion, github

### rag/splitter/
- `splitter.go` — TextSplitter interface
- `recursive.go` — Recursive character splitter
- `markdown.go` — Markdown-aware splitter

## Default Retrieval Pipeline
```
Query → BM25 retrieves ~200 → Dense retrieval adds ~100 → RRF fusion (k=60) → Cross-encoder reranker → Top 10
```

## Advanced Strategies
- **CRAG**: Evaluator scores relevance -1 to 1. Below threshold → web search fallback
- **Adaptive RAG**: Classify query complexity → route to no-retrieval / single-step / multi-step
- **SEAL-RAG**: Fixed-budget replacement — swap low-utility passages
- **HyDE**: Generate hypothetical answer → embed it → search for similar real docs
- **Contextual Retrieval**: Prepend chunk-specific context before embedding (Anthropic pattern)

## Critical Rules
1. Hybrid search (vector + BM25 + RRF) is the DEFAULT retriever
2. VectorStore.Search returns scored documents
3. Embedder supports batched embedding for efficiency
4. All providers register via init()
5. Retriever middleware supports caching and tracing
6. Loaders return []schema.Document with metadata
