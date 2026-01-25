package recursive

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/textsplitters/iface"
)

// RecursiveConfig is duplicated here to avoid import cycle.
type RecursiveConfig struct {
	LengthFunction func(string) int
	Separators     []string
	ChunkSize      int
	ChunkOverlap   int
}

// RecursiveCharacterTextSplitter splits text recursively using a hierarchy of separators.
type RecursiveCharacterTextSplitter struct {
	config *RecursiveConfig
	tracer trace.Tracer
}

// NewRecursiveCharacterTextSplitter creates a new RecursiveCharacterTextSplitter.
func NewRecursiveCharacterTextSplitter(config *RecursiveConfig) (*RecursiveCharacterTextSplitter, error) {
	// Basic validation
	if config.ChunkSize < 1 {
		return nil, newSplitterError("NewRecursiveCharacterTextSplitter", ErrCodeInvalidConfig, "ChunkSize must be >= 1", nil)
	}
	if config.ChunkOverlap < 0 || config.ChunkOverlap >= config.ChunkSize {
		return nil, newSplitterError("NewRecursiveCharacterTextSplitter", ErrCodeInvalidConfig, "ChunkOverlap must be >= 0 and < ChunkSize", nil)
	}

	// Set default separators if not provided
	if len(config.Separators) == 0 {
		config.Separators = []string{"\n\n", "\n", " ", ""}
	}

	return &RecursiveCharacterTextSplitter{
		config: config,
		tracer: otel.Tracer("github.com/lookatitude/beluga-ai/pkg/textsplitters/recursive"),
	}, nil
}

// SplitText implements the TextSplitter interface.
func (s *RecursiveCharacterTextSplitter) SplitText(ctx context.Context, text string) ([]string, error) {
	ctx, span := s.tracer.Start(ctx, "textsplitters.recursive.SplitText",
		trace.WithAttributes(
			attribute.String("splitter.type", "recursive_character"),
			attribute.Int("splitter.chunk_size", s.config.ChunkSize),
			attribute.Int("splitter.chunk_overlap", s.config.ChunkOverlap),
		))
	defer span.End()

	start := time.Now()

	if text == "" {
		span.RecordError(errors.New("empty text"))
		span.SetStatus(codes.Error, "empty text")
		return nil, newSplitterError("SplitText", ErrCodeEmptyInput, "text cannot be empty", nil)
	}

	lengthFn := s.config.LengthFunction
	if lengthFn == nil {
		lengthFn = func(s string) int { return len(s) }
	}

	// If text is smaller than chunk size, return as single chunk
	if lengthFn(text) <= s.config.ChunkSize {
		chunks := []string{text}
		span.SetAttributes(
			attribute.Int("splitter.output_count", len(chunks)),
			attribute.Int64("splitter.duration_ms", time.Since(start).Milliseconds()),
		)
		span.SetStatus(codes.Ok, "")
		return chunks, nil
	}

	chunks := s.splitTextRecursive(text, s.config.Separators, lengthFn)

	duration := time.Since(start)
	span.SetAttributes(
		attribute.Int("splitter.output_count", len(chunks)),
		attribute.Int64("splitter.duration_ms", duration.Milliseconds()),
	)
	span.SetStatus(codes.Ok, "")

	return chunks, nil
}

// splitTextRecursive recursively splits text using the separator hierarchy.
func (s *RecursiveCharacterTextSplitter) splitTextRecursive(text string, separators []string, lengthFn func(string) int) []string {
	if len(separators) == 0 {
		// Fallback: split character by character
		return s.splitByCharacters(text, lengthFn)
	}

	separator := separators[0]
	remainingSeparators := separators[1:]

	// Split by current separator
	parts := strings.Split(text, separator)

	// If we only got one part, try next separator
	if len(parts) == 1 {
		return s.splitTextRecursive(text, remainingSeparators, lengthFn)
	}

	var chunks []string
	var currentChunk strings.Builder

	for i, part := range parts {
		// Add separator back (except for empty separator and last part)
		if separator != "" && i < len(parts)-1 {
			part = part + separator
		}

		// Check if adding this part would exceed chunk size
		testChunk := currentChunk.String() + part
		if lengthFn(testChunk) > s.config.ChunkSize && currentChunk.Len() > 0 {
			// Save current chunk
			chunk := currentChunk.String()
			chunks = append(chunks, chunk)

			// Start new chunk with overlap
			if s.config.ChunkOverlap > 0 && len(chunks) > 0 {
				// Add overlap from previous chunk
				overlapText := s.getOverlapText(chunk, s.config.ChunkOverlap, lengthFn)
				currentChunk.Reset()
				currentChunk.WriteString(overlapText)
			} else {
				currentChunk.Reset()
			}
		}

		// If part itself is too large, recursively split it
		if lengthFn(part) > s.config.ChunkSize {
			subChunks := s.splitTextRecursive(part, remainingSeparators, lengthFn)
			// Add all but last sub-chunk
			for i := 0; i < len(subChunks)-1; i++ {
				if currentChunk.Len() > 0 {
					chunks = append(chunks, currentChunk.String())
					currentChunk.Reset()
				}
				chunks = append(chunks, subChunks[i])
			}
			// Last sub-chunk becomes the start of next chunk
			if len(subChunks) > 0 {
				currentChunk.WriteString(subChunks[len(subChunks)-1])
			}
		} else {
			currentChunk.WriteString(part)
		}
	}

	// Add final chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	// Ensure we have at least one chunk
	if len(chunks) == 0 {
		chunks = []string{text}
	}

	return chunks
}

