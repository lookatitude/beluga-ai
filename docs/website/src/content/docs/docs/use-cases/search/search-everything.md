---
title: Internal Search Everything Bot
description: "Build a unified search bot that queries across Confluence, GitHub, Slack, and databases. Single interface for all enterprise knowledge."
head:
  - tag: meta
    attrs:
      name: keywords
      content: "unified enterprise search, search bot, cross-system search, MCP search, knowledge discovery, Beluga AI, Go, internal tools"
---

Large enterprises maintain knowledge across fragmented systems — documentation in Confluence, code in GitHub, discussions in Slack, data in databases, policies in SharePoint. Each system has its own search interface with different query syntax and different result formats. Employees waste significant time switching between search interfaces, and cross-system questions ("who implemented this feature and what was the design rationale?") require manual correlation across multiple tools.

A unified search bot aggregates results from all sources, applies intelligent ranking, and provides a single interface for knowledge discovery. The orchestrator queries systems in parallel (for speed) and uses score-based ranking to merge results into a single relevance-ordered list, regardless of source.

## Solution Architecture

Beluga AI's server package provides both REST and MCP (Model Context Protocol) APIs for flexible integration. The search orchestrator queries multiple systems in parallel, aggregates results using score-based ranking, and returns unified results with source attribution and relevance scores.

Parallel querying is essential because total latency must be bounded by the slowest source, not the sum of all sources. Score-based ranking (rather than simple interleaving) ensures that a highly relevant result from one source ranks above marginally relevant results from another, regardless of which system responded first.

```
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│     User     │───▶│  Search API  │───▶│    Search    │
│    Query     │    │ (REST/MCP)   │    │ Orchestrator │
└──────────────┘    └──────────────┘    └──────┬───────┘
                                               │
                           ┌───────────────────┼───────────────────┐
                           │                   │                   │
                           ▼                   ▼                   ▼
                    ┌──────────────┐    ┌──────────────┐    ┌──────────────┐
                    │     Docs     │    │     Code     │    │     Wiki     │
                    │   Searcher   │    │   Searcher   │    │   Searcher   │
                    └──────┬───────┘    └──────┬───────┘    └──────┬───────┘
                           │                   │                   │
                           └───────────────────┼───────────────────┘
                                               │
                                               ▼
                                        ┌──────────────┐
                                        │    Result    │
                                        │  Aggregator  │
                                        │   + Ranker   │
                                        └──────┬───────┘
                                               │
                                               ▼
                                        ┌──────────────┐
                                        │   Unified    │
                                        │   Results    │
                                        └──────────────┘
```

## Search Server Setup

Create REST and MCP servers for flexible integration:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/server"
    "github.com/lookatitude/beluga-ai/rag/retriever"
    "go.opentelemetry.io/otel/trace"
    "go.opentelemetry.io/otel/metric"
)

type SearchEverythingBot struct {
    restServer   *server.RESTServer
    mcpServer    *server.MCPServer
    orchestrator *SearchOrchestrator
    tracer       trace.Tracer
    meter        metric.Meter
}

func NewSearchEverythingBot(ctx context.Context) (*SearchEverythingBot, error) {
    // Setup REST server
    restServer, err := server.NewRESTServer(
        server.WithHost("0.0.0.0"),
        server.WithPort(8080),
        server.WithBasePath("/api/v1"),
    )
    if err != nil {
        return nil, fmt.Errorf("create REST server: %w", err)
    }

    // Setup MCP server
    mcpServer, err := server.NewMCPServer(
        server.WithName("search-everything"),
        server.WithDescription("Search across all internal systems"),
    )
    if err != nil {
        return nil, fmt.Errorf("create MCP server: %w", err)
    }

    return &SearchEverythingBot{
        restServer:   restServer,
        mcpServer:    mcpServer,
        orchestrator: NewSearchOrchestrator(),
    }, nil
}
```

## Multi-Source Search Orchestration

Query all systems in parallel and aggregate results:

```go
package main

import (
    "context"
    "sync"

    "go.opentelemetry.io/otel/attribute"
)

type SearchOrchestrator struct {
    docSearcher  *DocumentationSearcher
    codeSearcher *CodeSearcher
    wikiSearcher *WikiSearcher
    dbSearcher   *DatabaseSearcher
}

type SearchResult struct {
    Title       string
    Content     string
    Source      string // "docs", "code", "wiki", "database"
    SourceURL   string
    Score       float64
    Metadata    map[string]interface{}
}

