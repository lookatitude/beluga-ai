---
title: "Advanced Meta-filtering"
package: "vectorstores"
category: "filtering"
complexity: "intermediate"
---

# Advanced Meta-filtering

## Problem

You need to perform complex metadata filtering on vector searches, such as filtering by date ranges, multiple categories, numeric comparisons, or combining multiple filter conditions with AND/OR logic.

## Solution

Implement a metadata filter builder that supports complex query conditions, type-aware filtering, and efficient execution. This works because Beluga AI's vectorstores support metadata filters, and you can build sophisticated filtering logic on top of the basic filter support.

## Code Example
```go
package main

import (
    "context"
    "fmt"
    "log"
    "time"
    
    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
    "go.opentelemetry.io/otel/trace"
    
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores/iface"
)

var tracer = otel.Tracer("beluga.vectorstores.filtering")

// FilterCondition represents a single filter condition
type FilterCondition struct {
    Key      string
    Operator string // "eq", "ne", "gt", "gte", "lt", "lte", "in", "contains"
    Value    interface{}
}

// MetadataFilterBuilder builds complex metadata filters
type MetadataFilterBuilder struct {
    conditions []FilterCondition
    logic      string // "AND" or "OR"
}

// NewMetadataFilterBuilder creates a new filter builder
func NewMetadataFilterBuilder() *MetadataFilterBuilder {
    return &MetadataFilterBuilder{
        conditions: []FilterCondition{},
        logic:      "AND",
    }
}

// WithLogic sets the logic operator (AND or OR)
func (mfb *MetadataFilterBuilder) WithLogic(logic string) *MetadataFilterBuilder {
    mfb.logic = logic
    return mfb
}

// Equals adds an equality condition
func (mfb *MetadataFilterBuilder) Equals(key string, value interface{}) *MetadataFilterBuilder {
    mfb.conditions = append(mfb.conditions, FilterCondition{
        Key:      key,
        Operator: "eq",
        Value:    value,
    })
    return mfb
}

// NotEquals adds a not-equals condition
func (mfb *MetadataFilterBuilder) NotEquals(key string, value interface{}) *MetadataFilterBuilder {
    mfb.conditions = append(mfb.conditions, FilterCondition{
        Key:      key,
        Operator: "ne",
        Value:    value,
    })
    return mfb
}

// GreaterThan adds a greater-than condition
func (mfb *MetadataFilterBuilder) GreaterThan(key string, value interface{}) *MetadataFilterBuilder {
    mfb.conditions = append(mfb.conditions, FilterCondition{
        Key:      key,
        Operator: "gt",
        Value:    value,
    })
    return mfb
}

// In adds an "in" condition (value must be in list)
func (mfb *MetadataFilterBuilder) In(key string, values []interface{}) *MetadataFilterBuilder {
    mfb.conditions = append(mfb.conditions, FilterCondition{
        Key:      key,
        Operator: "in",
        Value:    values,
    })
    return mfb
}

// DateRange adds a date range condition
func (mfb *MetadataFilterBuilder) DateRange(key string, start, end time.Time) *MetadataFilterBuilder {
    mfb.conditions = append(mfb.conditions, FilterCondition{
        Key:      key,
        Operator: "range",
        Value:    map[string]interface{}{"start": start, "end": end},
    })
    return mfb
}

// Build builds the filter map for vectorstore
func (mfb *MetadataFilterBuilder) Build() map[string]interface{} {
    if len(mfb.conditions) == 0 {
        return nil
    }
    
    if len(mfb.conditions) == 1 {
        return mfb.buildSingleCondition(mfb.conditions[0])
    }
    
    // Build complex filter
    filter := map[string]interface{}{
        "logic": mfb.logic,
        "conditions": []map[string]interface{}{},
    }
    
    for _, cond := range mfb.conditions {
        filter["conditions"] = append(filter["conditions"].([]map[string]interface{}), mfb.buildCondition(cond))
    }
    
    return filter
}

// buildSingleCondition builds a simple single-condition filter
func (mfb *MetadataFilterBuilder) buildSingleCondition(cond FilterCondition) map[string]interface{} {
    return mfb.buildCondition(cond)
}

// buildCondition builds a condition map
func (mfb *MetadataFilterBuilder) buildCondition(cond FilterCondition) map[string]interface{} {
    condition := map[string]interface{}{
        "key":      cond.Key,
        "operator": cond.Operator,
        "value":    cond.Value,
    }
    return condition
}

// ApplyFilter applies the filter to a vectorstore search
func ApplyFilter(ctx context.Context, store vectorstores.VectorStore, query string, k int, embedder interface{}, builder *MetadataFilterBuilder) ([]schema.Document, []float32, error) {
    ctx, span := tracer.Start(ctx, "filtering.apply_filter")
    defer span.End()
    
    filter := builder.Build()
    span.SetAttributes(
        attribute.Int("filter.condition_count", len(builder.conditions)),
        attribute.String("filter.logic", builder.logic),
    )
    
    // Convert filter to vectorstore format
    opts := []vectorstores.Option{}
    if filter != nil {
        // Apply each condition
        for key, value := range filter {
            if key != "logic" && key != "conditions" {
                opts = append(opts, vectorstores.WithMetadataFilter(key, value))
            }
        }
        
        // Handle complex filters
        if conditions, ok := filter["conditions"].([]map[string]interface{}); ok {
            for _, cond := range conditions {
                if key, ok := cond["key"].(string); ok {
                    opts = append(opts, vectorstores.WithMetadataFilter(key, cond["value"]))
                }
            }
        }
    }
    
    // Perform search with filters
    results, scores, err := store.SimilaritySearchByQuery(ctx, query, k, embedder, opts...)
    if err != nil {
        span.RecordError(err)
        span.SetStatus(trace.StatusError, err.Error())
        return nil, nil, err
    }
    
    // Post-filter if needed (for operators not supported by vectorstore)
    filteredResults, filteredScores := postFilter(results, scores, builder.conditions)
    
    span.SetAttributes(attribute.Int("results.count", len(filteredResults)))
    span.SetStatus(trace.StatusOK, "filter applied successfully")
    
    return filteredResults, filteredScores, nil
}

// postFilter applies filters that the vectorstore doesn't support natively
func postFilter(docs []schema.Document, scores []float32, conditions []FilterCondition) ([]schema.Document, []float32) {
    filteredDocs := []schema.Document{}
    filteredScores := []float32{}

    for i, doc := range docs {
        if matchesConditions(doc, conditions) {
            filteredDocs = append(filteredDocs, doc)
            filteredScores = append(filteredScores, scores[i])
        }
    }
    
    return filteredDocs, filteredScores
}

// matchesConditions checks if a document matches all conditions
func matchesConditions(doc schema.Document, conditions []FilterCondition) bool {
    meta := doc.GetMetadata()

    for _, cond := range conditions {
        if !matchesCondition(meta, cond) {
            return false
        }
    }
    
    return true
}

// matchesCondition checks if metadata matches a single condition
func matchesCondition(meta map[string]string, cond FilterCondition) bool {
    value, exists := meta[cond.Key]
    if !exists {
        return false
    }
    
    switch cond.Operator {
    case "eq":
        return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", cond.Value)
    case "ne":
        return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", cond.Value)
    case "in":
        if values, ok := cond.Value.([]interface{}); ok {
            for _, v := range values {
                if fmt.Sprintf("%v", value) == fmt.Sprintf("%v", v) {
                    return true
                }
            }
        }
        return false
    default:
        return true // Unknown operators pass through
    }
}

func main() {
    ctx := context.Background()

    // Create filter builder
    builder := NewMetadataFilterBuilder().
        Equals("category", "tech").
        DateRange("created_at", time.Now().AddDate(0, -1, 0), time.Now()).
        In("tags", []interface{}{"ai", "ml"})
    
    // Apply to search
    // store := yourVectorStore
    // embedder := yourEmbedder
    // results, scores, err := ApplyFilter(ctx, store, "machine learning", 10, embedder, builder)
    fmt.Println("Filter builder created")
}
```

