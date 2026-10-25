# Reference: Interfaces

Every exported interface in the Beluga library, grouped by layer. Each block is the interface declaration as it appears in source, including its method comments. This is a lookup reference; for the "why" behind an interface, follow the architecture docs.

This page is generated from the source tree (`go/ast` over every non-test `.go` file outside `internal/` and `cmd/`). When it disagrees with the code, the code wins; `go doc github.com/lookatitude/beluga-ai/v2/<package> <Interface>` prints the live declaration.

Types that look like extension points but are **structs**, not interfaces, are not listed here. Notable examples: `runtime.Runner`, `runtime.Team`, `agent.BaseAgent`, `tool.Registry`.

## Layer 1 — Foundation

### `core`

#### `core.FlowController`

FlowController provides backpressure control for streams. It allows producers to pause/resume based on consumer readiness and can be queried for current buffer state.

```go
type FlowController interface {
	// Acquire blocks until capacity is available or ctx is cancelled.
	Acquire(ctx context.Context) error

	// Release signals that capacity has been freed.
	Release()

	// TryAcquire attempts to acquire without blocking. Returns true if
	// successful, false if no capacity is currently available.
	TryAcquire() bool
}
```

#### `core.Lifecycle`

Lifecycle is the interface for components that require explicit start/stop management. Components are started in registration order and stopped in reverse order.

```go
type Lifecycle interface {
	// Start initialises the component. It should block until the component
	// is ready to serve, or return an error.
	Start(ctx context.Context) error

	// Stop gracefully shuts down the component. It should drain in-flight
	// work before returning.
	Stop(ctx context.Context) error

	// Health returns the current health status of the component.
	Health() HealthStatus
}
```

#### `core.Option`

Option is a functional option that can be applied to any configurable type. The target parameter receives the configuration struct to modify.

```go
type Option interface {
	Apply(target any)
}
```

#### `core.Runnable`

Runnable is the universal execution interface. Every component that can process input — LLMs, tools, agents, pipelines — implements Runnable.

```go
type Runnable interface {
	// Invoke executes the runnable synchronously and returns a single result.
	Invoke(ctx context.Context, input any, opts ...Option) (any, error)

	// Stream executes the runnable and returns an iterator of intermediate
	// results. Callers range over the returned iter.Seq2.
	Stream(ctx context.Context, input any, opts ...Option) iter.Seq2[any, error]
}
```

### `schema`

#### `schema.ContentPart`

ContentPart is the interface implemented by all multimodal content types. Each message contains a slice of ContentParts, enabling rich multimodal conversations that mix text, images, audio, video, and files.

```go
type ContentPart interface {
	// PartType returns the ContentType identifying this part.
	PartType() ContentType
}
```

#### `schema.Message`

Message is the interface implemented by all message types in a conversation. Messages carry multimodal content parts and optional metadata.

```go
type Message interface {
	// GetRole returns the role of the message sender.
	GetRole() Role
	// GetContent returns the content parts that make up this message.
	GetContent() []ContentPart
	// GetMetadata returns arbitrary metadata associated with this message.
	GetMetadata() map[string]any
}
```

### `config`

#### `config.Watcher`

Watcher watches for configuration changes and invokes a callback when the configuration is updated. Implementations may poll files, watch key-value stores, or subscribe to change notifications.

```go
type Watcher interface {
	// Watch starts watching for changes and calls callback whenever the
	// configuration changes. It blocks until ctx is cancelled or an
	// unrecoverable error occurs.
	Watch(ctx context.Context, callback func(newConfig any)) error

	// Close releases resources held by the watcher.
	Close() error
}
```

### `config/agentfile`

#### `agentfile.Deserializer`

Deserializer reads AgentFiles from bytes.

```go
type Deserializer interface {
	// Deserialize converts bytes to an AgentFile.
	Deserialize(ctx context.Context, data []byte) (*AgentFile, error)
}
```

#### `agentfile.Serializer`

Serializer writes AgentFiles to bytes.

```go
type Serializer interface {
	// Serialize converts an AgentFile to bytes.
	Serialize(ctx context.Context, af *AgentFile) ([]byte, error)
}
```

#### `agentfile.VersionMigrator`

VersionMigrator migrates AgentFiles between format versions.

```go
type VersionMigrator interface {
	// Migrate updates an AgentFile to the target version.
	Migrate(ctx context.Context, af *AgentFile, targetVersion string) (*AgentFile, error)
	// SupportedVersions returns the versions this migrator can handle.
	SupportedVersions() []string
}
```

### `config/declarative`

#### `declarative.Builder`

Builder constructs runtime objects from an AgentSpec.

```go
type Builder interface {
	// Build creates an agent configuration from a spec.
	// The returned AgentBuild contains all resolved components.
	Build(ctx context.Context, spec *AgentSpec) (*AgentBuild, error)
}
```

#### `declarative.Parser`

Parser reads and validates an AgentSpec from various sources.

```go
type Parser interface {
	// Parse reads an AgentSpec from raw bytes.
	Parse(ctx context.Context, data []byte) (*AgentSpec, error)
}
```

### `o11y`

#### `o11y.HealthChecker`

HealthChecker is implemented by any component that can report its health.

```go
type HealthChecker interface {
	// HealthCheck performs a health probe and returns the result.
	HealthCheck(ctx context.Context) HealthResult
}
```

#### `o11y.Span`

Span wraps an OpenTelemetry span with a simplified API for GenAI operations.

```go
type Span interface {
	// End finishes the span, recording its duration.
	End()

	// SetAttributes adds key-value attributes to the span.
	SetAttributes(attrs Attrs)

	// RecordError records an error on the span without setting its status.
	RecordError(err error)

	// SetStatus sets the span's status code and descriptive message.
	SetStatus(code StatusCode, msg string)
}
```

#### `o11y.TraceExporter`

TraceExporter is implemented by backends that capture detailed LLM call data for analysis, debugging, or cost tracking. Examples include Langfuse, Arize Phoenix, and custom analytics stores.

```go
type TraceExporter interface {
	// ExportLLMCall sends a completed LLM call record to the backend.
	ExportLLMCall(ctx context.Context, data LLMCallData) error
}
```

### `o11y/viz`

#### `viz.EventCollector`

EventCollector collects execution events and builds an ExecutionGraph.

```go
type EventCollector interface {
	// Record adds an execution event to the collector.
	Record(ctx context.Context, event ExecutionEvent) error
	// Build constructs an ExecutionGraph from all collected events.
	Build(ctx context.Context) (*ExecutionGraph, error)
}
```

#### `viz.GraphRenderer`

GraphRenderer renders an ExecutionGraph to a specific format.

```go
type GraphRenderer interface {
	// Render converts the graph into a byte representation (JSON, DOT, etc.).
	Render(ctx context.Context, graph *ExecutionGraph) ([]byte, error)
}
```

## Layer 2 — Cross-cutting

### `auth`

#### `auth.Policy`

Policy determines whether a subject is authorized to perform a given permission on a resource. Implementations must be safe for concurrent use.

```go
type Policy interface {
	// Name returns a unique identifier for this policy.
	Name() string

	// Authorize checks whether subject is allowed to perform permission on
	// resource. Returns (false, nil) for a clean deny. Returns (false, err)
	// when the decision cannot be made due to an error.
	Authorize(ctx context.Context, subject string, permission Permission, resource string) (bool, error)
}
```

### `auth/credential`

#### `credential.CredentialIssuer`

CredentialIssuer manages the lifecycle of agent credentials. Implementations must be safe for concurrent use.

```go
type CredentialIssuer interface {
	// Issue creates a new credential for the given agent with the specified
	// permissions and TTL.
	Issue(ctx context.Context, agentID string, permissions []string, ttl time.Duration) (*AgentCredential, error)

	// Revoke marks a credential as revoked by its ID.
	Revoke(ctx context.Context, credentialID string) error

	// Get retrieves a credential by its ID. Returns a core.ErrNotFound error
	// if the credential does not exist.
	Get(ctx context.Context, credentialID string) (*AgentCredential, error)
}
```

### `audit`

#### `audit.Logger`

Logger is the write-only interface for emitting audit entries. Implementations must be safe for concurrent use.

```go
type Logger interface {
	// Log records a single audit entry. Implementations may enrich the entry
	// with a generated ID and current timestamp if those fields are empty.
	Log(ctx context.Context, entry Entry) error
}
```

#### `audit.Store`

Store extends Logger with the ability to query historical entries. Implementations must be safe for concurrent use.

```go
type Store interface {
	Logger

	// Query returns entries matching the given filter. An empty filter
	// returns all entries up to Filter.Limit (or all entries when Limit ≤ 0).
	Query(ctx context.Context, filter Filter) ([]Entry, error)
}
```

### `cost`

#### `cost.BudgetChecker`

BudgetChecker evaluates an estimated Usage against a Budget and decides whether to proceed.

```go
type BudgetChecker interface {
	// Check compares the estimated Usage plus the already-consumed usage
	// (determined by querying the underlying Tracker) against the Budget.
	// It returns a BudgetDecision and any error encountered during the check.
	Check(ctx context.Context, budget Budget, estimated Usage) (BudgetDecision, error)
}
```

#### `cost.Tracker`

Tracker records LLM usage and answers aggregate queries about it. Implementations must be safe for concurrent use.

```go
type Tracker interface {
	// Record stores a single usage entry. The context is used for
	// cancellation and tenant propagation.
	Record(ctx context.Context, usage Usage) error

	// Query returns an aggregated Summary for all records matching the given
	// filter. An empty filter returns a summary over all records.
	Query(ctx context.Context, filter Filter) (*Summary, error)
}
```

### `state`

#### `state.Store`

Store is the interface for shared agent state storage. Implementations must be safe for concurrent use.

