// Package splitters provides implementations of the rag.Splitter interface.
package splitters

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/pkg/dataconnection"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

// CharacterSplitter splits text based on a fixed number of characters.
type CharacterSplitter struct {
	ChunkSize    int
	ChunkOverlap int
	Separator    string // Separator used to join chunks if needed (often "\n\n")
	// TODO: Add length function (e.g., rune count vs byte count)
	// TODO: Add option to keep separator in chunks
}

// Default values for CharacterSplitter
const (
	defaultChunkSize    = 1000
	defaultChunkOverlap = 200
	defaultSeparator    = "\n\n"
)

// NewCharacterSplitter creates a new CharacterSplitter.
func NewCharacterSplitter(options ...func(*CharacterSplitter)) *CharacterSplitter {
	splitter := &CharacterSplitter{
		ChunkSize:    defaultChunkSize,
		ChunkOverlap: defaultChunkOverlap,
		Separator:    defaultSeparator,
	}
	for _, opt := range options {
		opt(splitter)
	}
	if splitter.ChunkOverlap >= splitter.ChunkSize {
		fmt.Printf("Warning: ChunkOverlap (%d) >= ChunkSize (%d). Setting overlap to 0.\n", splitter.ChunkOverlap, splitter.ChunkSize)
		splitter.ChunkOverlap = 0
	}
	return splitter
}

// WithChunkSize sets the chunk size.
func WithChunkSize(size int) func(*CharacterSplitter) {
	return func(s *CharacterSplitter) {
		if size > 0 {
			s.ChunkSize = size
		}
	}
}

// WithChunkOverlap sets the chunk overlap.
func WithChunkOverlap(overlap int) func(*CharacterSplitter) {
	return func(s *CharacterSplitter) {
		if overlap >= 0 {
			s.ChunkOverlap = overlap
		}
	}
}

// WithSeparator sets the separator.
func WithSeparator(separator string) func(*CharacterSplitter) {
	return func(s *CharacterSplitter) {
		s.Separator = separator
	}
}

// SplitText splits a single text string into chunks.
func (s *CharacterSplitter) SplitText(ctx context.Context, text string) ([]string, error) {
	if text == "" {
		return []string{}, nil
	}

	chunks := make([]string, 0)
	start := 0
	textLen := len(text) // Using byte length for simplicity, consider rune length

	for start < textLen {
		end := start + s.ChunkSize
		if end > textLen {
			end = textLen
		}
		chunks = append(chunks, text[start:end])

		// Move start for the next chunk
		nextStart := start + s.ChunkSize - s.ChunkOverlap
		// Ensure nextStart doesn_t go backward or stay the same if overlap is large or chunksize small
		if nextStart <= start {
			nextStart = start + 1 // Move forward by at least one character
		}
		// Prevent infinite loop if chunk size is very small / overlap large
		if nextStart >= end && end != textLen {
			// This case should ideally be handled by the check in NewCharacterSplitter,
			// but as a safeguard, just move to the end of the current chunk.
			nextStart = end
		}
		start = nextStart
	}
	return chunks, nil
}

// mergeSplits combines text chunks with the specified separator and overlap logic.
// This is a helper, might not be directly needed by the interface but useful internally.
// func (s *CharacterSplitter) mergeSplits(splits []string) string {
// 	 // Implementation depends on how overlap is handled during merging
// 	 return strings.Join(splits, s.Separator)
// }

// SplitDocuments splits existing documents into smaller ones.
func (s *CharacterSplitter) SplitDocuments(ctx context.Context, documents []schema.Document) ([]schema.Document, error) {
	newDocs := make([]schema.Document, 0)
	for _, doc := range documents {
		chunks, err := s.SplitText(ctx, doc.PageContent) // Assuming PageContent is the field for content
		if err != nil {
			// Use doc.Metadata directly if GetMetadata is not a method
			source := "unknown"
			if doc.Metadata != nil && doc.Metadata["source"] != nil {
				if srcStr, ok := doc.Metadata["source"].(string); ok {
					source = srcStr
				}
			}
			return nil, fmt.Errorf("failed to split text for document from source %s: %w", source, err)
		}

		// Create new documents for each chunk, preserving metadata
		for i, chunk := range chunks {
			newMetadata := make(map[string]any)
			// Use doc.Metadata directly
			if doc.Metadata != nil {
				for k, v := range doc.Metadata {
					newMetadata[k] = v
				}
			}
			// Optionally add chunk number or other split-specific metadata
			newMetadata["chunk_index"] = i
			newDocs = append(newDocs, schema.NewDocument(chunk, newMetadata))
		}
	}
	return newDocs, nil
}

// CreateDocuments splits raw texts and creates new documents.
func (s *CharacterSplitter) CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error) {
	if metadatas != nil && len(texts) != len(metadatas) {
		return nil, fmt.Errorf("number of texts (%d) must match number of metadatas (%d)", len(texts), len(metadatas))
	}

	newDocs := make([]schema.Document, 0)
	for i, text := range texts {
		chunks, err := s.SplitText(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to split text at index %d: %w", i, err)
		}

		baseMetadata := make(map[string]any)
		if metadatas != nil && i < len(metadatas) && metadatas[i] != nil { // Added nil check for metadatas[i]
			baseMetadata = metadatas[i]
		}

		for j, chunk := range chunks {
			newMetadata := make(map[string]any)
			for k, v := range baseMetadata {
				newMetadata[k] = v
			}
			newMetadata["chunk_index"] = j
			newDocs = append(newDocs, schema.NewDocument(chunk, newMetadata))
		}
	}
	return newDocs, nil
}

// Ensure CharacterSplitter implements the interface.
var _ rag.Splitter = (*CharacterSplitter)(nil)

