# In-Memory Vector Store Provider Example

This example demonstrates how to use the in-memory vector store provider with Beluga AI.

## Prerequisites

- Go 1.21+

## Running the Example

```bash
go run main.go
```

## What This Example Shows

1. Creating an in-memory vector store with an embedder
2. Adding documents to the store
3. Searching by text query (using embeddings)
4. Searching by pre-computed vectors

## Key Features

- **No external dependencies**: In-memory store works without databases
- **Fast**: All operations happen in memory
- **Simple**: Perfect for development and testing
- **Embedder support**: Can use any embedder for text-based operations

## Use Cases

- Development and testing
- Small-scale applications
- Prototyping RAG pipelines
- Educational examples

## Limitations

- Data is lost when the process exits
- Not suitable for production at scale
- Memory usage grows with document count

## See Also

- [Vectorstores Package Documentation](../../../pkg/vectorstores/README.md)
- [In-Memory Provider Documentation](../../../pkg/vectorstores/providers/inmemory/README.md)
