package mock

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/eval"
	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

func TestNew_DefaultsAndModelID(t *testing.T) {
	t.Parallel()

	t.Run("empty cfg uses defaults", func(t *testing.T) {
		t.Parallel()
		m, err := New(config.ProviderConfig{})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if m.ModelID() != defaultModelID {
			t.Fatalf("ModelID=%q want %q", m.ModelID(), defaultModelID)
		}
	})

	t.Run("cfg.Model overrides default", func(t *testing.T) {
		t.Parallel()
		m, err := New(config.ProviderConfig{Model: "gpt-test"})
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		if m.ModelID() != "gpt-test" {
			t.Fatalf("ModelID=%q want gpt-test", m.ModelID())
		}
	})

	t.Run("WithModelID overrides cfg.Model", func(t *testing.T) {
		t.Parallel()
		m, err := New(config.ProviderConfig{Model: "gpt-test"}, WithModelID("override"))
		if err != nil {
			t.Fatalf("New: %v", err)
		}
		// cfg.Model applies after the options loop so cfg wins; document that
		// precedence here via assertion.
		if m.ModelID() != "gpt-test" {
			t.Fatalf("ModelID=%q want gpt-test (cfg.Model wins)", m.ModelID())
		}
	})
}

func TestGenerate_FixtureQueue(t *testing.T) {
	t.Parallel()

	fixtures := []Fixture{
		{Content: "first"},
		{Content: "second", ToolCalls: []schema.ToolCall{{Name: "lookup", Arguments: "{}"}}},
	}
	m, err := New(config.ProviderConfig{}, WithFixtures(fixtures))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	cases := []struct {
		name           string
		wantText       string
		wantToolCount  int
		wantToolCallID string
	}{
		{name: "first response text only", wantText: "first", wantToolCount: 0},
		{name: "second response has tool call", wantText: "second", wantToolCount: 1, wantToolCallID: "call_1_0"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			msg, err := m.Generate(ctx, nil)
			if err != nil {
				t.Fatalf("Generate: %v", err)
			}
			if msg.Text() != tc.wantText {
				t.Fatalf("Text=%q want %q", msg.Text(), tc.wantText)
			}
			if len(msg.ToolCalls) != tc.wantToolCount {
				t.Fatalf("ToolCalls=%d want %d", len(msg.ToolCalls), tc.wantToolCount)
			}
			if tc.wantToolCount > 0 && msg.ToolCalls[0].ID != tc.wantToolCallID {
				t.Fatalf("ToolCalls[0].ID=%q want %q", msg.ToolCalls[0].ID, tc.wantToolCallID)
			}
		})
	}
}

