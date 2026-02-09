---
title: Real-Time Data Analysis
description: Build an autonomous data analysis agent with tool access to databases, APIs, and streaming data using Beluga AI.
---

Business teams need real-time insights from diverse data sources — sales databases, analytics APIs, financial feeds, and operational metrics. Manually collecting, analyzing, and reporting this data is slow and does not scale. An AI agent with tool access can autonomously fetch data, perform calculations, and generate actionable insights on demand.

## Solution Architecture

Beluga AI's agent framework gives the analysis agent access to tools that connect to databases, REST APIs, and computation engines. The agent uses a planning strategy (ReAct or Self-Discover) to reason about which tools to use, fetches the data, analyzes it, and returns a natural language report with structured findings.

```
┌──────────────┐
│  Analyst      │
│  (Natural     │──────────────────────────────────────┐
│   Language    │                                       │
│   Query)      │                                       │
└──────┬───────┘                                       │
       │                                               │
       ▼                                               ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│  Agent       │───▶│  Tool:       │    │  Tool:       │
│  (ReAct      │    │  SQL Query   │    │  Calculator  │
│   Planner)   │───▶│              │    │              │
│              │    └──────────────┘    └──────────────┘
│              │───▶┌──────────────┐    ┌──────────────┐
│              │    │  Tool:       │    │  Tool:       │
│              │    │  REST API    │    │  Report Gen  │
└──────────────┘    └──────────────┘    └──────────────┘
```

## Building the Analysis Agent

Define tools that connect to your data sources, then let the agent reason about how to answer the question.

```go
package main

import (
    "context"
    "database/sql"
    "encoding/json"
    "fmt"
    "math"

    "github.com/lookatitude/beluga-ai/agent"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/tool"
    "github.com/lookatitude/beluga-ai/schema"

    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

// SQLQueryInput defines the input schema for the SQL tool.
type SQLQueryInput struct {
    Query string `json:"query" jsonschema:"description=Read-only SQL query to execute"`
}

// StatsInput defines the input schema for the statistics tool.
type StatsInput struct {
    Values []float64 `json:"values" jsonschema:"description=Numeric values to analyze"`
}

// StatsOutput holds computed statistics.
type StatsOutput struct {
    Mean   float64 `json:"mean"`
    Median float64 `json:"median"`
    StdDev float64 `json:"std_dev"`
    Min    float64 `json:"min"`
    Max    float64 `json:"max"`
    Count  int     `json:"count"`
}

func createAnalysisAgent(ctx context.Context, db *sql.DB) (agent.Agent, error) {
    model, err := llm.New("openai", nil)
    if err != nil {
        return nil, fmt.Errorf("create model: %w", err)
    }

    // SQL query tool — read-only access to the database
    sqlTool := tool.NewFuncTool[SQLQueryInput](
        "query_database",
        "Execute a read-only SQL query against the analytics database",
        func(ctx context.Context, input SQLQueryInput) (*tool.Result, error) {
            rows, err := db.QueryContext(ctx, input.Query)
            if err != nil {
                return tool.ErrorResult(err), nil
            }
            defer rows.Close()

            results, err := scanRowsToJSON(rows)
            if err != nil {
                return tool.ErrorResult(err), nil
            }

            data, err := json.MarshalIndent(results, "", "  ")
            if err != nil {
                return tool.ErrorResult(err), nil
            }
            return tool.TextResult(string(data)), nil
        },
    )

    // Statistics calculator tool
    statsTool := tool.NewFuncTool[StatsInput](
        "calculate_statistics",
        "Calculate mean, median, standard deviation, min, and max for a set of values",
        func(ctx context.Context, input StatsInput) (*tool.Result, error) {
            stats := computeStats(input.Values)
            data, err := json.MarshalIndent(stats, "", "  ")
            if err != nil {
                return tool.ErrorResult(err), nil
            }
            return tool.TextResult(string(data)), nil
        },
    )

    analysisAgent, err := agent.New(
        agent.WithID("data-analyst"),
        agent.WithPersona(agent.Persona{
            Role: "Senior Data Analyst",
            Goal: "Provide accurate, data-driven insights from available sources",
            Backstory: "You analyze data by querying databases, computing statistics, " +
                "and synthesizing findings into clear, actionable reports. " +
                "Always show your methodology and cite the data behind conclusions.",
        }),
        agent.WithModel(model),
        agent.WithTools(sqlTool, statsTool),
    )
    if err != nil {
        return nil, fmt.Errorf("create agent: %w", err)
    }

    return analysisAgent, nil
}
```

## Executing Analysis Queries

The agent autonomously decides which tools to use based on the question:

```go
func main() {
    ctx := context.Background()

    db, err := sql.Open("postgres", os.Getenv("DATABASE_URL"))
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    analysisAgent, err := createAnalysisAgent(ctx, db)
    if err != nil {
        log.Fatal(err)
    }

    // The agent will: 1) query the database, 2) compute stats, 3) generate report
    answer, err := analysisAgent.Invoke(ctx,
        "What were our top 5 products by revenue last quarter? "+
        "Include growth rates compared to the previous quarter.",
    )
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println(answer)
}
```

