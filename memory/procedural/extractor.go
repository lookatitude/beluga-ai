package procedural

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// SkillExtractor extracts procedural skills from execution traces. It
// abstracts the input/output of a successful task completion into a
// reusable skill definition.
type SkillExtractor interface {
	// Extract analyzes an input/output pair and optional metadata to produce
	// a Skill. Returns nil if no meaningful skill can be extracted.
	Extract(ctx context.Context, input, output string, metadata map[string]any) (*schema.Skill, error)
}

// LLMExtractor uses a ChatModel to extract skills from execution traces.
// It sends a structured prompt to the LLM asking it to abstract the
// procedure into a reusable skill definition.
type LLMExtractor struct {
	model llm.ChatModel
}

// Compile-time check that LLMExtractor implements SkillExtractor.
var _ SkillExtractor = (*LLMExtractor)(nil)

// NewLLMExtractor creates a new LLMExtractor with the given ChatModel.
// Returns an error if the model is nil.
func NewLLMExtractor(model llm.ChatModel) (*LLMExtractor, error) {
	if model == nil {
		return nil, fmt.Errorf("procedural: ChatModel must not be nil for LLMExtractor")
	}
	return &LLMExtractor{model: model}, nil
}

// Extract uses the ChatModel to analyze an input/output pair and produce a
// Skill. The LLM is prompted to identify the procedure, abstract it into
// reusable steps, and return a structured JSON skill definition.
func (e *LLMExtractor) Extract(ctx context.Context, input, output string, metadata map[string]any) (*schema.Skill, error) {
	prompt := buildExtractionPrompt(input, output, metadata)

	msgs := []schema.Message{
		schema.NewSystemMessage(extractionSystemPrompt),
		schema.NewHumanMessage(prompt),
	}

	resp, err := e.model.Generate(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("procedural/extractor: generate: %w", err)
	}

	text := resp.Text()
	if text == "" {
		return nil, nil
	}

	var skill schema.Skill
	if err := json.Unmarshal([]byte(text), &skill); err != nil {
		return nil, fmt.Errorf("procedural/extractor: parse skill JSON: %w", err)
	}

	if skill.Name == "" {
		return nil, nil
	}
	return &skill, nil
}

const extractionSystemPrompt = `You are a skill extraction assistant. Given an input/output pair from a successful task completion, extract a reusable procedural skill.

Return a JSON object with exactly these fields:
- "name": short kebab-case name for the skill
- "description": one-sentence description of what the skill accomplishes
- "steps": array of strings, each a clear action step in the procedure
- "triggers": array of keywords/phrases that should activate this skill
- "tags": array of category labels
- "confidence": number between 0 and 1 indicating extraction confidence

If no meaningful skill can be extracted, return an empty JSON object {}.
Return ONLY valid JSON, no markdown fences or explanation.`

func buildExtractionPrompt(input, output string, metadata map[string]any) string {
	prompt := fmt.Sprintf("Input:\n%s\n\nOutput:\n%s", input, output)
	if len(metadata) > 0 {
		metaJSON, err := json.Marshal(metadata)
		if err == nil {
			prompt += fmt.Sprintf("\n\nMetadata:\n%s", string(metaJSON))
		}
	}
	return prompt
}
