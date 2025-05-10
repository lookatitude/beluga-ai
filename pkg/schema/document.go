package schema

// Document represents a piece of text with associated metadata.
type Document struct {
	PageContent string            `json:"page_content"`
	Metadata    map[string]string `json:"metadata"`
	Score       float32           `json:"score,omitempty"` // Optional score, e.g., for search results
}

// NewDocument creates a new Document.
func NewDocument(pageContent string, metadata map[string]string) Document {
	return Document{
		PageContent: pageContent,
		Metadata:    metadata,
	}
}

