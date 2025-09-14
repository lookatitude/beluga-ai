// Package schema defines core data structures used throughout the framework.
package schema

// Document represents a piece of text, often with associated metadata.
// It can be used for retrieved context in RAG or as input/output for chains.
type Document struct {
	PageContent string
	Metadata    map[string]any
}

func NewDocument(pageContent string, metadata map[string]any) Document {
	if metadata == nil {
		metadata = make(map[string]any)
	}
	return Document{PageContent: pageContent, Metadata: metadata}
}

// GetContent returns the main text content of the document.
// This allows Document to be used somewhat interchangeably with Message in some contexts.
func (d Document) GetContent() string {
	return d.PageContent
}

// GetType returns a specific type identifier for Document, fulfilling the Message interface.
// Note: This might be conceptually slightly awkward, but allows using Documents where Messages are expected.
// Consider if a different interface or approach is better long-term.
func (d Document) GetType() MessageType {
	// Assigning a specific type, or perhaps a new "DocumentType" constant?
	// For now, let's use Generic, as it's a piece of text content.
	return MessageTypeGeneric // Or potentially a new MessageTypeDocument
}

// GetAdditionalArgs returns additional arguments associated with the message.
// For Document, this is typically nil or an empty map.
func (d Document) GetAdditionalArgs() map[string]any {
	return nil // Or return d.Metadata if that's more appropriate for the interface's intent
}

// Ensure Document implements the Message interface
var _ Message = (*Document)(nil)
