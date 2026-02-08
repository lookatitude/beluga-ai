// Package metrics provides built-in evaluation metrics for the Beluga AI
// eval framework. It includes LLM-as-judge metrics (faithfulness, relevance,
// hallucination), keyword-based checks (toxicity), and metadata-based metrics
// (latency, cost).
package metrics

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/lookatitude/beluga-ai/eval"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

const faithfulnessPrompt = `You are an evaluation judge. Given a question, context documents, and an answer, evaluate whether the answer is faithful to (grounded in) the context documents.

Question: %s

Context Documents:
%s

Answer: %s

Rate the faithfulness of the answer on a scale from 0.0 to 1.0:
- 1.0: The answer is fully supported by the context documents
- 0.5: The answer is partially supported
- 0.0: The answer contradicts or is not supported by the context

Respond with ONLY a single decimal number between 0.0 and 1.0.`

// Faithfulness evaluates whether an AI-generated answer is grounded in the
// provided context documents. It uses an LLM as a judge to assess faithfulness.
type Faithfulness struct {
	llm llm.ChatModel
}

// NewFaithfulness creates a new Faithfulness metric using the given LLM as judge.
func NewFaithfulness(model llm.ChatModel) *Faithfulness {
	return &Faithfulness{llm: model}
}

// Name returns "faithfulness".
func (f *Faithfulness) Name() string { return "faithfulness" }

// Score evaluates whether the output is faithfully grounded in the retrieved
// documents. Returns a score in [0.0, 1.0] where 1.0 means fully faithful.
func (f *Faithfulness) Score(ctx context.Context, sample eval.EvalSample) (float64, error) {
	docs := formatDocs(sample.RetrievedDocs)
	prompt := fmt.Sprintf(faithfulnessPrompt, sample.Input, docs, sample.Output)

	resp, err := f.llm.Generate(ctx, []schema.Message{
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return 0, fmt.Errorf("faithfulness: llm generate: %w", err)
	}

	return parseScore(resp.Text())
}

// formatDocs concatenates document contents into a numbered list.
func formatDocs(docs []schema.Document) string {
	if len(docs) == 0 {
		return "(no documents provided)"
	}
	var b strings.Builder
	for i, doc := range docs {
		fmt.Fprintf(&b, "[%d] %s\n", i+1, doc.Content)
	}
	return b.String()
}

// parseScore extracts a float64 score from an LLM response string.
func parseScore(text string) (float64, error) {
	text = strings.TrimSpace(text)
	score, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse score from response %q: %w", text, err)
	}
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	return score, nil
}
