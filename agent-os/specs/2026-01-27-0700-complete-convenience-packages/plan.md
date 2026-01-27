# Complete WIP Convenience Packages

## Overview

Implement the `Build()` methods for the 3 WIP convenience packages to make them fully functional:
- `pkg/convenience/agent/` - Create working agents
- `pkg/convenience/rag/` - Create working RAG pipelines
- `pkg/convenience/voiceagent/` - Create working voice agents

---

## Task 1: Save Spec Documentation

Create `agent-os/specs/2026-01-27-0700-complete-convenience-packages/` with:
- **plan.md** — This full plan
- **shape.md** — Shaping notes
- **standards.md** — Relevant standards
- **references.md** — Pointers to pkg/agents/ patterns

---

## Task 2: Complete `pkg/convenience/agent/`

### Files to Create/Modify

| File | Action |
|------|--------|
| `agent.go` | Modify - add builder fields, Build() method |
| `types.go` | Create - Agent interface |
| `agent_impl.go` | Create - convenienceAgent implementation |
| `errors.go` | Create - Op/Err/Code errors |
| `metrics.go` | Create - OTEL metrics |
| `agent_test.go` | Create - unit tests |
| `advanced_test.go` | Create - comprehensive tests |
| `test_utils.go` | Create - test helpers |

### Builder Additions

```go
// New fields
llm          llmsiface.LLM
llmProvider  string
chatModel    llmsiface.ChatModel
memory       memoryiface.Memory
memoryType   string
memorySize   int
tools        []agentsiface.Tool
timeout      time.Duration

// New methods
WithLLM(llm llmsiface.LLM) *Builder
WithLLMProvider(provider, apiKey string) *Builder
WithChatModel(model llmsiface.ChatModel) *Builder
WithBufferMemory(maxMessages int) *Builder
WithMemory(mem memoryiface.Memory) *Builder
WithTool(tool agentsiface.Tool) *Builder
WithTools(tools []agentsiface.Tool) *Builder
WithTimeout(timeout time.Duration) *Builder
```

### Agent Interface

```go
type Agent interface {
    Run(ctx context.Context, input string) (string, error)
    RunWithInputs(ctx context.Context, inputs map[string]any) (map[string]any, error)
    Stream(ctx context.Context, input string) (<-chan string, error)
    GetName() string
    GetTools() []agentsiface.Tool
    GetMemory() memoryiface.Memory
    Shutdown() error
}
```

### Build() Implementation

```go
func (b *Builder) Build(ctx context.Context) (Agent, error) {
    // 1. Validate required fields (LLM required)
    // 2. Resolve LLM via registry if provider name given
    // 3. Create memory if configured
    // 4. Build agent options from builder config
    // 5. Create underlying agent (BaseAgent or ReActAgent based on agentType)
    // 6. Wrap in convenienceAgent with memory integration
    // 7. Return Agent interface
}
```

### Error Codes

```go
const (
    ErrCodeMissingLLM     = "missing_llm"
    ErrCodeLLMCreation    = "llm_creation_failed"
    ErrCodeMemoryCreation = "memory_creation_failed"
    ErrCodeAgentCreation  = "agent_creation_failed"
    ErrCodeInvalidLLMType = "invalid_llm_type"
    ErrCodeExecution      = "execution_failed"
)
```

---

## Task 3: Complete `pkg/convenience/rag/`

### Files to Create/Modify

| File | Action |
|------|--------|
| `rag.go` | Modify - add builder fields, Build() method |
| `types.go` | Create - Pipeline interface |
| `pipeline_impl.go` | Create - ragPipeline implementation |
| `errors.go` | Create - Op/Err/Code errors |
| `metrics.go` | Create - OTEL metrics |
| `rag_test.go` | Create - unit tests |
| `advanced_test.go` | Create - comprehensive tests |
| `test_utils.go` | Create - test helpers |

### Builder Additions

