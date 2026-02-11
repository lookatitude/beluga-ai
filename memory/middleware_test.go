package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockMemory is a simple in-memory implementation for testing.
type mockMemory struct {
	saveErr   error
	loadErr   error
	searchErr error
	clearErr  error

	saveCalled   bool
	loadCalled   bool
	searchCalled bool
	clearCalled  bool
}

func (m *mockMemory) Save(ctx context.Context, input, output schema.Message) error {
	m.saveCalled = true
	return m.saveErr
}

func (m *mockMemory) Load(ctx context.Context, query string) ([]schema.Message, error) {
	m.loadCalled = true
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	return []schema.Message{
		&schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "loaded"}}},
	}, nil
}

func (m *mockMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	m.searchCalled = true
	if m.searchErr != nil {
		return nil, m.searchErr
	}
	return []schema.Document{{ID: "doc1", Content: "result"}}, nil
}

func (m *mockMemory) Clear(ctx context.Context) error {
	m.clearCalled = true
	return m.clearErr
}

var _ Memory = (*mockMemory)(nil)

func TestApplyMiddleware_None(t *testing.T) {
	mock := &mockMemory{}
	wrapped := ApplyMiddleware(mock)

	ctx := context.Background()
	input := &schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "input"}}}
	output := &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "output"}}}

	if err := wrapped.Save(ctx, input, output); err != nil {
		t.Errorf("Save returned error: %v", err)
	}
	if !mock.saveCalled {
		t.Error("Save not called on underlying memory")
	}
}

func TestApplyMiddleware_Single(t *testing.T) {
	mock := &mockMemory{}
	var mwCalled bool

	mw := func(next Memory) Memory {
		return &testMiddleware{next: next, onSave: func() { mwCalled = true }}
	}

	wrapped := ApplyMiddleware(mock, mw)

	ctx := context.Background()
	input := &schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "input"}}}
	output := &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "output"}}}

	if err := wrapped.Save(ctx, input, output); err != nil {
		t.Errorf("Save returned error: %v", err)
	}
	if !mwCalled {
		t.Error("middleware not called")
	}
	if !mock.saveCalled {
		t.Error("Save not called on underlying memory")
	}
}

func TestApplyMiddleware_Multiple(t *testing.T) {
	mock := &mockMemory{}
	var calls []string

	mw1 := func(next Memory) Memory {
		return &testMiddleware{
			next: next,
			onSave: func() { calls = append(calls, "mw1") },
		}
	}

	mw2 := func(next Memory) Memory {
		return &testMiddleware{
			next: next,
			onSave: func() { calls = append(calls, "mw2") },
		}
	}

	mw3 := func(next Memory) Memory {
		return &testMiddleware{
			next: next,
			onSave: func() { calls = append(calls, "mw3") },
		}
	}

	// Apply order: mw1, mw2, mw3 → execution order: mw1, mw2, mw3 (outside-in)
	wrapped := ApplyMiddleware(mock, mw1, mw2, mw3)

	ctx := context.Background()
	input := &schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "input"}}}
	output := &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "output"}}}

	if err := wrapped.Save(ctx, input, output); err != nil {
		t.Errorf("Save returned error: %v", err)
	}

	expected := []string{"mw1", "mw2", "mw3"}
	if len(calls) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(calls), calls)
	}

	for i, want := range expected {
		if calls[i] != want {
			t.Errorf("call %d: expected %q, got %q", i, want, calls[i])
		}
	}
}

type testMiddleware struct {
	next     Memory
	onSave   func()
	onLoad   func()
	onSearch func()
	onClear  func()
}

func (m *testMiddleware) Save(ctx context.Context, input, output schema.Message) error {
	if m.onSave != nil {
		m.onSave()
	}
	return m.next.Save(ctx, input, output)
}

func (m *testMiddleware) Load(ctx context.Context, query string) ([]schema.Message, error) {
	if m.onLoad != nil {
		m.onLoad()
	}
	return m.next.Load(ctx, query)
}

func (m *testMiddleware) Search(ctx context.Context, query string, k int) ([]schema.Document, error) {
	if m.onSearch != nil {
		m.onSearch()
	}
	return m.next.Search(ctx, query, k)
}

func (m *testMiddleware) Clear(ctx context.Context) error {
	if m.onClear != nil {
		m.onClear()
	}
	return m.next.Clear(ctx)
}

