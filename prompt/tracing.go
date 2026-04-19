package prompt

import (
	"context"

	"github.com/lookatitude/beluga-ai/v2/o11y"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// WithTracing returns middleware that wraps a PromptManager with OTel spans
// following the GenAI semantic conventions. Each operation produces a span
// named "prompt.<op>" carrying a gen_ai.operation.name attribute. Errors are
// recorded on the span and the status is set to StatusError on failure.
//
// Enable tracing by composing with other middleware:
//
//	mgr = prompt.ApplyMiddleware(mgr, prompt.WithTracing())
func WithTracing() Middleware {
	return func(next PromptManager) PromptManager {
		return &tracedManager{next: next}
	}
}

// tracedManager wraps a PromptManager and emits a span around each operation.
type tracedManager struct {
	next PromptManager
}

func (m *tracedManager) Get(name, version string) (*Template, error) {
	_, span := o11y.StartSpan(context.Background(), "prompt.get", o11y.Attrs{
		o11y.AttrOperationName: "prompt.get",
		"prompt.get.name":      name,
		"prompt.get.version":   version,
	})
	defer span.End()

	tmpl, err := m.next.Get(name, version)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	span.SetStatus(o11y.StatusOK, "")
	return tmpl, nil
}

func (m *tracedManager) Render(name string, vars map[string]any) ([]schema.Message, error) {
	_, span := o11y.StartSpan(context.Background(), "prompt.render", o11y.Attrs{
		o11y.AttrOperationName: "prompt.render",
		"prompt.render.name":   name,
	})
	defer span.End()

	msgs, err := m.next.Render(name, vars)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(o11y.StatusError, err.Error())
		return nil, err
	}
	span.SetAttributes(o11y.Attrs{"prompt.render.message_count": len(msgs)})
	span.SetStatus(o11y.StatusOK, "")
	return msgs, nil
}

func (m *tracedManager) List() []TemplateInfo {
	_, span := o11y.StartSpan(context.Background(), "prompt.list", o11y.Attrs{
		o11y.AttrOperationName: "prompt.list",
	})
	defer span.End()

	infos := m.next.List()
	span.SetAttributes(o11y.Attrs{"prompt.list.result_count": len(infos)})
	span.SetStatus(o11y.StatusOK, "")
	return infos
}

// Ensure tracedManager implements PromptManager at compile time.
var _ PromptManager = (*tracedManager)(nil)
