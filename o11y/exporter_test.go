package o11y

import (
	"context"
	"errors"
	"testing"
	"time"
)

// mockExporter records calls and optionally returns an error.
type mockExporter struct {
	calls []LLMCallData
	err   error
}

func (m *mockExporter) ExportLLMCall(_ context.Context, data LLMCallData) error {
	m.calls = append(m.calls, data)
	return m.err
}

func TestTraceExporter(t *testing.T) {
	t.Run("mock exporter records call", func(t *testing.T) {
		exp := &mockExporter{}
		data := LLMCallData{
			Model:       "gpt-4o",
			Provider:    "openai",
			InputTokens: 100,
			OutputTokens: 50,
			Duration:    500 * time.Millisecond,
			Cost:        0.002,
			Messages: []map[string]any{
				{"role": "user", "content": "hello"},
			},
			Response: map[string]any{"content": "hi there"},
			Metadata: map[string]any{"trace_id": "abc123"},
		}

		err := exp.ExportLLMCall(context.Background(), data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(exp.calls) != 1 {
			t.Fatalf("expected 1 call, got %d", len(exp.calls))
		}
		if exp.calls[0].Model != "gpt-4o" {
			t.Errorf("expected model 'gpt-4o', got %q", exp.calls[0].Model)
		}
		if exp.calls[0].InputTokens != 100 {
			t.Errorf("expected 100 input tokens, got %d", exp.calls[0].InputTokens)
		}
	})

	t.Run("exporter error propagates", func(t *testing.T) {
		exp := &mockExporter{err: errors.New("export failed")}
		err := exp.ExportLLMCall(context.Background(), LLMCallData{})
		if err == nil {
			t.Fatal("expected error")
		}
		if err.Error() != "export failed" {
			t.Errorf("expected 'export failed', got %q", err.Error())
		}
	})
}

func TestMultiExporter(t *testing.T) {
	t.Run("fans out to all exporters", func(t *testing.T) {
		exp1 := &mockExporter{}
		exp2 := &mockExporter{}
		multi := NewMultiExporter(exp1, exp2)

		data := LLMCallData{Model: "claude-4", Provider: "anthropic"}
		err := multi.ExportLLMCall(context.Background(), data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(exp1.calls) != 1 {
			t.Errorf("exp1: expected 1 call, got %d", len(exp1.calls))
		}
		if len(exp2.calls) != 1 {
			t.Errorf("exp2: expected 1 call, got %d", len(exp2.calls))
		}
	})

	t.Run("returns first error but calls all", func(t *testing.T) {
		exp1 := &mockExporter{err: errors.New("first failed")}
		exp2 := &mockExporter{}
		exp3 := &mockExporter{err: errors.New("third failed")}
		multi := NewMultiExporter(exp1, exp2, exp3)

		err := multi.ExportLLMCall(context.Background(), LLMCallData{})
		if err == nil {
			t.Fatal("expected error")
		}
		if err.Error() != "first failed" {
			t.Errorf("expected 'first failed', got %q", err.Error())
		}
		// All exporters should have been called.
		if len(exp1.calls) != 1 {
			t.Error("exp1 should have been called")
		}
		if len(exp2.calls) != 1 {
			t.Error("exp2 should have been called")
		}
		if len(exp3.calls) != 1 {
			t.Error("exp3 should have been called")
		}
	})

	t.Run("empty multi exporter succeeds", func(t *testing.T) {
		multi := NewMultiExporter()
		err := multi.ExportLLMCall(context.Background(), LLMCallData{})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestLLMCallDataFields(t *testing.T) {
	data := LLMCallData{
		Model:        "gemini-2.5-pro",
		Provider:     "google",
		InputTokens:  200,
		OutputTokens: 100,
		Duration:     time.Second,
		Cost:         0.005,
		Error:        "rate limited",
		Messages:     []map[string]any{{"role": "system", "content": "you are helpful"}},
		Response:     map[string]any{"text": "response"},
		Metadata:     map[string]any{"session_id": "s123"},
	}

	if data.Model != "gemini-2.5-pro" {
		t.Errorf("unexpected model: %s", data.Model)
	}
	if data.Error != "rate limited" {
		t.Errorf("unexpected error: %s", data.Error)
	}
	if data.Duration != time.Second {
		t.Errorf("unexpected duration: %v", data.Duration)
	}
}
