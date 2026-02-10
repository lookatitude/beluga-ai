# Beluga AI v2 — Architecture & Extensibility

This document describes the extensibility architecture: how every package follows the same patterns, how providers work, how tools work, and step-by-step guides for adding new providers and tools.

## Foundational Patterns

Every extensible package in Beluga follows four interlocking patterns. Learn these once — they apply everywhere.

```mermaid
graph LR
    subgraph extensionContract [Extension Contract]
        iface["Interface<br/>1-4 methods"]
        registry["Registry<br/>Register / New / List"]
        middleware["Middleware<br/>func T to T"]
        hooks["Hooks<br/>Before / After / On"]
    end
    iface --> registry
    registry --> middleware
    middleware --> hooks
```

### Pattern 1: Interface (the contract)

Every extensible component is a Go interface with 1-4 methods. Small interfaces are testable, composable, and easy to implement.

```go
// llm/llm.go — 4 methods
type ChatModel interface {
    Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error)
    Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error]
    BindTools(tools []schema.ToolDefinition) ChatModel
    ModelID() string
}

// tool/tool.go — 4 methods
type Tool interface {
    Name() string
    Description() string
    InputSchema() map[string]any
    Execute(ctx context.Context, input map[string]any) (*Result, error)
}

// memory/memory.go — 4 methods
type Memory interface {
    Save(ctx context.Context, input, output schema.Message) error
    Load(ctx context.Context, query string) ([]schema.Message, error)
    Search(ctx context.Context, query string, k int) ([]schema.Document, error)
    Clear(ctx context.Context) error
}

// rag/embedding/embedder.go — 2 methods
type Embedder interface {
    Embed(ctx context.Context, text string) ([]float32, error)
    EmbedBatch(ctx context.Context, texts []string) ([][]float32, error)
}
```

**Rules**:
- Accept interfaces, return structs.
- If an interface grows beyond 4 methods, split it.
- Optional capabilities use type assertions: `if br, ok := r.(BatchRetriever); ok { ... }`
- Compile-time check: `var _ ChatModel = (*OpenAIModel)(nil)`

### Pattern 2: Registry (discovery and construction)

Every extensible package has a global registry. Providers self-register in `init()`. Users construct instances with `New()`.

```go
// This exact pattern exists in 19 packages across the framework.

// Factory creates an instance from config.
type Factory func(cfg config.ProviderConfig) (ChatModel, error)

var (
    registryMu sync.RWMutex
    registry   = make(map[string]Factory)
)

// Register adds a provider factory. Called from init().
func Register(name string, f Factory) {
    registryMu.Lock()
    defer registryMu.Unlock()
    registry[name] = f
}

// New constructs an instance by name.
func New(name string, cfg config.ProviderConfig) (ChatModel, error) {
    registryMu.RLock()
    f, ok := registry[name]
    registryMu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("llm: unknown provider %q (registered: %v)", name, List())
    }
    return f(cfg)
}

// List returns all registered provider names, sorted.
func List() []string {
    registryMu.RLock()
    defer registryMu.RUnlock()
    names := make([]string, 0, len(registry))
    for name := range registry { names = append(names, name) }
    sort.Strings(names)
    return names
}
```

**Provider self-registration**:

```go
// llm/providers/openai/openai.go
package openai

func init() {
    llm.Register("openai", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
        return New(cfg)
    })
}
```

**User imports provider with blank identifier**:

```go
import (
    "github.com/lookatitude/beluga-ai/llm"
    _ "github.com/lookatitude/beluga-ai/llm/providers/openai"
)

model, err := llm.New("openai", config.ProviderConfig{
    APIKey: os.Getenv("OPENAI_API_KEY"),
    Model:  "gpt-4o",
})
```

**All 19 registries in the framework**:

```mermaid
graph TB
    subgraph registries [Registry Pattern — 19 Packages]
        llmR["llm.Register"]
        plannerR["agent.RegisterPlanner"]
        memR["memory.Register"]
        embedR["embedding.Register"]
        vsR["vectorstore.Register"]
        retR["retriever.Register"]
        loadR["loader.Register"]
        splitR["splitter.Register"]
        sttR["stt.Register"]
        ttsR["tts.Register"]
        s2sR["s2s.Register"]
        transportR["transport.Register"]
        wfR["workflow.Register"]
        stateR["state.Register"]
        guardR["guard.Register"]
        authR["auth.Register"]
        serverR["server.Register"]
        mcpR["mcp.registry.Register"]
        vadR["vad.Register"]
    end
```

