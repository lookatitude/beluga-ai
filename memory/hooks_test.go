package memory

import (
	"context"
	"errors"
	"testing"

	"github.com/lookatitude/beluga-ai/schema"
)

func TestComposeHooks_Empty(t *testing.T) {
	// Composing no hooks should return a noop Hooks struct.
	composed := ComposeHooks()

	ctx := context.Background()
	input := &schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "input"}}}
	output := &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "output"}}}

	// Calling nil hooks should not panic.
	if composed.BeforeSave != nil {
		if err := composed.BeforeSave(ctx, input, output); err != nil {
			t.Errorf("BeforeSave returned error: %v", err)
		}
	}
	if composed.AfterSave != nil {
		composed.AfterSave(ctx, input, output, nil)
	}
	if composed.BeforeLoad != nil {
		if err := composed.BeforeLoad(ctx, "query"); err != nil {
			t.Errorf("BeforeLoad returned error: %v", err)
		}
	}
	if composed.AfterLoad != nil {
		composed.AfterLoad(ctx, "query", nil, nil)
	}
	if composed.BeforeSearch != nil {
		if err := composed.BeforeSearch(ctx, "query", 5); err != nil {
			t.Errorf("BeforeSearch returned error: %v", err)
		}
	}
	if composed.AfterSearch != nil {
		composed.AfterSearch(ctx, "query", 5, nil, nil)
	}
	if composed.BeforeClear != nil {
		if err := composed.BeforeClear(ctx); err != nil {
			t.Errorf("BeforeClear returned error: %v", err)
		}
	}
	if composed.AfterClear != nil {
		composed.AfterClear(ctx, nil)
	}
}

func TestComposeHooks_Single(t *testing.T) {
	var calls []string

	hooks := Hooks{
		BeforeSave: func(ctx context.Context, input, output schema.Message) error {
			calls = append(calls, "BeforeSave")
			return nil
		},
		AfterSave: func(ctx context.Context, input, output schema.Message, err error) {
			calls = append(calls, "AfterSave")
		},
		BeforeLoad: func(ctx context.Context, query string) error {
			calls = append(calls, "BeforeLoad")
			return nil
		},
		AfterLoad: func(ctx context.Context, query string, msgs []schema.Message, err error) {
			calls = append(calls, "AfterLoad")
		},
		BeforeSearch: func(ctx context.Context, query string, k int) error {
			calls = append(calls, "BeforeSearch")
			return nil
		},
		AfterSearch: func(ctx context.Context, query string, k int, docs []schema.Document, err error) {
			calls = append(calls, "AfterSearch")
		},
		BeforeClear: func(ctx context.Context) error {
			calls = append(calls, "BeforeClear")
			return nil
		},
		AfterClear: func(ctx context.Context, err error) {
			calls = append(calls, "AfterClear")
		},
	}

	composed := ComposeHooks(hooks)
	ctx := context.Background()
	input := &schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "input"}}}
	output := &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "output"}}}

	// Test each hook.
	composed.BeforeSave(ctx, input, output)
	composed.AfterSave(ctx, input, output, nil)
	composed.BeforeLoad(ctx, "query")
	composed.AfterLoad(ctx, "query", nil, nil)
	composed.BeforeSearch(ctx, "query", 5)
	composed.AfterSearch(ctx, "query", 5, nil, nil)
	composed.BeforeClear(ctx)
	composed.AfterClear(ctx, nil)

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

