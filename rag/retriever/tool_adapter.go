package retriever

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
	"github.com/lookatitude/beluga-ai/tool"
)

// retrieverTool adapts a Retriever as a tool.Tool so that agents can invoke
// document retrieval as a tool call.
type retrieverTool struct {
	retriever   Retriever
	name        string
	description string
	topK        int
}

// Compile-time interface check.
var _ tool.Tool = (*retrieverTool)(nil)

// AsTool wraps a Retriever as a tool.Tool with the given name and description.
// The tool accepts a JSON input with a "query" field and returns matching
// document contents as text. An optional topK limits results (default 10).
func AsTool(r Retriever, name, description string, opts ...ToolAdapterOption) tool.Tool {
	t := &retrieverTool{
		retriever:   r,
		name:        name,
		description: description,
		topK:        10,
	}
	for _, o := range opts {
		o(t)
	}
	return t
}

// ToolAdapterOption configures the retriever tool adapter.
type ToolAdapterOption func(*retrieverTool)

// WithToolTopK sets the default top-k for the retriever tool.
func WithToolTopK(k int) ToolAdapterOption {
	return func(t *retrieverTool) {
		if k > 0 {
			t.topK = k
		}
	}
}

// Name returns the tool name.
func (t *retrieverTool) Name() string { return t.name }

// Description returns the tool description.
func (t *retrieverTool) Description() string { return t.description }

// InputSchema returns the JSON Schema for the tool input.
func (t *retrieverTool) InputSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "The search query to retrieve documents for",
			},
		},
		"required": []string{"query"},
	}
}

// Execute runs the retriever with the query from the input map and returns
// document contents as a text result.
func (t *retrieverTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
	queryRaw, ok := input["query"]
	if !ok {
		return nil, core.Errorf(core.ErrInvalidInput, "retriever tool %s: missing required field \"query\"", t.name)
	}

	query, ok := queryRaw.(string)
	if !ok {
		return nil, core.Errorf(core.ErrInvalidInput, "retriever tool %s: \"query\" must be a string", t.name)
	}

	if query == "" {
		return nil, core.Errorf(core.ErrInvalidInput, "retriever tool %s: \"query\" must not be empty", t.name)
	}

	docs, err := t.retriever.Retrieve(ctx, query, WithTopK(t.topK))
	if err != nil {
		return tool.ErrorResult(core.Errorf(core.ErrProviderDown, "retriever tool %s: %w", t.name, err)), nil
	}

	if len(docs) == 0 {
		return tool.TextResult("No documents found."), nil
	}

	return tool.TextResult(formatDocs(docs)), nil
}

// formatDocs renders documents as a numbered list for LLM consumption.
func formatDocs(docs []schema.Document) string {
	var b strings.Builder
	for i, doc := range docs {
		if i > 0 {
			b.WriteString("\n\n")
		}
		fmt.Fprintf(&b, "[%d] %s", i+1, doc.Content)
		if doc.Score > 0 {
			fmt.Fprintf(&b, " (score: %.2f)", doc.Score)
		}
	}
	return b.String()
}