### Pattern 3: Middleware (wrapping)

Middleware wraps an interface to add cross-cutting behavior. The signature is always `func(T) T`.

```go
// Definition
type Middleware func(ChatModel) ChatModel

// Application — outside-in (last middleware wraps first)
func ApplyMiddleware(model ChatModel, mws ...Middleware) ChatModel {
    for i := len(mws) - 1; i >= 0; i-- {
        model = mws[i](model)
    }
    return model
}

// Usage
model = llm.ApplyMiddleware(model,
    resilience.WithRetry(3, time.Second),    // outermost: retries wrap everything
    cache.WithLLMCache(myCache),             // middle: cache before rate-limit
    resilience.WithRateLimit(100, 10000),    // innermost: rate-limit at the boundary
)
```

**Execution order**:

```mermaid
graph LR
    call["Generate()"] --> retry["Retry MW"]
    retry --> cacheM["Cache MW"]
    cacheM --> rateLimit["RateLimit MW"]
    rateLimit --> provider["OpenAI Provider"]
    provider --> rateLimit
    rateLimit --> cacheM
    cacheM --> retry
    retry --> call
```

Middleware exists in: `llm`, `tool`, `agent`, `memory`, `orchestration`, `workflow`, `auth`.

### Pattern 4: Hooks (lifecycle interception)

Hooks are a struct with optional callback function fields. `nil` hooks are skipped. Multiple hooks compose via `ComposeHooks()`.

```go
// Definition — all fields optional
type Hooks struct {
    OnStart        func(ctx context.Context, input string) error
    OnEnd          func(ctx context.Context, result string, err error)
    OnError        func(ctx context.Context, err error) error
    BeforePlan     func(ctx context.Context, state PlannerState) error
    AfterPlan      func(ctx context.Context, actions []Action) error
    BeforeAct      func(ctx context.Context, action Action) error
    AfterAct       func(ctx context.Context, action Action, obs Observation) error
    OnToolCall     func(ctx context.Context, call ToolCallInfo) error
    OnToolResult   func(ctx context.Context, call ToolCallInfo, result *tool.Result) error
    OnIteration    func(ctx context.Context, iteration int) error
    OnHandoff      func(ctx context.Context, from, to string) error
    BeforeGenerate func(ctx context.Context) error
    AfterGenerate  func(ctx context.Context) error
}

// Composition — each receives output of previous
func ComposeHooks(hooks ...Hooks) Hooks { /* chains all non-nil callbacks */ }
```

**Hook naming convention**:

| Pattern | When | Signature |
|---------|------|-----------|
| `Before<Action>` | Before executing | `func(ctx, input) error` — can abort |
| `After<Action>` | After executing | `func(ctx, output, err) error` — can modify |
| `On<Event>` | When event occurs | `func(ctx, data) error` — observe or modify |

**Middleware vs Hooks**:

| Aspect | Middleware | Hooks |
|--------|-----------|-------|
| Scope | Wraps entire interface | Fires at specific points |
| Use case | Retry, cache, rate-limit, tracing | Audit, cost tracking, validation |
| Execution | Outermost first | Within the execution |
| Composition | `ApplyMiddleware(model, mw1, mw2)` | `ComposeHooks(h1, h2)` |

Hooks exist in: `agent`, `tool`, `memory`, `embedding`, `vectorstore`, `stt`, `tts`, `s2s`, `orchestration`, `workflow`, `auth`.

## Streaming Architecture

### The Primitive: `iter.Seq2[T, error]`

All public streaming APIs use Go 1.23+ range-over-func iterators.

```go
// core/stream.go
type Stream[T any] = iter.Seq2[Event[T], error]
```

**Producing a stream**:

