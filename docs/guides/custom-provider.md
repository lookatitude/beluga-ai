# Guide: Implement a Custom LLM Provider

**Time:** ~30 minutes
**You will build:** a custom LLM provider that calls a local Ollama server.
**Prerequisites:** [First Agent guide](./first-agent.md), [Provider Template](../patterns/provider-template.md).

## What you'll learn

- Implementing the `llm.ChatModel` interface.
- Translating provider-specific errors to `core.Error` codes.
- Registering the provider via `init()`.
- Writing a table-driven test against an `httptest` server.

Beluga already ships an Ollama provider (`llm/providers/ollama`). This guide builds a small one from scratch, in your own module and under its own name (`local-ollama`), so it can sit next to the built-in one. Registering a name that already exists silently replaces the earlier factory.

## Step 1 — scaffold the package

```bash
mkdir -p providers/localollama
touch providers/localollama/localollama.go providers/localollama/localollama_test.go
```

## Step 2 — implement `llm.ChatModel`

`llm.ChatModel` has four methods: `Generate`, `Stream`, `BindTools` and `ModelID`.

```go
// providers/localollama/localollama.go
//
// Package localollama wraps a local Ollama server as a Beluga LLM provider.
package localollama

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"iter"
	"net/http"
	"time"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

// compile-time check
var _ llm.ChatModel = (*Model)(nil)

// Model calls an Ollama server's /api/chat endpoint.
type Model struct {
	baseURL string
	model   string
	client  *http.Client
	tools   []schema.ToolDefinition
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// toChat converts Beluga messages to Ollama's wire format (text parts only).
func toChat(msgs []schema.Message) []chatMessage {
	roles := map[schema.Role]string{
		schema.RoleSystem: "system",
		schema.RoleHuman:  "user",
		schema.RoleAI:     "assistant",
		schema.RoleTool:   "tool",
	}
	out := make([]chatMessage, 0, len(msgs))
	for _, m := range msgs {
		var text string
		for _, p := range m.GetContent() {
			if tp, ok := p.(schema.TextPart); ok {
				text += tp.Text
			}
		}
		out = append(out, chatMessage{Role: roles[m.GetRole()], Content: text})
	}
	return out
}

// Generate sends the conversation and returns the complete reply.
func (m *Model) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
	body, err := json.Marshal(map[string]any{
		"model":    m.model,
		"messages": toChat(msgs),
		"stream":   false,
	})
	if err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, "localollama: encode request: %w", err)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, m.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return nil, core.Errorf(core.ErrInvalidInput, "localollama: build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, core.Errorf(core.ErrTimeout, "localollama: %w", ctx.Err())
		}
		return nil, core.Errorf(core.ErrProviderDown, "localollama: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// continue
	case http.StatusTooManyRequests:
		return nil, core.Errorf(core.ErrRateLimit, "localollama: rate limited (429)")
	case http.StatusUnauthorized, http.StatusForbidden:
		return nil, core.Errorf(core.ErrAuth, "localollama: status %d", resp.StatusCode)
	default:
		raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, core.Errorf(core.ErrProviderDown, "localollama: status %d: %s", resp.StatusCode, raw)
	}

	var out struct {
		Message chatMessage `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, core.Errorf(core.ErrProviderDown, "localollama: decode: %w", err)
	}
	ai := schema.NewAIMessage(out.Message.Content)
	ai.ModelID = m.model
	return ai, nil
}

// Stream returns the reply as an iterator. For brevity this example calls
// Generate and yields a single chunk; a real provider would read Ollama's
// streaming response line by line.
func (m *Model) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
	return func(yield func(schema.StreamChunk, error) bool) {
		ai, err := m.Generate(ctx, msgs, opts...)
		if err != nil {
			yield(schema.StreamChunk{}, err)
			return
		}
		yield(schema.StreamChunk{Delta: ai.Text(), FinishReason: "stop", ModelID: m.model}, nil)
	}
}

// BindTools returns a copy of the model that carries the tool definitions.
// This example stores them but does not send them to the server.
func (m *Model) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
	cp := *m
	cp.tools = append([]schema.ToolDefinition(nil), tools...)
	return &cp
}

// ModelID returns the configured model name.
func (m *Model) ModelID() string { return m.model }

