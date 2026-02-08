package splitter

import (
	"context"
	"strings"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	Register("recursive", func(cfg config.ProviderConfig) (TextSplitter, error) {
		var opts []RecursiveOption
		if cs, ok := config.GetOption[float64](cfg, "chunk_size"); ok {
			opts = append(opts, WithChunkSize(int(cs)))
		}
		if co, ok := config.GetOption[float64](cfg, "chunk_overlap"); ok {
			opts = append(opts, WithChunkOverlap(int(co)))
		}
		return NewRecursiveSplitter(opts...), nil
	})
}

// DefaultSeparators are the default separators used by RecursiveSplitter,
// ordered from most to least significant.
var DefaultSeparators = []string{"\n\n", "\n", " ", ""}

// RecursiveOption configures a RecursiveSplitter.
type RecursiveOption func(*RecursiveSplitter)

// WithChunkSize sets the maximum chunk size in characters.
func WithChunkSize(size int) RecursiveOption {
	return func(s *RecursiveSplitter) {
		if size > 0 {
			s.chunkSize = size
		}
	}
}

// WithChunkOverlap sets the number of overlapping characters between chunks.
func WithChunkOverlap(overlap int) RecursiveOption {
	return func(s *RecursiveSplitter) {
		if overlap >= 0 {
			s.chunkOverlap = overlap
		}
	}
}

// WithSeparators sets the separators to use for splitting, ordered from most
// to least significant.
func WithSeparators(seps []string) RecursiveOption {
	return func(s *RecursiveSplitter) {
		if len(seps) > 0 {
			s.separators = seps
		}
	}
}

// RecursiveSplitter splits text by recursively trying separators from most
// significant (paragraph break) to least significant (empty string / character
// level). It aims to keep chunks under chunkSize while maintaining semantic
// coherence.
type RecursiveSplitter struct {
	chunkSize    int
	chunkOverlap int
	separators   []string
}

// NewRecursiveSplitter creates a new RecursiveSplitter with the given options.
// Default: chunkSize=1000, chunkOverlap=200, separators=["\n\n", "\n", " ", ""].
func NewRecursiveSplitter(opts ...RecursiveOption) *RecursiveSplitter {
	s := &RecursiveSplitter{
		chunkSize:    1000,
		chunkOverlap: 200,
		separators:   DefaultSeparators,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Split divides text into chunks using recursive separator splitting.
func (s *RecursiveSplitter) Split(_ context.Context, text string) ([]string, error) {
	return s.splitText(text, s.separators), nil
}

// SplitDocuments splits each document's content and returns new documents per chunk.
func (s *RecursiveSplitter) SplitDocuments(ctx context.Context, docs []schema.Document) ([]schema.Document, error) {
	return splitDocumentsHelper(ctx, s, docs)
}

// splitText recursively splits text using the given separators.
func (s *RecursiveSplitter) splitText(text string, separators []string) []string {
	if len(text) <= s.chunkSize {
		if text = strings.TrimSpace(text); text != "" {
			return []string{text}
		}
		return nil
	}

	// Find the appropriate separator.
	separator := ""
	remainingSeps := separators
	for i, sep := range separators {
		if sep == "" || strings.Contains(text, sep) {
			separator = sep
			remainingSeps = separators[i+1:]
			break
		}
	}

	// Split by the chosen separator.
	var splits []string
	if separator == "" {
		// Character-level split.
		for i := 0; i < len(text); i += s.chunkSize {
			end := i + s.chunkSize
			if end > len(text) {
				end = len(text)
			}
			splits = append(splits, text[i:end])
		}
	} else {
		splits = strings.Split(text, separator)
	}

	// Merge small splits and recurse on large ones.
	var chunks []string
	var current strings.Builder

	for _, split := range splits {
		split = strings.TrimSpace(split)
		if split == "" {
			continue
		}

		candidate := split
		if current.Len() > 0 {
			candidate = current.String() + separator + split
		}

		if len(candidate) <= s.chunkSize {
			current.Reset()
			current.WriteString(candidate)
			continue
		}

		// Flush current buffer if non-empty.
		if current.Len() > 0 {
			if trimmed := strings.TrimSpace(current.String()); trimmed != "" {
				chunks = append(chunks, trimmed)
			}
			// Start overlap from current.
			overlap := s.getOverlap(current.String())
			current.Reset()
			if overlap != "" {
				current.WriteString(overlap)
			}
		}

		// If the split itself is too large, recurse with next separator.
		if len(split) > s.chunkSize && len(remainingSeps) > 0 {
			subChunks := s.splitText(split, remainingSeps)
			chunks = append(chunks, subChunks...)
			current.Reset()
		} else {
			if current.Len() > 0 {
				current.WriteString(separator)
			}
			current.WriteString(split)
		}
	}

	// Flush remaining.
	if current.Len() > 0 {
		if trimmed := strings.TrimSpace(current.String()); trimmed != "" {
			chunks = append(chunks, trimmed)
		}
	}

	return chunks
}

// getOverlap returns the trailing portion of text to use as overlap for the
// next chunk.
func (s *RecursiveSplitter) getOverlap(text string) string {
	if s.chunkOverlap <= 0 || len(text) <= s.chunkOverlap {
		return ""
	}
	return text[len(text)-s.chunkOverlap:]
}
