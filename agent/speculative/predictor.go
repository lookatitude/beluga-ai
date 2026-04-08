package speculative

import (
	"context"
	"strings"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// Predictor produces a fast prediction for a given input.
// Implementations may use a lighter/cheaper model, caching, or heuristics.
type Predictor interface {
	// Predict returns a prediction string and confidence score (0.0-1.0).
	Predict(ctx context.Context, input string) (prediction string, confidence float64, err error)
}

// LightModelPredictor uses a fast/cheap ChatModel to produce predictions.
// It sends the input as a simple human message and returns the model's response.
type LightModelPredictor struct {
	model llm.ChatModel
}

// compile-time check
var _ Predictor = (*LightModelPredictor)(nil)

// NewLightModelPredictor creates a Predictor backed by the given ChatModel.
// The model should be a fast, cheap model suitable for drafting predictions.
func NewLightModelPredictor(model llm.ChatModel) *LightModelPredictor {
	return &LightModelPredictor{model: model}
}

// Predict sends the input to the light model and returns its response.
// Confidence is derived from the response length heuristic: shorter, more
// decisive responses get higher confidence.
func (p *LightModelPredictor) Predict(ctx context.Context, input string) (string, float64, error) {
	msgs := []schema.Message{
		schema.NewHumanMessage(input),
	}

	resp, err := p.model.Generate(ctx, msgs)
	if err != nil {
		return "", 0, err
	}

	text := resp.Text()
	confidence := estimateConfidence(text)

	return text, confidence, nil
}

// estimateConfidence produces a heuristic confidence score based on the
// response characteristics. Shorter, more direct responses score higher.
func estimateConfidence(text string) float64 {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0.0
	}

	// Heuristic: shorter responses tend to be more confident/decisive.
	// Scale from 1.0 (very short) down to 0.3 (very long).
	words := len(strings.Fields(text))
	switch {
	case words <= 5:
		return 0.95
	case words <= 20:
		return 0.8
	case words <= 50:
		return 0.6
	case words <= 100:
		return 0.45
	default:
		return 0.3
	}
}