```go
type Store interface {
	// Get retrieves the value for the given key.
	// Returns nil, nil if the key does not exist.
	Get(ctx context.Context, key string) (any, error)

	// Set stores a value under the given key.
	Set(ctx context.Context, key string, value any) error

	// Delete removes the given key. Deleting a non-existent key is a no-op.
	Delete(ctx context.Context, key string) error

	// Watch returns an iter.Seq2 stream of StateChange notifications for
	// the given key. The subscription is established eagerly before Watch
	// returns, so events produced after this call but before the caller
	// starts iterating are buffered (implementation-defined capacity) and
	// will be delivered on the first iteration.
	//
	// The iterator ends when ctx is cancelled, the store is closed, or
	// the caller breaks out of the loop. Errors (including initial
	// subscription errors such as a closed store) are reported by yielding
	// a zero-value StateChange together with a non-nil error.
	Watch(ctx context.Context, key string) iter.Seq2[StateChange, error]

	// Close releases resources held by the store and closes all watcher
	// channels.
	Close() error
}
```

#### `state.VersionedStore`

VersionedStore extends Store with compare-and-swap semantics and per-key version tracking. Implementations maintain a monotonic revision counter per key, incremented on every mutation.

```go
type VersionedStore interface {
	Store

	// CompareAndSwap atomically sets the value for key only if the current
	// version matches expectedVersion. It returns the new version on success.
	// If the versions do not match, it returns ErrVersionMismatch.
	CompareAndSwap(ctx context.Context, key string, expectedVersion uint64, value any) (newVersion uint64, err error)

	// GetVersioned retrieves the value and current version for the given key.
	// Returns (nil, 0, nil) if the key does not exist.
	GetVersioned(ctx context.Context, key string) (value any, version uint64, err error)
}
```

### `workflow`

#### `workflow.DurableExecutor`

DurableExecutor is the interface for executing and managing durable workflows.

```go
type DurableExecutor interface {
	// Execute starts a new workflow execution.
	Execute(ctx context.Context, fn WorkflowFunc, opts WorkflowOptions) (WorkflowHandle, error)

	// Signal sends a signal to a running workflow.
	Signal(ctx context.Context, workflowID string, signal Signal) error

	// Query retrieves state from a running workflow.
	Query(ctx context.Context, workflowID string, queryType string) (any, error)

	// Cancel requests cancellation of a running workflow.
	Cancel(ctx context.Context, workflowID string) error
}
```

#### `workflow.SignalChannel`

SignalChannel is the interface for sending and receiving signals in workflows. It enables human-in-the-loop interactions by allowing external systems to communicate with running workflows.

```go
type SignalChannel interface {
	// Send transmits a signal to a specific workflow.
	// It returns an error if the signal cannot be delivered.
	Send(ctx context.Context, workflowID string, signal Signal) error

	// Receive blocks until a signal with the specified name is available for the workflow,
	// or the context is canceled/deadline exceeded.
	// It returns nil if no signal is received within the deadline.
	Receive(ctx context.Context, workflowID string, signalName string) (*Signal, error)
}
```

#### `workflow.WorkflowContext`

WorkflowContext extends context.Context with workflow-specific operations. It provides deterministic execution by recording activity results and replaying them during recovery.

```go
type WorkflowContext interface {
	context.Context

	// ExecuteActivity runs an activity within the workflow. Results are
	// recorded for replay during recovery.
	ExecuteActivity(fn ActivityFunc, input any, opts ...ActivityOption) (any, error)

	// ReceiveSignal returns an iterator that yields payloads delivered to
	// the named signal. Iteration ends when the workflow context is
	// canceled. Multiple calls with the same name are independent
	// subscribers and each receives every payload delivered after
	// subscription.
	ReceiveSignal(name string) iter.Seq2[any, error]

	// Sleep pauses the workflow for the given duration. Unlike time.Sleep,
	// this is recorded and replayed correctly during recovery.
	Sleep(d time.Duration) error
}
```

#### `workflow.WorkflowHandle`

WorkflowHandle provides access to a running or completed workflow.

```go
type WorkflowHandle interface {
	// ID returns the workflow identifier.
	ID() string
	// RunID returns the run identifier for this execution.
	RunID() string
	// Status returns the current workflow status.
	Status() WorkflowStatus
	// Result blocks until the workflow completes and returns its result.
	Result(ctx context.Context) (any, error)
}
```

#### `workflow.WorkflowStore`

WorkflowStore is the interface for persisting workflow state. Implementations provide durable storage for workflow execution history.

```go
type WorkflowStore interface {
	// Save persists the workflow state.
	Save(ctx context.Context, state WorkflowState) error

	// Load retrieves the workflow state by ID.
	Load(ctx context.Context, workflowID string) (*WorkflowState, error)

	// List returns workflows matching the filter.
	List(ctx context.Context, filter WorkflowFilter) ([]WorkflowState, error)

	// Delete removes a workflow state by ID.
	Delete(ctx context.Context, workflowID string) error
}
```

## Layer 3 — Capability

### `llm`

#### `llm.ChatModel`

ChatModel is the primary interface for interacting with language models. All LLM providers implement this interface, and the Router, middleware, and structured output layer all compose through it.

```go
type ChatModel interface {
	// Generate sends a batch of messages and returns a complete AI response.
	Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error)

	// Stream sends a batch of messages and returns an iterator of response chunks.
	// Consumers should range over the returned sequence. A non-nil error terminates
	// the stream.
	Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) iter.Seq2[schema.StreamChunk, error]

	// BindTools returns a new ChatModel that includes the given tool definitions
	// in every request. The original model is not modified.
	BindTools(tools []schema.ToolDefinition) ChatModel

	// ModelID returns the identifier of the underlying model (e.g. "gpt-4o").
	ModelID() string
}
```

#### `llm.ContextManager`

ContextManager fits a sequence of messages within a token budget. Different strategies (truncation, sliding window, etc.) implement this interface to manage context window limits.

```go
type ContextManager interface {
	// Fit trims or transforms msgs so the total token count does not exceed budget.
	// The returned messages preserve conversation coherence as much as possible.
	Fit(ctx context.Context, msgs []schema.Message, budget int) ([]schema.Message, error)
}
```

#### `llm.ModelSelector`

ModelSelector selects which ChatModel to use for a given request.

```go
type ModelSelector interface {
	// Select picks a model from the available models for the given messages.
	Select(ctx context.Context, models []ChatModel, msgs []schema.Message) (ChatModel, error)
}
```

#### `llm.Tokenizer`

Tokenizer provides token counting and encoding/decoding for text. Providers may supply model-specific tokenizers; SimpleTokenizer is a built-in approximation suitable for budget estimation.

```go
type Tokenizer interface {
	// Count returns the approximate number of tokens in text.
	Count(text string) int

	// CountMessages returns the approximate total token count for a slice
	// of messages, including any per-message overhead.
	CountMessages(msgs []schema.Message) int

	// Encode converts text to a sequence of token IDs.
	Encode(text string) []int

	// Decode converts token IDs back to text.
	Decode(tokens []int) string
}
```

### `llm/context`

#### `context.ContextStep`

ContextStep is a single step in the context engineering pipeline. Each step transforms the context items in some way.

```go
type ContextStep interface {
	// Process transforms the context items and returns the result.
	Process(ctx gocontext.Context, items []ContextItem) ([]ContextItem, error)
	// Name returns the step name for logging and debugging.
	Name() string
}
```

#### `context.Executor`

Executor orchestrates a sequence of ContextSteps to prepare optimal LLM context.

```go
type Executor interface {
	// Execute runs all pipeline steps in order.
	Execute(ctx gocontext.Context, input PipelineInput) (PipelineOutput, error)
}
```

### `llm/routing`

#### `routing.BudgetEnforcer`

BudgetEnforcer tracks and enforces cost budgets.

```go
type BudgetEnforcer interface {
	// Check returns whether the estimated cost is within budget.
	Check(ctx context.Context, estimatedCost float64) (bool, error)
	// CheckAndReserve atomically verifies that estimatedCost fits within the
	// remaining budget and reserves it. Returns (true, nil) when reserved.
	// Callers should later call Record with the actual cost; the difference
	// between reserved and actual spend can be reconciled by callers if
	// needed. This prevents TOCTOU races across concurrent callers.
	CheckAndReserve(ctx context.Context, estimatedCost float64) (bool, error)
	// Record records an actual cost expenditure.
	Record(ctx context.Context, cost float64) error
}
```

#### `routing.ComplexityClassifier`

ComplexityClassifier determines the complexity of a request.

```go
type ComplexityClassifier interface {
	// Classify returns the estimated complexity tier for the given messages.
	Classify(ctx context.Context, msgs []schema.Message) (ModelTier, error)
}
```

#### `routing.CostRouter`

CostRouter selects a model based on complexity and cost considerations.

```go
type CostRouter interface {
	// SelectModel chooses the best model for the given messages and tier.
	SelectModel(ctx context.Context, msgs []schema.Message) (ModelSelection, error)
}
```

### `tool`

#### `tool.MCPRegistry`

MCPRegistry provides discovery of MCP servers. Implementations may search a local directory, a remote registry service, or a static configuration.

```go
type MCPRegistry interface {
	// Search finds MCP servers matching the given query string.
	Search(ctx context.Context, query string) ([]MCPServerInfo, error)

	// Discover returns all known MCP servers.
	Discover(ctx context.Context) ([]MCPServerInfo, error)
}
```

#### `tool.Tool`

Tool is the interface that all tools implement. A tool has a name, description, JSON Schema for its input parameters, and an Execute method that performs the tool's action given a parsed input map.

```go
type Tool interface {
	// Name returns the unique identifier for this tool.
	Name() string

	// Description returns a human-readable description of what the tool does.
	// This is provided to the LLM to help it decide when to use the tool.
	Description() string

	// InputSchema returns a JSON Schema (as a map) describing the expected
	// input parameters for this tool.
	InputSchema() map[string]any

	// Execute runs the tool with the given input parameters and returns a
	// multimodal result. The input map corresponds to the InputSchema.
	Execute(ctx context.Context, input map[string]any) (*Result, error)
}
```

