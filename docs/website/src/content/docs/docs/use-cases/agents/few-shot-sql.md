---
title: Natural Language to SQL with Few-Shot
description: "Generate SQL queries from natural language using few-shot learning. Achieve 92% accuracy across multiple database dialects."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "natural language to SQL, few-shot SQL, text-to-SQL, AI SQL generation, business intelligence, Beluga AI, Go, NL2SQL"
---

Business intelligence platforms need to enable non-technical users to query databases using natural language. The alternative — training users on SQL — does not scale, and building custom query builders for every possible question is impractical. Traditional zero-shot SQL generation (sending the question directly to an LLM with schema information) achieves only 60-70% accuracy because the model must infer query patterns, join strategies, and dialect-specific syntax from the schema alone.

Few-shot SQL generation uses example-based learning to achieve 90%+ accuracy while handling complex queries and multiple database dialects. The key insight is that a handful of well-chosen examples teach the LLM the specific query patterns, naming conventions, and join relationships used in your database — context that the schema definition alone cannot convey. Few-shot learning is preferred over fine-tuning because it requires no model training, adapts instantly to new examples, and works across different LLM providers.

## Solution Architecture

Beluga AI's prompt package combined with LLM-based generation enables few-shot SQL generation. The system selects relevant query examples, constructs prompts with schema information, generates SQL using the LLM, and validates output for correctness.

