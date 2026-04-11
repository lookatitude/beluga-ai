package agent

import (
	"context"
	"iter"

	"github.com/lookatitude/beluga-ai/o11y"
	"github.com/lookatitude/beluga-ai/tool"
)

// WithTracing returns middleware that wraps an Agent with OTel spans following
// the GenAI semantic conventions. Each runtime operation produces a span named
// "agent.<op>" carrying gen_ai.operation.name and gen_ai.agent.name attributes.
// Errors are recorded on the span and the status is set to StatusError on
// failure.
//
// Enable tracing by composing with other middleware:
//
//	a = agent.ApplyMiddleware(a, agent.WithTracing(), agent.WithLogging())
func WithTracing() Middleware {
	return func(next Agent) Agent {
		return &tracedAgent{next: next}
	}
}

// tracedAgent wraps an Agent and emits a span around each runtime operation.
type tracedAgent struct {
	next Agent
}

// ID returns the underlying agent's identifier.
func (a *tracedAgent) ID() string { return a.next.ID() }

// Persona returns the underlying agent's persona.
func (a *tracedAgent) Persona() Persona { return a.next.Persona() }

// Tools returns the underlying agent's tools.
func (a *tracedAgent) Tools() []tool.Tool { return a.next.Tools() }

// Children returns the underlying agent's child agents.
func (a *tracedAgent) Children() []Agent { return a.next.Children() }

// Invoke executes the agent synchronously, emitting a span around the call.
func (a *tracedAgent) Invoke(ctx context.Context, input string, opts ...Option) (string, error) {
	ctx, span := o11y.StartSpan(ctx, "agent.invoke", o11y.Attrs{
		o11y.AttrOperationName: "agent.invoke",
		o11y.AttrAgentName:     a.next.ID(),
	})
	defer span.End()

	result, err := a.next.Invoke(ctx, input, opts...)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return result, err
	}
	span.SetStatus(o11y.StatusOK, "")
	return result, nil
}

// Stream executes the agent and returns an iterator of events, emitting a
// span that tracks the lifetime of the stream and any errors it yields.
func (a *tracedAgent) Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error] {
	return func(yield func(Event, error) bool) {
		ctx, span := o11y.StartSpan(ctx, "agent.stream", o11y.Attrs{
			o11y.AttrOperationName: "agent.stream",
			o11y.AttrAgentName:     a.next.ID(),
		})
		defer span.End()

		var streamErr error
		for event, err := range a.next.Stream(ctx, input, opts...) {
			if err != nil {
				streamErr = err
			}
			if !yield(event, err) {
				return
			}
			if err != nil {
				break
			}
		}
		if streamErr != nil {
			span.RecordError(streamErr)
			span.SetStatus(o11y.StatusError, streamErr.Error())
			return
		}
		span.SetStatus(o11y.StatusOK, "")
	}
}

// Ensure tracedAgent implements Agent at compile time.
var _ Agent = (*tracedAgent)(nil)
