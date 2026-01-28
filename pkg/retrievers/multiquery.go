// Package retrievers provides implementations of the core.Retriever interface.
package retrievers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/lookatitude/beluga-ai/pkg/core"
	"github.com/lookatitude/beluga-ai/pkg/llms/iface"
	"github.com/lookatitude/beluga-ai/pkg/schema"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// MultiQueryRetriever implements the core.Retriever interface by generating
// multiple query variations using an LLM and then retrieving documents for each.
// This approach helps improve retrieval by considering different phrasings and perspectives.
type MultiQueryRetriever struct {
	retriever     core.Retriever
	llm           iface.ChatModel
	tracer        trace.Tracer
	logger        *slog.Logger
	metrics       *Metrics
	numQueries    int
	enableTracing bool
	enableMetrics bool
}

// newMultiQueryRetrieverInternal creates a new MultiQueryRetriever with internal configuration.
func newMultiQueryRetrieverInternal(retriever core.Retriever, llm iface.ChatModel, config *RetrieverOptions) *MultiQueryRetriever {
	numQueries := config.DefaultK
	if numQueries <= 0 {
		numQueries = 3
	}
	if numQueries > 10 {
		numQueries = 10
	}

	return &MultiQueryRetriever{
		retriever:     retriever,
		llm:           llm,
		numQueries:    numQueries,
		enableTracing: config.EnableTracing,
		enableMetrics: config.EnableMetrics,
		logger:        config.Logger,
		tracer:        config.Tracer,
		metrics:       config.Metrics,
	}
}

// GetRelevantDocuments retrieves documents by generating multiple query variations
// and combining results from all variations.
func (m *MultiQueryRetriever) GetRelevantDocuments(ctx context.Context, query string) ([]schema.Document, error) {
	startTime := time.Now()

	// Create tracing span
	var span trace.Span
	if m.enableTracing && m.tracer != nil {
		ctx, span = m.tracer.Start(ctx, "multi_query_retriever.retrieve",
			trace.WithAttributes(
				attribute.String("retriever.type", "multi_query"),
				attribute.String("original_query", query),
				attribute.Int("num_queries", m.numQueries),
			))
		defer span.End()
	}

	// Log the retrieval operation
	if m.logger != nil {
		m.logger.Info("multi-query retrieval started",
			"original_query", query,
			"num_queries", m.numQueries,
		)
	}

	// Generate query variations
	queryVariations, err := m.generateQueryVariations(ctx, query)
	if err != nil {
		if m.enableTracing && span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		if m.logger != nil {
			m.logger.Error("failed to generate query variations",
				"error", err,
				"query", query,
			)
		}
		return nil, NewRetrieverError("GetRelevantDocuments", err, ErrCodeQueryGenerationFailed)
	}

	// Combine original query with variations
	allQueries := append([]string{query}, queryVariations...)

	if m.logger != nil {
		m.logger.Info("query variations generated",
			"num_variations", len(queryVariations),
			"variations", queryVariations,
		)
	}

	if m.enableTracing && span != nil {
		span.SetAttributes(
			attribute.Int("query_variations_generated", len(queryVariations)),
			attribute.StringSlice("query_variations", queryVariations),
		)
	}

	// Retrieve documents for each query variation
	allDocuments := make(map[string]schema.Document)
	documentScores := make(map[string]float32)

	for _, q := range allQueries {
		docs, err := m.retriever.GetRelevantDocuments(ctx, q)
		if err != nil {
			// Log warning but continue with other queries
			if m.logger != nil {
				m.logger.Warn("retrieval failed for query variation",
					"error", err,
					"query", q,
				)
			}
			continue
		}

		// Deduplicate documents by ID
		for _, doc := range docs {
			docID := doc.ID
			if docID == "" {
				// Generate a simple ID from content hash
				docID = fmt.Sprintf("doc-%d", len(doc.GetContent()))
			}

			if _, exists := allDocuments[docID]; !exists {
				allDocuments[docID] = doc
				documentScores[docID] = 1.0 // Default score
			}
		}
	}

	// Convert map to slice
	resultDocs := make([]schema.Document, 0, len(allDocuments))
	for _, doc := range allDocuments {
		resultDocs = append(resultDocs, doc)
	}

	duration := time.Since(startTime)

	// Record metrics
	if m.enableMetrics && m.metrics != nil {
		avgScore := 0.0
		if len(documentScores) > 0 {
			sum := 0.0
			for _, score := range documentScores {
				sum += float64(score)
			}
			avgScore = sum / float64(len(documentScores))
		}

		m.metrics.RecordRetrieval(ctx, "multi_query", duration, len(resultDocs), avgScore, nil)
	}

	// Log successful retrieval
	if m.logger != nil {
		m.logger.Info("multi-query retrieval completed",
			"documents_returned", len(resultDocs),
			"unique_documents", len(allDocuments),
			"duration", duration,
		)
	}

	if m.enableTracing && span != nil {
		span.SetAttributes(
			attribute.Int("documents_returned", len(resultDocs)),
			attribute.Int("unique_documents", len(allDocuments)),
			attribute.Float64("duration_seconds", duration.Seconds()),
		)
		span.SetStatus(codes.Ok, "retrieval completed")
	}

	return resultDocs, nil
}

