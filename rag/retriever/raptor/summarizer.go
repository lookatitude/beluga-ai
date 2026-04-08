package raptor

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// Summarizer produces a single summary from multiple text chunks. It is called
// once per cluster at each tree level during the RAPTOR build process.
type Summarizer interface {
	// Summarize combines the provided texts into a single coherent summary.
	Summarize(ctx context.Context, texts []string) (string, error)
}

// LLMSummarizer implements Summarizer using an llm.ChatModel to generate
// abstractive summaries of clustered text chunks.
type LLMSummarizer struct {
	model  llm.ChatModel
	prompt string
}

// Compile-time interface check.
var _ Summarizer = (*LLMSummarizer)(nil)

// LLMSummarizerOption configures an LLMSummarizer.
type LLMSummarizerOption func(*LLMSummarizer)

// WithSummaryPrompt overrides the default summarization prompt template.
// The prompt should contain a single %s placeholder where the concatenated
// texts will be inserted.
func WithSummaryPrompt(prompt string) LLMSummarizerOption {
	return func(s *LLMSummarizer) {
		s.prompt = prompt
	}
}

const defaultSummaryPrompt = `Summarize the following text passages into a single, coherent summary that captures the key information and themes. Be concise but comprehensive.

Text passages:
%s

Summary:`

// NewLLMSummarizer creates a Summarizer backed by the given ChatModel.
func NewLLMSummarizer(model llm.ChatModel, opts ...LLMSummarizerOption) *LLMSummarizer {
	s := &LLMSummarizer{
		model:  model,
		prompt: defaultSummaryPrompt,
	}
	for _, o := range opts {
		o(s)
	}
	return s
}

// Summarize concatenates the texts and asks the LLM to produce a summary.
func (s *LLMSummarizer) Summarize(ctx context.Context, texts []string) (string, error) {
	if len(texts) == 0 {
		return "", fmt.Errorf("raptor: summarize: no texts provided")
	}

	combined := strings.Join(texts, "\n\n---\n\n")
	prompt := fmt.Sprintf(s.prompt, combined)

	msgs := []schema.Message{
		schema.NewHumanMessage(prompt),
	}

	resp, err := s.model.Generate(ctx, msgs)
	if err != nil {
		return "", fmt.Errorf("raptor: summarize: %w", err)
	}

	return resp.Text(), nil
}
