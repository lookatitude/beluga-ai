package cognitive

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// LLMScorer classifies input complexity by asking a ChatModel to evaluate the
// query. This is more accurate than HeuristicScorer but incurs an LLM call
// cost for every scoring operation.
type LLMScorer struct {
	model llm.ChatModel
}

// Compile-time interface check.
var _ ComplexityScorer = (*LLMScorer)(nil)

// NewLLMScorer creates a new LLMScorer that uses the given ChatModel for
// complexity classification.
func NewLLMScorer(model llm.ChatModel) (*LLMScorer, error) {
	if model == nil {
		return nil, core.NewError("cognitive.llm_scorer", core.ErrInvalidInput, "model is required", nil)
	}
	return &LLMScorer{model: model}, nil
}

// classificationPrompt is the system prompt for the classification LLM call.
const classificationPrompt = `You are a query complexity classifier. Classify the user's query into exactly one category:

- simple: Factual lookup, greetings, simple Q&A, translations, definitions.
- moderate: Requires some reasoning, summarization, or light analysis.
- complex: Requires multi-step reasoning, comparison, in-depth analysis, math proofs, or creative synthesis.

Respond with ONLY one word: simple, moderate, or complex.`

// Score sends the input to the LLM for complexity classification.
func (s *LLMScorer) Score(ctx context.Context, input string) (ComplexityScore, error) {
	msgs := []schema.Message{
		schema.NewSystemMessage(classificationPrompt),
		schema.NewHumanMessage(input),
	}

	resp, err := s.model.Generate(ctx, msgs)
	if err != nil {
		return ComplexityScore{}, fmt.Errorf("cognitive: llm scorer: %w", err)
	}

	text := strings.TrimSpace(strings.ToLower(resp.Text()))

	var level ComplexityLevel
	var confidence float64

	switch {
	case strings.Contains(text, "complex"):
		level = Complex
		confidence = 0.85
	case strings.Contains(text, "moderate"):
		level = Moderate
		confidence = 0.80
	case strings.Contains(text, "simple"):
		level = Simple
		confidence = 0.90
	default:
		// Default to moderate if classification is unclear
		level = Moderate
		confidence = 0.50
	}

	return ComplexityScore{
		Level:      level,
		Confidence: confidence,
		Reason:     fmt.Sprintf("llm classification: %s", text),
	}, nil
}