```go
func (m *Model) Stream(ctx context.Context, msgs []schema.Message) iter.Seq2[schema.StreamChunk, error] {
    return func(yield func(schema.StreamChunk, error) bool) {
        stream, err := m.client.ChatCompletionStream(ctx, m.buildRequest(msgs))
        if err != nil {
            yield(schema.StreamChunk{}, m.mapError(err))
            return
        }
        defer stream.Close()

        for {
            select {
            case <-ctx.Done():
                yield(schema.StreamChunk{}, ctx.Err())
                return
            default:
            }

            chunk, err := stream.Recv()
            if err == io.EOF {
                return
            }
            if err != nil {
                yield(schema.StreamChunk{}, m.mapError(err))
                return
            }
            if !yield(m.convertChunk(chunk), nil) {
                return // consumer stopped — respect backpressure
            }
        }
    }
}
```

**Consuming a stream**:

```go
for chunk, err := range model.Stream(ctx, msgs) {
    if err != nil {
        log.Error("stream error", "err", err)
        break
    }
    fmt.Print(chunk.Text)
}
```

**Stream composition utilities** (all in `core/`):

| Function | Purpose |
|----------|---------|
| `CollectStream[T]` | Drain stream to `[]Event[T]` |
| `MapStream[T, U]` | Transform each event |
| `FilterStream[T]` | Keep events matching predicate |
| `MergeStreams[T]` | Combine multiple streams |
| `FanOut[T]` | Send to N consumers |
| `BufferedStream[T]` | Buffer with backpressure |

```mermaid
graph LR
    source["Source Stream"] --> mapS["MapStream<br/>Transform"]
    mapS --> filterS["FilterStream<br/>Predicate"]
    filterS --> consumer1["Consumer 1"]
    filterS --> consumer2["Consumer 2"]
```

**When to use channels**: Internal goroutine communication (voice frame processors, background workers). Never in public API return types.

## Error Architecture

### Typed Errors

```go
// core/errors.go
type Error struct {
    Op      string    // "llm.Generate", "tool.Execute", "rag.Retrieve"
    Code    ErrorCode // rate_limit, timeout, auth_error, etc.
    Message string    // Human-readable description
    Err     error     // Wrapped cause (supports errors.Is/As)
}
```

### Error Codes

| Code | Retryable | When |
|------|-----------|------|
| `rate_limit` | Yes | Provider throttled the request |
| `timeout` | Yes | Operation exceeded deadline |
| `provider_unavailable` | Yes | Provider is unreachable |
| `auth_error` | No | Authentication or authorization failed |
| `invalid_input` | No | Malformed or missing input |
| `tool_failed` | No | Tool execution returned an error |
| `guard_blocked` | No | Guard rejected the request |
| `budget_exhausted` | No | Token or cost budget exceeded |

### Error Flow

```mermaid
graph TD
    providerErr["Provider SDK Error<br/>(e.g. openai.APIError)"] --> mapErr["Provider maps to core.Error<br/>mapError(err)"]
    mapErr --> coreErr["core.Error{Op, Code, Message, Err}"]
    coreErr --> middleware["Middleware checks IsRetryable()"]
    middleware -->|retryable| retry["Retry with backoff"]
    middleware -->|not retryable| propagate["Propagate to caller"]
    retry -->|success| result["Result"]
    retry -->|max attempts| propagate
```

## Agent Architecture

### Executor Loop

The executor is planner-agnostic. It receives actions from the planner, executes them, collects observations, and asks the planner to replan.

```mermaid
stateDiagram-v2
    [*] --> ReceiveInput
    ReceiveInput --> Plan: Planner.Plan(state)
    Plan --> ExecuteActions
    ExecuteActions --> ToolExec: ActionType = tool
    ExecuteActions --> Handoff: ActionType = handoff
    ExecuteActions --> Respond: ActionType = respond
    ExecuteActions --> Finish: ActionType = finish
    ToolExec --> CollectObservation
    Handoff --> CollectObservation
    Respond --> EmitEvent
    EmitEvent --> CollectObservation
    CollectObservation --> Replan: Planner.Replan(state + observations)
    Replan --> ExecuteActions: more actions
    Replan --> Finish: done
    Finish --> [*]
```

### Planner Interface

```go
type Planner interface {
    Plan(ctx context.Context, state PlannerState) ([]Action, error)
    Replan(ctx context.Context, state PlannerState) ([]Action, error)
}
```

**Built-in planners**:

| Planner | Strategy | When to use |
|---------|----------|-------------|
| ReAct | Think → Act → Observe | General purpose (default) |
| Reflexion | Actor + Evaluator + Self-Reflection | Tasks requiring quality improvement |
| Plan-and-Execute | Full plan then step-by-step | Multi-step tasks with known structure |
| Structured | JSON output with schema validation | Structured data extraction |
| Conversational | Optimized multi-turn | Chat applications |

All planners register via `agent.RegisterPlanner("react", factory)`.

### Handoffs as Tools

Handoffs are converted to `transfer_to_{id}` tools that the LLM sees in its tool list.

```go
// Auto-generates transfer_to_support tool
handoff := agent.HandoffTo(supportAgent, "Transfer to support for billing issues")
tools := agent.HandoffsToTools([]agent.Handoff{handoff})
// tools[0].Name() == "transfer_to_support"
```

The LLM decides when to hand off. The executor handles the tool call by invoking the target agent. `InputFilter` controls what context passes. `IsEnabled` can disable handoffs dynamically.

### Workflow Agents

Deterministic orchestration without LLM — for predictable pipelines.

```mermaid
graph TB
    subgraph sequential [SequentialAgent]
        s1["Agent A"] --> s2["Agent B"] --> s3["Agent C"]
    end
    subgraph parallel [ParallelAgent]
        p1["Agent A"]
        p2["Agent B"]
        p3["Agent C"]
    end
    subgraph loop [LoopAgent]
        l1["Agent A"] -->|condition met?| l2{"Done?"}
        l2 -->|no| l1
        l2 -->|yes| l3["Result"]
    end
```

## The Provider System

### How Providers Work

Every provider follows the same lifecycle:

```mermaid
sequenceDiagram
    participant Init as Provider init()
    participant Registry as Package Registry
    participant User as User Code
    participant Provider as Provider Instance

    Init->>Registry: Register("openai", factory)
    Note over Init,Registry: Happens at import time

    User->>Registry: New("openai", config)
    Registry->>Provider: factory(config)
    Provider-->>Registry: ChatModel instance
    Registry-->>User: ChatModel
    User->>Provider: Generate(ctx, msgs)
    Provider-->>User: *AIMessage
```

### Provider Package Structure

```
llm/providers/openai/
├── openai.go          # Model struct, New(), init(), Generate(), Stream(), BindTools(), ModelID()
├── stream.go          # Streaming implementation (iter.Seq2 producer)
├── errors.go          # mapError() — maps OpenAI errors to core.Error
├── options.go         # Provider-specific options (if any)
├── openai_test.go     # Unit tests with httptest
└── testdata/          # Recorded HTTP responses
    ├── chat_completion.json
    └── stream_completion.jsonl
```

### Error Mapping

Every provider maps SDK-specific errors to `core.Error`:

```go
func (m *Model) mapError(op string, err error) error {
    var apiErr *openai.APIError
    if !errors.As(err, &apiErr) {
        return core.NewError(op, core.ErrProviderDown, "unknown error", err)
    }
    switch apiErr.StatusCode {
    case 401:
        return core.NewError(op, core.ErrAuth, apiErr.Message, err)
    case 429:
        return core.NewError(op, core.ErrRateLimit, apiErr.Message, err)
    case 408, 504:
        return core.NewError(op, core.ErrTimeout, apiErr.Message, err)
    case 400:
        return core.NewError(op, core.ErrInvalidInput, apiErr.Message, err)
    default:
        return core.NewError(op, core.ErrProviderDown, apiErr.Message, err)
    }
}
```

## The Tool System

### How Tools Work

Tools are instances (not factories). They live in a `tool.Registry` which is instance-based (Add/Get/List/Remove), unlike the global factory registries.

```mermaid
graph TB
    funcTool["FuncTool<br/>Wrap Go function"] --> registry["tool.Registry<br/>Add / Get / List"]
    mcpTool["MCP Tools<br/>Remote via Streamable HTTP"] --> registry
    handoffTool["Handoff Tools<br/>transfer_to_ agents"] --> registry
    builtinTool["Built-in Tools<br/>Calculator, HTTP, Shell"] --> registry
    registry --> agent["Agent binds tools to LLM"]
    agent --> llm["LLM sees tools in context"]
    llm --> toolCall["LLM emits ToolCall"]
    toolCall --> execute["tool.Execute(ctx, input)"]
    execute --> result["tool.Result<br/>Multimodal []ContentPart"]
```

