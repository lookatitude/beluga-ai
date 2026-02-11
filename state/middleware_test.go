package state

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStore is a minimal in-memory Store for testing middleware.
type mockStore struct {
	data     map[string]any
	getCalls int
	setCalls int
	delCalls int
}

func newMockStore() *mockStore {
	return &mockStore{data: make(map[string]any)}
}

func (m *mockStore) Get(ctx context.Context, key string) (any, error) {
	m.getCalls++
	return m.data[key], nil
}

func (m *mockStore) Set(ctx context.Context, key string, value any) error {
	m.setCalls++
	m.data[key] = value
	return nil
}

func (m *mockStore) Delete(ctx context.Context, key string) error {
	m.delCalls++
	delete(m.data, key)
	return nil
}

func (m *mockStore) Watch(ctx context.Context, key string) (<-chan StateChange, error) {
	ch := make(chan StateChange, 1)
	return ch, nil
}

func (m *mockStore) Close() error { return nil }

func TestApplyMiddleware_Order(t *testing.T) {
	var order []string

	mw1 := func(next Store) Store {
		return &orderWrapper{next: next, name: "mw1", order: &order}
	}
	mw2 := func(next Store) Store {
		return &orderWrapper{next: next, name: "mw2", order: &order}
	}

	base := newMockStore()
	wrapped := ApplyMiddleware(base, mw1, mw2)

	ctx := context.Background()
	_, _ = wrapped.Get(ctx, "k")

	// mw1 is outermost (first to execute), mw2 is inner.
	assert.Equal(t, []string{"mw1", "mw2"}, order)
}

// orderWrapper records the order of middleware execution.
type orderWrapper struct {
	next  Store
	name  string
	order *[]string
}

func (w *orderWrapper) Get(ctx context.Context, key string) (any, error) {
	*w.order = append(*w.order, w.name)
	return w.next.Get(ctx, key)
}
func (w *orderWrapper) Set(ctx context.Context, key string, value any) error {
	*w.order = append(*w.order, w.name)
	return w.next.Set(ctx, key, value)
}
func (w *orderWrapper) Delete(ctx context.Context, key string) error {
	return w.next.Delete(ctx, key)
}
func (w *orderWrapper) Watch(ctx context.Context, key string) (<-chan StateChange, error) {
	return w.next.Watch(ctx, key)
}
func (w *orderWrapper) Close() error { return w.next.Close() }

func TestWithHooks_BeforeGetAborts(t *testing.T) {
	errAbort := errors.New("abort get")
	hooks := Hooks{
		BeforeGet: func(ctx context.Context, key string) error {
			return errAbort
		},
	}

	base := newMockStore()
	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	_, err := wrapped.Get(context.Background(), "k")
	require.ErrorIs(t, err, errAbort)
	assert.Equal(t, 0, base.getCalls, "underlying Get should not be called")
}

func TestWithHooks_BeforeSetAborts(t *testing.T) {
	errAbort := errors.New("abort set")
	hooks := Hooks{
		BeforeSet: func(ctx context.Context, key string, value any) error {
			return errAbort
		},
	}

	base := newMockStore()
	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	err := wrapped.Set(context.Background(), "k", "v")
	require.ErrorIs(t, err, errAbort)
	assert.Equal(t, 0, base.setCalls)
}

func TestWithHooks_OnDeleteAborts(t *testing.T) {
	errAbort := errors.New("abort delete")
	hooks := Hooks{
		OnDelete: func(ctx context.Context, key string) error {
			return errAbort
		},
	}

	base := newMockStore()
	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	err := wrapped.Delete(context.Background(), "k")
	require.ErrorIs(t, err, errAbort)
	assert.Equal(t, 0, base.delCalls)
}

func TestWithHooks_OnWatchAborts(t *testing.T) {
	errAbort := errors.New("abort watch")
	hooks := Hooks{
		OnWatch: func(ctx context.Context, key string) error {
			return errAbort
		},
	}

	base := newMockStore()
	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	ch, err := wrapped.Watch(context.Background(), "k")
	require.ErrorIs(t, err, errAbort)
	assert.Nil(t, ch)
}

