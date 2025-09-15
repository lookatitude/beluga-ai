// Package vectorstore provides vector store memory implementations.
// It contains implementations that use vector stores for semantic retrieval.
package vectorstore

import (
	"context"
	"fmt"

	embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

// VectorStoreRetrieverMemory is a type of memory that retrieves context from a vector store.
// It uses an embedder to create vector representations of queries and a vector store
// to store and retrieve document embeddings.
type VectorStoreRetrieverMemory struct {
	Embedder        embeddingsiface.Embedder
	VectorStore     vectorstores.VectorStore
	MemoryKey       string // Key to use for the memory variables in input/output
	InputKey        string // Key for the input to the chain
	OutputKey       string // Key for the output of the chain
	ReturnDocs      bool   // Whether or not to return the docs
	ExcludeInputKey bool   // If InputKey should be excluded from the LLM input
	TopK            int    // Number of documents to retrieve
	// NameSpace       string // Namespace is typically handled by the VectorStore implementation or config, not as a generic option here
}

// VectorStoreMemoryOption is a function type for setting options on a VectorStoreRetrieverMemory.
type VectorStoreMemoryOption func(*VectorStoreRetrieverMemory)

// NewVectorStoreRetrieverMemory creates a new VectorStoreRetrieverMemory.
// It requires an embedder and a vector store, and can take optional configuration functions.
func NewVectorStoreRetrieverMemory(embedder embeddingsiface.Embedder, vectorStore vectorstores.VectorStore, options ...VectorStoreMemoryOption) *VectorStoreRetrieverMemory {
	m := &VectorStoreRetrieverMemory{
		Embedder:        embedder,
		VectorStore:     vectorStore,
		MemoryKey:       "history", // Default value
		InputKey:        "input",   // Default value
		OutputKey:       "output",  // Default value
		ReturnDocs:      false,     // Default value
		ExcludeInputKey: false,     // Default value
		TopK:            5,         // Default value
	}

	for _, option := range options {
		option(m)
	}

	return m
}

// WithMemoryKey sets the memory key for the VectorStoreRetrieverMemory.
func WithMemoryKey(key string) VectorStoreMemoryOption {
	return func(m *VectorStoreRetrieverMemory) {
		m.MemoryKey = key
	}
}

// WithInputKey sets the input key for the VectorStoreRetrieverMemory.
func WithInputKey(key string) VectorStoreMemoryOption {
	return func(m *VectorStoreRetrieverMemory) {
		m.InputKey = key
	}
}

// WithOutputKey sets the output key for the VectorStoreRetrieverMemory.
func WithOutputKey(key string) VectorStoreMemoryOption {
	return func(m *VectorStoreRetrieverMemory) {
		m.OutputKey = key
	}
}

// WithReturnDocs sets whether to return documents for the VectorStoreRetrieverMemory.
func WithReturnDocs(returnDocs bool) VectorStoreMemoryOption {
	return func(m *VectorStoreRetrieverMemory) {
		m.ReturnDocs = returnDocs
	}
}

// WithExcludeInputKey sets whether to exclude the input key for the VectorStoreRetrieverMemory.
func WithExcludeInputKey(exclude bool) VectorStoreMemoryOption {
	return func(m *VectorStoreRetrieverMemory) {
		m.ExcludeInputKey = exclude
	}
}

// WithK sets the number of documents to retrieve (top K) for the VectorStoreRetrieverMemory.
func WithK(k int) VectorStoreMemoryOption {
	return func(m *VectorStoreRetrieverMemory) {
		m.TopK = k
	}
}

// MemoryVariables returns the memory key.
func (m *VectorStoreRetrieverMemory) MemoryVariables() []string {
	return []string{m.MemoryKey}
}

// LoadMemoryVariables retrieves the memory variables for the given input.
// It embeds the input query, searches the vector store, and formats the retrieved documents.
func (m *VectorStoreRetrieverMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	query, ok := inputs[m.InputKey].(string)
	if !ok {
		return nil, fmt.Errorf(
			"error")
	}

	// Use SimilaritySearchByQuery from the VectorStore interface
	docs, _, err := m.VectorStore.SimilaritySearchByQuery(ctx, query, m.TopK, m.Embedder)
	if err != nil {
		return nil, fmt.Errorf("error")
	}

	var relevantHistory string
	for _, doc := range docs {
		relevantHistory += doc.PageContent + "\n"
	}

	memoryVariables := map[string]any{m.MemoryKey: relevantHistory}
	if m.ReturnDocs {
		// Ensure the key for documents is distinct and clear
		memoryVariables["retrieved_docs"] = docs
	}

	return memoryVariables, nil
}

// SaveContext saves the context of the current interaction to the vector store.
// It creates documents from the input and output and adds them to the vector store.
func (m *VectorStoreRetrieverMemory) SaveContext(ctx context.Context, inputs map[string]any, outputs map[string]any) error {
	inputVal, inputOk := inputs[m.InputKey].(string)
	outputVal, outputOk := outputs[m.OutputKey].(string)

	if !inputOk {
		return fmt.Errorf(
			"error")
	}
	if !outputOk {
		return fmt.Errorf(
			"error")
	}

	interactionContent := fmt.Sprintf("Input: %s\nOutput: %s", inputVal, outputVal)
	// Document metadata must be map[string]string
	doc := schema.Document{
		PageContent: interactionContent,
		Metadata:    map[string]string{"input": inputVal, "output": outputVal, "source": "conversation"},
	}

	// Use AddDocuments from the VectorStore interface, passing the embedder
	_, err := m.VectorStore.AddDocuments(ctx, []schema.Document{doc}, vectorstores.WithEmbedder(m.Embedder))
	if err != nil {
		return fmt.Errorf("error")
	}
	return nil
}

// Clear effectively clears the memory. For VectorStoreRetrieverMemory, this might mean
// clearing the associated namespace in the vector store if supported, or it might be a no-op
// if the vector store is managed externally or shared.
func (m *VectorStoreRetrieverMemory) Clear(ctx context.Context) error {
	fmt.Println("VectorStoreRetrieverMemory.Clear() called - specific implementation depends on VectorStore capabilities.")
	return nil
}

// Ensure VectorStoreRetrieverMemory implements the Memory interface.
var _ iface.Memory = (*VectorStoreRetrieverMemory)(nil)