### `tool/computeruse`

#### `computeruse.BrowserBackend`

BrowserBackend is the interface for browser automation. Implementations wrap headless browsers or browser automation frameworks.

```go
type BrowserBackend interface {
	// Navigate loads the given URL in the browser.
	Navigate(ctx context.Context, url string) error

	// Screenshot captures the current page as PNG bytes.
	Screenshot(ctx context.Context) ([]byte, error)

	// Click performs a mouse click at the given coordinates.
	Click(ctx context.Context, x, y int) error

	// Type enters text into the currently focused element.
	Type(ctx context.Context, text string) error
}
```

#### `computeruse.ComputerAction`

ComputerAction is the interface for performing screen interactions. Implementations drive actual computer or browser automation.

```go
type ComputerAction interface {
	// Execute performs the requested action and returns the result.
	Execute(ctx context.Context, req ActionRequest) (*ActionResult, error)

	// Screenshot captures the current screen state as PNG bytes.
	Screenshot(ctx context.Context) ([]byte, error)
}
```

#### `computeruse.ExtendedBrowser`

ExtendedBrowser is an optional interface for browsers that support JavaScript execution and scrolling. Check via type assertion.

```go
type ExtendedBrowser interface {
	BrowserBackend

	// Scroll scrolls the page by the given delta (positive = down).
	Scroll(ctx context.Context, x, y, delta int) error

	// ExecuteJS runs JavaScript in the browser and returns the result.
	ExecuteJS(ctx context.Context, script string) (string, error)
}
```

### `tool/learning`

#### `learning.CodeExecutor`

CodeExecutor defines the interface for executing dynamically generated tool code. Implementations control the sandboxing and safety guarantees of code execution.

```go
type CodeExecutor interface {
	// Execute runs the given code with the provided JSON input string and returns
	// the output as a string. The code is expected to be a valid Go function body.
	Execute(ctx context.Context, code string, input string) (string, error)
}
```

### `tool/sandbox`

#### `sandbox.Sandbox`

Sandbox is the interface for sandboxed code execution. Implementations execute arbitrary code in an isolated environment and return the result.

```go
type Sandbox interface {
	// Execute runs code in the sandbox with the given configuration and
	// returns the execution result. The context controls cancellation and
	// deadline; if the context is cancelled the execution is aborted.
	Execute(ctx context.Context, code string, cfg SandboxConfig) (ExecutionResult, error)

	// Close releases all resources held by the sandbox. After Close returns,
	// Execute must not be called.
	Close(ctx context.Context) error
}
```

### `memory`

#### `memory.GraphStore`

GraphStore is the interface for graph-based memory storage. Implementations provide entity-relationship storage for structured knowledge representation. All methods must be safe for concurrent use.

```go
type GraphStore interface {
	// AddEntity adds or updates an entity in the graph.
	AddEntity(ctx context.Context, entity Entity) error

	// AddRelation creates a directed relationship between two entities.
	AddRelation(ctx context.Context, from, to, relation string, props map[string]any) error

	// Query executes a query string (e.g., Cypher) against the graph and
	// returns matching entities and relations.
	Query(ctx context.Context, query string) ([]GraphResult, error)

	// Neighbors returns all entities and relations within the given depth
	// from the specified entity.
	Neighbors(ctx context.Context, entityID string, depth int) ([]Entity, []Relation, error)
}
```

#### `memory.Memory`

Memory is the primary interface for agent memory. Implementations provide the ability to save conversation turns, load relevant history, and search over stored documents. All methods must be safe for concurrent use.

```go
type Memory interface {
	// Save persists an input/output message pair from a conversation turn.
	Save(ctx context.Context, input, output schema.Message) error

	// Load retrieves messages relevant to the given query. The returned
	// messages are ordered by relevance or recency depending on the
	// implementation.
	Load(ctx context.Context, query string) ([]schema.Message, error)

	// Search finds documents relevant to the given query, returning at most
	// k results. This is primarily used by the archival tier.
	Search(ctx context.Context, query string, k int) ([]schema.Document, error)

	// Clear removes all stored data from this memory instance.
	Clear(ctx context.Context) error
}
```

#### `memory.MessageStore`

MessageStore is the interface for storing and retrieving conversation messages. Implementations provide the persistence backend for recall memory. All methods must be safe for concurrent use.

```go
type MessageStore interface {
	// Append adds a message to the store.
	Append(ctx context.Context, msg schema.Message) error

	// Search finds messages matching the query, returning at most k results.
	// The matching strategy (substring, semantic, etc.) depends on the
	// implementation.
	Search(ctx context.Context, query string, k int) ([]schema.Message, error)

	// All returns all stored messages in chronological order.
	All(ctx context.Context) ([]schema.Message, error)

	// Clear removes all messages from the store.
	Clear(ctx context.Context) error
}
```

#### `memory.TemporalGraphStore`

TemporalGraphStore extends GraphStore with bi-temporal query capabilities. It supports querying the knowledge graph as it was at a specific point in time, invalidating relations, and retrieving the full history of relations between entities.

```go
type TemporalGraphStore interface {
	GraphStore

	// QueryAsOf returns entities and relations valid at a specific point in time.
	// Only relations where ValidAt <= validTime and (InvalidAt is nil or InvalidAt > validTime)
	// are included. The opts parameter can be used to limit results.
	QueryAsOf(ctx context.Context, query string, validTime time.Time, opts ...QueryOption) ([]Entity, []Relation, error)

	// InvalidateRelation marks a relation as no longer valid at the given time.
	// It sets the InvalidAt field to invalidAt and ExpiredAt to the current system time.
	InvalidateRelation(ctx context.Context, relationID string, invalidAt time.Time) error

	// History returns all versions of relations between two entities, including
	// invalidated ones, ordered by ValidAt ascending.
	History(ctx context.Context, fromID, toID string) ([]Relation, error)
}
```

### `memory/associative`

#### `associative.NoteStore`

NoteStore is the persistence interface for notes in associative memory. All methods must be safe for concurrent use.

```go
type NoteStore interface {
	// Add stores a new note. Returns an error if a note with the same ID exists.
	Add(ctx context.Context, note *schema.Note) error

	// Get retrieves a note by ID. Returns a core.ErrNotFound error if the note
	// does not exist.
	Get(ctx context.Context, id string) (*schema.Note, error)

	// Update replaces a note in the store. Returns a core.ErrNotFound error if
	// the note does not exist.
	Update(ctx context.Context, note *schema.Note) error

	// Delete removes a note by ID. Returns a core.ErrNotFound error if the note
	// does not exist.
	Delete(ctx context.Context, id string) error

	// Search finds notes whose embeddings are most similar to the query vector,
	// returning at most k results ordered by descending similarity.
	Search(ctx context.Context, queryVec []float32, k int) ([]*schema.Note, error)

	// List returns all notes in the store ordered by creation time (oldest first).
	List(ctx context.Context) ([]*schema.Note, error)
}
```

### `memory/consolidation`

#### `consolidation.Compressor`

Compressor reduces the size of memory records while preserving their essential information. Implementations must be safe for concurrent use.

```go
type Compressor interface {
	// Compress takes a set of records and returns compressed versions.
	// The returned slice must have the same length as the input.
	Compress(ctx context.Context, records []Record) ([]Record, error)
}
```

#### `consolidation.ConsolidationStore`

ConsolidationStore is the storage interface used by the consolidation worker to read, update, and delete memory records. Implementations must be safe for concurrent use.

```go
type ConsolidationStore interface {
	// ListRecords returns up to limit records starting from offset.
	ListRecords(ctx context.Context, offset, limit int) ([]Record, error)

	// DeleteRecords removes the records with the given IDs.
	DeleteRecords(ctx context.Context, ids []string) error

	// UpdateRecords persists changes to the given records.
	UpdateRecords(ctx context.Context, records []Record) error

	// RecordAccess increments the access counter and updates the last
	// accessed timestamp for the given record ID.
	RecordAccess(ctx context.Context, id string) error
}
```

#### `consolidation.Evaluator`

Evaluator evaluates a set of records and returns a decision for each one. Implementations must be safe for concurrent use.

```go
type Evaluator interface {
	// Evaluate scores the given records and returns a decision (keep, prune,
	// or compress) for each record. The returned slice must have the same
	// length as the input.
	Evaluate(ctx context.Context, records []Record) ([]Decision, error)
}
```

### `memory/procedural`

#### `procedural.SkillExtractor`

SkillExtractor extracts procedural skills from execution traces. It abstracts the input/output of a successful task completion into a reusable skill definition.

```go
type SkillExtractor interface {
	// Extract analyzes an input/output pair and optional metadata to produce
	// a Skill. Returns nil if no meaningful skill can be extracted.
	Extract(ctx context.Context, input, output string, metadata map[string]any) (*schema.Skill, error)
}
```

### `memory/rl`

#### `rl.Decider`

Decider decides which memory action to take given the current observation features. Implementations range from rule-based heuristics to trained neural network models.

```go
type Decider interface {
	// Decide selects a MemoryAction and returns a confidence score in [0, 1].
	// The confidence indicates how certain the policy is about the chosen action.
	Decide(ctx context.Context, features PolicyFeatures) (MemoryAction, float64, error)
}
```

#### `rl.Deleter`

Deleter is an optional capability interface: memories that support granular entry deletion by ID. PolicyMemory routes ActionDelete through this interface when available. Memories that do not implement it cause ActionDelete to return an error explaining the missing capability.

```go
type Deleter interface {
	Delete(ctx context.Context, id string) error
}
```

#### `rl.FeatureExtractor`

FeatureExtractor computes PolicyFeatures from the current memory state and the incoming messages. Implementations may query the underlying memory to assess similarity, count entries, etc.