func TestComposeHooks_Multiple(t *testing.T) {
	var calls []string

	h1 := Hooks{
		BeforeSave: func(ctx context.Context, input, output schema.Message) error {
			calls = append(calls, "h1.BeforeSave")
			return nil
		},
		AfterSave: func(ctx context.Context, input, output schema.Message, err error) {
			calls = append(calls, "h1.AfterSave")
		},
	}

	h2 := Hooks{
		BeforeSave: func(ctx context.Context, input, output schema.Message) error {
			calls = append(calls, "h2.BeforeSave")
			return nil
		},
		AfterSave: func(ctx context.Context, input, output schema.Message, err error) {
			calls = append(calls, "h2.AfterSave")
		},
	}

	h3 := Hooks{
		BeforeSave: func(ctx context.Context, input, output schema.Message) error {
			calls = append(calls, "h3.BeforeSave")
			return nil
		},
		AfterSave: func(ctx context.Context, input, output schema.Message, err error) {
			calls = append(calls, "h3.AfterSave")
		},
	}

	composed := ComposeHooks(h1, h2, h3)
	ctx := context.Background()
	input := &schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "input"}}}
	output := &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "output"}}}

	composed.BeforeSave(ctx, input, output)
	composed.AfterSave(ctx, input, output, nil)

	expected := []string{
		"h1.BeforeSave", "h2.BeforeSave", "h3.BeforeSave",
		"h1.AfterSave", "h2.AfterSave", "h3.AfterSave",
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

func TestComposeHooks_BeforeError(t *testing.T) {
	var calls []string

	h1 := Hooks{
		BeforeSave: func(ctx context.Context, input, output schema.Message) error {
			calls = append(calls, "h1.BeforeSave")
			return nil
		},
	}

	h2 := Hooks{
		BeforeSave: func(ctx context.Context, input, output schema.Message) error {
			calls = append(calls, "h2.BeforeSave")
			return errors.New("h2 error")
		},
	}

	h3 := Hooks{
		BeforeSave: func(ctx context.Context, input, output schema.Message) error {
			calls = append(calls, "h3.BeforeSave")
			return nil
		},
	}

	composed := ComposeHooks(h1, h2, h3)
	ctx := context.Background()
	input := &schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "input"}}}
	output := &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "output"}}}

	err := composed.BeforeSave(ctx, input, output)
	if err == nil {
		t.Fatal("expected error from BeforeSave, got nil")
	}
	if err.Error() != "h2 error" {
		t.Errorf("expected 'h2 error', got %v", err)
	}

	// h3 should not be called after h2 returns error.
	expected := []string{"h1.BeforeSave", "h2.BeforeSave"}
	if len(calls) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(calls), calls)
	}

	for i, want := range expected {
		if calls[i] != want {
			t.Errorf("call %d: expected %q, got %q", i, want, calls[i])
		}
	}
}

func TestComposeHooks_OnError(t *testing.T) {
	originalErr := errors.New("original error")
	var calls []string

	h1 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			calls = append(calls, "h1.OnError")
			// Modify and return non-nil error - this short-circuits.
			if err == originalErr {
				return errors.New("modified by h1")
			}
			return err
		},
	}

	h2 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			calls = append(calls, "h2.OnError")
			// This should not be called because h1 short-circuits.
			return nil
		},
	}

	h3 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			calls = append(calls, "h3.OnError")
			return err
		},
	}

	composed := ComposeHooks(h1, h2, h3)
	ctx := context.Background()

	// h1 returns non-nil error, which short-circuits the chain.
	err := composed.OnError(ctx, originalErr)
	if err == nil {
		t.Fatal("expected modified error, got nil")
	}
	if err.Error() != "modified by h1" {
		t.Errorf("expected 'modified by h1', got %v", err)
	}

	expected := []string{"h1.OnError"}
	if len(calls) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(calls), calls)
	}
}

func TestComposeHooks_OnErrorReturnsOriginal(t *testing.T) {
	originalErr := errors.New("original error")
	var calls []string

	h1 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			calls = append(calls, "h1.OnError")
			return nil
		},
	}

	h2 := Hooks{
		OnError: func(ctx context.Context, err error) error {
			calls = append(calls, "h2.OnError")
			return nil
		},
	}

	composed := ComposeHooks(h1, h2)
	ctx := context.Background()

	// Both return nil, so composed should return the original error.
	err := composed.OnError(ctx, originalErr)
	if err != originalErr {
		t.Errorf("expected original error, got %v", err)
	}

	expected := []string{"h1.OnError", "h2.OnError"}
	if len(calls) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(calls), calls)
	}
}

func TestComposeHooks_PartialNilHooks(t *testing.T) {
	var calls []string

	h1 := Hooks{
		BeforeSave: func(ctx context.Context, input, output schema.Message) error {
			calls = append(calls, "h1.BeforeSave")
			return nil
		},
		// AfterSave is nil
	}

	h2 := Hooks{
		// BeforeSave is nil
		AfterSave: func(ctx context.Context, input, output schema.Message, err error) {
			calls = append(calls, "h2.AfterSave")
		},
	}

	composed := ComposeHooks(h1, h2)
	ctx := context.Background()
	input := &schema.HumanMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "input"}}}
	output := &schema.AIMessage{Parts: []schema.ContentPart{schema.TextPart{Text: "output"}}}

	composed.BeforeSave(ctx, input, output)
	composed.AfterSave(ctx, input, output, nil)

	expected := []string{"h1.BeforeSave", "h2.AfterSave"}
	if len(calls) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(calls), calls)
	}

	for i, want := range expected {
		if calls[i] != want {
			t.Errorf("call %d: expected %q, got %q", i, want, calls[i])
		}
	}
}
