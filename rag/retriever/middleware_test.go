package retriever

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/lookatitude/beluga-ai/schema"
)

// --- Tests for Middleware ---

func TestApplyMiddleware_SingleMiddleware(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("doc1", "doc2")}

	// Middleware that adds a prefix to doc IDs
	addPrefix := func(next Retriever) Retriever {
		return &prefixRetriever{next: next, prefix: "prefixed_"}
	}

	r := ApplyMiddleware(inner, addPrefix)
	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 2)
	assert.Equal(t, "prefixed_doc1", docs[0].ID)
	assert.Equal(t, "prefixed_doc2", docs[1].ID)
}

func TestApplyMiddleware_MultipleMiddleware(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("doc1")}

	var order []string

	mw1 := func(next Retriever) Retriever {
		return &orderTrackerRetriever{
			next:  next,
			name:  "mw1",
			order: &order,
		}
	}

	mw2 := func(next Retriever) Retriever {
		return &orderTrackerRetriever{
			next:  next,
			name:  "mw2",
			order: &order,
		}
	}

	mw3 := func(next Retriever) Retriever {
		return &orderTrackerRetriever{
			next:  next,
			name:  "mw3",
			order: &order,
		}
	}

	r := ApplyMiddleware(inner, mw1, mw2, mw3)
	_, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)

	// Middleware should be applied outside-in: mw1 is outermost
	require.Len(t, order, 3)
	assert.Equal(t, "mw1", order[0])
	assert.Equal(t, "mw2", order[1])
	assert.Equal(t, "mw3", order[2])
}

func TestApplyMiddleware_NoMiddleware(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("doc1")}

	r := ApplyMiddleware(inner)
	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.Equal(t, "doc1", docs[0].ID)
}

func TestWithHooks_BeforeRetrieve(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("doc1")}

	var called bool
	var capturedQuery string

	hooks := Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			called = true
			capturedQuery = query
			return nil
		},
	}

	r := ApplyMiddleware(inner, WithHooks(hooks))
	docs, err := r.Retrieve(context.Background(), "test query")
	require.NoError(t, err)
	assert.Len(t, docs, 1)

	assert.True(t, called, "BeforeRetrieve should be called")
	assert.Equal(t, "test query", capturedQuery)
}

func TestWithHooks_AfterRetrieve(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("doc1", "doc2")}

	var called bool
	var capturedDocs []schema.Document
	var capturedErr error

	hooks := Hooks{
		AfterRetrieve: func(ctx context.Context, docs []schema.Document, err error) {
			called = true
			capturedDocs = docs
			capturedErr = err
		},
	}

	r := ApplyMiddleware(inner, WithHooks(hooks))
	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 2)

	assert.True(t, called, "AfterRetrieve should be called")
	assert.Len(t, capturedDocs, 2)
	assert.NoError(t, capturedErr)
}

func TestWithHooks_BeforeRetrieveError(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("doc1")}

	hookErr := errors.New("hook rejected")
	hooks := Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			return hookErr
		},
	}

	r := ApplyMiddleware(inner, WithHooks(hooks))
	_, err := r.Retrieve(context.Background(), "query")
	require.Error(t, err)
	assert.Equal(t, hookErr, err)
}

func TestWithHooks_AfterRetrieveWithError(t *testing.T) {
	innerErr := errors.New("retrieval failed")
	inner := &mockRetriever{err: innerErr}

	var called bool
	var capturedErr error

	hooks := Hooks{
		AfterRetrieve: func(ctx context.Context, docs []schema.Document, err error) {
			called = true
			capturedErr = err
		},
	}

	r := ApplyMiddleware(inner, WithHooks(hooks))
	_, err := r.Retrieve(context.Background(), "query")
	require.Error(t, err)

	assert.True(t, called, "AfterRetrieve should be called even on error")
	assert.Equal(t, innerErr, capturedErr)
}

func TestWithHooks_NilHooks(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("doc1")}

	hooks := Hooks{
		// All hooks are nil
	}

	r := ApplyMiddleware(inner, WithHooks(hooks))
	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 1)
}

func TestWithHooks_CombinedWithOtherMiddleware(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("doc1")}

	var hookCalled bool
	hooks := Hooks{
		BeforeRetrieve: func(ctx context.Context, query string) error {
			hookCalled = true
			return nil
		},
	}

	addPrefix := func(next Retriever) Retriever {
		return &prefixRetriever{next: next, prefix: "test_"}
	}

	r := ApplyMiddleware(inner, WithHooks(hooks), addPrefix)
	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	assert.True(t, hookCalled)
	assert.Equal(t, "test_doc1", docs[0].ID)
}

// --- Helper types for middleware tests ---

// prefixRetriever adds a prefix to all doc IDs
type prefixRetriever struct {
	next   Retriever
	prefix string
}

