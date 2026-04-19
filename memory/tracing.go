package memory

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/o11y"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// WithTracing returns middleware that wraps a Memory with OTel spans following
// the GenAI semantic conventions. Each operation produces a span named
// "memory.<op>" carrying a gen_ai.operation.name attribute. Errors are
// recorded on the span and the status is set to StatusError on failure.
//
// Enable tracing by composing with other middleware:
//
//	mem = memory.ApplyMiddleware(mem, memory.WithTracing(), memory.WithHooks(h))
func WithTracing() Middleware {
	return func(next Memory) Memory {
		return &tracedMemory{next: next}
	}
}

// tracedMemory wraps a Memory and emits a span around each operation.
type tracedMemory struct {
	next Memory
}

func (m *tracedMemory) Save(ctx context.Context, input, output schema.Message) error {
	ctx, span := o11y.StartSpan(ctx, "memory.save", o11y.Attrs{
		o11y.AttrOperationName: "memory.save",
	})
	defer span.End()

	err := m.next.Save(ctx, input, output)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return err
	}
	span.SetStatus(o11y.StatusOK, "")
	return nil
}

func (m *tracedMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	ctx, span := o11y.StartSpan(ctx, "memory.load", o11y.Attrs{
		o11y.AttrOperationName: "memory.load",
	})
	defer span.End()

	msgs, err := m.next.Load(ctx, query)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	span.SetAttributes(o11y.Attrs{"memory.load.result_count": len(msgs)})
	span.SetStatus(o11y.StatusOK, "")
	return msgs, nil
}

func (m *tracedMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	ctx, span := o11y.StartSpan(ctx, "memory.search", o11y.Attrs{
		o11y.AttrOperationName: "memory.search",
		"memory.search.k":      k,
	})
	defer span.End()

	docs, err := m.next.Search(ctx, query, k)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	span.SetAttributes(o11y.Attrs{"memory.search.result_count": len(docs)})
	span.SetStatus(o11y.StatusOK, "")
	return docs, nil
}

func (m *tracedMemory) Clear(ctx context.Context) error {
	ctx, span := o11y.StartSpan(ctx, "memory.clear", o11y.Attrs{
		o11y.AttrOperationName: "memory.clear",
	})
	defer span.End()

	err := m.next.Clear(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return err
	}
	span.SetStatus(o11y.StatusOK, "")
	return nil
}

// Ensure tracedMemory implements Memory at compile time.
var _ Memory = (*tracedMemory)(nil)
