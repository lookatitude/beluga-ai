package structured

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/lookatitude/beluga-ai/config"
	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/rag/retriever"
	"github.com/lookatitude/beluga-ai/schema"
)

func init() {
	retriever.Register("structured", func(_ config.ProviderConfig) (retriever.Retriever, error) {
		return nil, core.NewError(
			"structured.new",
			core.ErrInvalidInput,
			"use structured.NewStructuredRetriever() with explicit dependencies; "+
				"registry entry exists for discovery only",
			nil,
		)
	})
}

// defaultMaxRetries is the number of retry attempts when the evaluator scores
// results below the threshold.
const defaultMaxRetries = 3

// defaultMinScore is the minimum evaluator score required to accept results.
const defaultMinScore = 0.5

// options holds the configurable options for a StructuredRetriever.
type options struct {
	generator  QueryGenerator
	executor   QueryExecutor
	evaluator  ResultEvaluator
	schema     SchemaInfo
	maxRetries int
	minScore   float64
	hooks      retriever.Hooks
}

// RetrieverOption configures a StructuredRetriever.
type RetrieverOption func(*options)

// WithGenerator sets the query generator.
func WithGenerator(g QueryGenerator) RetrieverOption {
	return func(o *options) { o.generator = g }
}

// WithExecutor sets the query executor.
func WithExecutor(e QueryExecutor) RetrieverOption {
	return func(o *options) { o.executor = e }
}

// WithEvaluator sets the result evaluator. When nil, evaluation is skipped
// and the first result set is always accepted.
func WithEvaluator(e ResultEvaluator) RetrieverOption {
	return func(o *options) { o.evaluator = e }
}

// WithSchema sets the database schema information used for query generation.
func WithSchema(s SchemaInfo) RetrieverOption {
	return func(o *options) { o.schema = s }
}

// WithMaxRetries sets the maximum number of generation retries when the
// evaluator score is below the threshold. Default is 3.
func WithMaxRetries(n int) RetrieverOption {
	return func(o *options) { o.maxRetries = n }
}

// WithMinScore sets the minimum evaluator score to accept a result set.
// Default is 0.5.
func WithMinScore(s float64) RetrieverOption {
	return func(o *options) { o.minScore = s }
}

// WithHooks sets lifecycle hooks on the retriever.
func WithHooks(h retriever.Hooks) RetrieverOption {
	return func(o *options) { o.hooks = h }
}

// StructuredRetriever implements [retriever.Retriever] by generating structured
// database queries from natural language, executing them, and optionally
// evaluating result relevance with a retry loop.
type StructuredRetriever struct {
	opts options
}

// Compile-time interface check.
var _ retriever.Retriever = (*StructuredRetriever)(nil)

// NewStructuredRetriever creates a retriever with the given options.
// A generator and executor are required.
func NewStructuredRetriever(opts ...RetrieverOption) (*StructuredRetriever, error) {
	o := options{
		maxRetries: defaultMaxRetries,
		minScore:   defaultMinScore,
	}
	for _, fn := range opts {
		fn(&o)
	}

	if o.generator == nil {
		return nil, core.NewError("structured.new", core.ErrInvalidInput, "generator is required", nil)
	}
	if o.executor == nil {
		return nil, core.NewError("structured.new", core.ErrInvalidInput, "executor is required", nil)
	}

	return &StructuredRetriever{opts: o}, nil
}

// Retrieve generates a structured query from the question, executes it, and
// returns the results as documents. If an evaluator is configured, low-scoring
// results trigger regeneration up to maxRetries times.
func (r *StructuredRetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
	cfg := retriever.ApplyOptions(opts...)

	if r.opts.hooks.BeforeRetrieve != nil {
		if err := r.opts.hooks.BeforeRetrieve(ctx, query); err != nil {
			return nil, err
		}
	}

	var (
		bestResult QueryResult
		bestScore  float64
		lastErr    error
		haveResult bool
	)

	attempts := r.opts.maxRetries + 1 // first attempt + retries
	for attempt := range attempts {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		generated, err := r.opts.generator.Generate(ctx, query, r.opts.schema)
		if err != nil {
			return nil, core.Errorf(core.ErrProviderDown, "structured.retrieve: generate (attempt %d): %w", attempt+1, err)
		}

		rows, err := r.opts.executor.Execute(ctx, generated)
		if err != nil {
			slog.WarnContext(ctx, "structured.retrieve: execute failed, retrying",
				"attempt", attempt+1,
				"error", err,
			)
			// Preserve any previously successful bestResult; only track last error
			// so we can report it if no successful attempt ever happens.
			lastErr = err
			continue
		}

		result := QueryResult{Query: generated, Results: rows}

		// If no evaluator, accept immediately.
		if r.opts.evaluator == nil {
			bestResult = result
			bestScore = 1.0
			haveResult = true
			break
		}

		score, err := r.opts.evaluator.Evaluate(ctx, query, rows)
		if err != nil {
			slog.WarnContext(ctx, "structured.retrieve: evaluate failed, accepting results",
				"attempt", attempt+1,
				"error", err,
			)
			bestResult = result
			bestScore = 0.5 // assume moderate relevance on evaluation failure
			haveResult = true
			break
		}

		if !haveResult || score > bestScore {
			bestResult = result
			bestScore = score
			haveResult = true
		}

		if score >= r.opts.minScore {
			break
		}

		slog.InfoContext(ctx, "structured.retrieve: score below threshold, retrying",
			"attempt", attempt+1,
			"score", score,
			"threshold", r.opts.minScore,
		)
	}

	if !haveResult {
		if lastErr != nil {
			return nil, core.Errorf(core.ErrProviderDown, "structured.retrieve: all attempts failed: %w", lastErr)
		}
		return nil, core.NewError(
			"structured.retrieve",
			core.ErrInvalidInput,
			"no results produced after retries",
			nil,
		)
	}

	docs := resultsToDocs(bestResult, bestScore, cfg.TopK)

	if r.opts.hooks.AfterRetrieve != nil {
		r.opts.hooks.AfterRetrieve(ctx, docs, nil)
	}

	return docs, nil
}

// resultsToDocs converts query results into schema.Document values.
func resultsToDocs(result QueryResult, score float64, topK int) []schema.Document {
	rows := result.Results
	if topK > 0 && len(rows) > topK {
		rows = rows[:topK]
	}

	docs := make([]schema.Document, 0, len(rows))
	for i, row := range rows {
		content, err := json.Marshal(row)
		if err != nil {
			content = []byte(fmt.Sprintf("%v", row))
		}

		docs = append(docs, schema.Document{
			ID:      fmt.Sprintf("structured-%d", i),
			Content: string(content),
			Score:   score,
			Metadata: map[string]any{
				"query":  result.Query,
				"source": "structured",
				"row":    i,
			},
		})
	}

	return docs
}
