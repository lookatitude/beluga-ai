---
title: "Advanced Metadata Filtering"
description: "Build complex metadata filters for vector searches with date ranges, categories, and AND/OR logic."
---

## Problem

You need to perform complex metadata filtering on vector searches, such as filtering by date ranges, multiple categories, numeric comparisons, or combining multiple filter conditions with AND/OR logic.

## Solution

Implement a metadata filter builder that supports complex query conditions, type-aware filtering, and efficient execution. Use a fluent API to compose filters, with native vectorstore filtering where possible and post-filtering as a fallback for unsupported operators.

## Code Example

```go
package main

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var tracer = otel.Tracer("beluga.vectorstores.filtering")

// FilterCondition represents a single filter condition.
type FilterCondition struct {
	Key      string
	Operator string // "eq", "ne", "gt", "gte", "lt", "lte", "in", "range"
	Value    interface{}
}

// MetadataFilterBuilder builds complex metadata filters.
type MetadataFilterBuilder struct {
	conditions []FilterCondition
	logic      string // "AND" or "OR"
}

func NewMetadataFilterBuilder() *MetadataFilterBuilder {
	return &MetadataFilterBuilder{
		conditions: []FilterCondition{},
		logic:      "AND",
	}
}

func (mfb *MetadataFilterBuilder) WithLogic(logic string) *MetadataFilterBuilder {
	mfb.logic = logic
	return mfb
}

func (mfb *MetadataFilterBuilder) Equals(key string, value interface{}) *MetadataFilterBuilder {
	mfb.conditions = append(mfb.conditions, FilterCondition{Key: key, Operator: "eq", Value: value})
	return mfb
}

func (mfb *MetadataFilterBuilder) NotEquals(key string, value interface{}) *MetadataFilterBuilder {
	mfb.conditions = append(mfb.conditions, FilterCondition{Key: key, Operator: "ne", Value: value})
	return mfb
}

func (mfb *MetadataFilterBuilder) In(key string, values []interface{}) *MetadataFilterBuilder {
	mfb.conditions = append(mfb.conditions, FilterCondition{Key: key, Operator: "in", Value: values})
	return mfb
}

func (mfb *MetadataFilterBuilder) DateRange(key string, start, end time.Time) *MetadataFilterBuilder {
	mfb.conditions = append(mfb.conditions, FilterCondition{
		Key:      key,
		Operator: "range",
		Value:    map[string]interface{}{"start": start, "end": end},
	})
	return mfb
}

// Build returns the filter as a map for vectorstore consumption.
func (mfb *MetadataFilterBuilder) Build() map[string]interface{} {
	if len(mfb.conditions) == 0 {
		return nil
	}

	conditionMaps := make([]map[string]interface{}, 0, len(mfb.conditions))
	for _, cond := range mfb.conditions {
		conditionMaps = append(conditionMaps, map[string]interface{}{
			"key":      cond.Key,
			"operator": cond.Operator,
			"value":    cond.Value,
		})
	}

	return map[string]interface{}{
		"logic":      mfb.logic,
		"conditions": conditionMaps,
	}
}

// MatchesDocument checks if a document's metadata matches the filter conditions.
func (mfb *MetadataFilterBuilder) MatchesDocument(meta map[string]string) bool {
	for _, cond := range mfb.conditions {
		value, exists := meta[cond.Key]
		if !exists {
			return false
		}
		switch cond.Operator {
		case "eq":
			if value != fmt.Sprintf("%v", cond.Value) {
				return false
			}
		case "ne":
			if value == fmt.Sprintf("%v", cond.Value) {
				return false
			}
		case "in":
			found := false
			if values, ok := cond.Value.([]interface{}); ok {
				for _, v := range values {
					if value == fmt.Sprintf("%v", v) {
						found = true
						break
					}
				}
			}
			if !found {
				return false
			}
		}
	}
	return true
}

func main() {
	ctx := context.Background()
	_, span := tracer.Start(ctx, "example")
	defer span.End()

	builder := NewMetadataFilterBuilder().
		Equals("category", "tech").
		DateRange("created_at", time.Now().AddDate(0, -1, 0), time.Now()).
		In("tags", []interface{}{"ai", "ml"})

	filter := builder.Build()
	span.SetAttributes(attribute.Int("filter.conditions", len(builder.conditions)))
	fmt.Printf("Filter: %v\n", filter)
}
```

## Explanation

1. **Fluent builder pattern** -- Method chaining makes filter construction readable and allows composing filters incrementally. Each method returns the builder for continued chaining.

2. **Operator support** -- Multiple operators (equals, not-equals, in, range) cover most filtering needs. The vectorstore may support some natively, while others require post-filtering.

3. **Post-filtering fallback** -- The `MatchesDocument` method provides client-side filtering for operators the vectorstore does not support natively, ensuring all filter types work regardless of backend.

**Key insight:** Use native vectorstore filtering for performance, but provide post-filtering as a fallback for complex conditions. This gives you flexibility without sacrificing functionality.

## Variations

### Filter Validation

Validate filters before execution to catch conflicting conditions early:

```go
func (mfb *MetadataFilterBuilder) Validate() error {
	keys := make(map[string]int)
	for _, cond := range mfb.conditions {
		keys[cond.Key]++
	}
	for key, count := range keys {
		if count > 1 && mfb.logic == "AND" {
			return fmt.Errorf("conflicting conditions on key %q", key)
		}
	}
	return nil
}
```

## Related Recipes

- **[Reindexing Status Tracking](./reindexing-tracking)** -- Track reindexing operations
- **[Metadata-Aware Clustering](./metadata-clustering)** -- Cluster embeddings with metadata
