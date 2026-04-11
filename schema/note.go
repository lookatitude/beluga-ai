package schema

import "time"

// Note represents a Zettelkasten-style knowledge unit in associative memory.
// Each note captures a piece of content along with its semantic enrichment
// (keywords, tags, description), an embedding vector for similarity search,
// and bidirectional links to related notes.
type Note struct {
	// ID is the unique identifier for this note.
	ID string

	// Content is the raw text content of the note.
	Content string

	// Keywords are semantically extracted key terms from the content.
	Keywords []string

	// Tags are categorical labels assigned to the note.
	Tags []string

	// Description is a concise summary generated from the content.
	Description string

	// Embedding is the vector representation of the note content.
	// May be nil if the note has not been embedded.
	Embedding []float32

	// Links holds the IDs of notes that are semantically related to this one.
	Links []string

	// CreatedAt is the timestamp when the note was first created.
	CreatedAt time.Time

	// UpdatedAt is the timestamp of the most recent modification.
	UpdatedAt time.Time

	// Metadata holds arbitrary key-value pairs associated with this note.
	Metadata map[string]any
}
