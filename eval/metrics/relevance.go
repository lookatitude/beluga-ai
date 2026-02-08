package metrics

import (
	"context"
	"fmt"

	"github.com/lookatitude/beluga-ai/eval"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

const relevancePrompt = `You are an evaluation judge. Given a question and an answer, evaluate whether the answer is relevant to and adequately addresses the question.

Question: %s

Answer: %s

Rate the relevance of the answer on a scale from 0.0 to 1.0:
- 1.0: The answer directly and completely addresses the question
- 0.5: The answer partially addresses the question
- 0.0: The answer is irrelevant to the question

Respond with ONLY a single decimal number between 0.0 and 1.0.`

// Relevance evaluates whether an AI-generated answer adequately addresses the
// input question. It uses an LLM as a judge to assess relevance.
type Relevance struct {
	llm llm.ChatModel
}

// NewRelevance creates a new Relevance metric using the given LLM as judge.
func NewRelevance(model llm.ChatModel) *Relevance {
	return &Relevance{llm: model}
}

// Name returns "relevance".
func (r *Relevance) Name() string { return "relevance" }

// Score evaluates whether the output answers the input question.
// Returns a score in [0.0, 1.0] where 1.0 means fully relevant.
func (r *Relevance) Score(ctx context.Context, sample eval.EvalSample) (float64, error) {
	prompt := fmt.Sprintf(relevancePrompt, sample.Input, sample.Output)

	resp, err := r.llm.Generate(ctx, []schema.Message{
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return 0, fmt.Errorf("relevance: llm generate: %w", err)
	}

	return parseScore(resp.Text())
}