### FuncTool — Wrapping Go Functions

`FuncTool` auto-generates JSON Schema from Go struct tags:

```go
type WeatherInput struct {
    City  string `json:"city" description:"City name" required:"true"`
    Units string `json:"units" description:"celsius or fahrenheit" default:"celsius"`
}

weatherTool := tool.NewFuncTool("get_weather", "Get current weather",
    func(ctx context.Context, input WeatherInput) (*tool.Result, error) {
        weather := fetchWeather(input.City, input.Units)
        return tool.TextResult(weather), nil
    },
)
```

Supported struct tags: `json` (field name), `description`, `required`, `default`, `enum`, `minimum`, `maximum`.

### MCP Integration

Tools from remote MCP servers are wrapped as native `tool.Tool` instances:

```go
tools, err := tool.FromMCP(ctx, "https://mcp-server.example.com", tool.MCPOptions{
    Transport: "streamable-http",
    SessionID: "my-session",
})
// tools are now []tool.Tool — use like any other tool
```

### Tool Middleware and Hooks

```go
// Middleware wraps tools
tool = tool.ApplyMiddleware(myTool,
    tool.WithTimeout(10 * time.Second),
    tool.WithRetry(3),
)

// Hooks observe execution
hooks := tool.Hooks{
    BeforeExecute: func(ctx context.Context, name string, input map[string]any) error {
        slog.Info("tool call", "name", name, "input", input)
        return nil
    },
    AfterExecute: func(ctx context.Context, name string, result *tool.Result, err error) {
        slog.Info("tool result", "name", name, "error", err)
    },
}
```

## How To: Add a New LLM Provider

### Step 1: Create the provider package

```
llm/providers/myprovider/
├── myprovider.go
├── myprovider_test.go
└── testdata/
```

### Step 2: Implement ChatModel

```go
package myprovider

import (
    "context"
    "iter"

    "github.com/lookatitude/beluga-ai/config"
    "github.com/lookatitude/beluga-ai/core"
    "github.com/lookatitude/beluga-ai/llm"
    "github.com/lookatitude/beluga-ai/schema"
)

// Compile-time interface check
var _ llm.ChatModel = (*Model)(nil)

type Model struct {
    client *Client
    model  string
    tools  []schema.ToolDefinition
}

func New(cfg config.ProviderConfig) (*Model, error) {
    if cfg.APIKey == "" {
        return nil, core.NewError("myprovider.new", core.ErrAuth, "API key required", nil)
    }
    return &Model{
        client: newClient(cfg.APIKey, cfg.BaseURL),
        model:  cfg.Model,
    }, nil
}

func (m *Model) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
    req := m.buildRequest(msgs, opts...)
    resp, err := m.client.Complete(ctx, req)
    if err != nil {
        return nil, m.mapError("llm.generate", err)
    }
    return m.convertResponse(resp), nil
}

func (m *Model) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) iter.Seq2[schema.StreamChunk, error] {
    return func(yield func(schema.StreamChunk, error) bool) {
        req := m.buildRequest(msgs, opts...)
        stream, err := m.client.StreamComplete(ctx, req)
        if err != nil {
            yield(schema.StreamChunk{}, m.mapError("llm.stream", err))
            return
        }
        defer stream.Close()

        for {
            select {
            case <-ctx.Done():
                yield(schema.StreamChunk{}, ctx.Err())
                return
            default:
            }
            chunk, err := stream.Next()
            if err == io.EOF { return }
            if err != nil {
                yield(schema.StreamChunk{}, m.mapError("llm.stream", err))
                return
            }
            if !yield(m.convertChunk(chunk), nil) { return }
        }
    }
}

func (m *Model) BindTools(tools []schema.ToolDefinition) llm.ChatModel {
    return &Model{client: m.client, model: m.model, tools: tools}
}

func (m *Model) ModelID() string { return "myprovider/" + m.model }
```

### Step 3: Register in init()

```go
func init() {
    llm.Register("myprovider", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
        return New(cfg)
    })
}
```

### Step 4: Map errors

```go
func (m *Model) mapError(op string, err error) error {
    // Map provider-specific errors to core.Error with correct ErrorCode
    // See the Error Architecture section above
}
```

### Step 5: Write tests

