# References

## Pattern Sources

### Error Handling Pattern
- `pkg/agents/errors.go` - AgentError with Op/Err/Code
- `pkg/stt/errors.go` - STTError example
- `pkg/tts/errors.go` - TTSError example

### Metrics Pattern
- `pkg/agents/metrics.go` - Comprehensive metrics with tracer
- `pkg/stt/metrics.go` - Voice-specific metrics
- `pkg/memory/metrics.go` - Operation metrics

### Builder Pattern
- `pkg/agents/agents.go` - Agent creation with options
- `pkg/voicesession/session.go` - VoiceSession builder
- `pkg/vectorstores/vectorstores.go` - Factory pattern

### Interface Design
- `pkg/agents/iface/agent.go` - Agent, Planner, Tool interfaces
- `pkg/llms/iface/chat_model.go` - ChatModel interface
- `pkg/memory/iface/memory.go` - Memory interface
- `pkg/embeddings/iface/iface.go` - Embedder interface
- `pkg/voiceutils/iface/` - STT, TTS, VAD interfaces

## Key Files for Agent Package

| File | What to learn |
|------|---------------|
| `pkg/agents/providers/base/base_agent.go` | Base agent implementation |
| `pkg/agents/providers/react/agent.go` | ReAct agent pattern |
| `pkg/memory/memory.go` | Memory factory and utilities |
| `pkg/memory/internal/buffer/buffer.go` | Buffer memory implementation |

## Key Files for RAG Package

| File | What to learn |
|------|---------------|
| `pkg/vectorstores/providers/inmemory/inmemory_vectorstore.go` | In-memory store |
| `pkg/embeddings/embeddings.go` | Embedder factory |
| `pkg/textsplitters/textsplitters.go` | Text splitting |
| `pkg/retrievers/vectorstore.go` | Vector store retriever |

## Key Files for VoiceAgent Package

| File | What to learn |
|------|---------------|
| `pkg/voicesession/session.go` | Session creation |
| `pkg/voicesession/internal/session_impl.go` | Session implementation |
| `pkg/stt/stt.go` | STT provider factory |
| `pkg/tts/tts.go` | TTS provider factory |
| `pkg/vad/vad.go` | VAD provider factory |

## Import Patterns

### Agent Package Imports
```go
import (
    agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
    llmsiface "github.com/lookatitude/beluga-ai/pkg/llms/iface"
    memoryiface "github.com/lookatitude/beluga-ai/pkg/memory/iface"
    "github.com/lookatitude/beluga-ai/pkg/memory"
    "github.com/lookatitude/beluga-ai/pkg/core"
)
```

### RAG Package Imports
```go
import (
    embeddingsiface "github.com/lookatitude/beluga-ai/pkg/embeddings/iface"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/inmemory"
    "github.com/lookatitude/beluga-ai/pkg/textsplitters"
    "github.com/lookatitude/beluga-ai/pkg/schema"
)
```

### VoiceAgent Package Imports
```go
import (
    sttiface "github.com/lookatitude/beluga-ai/pkg/stt/iface"
    ttsiface "github.com/lookatitude/beluga-ai/pkg/tts/iface"
    vadiface "github.com/lookatitude/beluga-ai/pkg/vad/iface"
    agentsiface "github.com/lookatitude/beluga-ai/pkg/agents/iface"
    "github.com/lookatitude/beluga-ai/pkg/voicesession"
)
```
