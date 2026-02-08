package metrics

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/eval"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

const hallucinationPrompt = `You are an evaluation judge. Given a question, context documents, and an answer, detect whether the answer contains fabricated or hallucinated information that is not supported by the context or commonly known facts.

Question: %s

Context Documents:
%s

Answer: %s

Rate the answer on a scale from 0.0 to 1.0:
- 1.0: No hallucination detected; all claims are supported or commonly known
- 0.5: Some claims may be unsupported but are not clearly fabricated
- 0.0: The answer contains clearly fabricated or false information

Respond with ONLY a single decimal number between 0.0 and 1.0.`

// Hallucination detects fabricated facts in AI-generated answers by comparing
// them against the provided context documents. It uses an LLM as a judge.
// A score of 1.0 means no hallucination was detected.
type Hallucination struct {
	llm llm.ChatModel
}

// NewHallucination creates a new Hallucination metric using the given LLM as judge.
func NewHallucination(model llm.ChatModel) *Hallucination {
	return &Hallucination{llm: model}
}

// Name returns "hallucination".
func (h *Hallucination) Name() string { return "hallucination" }

// Score evaluates whether the output contains fabricated information.
// Returns a score in [0.0, 1.0] where 1.0 means no hallucination detected.
func (h *Hallucination) Score(ctx context.Context, sample eval.EvalSample) (float64, error) {
	docs := formatDocs(sample.RetrievedDocs)
	prompt := fmt.Sprintf(hallucinationPrompt, sample.Input, docs, sample.Output)

	resp, err := h.llm.Generate(ctx, []schema.Message{
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return 0, fmt.Errorf("hallucination: llm generate: %w", err)
	}

	return parseScore(resp.Text())
}