```go
func TestGenerate(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        data, _ := os.ReadFile("testdata/chat_completion.json")
        w.Header().Set("Content-Type", "application/json")
        w.Write(data)
    }))
    defer server.Close()

    model, err := New(config.ProviderConfig{
        APIKey:  "test-key",
        Model:   "my-model",
        BaseURL: server.URL,
    })
    require.NoError(t, err)

    resp, err := model.Generate(context.Background(), []schema.Message{
        schema.NewHumanMessage("hello"),
    })
    require.NoError(t, err)
    assert.NotEmpty(t, resp.Text())
}
```

### Step 6: Users import it

```go
import _ "github.com/lookatitude/beluga-ai/llm/providers/myprovider"

model, err := llm.New("myprovider", config.ProviderConfig{...})
```

### Provider Checklist

- [ ] Implements the full interface (all methods)
- [ ] Registers via `init()` with parent package's `Register()`
- [ ] Maps all provider errors to `core.Error` with correct ErrorCode
- [ ] Supports context cancellation (checks `ctx.Done()` in streams)
- [ ] Includes token/usage metrics in responses
- [ ] Compile-time check: `var _ Interface = (*Impl)(nil)`
- [ ] Unit tests with httptest and recorded responses
- [ ] Handles tool calling in both Generate and Stream paths

## How To: Add a New Tool

### Option A: FuncTool (wrap a Go function)

```go
type SearchInput struct {
    Query string `json:"query" description:"Search query" required:"true"`
    Limit int    `json:"limit" description:"Max results" default:"10"`
}

searchTool := tool.NewFuncTool("web_search", "Search the web",
    func(ctx context.Context, input SearchInput) (*tool.Result, error) {
        results, err := mySearchAPI.Search(ctx, input.Query, input.Limit)
        if err != nil {
            return nil, err
        }
        return tool.TextResult(formatResults(results)), nil
    },
)
```

### Option B: Implement the Tool interface

```go
type DatabaseTool struct {
    db *sql.DB
}

var _ tool.Tool = (*DatabaseTool)(nil)

func (t *DatabaseTool) Name() string        { return "query_database" }
func (t *DatabaseTool) Description() string  { return "Execute a read-only SQL query" }
func (t *DatabaseTool) InputSchema() map[string]any {
    return map[string]any{
        "type": "object",
        "properties": map[string]any{
            "sql": map[string]any{
                "type":        "string",
                "description": "SQL SELECT query to execute",
            },
        },
        "required": []string{"sql"},
    }
}

func (t *DatabaseTool) Execute(ctx context.Context, input map[string]any) (*tool.Result, error) {
    query, ok := input["sql"].(string)
    if !ok {
        return tool.ErrorResult(fmt.Errorf("sql field required")), nil
    }
    rows, err := t.db.QueryContext(ctx, query)
    if err != nil {
        return nil, err
    }
    defer rows.Close()
    // ... format results
    return tool.TextResult(formatted), nil
}
```

### Option C: Tools from MCP server

```go
tools, err := tool.FromMCP(ctx, "https://my-mcp-server.com", tool.MCPOptions{})
```

### Register tools with an agent

```go
registry := tool.NewRegistry()
registry.Add(searchTool)
registry.Add(&DatabaseTool{db: myDB})

myAgent := agent.NewBaseAgent(
    agent.WithID("assistant"),
    agent.WithTools(registry.List()...),
    agent.WithLLM(model),
)
```

## How To: Add a New Provider for Any Package

The same pattern applies to all 19 registries. Here is the generic procedure:

### 1. Identify the interface

| Package | Interface | Key method |
|---------|-----------|------------|
| `llm` | `ChatModel` | Generate, Stream |
| `rag/embedding` | `Embedder` | Embed, EmbedBatch |
| `rag/vectorstore` | `VectorStore` | Add, Search, Delete |
| `rag/retriever` | `Retriever` | Retrieve |
| `rag/loader` | `DocumentLoader` | Load |
| `rag/splitter` | `TextSplitter` | Split |
| `voice/stt` | `STT` | Transcribe |
| `voice/tts` | `TTS` | Synthesize |
| `voice/s2s` | `S2S` | NewSession |
| `voice/transport` | `AudioTransport` | Connect, Send, Receive |
| `memory` | `Memory` | Save, Load, Search, Clear |
| `workflow` | `DurableExecutor` | Execute, Signal, Query, Cancel |
| `guard` | `Guard` | Validate |
| `auth` | `Policy` | Authorize |
| `state` | `Store` | Get, Set, Delete, Watch |
| `cache` | `Cache` | Get, Set, GetSemantic |
| `server` | `ServerAdapter` | Mount, Start, Stop |