// New builds a Model from a ProviderConfig. BaseURL, Model and Timeout fall
// back to local defaults.
func New(cfg config.ProviderConfig) (llm.ChatModel, error) {
	m := &Model{baseURL: cfg.BaseURL, model: cfg.Model}
	if m.baseURL == "" {
		m.baseURL = "http://localhost:11434"
	}
	if m.model == "" {
		m.model = "llama3.2"
	}
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	m.client = &http.Client{Timeout: timeout}
	return m, nil
}

func init() {
	llm.Register("local-ollama", New)
}
```

## Step 3 — write the test

```go
// providers/localollama/localollama_test.go
package localollama

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"github.com/lookatitude/beluga-ai/v2/config"
	"github.com/lookatitude/beluga-ai/v2/core"
	"github.com/lookatitude/beluga-ai/v2/llm"
	"github.com/lookatitude/beluga-ai/v2/schema"
)

func TestModel_Generate(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		body     string
		wantCode core.ErrorCode
		wantText string
	}{
		{
			name:     "success",
			status:   http.StatusOK,
			body:     `{"message":{"role":"assistant","content":"hello"}}`,
			wantText: "hello",
		},
		{name: "rate limit", status: http.StatusTooManyRequests, wantCode: core.ErrRateLimit},
		{name: "auth error", status: http.StatusUnauthorized, wantCode: core.ErrAuth},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()

			m, err := New(config.ProviderConfig{Provider: "local-ollama", BaseURL: srv.URL, Model: "test"})
			if err != nil {
				t.Fatal(err)
			}
			resp, err := m.Generate(context.Background(), []schema.Message{schema.NewHumanMessage("hi")})
			if tt.wantCode != "" {
				var ce *core.Error
				if !errors.As(err, &ce) || ce.Code != tt.wantCode {
					t.Fatalf("err = %v, want code %s", err, tt.wantCode)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if resp.Text() != tt.wantText {
				t.Errorf("text = %q, want %q", resp.Text(), tt.wantText)
			}
		})
	}
}

func TestRegistration(t *testing.T) {
	if !slices.Contains(llm.List(), "local-ollama") {
		t.Fatal("local-ollama not registered")
	}
}
```

## Step 4 — use it

```go
import _ "example.com/myrepo/providers/localollama"

model, err := llm.New("local-ollama", config.ProviderConfig{
	Provider: "local-ollama",
	BaseURL:  "http://localhost:11434",
	Model:    "llama3.2",
})
```

## Checklist

- [ ] `var _ llm.ChatModel = (*Model)(nil)` compile-time check.
- [ ] `context.Context` is the first parameter on every public method.
- [ ] Context cancellation respected in the HTTP call (`NewRequestWithContext`).
- [ ] All errors use `core.Errorf` with an appropriate `ErrorCode`.
- [ ] Rate limits map to `ErrRateLimit`, auth to `ErrAuth`, timeouts to `ErrTimeout`.
- [ ] `init()` registers the provider under a name no other provider uses.
- [ ] The factory takes a `config.ProviderConfig` and applies defaults.
- [ ] `Stream` returns an `iter.Seq2[schema.StreamChunk, error]` and stops when `yield` returns false.
- [ ] Tests cover the happy path, rate limit and auth errors.
- [ ] A test verifies that `init()` registered the name.

## Common mistakes

- **Forgetting `NewRequestWithContext`.** Without it, cancellation doesn't propagate and the HTTP call hangs forever on context cancel.
- **Returning `fmt.Errorf` instead of `core.Errorf`.** Middleware can't decide whether to retry.
- **Assuming the server is reachable in tests.** Use `httptest.NewServer` or record fixtures; don't hit real endpoints in unit tests.
- **Forgetting the blank import.** If your user doesn't write `_ "example.com/myrepo/providers/localollama"`, `llm.New("local-ollama", ...)` returns a `core.ErrNotFound` error.

## Related

- [Provider Template pattern](../patterns/provider-template.md) — the universal template.
- [Registry + Factory pattern](../patterns/registry-factory.md) — the registry side.
- [03 — Extensibility Patterns](../architecture/03-extensibility-patterns.md) — full mechanism rationale.