```go
// New fields
embedder         embeddingsiface.Embedder
embedderProvider string
vectorStore      vectorstores.VectorStore
vectorStoreType  string
llm              llmsiface.ChatModel
llmProvider      string
systemPrompt     string
returnSources    bool
scoreThreshold   float32

// New methods
WithEmbedder(embedder embeddingsiface.Embedder) *Builder
WithEmbedderProvider(provider, apiKey string) *Builder
WithVectorStore(store vectorstores.VectorStore) *Builder
WithInMemoryVectorStore() *Builder
WithLLM(llm llmsiface.ChatModel) *Builder
WithLLMProvider(provider, apiKey, model string) *Builder
WithSystemPrompt(prompt string) *Builder
WithReturnSources(enabled bool) *Builder
WithScoreThreshold(threshold float32) *Builder
```

### Pipeline Interface

```go
type Pipeline interface {
    Query(ctx context.Context, query string) (string, error)
    QueryWithSources(ctx context.Context, query string) (string, []schema.Document, error)
    IngestDocuments(ctx context.Context) error
    IngestFromPaths(ctx context.Context, paths []string) error
    AddDocuments(ctx context.Context, docs []schema.Document) error
    Search(ctx context.Context, query string, k int) ([]schema.Document, []float32, error)
    GetDocumentCount() int
    Clear(ctx context.Context) error
}
```

### Build() Implementation

```go
func (b *Builder) Build(ctx context.Context) (Pipeline, error) {
    // 1. Validate required fields (embedder required)
    // 2. Resolve embedder via registry if provider name given
    // 3. Resolve/create vector store (default: in-memory)
    // 4. Resolve LLM if configured (optional for retrieval-only)
    // 5. Create text splitter with chunk config
    // 6. Create retriever with topK and threshold
    // 7. Return ragPipeline with all components
}
```

### Error Codes

```go
const (
    ErrCodeMissingEmbedder     = "missing_embedder"
    ErrCodeEmbedderCreation    = "embedder_creation_failed"
    ErrCodeVectorStoreCreation = "vectorstore_creation_failed"
    ErrCodeLLMCreation         = "llm_creation_failed"
    ErrCodeSplitterCreation    = "splitter_creation_failed"
    ErrCodeRetrieverCreation   = "retriever_creation_failed"
    ErrCodeRetrievalFailed     = "retrieval_failed"
    ErrCodeGenerationFailed    = "generation_failed"
    ErrCodeNoLLM               = "no_llm_configured"
)
```

---

## Task 4: Complete `pkg/convenience/voiceagent/`

### Files to Create/Modify

| File | Action |
|------|--------|
| `voiceagent.go` | Modify - add builder fields, Build() method |
| `types.go` | Create - VoiceAgent, Session interfaces |
| `voiceagent_impl.go` | Create - convenienceVoiceAgent implementation |
| `session_impl.go` | Create - voiceSession implementation |
| `errors.go` | Create - Op/Err/Code errors |
| `metrics.go` | Create - OTEL metrics |
| `voiceagent_test.go` | Create - unit tests |
| `advanced_test.go` | Create - comprehensive tests |
| `test_utils.go` | Create - test helpers |

### Builder Additions

```go
// New fields
stt          sttiface.STTProvider
sttConfig    *stt.Config
tts          ttsiface.TTSProvider
ttsConfig    *tts.Config
vad          vadiface.VADProvider
vadConfig    *vad.Config
llm          llmsiface.ChatModel
llmConfig    *llms.Config
agent        agentsiface.CompositeAgent
memory       memoryiface.Memory
tools        []agentsiface.Tool
sessionConfig *voicesession.Config
onTranscript func(text string, isFinal bool)
onResponse   func(text string)
onError      func(error)

// New methods
WithSTTInstance(stt sttiface.STTProvider) *Builder
WithSTTConfig(config *stt.Config) *Builder
WithTTSInstance(tts ttsiface.TTSProvider) *Builder
WithTTSConfig(config *tts.Config) *Builder
WithVADInstance(vad vadiface.VADProvider) *Builder
WithLLMInstance(llm llmsiface.ChatModel) *Builder
WithAgent(agent agentsiface.CompositeAgent) *Builder
WithTools(tools []agentsiface.Tool) *Builder
WithOnTranscript(fn func(text string, isFinal bool)) *Builder
WithOnResponse(fn func(text string)) *Builder
WithOnError(fn func(error)) *Builder
WithSessionConfig(config *voicesession.Config) *Builder
```

