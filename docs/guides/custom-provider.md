# Guide: Implement a Custom LLM Provider

**Time:** ~30 minutes
**You will build:** a custom LLM provider that calls a local Ollama server.
**Prerequisites:** [First Agent guide](./first-agent.md), [Provider Template](../patterns/provider-template.md).

## What you'll learn

- Implementing the `llm.Provider` interface.
- Translating provider-specific errors to `core.Error` codes.
- Registering the provider via `init()`.
- Writing a conformance test.

## Step 1 — scaffold the package

```bash
mkdir -p llm/providers/ollama
touch llm/providers/ollama/ollama.go llm/providers/ollama/ollama_test.go
```

## Step 2 — implement `llm.Provider`

```go
// llm/providers/ollama/ollama.go
//
// Package ollama wraps a local Ollama server as a Beluga LLM provider.
package ollama

import (
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net/http"
    "time"

    "github.com/lookatitude/beluga-ai/core"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

// compile-time check
var _ llm.Provider = (*Provider)(nil)

// Provider calls an Ollama server.
type Provider struct {
    baseURL string
    model   string
    client  *http.Client
}

func (p *Provider) Generate(ctx context.Context, req llm.Request) (*llm.Response, error) {
    body, _ := json.Marshal(map[string]any{
        "model":    p.model,
        "messages": req.Messages,
        "stream":   false,
    })
    httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/api/chat", bytes.NewReader(body))
    if err != nil {
        return nil, core.Errorf(core.ErrInvalidInput, "ollama: build request: %w", err)
    }
    httpReq.Header.Set("Content-Type", "application/json")

    resp, err := p.client.Do(httpReq)
    if err != nil {
        if ctx.Err() != nil {
            return nil, core.Errorf(core.ErrTimeout, "ollama: %w", ctx.Err())
        }
        return nil, core.Errorf(core.ErrProviderDown, "ollama: %w", err)
    }
    defer resp.Body.Close()

    switch resp.StatusCode {
    case http.StatusOK:
        // continue
    case http.StatusTooManyRequests:
        return nil, core.Errorf(core.ErrRateLimit, "ollama: 429")
    case http.StatusUnauthorized:
        return nil, core.Errorf(core.ErrAuth, "ollama: 401")
    default:
        raw, _ := io.ReadAll(resp.Body)
        return nil, core.Errorf(core.ErrProviderDown, "ollama: status %d: %s", resp.StatusCode, raw)
    }

    var out struct {
        Message schema.Message `json:"message"`
    }
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        return nil, core.Errorf(core.ErrProviderDown, "ollama: decode: %w", err)
    }
    return &llm.Response{Message: out.Message}, nil
}

func (p *Provider) Stream(ctx context.Context, req llm.Request) (*core.Stream[core.Event[llm.Chunk]], error) {
    // Ollama supports streaming; for brevity, this example uses Generate + single event.
    resp, err := p.Generate(ctx, req)
    if err != nil {
        return nil, err
    }
    return core.NewStream("ollama.stream", func(yield func(int, core.Event[llm.Chunk]) bool) {
        if !yield(0, core.Event[llm.Chunk]{
            Type:    core.EventData,
            Payload: llm.Chunk{Text: resp.Message.Content},
        }) {
            return
        }
        yield(1, core.Event[llm.Chunk]{Type: core.EventDone})
    }), nil
}

func init() {
    if err := llm.Register("ollama", newFactory()); err != nil {
        panic(err)
    }
}

func newFactory() llm.Factory {
    return func(cfg llm.Config) (llm.Provider, error) {
        baseURL, _ := cfg.GetString("base_url", "http://localhost:11434")
        model, _ := cfg.GetString("model", "llama3.2")
        timeout, _ := cfg.GetDuration("timeout", 60*time.Second)
        return &Provider{
            baseURL: baseURL,
            model:   model,
            client:  &http.Client{Timeout: timeout},
        }, nil
    }
}
```

## Step 3 — write the test

```go
// llm/providers/ollama/ollama_test.go
package ollama

import (
    "context"
    "net/http"
    "net/http/httptest"
    "testing"

    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

func TestProvider_Generate(t *testing.T) {
    tests := []struct {
        name     string
        status   int
        body     string
        wantErr  bool
        wantText string
    }{
        {
            name:     "success",
            status:   200,
            body:     `{"message":{"role":"assistant","content":"hello"}}`,
            wantText: "hello",
        },
        {
            name:    "rate limit",
            status:  429,
            body:    "",
            wantErr: true,
        },
        {
            name:    "auth error",
            status:  401,
            body:    "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
                w.WriteHeader(tt.status)
                _, _ = w.Write([]byte(tt.body))
            }))
            defer srv.Close()

            p := &Provider{baseURL: srv.URL, model: "test", client: srv.Client()}
            resp, err := p.Generate(context.Background(), llm.Request{
                Messages: []schema.Message{{Role: "user", Content: "hi"}},
            })
            if (err != nil) != tt.wantErr {
                t.Fatalf("err=%v wantErr=%v", err, tt.wantErr)
            }
            if err == nil && resp.Message.Content != tt.wantText {
                t.Errorf("content=%q want=%q", resp.Message.Content, tt.wantText)
            }
        })
    }
}

func TestRegistration(t *testing.T) {
    found := false
    for _, name := range llm.List() {
        if name == "ollama" {
            found = true
            break
        }
    }
    if !found {
        t.Fatal("ollama not registered")
    }
}
```

## Step 4 — use it

```go
import _ "example.com/myrepo/llm/providers/ollama"

model, err := llm.New("ollama", llm.Config{
    "base_url": "http://localhost:11434",
    "model":    "llama3.2",
})
```

## Checklist

- [ ] `var _ llm.Provider = (*Provider)(nil)` compile-time check.
- [ ] `context.Context` is first parameter on every public method.
- [ ] Context cancellation respected in HTTP call (`NewRequestWithContext`).
- [ ] All errors use `core.Errorf` with appropriate `ErrorCode`.
- [ ] Rate limits → `ErrRateLimit`, auth → `ErrAuth`, timeouts → `ErrTimeout`.
- [ ] `init()` registers the provider; panic on duplicate.
- [ ] Factory accepts a `llm.Config` and validates it.
- [ ] Stream implementation yields `EventDone` at termination.
- [ ] Test covers happy path, rate limit, auth error.
- [ ] Test verifies `init()` registered the name.

## Common mistakes

- **Forgetting `NewRequestWithContext`.** Without it, cancellation doesn't propagate and the HTTP call hangs forever on context cancel.
- **Returning `fmt.Errorf` instead of `core.Errorf`.** Middleware can't decide whether to retry.
- **Assuming the server is reachable in tests.** Use `httptest.NewServer` or record fixtures; don't hit real endpoints in unit tests.
- **Forgetting the blank-import.** If your user doesn't write `_ "example.com/myrepo/llm/providers/ollama"`, `llm.New("ollama", ...)` returns "not found".

## Related

- [Provider Template pattern](../patterns/provider-template.md) — the universal template.
- [Registry + Factory pattern](../patterns/registry-factory.md) — the registry side.
- [03 — Extensibility Patterns](../architecture/03-extensibility-patterns.md) — full mechanism rationale.
