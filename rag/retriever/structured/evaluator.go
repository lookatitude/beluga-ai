package structured

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// ResultEvaluator judges how relevant query results are to the original
// question. The score is in the range [0, 1] where 1 means perfectly
// relevant.
type ResultEvaluator interface {
	// Evaluate returns a relevance score for the given results with respect
	// to the question.
	Evaluate(ctx context.Context, question string, results []map[string]any) (float64, error)
}

// LLMEvaluator uses an [llm.ChatModel] to judge result relevance.
type LLMEvaluator struct {
	model llm.ChatModel
}

// Compile-time interface check.
var _ ResultEvaluator = (*LLMEvaluator)(nil)

// NewLLMEvaluator creates a result evaluator backed by the given chat model.
func NewLLMEvaluator(model llm.ChatModel) *LLMEvaluator {
	return &LLMEvaluator{model: model}
}

const evaluatorSystemPrompt = `You are a relevance evaluator. Given a question and database query results, rate how well the results answer the question.
Output ONLY a single floating-point number between 0.0 and 1.0.
- 1.0 means the results perfectly answer the question.
- 0.0 means the results are completely irrelevant or empty.
Do not include any other text.`

// Evaluate asks the LLM to rate the relevance of the results.
func (e *LLMEvaluator) Evaluate(ctx context.Context, question string, results []map[string]any) (float64, error) {
	if len(results) == 0 {
		return 0.0, nil
	}

	// Limit the results preview to avoid overly large prompts.
	preview := resultsPreview(results, 20)

	prompt := fmt.Sprintf("Question: %s\n\nQuery Results:\n%s\n\nRelevance Score:", question, preview)
	resp, err := e.model.Generate(ctx, []schema.Message{
		schema.NewSystemMessage(evaluatorSystemPrompt),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return 0, fmt.Errorf("structured.evaluate: llm call: %w", err)
	}

	score, err := parseScore(resp.Text())
	if err != nil {
		return 0, fmt.Errorf("structured.evaluate: parse score: %w", err)
	}

	return score, nil
}

// resultsPreview returns a JSON string of up to maxRows result rows.
func resultsPreview(results []map[string]any, maxRows int) string {
	subset := results
	if len(subset) > maxRows {
		subset = subset[:maxRows]
	}
	data, err := json.Marshal(subset)
	if err != nil {
		return fmt.Sprintf("[%d rows, marshal error]", len(results))
	}
	return string(data)
}

// parseScore extracts a float64 in [0, 1] from the LLM response text.
func parseScore(text string) (float64, error) {
	s := strings.TrimSpace(text)
	score, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("expected float, got %q: %w", s, err)
	}
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score, nil
}
