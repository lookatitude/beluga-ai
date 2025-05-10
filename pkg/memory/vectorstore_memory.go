package memory

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/vectorstores"
)

// VectorStoreRetrieverMemory is a type of memory that retrieves context from a vector store.
// It uses an embedder to create vector representations of queries and a vector store
// to store and retrieve document embeddings.
type VectorStoreRetrieverMemory struct {
	Embedder        iface.Embedder
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

// NewVectorStoreMemory creates a new VectorStoreRetrieverMemory.
// It requires an embedder and a vector store, and can take optional configuration functions.
func NewVectorStoreMemory(embedder iface.Embedder, vectorStore vectorstores.VectorStore, options ...VectorStoreMemoryOption) *VectorStoreRetrieverMemory {
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

// GetMemoryVariables returns the memory key.
func (m *VectorStoreRetrieverMemory) GetMemoryVariables(ctx context.Context, inputs map[string]interface{}) ([]string, error) {
	return []string{m.MemoryKey}, nil
}

// LoadMemoryVariables retrieves the memory variables for the given input.
// It embeds the input query, searches the vector store, and formats the retrieved documents.
func (m *VectorStoreRetrieverMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]interface{}) (map[string]interface{}, error) {
	query, ok := inputs[m.InputKey].(string)
	if !ok {
		return nil, fmt.Errorf("input key 	%s	 not found in inputs or not a string", m.InputKey)
	}

	// Use SimilaritySearchByQuery from the VectorStore interface
	docs, _, err := m.VectorStore.SimilaritySearchByQuery(ctx, query, m.TopK, m.Embedder)
	if err != nil {
		return nil, fmt.Errorf("failed to perform similarity search by query: %w", err)
	}

	var relevantHistory string
	for _, doc := range docs {
		relevantHistory += doc.PageContent + "\n"
	}

	memoryVariables := map[string]interface{}{m.MemoryKey: relevantHistory}
	if m.ReturnDocs {
		// Ensure the key for documents is distinct and clear
		memoryVariables["retrieved_docs"] = docs
	}

	return memoryVariables, nil
}

// SaveContext saves the context of the current interaction to the vector store.
// It creates documents from the input and output and adds them to the vector store.
// The Memory interface expects outputs to be map[string]string.
func (m *VectorStoreRetrieverMemory) SaveContext(ctx context.Context, inputs map[string]interface{}, outputs map[string]string) error {
	inputVal, inputOk := inputs[m.InputKey].(string)
	outputVal, outputOk := outputs[m.OutputKey]

	if !inputOk {
		return fmt.Errorf("input key 	%s	 not found in inputs or not a string", m.InputKey)
	}
	if !outputOk {
		return fmt.Errorf("output key 	%s	 not found in outputs", m.OutputKey)
	}

	interactionContent := fmt.Sprintf("Input: %s\nOutput: %s", inputVal, outputVal)
	// Document metadata must be map[string]string
	doc := schema.Document{
		PageContent: interactionContent,
		Metadata:    map[string]string{"input": inputVal, "output": outputVal, "source": "conversation"},
	}

	// Use AddDocuments from the VectorStore interface, passing the embedder
	err := m.VectorStore.AddDocuments(ctx, []schema.Document{doc}, m.Embedder)
	if err != nil {
		return fmt.Errorf("failed to add document to vector store: %w", err)
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

// GetMemoryType returns the type of memory.
func (m *VectorStoreRetrieverMemory) GetMemoryType() string {
	return "vectorstore"
}

// Ensure VectorStoreRetrieverMemory implements the Memory interface.
var _ Memory = (*VectorStoreRetrieverMemory)(nil)