```go
type FeatureExtractor interface {
	// Extract derives observation features from the memory state and the
	// input/output messages being saved.
	Extract(ctx context.Context, mem memory.Memory, input, output schema.Message) (PolicyFeatures, error)
}
```

#### `rl.Rewarder`

Rewarder computes per-step rewards for an episode. The returned slice must have the same length as episode.Steps. Implementations map task-level outcomes to step-level credit assignments.

```go
type Rewarder interface {
	// Compute returns a reward value for each step in the episode.
	// Positive rewards encourage the action; negative rewards discourage it.
	Compute(ctx context.Context, episode Episode) ([]float64, error)
}
```

#### `rl.Sizer`

Sizer is an optional capability interface: memories that can report their current total entry count. The DefaultFeatureExtractor uses Size() when available so that HeuristicPolicy can make informed ActionDelete decisions based on MaxStoreSize.

```go
type Sizer interface {
	Size(ctx context.Context) (int, error)
}
```

### `memory/shared`

#### `shared.SharedStore`

SharedStore is the storage backend for shared memory fragments. Implementations must be safe for concurrent use.

```go
type SharedStore interface {
	// WriteFragment persists a fragment. If the fragment's key already exists,
	// the store updates it in place and increments the version.
	WriteFragment(ctx context.Context, frag *Fragment) error

	// ReadFragment retrieves a fragment by key. Returns a core.ErrNotFound
	// error if the key does not exist.
	ReadFragment(ctx context.Context, key string) (*Fragment, error)

	// ListFragments returns all fragments matching the given scope. If scope
	// is empty, all fragments are returned.
	ListFragments(ctx context.Context, scope Scope) ([]*Fragment, error)

	// DeleteFragment removes the fragment with the given key. Deleting a
	// non-existent key is a no-op.
	DeleteFragment(ctx context.Context, key string) error

	// UpdateProvenance updates only the provenance field of the fragment
	// identified by key, without bumping the version or applying conflict
	// policy. This is used internally after a write to attach a provenance
	// hash that reflects the final stored content (for AppendOnly the
	// post-write content is the accumulated string, not the delta).
	// Returns core.ErrNotFound if the key does not exist.
	UpdateProvenance(ctx context.Context, key string, prov *Provenance) error
}
```

### `memory/temporal`

#### `temporal.ConflictResolver`

ConflictResolver determines how to handle temporal conflicts between relations. When a new relation is added that may contradict existing relations, the resolver decides which relations should be invalidated.

```go
type ConflictResolver interface {
	// Resolve examines the newRelation against a set of candidate relations and
	// returns the list of candidates that should be invalidated. The returned
	// relations will have their InvalidAt and ExpiredAt fields set accordingly.
	Resolve(ctx context.Context, newRelation *memory.Relation, candidates []memory.Relation) ([]memory.Relation, error)
}
```

### `rag/embedding`

#### `embedding.Embedder`

Embedder converts text into dense vector embeddings. Implementations must be safe for concurrent use.

```go
type Embedder interface {
	// Embed produces embeddings for a batch of texts. The returned slice has
	// the same length as texts, with each element being a float32 vector of
	// Dimensions() length.
	Embed(ctx context.Context, texts []string) ([][]float32, error)

	// EmbedSingle is a convenience method that embeds a single text and
	// returns its vector.
	EmbedSingle(ctx context.Context, text string) ([]float32, error)

	// Dimensions returns the dimensionality of the embedding vectors
	// produced by this embedder.
	Dimensions() int
}
```

#### `embedding.MultiVectorEmbedder`

MultiVectorEmbedder produces per-token embeddings for late interaction retrieval models such as ColBERT and ColPali. Unlike [Embedder], which produces a single vector per text, MultiVectorEmbedder returns a slice of token-level vectors for each input text.

```go
type MultiVectorEmbedder interface {
	// EmbedMulti produces per-token embeddings for a batch of texts. The
	// returned slice has the same length as texts, where each element is a
	// slice of token-level float32 vectors.
	EmbedMulti(ctx context.Context, texts []string) ([][][]float32, error)

	// TokenDimensions returns the dimensionality of the per-token embedding
	// vectors produced by this embedder.
	TokenDimensions() int
}
```

### `rag/loader`

#### `loader.DocumentLoader`

DocumentLoader loads content from a source and returns documents. Implementations handle specific formats (text, JSON, CSV, etc.) and register themselves via init().

```go
type DocumentLoader interface {
	// Load reads content from the given source (typically a file path or URL)
	// and returns a slice of documents.
	Load(ctx context.Context, source string) ([]schema.Document, error)
}
```

#### `loader.Transformer`

Transformer applies a transformation to a single document. Transformers are used in pipelines to enrich or modify documents after loading (e.g., adding metadata, cleaning content).

```go
type Transformer interface {
	// Transform applies a transformation to the given document and returns
	// the modified document.
	Transform(ctx context.Context, doc schema.Document) (schema.Document, error)
}
```

### `rag/retriever`

#### `retriever.BM25Searcher`

BM25Searcher performs keyword-based BM25 search. Implementations may use an external search engine (Elasticsearch, Typesense) or an in-memory index.

```go
type BM25Searcher interface {
	// Search returns documents matching the query ranked by BM25 score.
	Search(ctx context.Context, query string, k int) ([]schema.Document, error)
}
```

#### `retriever.Decomposer`

Decomposer breaks a complex query into sub-questions. Each sub-question includes a routing hint indicating which named retriever should handle it.

```go
type Decomposer interface {
	// Decompose splits query into sub-questions, each tagged with a retriever
	// name from the available set.
	Decompose(ctx context.Context, query string, available []string) ([]SubQuestion, error)
}
```

#### `retriever.Fuser`

Fuser combines multiple result sets into a single ranked list.

```go
type Fuser interface {
	// Fuse merges multiple result sets into a single ranked list of documents
	// with updated Score fields.
	Fuse(ctx context.Context, results [][]schema.Document) ([]schema.Document, error)
}
```

#### `retriever.Reranker`

Reranker re-scores a set of documents for relevance to a query. Cross-encoder models and LLM-based scorers implement this interface.

```go
type Reranker interface {
	// Rerank re-scores the given documents for the query and returns them
	// in descending order of relevance with updated Score fields.
	Rerank(ctx context.Context, query string, docs []schema.Document) ([]schema.Document, error)
}
```

#### `retriever.Retriever`

Retriever searches for documents relevant to a query. Implementations must be safe for concurrent use.

```go
type Retriever interface {
	// Retrieve returns documents relevant to the query, ordered by
	// decreasing relevance. Options configure top-k, threshold, and
	// metadata filters.
	Retrieve(ctx context.Context, query string, opts ...Option) ([]schema.Document, error)
}
```

#### `retriever.WebSearcher`

WebSearcher performs web search queries. Implementations may wrap external search APIs (Google, Bing, Tavily, etc.).

```go
type WebSearcher interface {
	// Search returns documents from the web matching the query.
	Search(ctx context.Context, query string, k int) ([]schema.Document, error)
}
```

### `rag/retriever/colbert`

#### `colbert.ColBERTIndex`

ColBERTIndex stores pre-computed per-token document embeddings and supports search by MaxSim scoring. Implementations must be safe for concurrent use.

```go
type ColBERTIndex interface {
	// Add stores per-token embeddings for a document. If a document with the
	// same ID already exists, its embeddings are replaced.
	Add(ctx context.Context, id string, tokenVecs [][]float32) error

	// Search finds the top-k documents most similar to the query token
	// vectors, scored by MaxSim. Results are ordered by decreasing score.
	Search(ctx context.Context, queryVecs [][]float32, k int) ([]IndexResult, error)
}
```

### `rag/retriever/raptor`

#### `raptor.Clusterer`

Clusterer groups embedding vectors into clusters. Each returned cluster is a slice of indices into the original embeddings slice.

```go
type Clusterer interface {
	// Cluster partitions embeddings into groups and returns the indices of
	// each group. The outer slice has one entry per cluster; the inner slice
	// holds indices into the embeddings parameter.
	Cluster(ctx context.Context, embeddings [][]float32) ([][]int, error)
}
```

#### `raptor.Summarizer`

Summarizer produces a single summary from multiple text chunks. It is called once per cluster at each tree level during the RAPTOR build process.

```go
type Summarizer interface {
	// Summarize combines the provided texts into a single coherent summary.
	Summarize(ctx context.Context, texts []string) (string, error)
}
```

### `rag/retriever/structured`

#### `structured.QueryExecutor`

QueryExecutor runs a database query and returns its results.

```go
type QueryExecutor interface {
	// Execute runs the given query string against the target database and
	// returns the result rows as a slice of maps.
	Execute(ctx context.Context, query string) ([]map[string]any, error)
}
```

#### `structured.QueryGenerator`

QueryGenerator translates a natural-language question into a database query using the provided schema information.

```go
type QueryGenerator interface {
	// Generate produces a database query string from a question and schema.
	// The returned string should be valid Cypher or SQL depending on the
	// schema dialect.
	Generate(ctx context.Context, question string, info SchemaInfo) (string, error)
}
```

#### `structured.ResultEvaluator`

ResultEvaluator judges how relevant query results are to the original question. The score is in the range [0, 1] where 1 means perfectly relevant.

```go
type ResultEvaluator interface {
	// Evaluate returns a relevance score for the given results with respect
	// to the question.
	Evaluate(ctx context.Context, question string, results []map[string]any) (float64, error)
}
```

### `rag/splitter`

#### `splitter.TextSplitter`

TextSplitter splits text content into smaller chunks. Implementations determine chunk boundaries based on different strategies (characters, tokens, document structure).

```go
type TextSplitter interface {
	// Split divides text into a slice of chunks.
	Split(ctx context.Context, text string) ([]string, error)

	// SplitDocuments splits each document's content and returns new documents
	// for each chunk, preserving and augmenting the original metadata.
	SplitDocuments(ctx context.Context, docs []schema.Document) ([]schema.Document, error)
}
```