func TestWithHooks_BeforeSave(t *testing.T) {
	mock := &mockMemory{}
	var called bool

	hooks := Hooks{
		BeforeSave: func(ctx context.Context, input, output schema.Message) error {
			called = true
			return nil
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	input := &schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "input"}}}
	output := &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "output"}}}

	if err := wrapped.Save(ctx, input, output); err != nil {
		t.Errorf("Save returned error: %v", err)
	}
	if !called {
		t.Error("BeforeSave hook not called")
	}
	if !mock.saveCalled {
		t.Error("Save not called on underlying memory")
	}
}

func TestWithHooks_AfterSave(t *testing.T) {
	mock := &mockMemory{}
	var called bool

	hooks := Hooks{
		AfterSave: func(ctx context.Context, input, output schema.Message, err error) {
			called = true
			if err != nil {
				t.Errorf("AfterSave received error: %v", err)
			}
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	input := &schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "input"}}}
	output := &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "output"}}}

	if err := wrapped.Save(ctx, input, output); err != nil {
		t.Errorf("Save returned error: %v", err)
	}
	if !called {
		t.Error("AfterSave hook not called")
	}
}

func TestWithHooks_BeforeSaveError(t *testing.T) {
	mock := &mockMemory{}
	expectedErr := errors.New("before save error")

	hooks := Hooks{
		BeforeSave: func(ctx context.Context, input, output schema.Message) error {
			return expectedErr
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	input := &schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "input"}}}
	output := &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "output"}}}

	err := wrapped.Save(ctx, input, output)
	if err == nil {
		t.Fatal("expected error from Save, got nil")
	}
	if err != expectedErr {
		t.Errorf("expected %v, got %v", expectedErr, err)
	}
	if mock.saveCalled {
		t.Error("Save should not be called on underlying memory when BeforeSave returns error")
	}
}