func (p *prefixRetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	docs, err := p.next.Retrieve(ctx, query, opts...)
	if err != nil {
		return nil, err
	}

	result := make([]schema.Document, len(docs))
	for i, doc := range docs {
		doc.ID = p.prefix + doc.ID
		result[i] = doc
	}
	return result, nil
}

// orderTrackerRetriever tracks the order middleware is called
type orderTrackerRetriever struct {
	next  Retriever
	name  string
	order *[]string
}

func (o *orderTrackerRetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	*o.order = append(*o.order, o.name)
	return o.next.Retrieve(ctx, query, opts...)
}

// errorInjectingRetriever can inject errors
type errorInjectingRetriever struct {
	next Retriever
	err  error
}

func (e *errorInjectingRetriever) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	if e.err != nil {
		return nil, e.err
	}
	return e.next.Retrieve(ctx, query, opts...)
}

func TestMiddleware_ErrorInjection(t *testing.T) {
	inner := &mockRetriever{docs: makeDocs("doc1")}
	injectedErr := errors.New("injected error")

	injectError := func(next Retriever) Retriever {
		return &errorInjectingRetriever{next: next, err: injectedErr}
	}

	r := ApplyMiddleware(inner, injectError)
	_, err := r.Retrieve(context.Background(), "query")
	require.Error(t, err)
	assert.Equal(t, injectedErr, err)
}

func TestMiddleware_Chaining(t *testing.T) {
	// Use a mock that returns copies to avoid mutation issues
	inner := retrieverFunc(func(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
		return []schema.Document{{ID: "doc1", Score: 1.0}}, nil
	})

	// Middleware 1: Multiplies scores by 2
	multiplier := func(next Retriever) Retriever {
		return retrieverFunc(func(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
			docs, err := next.Retrieve(ctx, query, opts...)
			if err != nil {
				return nil, err
			}
			result := make([]schema.Document, len(docs))
			for i := range docs {
				result[i] = docs[i]
				result[i].Score *= 2
			}
			return result, nil
		})
	}

	// Middleware 2: Adds 10 to scores
	adder := func(next Retriever) Retriever {
		return retrieverFunc(func(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
			docs, err := next.Retrieve(ctx, query, opts...)
			if err != nil {
				return nil, err
			}
			result := make([]schema.Document, len(docs))
			for i := range docs {
				result[i] = docs[i]
				result[i].Score += 10
			}
			return result, nil
		})
	}

	// Apply: multiplier then adder
	// Middleware are applied in reverse: adder wraps (multiplier wraps base)
	// But execution is INSIDE-OUT: base (1.0) -> adder adds 10 (11.0) -> multiplier *2 (22.0)
	// Actually, ApplyMiddleware applies RIGHT to LEFT:
	// - First: multiplier wraps base
	// - Then: adder wraps (multiplier wrapping base)
	// So call chain is: adder(multiplier(base))
	// Execution: base returns 1.0 -> adder adds 10 = 11.0 -> multiplier *2 = 22.0
	r := ApplyMiddleware(inner, multiplier, adder)
	docs, err := r.Retrieve(context.Background(), "query")
	require.NoError(t, err)
	assert.Len(t, docs, 1)
	// First middleware in list (multiplier) is INNER, last (adder) is OUTER
	// But when calling: outer calls inner, so execution is LIFO
	// adder(multiplier(base)): base->adder->multiplier: 1.0 + 10 = 11.0, 11.0 * 2 = 22.0
	assert.Equal(t, 22.0, docs[0].Score)
}

// retrieverFunc is a function adapter for Retriever
type retrieverFunc func(ctx context.Context, query string, opts ...Option) ([]schema.Document, error)

func (f retrieverFunc) Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
	return f(ctx, query, opts...)
}

func TestMiddleware_ContextPropagation(t *testing.T) {
	type ctxKey string
	const key ctxKey = "test-key"

	inner := &mockRetriever{docs: makeDocs("doc1")}

	var capturedValue string
	captureContext := func(next Retriever) Retriever {
		return retrieverFunc(func(ctx context.Context, query string, opts ...Option) ([]schema.Document, error) {
			if v := ctx.Value(key); v != nil {
				capturedValue = v.(string)
			}
			return next.Retrieve(ctx, query, opts...)
		})
	}

	r := ApplyMiddleware(inner, captureContext)

	ctx := context.WithValue(context.Background(), key, "test-value")
	_, err := r.Retrieve(ctx, "query")
	require.NoError(t, err)
	assert.Equal(t, "test-value", capturedValue)
}

// Compile-time interface checks
var (
	_ Retriever = (*prefixRetriever)(nil)
	_ Retriever = (*orderTrackerRetriever)(nil)
	_ Retriever = (*errorInjectingRetriever)(nil)
	_ Retriever = (retrieverFunc)(nil)
)
