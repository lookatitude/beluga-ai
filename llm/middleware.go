package llm

import (
	"context"
	"iter"
	"log/slog"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
)

// Middleware wraps a ChatModel to add cross-cutting behaviour.
// Middlewares are composed via ApplyMiddleware and applied outside-in
// (the last middleware in the list is the outermost wrapper).
type Middleware func(ChatModel) ChatModel

// ApplyMiddleware wraps model with the given middlewares in reverse order so
// that the first middleware in the list is the outermost (first to execute).
func ApplyMiddleware(model ChatModel, mws ...Middleware) ChatModel {
	for i := len(mws) - 1; i >= 0; i-- {
		model = mws[i](model)
	}
	return model
}

// WithHooks returns middleware that invokes the given Hooks around
// Generate and Stream calls.
func WithHooks(hooks Hooks) Middleware {
	return func(next ChatModel) ChatModel {
		return &hookedModel{next: next, hooks: hooks}
	}
}

type hookedModel struct {
	next  ChatModel
	hooks Hooks
}

func (m *hookedModel) Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
	if m.hooks.BeforeGenerate != nil {
		if err := m.hooks.BeforeGenerate(ctx, msgs); err != nil {
			return nil, err
		}
	}

	resp, err := m.next.Generate(ctx, msgs, opts...)

	if err != nil && m.hooks.OnError != nil {
		err = m.hooks.OnError(ctx, err)
	}

	if resp != nil {
		for _, tc := range resp.ToolCalls {
			if m.hooks.OnToolCall != nil {
				m.hooks.OnToolCall(ctx, tc)
			}
		}
	}

	if m.hooks.AfterGenerate != nil {
		m.hooks.AfterGenerate(ctx, resp, err)
	}

	return resp, err
}

func (m *hookedModel) Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	if m.hooks.BeforeGenerate != nil {
		if err := m.hooks.BeforeGenerate(ctx, msgs); err != nil {
			return func(yield func(schema.StreamChunk, error) bool) {
				yield(schema.StreamChunk{}, err)
			}
		}
	}

	inner := m.next.Stream(ctx, msgs, opts...)
	return func(yield func(schema.StreamChunk, error) bool) {
		for chunk, err := range inner {
			if err != nil {
				if m.hooks.OnError != nil {
					err = m.hooks.OnError(ctx, err)
				}
				if err != nil {
					yield(schema.StreamChunk{}, err)
				}
				return
			}

			if m.hooks.OnStream != nil {
				m.hooks.OnStream(ctx, chunk)
			}

			for _, tc := range chunk.ToolCalls {
				if m.hooks.OnToolCall != nil {
					m.hooks.OnToolCall(ctx, tc)
				}
			}

			if !yield(chunk, nil) {
				return
			}
		}
	}
}

func (m *hookedModel) BindTools(tools []schema.ToolDefinition) ChatModel {
	return &hookedModel{next: m.next.BindTools(tools), hooks: m.hooks}
}

func (m *hookedModel) ModelID() string { return m.next.ModelID() }

// WithLogging returns middleware that logs Generate and Stream calls using
// the provided slog.Logger.
func WithLogging(logger *slog.Logger) Middleware {
	return func(next ChatModel) ChatModel {
		return &loggingModel{next: next, logger: logger}
	}
}

type loggingModel struct {
	next   ChatModel
	logger *slog.Logger
}

func (m *loggingModel) Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
	m.logger.InfoContext(ctx, "llm.generate.start",
		"model", m.next.ModelID(),
		"messages", len(msgs),
	)
	resp, err := m.next.Generate(ctx, msgs, opts...)
	if err != nil {
		m.logger.ErrorContext(ctx, "llm.generate.error",
			"model", m.next.ModelID(),
			"error", err,
		)
		return resp, err
	}
	m.logger.InfoContext(ctx, "llm.generate.done",
		"model", m.next.ModelID(),
		"input_tokens", resp.Usage.InputTokens,
		"output_tokens", resp.Usage.OutputTokens,
	)
	return resp, nil
}

func (m *loggingModel) Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	m.logger.InfoContext(ctx, "llm.stream.start",
		"model", m.next.ModelID(),
		"messages", len(msgs),
	)
	inner := m.next.Stream(ctx, msgs, opts...)
	return func(yield func(schema.StreamChunk, error) bool) {
		for chunk, err := range inner {
			if err != nil {
				m.logger.ErrorContext(ctx, "llm.stream.error",
					"model", m.next.ModelID(),
					"error", err,
				)
				yield(schema.StreamChunk{}, err)
				return
			}
			if !yield(chunk, nil) {
				return
			}
		}
		m.logger.InfoContext(ctx, "llm.stream.done",
			"model", m.next.ModelID(),
		)
	}
}

func (m *loggingModel) BindTools(tools []schema.ToolDefinition) ChatModel {
	return &loggingModel{next: m.next.BindTools(tools), logger: m.logger}
}

func (m *loggingModel) ModelID() string { return m.next.ModelID() }

// WithFallback returns middleware that falls back to an alternative ChatModel
// when the primary model returns a retryable error (as determined by
// core.IsRetryable).
func WithFallback(fallback ChatModel) Middleware {
	return func(next ChatModel) ChatModel {
		return &fallbackModel{primary: next, fallback: fallback}
	}
}

type fallbackModel struct {
	primary  ChatModel
	fallback ChatModel
}

func (m *fallbackModel) Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
	resp, err := m.primary.Generate(ctx, msgs, opts...)
	if err != nil && core.IsRetryable(err) {
		return m.fallback.Generate(ctx, msgs, opts...)
	}
	return resp, err
}

func (m *fallbackModel) Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	// Try the primary first; if the first chunk is an error, fall back.
	inner := m.primary.Stream(ctx, msgs, opts...)
	return func(yield func(schema.StreamChunk, error) bool) {
		first := true
		for chunk, err := range inner {
			if first && err != nil && core.IsRetryable(err) {
				// Fall back to the secondary model.
				for fbChunk, fbErr := range m.fallback.Stream(ctx, msgs, opts...) {
					if !yield(fbChunk, fbErr) {
						return
					}
					if fbErr != nil {
						return
					}
				}
				return
			}
			first = false
			if !yield(chunk, err) {
				return
			}
			if err != nil {
				return
			}
		}
	}
}

func (m *fallbackModel) BindTools(tools []schema.ToolDefinition) ChatModel {
	return &fallbackModel{
		primary:  m.primary.BindTools(tools),
		fallback: m.fallback.BindTools(tools),
	}
}

func (m *fallbackModel) ModelID() string { return m.primary.ModelID() }
