package server

import (
	"context"
	"net/http"

	"github.com/lookatitude/beluga-ai/v2/agent"
	"github.com/lookatitude/beluga-ai/v2/o11y"
)

// WithTracing returns middleware that wraps a ServerAdapter with OTel spans
// following the GenAI semantic conventions. Each operation produces a span
// named "server.<method>" carrying a gen_ai.operation.name attribute. Errors
// are recorded on the span and the status is set to StatusError on failure.
//
// Enable tracing by composing with other middleware:
//
//	s = server.ApplyMiddleware(s, server.WithTracing())
func WithTracing() Middleware {
	return func(next ServerAdapter) ServerAdapter {
		return &tracedServer{next: next}
	}
}

// tracedServer wraps a ServerAdapter and emits a span around each operation.
type tracedServer struct {
	next ServerAdapter
}

func (s *tracedServer) RegisterAgent(path string, a agent.Agent) error {
	_, span := o11y.StartSpan(context.Background(), "server.register_agent", o11y.Attrs{
		o11y.AttrOperationName: "server.register_agent",
		"server.path":          path,
	})
	defer span.End()

	if err := s.next.RegisterAgent(path, a); err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return err
	}
	span.SetStatus(o11y.StatusOK, "")
	return nil
}

func (s *tracedServer) RegisterHandler(path string, handler http.Handler) error {
	_, span := o11y.StartSpan(context.Background(), "server.register_handler", o11y.Attrs{
		o11y.AttrOperationName: "server.register_handler",
		"server.path":          path,
	})
	defer span.End()

	if err := s.next.RegisterHandler(path, handler); err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return err
	}
	span.SetStatus(o11y.StatusOK, "")
	return nil
}

func (s *tracedServer) Serve(ctx context.Context, addr string) error {
	ctx, span := o11y.StartSpan(ctx, "server.serve", o11y.Attrs{
		o11y.AttrOperationName: "server.serve",
		"server.addr":          addr,
	})
	defer span.End()

	if err := s.next.Serve(ctx, addr); err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return err
	}
	span.SetStatus(o11y.StatusOK, "")
	return nil
}

func (s *tracedServer) Shutdown(ctx context.Context) error {
	ctx, span := o11y.StartSpan(ctx, "server.shutdown", o11y.Attrs{
		o11y.AttrOperationName: "server.shutdown",
	})
	defer span.End()

	if err := s.next.Shutdown(ctx); err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return err
	}
	span.SetStatus(o11y.StatusOK, "")
	return nil
}

// Ensure tracedServer implements ServerAdapter at compile time.
var _ ServerAdapter = (*tracedServer)(nil)