### 2. Create the package

```
<parent>/providers/<name>/
├── <name>.go          # Implementation + New() + init()
├── <name>_test.go     # Tests
└── testdata/          # Test fixtures (optional)
```

### 3. Implement, register, test

```go
package myprovider

var _ parentpkg.Interface = (*MyImpl)(nil) // compile-time check

func init() {
    parentpkg.Register("myprovider", func(cfg config.ProviderConfig) (parentpkg.Interface, error) {
        return New(cfg)
    })
}

func New(cfg config.ProviderConfig) (*MyImpl, error) { /* validate config, create instance */ }
// ... implement all interface methods
// ... map errors to core.Error
// ... respect context cancellation
```

## Observability Architecture

### OpenTelemetry Integration

Beluga uses `gen_ai.*` semantic conventions for all AI operations.

```mermaid
graph LR
    llmCall["LLM Generate"] --> span["OTel Span<br/>gen_ai.operation.name = chat<br/>gen_ai.request.model = gpt-4o"]
    span --> metrics["OTel Metrics<br/>gen_ai.client.token.usage<br/>gen_ai.client.operation.duration"]
    span --> logs["slog Structured Log<br/>operation, model, latency, tokens"]
    span --> exporter["TraceExporter<br/>langsmith / langfuse / phoenix / opik"]
```

**Span attributes** (per GenAI semantic conventions):

| Attribute | Example |
|-----------|---------|
| `gen_ai.operation.name` | `chat`, `embeddings` |
| `gen_ai.provider.name` | `openai`, `anthropic` |
| `gen_ai.request.model` | `gpt-4o` |
| `gen_ai.response.model` | `gpt-4o-2024-08-06` |
| `gen_ai.usage.input_tokens` | `150` |
| `gen_ai.usage.output_tokens` | `50` |
| `gen_ai.request.temperature` | `0.7` |

### Health Checks

Every `Lifecycle` component exposes `Health() HealthStatus`. The `App` aggregates all component health into a single endpoint.

States: `healthy`, `degraded`, `unhealthy`.

## Configuration Architecture

### ProviderConfig

All providers accept `config.ProviderConfig`:

```go
type ProviderConfig struct {
    APIKey   string
    Model    string
    BaseURL  string
    Options  map[string]any // provider-specific options
}
```

### Loading

```go
cfg, err := config.Load[MyConfig]("config.yaml")  // file
cfg, err := config.LoadFromEnv[MyConfig]()         // environment
config.MergeEnv(cfg)                                // overlay env on file config
config.Validate(cfg)                                // validate
```

### Hot Reload

```go
watcher := config.NewFileWatcher("config.yaml")
watcher.OnChange(func(cfg MyConfig) { /* apply new config */ })
watcher.Start(ctx)
```

## Resilience Architecture

### Composable Patterns

```mermaid
graph LR
    call["API Call"] --> retry["Retry<br/>3 attempts, exp backoff"]
    retry --> cb["Circuit Breaker<br/>closed/open/half-open"]
    cb --> rl["Rate Limiter<br/>100 RPM, 10K TPM"]
    rl --> provider["Provider"]
```

Each pattern is a middleware: `func(ChatModel) ChatModel`.

| Pattern | Purpose | Config |
|---------|---------|--------|
| **Retry** | Handle transient errors | Max attempts, backoff, jitter |
| **Circuit Breaker** | Prevent cascading failures | Threshold, timeout, half-open probes |
| **Hedge** | Reduce tail latency | Parallel to N providers, use first result |
| **Rate Limit** | Prevent overload | RPM, TPM, MaxConcurrent |

### Composition

```go
model = llm.ApplyMiddleware(model,
    resilience.WithRetry(3, resilience.ExponentialBackoff(time.Second)),
    resilience.WithCircuitBreaker(resilience.CBConfig{Threshold: 5, Timeout: 30*time.Second}),
    resilience.WithRateLimit(resilience.RateLimitConfig{RPM: 100, TPM: 10000}),
)
```