// generateQueryVariations generates multiple query variations using the LLM.
func (m *MultiQueryRetriever) generateQueryVariations(ctx context.Context, originalQuery string) ([]string, error) {
	prompt := fmt.Sprintf(`You are an AI language model assistant. Your task is to generate %d different versions of the given user question to retrieve relevant documents from a vector database. By generating multiple perspectives on the user question, your goal is to help the user overcome some of the limitations of distance-based similarity search.

Generate %d alternative questions that are similar in meaning but phrased differently. Return only the questions, one per line, without numbering or bullets.

Original question: %s`, m.numQueries, m.numQueries, originalQuery)

	messages := []schema.Message{
		schema.NewSystemMessage("You are a helpful assistant that generates query variations for information retrieval."),
		schema.NewHumanMessage(prompt),
	}

	response, err := m.llm.Generate(ctx, messages)
	if err != nil {
		return nil, fmt.Errorf("failed to generate query variations: %w", err)
	}

	// Parse response into individual queries
	responseText := response.GetContent()
	queries := m.parseQueryVariations(responseText)

	// Limit to requested number of variations
	if len(queries) > m.numQueries {
		queries = queries[:m.numQueries]
	}

	return queries, nil
}

// parseQueryVariations parses the LLM response to extract individual query variations.
func (m *MultiQueryRetriever) parseQueryVariations(responseText string) []string {
	lines := strings.Split(responseText, "\n")
	queries := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Remove common prefixes like "1.", "- ", "* "
		line = strings.TrimPrefix(line, "-")
		line = strings.TrimPrefix(line, "*")
		line = strings.TrimPrefix(line, "•")
		// Remove numbered prefixes
		for i := 1; i <= 10; i++ {
			prefix := fmt.Sprintf("%d.", i)
			line = strings.TrimPrefix(line, prefix)
		}
		line = strings.TrimSpace(line)

		if line != "" {
			queries = append(queries, line)
		}
	}

	return queries
}

// --- core.Runnable Implementation ---

// Invoke implements the core.Runnable interface.
func (m *MultiQueryRetriever) Invoke(ctx context.Context, input any, options ...core.Option) (any, error) {
	query, ok := input.(string)
	if !ok {
		return nil, NewRetrieverErrorWithMessage("Invoke", nil, ErrCodeInvalidInput,
			fmt.Sprintf("invalid input type for MultiQueryRetriever: expected string, got %T", input))
	}
	return m.GetRelevantDocuments(ctx, query)
}

// Batch implements the core.Runnable interface.
func (m *MultiQueryRetriever) Batch(ctx context.Context, inputs []any, options ...core.Option) ([]any, error) {
	results := make([]any, len(inputs))
	var firstErr error
	for i, input := range inputs {
		output, err := m.Invoke(ctx, input, options...)
		if err != nil && firstErr == nil {
			firstErr = NewRetrieverErrorWithMessage("Batch", err, ErrCodeRetrievalFailed,
				fmt.Sprintf("error processing batch item %d", i))
		}
		results[i] = output
	}
	return results, firstErr
}

// Stream implements the core.Runnable interface.
func (m *MultiQueryRetriever) Stream(ctx context.Context, input any, options ...core.Option) (<-chan any, error) {
	return nil, NewRetrieverErrorWithMessage("Stream", nil, ErrCodeInvalidInput,
		"streaming is not supported by MultiQueryRetriever")
}

// CheckHealth implements the core.HealthChecker interface.
func (m *MultiQueryRetriever) CheckHealth(ctx context.Context) error {
	// Create tracing span for health check
	var span trace.Span
	if m.enableTracing && m.tracer != nil {
		ctx, span = m.tracer.Start(ctx, "multi_query_retriever.health_check")
		defer span.End()
	}

	// Validate configuration
	if m.retriever == nil {
		err := NewRetrieverErrorWithMessage("CheckHealth", nil, ErrCodeInvalidConfig,
			"retriever is not configured")
		if m.enableTracing && span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}

	if m.llm == nil {
		err := NewRetrieverErrorWithMessage("CheckHealth", nil, ErrCodeInvalidConfig,
			"LLM is not configured")
		if m.enableTracing && span != nil {
			span.RecordError(err)
			span.SetStatus(codes.Error, err.Error())
		}
		return err
	}

	// Check underlying retriever health if it supports it
	if healthChecker, ok := m.retriever.(core.HealthChecker); ok {
		if err := healthChecker.CheckHealth(ctx); err != nil {
			if m.enableTracing && span != nil {
				span.RecordError(err)
				span.SetStatus(codes.Error, err.Error())
			}
			return err
		}
	}

	if m.enableTracing && span != nil {
		span.SetStatus(codes.Ok, "health check passed")
	}

	return nil
}

// Compile-time check to ensure MultiQueryRetriever implements interfaces.
var (
	_ core.Retriever     = (*MultiQueryRetriever)(nil)
	_ core.Runnable      = (*MultiQueryRetriever)(nil)
	_ core.HealthChecker = (*MultiQueryRetriever)(nil)
)