### VoiceAgent Interface

```go
type VoiceAgent interface {
    StartSession(ctx context.Context) (Session, error)
    ProcessAudio(ctx context.Context, audio []byte) ([]byte, error)
    ProcessText(ctx context.Context, text string) (string, error)
    GetSTT() sttiface.STTProvider
    GetTTS() ttsiface.TTSProvider
    GetAgent() agentsiface.CompositeAgent
    Shutdown() error
}

type Session interface {
    ID() string
    Start(ctx context.Context) error
    Stop() error
    SendAudio(ctx context.Context, audio []byte) error
    ReceiveAudio() <-chan []byte
    GetTranscript() string
    IsActive() bool
}
```

### Build() Implementation

```go
func (b *Builder) Build(ctx context.Context) (VoiceAgent, error) {
    // 1. Validate required fields (STT and TTS required)
    // 2. Resolve STT via registry if provider name given
    // 3. Resolve TTS via registry if provider name given
    // 4. Resolve VAD if configured (optional)
    // 5. Resolve/create agent if LLM or agent provided
    // 6. Create memory if enabled
    // 7. Return convenienceVoiceAgent with all components
}
```

### Error Codes

```go
const (
    ErrCodeMissingSTT      = "missing_stt"
    ErrCodeMissingTTS      = "missing_tts"
    ErrCodeSTTCreation     = "stt_creation_failed"
    ErrCodeTTSCreation     = "tts_creation_failed"
    ErrCodeVADCreation     = "vad_creation_failed"
    ErrCodeAgentCreation   = "agent_creation_failed"
    ErrCodeMemoryCreation  = "memory_creation_failed"
    ErrCodeSessionCreation = "session_creation_failed"
    ErrCodeTranscription   = "transcription_failed"
    ErrCodeSynthesis       = "synthesis_failed"
)
```

---

## Task 5: Add Integration Tests

Create `tests/integration/convenience/` tests for:
- Agent package with mock LLM
- RAG pipeline with mock embedder and vector store
- Voice agent with mock STT/TTS

Update existing `convenience_test.go` with Build() method tests.

---

## Task 6: Update Documentation

Update each package's README.md with:
- Complete API reference
- Usage examples with Build() method
- Error handling documentation
- Configuration options

---

## Critical Files (Reference)

| File | Purpose |
|------|---------|
| `pkg/agents/agents.go` | Agent creation patterns |
| `pkg/agents/errors.go` | Op/Err/Code error pattern |
| `pkg/agents/metrics.go` | OTEL metrics pattern |
| `pkg/voicesession/session.go` | Voice session patterns |
| `pkg/vectorstores/vectorstores.go` | RAG component patterns |
| `pkg/retrievers/retrievers.go` | Retriever patterns |

---

## Verification

After implementation:
```bash
# Run unit tests for convenience packages
go test -v ./pkg/convenience/...

# Run with race detection
go test -race -v ./pkg/convenience/...

# Run integration tests
go test -v ./tests/integration/convenience/...

# Run full CI
make ci-local
```

---

## Standards Applied

1. **backend/op-err-code** — All errors use Op/Err/Code pattern
2. **backend/factory-signature** — Build() returns interface, error
3. **backend/otel-spans** — Tracing in Build() and key operations
4. **global/required-files** — Each package has errors.go, metrics.go, test_utils.go, advanced_test.go
5. **testing/advanced-test** — Table-driven tests, concurrency tests, error branch coverage