func TestGenerate_FallbackOnExhaustion(t *testing.T) {
	t.Parallel()

	m, err := New(config.ProviderConfig{}, WithFallback(Fixture{Content: "done"}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	msg, err := m.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if msg.Text() != "done" {
		t.Fatalf("Text=%q want done", msg.Text())
	}
	if len(msg.ToolCalls) != 0 {
		t.Fatalf("ToolCalls=%d want 0 (fallback must be a final answer)", len(msg.ToolCalls))
	}
}

func TestGenerate_FixtureError(t *testing.T) {
	t.Parallel()

	m, err := New(config.ProviderConfig{}, WithFixtures([]Fixture{{Error: "boom"}}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	_, err = m.Generate(context.Background(), nil)
	if err == nil {
		t.Fatalf("Generate: want error")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) {
		t.Fatalf("error type = %T, want *core.Error", err)
	}
	if coreErr.Code != core.ErrInvalidInput {
		t.Fatalf("code = %q want %q", coreErr.Code, core.ErrInvalidInput)
	}
}

func TestGenerate_ContextCancelled(t *testing.T) {
	t.Parallel()

	m, err := New(config.ProviderConfig{}, WithFixtures([]Fixture{{Content: "ignored"}}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m.Generate(ctx, nil); err == nil {
		t.Fatalf("Generate on cancelled ctx: want error")
	}
	if m.Calls() != 0 {
		t.Fatalf("Calls=%d want 0 (cancelled ctx must not advance queue)", m.Calls())
	}
}

func TestStream_EmitsSingleChunkAndFinish(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		fixture    Fixture
		wantDelta  string
		wantFinish string
		wantCalls  int
	}{
		{
			name:       "text content stops",
			fixture:    Fixture{Content: "hello"},
			wantDelta:  "hello",
			wantFinish: "stop",
		},
		{
			name:       "tool call finish",
			fixture:    Fixture{ToolCalls: []schema.ToolCall{{Name: "t"}}},
			wantFinish: "tool_calls",
			wantCalls:  1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m, err := New(config.ProviderConfig{}, WithFixtures([]Fixture{tc.fixture}))
			if err != nil {
				t.Fatalf("New: %v", err)
			}
			var got []schema.StreamChunk
			for chunk, err := range m.Stream(context.Background(), nil) {
				if err != nil {
					t.Fatalf("Stream: %v", err)
				}
				got = append(got, chunk)
			}
			if len(got) != 1 {
				t.Fatalf("chunks=%d want 1", len(got))
			}
			if got[0].Delta != tc.wantDelta {
				t.Fatalf("Delta=%q want %q", got[0].Delta, tc.wantDelta)
			}
			if got[0].FinishReason != tc.wantFinish {
				t.Fatalf("FinishReason=%q want %q", got[0].FinishReason, tc.wantFinish)
			}
			if len(got[0].ToolCalls) != tc.wantCalls {
				t.Fatalf("ToolCalls=%d want %d", len(got[0].ToolCalls), tc.wantCalls)
			}
		})
	}
}

func TestBindTools_SharesFixtureQueue(t *testing.T) {
	t.Parallel()

	m, err := New(config.ProviderConfig{}, WithFixtures([]Fixture{
		{Content: "one"},
		{Content: "two"},
	}))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	tools := []schema.ToolDefinition{{Name: "search"}}
	bound := m.BindTools(tools)

	if got, ok := bound.(*ChatModel); !ok || got.BoundTools()[0].Name != "search" {
		t.Fatalf("BoundTools not recorded: ok=%v tools=%v", ok, got)
	}

	msg1, err := m.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("Generate parent: %v", err)
	}
	msg2, err := bound.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("Generate bound: %v", err)
	}
	if msg1.Text() != "one" || msg2.Text() != "two" {
		t.Fatalf("shared queue broken: parent=%q bound=%q", msg1.Text(), msg2.Text())
	}
	if m.Calls() != 2 {
		t.Fatalf("Calls=%d want 2 (shared state)", m.Calls())
	}
}

func TestBindTools_SatisfiesChatModelInterface(t *testing.T) {
	t.Parallel()

	m, err := New(config.ProviderConfig{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	var _ llm.ChatModel = m.BindTools(nil)
}

func TestRegistryFactory(t *testing.T) {
	t.Parallel()

	got, err := llm.New("mock", config.ProviderConfig{Model: "from-registry"})
	if err != nil {
		t.Fatalf("llm.New(mock): %v", err)
	}
	if got.ModelID() != "from-registry" {
		t.Fatalf("ModelID=%q want from-registry", got.ModelID())
	}
}

func TestFixturesFromJSONFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "fixtures.json")
	blob, err := json.Marshal([]Fixture{{Content: "from-file"}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		t.Fatalf("write fixtures: %v", err)
	}

	m, err := New(config.ProviderConfig{
		Options: map[string]any{"fixtures_file": path},
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	msg, err := m.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if msg.Text() != "from-file" {
		t.Fatalf("Text=%q want from-file", msg.Text())
	}
}

func TestFixturesFromEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "fixtures.json")
	blob, err := json.Marshal([]Fixture{{Content: "from-env"}})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(path, blob, 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv(envFixturesPath, path)

	m, err := New(config.ProviderConfig{})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	msg, err := m.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if msg.Text() != "from-env" {
		t.Fatalf("Text=%q want from-env", msg.Text())
	}
}

func TestFixturesFromOptionsInline(t *testing.T) {
	t.Parallel()

	// Simulate JSON-decoded cfg.Options with a []any payload.
	raw := []any{
		map[string]any{"content": "inline-0"},
		map[string]any{"content": "inline-1"},
	}
	m, err := New(config.ProviderConfig{Options: map[string]any{"fixtures": raw}})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	got, err := m.Generate(context.Background(), nil)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	if got.Text() != "inline-0" {
		t.Fatalf("Text=%q want inline-0", got.Text())
	}
}

func TestFixturesFile_BadPathReturnsError(t *testing.T) {
	t.Parallel()

	_, err := New(config.ProviderConfig{
		Options: map[string]any{"fixtures_file": "/does/not/exist.json"},
	})
	if err == nil {
		t.Fatalf("New: want error for missing fixtures file")
	}
	var coreErr *core.Error
	if !errors.As(err, &coreErr) || coreErr.Code != core.ErrInvalidInput {
		t.Fatalf("error = %v, want core.ErrInvalidInput", err)
	}
}

func TestFixturesFromTurns_EmptyAndNil(t *testing.T) {
	t.Parallel()

	if got := FixturesFromTurns(nil); got != nil {
		t.Fatalf("FixturesFromTurns(nil)=%v want nil", got)
	}
	if got := FixturesFromTurns([]eval.Turn{}); got != nil {
		t.Fatalf("FixturesFromTurns(empty)=%v want nil", got)
	}
}

func TestFixturesFromTurns_SkipsNonAssistantTurns(t *testing.T) {
	t.Parallel()

	turns := []eval.Turn{
		{Role: "user", Content: "hi"},
		{Role: "system", Content: "you are a bot"},
		{Role: "tool", Content: `{"result":"ok"}`},
	}
	if got := FixturesFromTurns(turns); len(got) != 0 {
		t.Fatalf("FixturesFromTurns(non-assistant-only)=%v want empty", got)
	}
}

func TestFixturesFromTurns_ToolCallThenText(t *testing.T) {
	t.Parallel()

	turns := []eval.Turn{
		{Role: "user", Content: "what's the weather?"},
		{Role: "assistant", ToolCalls: []schema.ToolCall{
			{ID: "call_1", Name: "get_weather", Arguments: `{"city":"Lisbon"}`},
		}},
		{Role: "tool", Content: `{"temp":"22C"}`},
		{Role: "assistant", Content: "It's 22°C in Lisbon."},
	}

	got := FixturesFromTurns(turns)
	if len(got) != 2 {
		t.Fatalf("FixturesFromTurns: got %d fixtures, want 2 (one per assistant turn)", len(got))
	}
	if len(got[0].ToolCalls) != 1 {
		t.Fatalf("fixture[0].ToolCalls=%d want 1", len(got[0].ToolCalls))
	}
	if got[0].ToolCalls[0].Name != "get_weather" {
		t.Fatalf("fixture[0].ToolCalls[0].Name=%q want get_weather", got[0].ToolCalls[0].Name)
	}
	if got[0].Content != "" {
		t.Fatalf("fixture[0].Content=%q want empty (tool-call turn)", got[0].Content)
	}
	if got[1].Content != "It's 22°C in Lisbon." {
		t.Fatalf("fixture[1].Content=%q want final assistant text", got[1].Content)
	}
	if len(got[1].ToolCalls) != 0 {
		t.Fatalf("fixture[1].ToolCalls=%v want empty", got[1].ToolCalls)
	}
}

func TestFixturesFromTurns_DrivesMockReplay(t *testing.T) {
	t.Parallel()

	turns := []eval.Turn{
		{Role: "user", Content: "q"},
		{Role: "assistant", ToolCalls: []schema.ToolCall{
			{ID: "call_a", Name: "search", Arguments: `{"q":"beluga"}`},
		}},
		{Role: "tool", Content: "result"},
		{Role: "assistant", Content: "final"},
	}

	m, err := New(config.ProviderConfig{}, WithFixtures(FixturesFromTurns(turns)))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	ctx := context.Background()

	first, err := m.Generate(ctx, nil)
	if err != nil {
		t.Fatalf("Generate #1: %v", err)
	}
	if len(first.ToolCalls) != 1 || first.ToolCalls[0].Name != "search" {
		t.Fatalf("Generate #1 ToolCalls=%v want [search]", first.ToolCalls)
	}

	second, err := m.Generate(ctx, nil)
	if err != nil {
		t.Fatalf("Generate #2: %v", err)
	}
	if second.Text() != "final" {
		t.Fatalf("Generate #2 Text=%q want final", second.Text())
	}
}

func TestFixturesFromTurns_DefensiveCopy(t *testing.T) {
	t.Parallel()

	original := []schema.ToolCall{{ID: "c", Name: "orig"}}
	turns := []eval.Turn{{Role: "assistant", ToolCalls: original}}

	got := FixturesFromTurns(turns)
	if len(got) != 1 || len(got[0].ToolCalls) != 1 {
		t.Fatalf("unexpected fixture shape: %+v", got)
	}
	// Mutate caller's slice — the helper must have copied.
	original[0].Name = "mutated"
	if got[0].ToolCalls[0].Name != "orig" {
		t.Fatalf("fixture ToolCalls aliased caller slice: got Name=%q", got[0].ToolCalls[0].Name)
	}
}

func TestConcurrentGenerate_RaceFree(t *testing.T) {
	t.Parallel()

	fixtures := make([]Fixture, 64)
	for i := range fixtures {
		fixtures[i] = Fixture{Content: "fx"}
	}
	m, err := New(config.ProviderConfig{}, WithFixtures(fixtures))
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	var wg sync.WaitGroup
	ctx := context.Background()
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 4; j++ {
				if _, err := m.Generate(ctx, nil); err != nil {
					t.Errorf("Generate: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()
	if m.Calls() != 64 {
		t.Fatalf("Calls=%d want 64", m.Calls())
	}
}
