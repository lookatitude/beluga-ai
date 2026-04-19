# Reference: Interfaces

Every public interface in Beluga, grouped by layer. This is a lookup reference — for the "why" behind each interface, follow the links to the architecture docs.

For live `file:line` anchors, see [`.wiki/architecture/package-map.md`](../../.wiki/architecture/package-map.md).

## Layer 1 — Foundation

### `core.Runnable`

```go
type Runnable interface {
    Invoke(ctx context.Context, input any) (any, error)
    Stream(ctx context.Context, input any) (*Stream[Event[any]], error)
}
```

The universal composable unit. Every agent, tool, retriever, planner is (or can be) a `Runnable`. See [DOC-02](../architecture/02-core-primitives.md).

### `core.Stream[T]`

```go
type Stream[T any] struct { /* opaque */ }
func (s *Stream[T]) Range(yield func(int, T) bool)
```

Wraps an `iter.Seq2[int, T]`. See [DOC-02](../architecture/02-core-primitives.md) and [Streaming pattern](../patterns/streaming-iter-seq2.md).

### `core.Error`

```go
type Error struct {
    Op      string
    Code    ErrorCode
    Message string
    Err     error
}

func (e *Error) Error() string
func (e *Error) Unwrap() error
func (e *Error) Is(target error) bool

func IsRetryable(err error) bool
func Errorf(code ErrorCode, format string, args ...any) *Error
```

See [Error Handling pattern](../patterns/error-handling.md).

## Layer 3 — Capability

### `llm.Provider`

```go
type Provider interface {
    Generate(ctx context.Context, req Request) (*Response, error)
    Stream(ctx context.Context, req Request) (*core.Stream[core.Event[Chunk]], error)
}

type Factory func(cfg Config) (Provider, error)

func Register(name string, factory Factory) error
func New(name string, cfg Config) (Provider, error)
func List() []string
```

### `tool.Tool`

```go
type Tool interface {
    Name() string
    Description() string
    InputSchema() map[string]any
    Execute(ctx context.Context, input map[string]any) (*Result, error)
}

type Middleware func(Tool) Tool
func ApplyMiddleware(t Tool, mws ...Middleware) Tool

type Hooks struct {
    OnStart func(ctx context.Context, name string, input map[string]any) error
    OnEnd   func(ctx context.Context, name string) error
    OnError func(ctx context.Context, name string, err error) error
}
func ComposeHooks(hks ...Hooks) Hooks
```

### `memory.Memory`

```go
type Memory interface {
    Load(ctx context.Context, query string) ([]schema.Message, error)
    Save(ctx context.Context, messages []schema.Message) error
    Clear(ctx context.Context) error
}

type MessageStore interface {
    Append(ctx context.Context, msg schema.Message) error
    List(ctx context.Context, limit int) ([]schema.Message, error)
    Clear(ctx context.Context) error
}
```

### `rag.Retriever`

```go
type Retriever interface {
    Retrieve(ctx context.Context, query string, k int) ([]Document, error)
}

type Embedder interface {
    Embed(ctx context.Context, texts []string) ([][]float32, error)
    Dimensions() int
}

type VectorStore interface {
    Upsert(ctx context.Context, docs []Document, vectors [][]float32) error
    Search(ctx context.Context, vector []float32, k int) ([]Document, error)
    Delete(ctx context.Context, ids []string) error
}
```

### `guard.Guard`

```go
type Guard interface {
    InspectInput(ctx context.Context, input GuardInput) (GuardResult, error)
    InspectOutput(ctx context.Context, output GuardOutput) (GuardResult, error)
    InspectTool(ctx context.Context, tool GuardTool) (GuardResult, error)
}

type Decision int

const (
    DecisionAllow Decision = iota
    DecisionReview
    DecisionBlock
)

type GuardResult struct {
    Decision Decision
    Reason   string
}
```

### `voice.FrameProcessor`

```go
type FrameProcessor interface {
    Process(ctx context.Context, in <-chan Frame, out chan<- Frame) error
}

type Transport interface {
    Start(ctx context.Context) error
    AudioIn() <-chan Frame
    AudioOut() chan<- Frame
    Stop() error
}
```

## Layer 4 — Protocol

### `protocol.Server`

```go
type Server interface {
    Start(ctx context.Context, addr string) error
    Stop(ctx context.Context) error
}
```

Individual protocols (`mcp`, `a2a`, `rest`, `grpc`, `ws`) each implement `Server`.

## Layer 5 — Orchestration

### `orchestration.OrchestrationPattern`

```go
type OrchestrationPattern interface {
    Name() string
    Run(ctx context.Context, team Team, input any) (*core.Stream[core.Event[any]], error)
}
```

Built-in implementations: `Supervisor`, `Handoff`, `ScatterGather`, `Pipeline`, `Blackboard`.

## Layer 6 — Agent runtime

### `agent.Agent`

```go
type Agent interface {
    core.Runnable
    ID() string
    Persona() Persona
    Tools() []Tool
    Card() AgentCard
    Children() []Agent
}
```

### `agent.Planner`

```go
type Planner interface {
    Plan(ctx context.Context, state PlannerState) ([]Action, error)
    Replan(ctx context.Context, state PlannerState, obs Observation) ([]Action, error)
}

type Action interface{ isAction() }

type ActionTool    struct { Tool string; Input map[string]any }
type ActionRespond struct { Text string }
type ActionFinish  struct { Output any }
type ActionHandoff struct { Target string }
```

### `runtime.Runner`

```go
type Runner interface {
    Run(ctx context.Context, input any) (*core.Stream[core.Event[any]], error)
    Serve(ctx context.Context, addr string) error
    Stop(ctx context.Context) error
}

type Plugin interface {
    Name() string
    BeforeTurn(ctx context.Context, session *Session, input any) (any, error)
    AfterTurn(ctx context.Context, session *Session, events []core.Event[any]) error
}
```

### `runtime.SessionService`

```go
type SessionService interface {
    LoadOrCreate(ctx context.Context, id string) (*Session, error)
    Save(ctx context.Context, session *Session) error
    Delete(ctx context.Context, id string) error
}
```

## Layer 2 — Cross-cutting

### `auth.Authenticator`

```go
type Authenticator interface {
    Authenticate(ctx context.Context, req Request) (Claims, error)
}

type Authorizer interface {
    Authorize(ctx context.Context, claims Claims, capability string) error
}
```

### `workflow.Engine`

```go
type Engine interface {
    StartWorkflow(ctx context.Context, id string, spec WorkflowSpec) error
    SignalWorkflow(ctx context.Context, id string, name string, payload any) error
    WaitWorkflow(ctx context.Context, id string) (Result, error)
}
```

## How to find the latest

The interfaces in this reference are stable. For the most current view (with `file:line` anchors, method signatures exactly as they appear in source, and package-specific types), query the wiki:

```bash
.claude/hooks/wiki-query.sh <interface-name>
go doc github.com/lookatitude/beluga-ai/v2/<package>
```

## Related

- [03 — Extensibility Patterns](../architecture/03-extensibility-patterns.md) — how these interfaces compose.
- [18 — Package Dependency Map](../architecture/18-package-dependency-map.md) — which layer each lives in.
- [`.wiki/architecture/package-map.md`](../../.wiki/architecture/package-map.md) — live source references.
