package consolidation

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// Compressor reduces the size of memory records while preserving their
// essential information. Implementations must be safe for concurrent use.
type Compressor interface {
	// Compress takes a set of records and returns compressed versions.
	// The returned slice must have the same length as the input.
	Compress(ctx context.Context, records []Record) ([]Record, error)
}

// SummaryCompressor uses an LLM ChatModel to summarise record content into
// a shorter form.
type SummaryCompressor struct {
	model llm.ChatModel
}

// Compile-time interface check.
var _ Compressor = (*SummaryCompressor)(nil)

// NewSummaryCompressor creates a compressor backed by the given ChatModel.
func NewSummaryCompressor(model llm.ChatModel) *SummaryCompressor {
	return &SummaryCompressor{model: model}
}

// Compress summarises each record's content using the configured LLM. Each
// record is processed independently. The original record ID, metadata, and
// utility scores are preserved; only the content is replaced.
func (c *SummaryCompressor) Compress(ctx context.Context, records []Record) ([]Record, error) {
	compressed := make([]Record, len(records))
	for i, r := range records {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		summary, err := c.summarise(ctx, r.Content)
		if err != nil {
			return nil, fmt.Errorf("consolidation: compress record %q: %w", r.ID, err)
		}

		out := r
		out.Content = summary
		if out.Metadata == nil {
			out.Metadata = make(map[string]any)
		}
		out.Metadata["compressed"] = true
		compressed[i] = out
	}
	return compressed, nil
}

// summarise calls the LLM to produce a concise summary of the given text.
func (c *SummaryCompressor) summarise(ctx context.Context, text string) (string, error) {
	msgs := []schema.Message{
		schema.NewSystemMessage(
			"You are a memory consolidation assistant. Summarise the following memory " +
				"into a concise form that preserves the key facts, decisions, and context. " +
				"Output only the summary, no preamble.",
		),
		schema.NewHumanMessage(text),
	}

	resp, err := c.model.Generate(ctx, msgs)
	if err != nil {
		return "", err
	}

	summary := strings.TrimSpace(resp.Text())
	if summary == "" {
		return text, nil // fall back to original if LLM returned empty
	}
	return summary, nil
}
