package schema

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

