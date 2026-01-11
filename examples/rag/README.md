# RAG Pipeline Examples

This directory contains examples demonstrating Retrieval-Augmented Generation (RAG) pipelines in the Beluga AI Framework.

## Examples

### [simple](simple/)

Complete RAG pipeline from document ingestion to generation.

**What you'll learn:**
- Document preparation
- Embedding generation
- Vector store operations
- Query and retrieval
- Context-augmented generation

**Run:**
```bash
cd simple
go run main.go
```

### [with_memory](with_memory/)

RAG pipeline with conversation memory for multi-turn interactions.

**What you'll learn:**
- Combining RAG with memory
- Multi-turn RAG conversations
- Context building from multiple sources

**Run:**
```bash
cd with_memory
go run main.go
```

### [advanced](advanced/)

Advanced RAG patterns including filtering and multiple retrieval strategies.

**What you'll learn:**
- Advanced retrieval strategies
- Metadata filtering
- Score thresholds
- Multiple retrieval methods

**Run:**
```bash
cd advanced
go run main.go
```

### [with_loaders](with_loaders/)

RAG pipeline using document loaders and text splitters for real-world document ingestion.

**What you'll learn:**
- Loading documents from directories and files using `documentloaders`
- Splitting documents into chunks using `textsplitters`
- Integrating loaders and splitters into RAG pipeline
- Chunk metadata preservation
- Configuring loaders (depth, extensions, concurrency)
- Configuring splitters (chunk size, overlap, separators)

**Run:**
```bash
cd with_loaders
go run main.go
```

**Related Examples:**
- See `examples/documentloaders/` for more loader examples
- See `examples/textsplitters/` for more splitter examples

## Prerequisites

- Go 1.21 or later
- Beluga AI Framework
- (Optional) OpenAI API key for embeddings and LLM

## Learning Path

1. Start with `simple` to understand RAG fundamentals
2. Add `with_memory` for conversation context
3. Use `with_loaders` to learn document ingestion patterns with loaders and splitters
4. Explore `advanced` for optimization techniques

## Document Ingestion

For real-world RAG applications, use the `documentloaders` and `textsplitters` packages:

- **Document Loaders**: Load documents from files, directories, and other sources
  - `RecursiveDirectoryLoader`: Recursively load files from directories
  - `TextLoader`: Load single text files
  - See `examples/documentloaders/` for usage examples

- **Text Splitters**: Split documents into chunks for embedding
  - `RecursiveCharacterTextSplitter`: General-purpose text splitting
  - `MarkdownTextSplitter`: Markdown-aware splitting
  - See `examples/textsplitters/` for usage examples

**Migration Guide**: If you're currently creating documents manually, consider migrating to loaders:
- Replace manual document creation with `documentloaders.NewDirectoryLoader()`
- Replace manual text splitting with `textsplitters.NewRecursiveCharacterTextSplitter()`
- See `with_loaders` example for the complete pattern

## Related Documentation

- [RAG Concepts](../../docs/concepts/rag.md)
- [RAG Recipes](../../docs/cookbook/rag-recipes.md)
- [Document Loading](../../docs/concepts/document-loading.md) (coming soon)
- [Text Splitting](../../docs/concepts/text-splitting.md) (coming soon)
- [Vector Stores](../../docs/providers/vectorstores/)
