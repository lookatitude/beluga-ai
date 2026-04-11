package state

import (
	"context"
	"iter"

	"github.com/lookatitude/beluga-ai/o11y"
)

// WithTracing returns middleware that wraps a Store with OTel spans following
// the GenAI semantic conventions. Each operation produces a span named
// "state.<op>" carrying a gen_ai.operation.name attribute. Errors are
// recorded on the span and the status is set to StatusError on failure.
//
// Enable tracing by composing with other middleware:
//
//	s = state.ApplyMiddleware(s, state.WithTracing(), state.WithHooks(h))
func WithTracing() Middleware {
	return func(next Store) Store {
		return &tracedStore{next: next}
	}
}

// tracedStore wraps a Store and emits a span around each operation.
type tracedStore struct {
	next Store
}

func (s *tracedStore) Get(ctx context.Context, key string) (any, error) {
	ctx, span := o11y.StartSpan(ctx, "state.get", o11y.Attrs{
		o11y.AttrOperationName: "state.get",
	})
	defer span.End()

	val, err := s.next.Get(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	span.SetStatus(o11y.StatusOK, "")
	return val, nil
}

func (s *tracedStore) Set(ctx context.Context, key string, value any) error {
	ctx, span := o11y.StartSpan(ctx, "state.set", o11y.Attrs{
		o11y.AttrOperationName: "state.set",
	})
	defer span.End()

	err := s.next.Set(ctx, key, value)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return err
	}
	span.SetStatus(o11y.StatusOK, "")
	return nil
}

func (s *tracedStore) Delete(ctx context.Context, key string) error {
	ctx, span := o11y.StartSpan(ctx, "state.delete", o11y.Attrs{
		o11y.AttrOperationName: "state.delete",
	})
	defer span.End()

	err := s.next.Delete(ctx, key)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return err
	}
	span.SetStatus(o11y.StatusOK, "")
	return nil
}

// Watch wraps the underlying Watch in a span that covers subscription
// establishment. The span is started, the inner Watch is invoked (eagerly
// subscribing), and the span is ended before the iterator is returned so
// that subscription-time attributes and errors are captured without
// spanning the (potentially long-lived) iteration itself.
func (s *tracedStore) Watch(ctx context.Context, key string) iter.Seq2[StateChange, error] {
	_, span := o11y.StartSpan(ctx, "state.watch", o11y.Attrs{
		o11y.AttrOperationName: "state.watch",
	})
	defer span.End()

	seq := s.next.Watch(ctx, key)
	span.SetStatus(o11y.StatusOK, "")
	return seq
}

func (s *tracedStore) Close() error {
	_, span := o11y.StartSpan(context.Background(), "state.close", o11y.Attrs{
		o11y.AttrOperationName: "state.close",
	})
	defer span.End()

	err := s.next.Close()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return err
	}
	span.SetStatus(o11y.StatusOK, "")
	return nil
}

// Ensure tracedStore implements Store at compile time.
var _ Store = (*tracedStore)(nil)
