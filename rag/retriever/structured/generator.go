package structured

import (
	"context"
	"fmt"
	"strings"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/llm"
	"github.com/lookatitude/beluga-ai/schema"
)

// QueryGenerator translates a natural-language question into a database query
// using the provided schema information.
type QueryGenerator interface {
	// Generate produces a database query string from a question and schema.
	// The returned string should be valid Cypher or SQL depending on the
	// schema dialect.
	Generate(ctx context.Context, question string, info SchemaInfo) (string, error)
}

// LLMCypherGenerator uses an [llm.ChatModel] to generate Cypher queries.
type LLMCypherGenerator struct {
	model llm.ChatModel
}

// Compile-time interface check.
var _ QueryGenerator = (*LLMCypherGenerator)(nil)

// NewLLMCypherGenerator creates a Cypher query generator backed by the given
// chat model.
func NewLLMCypherGenerator(model llm.ChatModel) *LLMCypherGenerator {
	return &LLMCypherGenerator{model: model}
}

// Generate translates a natural-language question into a Cypher query.
func (g *LLMCypherGenerator) Generate(ctx context.Context, question string, info SchemaInfo) (string, error) {
	if question == "" {
		return "", core.NewError("structured.generate", core.ErrInvalidInput, "question must not be empty", nil)
	}

	prompt := buildCypherPrompt(question, info)
	resp, err := g.model.Generate(ctx, []schema.Message{
		schema.NewSystemMessage(cypherSystemPrompt),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return "", fmt.Errorf("structured.generate: llm call: %w", err)
	}

	return extractQuery(resp.Text()), nil
}

// LLMSQLGenerator uses an [llm.ChatModel] to generate SQL queries.
type LLMSQLGenerator struct {
	model llm.ChatModel
}

// Compile-time interface check.
var _ QueryGenerator = (*LLMSQLGenerator)(nil)

// NewLLMSQLGenerator creates a SQL query generator backed by the given
// chat model.
func NewLLMSQLGenerator(model llm.ChatModel) *LLMSQLGenerator {
	return &LLMSQLGenerator{model: model}
}

// Generate translates a natural-language question into a SQL query.
func (g *LLMSQLGenerator) Generate(ctx context.Context, question string, info SchemaInfo) (string, error) {
	if question == "" {
		return "", core.NewError("structured.generate", core.ErrInvalidInput, "question must not be empty", nil)
	}

	prompt := buildSQLPrompt(question, info)
	resp, err := g.model.Generate(ctx, []schema.Message{
		schema.NewSystemMessage(sqlSystemPrompt),
		schema.NewHumanMessage(prompt),
	})
	if err != nil {
		return "", fmt.Errorf("structured.generate: llm call: %w", err)
	}

	return extractQuery(resp.Text()), nil
}

const cypherSystemPrompt = `You are a Cypher query expert. Given the graph schema and a question, generate a valid Cypher query.
Output ONLY the Cypher query. Do not include explanations, markdown formatting, or code fences.
Use only node labels, relationship types, and properties present in the schema.`

const sqlSystemPrompt = `You are a SQL query expert. Given the database schema and a question, generate a valid SQL SELECT query.
Output ONLY the SQL query. Do not include explanations, markdown formatting, or code fences.
Use only tables and columns present in the schema. Generate read-only queries only.`

// buildCypherPrompt constructs the user prompt for Cypher generation.
func buildCypherPrompt(question string, info SchemaInfo) string {
	var b strings.Builder
	b.WriteString("Graph Schema:\n")
	writeSchemaDescription(&b, info)
	b.WriteString("\nQuestion: ")
	b.WriteString(question)
	b.WriteString("\n\nCypher Query:")
	return b.String()
}

// buildSQLPrompt constructs the user prompt for SQL generation.
func buildSQLPrompt(question string, info SchemaInfo) string {
	var b strings.Builder
	b.WriteString("Database Schema:\n")
	writeSchemaDescription(&b, info)
	b.WriteString("\nQuestion: ")
	b.WriteString(question)
	b.WriteString("\n\nSQL Query:")
	return b.String()
}

// writeSchemaDescription writes a textual representation of the schema.
func writeSchemaDescription(b *strings.Builder, info SchemaInfo) {
	for _, t := range info.Tables {
		b.WriteString("  ")
		b.WriteString(t.Name)
		if t.Description != "" {
			b.WriteString(" (")
			b.WriteString(t.Description)
			b.WriteString(")")
		}
		b.WriteString(":\n")
		for _, c := range t.Columns {
			b.WriteString("    - ")
			b.WriteString(c.Name)
			if c.Type != "" {
				b.WriteString(" : ")
				b.WriteString(c.Type)
			}
			if c.Description != "" {
				b.WriteString(" -- ")
				b.WriteString(c.Description)
			}
			b.WriteString("\n")
		}
	}
	if len(info.Relationships) > 0 {
		b.WriteString("  Relationships:\n")
		for _, r := range info.Relationships {
			b.WriteString("    ")
			b.WriteString(r.From)
			b.WriteString(" -[")
			b.WriteString(r.Type)
			b.WriteString("]-> ")
			b.WriteString(r.To)
			b.WriteString("\n")
		}
	}
	if info.ExtraContext != "" {
		b.WriteString("\n  Additional context: ")
		b.WriteString(info.ExtraContext)
		b.WriteString("\n")
	}
}

// extractQuery strips markdown code fences and trims whitespace from LLM
// output to produce a clean query string.
func extractQuery(raw string) string {
	s := strings.TrimSpace(raw)

	// Remove code fences (```cypher ... ``` or ```sql ... ```).
	if strings.HasPrefix(s, "```") {
		lines := strings.Split(s, "\n")
		// Drop first and last lines (code fence markers).
		if len(lines) >= 2 {
			end := len(lines) - 1
			if strings.TrimSpace(lines[end]) == "```" {
				lines = lines[1:end]
			} else {
				lines = lines[1:]
			}
			s = strings.TrimSpace(strings.Join(lines, "\n"))
		}
	}

	return s
}