// splitByCharacters splits text character by character as a fallback.
func (s *RecursiveCharacterTextSplitter) splitByCharacters(text string, lengthFn func(string) int) []string {
	var chunks []string
	var currentChunk strings.Builder

	for _, char := range text {
		testChunk := currentChunk.String() + string(char)
		if lengthFn(testChunk) > s.config.ChunkSize && currentChunk.Len() > 0 {
			chunks = append(chunks, currentChunk.String())
			currentChunk.Reset()

			// Add overlap
			if s.config.ChunkOverlap > 0 && len(chunks) > 0 {
				overlapText := s.getOverlapText(chunks[len(chunks)-1], s.config.ChunkOverlap, lengthFn)
				currentChunk.WriteString(overlapText)
			}
		}
		currentChunk.WriteRune(char)
	}

	if currentChunk.Len() > 0 {
		chunks = append(chunks, currentChunk.String())
	}

	if len(chunks) == 0 {
		chunks = []string{text}
	}

	return chunks
}

// getOverlapText extracts overlap text from the end of a chunk.
func (s *RecursiveCharacterTextSplitter) getOverlapText(chunk string, overlapSize int, lengthFn func(string) int) string {
	if lengthFn(chunk) <= overlapSize {
		return chunk
	}

	// Try to find a good break point (space, newline, etc.)
	for i := len(chunk) - overlapSize; i < len(chunk); i++ {
		if i > 0 && (chunk[i] == ' ' || chunk[i] == '\n') {
			return chunk[i+1:]
		}
	}

	// Fallback: just take the last N characters
	runes := []rune(chunk)
	if len(runes) <= overlapSize {
		return chunk
	}
	return string(runes[len(runes)-overlapSize:])
}

// SplitDocuments implements the TextSplitter interface.
func (s *RecursiveCharacterTextSplitter) SplitDocuments(
	ctx context.Context,
	documents []schema.Document,
) ([]schema.Document, error) {
	ctx, span := s.tracer.Start(ctx, "textsplitters.recursive.SplitDocuments",
		trace.WithAttributes(
			attribute.String("splitter.type", "recursive_character"),
			attribute.Int("splitter.input_count", len(documents)),
		))
	defer span.End()

	start := time.Now()
	var allChunks []schema.Document

	for _, doc := range documents {
		chunks, err := s.SplitText(ctx, doc.PageContent)
		if err != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
			return nil, err
		}

		// Create chunk documents with metadata
		for i, chunk := range chunks {
			chunkDoc := schema.Document{
				PageContent: chunk,
				Metadata:    make(map[string]string),
			}

			// Copy original metadata
			for k, v := range doc.Metadata {
				chunkDoc.Metadata[k] = v
			}

			// Add chunk metadata
			chunkDoc.Metadata["chunk_index"] = strconv.Itoa(i)
			chunkDoc.Metadata["chunk_total"] = strconv.Itoa(len(chunks))

			allChunks = append(allChunks, chunkDoc)
		}
	}

	duration := time.Since(start)
	span.SetAttributes(
		attribute.Int("splitter.output_count", len(allChunks)),
		attribute.Int64("splitter.duration_ms", duration.Milliseconds()),
	)
	span.SetStatus(codes.Ok, "")

	return allChunks, nil
}

// CreateDocuments implements the TextSplitter interface.
func (s *RecursiveCharacterTextSplitter) CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error) {
	ctx, span := s.tracer.Start(ctx, "textsplitters.recursive.CreateDocuments",
		trace.WithAttributes(
			attribute.String("splitter.type", "recursive_character"),
			attribute.Int("splitter.input_count", len(texts)),
		))
	defer span.End()

	// Create documents from texts and metadatas
	documents := make([]schema.Document, len(texts))
	for i, text := range texts {
		doc := schema.Document{
			PageContent: text,
			Metadata:    make(map[string]string),
		}

		// Add metadata if provided
		if i < len(metadatas) && metadatas[i] != nil {
			for k, v := range metadatas[i] {
				doc.Metadata[k] = fmt.Sprintf("%v", v)
			}
		}

		documents[i] = doc
	}

	// Split the documents
	return s.SplitDocuments(ctx, documents)
}

// Ensure RecursiveCharacterTextSplitter implements iface.TextSplitter.
var _ iface.TextSplitter = (*RecursiveCharacterTextSplitter)(nil)