func TestWithHooks_BeforeLoad(t *testing.T) {
	mock := &mockMemory{}
	var called bool

	hooks := Hooks{
		BeforeLoad: func(ctx context.Context, query string) error {
			called = true
			if query != "test query" {
				t.Errorf("expected query 'test query', got %q", query)
			}
			return nil
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	_, err := wrapped.Load(ctx, "test query")
	if err != nil {
		t.Errorf("Load returned error: %v", err)
	}
	if !called {
		t.Error("BeforeLoad hook not called")
	}
	if !mock.loadCalled {
		t.Error("Load not called on underlying memory")
	}
}

func TestWithHooks_AfterLoad(t *testing.T) {
	mock := &mockMemory{}
	var called bool

	hooks := Hooks{
		AfterLoad: func(ctx context.Context, query string, msgs []schema.Message, err error) {
			called = true
			if query != "test query" {
				t.Errorf("expected query 'test query', got %q", query)
			}
			if len(msgs) != 1 {
				t.Errorf("expected 1 message, got %d", len(msgs))
			}
			if err != nil {
				t.Errorf("AfterLoad received error: %v", err)
			}
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	_, err := wrapped.Load(ctx, "test query")
	if err != nil {
		t.Errorf("Load returned error: %v", err)
	}
	if !called {
		t.Error("AfterLoad hook not called")
	}
}

func TestWithHooks_BeforeSearch(t *testing.T) {
	mock := &mockMemory{}
	var called bool

	hooks := Hooks{
		BeforeSearch: func(ctx context.Context, query string, k int) error {
			called = true
			if query != "test query" {
				t.Errorf("expected query 'test query', got %q", query)
			}
			if k != 5 {
				t.Errorf("expected k=5, got %d", k)
			}
			return nil
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	_, err := wrapped.Search(ctx, "test query", 5)
	if err != nil {
		t.Errorf("Search returned error: %v", err)
	}
	if !called {
		t.Error("BeforeSearch hook not called")
	}
	if !mock.searchCalled {
		t.Error("Search not called on underlying memory")
	}
}

func TestWithHooks_AfterSearch(t *testing.T) {
	mock := &mockMemory{}
	var called bool

	hooks := Hooks{
		AfterSearch: func(ctx context.Context, query string, k int, docs []schema.Document, err error) {
			called = true
			if query != "test query" {
				t.Errorf("expected query 'test query', got %q", query)
			}
			if k != 5 {
				t.Errorf("expected k=5, got %d", k)
			}
			if len(docs) != 1 {
				t.Errorf("expected 1 document, got %d", len(docs))
			}
			if err != nil {
				t.Errorf("AfterSearch received error: %v", err)
			}
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	_, err := wrapped.Search(ctx, "test query", 5)
	if err != nil {
		t.Errorf("Search returned error: %v", err)
	}
	if !called {
		t.Error("AfterSearch hook not called")
	}
}

func TestWithHooks_BeforeClear(t *testing.T) {
	mock := &mockMemory{}
	var called bool

	hooks := Hooks{
		BeforeClear: func(ctx context.Context) error {
			called = true
			return nil
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	err := wrapped.Clear(ctx)
	if err != nil {
		t.Errorf("Clear returned error: %v", err)
	}
	if !called {
		t.Error("BeforeClear hook not called")
	}
	if !mock.clearCalled {
		t.Error("Clear not called on underlying memory")
	}
}

func TestWithHooks_AfterClear(t *testing.T) {
	mock := &mockMemory{}
	var called bool

	hooks := Hooks{
		AfterClear: func(ctx context.Context, err error) {
			called = true
			if err != nil {
				t.Errorf("AfterClear received error: %v", err)
			}
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	err := wrapped.Clear(ctx)
	if err != nil {
		t.Errorf("Clear returned error: %v", err)
	}
	if !called {
		t.Error("AfterClear hook not called")
	}
}

func TestWithHooks_OnError(t *testing.T) {
	originalErr := errors.New("save error")
	mock := &mockMemory{saveErr: originalErr}
	modifiedErr := errors.New("modified error")

	hooks := Hooks{
		OnError: func(ctx context.Context, err error) error {
			if err == originalErr {
				return modifiedErr
			}
			return err
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	input := &schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "input"}}}
	output := &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "output"}}}

	err := wrapped.Save(ctx, input, output)
	if err != modifiedErr {
		t.Errorf("expected modified error, got %v", err)
	}
}

func TestWithHooks_OnErrorSuppresses(t *testing.T) {
	originalErr := errors.New("save error")
	mock := &mockMemory{saveErr: originalErr}

	hooks := Hooks{
		OnError: func(ctx context.Context, err error) error {
			// Suppress the error by returning nil.
			return nil
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	input := &schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "input"}}}
	output := &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "output"}}}

	err := wrapped.Save(ctx, input, output)
	if err != nil {
		t.Errorf("expected nil error after suppression, got %v", err)
	}
}

func TestWithHooks_AllOperations(t *testing.T) {
	mock := &mockMemory{}
	var calls []string

	hooks := Hooks{
		BeforeSave:   func(ctx context.Context, input, output schema.Message) error { calls = append(calls, "BeforeSave"); return nil },
		AfterSave:    func(ctx context.Context, input, output schema.Message, err error) { calls = append(calls, "AfterSave") },
		BeforeLoad:   func(ctx context.Context, query string) error { calls = append(calls, "BeforeLoad"); return nil },
		AfterLoad:    func(ctx context.Context, query string, msgs []schema.Message, err error) { calls = append(calls, "AfterLoad") },
		BeforeSearch: func(ctx context.Context, query string, k int) error { calls = append(calls, "BeforeSearch"); return nil },
		AfterSearch:  func(ctx context.Context, query string, k int, docs []schema.Document, err error) { calls = append(calls, "AfterSearch") },
		BeforeClear:  func(ctx context.Context) error { calls = append(calls, "BeforeClear"); return nil },
		AfterClear:   func(ctx context.Context, err error) { calls = append(calls, "AfterClear") },
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	input := &schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "input"}}}
	output := &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "output"}}}

	wrapped.Save(ctx, input, output)
	wrapped.Load(ctx, "query")
	wrapped.Search(ctx, "query", 5)
	wrapped.Clear(ctx)

	expected := []string{
		"BeforeSave", "AfterSave",
		"BeforeLoad", "AfterLoad",
		"BeforeSearch", "AfterSearch",
		"BeforeClear", "AfterClear",
	}

	if len(calls) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(calls), calls)
	}

	for i, want := range expected {
		if calls[i] != want {
			t.Errorf("call %d: expected %q, got %q", i, want, calls[i])
		}
	}
}

func TestHookedMemory_InterfaceCompliance(t *testing.T) {
	var _ Memory = (*hookedMemory)(nil)
}

