package associative

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// Enrichment holds the LLM-generated metadata for a note.
type Enrichment struct {
	// Keywords are key terms extracted from the content.
	Keywords []string `json:"keywords"`
	// Tags are categorical labels for the content.
	Tags []string `json:"tags"`
	// Description is a concise summary of the content.
	Description string `json:"description"`
}

// NoteEnricher uses an LLM to extract keywords, tags, and a description
// from raw note content. If no LLM is provided, enrichment is skipped and
// an empty Enrichment is returned.
type NoteEnricher struct {
	model   llm.ChatModel
	maxTags int
}

// NewNoteEnricher creates a NoteEnricher. If model is nil, Enrich returns
// empty enrichments without error.
func NewNoteEnricher(model llm.ChatModel, maxTags int) *NoteEnricher {
	if maxTags <= 0 {
		maxTags = 8
	}
	return &NoteEnricher{
		model:   model,
		maxTags: maxTags,
	}
}

// Enrich generates keywords, tags, and a description for the given content.
// If no LLM model is configured, returns an empty Enrichment.
func (e *NoteEnricher) Enrich(ctx context.Context, content string) (*Enrichment, error) {
	if e.model == nil {
		return &Enrichment{}, nil
	}
	if content == "" {
		return &Enrichment{}, nil
	}

	prompt := fmt.Sprintf(`Analyze the following content and extract structured metadata.

Return a JSON object with exactly these fields:
- "keywords": an array of 3-7 key terms or phrases that capture the main concepts
- "tags": an array of 1-%d categorical labels (single words, lowercase)
- "description": a concise 1-2 sentence summary

Content:
---
%s
---

Respond with ONLY the JSON object, no markdown fences or other text.`, e.maxTags, content)

	msgs := []schema.Message{
		schema.NewSystemMessage("You are a knowledge extraction assistant. You output valid JSON only."),
		schema.NewHumanMessage(prompt),
	}

	resp, err := e.model.Generate(ctx, msgs)
	if err != nil {
		return nil, fmt.Errorf("associative.enricher: LLM generate: %w", err)
	}

	text := resp.Text()
	text = strings.TrimSpace(text)
	// Strip markdown fences if present.
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var enrichment Enrichment
	if err := json.Unmarshal([]byte(text), &enrichment); err != nil {
		return nil, fmt.Errorf("associative.enricher: parse LLM response: %w", err)
	}

	// Enforce tag limit.
	if len(enrichment.Tags) > e.maxTags {
		enrichment.Tags = enrichment.Tags[:e.maxTags]
	}

	return &enrichment, nil
}
