// Package vectorstore provides vector store memory implementations.
// This file re-exports types from internal/vectorstore for the provider pattern.
package vectorstore

import (
	"github.com/lookatitude/beluga-ai/pkg/memory/internal/vectorstore"
)

// Type aliases for backward compatibility and provider pattern.
type (
	// VectorStoreMemory uses a vector store to retrieve relevant context from past interactions.
	VectorStoreMemory = vectorstore.VectorStoreMemory

	// VectorStoreRetrieverMemory is a memory that uses a vector store retriever.
	VectorStoreRetrieverMemory = vectorstore.VectorStoreRetrieverMemory

	// VectorStoreMemoryOption is a functional option for configuring VectorStoreRetrieverMemory.
	VectorStoreMemoryOption = vectorstore.VectorStoreMemoryOption
)

// NewVectorStoreMemory creates a new VectorStoreMemory.
var NewVectorStoreMemory = vectorstore.NewVectorStoreMemory

// NewVectorStoreRetrieverMemory creates a new VectorStoreRetrieverMemory.
var NewVectorStoreRetrieverMemory = vectorstore.NewVectorStoreRetrieverMemory

// Functional options for VectorStoreRetrieverMemory.
var (
	WithMemoryKey       = vectorstore.WithMemoryKey
	WithInputKey        = vectorstore.WithInputKey
	WithOutputKey       = vectorstore.WithOutputKey
	WithReturnDocs      = vectorstore.WithReturnDocs
	WithExcludeInputKey = vectorstore.WithExcludeInputKey
	WithK               = vectorstore.WithK
)
