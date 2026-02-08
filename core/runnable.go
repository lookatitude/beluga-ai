package core

import (
	"context"
	"iter"
	"sync"
)

// Runnable is the universal execution interface. Every component that can
// process input — LLMs, tools, agents, pipelines — implements Runnable.
//
//	result, err := r.Invoke(ctx, input)
//
//	for val, err := range r.Stream(ctx, input) { ... }
type Runnable interface {
	// Invoke executes the runnable synchronously and returns a single result.
	Invoke(ctx context.Context, input any, opts ...Option) (any, error)

	// Stream executes the runnable and returns an iterator of intermediate
	// results. Callers range over the returned iter.Seq2.
	Stream(ctx context.Context, input any, opts ...Option) iter.Seq2[any, error]
}

// Pipe composes two Runnables sequentially: the output of a becomes the input
// of b.
func Pipe(a, b Runnable) Runnable {
	return &pipeRunnable{a: a, b: b}
}

type pipeRunnable struct {
	a, b Runnable
}

func (p *pipeRunnable) Invoke(ctx context.Context, input any, opts ...Option) (any, error) {
	mid, err := p.a.Invoke(ctx, input, opts...)
	if err != nil {
		return nil, err
	}
	return p.b.Invoke(ctx, mid, opts...)
}

func (p *pipeRunnable) Stream(ctx context.Context, input any, opts ...Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		// Invoke a to get the intermediate result, then stream b.
		mid, err := p.a.Invoke(ctx, input, opts...)
		if err != nil {
			yield(nil, err)
			return
		}
		for val, sErr := range p.b.Stream(ctx, mid, opts...) {
			if !yield(val, sErr) {
				return
			}
			if sErr != nil {
				return
			}
		}
	}
}

// Parallel fans out to multiple Runnables concurrently and returns all results
// as a []any slice. If any Runnable returns an error, the first error is
// returned and remaining results may be incomplete.
func Parallel(runnables ...Runnable) Runnable {
	return &parallelRunnable{runnables: runnables}
}

type parallelRunnable struct {
	runnables []Runnable
}

func (p *parallelRunnable) Invoke(ctx context.Context, input any, opts ...Option) (any, error) {
	results := make([]any, len(p.runnables))
	errs := make([]error, len(p.runnables))
	var wg sync.WaitGroup
	wg.Add(len(p.runnables))

	for i, r := range p.runnables {
		go func(i int, r Runnable) {
			defer wg.Done()
			results[i], errs[i] = r.Invoke(ctx, input, opts...)
		}(i, r)
	}
	wg.Wait()

	for _, err := range errs {
		if err != nil {
			return results, err
		}
	}
	return results, nil
}

func (p *parallelRunnable) Stream(ctx context.Context, input any, opts ...Option) iter.Seq2[any, error] {
	return func(yield func(any, error) bool) {
		result, err := p.Invoke(ctx, input, opts...)
		yield(result, err)
	}
}