### `rag/vectorstore`

#### `vectorstore.VectorStore`

VectorStore stores document embeddings and supports similarity search. Implementations must be safe for concurrent use.

```go
type VectorStore interface {
	// Add inserts documents with their corresponding embeddings into the store.
	// The docs and embeddings slices must have the same length.
	Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error

	// Search finds the k most similar documents to the query vector.
	// Results are returned in descending order of similarity with their
	// Score field populated.
	Search(ctx context.Context, query []float32, k int, opts ...SearchOption) ([]schema.Document, error)

	// Delete removes documents with the given IDs from the store.
	Delete(ctx context.Context, ids []string) error
}
```

### `voice`

#### `voice.ActivityDetector`

ActivityDetector detects voice activity in audio data.

```go
type ActivityDetector interface {
	// DetectActivity analyses an audio chunk and returns whether speech is
	// present along with an event type indicating state transitions.
	DetectActivity(ctx context.Context, audio []byte) (ActivityResult, error)
}
```

#### `voice.FrameProcessor`

FrameProcessor processes a stream of frames. Each processor is a pure transformer over an iter.Seq2[Frame, error] stream: it consumes input frames lazily and yields output frames to its caller. A fatal error is delivered by yielding a zero Frame paired with the error and then ending the iterator.

```go
type FrameProcessor interface {
	// Process returns an output stream derived from the input stream.
	// The returned iterator is single-use and safe for exactly one consumer.
	Process(ctx context.Context, in iter.Seq2[Frame, error]) iter.Seq2[Frame, error]
}
```

#### `voice.LLMProcessor`

LLMProcessor is a local interface for LLM integration in the voice pipeline. It wraps an LLM as a FrameProcessor that converts text frames to text frames.

```go
type LLMProcessor interface {
	FrameProcessor
}
```

#### `voice.S2SProcessor`

S2SProcessor is a local interface for speech-to-speech processors. Concrete implementations live in voice/s2s/.

```go
type S2SProcessor interface {
	FrameProcessor
}
```

#### `voice.SwitchPolicy`

SwitchPolicy determines when a hybrid pipeline should switch between S2S and cascade modes.

```go
type SwitchPolicy interface {
	// ShouldSwitch returns true if the pipeline should switch to the
	// other mode given the current state.
	ShouldSwitch(ctx context.Context, state PipelineState) bool
}
```

#### `voice.Synthesizer`

Synthesizer is a local interface for text-to-speech processors. This avoids importing the voice/tts package directly.

```go
type Synthesizer interface {
	FrameProcessor
}
```

#### `voice.Transcriber`

Transcriber is a local interface for speech-to-text processors. This avoids importing the voice/stt package directly.

```go
type Transcriber interface {
	FrameProcessor
}
```

#### `voice.Transport`

Transport is a local interface for audio input/output transport. Concrete implementations live in voice/transport/.

```go
type Transport interface {
	// Recv returns an iterator of incoming audio frames from the client.
	// Transport-level errors are delivered via the iterator's second element;
	// a non-nil error terminates the stream.
	Recv(ctx context.Context) iter.Seq2[Frame, error]

	// Send writes an outgoing audio frame to the client.
	Send(ctx context.Context, frame Frame) error

	// Close shuts down the transport connection.
	Close() error
}
```

### `voice/diarization`

#### `diarization.Diarizer`

Diarizer identifies different speakers within audio data.

```go
type Diarizer interface {
	// Diarize processes audio data and returns speaker segments.
	Diarize(ctx context.Context, audio []byte, opts ...DiarizeOption) ([]SpeakerSegment, error)
}
```

#### `diarization.SpeakerTracker`

SpeakerTracker maintains speaker identity across multiple diarization calls, allowing consistent speaker IDs across a session.

```go
type SpeakerTracker interface {
	// Track assigns consistent speaker IDs to segments based on prior observations.
	Track(ctx context.Context, segments []SpeakerSegment) ([]SpeakerSegment, error)
	// Reset clears all tracked speaker state.
	Reset(ctx context.Context) error
}
```

### `voice/s2s`

#### `s2s.S2S`

S2S is the speech-to-speech interface. Implementations provide bidirectional audio streaming via native provider protocols (e.g., OpenAI Realtime API, Gemini Live, Amazon Nova).

```go
type S2S interface {
	// Start initiates a new S2S session with the provider.
	Start(ctx context.Context, opts ...Option) (Session, error)
}
```

#### `s2s.Session`

Session represents an active bidirectional audio session with an S2S provider.

```go
type Session interface {
	SessionSender
	SessionReceiver
	SessionControl
}
```

#### `s2s.SessionControl`

SessionControl governs session lifecycle — interruption and termination.

```go
type SessionControl interface {
	// Interrupt signals that the user has interrupted the model's output.
	Interrupt(ctx context.Context) error

	// Close terminates the session and releases resources.
	Close() error
}
```

#### `s2s.SessionReceiver`

SessionReceiver delivers output events from the provider to the client.

```go
type SessionReceiver interface {
	// Recv returns an iterator of session events. The iterator ends when
	// the session closes or ctx is cancelled. Transport-level errors are
	// delivered as the second element of the yielded pair; event-level
	// errors surface as SessionEvent{Type: EventError} values.
	Recv(ctx context.Context) iter.Seq2[SessionEvent, error]
}
```

#### `s2s.SessionSender`

SessionSender carries input (audio, text, tool results) from the client to the S2S provider. Split from Session so consumers that only send (e.g., a push-to-talk UI controller) can depend on the narrowest surface.

```go
type SessionSender interface {
	// SendAudio sends an audio chunk to the provider.
	SendAudio(ctx context.Context, audio []byte) error

	// SendText sends a text message to the provider (for instructions or prompts).
	SendText(ctx context.Context, text string) error

	// SendToolResult sends a tool execution result back to the model.
	SendToolResult(ctx context.Context, result schema.ToolResult) error
}
```

### `voice/stt`

#### `stt.STT`

STT is the speech-to-text interface. Implementations convert audio data to text, supporting both batch and streaming modes.

```go
type STT interface {
	// Transcribe converts a complete audio buffer to text.
	Transcribe(ctx context.Context, audio []byte, opts ...Option) (string, error)

	// TranscribeStream converts a streaming audio source to a stream of
	// transcript events. Partial results are emitted as they become available.
	TranscribeStream(ctx context.Context, audioStream iter.Seq2[[]byte, error], opts ...Option) iter.Seq2[TranscriptEvent, error]
}
```

### `voice/telephony`

#### `telephony.CallRouter`

CallRouter routes incoming calls to appropriate handlers.

```go
type CallRouter interface {
	// Route determines how to handle an incoming call based on its metadata.
	Route(ctx context.Context, call IncomingCall) (CallAction, error)
}
```

#### `telephony.SIPEndpoint`

SIPEndpoint represents a SIP connection endpoint for making and receiving calls.

```go
type SIPEndpoint interface {
	// Connect establishes a connection to the SIP server.
	Connect(ctx context.Context) error
	// Disconnect closes the SIP connection.
	Disconnect(ctx context.Context) error
	// Status returns the current connection status.
	Status(ctx context.Context) (EndpointStatus, error)
}
```

#### `telephony.TelephonyProvider`

TelephonyProvider is the main interface for telephony service providers.

```go
type TelephonyProvider interface {
	// PlaceCall initiates an outbound call.
	PlaceCall(ctx context.Context, req CallRequest) (*Call, error)
	// HangUp terminates an active call.
	HangUp(ctx context.Context, callID string) error
}
```

### `voice/transport`

#### `transport.AudioTransport`

AudioTransport is the interface for bidirectional audio I/O between clients and the voice pipeline. Implementations handle the underlying transport protocol (WebSocket, WebRTC via LiveKit/Daily, etc.).

```go
type AudioTransport interface {
	// Recv returns an iterator of incoming audio frames from the remote client.
	// Transport-level errors are delivered via the iterator's second element;
	// a non-nil error terminates the stream. The iterator ends when the
	// connection closes or ctx is cancelled.
	Recv(ctx context.Context) iter.Seq2[voice.Frame, error]

	// Send writes an outgoing frame to the remote client.
	Send(ctx context.Context, frame voice.Frame) error

	// AudioOut returns a writer for raw audio output. This is useful for
	// piping synthesized audio directly without frame wrapping.
	AudioOut() io.Writer

	// Close shuts down the transport and releases resources.
	Close() error
}
```

### `voice/tts`

#### `tts.TTS`

TTS is the text-to-speech interface. Implementations convert text to audio, supporting both batch and streaming modes.

```go
type TTS interface {
	// Synthesize converts text to a complete audio buffer.
	Synthesize(ctx context.Context, text string, opts ...Option) ([]byte, error)

	// SynthesizeStream converts a streaming text source to a stream of audio
	// chunks. Audio chunks are emitted as they become available, enabling
	// low-latency playback.
	SynthesizeStream(ctx context.Context, textStream iter.Seq2[string, error], opts ...Option) iter.Seq2[[]byte, error]
}
```

### `guard`

#### `guard.Guard`

Guard validates content at any stage of the safety pipeline. Implementations must be safe for concurrent use.

```go
type Guard interface {
	// Name returns a unique identifier for this guard, used in logging and
	// error reporting.
	Name() string

	// Validate checks the given input and returns a result indicating whether
	// the content is allowed, along with an optional modified version.
	Validate(ctx context.Context, input GuardInput) (GuardResult, error)
}
```

### `guard/degradation`

#### `degradation.PolicyEvaluator`

PolicyEvaluator evaluates a severity score and determines the appropriate autonomy level for agent operation.

```go
type PolicyEvaluator interface {
	// Evaluate maps a severity score in [0.0, 1.0] to an AutonomyLevel.
	Evaluate(severity float64) AutonomyLevel
}
```

