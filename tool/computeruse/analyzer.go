package computeruse

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

const analyzePrompt = "Describe what you see in this screenshot. Focus on interactive elements (buttons, links, text fields, menus) and their positions. Be concise."

// analyzerOptions holds configuration for ScreenAnalyzer.
type analyzerOptions struct {
	model  llm.ChatModel
	prompt string
}

// AnalyzerOption configures a ScreenAnalyzer.
type AnalyzerOption func(*analyzerOptions)

// WithAnalyzerModel sets the multimodal LLM used for screenshot analysis.
func WithAnalyzerModel(m llm.ChatModel) AnalyzerOption {
	return func(o *analyzerOptions) {
		o.model = m
	}
}

// WithAnalyzerPrompt sets a custom analysis prompt.
func WithAnalyzerPrompt(prompt string) AnalyzerOption {
	return func(o *analyzerOptions) {
		o.prompt = prompt
	}
}

// ScreenAnalyzer uses a multimodal LLM to describe screenshots, enabling
// vision-based interaction for agents that need to understand screen content.
type ScreenAnalyzer struct {
	opts analyzerOptions
}

// NewScreenAnalyzer creates a new ScreenAnalyzer with the given options.
func NewScreenAnalyzer(opts ...AnalyzerOption) (*ScreenAnalyzer, error) {
	o := analyzerOptions{prompt: analyzePrompt}
	for _, opt := range opts {
		opt(&o)
	}
	if o.model == nil {
		return nil, core.NewError("computeruse.analyzer.new", core.ErrInvalidInput, "model is required", nil)
	}
	return &ScreenAnalyzer{opts: o}, nil
}

// Analyze sends a screenshot to the multimodal LLM and returns a textual
// description of the screen content.
func (a *ScreenAnalyzer) Analyze(ctx context.Context, screenshot []byte) (string, error) {
	if len(screenshot) == 0 {
		return "", core.NewError("computeruse.analyzer.analyze", core.ErrInvalidInput, "screenshot must not be empty", nil)
	}

	msg := &schema.HumanMessage{
		Parts: []schema.ContentPart{
			schema.TextPart{Text: a.opts.prompt},
			schema.ImagePart{Data: screenshot, MimeType: "image/png"},
		},
	}

	resp, err := a.opts.model.Generate(ctx, []schema.Message{msg})
	if err != nil {
		return "", fmt.Errorf("screen analyzer: llm generate: %w", err)
	}

	return resp.Text(), nil
}