## Explanation

Let's break down what's happening:

1. **Fluent builder pattern** - Notice how we use method chaining to build complex filters. This makes the code readable and allows composing filters incrementally.

2. **Operator support** - We support various operators (equals, not-equals, in, range) that cover most filtering needs. The vectorstore may support some natively, while others require post-filtering.

3. **Post-filtering fallback** - For operators the vectorstore doesn't support natively, we post-filter results. This ensures all filter types work, though native filtering is more efficient.

```go
**Key insight:** Use native vectorstore filtering when possible for performance, but provide post-filtering as a fallback for complex conditions. This gives you flexibility without sacrificing functionality.

## Testing

```
Here's how to test this solution:
```go
func TestMetadataFilterBuilder_BuildsComplexFilter(t *testing.T) {
    builder := NewMetadataFilterBuilder().
        Equals("category", "tech").
        In("tags", []interface{}{"ai", "ml"})
    
    filter := builder.Build()
    require.NotNil(t, filter)
    require.Len(t, builder.conditions, 2)
}

## Variations

### Filter Caching

Cache filter results for repeated queries:
type CachedFilter struct {
    cache map[string][]schema.Document
}
```

### Filter Validation

Validate filters before execution:
```go
func (mfb *MetadataFilterBuilder) Validate() error {
    // Check for conflicting conditions
}
```

## Related Recipes

- **[Vectorstores Re-indexing Status Tracking](./vectorstores-reindexing-status-tracking.md)** - Track reindexing operations
- **[Embeddings Metadata-aware Embedding Clusters](./embeddings-metadata-aware-clusters.md)** - Cluster with metadata
- **[Vectorstores Package Guide](../package_design_patterns.md)** - For a deeper understanding of vectorstores
