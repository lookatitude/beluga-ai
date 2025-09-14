package internal

import (
	"github.com/lookatitude/beluga-ai/pkg/schema/iface"
)

// Document represents a piece of text with associated metadata.
type Document struct {
	ID          string            `json:"id,omitempty"` // Optional unique ID for the document
	PageContent string            `json:"page_content"`
	Metadata    map[string]string `json:"metadata"`
	Embedding   []float32         `json:"embedding,omitempty"` // Optional embedding vector for the document
	Score       float32           `json:"score,omitempty"`     // Optional score, e.g., for search results
}

// NewDocument creates a new Document.
func NewDocument(pageContent string, metadata map[string]string) Document {
	return Document{
		PageContent: pageContent,
		Metadata:    metadata,
	}
}

// NewDocumentWithID creates a new Document with an ID.
func NewDocumentWithID(id string, pageContent string, metadata map[string]string) Document {
	return Document{
		ID:          id,
		PageContent: pageContent,
		Metadata:    metadata,
	}
}

// GetType returns the message type for a document (treated as system message).
func (d Document) GetType() MessageType {
	return RoleSystem
}

// GetContent returns the page content of the document.
func (d Document) GetContent() string {
	return d.PageContent
}

// ToolCalls returns an empty slice for documents.
func (d Document) ToolCalls() []ToolCall {
	return nil
}

// AdditionalArgs returns an empty map for documents.
func (d Document) AdditionalArgs() map[string]interface{} {
	return make(map[string]interface{})
}

// Ensure Document implements the Message interface.
var _ iface.Message = (*Document)(nil)