### `guard/memory`

#### `memory.AnomalyDetector`

AnomalyDetector inspects content destined for memory and reports whether it appears anomalous. Implementations must be safe for concurrent use.

```go
type AnomalyDetector interface {
	// Name returns a unique identifier for this detector.
	Name() string

	// Detect analyzes content and returns an AnomalyResult describing
	// whether the content is suspicious.
	Detect(ctx context.Context, content string) (AnomalyResult, error)
}
```

### `prompt`

#### `prompt.PromptManager`

PromptManager provides versioned access to prompt templates. Implementations load templates from various backends (filesystem, database, etc.) and support rendering templates into message sequences.

```go
type PromptManager interface {
	// Get retrieves a template by name and version. If version is empty,
	// the latest version is returned.
	Get(name, version string) (*Template, error)

	// Render retrieves a template by name (latest version), renders it with
	// the given variables, and returns the result as a slice of schema.Message.
	// The rendered text is returned as a single SystemMessage.
	Render(name string, vars map[string]any) ([]schema.Message, error)

	// List returns summary information for all available templates.
	List() []TemplateInfo
}
```

### `cache`

#### `cache.Cache`

Cache is the interface that all cache backends implement. It provides basic key-value operations with TTL support.

```go
type Cache interface {
	// Get retrieves a value by key. Returns the value, whether the key was
	// found, and any error. A missing key returns (nil, false, nil).
	Get(ctx context.Context, key string) (any, bool, error)

	// Set stores a value with the given key and TTL. A zero TTL means the
	// entry uses the cache's default TTL. A negative TTL means no expiration.
	Set(ctx context.Context, key string, value any, ttl time.Duration) error

	// Delete removes a key from the cache. Deleting a non-existent key is a no-op.
	Delete(ctx context.Context, key string) error

	// Clear removes all entries from the cache.
	Clear(ctx context.Context) error
}
```

### `eval`

#### `eval.Augmenter`

Augmenter generates additional evaluation samples from an existing sample. Implementations may use an LLM to paraphrase, add noise, or create adversarial variants for more robust evaluation.

```go
type Augmenter interface {
	// Augment generates one or more new samples derived from the given sample.
	Augment(ctx context.Context, sample EvalSample) ([]EvalSample, error)
}
```

#### `eval.Metric`

Metric is the interface that all evaluation metrics implement. A Metric scores a single evaluation sample and returns a float64 in [0, 1].

```go
type Metric interface {
	// Name returns the unique name of this metric (e.g., "faithfulness").
	Name() string

	// Score evaluates a single sample and returns a score in [0.0, 1.0].
	// Higher scores indicate better quality for quality metrics.
	Score(ctx context.Context, sample EvalSample) (float64, error)
}
```

### `eval/clustering`

#### `clustering.ConversationClusterer`

ConversationClusterer groups conversations into clusters based on similarity.

```go
type ConversationClusterer interface {
	// Cluster groups the given conversations into clusters.
	Cluster(ctx context.Context, convs []Conversation) ([]Cluster, error)
}
```

#### `clustering.PatternDetector`

PatternDetector identifies recurring patterns across conversations.

```go
type PatternDetector interface {
	// Detect finds patterns in a set of conversations and returns them.
	Detect(ctx context.Context, convs []Conversation) ([]Pattern, error)
}
```

#### `clustering.SimilarityScorer`

SimilarityScorer computes the similarity between two conversations. The result must be in [0.0, 1.0] where 1.0 means identical.

```go
type SimilarityScorer interface {
	// Score returns a similarity score in [0.0, 1.0] between two conversations.
	Score(ctx context.Context, a, b Conversation) (float64, error)
}
```

### `eval/redteam`

#### `redteam.AttackPattern`

AttackPattern generates adversarial prompts for a specific attack category.

```go
type AttackPattern interface {
	// Category returns the attack category this pattern belongs to.
	Category() AttackCategory

	// Generate produces a set of adversarial prompts for testing.
	Generate(ctx context.Context) ([]string, error)
}
```

### `eval/simulation`

#### `simulation.SimEnvironment`

SimEnvironment is the interface for resettable simulation environments. Implementations model the world that agents interact with during simulation-based testing.

```go
type SimEnvironment interface {
	// Reset initializes or resets the environment to its starting state
	// and returns the initial observation.
	Reset(ctx context.Context) (*Observation, error)

	// Step applies an action to the environment and returns the resulting
	// observation. The action is a free-form string interpreted by the
	// environment.
	Step(ctx context.Context, action string) (*Observation, error)

	// Observe returns the current observation without modifying state.
	Observe(ctx context.Context) (*Observation, error)

	// Close releases any resources held by the environment.
	Close() error
}
```

### `eval/trajectory`

#### `trajectory.TrajectoryMetric`

TrajectoryMetric is the interface that all trajectory evaluation metrics implement. A TrajectoryMetric scores a complete agent trajectory and returns detailed per-step and overall scores.

```go
type TrajectoryMetric interface {
	// Name returns the unique name of this metric (e.g., "tool_selection").
	Name() string

	// ScoreTrajectory evaluates a single trajectory and returns a detailed score.
	// The overall score should be in [0.0, 1.0] where higher is better.
	ScoreTrajectory(ctx context.Context, t Trajectory) (*TrajectoryScore, error)
}
```

### `hitl`

#### `hitl.Manager`

Manager is the interface for managing human-in-the-loop interactions. Implementations must be safe for concurrent use.

```go
type Manager interface {
	// RequestInteraction sends an interaction request and waits for a response.
	// It blocks until a response is received, the request times out, or the
	// context is canceled.
	RequestInteraction(ctx context.Context, req InteractionRequest) (*InteractionResponse, error)

	// AddPolicy registers an approval policy. Policies are evaluated in the
	// order they are added; the first matching policy wins.
	AddPolicy(policy ApprovalPolicy) error

	// ShouldApprove checks whether an action can be auto-approved based on
	// registered policies, the tool name, confidence level, and risk level.
	// Returns true if auto-approval is granted, false if human approval is needed.
	ShouldApprove(ctx context.Context, toolName string, confidence float64, risk RiskLevel) (bool, error)

	// Respond provides a response to a pending interaction request.
	Respond(ctx context.Context, requestID string, resp InteractionResponse) error
}
```

#### `hitl.Notifier`

Notifier is the interface for sending interaction notifications to humans.

```go
type Notifier interface {
	// Notify sends a notification about a pending interaction request.
	Notify(ctx context.Context, req InteractionRequest) error
}
```

## Layer 4 — Protocol and serving

### `protocol/agui`

#### `agui.AGUIHandler`

AGUIHandler serves agents via the AG-UI protocol, providing a standardized interface for UI clients to discover and interact with agents.

```go
type AGUIHandler interface {
	// ServeHTTP handles AG-UI protocol requests.
	http.Handler
}
```

### `protocol/mcp`

#### `mcp.AsyncHandler`

AsyncHandler manages asynchronous operations. Implementations must be safe for concurrent use.

```go
type AsyncHandler interface {
	// Start begins an asynchronous operation and returns its ID.
	// The provided function runs asynchronously; its result or error is
	// stored in the operation.
	Start(ctx context.Context, fn func(ctx context.Context) (any, error)) (string, error)

	// Poll returns the current state of an asynchronous operation.
	Poll(ctx context.Context, id string) (*AsyncOperation, error)

	// Cancel requests cancellation of a running operation. It returns an
	// error if the operation does not exist or cannot be cancelled.
	Cancel(ctx context.Context, id string) error
}
```

#### `mcp.ElicitationHandler`

ElicitationHandler handles user elicitation requests. Implementations connect to the appropriate user interface (CLI prompt, web UI, etc.).

```go
type ElicitationHandler interface {
	// Elicit sends a request for user input and returns the response.
	// It must respect context cancellation for timeout handling.
	Elicit(ctx context.Context, req ElicitationRequest) (*ElicitationResponse, error)
}
```

#### `mcp.OAuthProvider`

OAuthProvider handles the OAuth 2.0 authorization flow for MCP servers.

```go
type OAuthProvider interface {
	// Authorize returns the authorization URL that the user should visit.
	// The state parameter is used for CSRF protection and must be verified
	// when handling the callback.
	Authorize(ctx context.Context, state string) (authURL string, err error)

	// Exchange trades an authorization code for an access token.
	Exchange(ctx context.Context, code string) (*Token, error)

	// Refresh obtains a new access token using a refresh token.
	Refresh(ctx context.Context, refreshToken string) (*Token, error)
}
```

### `protocol/mcp/registry`

#### `registry.MCPClientInterface`

MCPClientInterface abstracts the MCP client for testing.

```go
type MCPClientInterface interface {
	Initialize(ctx context.Context) (*mcp.ServerCapabilities, error)
	ListTools(ctx context.Context) ([]mcp.ToolInfo, error)
	CallTool(ctx context.Context, name string, args map[string]any) (*mcp.ToolCallResult, error)
}
```

### `server`

#### `server.ServerAdapter`

ServerAdapter is the interface that HTTP framework adapters must implement. Each adapter wraps a specific HTTP framework and exposes a uniform API for registering agents and handlers.

```go
type ServerAdapter interface {
	// RegisterAgent registers an agent at the given path prefix. The adapter
	// creates sub-routes for invoke and stream endpoints automatically.
	RegisterAgent(path string, a agent.Agent) error

	// RegisterHandler registers a raw http.Handler at the given path.
	RegisterHandler(path string, handler http.Handler) error

	// Serve starts the HTTP server on the given address. It blocks until the
	// server exits or the context is canceled.
	Serve(ctx context.Context, addr string) error

	// Shutdown gracefully shuts down the server, allowing in-flight requests
	// to complete within the context deadline.
	Shutdown(ctx context.Context) error
}
```

### `server/playground`

#### `playground.AgentSelector`