type SourceResults struct {
    Source  string
    Results []SearchResult
    Error   error
}

func (s *SearchOrchestrator) SearchEverything(ctx context.Context, query string, limit int) ([]SearchResult, error) {
    // Search all sources in parallel
    resultsChan := make(chan SourceResults, 4)
    var wg sync.WaitGroup

    wg.Add(4)
    go func() {
        defer wg.Done()
        results, err := s.docSearcher.Search(ctx, query)
        resultsChan <- SourceResults{Source: "docs", Results: results, Error: err}
    }()

    go func() {
        defer wg.Done()
        results, err := s.codeSearcher.Search(ctx, query)
        resultsChan <- SourceResults{Source: "code", Results: results, Error: err}
    }()

    go func() {
        defer wg.Done()
        results, err := s.wikiSearcher.Search(ctx, query)
        resultsChan <- SourceResults{Source: "wiki", Results: results, Error: err}
    }()

    go func() {
        defer wg.Done()
        results, err := s.dbSearcher.Search(ctx, query)
        resultsChan <- SourceResults{Source: "database", Results: results, Error: err}
    }()

    go func() {
        wg.Wait()
        close(resultsChan)
    }()

    // Collect all results
    allResults := make([]SearchResult, 0)
    for sourceResults := range resultsChan {
        if sourceResults.Error != nil {
            // Log error but continue with other sources
            continue
        }
        allResults = append(allResults, sourceResults.Results...)
    }

    // Rank and limit results
    ranked := s.rankResults(allResults, query)
    if len(ranked) > limit {
        ranked = ranked[:limit]
    }

    return ranked, nil
}

func (s *SearchOrchestrator) rankResults(results []SearchResult, query string) []SearchResult {
    // Score-based ranking combining:
    // 1. Source relevance score
    // 2. Freshness (if available in metadata)
    // 3. Query term matching
    // 4. Source authority (docs > wiki > code > database)

    sourceWeights := map[string]float64{
        "docs":     1.0,
        "wiki":     0.9,
        "code":     0.8,
        "database": 0.7,
    }

    for i := range results {
        weight := sourceWeights[results[i].Source]
        results[i].Score *= weight
    }

    // Sort by score descending
    sort.Slice(results, func(i, j int) bool {
        return results[i].Score > results[j].Score
    })

    return results
}
```

## Search API Endpoints

Expose REST and MCP interfaces:

```go
package main

import (
    "encoding/json"
    "net/http"
)

func (s *SearchEverythingBot) SetupAPIs(ctx context.Context) error {
    // REST endpoint
    s.restServer.HandleFunc("POST", "/search", func(w http.ResponseWriter, r *http.Request) {
        var req struct {
            Query string `json:"query"`
            Limit int    `json:"limit,omitempty"`
        }

        if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
            http.Error(w, "invalid request", http.StatusBadRequest)
            return
        }

        if req.Limit == 0 {
            req.Limit = 10
        }

        results, err := s.orchestrator.SearchEverything(r.Context(), req.Query, req.Limit)
        if err != nil {
            http.Error(w, err.Error(), http.StatusInternalServerError)
            return
        }

        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(map[string]interface{}{
            "query":   req.Query,
            "results": results,
            "count":   len(results),
        })
    })

    // MCP tool registration
    s.mcpServer.RegisterTool("search_everything", server.Tool{
        Description: "Search across all internal systems (docs, code, wikis, databases)",
        InputSchema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "query": map[string]interface{}{
                    "type":        "string",
                    "description": "Search query",
                },
                "limit": map[string]interface{}{
                    "type":        "integer",
                    "description": "Maximum number of results (default: 10)",
                    "default":     10,
                },
            },
            "required": []string{"query"},
        },
    }, s.handleMCPSearch)

    return nil
}

func (s *SearchEverythingBot) handleMCPSearch(ctx context.Context, params map[string]interface{}) (interface{}, error) {
    query := params["query"].(string)
    limit := 10
    if l, ok := params["limit"].(int); ok {
        limit = l
    }

    results, err := s.orchestrator.SearchEverything(ctx, query, limit)
    if err != nil {
        return nil, err
    }

    return map[string]interface{}{
        "query":   query,
        "results": results,
        "count":   len(results),
    }, nil
}
```

## System-Specific Searchers

Implement adapters for each source system:

```go
package main

