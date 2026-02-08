package memory

import (
	"context"

	"github.com/lookatitude/beluga-ai/schema"
)

// Middleware wraps a Memory to add cross-cutting behavior.
// Middlewares are composed via ApplyMiddleware and applied outside-in
// (the last middleware in the list is the outermost wrapper).
type Middleware func(Memory) Memory

// ApplyMiddleware wraps mem with the given middlewares in reverse order so
// that the first middleware in the list is the outermost (first to execute).
func ApplyMiddleware(mem Memory, mws ...Middleware) Memory {
	for i := len(mws) - 1; i >= 0; i-- {
		mem = mws[i](mem)
	}
	return mem
}

// WithHooks returns middleware that invokes the given Hooks around each
// Memory operation.
func WithHooks(hooks Hooks) Middleware {
	return func(next Memory) Memory {
		return &hookedMemory{next: next, hooks: hooks}
	}
}

// hookedMemory wraps a Memory with hook callbacks.
type hookedMemory struct {
	next  Memory
	hooks Hooks
}

func (m *hookedMemory) Save(ctx context.Context, input, output schema.Message) error {
	// BeforeSave
	if m.hooks.BeforeSave != nil {
		if err := m.hooks.BeforeSave(ctx, input, output); err != nil {
			return err
		}
	}

	// Execute
	err := m.next.Save(ctx, input, output)

	// OnError
	if err != nil && m.hooks.OnError != nil {
		err = m.hooks.OnError(ctx, err)
	}

	// AfterSave
	if m.hooks.AfterSave != nil {
		m.hooks.AfterSave(ctx, input, output, err)
	}

	return err
}

func (m *hookedMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	// BeforeLoad
	if m.hooks.BeforeLoad != nil {
		if err := m.hooks.BeforeLoad(ctx, query); err != nil {
			return nil, err
		}
	}

	// Execute
	msgs, err := m.next.Load(ctx, query)

	// OnError
	if err != nil && m.hooks.OnError != nil {
		err = m.hooks.OnError(ctx, err)
	}

	// AfterLoad
	if m.hooks.AfterLoad != nil {
		m.hooks.AfterLoad(ctx, query, msgs, err)
	}

	return msgs, err
}

func (m *hookedMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	// BeforeSearch
	if m.hooks.BeforeSearch != nil {
		if err := m.hooks.BeforeSearch(ctx, query, k); err != nil {
			return nil, err
		}
	}

	// Execute
	docs, err := m.next.Search(ctx, query, k)

	// OnError
	if err != nil && m.hooks.OnError != nil {
		err = m.hooks.OnError(ctx, err)
	}

	// AfterSearch
	if m.hooks.AfterSearch != nil {
		m.hooks.AfterSearch(ctx, query, k, docs, err)
	}

	return docs, err
}

func (m *hookedMemory) Clear(ctx context.Context) error {
	// BeforeClear
	if m.hooks.BeforeClear != nil {
		if err := m.hooks.BeforeClear(ctx); err != nil {
			return err
		}
	}

	// Execute
	err := m.next.Clear(ctx)

	// OnError
	if err != nil && m.hooks.OnError != nil {
		err = m.hooks.OnError(ctx, err)
	}

	// AfterClear
	if m.hooks.AfterClear != nil {
		m.hooks.AfterClear(ctx, err)
	}

	return err
}

// Ensure hookedMemory implements Memory interface at compile time.
var _ Memory = (*hookedMemory)(nil)
