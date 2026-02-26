package splitter

import (
	"context"
	"strings"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	Register("token", func(cfg config.ProviderConfig) (TextSplitter, error) {
		var opts []TokenOption
		if cs, ok := config.GetOption[float64](cfg, "chunk_size"); ok {
			opts = append(opts, WithTokenChunkSize(int(cs)))
		}
		if co, ok := config.GetOption[float64](cfg, "chunk_overlap"); ok {
			opts = append(opts, WithTokenChunkOverlap(int(co)))
		}
		return NewTokenSplitter(opts...), nil
	})
}

// TokenOption configures a TokenSplitter.
type TokenOption func(*TokenSplitter)

// WithTokenChunkSize sets the maximum chunk size in tokens.
func WithTokenChunkSize(size int) TokenOption {
	return func(s *TokenSplitter) {
		if size > 0 {
			s.chunkSize = size
		}
	}
}

// WithTokenChunkOverlap sets the number of overlapping tokens between chunks.
func WithTokenChunkOverlap(overlap int) TokenOption {
	return func(s *TokenSplitter) {
		if overlap >= 0 {
			s.chunkOverlap = overlap
		}
	}
}

// WithTokenizer sets the tokenizer to use for counting tokens.
// If nil, a SimpleTokenizer is used as the default.
func WithTokenizer(t llm.Tokenizer) TokenOption {
	return func(s *TokenSplitter) {
		if t != nil {
			s.tokenizer = t
		}
	}
}

// TokenSplitter splits text by token count using an llm.Tokenizer. It first
// splits on word boundaries and then groups words into chunks that stay under
// the token limit.
type TokenSplitter struct {
	chunkSize    int
	chunkOverlap int
	tokenizer    llm.Tokenizer
}

// NewTokenSplitter creates a new TokenSplitter with the given options.
// Default: chunkSize=500 tokens, chunkOverlap=50, tokenizer=SimpleTokenizer.
func NewTokenSplitter(opts ...TokenOption) *TokenSplitter {
	s := &TokenSplitter{
		chunkSize:    500,
		chunkOverlap: 50,
		tokenizer:    &llm.SimpleTokenizer{},
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// Split divides text into chunks of at most chunkSize tokens.
func (s *TokenSplitter) Split(_ context.Context, text string) ([]string, error) {
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil, nil
	}

	var chunks []string
	start := 0

	for start < len(words) {
		end := s.findChunkEnd(words, start)

		chunk := strings.Join(words[start:end], " ")
		if trimmed := strings.TrimSpace(chunk); trimmed != "" {
			chunks = append(chunks, trimmed)
		}

		start = s.nextStart(words, start, end)
	}

	return chunks, nil
}

// findChunkEnd finds the end index for a chunk starting at start, respecting the token limit.
func (s *TokenSplitter) findChunkEnd(words []string, start int) int {
	end := start
	tokenCount := 0
	for end < len(words) {
		wordTokens := s.tokenizer.Count(words[end])
		if tokenCount+wordTokens > s.chunkSize && end > start {
			break
		}
		tokenCount += wordTokens
		end++
	}
	return end
}

// nextStart calculates the next start position, applying overlap if applicable.
func (s *TokenSplitter) nextStart(words []string, start, end int) int {
	if s.chunkOverlap > 0 && end < len(words) {
		overlapStart := end
		overlapTokens := 0
		for overlapStart > start {
			wt := s.tokenizer.Count(words[overlapStart-1])
			if overlapTokens+wt > s.chunkOverlap {
				break
			}
			overlapTokens += wt
			overlapStart--
		}
		return overlapStart
	}
	return end
}

// SplitDocuments splits each document's content and returns new documents per chunk.
func (s *TokenSplitter) SplitDocuments(ctx context.Context, docs []schema.Document) ([]schema.Document, error) {
	return splitDocumentsHelper(ctx, s, docs)
}
