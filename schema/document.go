package schema

// Document represents a piece of content for RAG (Retrieval-Augmented Generation).
// Documents are the primary unit of storage and retrieval in the RAG pipeline,
// carrying text content, metadata for filtering, optional relevance scores from
// retrieval, and optional embedding vectors.
type Document struct {
	// ID is the unique identifier for this document.
	ID string
	// Content is the text content of the document.
	Content string
	// Metadata holds arbitrary key-value pairs associated with the document,
	// used for filtering and contextual information.
	Metadata map[string]any
	// Score is the relevance score assigned by a retriever. Zero if not yet scored.
	Score float64
	// Embedding is the vector embedding of the document content.
	// May be nil if the document has not been embedded.
	Embedding []float32
}