func TestWithHooks_AfterGetCalled(t *testing.T) {
	var afterKey string
	var afterVal any
	hooks := Hooks{
		AfterGet: func(ctx context.Context, key string, value any, err error) {
			afterKey = key
			afterVal = value
		},
	}

	base := newMockStore()
	_ = base.Set(context.Background(), "k", 42)
	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	val, err := wrapped.Get(context.Background(), "k")
	require.NoError(t, err)
	assert.Equal(t, 42, val)
	assert.Equal(t, "k", afterKey)
	assert.Equal(t, 42, afterVal)
}

func TestWithHooks_AfterSetCalled(t *testing.T) {
	var afterKey string
	var afterErr error
	hooks := Hooks{
		AfterSet: func(ctx context.Context, key string, value any, err error) {
			afterKey = key
			afterErr = err
		},
	}

	base := newMockStore()
	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	err := wrapped.Set(context.Background(), "mykey", "val")
	require.NoError(t, err)
	assert.Equal(t, "mykey", afterKey)
	assert.NoError(t, afterErr)
}

func TestWithHooks_OnErrorSuppresses(t *testing.T) {
	hooks := Hooks{
		OnError: func(ctx context.Context, err error) error {
			return nil // suppress the error
		},
	}

	errStore := &errorStore{err: errors.New("fail")}
	wrapped := ApplyMiddleware(errStore, WithHooks(hooks))

	_, err := wrapped.Get(context.Background(), "k")
	assert.NoError(t, err, "OnError returning nil should suppress error")
}

func TestWithHooks_OnError_SetError(t *testing.T) {
	originalErr := errors.New("set failed")
	var capturedErr error
	hooks := Hooks{
		OnError: func(ctx context.Context, err error) error {
			capturedErr = err
			return nil // suppress the error
		},
	}

	errStore := &errorStore{err: originalErr}
	wrapped := ApplyMiddleware(errStore, WithHooks(hooks))

	err := wrapped.Set(context.Background(), "k", "v")
	assert.NoError(t, err, "OnError returning nil should suppress Set error")
	assert.ErrorIs(t, capturedErr, originalErr, "OnError should receive the original error")
}

func TestWithHooks_OnError_DeleteError(t *testing.T) {
	originalErr := errors.New("delete failed")
	var capturedErr error
	hooks := Hooks{
		OnError: func(ctx context.Context, err error) error {
			capturedErr = err
			return nil // suppress the error
		},
	}

	errStore := &errorStore{err: originalErr}
	wrapped := ApplyMiddleware(errStore, WithHooks(hooks))

	err := wrapped.Delete(context.Background(), "k")
	assert.NoError(t, err, "OnError returning nil should suppress Delete error")
	assert.ErrorIs(t, capturedErr, originalErr, "OnError should receive the original error")
}

func TestWithHooks_OnError_WatchError(t *testing.T) {
	originalErr := errors.New("watch failed")
	var capturedErr error
	hooks := Hooks{
		OnError: func(ctx context.Context, err error) error {
			capturedErr = err
			return nil // suppress the error
		},
	}

	errStore := &errorStore{err: originalErr}
	wrapped := ApplyMiddleware(errStore, WithHooks(hooks))

	ch, err := wrapped.Watch(context.Background(), "k")
	assert.NoError(t, err, "OnError returning nil should suppress Watch error")
	assert.Nil(t, ch, "channel should remain nil when Watch fails")
	assert.ErrorIs(t, capturedErr, originalErr, "OnError should receive the original error")
}

func TestWithHooks_ClosePassesThrough(t *testing.T) {
	hooks := Hooks{}
	base := newMockStore()
	wrapped := ApplyMiddleware(base, WithHooks(hooks))

	err := wrapped.Close()
	assert.NoError(t, err)
}

// errorStore always returns an error for Get.
type errorStore struct {
	err error
}

func (e *errorStore) Get(ctx context.Context, key string) (any, error) {
	return nil, e.err
}
func (e *errorStore) Set(ctx context.Context, key string, value any) error {
	return e.err
}
func (e *errorStore) Delete(ctx context.Context, key string) error {
	return e.err
}
func (e *errorStore) Watch(ctx context.Context, key string) (<-chan StateChange, error) {
	return nil, e.err
}
func (e *errorStore) Close() error { return nil }
