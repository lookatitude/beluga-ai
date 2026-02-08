package llm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/internal/jsonutil"
	"github.com/lookatitude/beluga-ai/schema"
)

// StructuredOutput wraps a ChatModel to produce typed Go values. It generates
// a JSON Schema from T, instructs the model to respond in JSON, parses the
// response, and optionally retries on parse failures.
type StructuredOutput[T any] struct {
	model      ChatModel
	schema     map[string]any
	maxRetries int
}

// StructuredOption configures a StructuredOutput.
type StructuredOption func(*structuredConfig)

type structuredConfig struct {
	maxRetries int
}

// WithMaxRetries sets the maximum number of retry attempts when the model
// produces unparseable JSON. Defaults to 2.
func WithMaxRetries(n int) StructuredOption {
	return func(cfg *structuredConfig) {
		cfg.maxRetries = n
	}
}

// NewStructured creates a StructuredOutput[T] that uses model for generation.
// The JSON Schema is derived from the type parameter T using reflection.
func NewStructured[T any](model ChatModel, opts ...StructuredOption) *StructuredOutput[T] {
	cfg := &structuredConfig{maxRetries: 2}
	for _, opt := range opts {
		opt(cfg)
	}
	var zero T
	return &StructuredOutput[T]{
		model:      model,
		schema:     jsonutil.GenerateSchema(zero),
		maxRetries: cfg.maxRetries,
	}
}

// Generate sends the messages to the model with JSON Schema response format
// and parses the response into T. If parsing fails, it retries up to
// maxRetries times, including the parse error in the conversation to help
// the model self-correct.
func (s *StructuredOutput[T]) Generate(ctx context.Context, msgs []schema.Message) (T, error) {
	var zero T

	opts := []GenerateOption{
		WithResponseFormat(ResponseFormat{
			Type:   "json_schema",
			Schema: s.schema,
		}),
	}

	var lastErr error
	for attempt := 0; attempt <= s.maxRetries; attempt++ {
		resp, err := s.model.Generate(ctx, msgs, opts...)
		if err != nil {
			return zero, err
		}

		text := resp.Text()
		var result T
		if err := json.Unmarshal([]byte(text), &result); err != nil {
			lastErr = fmt.Errorf("structured output: parse attempt %d: %w", attempt+1, err)
			// Add the failed response and parse error to messages for self-correction.
			msgs = append(msgs,
				schema.NewAIMessage(text),
				schema.NewHumanMessage(fmt.Sprintf(
					"Your response was not valid JSON. Error: %s\nPlease respond with valid JSON matching the schema.",
					err.Error(),
				)),
			)
			continue
		}
		return result, nil
	}

	return zero, core.NewError("llm.structured", core.ErrInvalidInput,
		fmt.Sprintf("failed to parse structured output after %d attempts", s.maxRetries+1), lastErr)
}

// Schema returns the JSON Schema used for structured output.
func (s *StructuredOutput[T]) Schema() map[string]any {
	return s.schema
}