AgentSelector provides a list of available agents and retrieves them by ID.

```go
type AgentSelector interface {
	// List returns the IDs of all available agents.
	List(ctx context.Context) []string
	// Get returns the agent with the given ID, or nil if not found.
	Get(ctx context.Context, id string) agent.Agent
}
```

#### `playground.StreamAdapter`

StreamAdapter converts an agent's streaming output to a format suitable for Server-Sent Events delivery.

```go
type StreamAdapter interface {
	// WriteEvents reads events from the agent stream and writes them
	// as SSE to the http.ResponseWriter.
	WriteEvents(ctx context.Context, w http.ResponseWriter, a agent.Agent, input string) error
}
```

## Layer 6 — Agent runtime

### `agent`

#### `agent.Agent`

Agent is the primary interface for all agents. An agent has an identity, persona, tools, optional children (for orchestration), and can be invoked synchronously or streamed.

```go
type Agent interface {
	AgentMetadata
	AgentExecutor
}
```

#### `agent.AgentExecutor`

AgentExecutor is the runtime-behaviour surface of an agent. Consumers that only need to run an agent (middleware, retry wrappers, orchestration patterns) can depend on AgentExecutor rather than the full Agent.

```go
type AgentExecutor interface {
	// Invoke executes the agent synchronously and returns a text result.
	Invoke(ctx context.Context, input string, opts ...Option) (string, error)

	// Stream executes the agent and returns an iterator of events.
	Stream(ctx context.Context, input string, opts ...Option) iter.Seq2[Event, error]
}
```

#### `agent.AgentMetadata`

AgentMetadata exposes an agent's static identity and composition. Consumers that only need to introspect an agent (for routing, telemetry, rendering an AgentCard) can depend on AgentMetadata rather than the full Agent.

```go
type AgentMetadata interface {
	// ID returns the unique identifier for this agent.
	ID() string

	// Persona returns the agent's persona (role, goal, backstory).
	Persona() Persona

	// Tools returns the tools available to this agent.
	Tools() []tool.Tool

	// Children returns child agents for orchestration.
	Children() []Agent
}
```

#### `agent.Controller`

Controller decides the next operation to apply to the thought graph. It drives the graph construction process by examining the current state and choosing the appropriate transformation.

```go
type Controller interface {
	// NextOperation examines the graph and returns the next operation, or nil
	// to signal that the graph is complete.
	NextOperation(ctx context.Context, graph *ThoughtGraph) (*Operation, error)
}
```

#### `agent.EventBus`

EventBus is the interface for agent-to-agent async messaging.

```go
type EventBus interface {
	// Publish sends an event to the given topic.
	Publish(ctx context.Context, topic string, event AgentEvent) error
	// Subscribe registers a handler for events on the given topic.
	Subscribe(ctx context.Context, topic string, handler func(AgentEvent)) (Subscription, error)
}
```

#### `agent.Memory`

Memory is the interface for agent memory persistence. Implementations provide the ability to save and load conversation history.

```go
type Memory interface {
	// Save persists messages for the given session.
	Save(ctx context.Context, sessionID string, messages []any) error
	// Load retrieves persisted messages for the given session.
	Load(ctx context.Context, sessionID string) ([]any, error)
}
```

#### `agent.Planner`

Planner is the interface for agent reasoning strategies. A planner decides what actions the agent should take given the current state, and can replan based on observations from executed actions.

```go
type Planner interface {
	// Plan generates a list of actions given the current planner state.
	Plan(ctx context.Context, state PlannerState) ([]Action, error)

	// Replan generates updated actions based on new observations. Called after
	// actions have been executed and observations collected.
	Replan(ctx context.Context, state PlannerState) ([]Action, error)
}
```

#### `agent.Unsubscriber`

Unsubscriber represents an active event subscription that can be cancelled.

```go
type Unsubscriber interface {
	// Unsubscribe removes this subscription.
	Unsubscribe() error
}
```

### `agent/codeact`

#### `codeact.CodeExecutor`

CodeExecutor executes code actions and returns results. Implementations must respect context cancellation and enforce timeouts.

```go
type CodeExecutor interface {
	// Execute runs the given code action and returns the result.
	Execute(ctx context.Context, action CodeAction) (CodeResult, error)
}
```

### `agent/cognitive`

#### `cognitive.ComplexityScorer`

ComplexityScorer classifies the complexity of an input query to determine routing between System 1 and System 2 agents.

```go
type ComplexityScorer interface {
	// Score evaluates the input and returns a complexity classification.
	Score(ctx context.Context, input string) (ComplexityScore, error)
}
```

### `agent/contract`

#### `contract.ContractProvider`

ContractProvider is an optional interface that agents may implement to expose their semantic contract. Use ContractOf to safely extract the contract from any agent.

```go
type ContractProvider interface {
	// Contract returns the agent's semantic contract, or nil if none is set.
	Contract() *schema.Contract
}
```

### `agent/evolving`

#### `evolving.ExperienceDistiller`

ExperienceDistiller extracts reusable knowledge from raw interactions.

```go
type ExperienceDistiller interface {
	// Distill converts raw interactions into structured experiences.
	Distill(ctx context.Context, interactions []Interaction) ([]Experience, error)
}
```

#### `evolving.MetaOptimizer`

MetaOptimizer analyzes agent performance and suggests improvements.

```go
type MetaOptimizer interface {
	// Optimize analyzes experiences and returns optimization suggestions.
	Optimize(ctx context.Context, experiences []Experience) ([]Suggestion, error)
}
```

### `agent/metacognitive`

#### `metacognitive.HeuristicExtractor`

HeuristicExtractor derives heuristics from execution signals and the current self-model. Implementations range from simple rule-based extraction to LLM-powered analysis.

```go
type HeuristicExtractor interface {
	// Extract analyzes the signals and model to produce new heuristics.
	Extract(ctx context.Context, signals MonitoringSignals, model *SelfModel) ([]Heuristic, error)
}
```

#### `metacognitive.SelfModelStore`

SelfModelStore persists and retrieves self-models across sessions.

```go
type SelfModelStore interface {
	// Load retrieves the self-model for the given agent ID.
	// Returns a new empty model if none exists.
	Load(ctx context.Context, agentID string) (*SelfModel, error)

	// Save persists the self-model.
	Save(ctx context.Context, model *SelfModel) error

	// SearchHeuristics finds the k most relevant heuristics for a query.
	// Implementations may use embedding similarity or keyword matching.
	SearchHeuristics(ctx context.Context, agentID, query string, k int) ([]Heuristic, error)
}
```

### `agent/plancache`

#### `plancache.Matcher`

Matcher scores how well an input matches a cached template. Implementations return a score in [0.0, 1.0] where 1.0 is a perfect match and 0.0 is no match at all.

```go
type Matcher interface {
	// Score returns a similarity score between the input and the template.
	// The score must be in the range [0.0, 1.0].
	Score(ctx context.Context, input string, tmpl *Template) (float64, error)
}
```

#### `plancache.Store`

Store persists and retrieves plan templates. Implementations must be safe for concurrent use.

```go
type Store interface {
	// Save persists a template. If a template with the same ID already exists,
	// it is updated.
	Save(ctx context.Context, tmpl *Template) error

	// Get retrieves a template by ID. Returns an error with code
	// ErrTemplateNotFound if the template does not exist.
	Get(ctx context.Context, id string) (*Template, error)

	// List returns all templates for the given agent ID. Returns an empty
	// slice if no templates are found.
	List(ctx context.Context, agentID string) ([]*Template, error)

	// Delete removes a template by ID. Returns nil if the template does not
	// exist.
	Delete(ctx context.Context, id string) error
}
```

### `agent/speculative`

#### `speculative.Predictor`

Predictor produces a fast prediction for a given input. Implementations may use a lighter/cheaper model, caching, or heuristics.

```go
type Predictor interface {
	// Predict returns a prediction string and confidence score (0.0-1.0).
	Predict(ctx context.Context, input string) (prediction string, confidence float64, err error)
}
```

#### `speculative.Validator`

Validator checks whether a prediction matches the ground truth.

```go
type Validator interface {
	// Validate returns true if the prediction is considered equivalent to
	// the ground truth.
	Validate(ctx context.Context, prediction, groundTruth string) (valid bool, err error)
}
```

### `runtime`

#### `runtime.OrchestrationPattern`

OrchestrationPattern defines how a Team coordinates its member agents. Implementations determine the execution strategy: sequential pipeline, parallel scatter-gather, supervisor delegation, and so on.

```go
type OrchestrationPattern interface {
	// Execute runs the orchestration strategy over the given agents with the
	// provided input. It returns an iterator of agent events and errors,
	// following the iter.Seq2 streaming convention.
	Execute(ctx context.Context, agents []agent.Agent, input string) iter.Seq2[agent.Event, error]
}
```

#### `runtime.Plugin`

Plugin intercepts agent execution at the Runner level. All methods are called for every turn. Implementations should be lightweight.

```go
type Plugin interface {
	// Name returns a unique identifier for this plugin.
	Name() string

	// BeforeTurn runs before each agent invocation. It may modify the input
	// message before it reaches the agent.
	BeforeTurn(ctx context.Context, session *Session, input schema.Message) (schema.Message, error)

	// AfterTurn runs after each agent invocation. It may modify the events
	// produced by the agent before they are returned to the caller.
	AfterTurn(ctx context.Context, session *Session, events []agent.Event) ([]agent.Event, error)

	// OnError runs when an error occurs during agent execution.
	// It may wrap, suppress, or replace the error.
	OnError(ctx context.Context, err error) error
}
```

#### `runtime.SessionService`

SessionService manages session lifecycle: creation, retrieval, mutation, and deletion. Implementations must be safe for concurrent use.

