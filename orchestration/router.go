package orchestration

import (
	"context"
	"fmt"
	"iter"

	"github.com/lookatitude/beluga-ai/core"
)

// ClassifierFunc classifies an input and returns a route name.
type ClassifierFunc func(ctx context.Context, input any) (string, error)

// Router dispatches input to one of several named routes based on a
// classifier function. If the classified route is not found, an optional
// fallback handler is used. If no fallback is set, an error is returned.
type Router struct {
	classifier ClassifierFunc
	routes     map[string]core.Runnable
	fallback   core.Runnable
}

// NewRouter creates a Router with the given classifier function.
func NewRouter(classifier ClassifierFunc) *Router {
	return &Router{
		classifier: classifier,
		routes:     make(map[string]core.Runnable),
	}
}

// AddRoute registers a named route handler. Returns the Router for chaining.
func (r *Router) AddRoute(name string, handler core.Runnable) *Router {
	r.routes[name] = handler
	return r
}

// SetFallback sets the fallback handler for unrecognized routes.
// Returns the Router for chaining.
func (r *Router) SetFallback(handler core.Runnable) *Router {
	r.fallback = handler
	return r
}

// Invoke classifies the input and dispatches to the matching route.
func (r *Router) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
	handler, err := r.resolve(ctx, input)
	if err != nil {
		return nil, err
	}
	result, err := handler.Invoke(ctx, input, opts...)
	if err != nil {
		return nil, fmt.Errorf("orchestration/router: %w", err)
	}
	return result, nil
}

// Stream classifies the input and streams from the matching route.
func (r *Router) Stream(ctx context.Context, input any, opts ...core.Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		handler, err := r.resolve(ctx, input)
		if err != nil {
			yield(nil, err)
			return
		}
		for val, sErr := range handler.Stream(ctx, input, opts...) {
			if !yield(val, sErr) {
				return
			}
			if sErr != nil {
				return
			}
		}
	}
}

// resolve finds the appropriate handler for the given input.
func (r *Router) resolve(ctx context.Context, input any) (core.Runnable, error) {
	route, err := r.classifier(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("orchestration/router: classifier: %w", err)
	}

	handler, ok := r.routes[route]
	if ok {
		return handler, nil
	}

	if r.fallback != nil {
		return r.fallback, nil
	}

	return nil, fmt.Errorf("orchestration/router: unknown route %q and no fallback set", route)
}
