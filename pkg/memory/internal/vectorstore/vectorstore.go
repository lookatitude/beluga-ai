// Package vectorstore provides vector store memory implementations.
// It contains implementations that use vector stores for semantic retrieval.
package vectorstore

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/memory/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// VectorStoreMemory uses a vector store to retrieve relevant context from past interactions.
// It embeds the input/output and stores it, then retrieves similar documents during loading.
type VectorStoreMemory struct {
	Retriever     core.Retriever // Retriever interface (often backed by a VectorStore + Embedder)
	MemoryKey     string         // Key name for the retrieved documents variable in prompts
	InputKey      string         // Key name for the user input variable (used for retrieval query and saving)
	OutputKey     string         // Key name for the AI output variable (used for saving)
	ReturnDocs    bool           // If true, LoadMemoryVariables returns []schema.Document, otherwise a formatted string
	NumDocsToKeep int            // Number of relevant documents to retrieve
	// TODO: Add options for formatting retrieved documents into a string
	// TODO: Add options for creating the document to save (e.g., metadata)
}

// NewVectorStoreMemory creates a new VectorStoreMemory.
func NewVectorStoreMemory(retriever core.Retriever, memoryKey string, returnDocs bool, k int) *VectorStoreMemory {
	key := memoryKey
	if key == "" {
		key = "history" // Default memory key, though it holds retrieved docs
	}
	numDocs := k
	if numDocs <= 0 {
		numDocs = 4 // Default number of documents to retrieve
	}
	return &VectorStoreMemory{
		Retriever:     retriever,
		MemoryKey:     key,
		ReturnDocs:    returnDocs,
		NumDocsToKeep: numDocs,
	}
}

// MemoryVariables returns the key used for the retrieved documents.
func (m *VectorStoreMemory) MemoryVariables() []string {
	return []string{m.MemoryKey}
}

// formatDocuments formats retrieved documents into a single string.
// TODO: Make this configurable.
func formatDocuments(docs []schema.Message) string {
	formatted := "Relevant context:\n"
	var formattedSb54 strings.Builder
	for _, doc := range docs {
		// Assuming schema.Document can be treated like schema.Message for GetContent()
		// This might need adjustment based on actual schema.Document structure.
		_, _ = formattedSb54.WriteString(fmt.Sprintf("- %s\n", doc.GetContent()))
	}
	formatted += formattedSb54.String()
	return formatted
}

// LoadMemoryVariables retrieves documents relevant to the current input.
func (m *VectorStoreMemory) LoadMemoryVariables(ctx context.Context, inputs map[string]any) (map[string]any, error) {
	inputKey := m.InputKey
	var err error

	// Determine input key if not explicitly set
	if inputKey == "" {
		if len(inputs) == 1 {
			for k := range inputs {
				inputKey = k
			}
		} else {
			// Try common keys
			if _, ok := inputs["input"]; ok {
				inputKey = "input"
			} else if _, ok := inputs["question"]; ok {
				inputKey = "question"
			} else {
				return nil, errors.New("could not determine input key for retrieval query from multiple inputs")
			}
		}
	}

	queryVal, ok := inputs[inputKey]
	if !ok {
		return nil, errors.New("error")
	}
	queryStr, ok := queryVal.(string)
	if !ok {
		return nil, errors.New("error")
	}

	// Retrieve relevant documents
	// Assuming Retriever interface takes string query and returns []schema.Document
	// The concrete implementation (e.g., VectorStoreRetriever) handles embedding.
	docsAny, err := m.Retriever.Invoke(ctx, queryStr)
	if err != nil {
		return nil, errors.New("error")
	}

	docs, ok := docsAny.([]schema.Document)
	if !ok {
		return nil, errors.New("error")
	}

	// Limit number of documents if necessary (though retriever might handle this)
	if len(docs) > m.NumDocsToKeep {
		docs = docs[:m.NumDocsToKeep]
	}

	var memoryValue any
	if m.ReturnDocs {
		memoryValue = docs
	} else {
		// Convert []schema.Document to []schema.Message for formatting (assuming compatibility)
		// This is a simplification and might need adjustment.
		msgs := make([]schema.Message, len(docs))
		for i, d := range docs {
			msgs[i] = d // Direct assignment assumes Document implements Message or has compatible GetContent
		}
		memoryValue = formatDocuments(msgs)
	}

	return map[string]any{m.MemoryKey: memoryValue}, nil
}