## Guard Pipeline

Three-stage safety pipeline. Guards run automatically at each stage.

```mermaid
graph LR
    input["User Input"] --> stage1["Stage 1: Input Guards<br/>PII detection, Prompt injection,<br/>Content policy"]
    stage1 -->|pass| llm["LLM Generate"]
    stage1 -->|block| blocked["Blocked: guard_blocked"]
    llm --> stage2["Stage 2: Output Guards<br/>Toxicity, Hallucination,<br/>Compliance"]
    stage2 -->|pass| tools["Tool Execution"]
    stage2 -->|block| blocked
    tools --> stage3["Stage 3: Tool Guards<br/>Authorization, Input validation,<br/>Rate limiting"]
    stage3 -->|pass| result["Result"]
    stage3 -->|block| blocked
```

Guard providers: guardrailsai, lakera, llmguard, azuresafety, nemo.

## Memory Architecture

Three-tier memory inspired by MemGPT:

```mermaid
graph TB
    agent["Agent"] --> composite["CompositeMemory"]
    composite --> coreM["Core Memory<br/>Always in context<br/>Persona + User info"]
    composite --> recallM["Recall Memory<br/>Searchable history<br/>via MessageStore"]
    composite --> archivalM["Archival Memory<br/>Long-term knowledge<br/>via VectorStore + Embedder"]
    composite --> graphM["Graph Memory<br/>Structured relationships<br/>via GraphStore"]
```

| Tier | Latency | Capacity | Use case |
|------|---------|----------|----------|
| Core | 0ms (in prompt) | Small (system prompt) | Persona, user preferences |
| Recall | ~1ms | Medium (conversation) | Recent conversation history |
| Archival | ~10ms | Large (knowledge base) | Documents, past conversations |
| Graph | ~10ms | Large (relationships) | Entity relationships, knowledge graph |

## Protocol Architecture

### MCP (Model Context Protocol)

```mermaid
sequenceDiagram
    participant Client as Beluga Agent
    participant Server as MCP Server

    Client->>Server: POST /mcp (JSON-RPC: tools/list)
    Server-->>Client: Tool definitions
    Client->>Server: POST /mcp (JSON-RPC: tools/call)
    Server-->>Client: Tool result (JSON or SSE stream)
    Client->>Server: DELETE /mcp
    Note over Client,Server: Streamable HTTP transport<br/>Mcp-Session-Id header
```

### A2A (Agent-to-Agent)

```mermaid
sequenceDiagram
    participant ClientAgent as Client Agent
    participant RemoteAgent as Remote Agent

    ClientAgent->>RemoteAgent: GET /.well-known/agent.json
    RemoteAgent-->>ClientAgent: AgentCard (capabilities, skills, auth)
    ClientAgent->>RemoteAgent: SendMessage (JSON-RPC or gRPC)
    RemoteAgent-->>ClientAgent: Task{status: working}
    RemoteAgent-->>ClientAgent: Task{status: completed, artifacts: [...]}
```

Task lifecycle: `submitted` → `working` → `input-required` → `completed` / `failed` / `canceled`.

## Design Guarantees

1. **No circular imports** — dependency flows strictly downward: foundation → capability → infrastructure → protocol.
2. **Zero external deps in foundation** — `core/` and `schema/` depend only on stdlib + OTel.
3. **Every public function takes `context.Context` first** — no exceptions.
4. **Every interface has a compile-time check** — `var _ Interface = (*Impl)(nil)`.
5. **Every provider maps errors** — no raw SDK errors leak through.
6. **Every stream respects cancellation** — checks `ctx.Done()` and `yield` return value.
7. **Every extensible package uses the same registry** — `Register()` + `New()` + `List()`.
8. **Every hook is optional** — `nil` hooks are skipped, never panic.
9. **Every middleware composes** — `ApplyMiddleware(base, mw1, mw2, ...)` works uniformly.
10. **Invoke is always derived from Stream** — `Stream → Collect → return last`.

## Cross-References

- [concepts.md](concepts.md) — Design principles and key decisions
- [packages.md](packages.md) — Package layout, interfaces, and how packages work together
- [providers.md](providers.md) — Full provider catalog with counts and categories