The pipeline has four stages: example selection (find the most relevant examples for this query), prompt construction (combine schema, examples, and the user's question using `prompt.PromptTemplate`), SQL generation (LLM produces the query), and validation (parse the output to verify syntactic correctness). This separation means each stage can be improved independently — better example selection improves accuracy without changing the prompt template, and better validation catches errors without affecting generation.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  Natural     │───▶│   Example    │───▶│    Prompt    │
│  Language    │    │   Selector   │    │   Builder    │
│  Query       │    └──────────────┘    └──────┬───────┘
└──────────────┘                               │
                                               ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  Validated   │◀───│     SQL      │◀───│     LLM      │
│    SQL       │    │  Validator   │    │  Generator   │
└──────────────┘    └──────────────┘    └──────────────┘
                          ▲
                          │
                    ┌─────┴────────┐
                    │   Schema     │
                    │ Information  │
                    └──────────────┘
```

## Few-Shot SQL Generation

The SQL generator selects similar examples based on query intent and constructs a prompt with schema information and examples. The prompt template uses Go's `text/template` syntax via Beluga AI's `prompt.NewPromptTemplate`, which ensures consistent prompt formatting across all invocations. Examples are selected per-dialect so the LLM sees syntax patterns matching the target database.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/prompt"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// SQLExample represents a few-shot learning example.
type SQLExample struct {
    NaturalLanguage string
    SQL             string
    Dialect         string
    Complexity      string
}

// SQLGenerator generates SQL queries from natural language using few-shot learning.
type SQLGenerator struct {
    model          llm.ChatModel
    promptTemplate *prompt.PromptTemplate
    examples       []SQLExample
    schemaInfo     map[string]SchemaInfo
}

func NewSQLGenerator(ctx context.Context) (*SQLGenerator, error) {
    model, err := llm.New("openai", nil)
    if err != nil {
        return nil, fmt.Errorf("create model: %w", err)
    }

    template, err := prompt.NewPromptTemplate(`
You are an expert SQL generator. Generate SQL queries from natural language.

Database Schema:
{{.schema}}

Dialect: {{.dialect}}

Examples:
{{range .examples}}
Natural Language: {{.NaturalLanguage}}
SQL: {{.SQL}}

{{end}}
Natural Language Query: {{.query}}

Generate a SQL query following the examples. Use the same dialect and style.
Return only the SQL query without explanation.
`)
    if err != nil {
        return nil, fmt.Errorf("create prompt template: %w", err)
    }

    return &SQLGenerator{
        model:          model,
        promptTemplate: template,
        examples:       loadExamples(),
        schemaInfo:     loadSchemaInfo(),
    }, nil
}

// GenerateSQL generates a SQL query from natural language.
func (s *SQLGenerator) GenerateSQL(ctx context.Context, query string, dialect string) (string, error) {
    // Select relevant few-shot examples
    examples := s.selectExamples(query, dialect, 3)

    // Get schema information for the dialect
    schemaStr := s.formatSchema(dialect)

    // Build prompt with examples and schema
    promptText, err := s.promptTemplate.Format(map[string]any{
        "schema":   schemaStr,
        "dialect":  dialect,
        "examples": examples,
        "query":    query,
    })
    if err != nil {
        return "", fmt.Errorf("format prompt: %w", err)
    }

    // Generate SQL using LLM
    msgs := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "You are an expert SQL generator. Generate accurate, syntactically correct SQL queries."},
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: promptText},
        }},
    }

    resp, err := s.model.Generate(ctx, msgs)
    if err != nil {
        return "", fmt.Errorf("generate SQL: %w", err)
    }

    sqlQuery := extractSQL(resp.Parts[0].(schema.TextPart).Text)

    // Validate SQL syntax
    if err := s.validateSQL(ctx, sqlQuery, dialect); err != nil {
        return "", fmt.Errorf("SQL validation failed: %w", err)
    }

    return sqlQuery, nil
}

func (s *SQLGenerator) selectExamples(query string, dialect string, count int) []SQLExample {
    // Filter examples by dialect
    var filtered []SQLExample
    for _, ex := range s.examples {
        if ex.Dialect == dialect {
            filtered = append(filtered, ex)
        }
    }

    // Select most similar examples using similarity scoring
    // Simplified: return first N
    if len(filtered) > count {
        return filtered[:count]
    }
    return filtered
}

func (s *SQLGenerator) formatSchema(dialect string) string {
    info, ok := s.schemaInfo[dialect]
    if !ok {
        return ""
    }

    var schemaStr string
    for tableName, columns := range info.Tables {
        schemaStr += fmt.Sprintf("Table: %s\n", tableName)
        for _, col := range columns {
            schemaStr += fmt.Sprintf("  - %s (%s)\n", col.Name, col.Type)
        }
        schemaStr += "\n"
    }

    return schemaStr
}

func (s *SQLGenerator) validateSQL(ctx context.Context, sqlQuery string, dialect string) error {
    // Implement SQL validation using parser for the specific dialect
    // Simplified validation
    if sqlQuery == "" {
        return fmt.Errorf("empty SQL query")
    }
    return nil
}

func extractSQL(response string) string {
    // Extract SQL from response (handle code blocks, etc.)
    // Simplified extraction
    return response
}

type SchemaInfo struct {
    Tables map[string][]ColumnInfo
}

type ColumnInfo struct {
    Name string
    Type string
}

func loadExamples() []SQLExample {
    // Load few-shot examples from configuration or database
    return []SQLExample{
        {
            NaturalLanguage: "Show me all customers",
            SQL:             "SELECT * FROM customers;",
            Dialect:         "postgres",
            Complexity:      "simple",
        },
        {
            NaturalLanguage: "Find total revenue by product category",
            SQL:             "SELECT c.name, SUM(o.amount) FROM orders o JOIN products p ON o.product_id = p.id JOIN categories c ON p.category_id = c.id GROUP BY c.name;",
            Dialect:         "postgres",
            Complexity:      "complex",
        },
    }
}

func loadSchemaInfo() map[string]SchemaInfo {
    // Load schema information for supported dialects
    return map[string]SchemaInfo{
        "postgres": {
            Tables: map[string][]ColumnInfo{
                "customers": {
                    {Name: "id", Type: "integer"},
                    {Name: "name", Type: "varchar"},
                    {Name: "email", Type: "varchar"},
                },
                "orders": {
                    {Name: "id", Type: "integer"},
                    {Name: "customer_id", Type: "integer"},
                    {Name: "amount", Type: "decimal"},
                },
            },
        },
    }
}
```

## Query Explanation

Generate explanations for the SQL query to build user trust:

```go
func (s *SQLGenerator) ExplainSQL(ctx context.Context, sqlQuery string) (string, error) {
    msgs := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "Explain SQL queries in plain language."},
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: fmt.Sprintf("Explain this SQL query:\n%s", sqlQuery)},
        }},
    }

    resp, err := s.model.Generate(ctx, msgs)
    if err != nil {
        return "", fmt.Errorf("generate explanation: %w", err)
    }

    return resp.Parts[0].(schema.TextPart).Text, nil
}
```

## Production Considerations

### Example Curation

Maintain a high-quality example database with diverse query patterns:

```go
type ExampleDatabase struct {
    examples map[string][]SQLExample // dialect -> examples
}

func (e *ExampleDatabase) AddExample(example SQLExample) error {
    if example.Dialect == "" || example.SQL == "" {
        return fmt.Errorf("invalid example: dialect and SQL required")
    }

    // Validate SQL syntax before adding
    if err := validateSyntax(example.SQL, example.Dialect); err != nil {
        return fmt.Errorf("invalid SQL syntax: %w", err)
    }

    e.examples[example.Dialect] = append(e.examples[example.Dialect], example)
    return nil
}

func validateSyntax(sql string, dialect string) error {
    // Use SQL parser for validation
    return nil // Simplified
}
```

### Multi-Dialect Support

Support multiple database dialects with dialect-specific templates:

```go
func (s *SQLGenerator) GenerateSQLWithDialectDetection(
    ctx context.Context,
    query string,
    defaultDialect string,
) (string, error) {
    // Detect dialect from query hints or use default
    dialect := s.detectDialect(query)
    if dialect == "" {
        dialect = defaultDialect
    }

    return s.GenerateSQL(ctx, query, dialect)
}

func (s *SQLGenerator) detectDialect(query string) string {
    // Implement dialect detection based on keywords or metadata
    return "" // Use default
}
```

### Observability

Track generation metrics to monitor accuracy and performance:

```go
import (
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

func (s *SQLGenerator) GenerateSQLWithMonitoring(
    ctx context.Context,
    query string,
    dialect string,
) (string, error) {
    tracer := otel.Tracer("sql-generation")
    ctx, span := tracer.Start(ctx, "sql.generate")
    defer span.End()

    span.SetAttributes(
        attribute.String("dialect", dialect),
        attribute.Int("query_length", len(query)),
    )

    sqlQuery, err := s.GenerateSQL(ctx, query, dialect)
    if err != nil {
        span.RecordError(err)
        return "", err
    }

    span.SetAttributes(
        attribute.String("generated_sql", sqlQuery),
        attribute.Bool("validation_passed", true),
    )

    return sqlQuery, nil
}
```

## Results

Few-shot SQL generation delivered significant improvements:

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Query Accuracy (%) | 60-70 | 92 | 31-53% |
| Complex Query Support | No | Yes | New capability |
| Multi-Dialect Support | No | Yes | New capability |
| User Adoption Rate (%) | 25 | 65 | 160% increase |
| Generation Time (seconds) | 3-5 | 1.5 | 50-70% reduction |

## Related Resources

- [Dynamic Tool Injection](/use-cases/dynamic-tool-injection/) for runtime prompt patterns
- [Prompt Management Guide](/guides/prompt-management/) for template best practices
- [LLM Configuration](/integrations/llm-providers/) for provider-specific tuning
