package llm

import (
	"context"
	"errors"
	"iter"
	"sync/atomic"

	"github.com/lookatitude/beluga-ai/core"
	"github.com/lookatitude/beluga-ai/schema"
)

// ModelSelector selects which ChatModel to use for a given request.
type ModelSelector interface {
	// Select picks a model from the available models for the given messages.
	Select(ctx context.Context, models []ChatModel, msgs []schema.Message) (ChatModel, error)
}

// RouterOption configures a Router.
type RouterOption func(*Router)

// WithStrategy sets the routing strategy. Defaults to RoundRobin if unset.
func WithStrategy(s ModelSelector) RouterOption {
	return func(r *Router) {
		r.strategy = s
	}
}

// WithModels sets the pool of models the router can select from.
func WithModels(models ...ChatModel) RouterOption {
	return func(r *Router) {
		r.models = models
	}
}

// Router implements ChatModel by delegating to one of several backend models
// chosen by a pluggable ModelSelector. This allows transparent load balancing,
// failover, and cost optimization across multiple LLM providers.
type Router struct {
	models   []ChatModel
	strategy ModelSelector
	tools    []schema.ToolDefinition
}

// NewRouter creates a Router with the given options. If no strategy is set,
// RoundRobin is used by default.
func NewRouter(opts ...RouterOption) *Router {
	r := &Router{}
	for _, opt := range opts {
		opt(r)
	}
	if r.strategy == nil {
		r.strategy = &RoundRobin{}
	}
	return r
}

func (r *Router) selectModel(ctx context.Context, msgs []schema.Message) (ChatModel, error) {
	if len(r.models) == 0 {
		return nil, core.NewError("llm.router", core.ErrInvalidInput, "no models configured", nil)
	}
	return r.strategy.Select(ctx, r.models, msgs)
}

// Generate delegates to the model selected by the strategy.
func (r *Router) Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
	model, err := r.selectModel(ctx, msgs)
	if err != nil {
		return nil, err
	}
	if len(r.tools) > 0 {
		model = model.BindTools(r.tools)
	}
	return model.Generate(ctx, msgs, opts...)
}

// Stream delegates to the model selected by the strategy.
func (r *Router) Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	model, err := r.selectModel(ctx, msgs)
	if err != nil {
		return func(yield func(schema.StreamChunk, error) bool) {
			yield(schema.StreamChunk{}, err)
		}
	}
	if len(r.tools) > 0 {
		model = model.BindTools(r.tools)
	}
	return model.Stream(ctx, msgs, opts...)
}

// BindTools returns a new Router with the given tools applied to whichever
// model is selected.
func (r *Router) BindTools(tools []schema.ToolDefinition) ChatModel {
	clone := &Router{
		models:   r.models,
		strategy: r.strategy,
		tools:    tools,
	}
	return clone
}

// ModelID returns "router" since the actual model varies per request.
func (r *Router) ModelID() string { return "router" }

// RoundRobin selects models in round-robin order.
type RoundRobin struct {
	counter atomic.Uint64
}

// Select picks the next model in round-robin order.
func (rr *RoundRobin) Select(_ context.Context, models []ChatModel, _ []schema.Message) (ChatModel, error) {
	if len(models) == 0 {
		return nil, errors.New("llm: round-robin: no models")
	}
	idx := rr.counter.Add(1) - 1
	return models[idx%uint64(len(models))], nil
}

// FailoverChain tries models in order, falling back to the next on retryable
// errors. If all models fail, the last error is returned.
type FailoverChain struct{}

// Select tries each model in order, returning the first that does not produce
// a retryable error from a test Generate call. For the failover strategy,
// the selection itself just returns the first model; actual failover happens
// in Generate/Stream via the FailoverRouter wrapper.
func (fc *FailoverChain) Select(_ context.Context, models []ChatModel, _ []schema.Message) (ChatModel, error) {
	if len(models) == 0 {
		return nil, errors.New("llm: failover: no models")
	}
	// Return the first model; failover is handled by FailoverRouter.
	return models[0], nil
}

// FailoverRouter wraps multiple models and tries each in order, falling back
// on retryable errors. Unlike the basic Router+FailoverChain (which only
// selects a model), FailoverRouter actually retries across models.
type FailoverRouter struct {
	models []ChatModel
	tools  []schema.ToolDefinition
}

// NewFailoverRouter creates a FailoverRouter from the given models.
func NewFailoverRouter(models ...ChatModel) *FailoverRouter {
	return &FailoverRouter{models: models}
}

// Generate tries each model in order until one succeeds or a non-retryable
// error occurs.
func (fr *FailoverRouter) Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) {
	var lastErr error
	for _, model := range fr.models {
		if len(fr.tools) > 0 {
			model = model.BindTools(fr.tools)
		}
		resp, err := model.Generate(ctx, msgs, opts...)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if !core.IsRetryable(err) {
			return nil, err
		}
	}
	return nil, lastErr
}

// Stream tries each model in order. If the first chunk from a model is an
// error and it is retryable, the next model is tried.
func (fr *FailoverRouter) Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(yield func(schema.StreamChunk, error) bool) {
		for _, model := range fr.models {
			if len(fr.tools) > 0 {
				model = model.BindTools(fr.tools)
			}
			inner := model.Stream(ctx, msgs, opts...)
			failed := false
			for chunk, err := range inner {
				if err != nil && core.IsRetryable(err) {
					failed = true
					break
				}
				if !yield(chunk, err) {
					return
				}
				if err != nil {
					return
				}
			}
			if !failed {
				return
			}
		}
		yield(schema.StreamChunk{}, core.NewError("llm.failover", core.ErrProviderDown, "all models failed", nil))
	}
}

// BindTools returns a new FailoverRouter with the given tools.
func (fr *FailoverRouter) BindTools(tools []schema.ToolDefinition) ChatModel {
	return &FailoverRouter{models: fr.models, tools: tools}
}

// ModelID returns "failover-router".
func (fr *FailoverRouter) ModelID() string { return "failover-router" }