// SaveContext embeds the input/output pair and saves it to the vector store.
func (m *VectorStoreMemory) SaveContext(ctx context.Context, inputs, outputs map[string]any) error {
	inputKey := m.InputKey
	outputKey := m.OutputKey

	// Determine input/output keys if not explicitly set
	if inputKey == "" || outputKey == "" {
		detectedInputKey, detectedOutputKey := getInputOutputKeys(inputs, outputs)
		if inputKey == "" {
			inputKey = detectedInputKey
		}
		if outputKey == "" {
			outputKey = detectedOutputKey
		}
	}

	inputVal, inputOk := inputs[inputKey]
	outputVal, outputOk := outputs[outputKey]

	if !inputOk {
		return errors.New("error")
	}
	if !outputOk {
		return errors.New("error")
	}

	inputStr, inputStrOk := inputVal.(string)
	outputStr, outputStrOk := outputVal.(string)

	if !inputStrOk {
		return errors.New("error")
	}
	if !outputStrOk {
		return errors.New("error")
	}

	// Combine input and output into a single document content for saving.
	// TODO: Make this formatting configurable.
	docContent := fmt.Sprintf("Input: %s\nOutput: %s", inputStr, outputStr)
	doc := schema.NewDocument(docContent, nil) // TODO: Add metadata?

	// Add the document to the vector store via the retriever (if it supports adding)
	// This assumes the retriever might wrap a vector store that has an AddDocuments method.
	// This is a conceptual dependency and might need a different approach, e.g., direct VectorStore access.
	if m.Retriever == nil {
		return errors.New("retriever is nil")
	}

	type DocumentAdder interface {
		AddDocuments(ctx context.Context, documents []schema.Document) ([]string, error)
	}

	adder, ok := m.Retriever.(DocumentAdder)
	if !ok {
		// If the retriever doesn't support adding, we can't save context this way.
		// Log a warning or return an error based on desired behavior.
		_, _ = fmt.Printf("Warning: Retriever used with VectorStoreMemory does not support adding documents. Context not saved to vector store.\n")
		return nil // Or return an error if saving is critical
	}

	_, err := adder.AddDocuments(ctx, []schema.Document{doc})
	if err != nil {
		return errors.New("error")
	}

	return nil
}

// Clear might be a no-op for vector store memory, as clearing might affect other uses.
// Alternatively, it could attempt to delete documents associated with a specific session ID if implemented.
func (m *VectorStoreMemory) Clear(ctx context.Context) error {
	// No-op for now. Implement deletion logic if the underlying vector store supports it
	// and if session management/metadata allows identifying documents to clear.
	_, _ = fmt.Println("Warning: Clear() called on VectorStoreMemory, but not implemented. Vector store content remains unchanged.")
	return nil
}

// getInputOutputKeys determines the input and output keys from the given maps.
func getInputOutputKeys(inputs, outputs map[string]any) (string, string) {
	if len(inputs) == 0 || len(outputs) == 0 {
		return "input", "output"
	}

	// Common input/output key names
	possibleInputKeys := []string{"input", "query", "question", "human_input", "user_input"}
	possibleOutputKeys := []string{"output", "result", "answer", "ai_output", "response"}

	// Try to find known input key
	var inputKey string
	for _, key := range possibleInputKeys {
		if _, ok := inputs[key]; ok {
			inputKey = key
			break
		}
	}

	// If no known input key, use the first key
	if inputKey == "" {
		for k := range inputs {
			inputKey = k
			break
		}
	}

	// Try to find known output key
	var outputKey string
	for _, key := range possibleOutputKeys {
		if _, ok := outputs[key]; ok {
			outputKey = key
			break
		}
	}

	// If no known output key, use the first key
	if outputKey == "" {
		for k := range outputs {
			outputKey = k
			break
		}
	}

	return inputKey, outputKey
}

// Ensure VectorStoreMemory implements the interface.
var _ iface.Memory = (*VectorStoreMemory)(nil)