// TestHookedMemory_Load_OnError tests that Load's OnError hook is called
// and can modify the error. This covers lines 73-74 in middleware.go.
func TestHookedMemory_Load_OnError(t *testing.T) {
	originalErr := errors.New("load error")
	mock := &mockMemory{loadErr: originalErr}
	modifiedErr := errors.New("modified load error")

	var onErrorCalled bool
	hooks := Hooks{
		OnError: func(ctx context.Context, err error) error {
			onErrorCalled = true
			if err == originalErr {
				return modifiedErr
			}
			return err
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	msgs, err := wrapped.Load(ctx, "test query")
	require.Error(t, err)
	assert.Equal(t, modifiedErr, err)
	assert.Nil(t, msgs)
	assert.True(t, onErrorCalled)
}

// TestHookedMemory_Load_AfterLoad tests that AfterLoad hook is called
// with results. This covers lines 78-79 in middleware.go.
func TestHookedMemory_Load_AfterLoad(t *testing.T) {
	mock := &mockMemory{}

	var afterLoadCalled bool
	var receivedMsgs []schema.Message
	var receivedQuery string
	var receivedErr error

	hooks := Hooks{
		AfterLoad: func(ctx context.Context, query string, msgs []schema.Message, err error) {
			afterLoadCalled = true
			receivedQuery = query
			receivedMsgs = msgs
			receivedErr = err
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	msgs, err := wrapped.Load(ctx, "test query")
	require.NoError(t, err)
	assert.True(t, afterLoadCalled)
	assert.Equal(t, "test query", receivedQuery)
	assert.Len(t, receivedMsgs, 1)
	assert.Nil(t, receivedErr)
	assert.Equal(t, msgs, receivedMsgs)
}

// TestHookedMemory_Search_OnError tests that Search's OnError hook is called
// and can suppress the error. This covers lines 97-98 in middleware.go.
func TestHookedMemory_Search_OnError(t *testing.T) {
	originalErr := errors.New("search error")
	mock := &mockMemory{searchErr: originalErr}

	var onErrorCalled bool
	hooks := Hooks{
		OnError: func(ctx context.Context, err error) error {
			onErrorCalled = true
			// Suppress the error by returning nil
			return nil
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	docs, err := wrapped.Search(ctx, "test query", 5)
	require.NoError(t, err) // Error was suppressed
	assert.Nil(t, docs)
	assert.True(t, onErrorCalled)
}

// TestHookedMemory_Search_AfterSearch tests that AfterSearch hook is called
// with results. This covers lines 102-103 in middleware.go.
func TestHookedMemory_Search_AfterSearch(t *testing.T) {
	mock := &mockMemory{}

	var afterSearchCalled bool
	var receivedDocs []schema.Document
	var receivedQuery string
	var receivedK int
	var receivedErr error

	hooks := Hooks{
		AfterSearch: func(ctx context.Context, query string, k int, docs []schema.Document, err error) {
			afterSearchCalled = true
			receivedQuery = query
			receivedK = k
			receivedDocs = docs
			receivedErr = err
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	docs, err := wrapped.Search(ctx, "test query", 5)
	require.NoError(t, err)
	assert.True(t, afterSearchCalled)
	assert.Equal(t, "test query", receivedQuery)
	assert.Equal(t, 5, receivedK)
	assert.Len(t, receivedDocs, 1)
	assert.Nil(t, receivedErr)
	assert.Equal(t, docs, receivedDocs)
}

// TestHookedMemory_Clear_OnError tests that Clear's OnError hook is called
// and can modify the error. This covers lines 121-122 in middleware.go.
func TestHookedMemory_Clear_OnError(t *testing.T) {
	originalErr := errors.New("clear error")
	mock := &mockMemory{clearErr: originalErr}
	modifiedErr := errors.New("modified clear error")

	var onErrorCalled bool
	hooks := Hooks{
		OnError: func(ctx context.Context, err error) error {
			onErrorCalled = true
			if err == originalErr {
				return modifiedErr
			}
			return err
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	err := wrapped.Clear(ctx)
	require.Error(t, err)
	assert.Equal(t, modifiedErr, err)
	assert.True(t, onErrorCalled)
}

// TestHookedMemory_Clear_AfterClear tests that AfterClear hook is called.
// This covers lines 126-127 in middleware.go.
func TestHookedMemory_Clear_AfterClear(t *testing.T) {
	mock := &mockMemory{}

	var afterClearCalled bool
	var receivedErr error

	hooks := Hooks{
		AfterClear: func(ctx context.Context, err error) {
			afterClearCalled = true
			receivedErr = err
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	err := wrapped.Clear(ctx)
	require.NoError(t, err)
	assert.True(t, afterClearCalled)
	assert.Nil(t, receivedErr)
}

// TestHookedMemory_AfterLoad_WithError tests AfterLoad receives error
func TestHookedMemory_AfterLoad_WithError(t *testing.T) {
	loadErr := errors.New("load failed")
	mock := &mockMemory{loadErr: loadErr}

	var afterLoadCalled bool
	var receivedErr error

	hooks := Hooks{
		AfterLoad: func(ctx context.Context, query string, msgs []schema.Message, err error) {
			afterLoadCalled = true
			receivedErr = err
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	_, err := wrapped.Load(ctx, "query")
	require.Error(t, err)
	assert.True(t, afterLoadCalled)
	assert.Equal(t, loadErr, receivedErr)
}

// TestHookedMemory_AfterSearch_WithError tests AfterSearch receives error
func TestHookedMemory_AfterSearch_WithError(t *testing.T) {
	searchErr := errors.New("search failed")
	mock := &mockMemory{searchErr: searchErr}

	var afterSearchCalled bool
	var receivedErr error

	hooks := Hooks{
		AfterSearch: func(ctx context.Context, query string, k int, docs []schema.Document, err error) {
			afterSearchCalled = true
			receivedErr = err
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	_, err := wrapped.Search(ctx, "query", 5)
	require.Error(t, err)
	assert.True(t, afterSearchCalled)
	assert.Equal(t, searchErr, receivedErr)
}

// TestHookedMemory_AfterClear_WithError tests AfterClear receives error
func TestHookedMemory_AfterClear_WithError(t *testing.T) {
	clearErr := errors.New("clear failed")
	mock := &mockMemory{clearErr: clearErr}

	var afterClearCalled bool
	var receivedErr error

	hooks := Hooks{
		AfterClear: func(ctx context.Context, err error) {
			afterClearCalled = true
			receivedErr = err
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	err := wrapped.Clear(ctx)
	require.Error(t, err)
	assert.True(t, afterClearCalled)
	assert.Equal(t, clearErr, receivedErr)
}

// TestHookedMemory_BeforeLoad_Error tests that BeforeLoad error aborts Load.
// This covers lines 64-66 in middleware.go.
func TestHookedMemory_BeforeLoad_Error(t *testing.T) {
	mock := &mockMemory{}
	beforeErr := errors.New("before load error")

	hooks := Hooks{
		BeforeLoad: func(ctx context.Context, query string) error {
			return beforeErr
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	msgs, err := wrapped.Load(ctx, "query")
	require.Error(t, err)
	assert.Equal(t, beforeErr, err)
	assert.Nil(t, msgs)
	assert.False(t, mock.loadCalled) // Load should not be called
}

// TestHookedMemory_BeforeSearch_Error tests that BeforeSearch error aborts Search.
// This covers lines 88-90 in middleware.go.
func TestHookedMemory_BeforeSearch_Error(t *testing.T) {
	mock := &mockMemory{}
	beforeErr := errors.New("before search error")

	hooks := Hooks{
		BeforeSearch: func(ctx context.Context, query string, k int) error {
			return beforeErr
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	docs, err := wrapped.Search(ctx, "query", 5)
	require.Error(t, err)
	assert.Equal(t, beforeErr, err)
	assert.Nil(t, docs)
	assert.False(t, mock.searchCalled) // Search should not be called
}

// TestHookedMemory_BeforeClear_Error tests that BeforeClear error aborts Clear.
// This covers lines 112-114 in middleware.go.
func TestHookedMemory_BeforeClear_Error(t *testing.T) {
	mock := &mockMemory{}
	beforeErr := errors.New("before clear error")

	hooks := Hooks{
		BeforeClear: func(ctx context.Context) error {
			return beforeErr
		},
	}

	wrapped := WithHooks(hooks)(mock)

	ctx := context.Background()
	err := wrapped.Clear(ctx)
	require.Error(t, err)
	assert.Equal(t, beforeErr, err)
	assert.False(t, mock.clearCalled) // Clear should not be called
}