```go
type SessionService interface {
	// Create allocates a new Session for the given agentID and persists it.
	// It returns the newly created session or an error.
	Create(ctx context.Context, agentID string) (*Session, error)

	// Get retrieves the Session identified by sessionID. It returns a
	// core.Error with code ErrNotFound if no such session exists.
	Get(ctx context.Context, sessionID string) (*Session, error)

	// Update persists the current state of an existing session. It returns a
	// core.Error with code ErrNotFound if the session does not exist.
	Update(ctx context.Context, session *Session) error

	// Delete removes the session identified by sessionID. It returns a
	// core.Error with code ErrNotFound if no such session exists.
	Delete(ctx context.Context, sessionID string) error
}
```

### `runtime/replay`

#### `replay.CheckpointStore`

CheckpointStore is the interface for persisting and retrieving checkpoints. Implementations must be safe for concurrent use.

```go
type CheckpointStore interface {
	// Save persists a checkpoint. If a checkpoint with the same ID already
	// exists, it is overwritten.
	Save(ctx context.Context, cp *Checkpoint) error

	// Get retrieves a checkpoint by its ID. Returns an error if not found.
	Get(ctx context.Context, id string) (*Checkpoint, error)

	// List returns all checkpoint IDs for a given session, ordered by
	// TurnIndex ascending.
	List(ctx context.Context, sessionID string) ([]string, error)

	// Delete removes a checkpoint by its ID. Returns an error if not found.
	Delete(ctx context.Context, id string) error
}
```

### `runtime/sleeptime`

#### `sleeptime.IdleDetector`

IdleDetector determines whether a session is currently idle (no active user interaction). Implementations must be safe for concurrent use.

```go
type IdleDetector interface {
	// IsIdle reports whether the session is currently idle.
	IsIdle() bool

	// OnActivity records that user activity has occurred, resetting the
	// inactivity timer.
	OnActivity()
}
```

#### `sleeptime.Task`

Task defines a unit of background work that can run during idle periods. Implementations must respect context cancellation for preemption.

```go
type Task interface {
	// Name returns a unique identifier for this task.
	Name() string

	// Priority returns the execution priority of this task.
	Priority() Priority

	// ShouldRun reports whether the task should execute given the current
	// session state. Tasks that have nothing to do should return false.
	ShouldRun(ctx context.Context, state SessionState) bool

	// Run executes the task. It must respect context cancellation and return
	// promptly when the context is cancelled (preemption on wake).
	Run(ctx context.Context, state SessionState) (TaskResult, error)
}
```

### `runtime/teambuilder`

#### `teambuilder.ScoredSelector`

ScoredSelector is an optional capability interface that a Selector may implement to expose the concrete relevance scores it computed for each selected entry. Consumers (e.g. TeamBuilder hooks) should prefer SelectScored when the selector supports it, and fall back to Select for plain selectors that do not carry score information.

```go
type ScoredSelector interface {
	Selector
	// SelectScored returns a ranked subset of candidates with their
	// concrete relevance scores in [0.0, 1.0], ordered by Score descending.
	SelectScored(ctx context.Context, task string, candidates []PoolEntry) ([]ScoredPoolEntry, error)
}
```

#### `teambuilder.Selector`

Selector defines the strategy for selecting agents from a pool of candidates based on a task description. Implementations range from simple keyword matching to LLM-based semantic selection.

```go
type Selector interface {
	// Select returns a ranked subset of candidates suitable for the given task.
	// The returned entries are ordered by relevance (best match first).
	// An empty result indicates no suitable agents were found.
	Select(ctx context.Context, task string, candidates []PoolEntry) ([]PoolEntry, error)
}
```

### `orchestration`

#### `orchestration.OrchestrationPattern`

OrchestrationPattern is the extension interface for named orchestration patterns. Every pattern that participates in the orchestration registry must implement this interface.

```go
type OrchestrationPattern interface {
	core.Runnable

	// Name returns the human-readable identifier for this pattern.
	Name() string
}
```

### `orchestration/debate`

#### `debate.ConvergenceDetector`

ConvergenceDetector determines whether a debate has reached a point where further rounds are unlikely to produce new insights.

```go
type ConvergenceDetector interface {
	// Check evaluates the current debate state and returns whether
	// convergence has been achieved.
	Check(ctx context.Context, state DebateState) (ConvergenceResult, error)
}
```

#### `debate.Protocol`

Protocol determines how agents participate in each round of a debate. Implementations return a map of agent ID to prompt string for each round.

```go
type Protocol interface {
	// NextRound returns the prompts to send to each agent for the next round.
	// The keys are agent IDs, the values are the formatted prompts.
	NextRound(ctx context.Context, state DebateState) (map[string]string, error)
}
```

#### `debate.TwoPassProtocol`

TwoPassProtocol is an optional interface implemented by protocols that require a second evaluation pass after participant contributions are collected for the current round. Protocols that do not need to see the current round's contributions should implement only Protocol.

```go
type TwoPassProtocol interface {
	Protocol
	// FollowUp is called after the first-pass responses for the current
	// round have been collected. The implementation may inspect
	// currentRound.Contributions to build prompts for a second pass
	// (for example, a judge that evaluates the current round's arguments).
	// Returning an empty map signals no follow-up prompts for this round.
	FollowUp(ctx context.Context, state DebateState, currentRound Round) (map[string]string, error)
}
```

## Kubernetes overlay (outside the library)

### `k8s/operator`

#### `operator.Reconciler`

Reconciler defines the reconciliation interface for Beluga AI custom resources. Implementations compute the desired Kubernetes resource state from a CRD and return it as plain Go structs — they never call the Kubernetes API directly.

```go
type Reconciler interface {
	// ReconcileAgent derives the desired Kubernetes resources for a single Agent
	// CRD. It returns an error if the spec is invalid (e.g. missing ModelRef).
	ReconcileAgent(ctx context.Context, agent *AgentResource) (*ReconcileResult, error)

	// ReconcileTeam derives the desired Kubernetes resources for every member
	// agent defined in a Team CRD. It returns an error if the spec is invalid.
	ReconcileTeam(ctx context.Context, team *TeamResource) (*TeamReconcileResult, error)
}
```

#### `operator.ResourceStore`

ResourceStore is the read interface the controller uses to fetch the current state of custom resources. Callers provide an implementation backed by their preferred K8s client.

```go
type ResourceStore interface {
	// GetAgent returns the Agent CRD with the given name and namespace.
	GetAgent(ctx context.Context, namespace, name string) (*AgentResource, error)

	// GetTeam returns the Team CRD with the given name and namespace.
	GetTeam(ctx context.Context, namespace, name string) (*TeamResource, error)
}
```

#### `operator.StatusWriter`

StatusWriter persists the reconciliation status back to the cluster. Callers provide an implementation backed by their preferred K8s client.

```go
type StatusWriter interface {
	// UpdateAgentStatus writes the observed status for the named Agent CRD.
	UpdateAgentStatus(ctx context.Context, namespace, name string, status AgentStatus) error

	// UpdateTeamStatus writes the observed status for the named Team CRD.
	UpdateTeamStatus(ctx context.Context, namespace, name string, status TeamStatus) error
}
```

## Provider client seams

Provider packages export small interfaces over their vendor SDK clients so tests can substitute a fake. They are not extension points of the framework itself; they are listed here only for completeness.

| Package | Interface | Purpose |
|---|---|---|
| `llm/providers/bedrock` | `ConverseAPI` | ConverseAPI defines the subset of bedrockruntime.Client methods we need. This allows injection of mock clients for testing. |
| `memory/stores/mongodb` | `Collection` | Collection defines the subset of mongo.Collection methods used by this store. This interface enables testing with mock implementations. |
| `memory/stores/postgres` | `DBTX` | DBTX is the minimal interface satisfied by both pgx.Conn, pgxpool.Pool, and pgxmock for testing. |
| `rag/vectorstore/providers/chroma` | `HTTPClient` | HTTPClient abstracts HTTP calls for testability. |
| `rag/vectorstore/providers/elasticsearch` | `HTTPClient` | HTTPClient abstracts HTTP calls for testability. |
| `rag/vectorstore/providers/milvus` | `HTTPClient` | HTTPClient abstracts HTTP calls for testability. |
| `rag/vectorstore/providers/mongodb` | `HTTPClient` | HTTPClient abstracts HTTP calls for testability. |
| `rag/vectorstore/providers/pgvector` | `Pool` | Pool abstracts pgx pool operations for testability. |
| `rag/vectorstore/providers/pinecone` | `HTTPClient` | HTTPClient abstracts HTTP calls for testability. |
| `rag/vectorstore/providers/qdrant` | `HTTPClient` | HTTPClient abstracts HTTP calls for testability. |
| `rag/vectorstore/providers/redis` | `RedisClient` | RedisClient abstracts the Redis client for testability. |
| `rag/vectorstore/providers/sqlitevec` | `DB` | DB abstracts the database interface for testability. |
| `rag/vectorstore/providers/turbopuffer` | `HTTPClient` | HTTPClient abstracts HTTP calls for testability. |
| `rag/vectorstore/providers/vespa` | `HTTPClient` | HTTPClient abstracts HTTP calls for testability. |
| `rag/vectorstore/providers/weaviate` | `HTTPClient` | HTTPClient abstracts HTTP calls for testability. |
| `workflow/providers/dapr` | `StateClient` | StateClient defines the subset of the Dapr client interface used for state operations. |
| `workflow/providers/inngest` | `HTTPClient` | HTTPClient defines the HTTP client interface for Inngest API calls. |
| `workflow/providers/kafka` | `Reader` | Reader defines the interface for reading Kafka messages. |
| `workflow/providers/kafka` | `Writer` | Writer defines the interface for writing Kafka messages. |

## Related

- [03 — Extensibility Patterns](../architecture/03-extensibility-patterns.md) — how these interfaces compose.
- [18 — Package Dependency Map](../architecture/18-package-dependency-map.md) — which layer each package lives in.
- [Providers](./providers.md) — the registered implementations of the provider interfaces.