## Streaming Analysis Results

For long-running analyses, stream intermediate results so the user sees progress:

```go
for event, err := range analysisAgent.Stream(ctx, "Analyze customer churn trends") {
    if err != nil {
        log.Fatal(err)
    }

    switch event.Type {
    case agent.EventToolCall:
        fmt.Printf("Querying: %s\n", event.ToolCall.Name)
    case agent.EventToolResult:
        fmt.Printf("Got data (%d chars)\n", len(event.ToolResult.Content[0].(schema.TextPart).Text))
    case agent.EventText:
        fmt.Print(event.Text)
    }
}
```

## Structured Report Output

When you need structured reports rather than free-form text, combine the agent with structured output:

```go
type AnalysisReport struct {
    Title        string    `json:"title"`
    Summary      string    `json:"summary"`
    KeyFindings  []Finding `json:"key_findings"`
    Methodology  string    `json:"methodology"`
    DataSources  []string  `json:"data_sources"`
}

type Finding struct {
    Metric     string  `json:"metric"`
    Value      string  `json:"value"`
    Trend      string  `json:"trend" jsonschema:"enum=up,down,flat"`
    Confidence float64 `json:"confidence"`
}

func generateReport(ctx context.Context, model llm.ChatModel, rawAnalysis string) (AnalysisReport, error) {
    structured := llm.NewStructured[AnalysisReport](model)

    msgs := []schema.Message{
        &schema.SystemMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: "Structure this analysis into a formal report."},
        }},
        &schema.HumanMessage{Parts: []schema.ContentPart{
            schema.TextPart{Text: rawAnalysis},
        }},
    }

    return structured.Generate(ctx, msgs)
}
```

## Adding External API Tools

Extend the agent's capabilities by adding tools for external data sources:

```go
type APIQueryInput struct {
    Endpoint string            `json:"endpoint" jsonschema:"description=API endpoint path"`
    Params   map[string]string `json:"params" jsonschema:"description=Query parameters"`
}

apiTool := tool.NewFuncTool[APIQueryInput](
    "query_analytics_api",
    "Fetch data from the analytics REST API",
    func(ctx context.Context, input APIQueryInput) (*tool.Result, error) {
        resp, err := analyticsClient.Get(ctx, input.Endpoint, input.Params)
        if err != nil {
            return tool.ErrorResult(err), nil
        }
        return tool.TextResult(string(resp)), nil
    },
)
```

## Production Considerations

### Safety Guards

Protect the database from harmful queries using Beluga AI's guard pipeline:

```go
import "github.com/lookatitude/beluga-ai/guard"

// Guard that blocks write operations in SQL
sqlGuard := guard.GuardFunc("sql-read-only", func(ctx context.Context, input guard.GuardInput) (guard.GuardResult, error) {
    // Check for write keywords
    upper := strings.ToUpper(input.Content)
    for _, keyword := range []string{"INSERT", "UPDATE", "DELETE", "DROP", "ALTER", "TRUNCATE"} {
        if strings.Contains(upper, keyword) {
            return guard.GuardResult{
                Allowed: false,
                Reason:  "Write operations are not permitted",
            }, nil
        }
    }
    return guard.GuardResult{Allowed: true}, nil
})
```

### Resilience

Wrap database and API calls with retry logic for transient failures:

```go
import "github.com/lookatitude/beluga-ai/resilience"

policy := resilience.RetryPolicy{
    MaxAttempts:    3,
    InitialBackoff: 200 * time.Millisecond,
    MaxBackoff:     2 * time.Second,
    BackoffFactor:  2.0,
    Jitter:         true,
}

result, err := resilience.Retry(ctx, policy, func(ctx context.Context) (string, error) {
    return analysisAgent.Invoke(ctx, question)
})
```

### Observability

Track agent execution, tool calls, and query performance:

```go
tracer := otel.Tracer("data-analyst")
ctx, span := tracer.Start(ctx, "analysis.query")
defer span.End()

span.SetAttributes(
    attribute.String("analysis.question", question),
    attribute.Int("analysis.tool_calls", toolCallCount),
    attribute.Float64("analysis.duration_ms", durationMs),
)
```

### Scaling

- **Connection pooling**: Use `sql.DB` connection pooling for database queries. Set `MaxOpenConns` and `MaxIdleConns` appropriately.
- **Query timeouts**: Set `context.WithTimeout` on each database query to prevent long-running queries from blocking the agent.
- **Caching**: Cache frequently requested analyses with TTL-based expiration. Use Beluga AI's `cache/` package for semantic cache.
- **Rate limiting**: Limit the number of concurrent agent executions to prevent database overload.

### Security

- Use read-only database credentials for the SQL tool
- Validate and sanitize SQL queries through the guard pipeline before execution
- Restrict API tool access to approved endpoints
- Log all queries for audit purposes using OpenTelemetry

## Related Resources

- [Building Your First Agent](/guides/first-agent/) for planner strategy selection
- [Tools & MCP](/guides/tools-and-mcp/) for building custom tools
- [LLM Recipes](/cookbook/llm-recipes/) for typed responses
