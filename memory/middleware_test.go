package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
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