import (
    "context"
    "github.com/lookatitude/beluga-ai/rag/retriever"
)

type DocumentationSearcher struct {
    retriever retriever.Retriever
}

func (d *DocumentationSearcher) Search(ctx context.Context, query string) ([]SearchResult, error) {
    docs, err := d.retriever.Retrieve(ctx, query, retriever.WithTopK(20))
    if err != nil {
        return nil, err
    }

    results := make([]SearchResult, len(docs))
    for i, doc := range docs {
        results[i] = SearchResult{
            Title:     doc.Metadata["title"].(string),
            Content:   doc.Content,
            Source:    "docs",
            SourceURL: doc.Metadata["url"].(string),
            Score:     doc.Metadata["score"].(float64),
            Metadata:  doc.Metadata,
        }
    }

    return results, nil
}

type CodeSearcher struct {
    retriever retriever.Retriever
}

func (c *CodeSearcher) Search(ctx context.Context, query string) ([]SearchResult, error) {
    docs, err := c.retriever.Retrieve(ctx, query, retriever.WithTopK(20))
    if err != nil {
        return nil, err
    }

    results := make([]SearchResult, len(docs))
    for i, doc := range docs {
        results[i] = SearchResult{
            Title:     doc.Metadata["file_path"].(string),
            Content:   doc.Content,
            Source:    "code",
            SourceURL: doc.Metadata["repo_url"].(string),
            Score:     doc.Metadata["score"].(float64),
            Metadata:  doc.Metadata,
        }
    }

    return results, nil
}
```

## Production Considerations

### Observability

Track search metrics and source performance:

```go
import (
    "github.com/lookatitude/beluga-ai/o11y"
    "go.opentelemetry.io/otel/attribute"
)

func (s *SearchOrchestrator) SearchWithObservability(ctx context.Context, query string, limit int) ([]SearchResult, error) {
    ctx, span := s.tracer.Start(ctx, "search.everything")
    defer span.End()

    span.SetAttributes(
        attribute.String("query", query),
        attribute.Int("limit", limit),
    )

    start := time.Now()
    results, err := s.SearchEverything(ctx, query, limit)
    duration := time.Since(start)

    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(
        attribute.Int("results.count", len(results)),
        attribute.Float64("duration.ms", float64(duration.Milliseconds())),
    )

    s.meter.RecordHistogram(ctx, "search.duration", duration.Milliseconds())
    s.meter.IncrementCounter(ctx, "search.requests")

    return results, nil
}
```

### Caching

Cache frequent queries to reduce latency:

```go
import "github.com/lookatitude/beluga-ai/cache"

type CachedSearchOrchestrator struct {
    SearchOrchestrator
    cache cache.Cache
}

func (c *CachedSearchOrchestrator) SearchEverything(ctx context.Context, query string, limit int) ([]SearchResult, error) {
    cacheKey := fmt.Sprintf("search:%s:%d", query, limit)

    // Check cache first
    if cached, ok := c.cache.Get(ctx, cacheKey); ok {
        return cached.([]SearchResult), nil
    }

    // Perform search
    results, err := c.SearchOrchestrator.SearchEverything(ctx, query, limit)
    if err != nil {
        return nil, err
    }

    // Cache results for 5 minutes
    c.cache.Set(ctx, cacheKey, results, 5*time.Minute)

    return results, nil
}
```

### Access Control

Filter results based on user permissions:

```go
func (s *SearchOrchestrator) SearchWithAccessControl(ctx context.Context, userID, query string, limit int) ([]SearchResult, error) {
    results, err := s.SearchEverything(ctx, query, limit*2)
    if err != nil {
        return nil, err
    }

    // Filter based on user permissions
    filtered := make([]SearchResult, 0)
    for _, result := range results {
        if s.canAccess(userID, result) {
            filtered = append(filtered, result)
            if len(filtered) >= limit {
                break
            }
        }
    }

    return filtered, nil
}

func (s *SearchOrchestrator) canAccess(userID string, result SearchResult) bool {
    // Check document-level permissions
    // Implementation depends on your access control system
    return true
}
```

## Related Resources

- [Server Package Guide](/guides/server-patterns/) for REST and MCP APIs
- [Retriever Guide](/guides/retriever-patterns/) for multi-source retrieval
- [Customer Support Gateway](/use-cases/support-gateway/) for API gateway patterns
- [Knowledge QA System](/use-cases/knowledge-qa/) for semantic search patterns
