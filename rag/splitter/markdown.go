package splitter

import (
	"context"
	"strings"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	Register("markdown", func(cfg config.ProviderConfig) (TextSplitter, error) {
		var opts []MarkdownOption
		if cs, ok := config.GetOption[float64](cfg, "chunk_size"); ok {
			opts = append(opts, WithMarkdownChunkSize(int(cs)))
		}
		if co, ok := config.GetOption[float64](cfg, "chunk_overlap"); ok {
			opts = append(opts, WithMarkdownChunkOverlap(int(co)))
		}
		if ph, ok := config.GetOption[bool](cfg, "preserve_headers"); ok {
			opts = append(opts, WithPreserveHeaders(ph))
		}
		return NewMarkdownSplitter(opts...), nil
	})
}

// MarkdownOption configures a MarkdownSplitter.
type MarkdownOption func(*MarkdownSplitter)

// WithMarkdownChunkSize sets the maximum chunk size in characters.
func WithMarkdownChunkSize(size int) MarkdownOption {
	return func(s *MarkdownSplitter) {
		if size > 0 {
			s.chunkSize = size
		}
	}
}

// WithMarkdownChunkOverlap sets the number of overlapping characters.
func WithMarkdownChunkOverlap(overlap int) MarkdownOption {
	return func(s *MarkdownSplitter) {
		if overlap >= 0 {
			s.chunkOverlap = overlap
		}
	}
}

// WithPreserveHeaders sets whether to prepend parent headers to each chunk.
func WithPreserveHeaders(preserve bool) MarkdownOption {
	return func(s *MarkdownSplitter) {
		s.preserveHeaders = preserve
	}
}

// MarkdownSplitter splits Markdown text on heading boundaries, preserving
// header hierarchy. Each section under a heading becomes a chunk. When
// preserveHeaders is enabled, parent heading context is prepended to each
// chunk.
type MarkdownSplitter struct {
	chunkSize       int
	chunkOverlap    int
	preserveHeaders bool
}

// NewMarkdownSplitter creates a new MarkdownSplitter with the given options.
// Default: chunkSize=1000, chunkOverlap=0, preserveHeaders=true.
func NewMarkdownSplitter(opts ...MarkdownOption) *MarkdownSplitter {
	s := &MarkdownSplitter{
		chunkSize:       1000,
		chunkOverlap:    0,
		preserveHeaders: true,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// markdownSection represents a heading and its content.
type markdownSection struct {
	level   int
	heading string
	content string
}

// Split divides Markdown text into chunks based on heading structure.
func (s *MarkdownSplitter) Split(_ context.Context, text string) ([]string, error) {
	sections := s.parseSections(text)
	return s.buildChunks(sections), nil
}

// SplitDocuments splits each document's content and returns new documents per chunk.
func (s *MarkdownSplitter) SplitDocuments(ctx context.Context, docs []schema.Document) ([]schema.Document, error) {
	return splitDocumentsHelper(ctx, s, docs)
}

// parseSections parses Markdown text into sections based on headings.
func (s *MarkdownSplitter) parseSections(text string) []markdownSection {
	lines := strings.Split(text, "\n")
	var sections []markdownSection
	var currentContent strings.Builder
	currentLevel := 0
	currentHeading := ""

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		level := headingLevel(trimmed)

		if level > 0 {
			// Flush previous section.
			if currentContent.Len() > 0 || currentHeading != "" {
				sections = append(sections, markdownSection{
					level:   currentLevel,
					heading: currentHeading,
					content: strings.TrimSpace(currentContent.String()),
				})
			}
			currentLevel = level
			currentHeading = trimmed
			currentContent.Reset()
		} else {
			if currentContent.Len() > 0 {
				currentContent.WriteString("\n")
			}
			currentContent.WriteString(line)
		}
	}

	// Flush final section.
	if currentContent.Len() > 0 || currentHeading != "" {
		sections = append(sections, markdownSection{
			level:   currentLevel,
			heading: currentHeading,
			content: strings.TrimSpace(currentContent.String()),
		})
	}

	return sections
}

// buildChunks creates chunks from sections, optionally prepending header context.
func (s *MarkdownSplitter) buildChunks(sections []markdownSection) []string {
	var chunks []string

	// Track active header hierarchy.
	headers := make(map[int]string) // level → heading text

	for _, sec := range sections {
		// Update header hierarchy.
		if sec.level > 0 {
			headers[sec.level] = sec.heading
			// Clear lower-level headers.
			for l := range headers {
				if l > sec.level {
					delete(headers, l)
				}
			}
		}

		// Build chunk content.
		var chunk strings.Builder

		if s.preserveHeaders && sec.level > 0 {
			// Prepend parent headers.
			for l := 1; l < sec.level; l++ {
				if h, ok := headers[l]; ok {
					chunk.WriteString(h)
					chunk.WriteString("\n")
				}
			}
			chunk.WriteString(sec.heading)
			chunk.WriteString("\n")
		}

		if sec.content != "" {
			if chunk.Len() > 0 {
				chunk.WriteString("\n")
			}
			chunk.WriteString(sec.content)
		}

		text := strings.TrimSpace(chunk.String())
		if text == "" {
			continue
		}

		// If chunk exceeds size, use recursive character splitting as fallback.
		if len(text) > s.chunkSize {
			rs := NewRecursiveSplitter(
				WithChunkSize(s.chunkSize),
				WithChunkOverlap(s.chunkOverlap),
			)
			subChunks := rs.splitText(text, DefaultSeparators)
			chunks = append(chunks, subChunks...)
		} else {
			chunks = append(chunks, text)
		}
	}

	return chunks
}

// headingLevel returns the ATX heading level (1-6) or 0 if not a heading.
func headingLevel(line string) int {
	if !strings.HasPrefix(line, "#") {
		return 0
	}
	level := 0
	for _, ch := range line {
		if ch == '#' {
			level++
		} else {
			break
		}
	}
	if level > 6 {
		return 0
	}
	// Must be followed by a space or be just "#"s.
	if level < len(line) && line[level] != ' ' {
		return 0
	}
	return level
}
