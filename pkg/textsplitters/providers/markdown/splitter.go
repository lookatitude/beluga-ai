package markdown

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/lookatitude/beluga-ai/pkg/schema"
	"github.com/lookatitude/beluga-ai/pkg/textsplitters/iface"
)

// MarkdownConfig is duplicated here to avoid import cycle.
type MarkdownConfig struct {
	ChunkSize        int
	ChunkOverlap     int
	LengthFunction   func(string) int
	HeadersToSplitOn []string
	ReturnEachLine   bool
}

// MarkdownTextSplitter splits markdown text respecting header boundaries and code blocks.
type MarkdownTextSplitter struct {
	config *MarkdownConfig
	tracer trace.Tracer
}

// NewMarkdownTextSplitter creates a new MarkdownTextSplitter.
func NewMarkdownTextSplitter(config *MarkdownConfig) (*MarkdownTextSplitter, error) {
	// Basic validation
	if config.ChunkSize < 1 {
		return nil, newSplitterError("NewMarkdownTextSplitter", ErrCodeInvalidConfig, "ChunkSize must be >= 1", nil)
	}
	if config.ChunkOverlap < 0 || config.ChunkOverlap >= config.ChunkSize {
		return nil, newSplitterError("NewMarkdownTextSplitter", ErrCodeInvalidConfig, "ChunkOverlap must be >= 0 and < ChunkSize", nil)
	}

	// Set default headers if not provided
	if len(config.HeadersToSplitOn) == 0 {
		config.HeadersToSplitOn = []string{"#", "##", "###", "####", "#####", "######"}
	}

	return &MarkdownTextSplitter{
		config: config,
		tracer: otel.Tracer("github.com/lookatitude/beluga-ai/pkg/textsplitters/markdown"),
	}, nil
}

// SplitText implements the TextSplitter interface.
func (s *MarkdownTextSplitter) SplitText(ctx context.Context, text string) ([]string, error) {
	ctx, span := s.tracer.Start(ctx, "textsplitters.markdown.SplitText",
		trace.WithAttributes(
			attribute.String("splitter.type", "markdown"),
			attribute.Int("splitter.chunk_size", s.config.ChunkSize),
		))
	defer span.End()

	start := time.Now()

	if text == "" {
		span.RecordError(fmt.Errorf("empty text"))
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

	// Split by headers first
	chunks := s.splitByHeaders(text, lengthFn)

	duration := time.Since(start)
	span.SetAttributes(
		attribute.Int("splitter.output_count", len(chunks)),
		attribute.Int64("splitter.duration_ms", duration.Milliseconds()),
	)
	span.SetStatus(codes.Ok, "")

	return chunks, nil
}

// splitByHeaders splits markdown text at header boundaries.
func (s *MarkdownTextSplitter) splitByHeaders(text string, lengthFn func(string) int) []string {
	// Build regex pattern for headers
	headerPattern := s.buildHeaderPattern()

	// Find all header positions
	lines := strings.Split(text, "\n")
	var chunks []string
	var currentChunk strings.Builder
	var inCodeBlock bool

	for i, line := range lines {
		// Track code blocks
		if strings.HasPrefix(line, "```") {
			inCodeBlock = !inCodeBlock
		}

		// Check if line is a header (and not in code block)
		isHeader := !inCodeBlock && headerPattern.MatchString(line)

		// If we hit a header and current chunk is getting large, split
		if isHeader && lengthFn(currentChunk.String()) > s.config.ChunkSize && currentChunk.Len() > 0 {
			chunk := currentChunk.String()
			chunks = append(chunks, strings.TrimSpace(chunk))
			currentChunk.Reset()

			// Add overlap if configured
			if s.config.ChunkOverlap > 0 && len(chunks) > 0 {
				overlapText := s.getOverlapText(chunks[len(chunks)-1], s.config.ChunkOverlap, lengthFn)
				currentChunk.WriteString(overlapText)
			}
		}

		currentChunk.WriteString(line)
		if i < len(lines)-1 {
			currentChunk.WriteString("\n")
		}

		// If chunk exceeds size limit, force split (preserving code blocks if possible)
		if lengthFn(currentChunk.String()) > s.config.ChunkSize && !inCodeBlock {
			// Try to find a good split point
			chunkText := currentChunk.String()
			splitPoint := s.findSplitPoint(chunkText, lengthFn)
			if splitPoint > 0 {
				chunks = append(chunks, strings.TrimSpace(chunkText[:splitPoint]))
				remaining := chunkText[splitPoint:]
				currentChunk.Reset()
				if s.config.ChunkOverlap > 0 {
					overlapText := s.getOverlapText(chunks[len(chunks)-1], s.config.ChunkOverlap, lengthFn)
					currentChunk.WriteString(overlapText)
				}
				currentChunk.WriteString(remaining)
			}
		}
	}

	// Add final chunk
	if currentChunk.Len() > 0 {
		chunks = append(chunks, strings.TrimSpace(currentChunk.String()))
	}

	// Ensure we have at least one chunk
	if len(chunks) == 0 {
		chunks = []string{text}
	}

	return chunks
}

// buildHeaderPattern builds a regex pattern to match markdown headers.
func (s *MarkdownTextSplitter) buildHeaderPattern() *regexp.Regexp {
	var patterns []string
	for _, header := range s.config.HeadersToSplitOn {
		// Escape special regex characters
		escaped := regexp.QuoteMeta(header)
		patterns = append(patterns, "^"+escaped+"\\s+")
	}
	pattern := strings.Join(patterns, "|")
	return regexp.MustCompile(pattern)
}

// findSplitPoint finds a good point to split text while respecting chunk size.
func (s *MarkdownTextSplitter) findSplitPoint(text string, lengthFn func(string) int) int {
	targetSize := s.config.ChunkSize - s.config.ChunkOverlap
	if lengthFn(text) <= targetSize {
		return len(text)
	}

	// Try to find a newline or space near the target size
	targetPos := len(text) * targetSize / lengthFn(text)
	for i := targetPos; i > targetPos-100 && i > 0; i-- {
		if i < len(text) && (text[i] == '\n' || text[i] == ' ') {
			return i + 1
		}
	}

	// Fallback: just split at target position
	return targetPos
}

// getOverlapText extracts overlap text from the end of a chunk.
func (s *MarkdownTextSplitter) getOverlapText(chunk string, overlapSize int, lengthFn func(string) int) string {
	if lengthFn(chunk) <= overlapSize {
		return chunk
	}

	// Try to find a good break point
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
func (s *MarkdownTextSplitter) SplitDocuments(ctx context.Context, documents []schema.Document) ([]schema.Document, error) {
	ctx, span := s.tracer.Start(ctx, "textsplitters.markdown.SplitDocuments",
		trace.WithAttributes(
			attribute.String("splitter.type", "markdown"),
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
			chunkDoc.Metadata["chunk_index"] = fmt.Sprintf("%d", i)
			chunkDoc.Metadata["chunk_total"] = fmt.Sprintf("%d", len(chunks))

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
func (s *MarkdownTextSplitter) CreateDocuments(ctx context.Context, texts []string, metadatas []map[string]any) ([]schema.Document, error) {
	ctx, span := s.tracer.Start(ctx, "textsplitters.markdown.CreateDocuments",
		trace.WithAttributes(
			attribute.String("splitter.type", "markdown"),
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

// Ensure MarkdownTextSplitter implements iface.TextSplitter
var _ iface.TextSplitter = (*MarkdownTextSplitter)(nil)
