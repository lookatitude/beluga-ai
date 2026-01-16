# Beluga AI Framework - Package Catalog

**Last Updated**: 2025-01-07  
**Purpose**: Comprehensive catalog of all packages in `pkg/` with descriptions, intents, and sub-package structure

## Table of Contents

1. [Foundation Packages](#foundation-packages)
2. [Provider Packages](#provider-packages)
3. [Higher-Level Packages](#higher-level-packages)
4. [Voice Packages](#voice-packages)
5. [Messaging Packages](#messaging-packages)
6. [Supporting Packages](#supporting-packages)

---

## Foundation Packages

### pkg/schema

**Description**: Core data structures and message types for the entire framework. Serves as the data contract layer ensuring consistent communication and interoperability across all framework components.

**Intent**: Single source of truth for data exchange between packages. Provides standardized, well-defined data structures with type safety, validation, and observability.

**Key Components**:
- Messages: Standardized message formats for human-AI-system communication
- Documents: Structured document representations with metadata and embeddings
- Configurations: Validated configuration schemas
- Agent I/O: Agent-to-agent communication types

**Sub-packages**:
- `iface/`: Interface definitions and error handling
- `internal/`: Private implementations (message.go, document.go, history.go, agent_io.go)

**Dependencies**: None (foundation package)

---

### pkg/core

**Description**: Fundamental utilities and core models serving as the foundational "glue" layer of the Beluga AI Framework.

**Intent**: Provides essential abstractions, dependency injection, observability, and error handling that orchestrates components throughout the system.

**Key Components**:
- Runnable interface: Central abstraction for executable components
- Dependency injection container: Type-safe dependency resolution
- Error handling: Framework-wide standardized error types
- Utilities: Common operations and helpers

**Sub-packages**:
- `iface/`: Core interface definitions
- `model/`: Core model types
- `utils/`: Utility functions

**Dependencies**: `pkg/schema`, `pkg/monitoring`

---

### pkg/config

**Description**: Framework-wide configuration loading and validation system supporting multiple configuration sources.

**Intent**: Centralized config management with Viper provider support, environment variables, and programmatic configuration. Provides schema-based validation and composite providers.

**Key Components**:
- Configuration providers: Viper, composite, custom providers
- Validation: Schema-based validation with error handling
- Loading: Multiple source support (YAML, JSON, TOML, env vars)

**Sub-packages**:
- `iface/`: Provider interfaces and types
- `internal/loader/`: Configuration loading logic
- `providers/viper/`: Viper-based provider
- `providers/composite/`: Composite provider for chaining

**Dependencies**: `pkg/schema`, `pkg/monitoring`

---

### pkg/monitoring

**Description**: OpenTelemetry integration for observability across the entire framework.

**Intent**: Provides metrics, tracing, structured logging, and health checks. Ensures all framework components have consistent observability.

**Key Components**:
- Metrics: OpenTelemetry metrics collection
- Tracing: Distributed tracing support
- Logging: Structured logging with trace/span IDs
- Health checks: Health monitoring and status reporting

**Sub-packages**:
- `iface/`: Monitoring interfaces
- `internal/metrics/`: Metrics implementation
- `internal/tracer/`: Tracing implementation
- `internal/logger/`: Logging implementation
- `internal/health/`: Health check implementation
- `internal/safety/`: Safety monitoring
- `internal/ethics/`: Ethics monitoring
- `internal/best_practices/`: Best practices monitoring
- `providers/`: Monitoring provider implementations

**Dependencies**: None (foundation package)

---

## Provider Packages

### pkg/llms

**Description**: Large Language Model interactions with multi-provider support.

**Intent**: Unified interface for LLM operations across multiple providers (OpenAI, Anthropic, Bedrock, Ollama). Provides streaming, tool calling, batch processing, and comprehensive error handling.

**Key Components**:
- LLMCaller interface: Core LLM operations
- Streaming support: Real-time response streaming
- Tool calling: Cross-provider function calling
- Batch processing: Concurrent batch operations

**Sub-packages**:
- `iface/`: LLM interface definitions
- `internal/common/`: Shared utilities
- `providers/openai/`: OpenAI provider
- `providers/anthropic/`: Anthropic provider
- `providers/bedrock/`: AWS Bedrock provider
- `providers/gemini/`: Google Gemini provider
- `providers/grok/`: xAI Grok provider
- `providers/groq/`: Groq provider
- `providers/ollama/`: Ollama local provider
- `providers/mock/`: Mock provider for testing

**Dependencies**: `pkg/schema`, `pkg/config`, `pkg/monitoring`, `pkg/core`

---

### pkg/embeddings

**Description**: Text embeddings for vector stores and semantic search.

**Intent**: Multi-provider embedding generation for RAG applications. Supports semantic search, similarity comparison, and clustering.

**Key Components**:
- Embedder interface: Core embedding operations
- Multi-provider support: OpenAI, Cohere, Google Multimodal, Ollama
- Batch embedding: Efficient bulk operations

**Sub-packages**:
- `iface/`: Embedder interface definitions
- `internal/`: Private implementations
- `providers/openai/`: OpenAI embeddings
- `providers/cohere/`: Cohere embeddings
- `providers/google_multimodal/`: Google multimodal embeddings
- `providers/ollama/`: Ollama embeddings
- `providers/mock/`: Mock provider
- `registry/`: Provider registry
- `testdata/`: Test data
- `testinit/`: Test initialization
- `testutils/`: Test utilities

**Dependencies**: `pkg/schema`, `pkg/config`, `pkg/monitoring`

---

### pkg/vectorstores

**Description**: Vector database interfaces for RAG applications.

**Intent**: Abstraction over vector databases (PGVector, InMemory, etc.) for efficient similarity search and document storage.

**Key Components**:
- VectorStore interface: Core vector operations
- Similarity search: Cosine similarity with configurable algorithms
- Batch operations: Efficient bulk document processing

**Sub-packages**:
- `iface/`: VectorStore interface definitions
- `internal/`: Private implementations
- `providers/`: Vector store provider implementations

**Dependencies**: `pkg/schema`, `pkg/config`, `pkg/monitoring`, `pkg/embeddings`

---

### pkg/prompts

**Description**: Prompt templating and management system.

**Intent**: Template rendering and prompt configuration with validation, caching, and provider-specific formatting.

**Key Components**:
- PromptFormatter interface: Core formatting operations
- Template management: String and chat templates
- Adapter pattern: Provider-specific formatting
- Template caching: Performance optimization

**Sub-packages**:
- `iface/`: Template and formatter interfaces
- `internal/`: Template and adapter implementations
- `providers/`: Provider implementations and mocks

**Dependencies**: `pkg/schema`, `pkg/config`

---

## Higher-Level Packages

### pkg/chatmodels

**Description**: Chat-based LLM interactions with conversation support.

**Intent**: Message generation/streaming, tool binding, health checks. Provides chat-oriented interface over LLM providers.

**Key Components**:
- ChatModel interface: Chat-based operations
- Streaming support: Real-time message streaming
- Tool binding: Function calling integration
- Health checks: Model availability monitoring

**Sub-packages**:
- `iface/`: ChatModel interface definitions
- `internal/mock/`: Mock implementations
- `providers/openai/`: OpenAI chat provider
- `registry/`: Provider registry
- `testinit/`: Test initialization

**Dependencies**: `pkg/llms`, `pkg/schema`, `pkg/prompts`

---

### pkg/memory

**Description**: Conversation memory for agents and LLMs.

**Intent**: Manages conversation history and context with multiple memory types (buffer, summary, vectorstore, window).

**Key Components**:
- Memory interface: Core memory operations
- Multiple types: Buffer, window, summary, vectorstore memory
- History management: Conversation history storage and retrieval

**Sub-packages**:
- `iface/`: Memory interface definitions
- `internal/buffer/`: Buffer memory implementation
- `internal/summary/`: Summary-based memory
- `internal/vectorstore/`: Vector store memory
- `internal/window/`: Window-based memory
- `internal/redis/`: Redis-backed memory
- `internal/mock/`: Mock implementations
- `providers/`: Provider implementations

**Dependencies**: `pkg/schema`, `pkg/config`, `pkg/monitoring`, `pkg/vectorstores`, `pkg/embeddings`

---

### pkg/retrievers

**Description**: Information retrieval from data sources for RAG pipelines.

**Intent**: Composes with vectorstores/embeddings for retrieval-augmented generation. Supports multiple retrieval strategies.

**Key Components**:
- Retriever interface: Core retrieval operations
- VectorStoreRetriever: Vector similarity-based retrieval
- Multiple strategies: Vector search, keyword-based, hybrid

**Sub-packages**:
- `iface/`: Retriever interface definitions
- `internal/mock/`: Mock implementations

**Dependencies**: `pkg/vectorstores`, `pkg/embeddings`, `pkg/schema`, `pkg/core`

---

### pkg/agents

**Description**: AI agent framework for autonomous agents.

**Intent**: Autonomous agents with reasoning, planning, and tool execution. Supports multiple agent architectures (BaseAgent, ReActAgent).

**Key Components**:
- Agent interface: Core agent operations
- Streaming agents: Real-time streaming execution
- Tool integration: Seamless tool registry integration
- Executor: Plan execution and tool orchestration

**Sub-packages**:
- `iface/`: Agent interface definitions
- `internal/base/`: Base agent implementation
- `internal/executor/`: Plan execution and tool orchestration
- `internal/mock/`: Mock implementations
- `providers/react/`: ReAct agent provider
- `providers/planexecute/`: Plan-execute agent provider
- `tools/`: Tool integration and registry
  - `tools/api/`: API tool
  - `tools/gofunc/`: Go function tool
  - `tools/mcp/`: MCP tool
  - `tools/shell/`: Shell tool
  - `tools/providers/`: Provider-specific tools

**Dependencies**: `pkg/llms`, `pkg/schema`, `pkg/config`, `pkg/monitoring`, `pkg/core`, `pkg/memory`

---

### pkg/orchestration

**Description**: Complex workflow management for multi-step AI operations.

**Intent**: Chains, graphs, workflows, scheduler, message bus. Enables complex process management and inter-component communication.

**Key Components**:
- Orchestrator interface: Core orchestration operations
- Chains: Sequential step execution
- Graphs: Dependency management and DAG execution
- Workflows: Distributed workflow coordination
- Scheduler: Task scheduling and execution
- MessageBus: Inter-agent communication

**Sub-packages**:
- `iface/`: Orchestration interface definitions
- `internal/scheduler/`: Task scheduling
- `internal/messagebus/`: Inter-agent communication
- `internal/monitoring/`: Workflow monitoring
- `internal/task_chaining/`: Task chaining utilities
- `providers/chain/`: Chain orchestration provider
- `providers/graph/`: Graph orchestration provider
- `providers/workflow/`: Workflow orchestration provider (Temporal)

**Dependencies**: `pkg/agents`, `pkg/schema`, `pkg/config`, `pkg/monitoring`, `pkg/core`

---

### pkg/server

**Description**: Expose framework components as APIs.

**Intent**: REST, MCP, streaming support. Provides HTTP API endpoints and Model Context Protocol server.

**Key Components**:
- Server interface: Core server operations
- REST provider: HTTP API with streaming support
- MCP provider: Model Context Protocol server
- Middleware: Extensible middleware system

**Sub-packages**:
- `iface/`: Server interface definitions
- `providers/rest/`: REST API provider
- `providers/mcp/`: MCP server provider

**Dependencies**: `pkg/agents`, `pkg/orchestration`, `pkg/schema`

---

## Voice Packages

### pkg/voice/stt

**Description**: Speech-to-Text providers for converting audio to text.

**Intent**: Multi-provider STT support (Deepgram, Google, Azure, OpenAI) with streaming, REST fallback, and comprehensive error handling.

**Key Components**:
- STTProvider interface: Core transcription operations
- Streaming support: Real-time transcription with interim results
- REST fallback: Automatic fallback when streaming unavailable

**Sub-packages**:
- `iface/`: STTProvider interface definitions
- `internal/`: Private implementations
- `providers/deepgram/`: Deepgram provider
- `providers/google/`: Google Cloud Speech-to-Text
- `providers/azure/`: Azure Speech Services
- `providers/openai/`: OpenAI Whisper

**Dependencies**: `pkg/schema`, `pkg/config`, `pkg/monitoring`

---

### pkg/voice/tts

**Description**: Text-to-Speech providers for converting text to speech audio.

**Intent**: Multi-provider TTS support (OpenAI, Google, Azure, ElevenLabs) with streaming, voice customization, and SSML support.

**Key Components**:
- TTSProvider interface: Core speech generation operations
- Streaming support: Real-time audio generation
- Voice customization: Multiple voices, speeds, pitches, styles

**Sub-packages**:
- `iface/`: TTSProvider interface definitions
- `internal/`: Private implementations
- `providers/openai/`: OpenAI TTS
- `providers/google/`: Google Cloud Text-to-Speech
- `providers/azure/`: Azure Speech Services
- `providers/elevenlabs/`: ElevenLabs TTS

**Dependencies**: `pkg/schema`, `pkg/config`, `pkg/monitoring`

---

### pkg/voice/s2s

**Description**: Speech-to-Speech (end-to-end speech) providers for direct speech conversion.

**Intent**: Direct speech-to-speech conversion without intermediate text. Supports built-in reasoning or external agent integration.

**Key Components**:
- S2SProvider interface: Core speech-to-speech operations
- Streaming support: Real-time bidirectional streaming
- Reasoning modes: Built-in vs external agent reasoning
- Fallback: Automatic fallback between providers

**Sub-packages**:
- `iface/`: S2SProvider interface definitions
- `internal/`: Private implementations
- `providers/amazon_nova/`: Amazon Nova 2 Sonic
- `providers/grok/`: xAI Grok Voice Agent
- `providers/gemini/`: Google Gemini 2.5 Flash Native Audio
- `providers/openai_realtime/`: OpenAI Realtime API

**Dependencies**: `pkg/schema`, `pkg/config`, `pkg/monitoring`

---

### pkg/voice/vad

**Description**: Voice Activity Detection for detecting speech in audio streams.

**Intent**: Detect voice activity in audio streams with low latency. Filters out silence and noise.

**Key Components**:
- VADProvider interface: Core VAD operations
- Streaming support: Real-time voice activity detection
- Low latency: Optimized for real-time processing

**Sub-packages**:
- `iface/`: VADProvider interface definitions
- `internal/`: Private implementations
- `providers/silero/`: Silero ONNX-based VAD
- `providers/webrtc/`: WebRTC VAD
- `providers/rnnoise/`: RNNoise-based VAD
- `providers/energy/`: Energy-based VAD

**Dependencies**: `pkg/schema`, `pkg/config`, `pkg/monitoring`

---

### pkg/voice/turndetection

**Description**: Turn Detection for identifying when speakers finish talking.

**Intent**: Identify complete user utterances for better conversation flow. Supports heuristic and ML-based detection.

**Key Components**:
- TurnDetector interface: Core turn detection operations
- Silence-based: Detects turns based on silence duration
- ML-based: Uses machine learning models for accurate detection

**Sub-packages**:
- `iface/`: TurnDetector interface definitions
- `internal/`: Private implementations
- `providers/heuristic/`: Rule-based turn detection
- `providers/onnx/`: ML-based turn detection

**Dependencies**: `pkg/schema`, `pkg/config`, `pkg/monitoring`

---

### pkg/voice/transport

**Description**: Audio transport layer for various protocols.

**Intent**: WebSocket, WebRTC transport implementations with connection management, automatic reconnection, and error handling.

**Key Components**:
- Transport interface: Core transport operations
- Real-time audio: Low-latency audio streaming
- Connection management: Automatic reconnection and error handling

**Sub-packages**:
- `iface/`: Transport interface definitions
- `internal/`: Private implementations
- `providers/websocket/`: WebSocket transport
- `providers/webrtc/`: WebRTC transport

**Dependencies**: `pkg/schema`, `pkg/config`, `pkg/monitoring`

---

### pkg/voice/noise

**Description**: Noise cancellation for removing background noise from audio.

**Intent**: Remove background noise from audio signals with real-time processing and adaptive algorithms.

**Key Components**:
- NoiseCancellation interface: Core noise cancellation operations
- Real-time processing: Low-latency noise reduction
- Adaptive algorithms: Self-adjusting noise profiles

**Sub-packages**:
- `iface/`: NoiseCancellation interface definitions
- `internal/`: Private implementations
- `providers/rnnoise/`: RNNoise-based noise suppression
- `providers/spectral/`: Spectral subtraction
- `providers/webrtc/`: WebRTC noise suppression

**Dependencies**: `pkg/schema`, `pkg/config`, `pkg/monitoring`

---

### pkg/voice/session

**Description**: Complete voice session management with lifecycle, state machine, and audio processing pipeline.

**Intent**: Manages complete lifecycle of voice interactions including audio input/output, transcription, synthesis, agent communication, error recovery, interruption handling, and preemptive generation.

**Key Components**:
- VoiceSession interface: Core session operations
- Lifecycle management: Start, stop, state management
- State machine: Session state transitions
- Audio processing: Complete STT → Agent → TTS pipeline
- Error recovery: Automatic retry with exponential backoff
- Interruption handling: Configurable interruption detection
- Preemptive generation: Generate responses based on interim transcripts
- Long utterance handling: Chunking and buffering

**Sub-packages**:
- `iface/`: VoiceSession interface definitions
- `internal/`: Private implementations
  - `lifecycle.go`: Start/stop operations
  - `state.go`: State machine
  - `audio_processing.go`: Audio processing pipeline
  - `agent_integration.go`: Agent integration
  - `s2s_integration.go`: S2S provider integration
  - `stt_integration.go`: STT integration
  - `timeout.go`: Session timeout management
  - `memory_integration.go`: Memory integration
  - `agent_context.go`: Agent context management
  - `transcript_processing.go`: Transcript processing
  - `streaming_stt.go`: Streaming STT support
  - `say.go`: TTS playback management
- `internal/mock/`: Mock implementations

**Dependencies**: `pkg/voice/stt`, `pkg/voice/tts`, `pkg/voice/s2s`, `pkg/voice/vad`, `pkg/voice/turndetection`, `pkg/voice/transport`, `pkg/voice/noise`, `pkg/agents`, `pkg/schema`, `pkg/monitoring`

---

### pkg/voice/backend

**Description**: Voice backend abstraction for provider-agnostic voice operations.

**Intent**: Provider-agnostic voice backend interface supporting multiple providers (LiveKit, Pipecat, Vocode, Vapi, Cartesia, Twilio) with unified API.

**Key Components**:
- VoiceBackend interface: Core backend operations
- Session management: Thread-safe session lifecycle
- Pipeline types: STT/TTS and S2S pipeline support
- Multi-user scalability: 100+ concurrent conversations

**Sub-packages**:
- `iface/`: VoiceBackend interface definitions
- `internal/`: Private implementations (session manager, pipeline orchestrator)
- `providers/mock/`: Mock backend provider
- `providers/vapi/`: Vapi provider
- `providers/vocode/`: Vocode provider
- `providers/pipecat/`: Pipecat provider
- `providers/livekit/`: LiveKit provider
- `providers/cartesia/`: Cartesia provider

**Dependencies**: `pkg/voice/session`, `pkg/voice/stt`, `pkg/voice/tts`, `pkg/voice/s2s`, `pkg/schema`, `pkg/config`, `pkg/monitoring`

---

### pkg/voice/iface

**Description**: Shared voice interfaces and types for all voice packages.

**Intent**: Common interfaces and types used across all voice packages to ensure consistency and interoperability.

**Key Components**:
- Shared interfaces: STTProvider, TTSProvider, S2SProvider, VADProvider, etc.
- Common types: Audio formats, session states, pipeline types

**Dependencies**: `pkg/schema`

---

### pkg/voice/providers

**Description**: Provider-specific voice implementations (e.g., Twilio).

**Intent**: Provider-specific implementations that integrate with external voice services.

**Sub-packages**:
- `twilio/`: Twilio voice provider implementation

**Dependencies**: `pkg/voice/backend`, `pkg/voice/session`, `pkg/voice/stt`, `pkg/voice/tts`, `pkg/schema`, `pkg/config`, `pkg/monitoring`

---

## Messaging Packages

### pkg/messaging

**Description**: Multi-channel conversational backends for SMS, WhatsApp, and chat messaging.

**Intent**: Multi-channel messaging with conversation management, message sending/receiving, webhook handling, and memory persistence.

**Key Components**:
- ConversationalBackend interface: Core messaging operations
- Conversation management: Create, list, update, delete conversations
- Message operations: Send and receive messages
- Participant management: Add/remove participants
- Webhook handling: Event-driven message processing

**Sub-packages**:
- `iface/`: ConversationalBackend interface definitions
- `internal/`: Private implementations
- `providers/twilio/`: Twilio Conversations API provider

**Dependencies**: `pkg/schema`, `pkg/config`, `pkg/monitoring`, `pkg/memory`

---

## Supporting Packages

### pkg/documentloaders

**Description**: Document loading from various sources (files, directories, URLs).

**Intent**: Load documents from various sources into the framework's document format for RAG applications.

**Key Components**:
- DocumentLoader interface: Core loading operations
- RecursiveDirectoryLoader: Recursively loads from directories
- TextLoader: Loads single text files
- Lazy loading: Streaming support via LazyLoad()

**Sub-packages**:
- `iface/`: DocumentLoader interface definitions
- `internal/`: Private implementations
- `providers/directory/`: Directory loader
- `providers/text/`: Text file loader

**Dependencies**: `pkg/schema`, `pkg/config`, `pkg/monitoring`

---

### pkg/textsplitters

**Description**: Text splitting utilities for chunking text and documents.

**Intent**: Split text and documents into chunks suitable for embedding and retrieval in RAG pipelines.

**Key Components**:
- TextSplitter interface: Core splitting operations
- RecursiveCharacterTextSplitter: Recursively splits using separator hierarchy
- MarkdownTextSplitter: Markdown-aware splitting

**Sub-packages**:
- `iface/`: TextSplitter interface definitions
- `internal/`: Private implementations
- `providers/`: Splitter implementations

**Dependencies**: `pkg/schema`, `pkg/config`, `pkg/monitoring`

---

### pkg/multimodal

**Description**: Multimodal AI capabilities for image, video, and audio processing.

**Intent**: Unified interface for multimodal model operations (text+image/video/audio) that integrates with existing LLM and embedding providers.

**Key Components**:
- MultimodalModel interface: Core multimodal operations
- Provider registry: Global registry pattern
- Capability detection: Automatic routing based on provider capabilities
- Graceful fallbacks: Falls back to text-only when modality not supported

**Sub-packages**:
- `iface/`: MultimodalModel interface definitions
- `internal/`: Private implementations
- `providers/openai/`: OpenAI multimodal provider
- `providers/anthropic/`: Anthropic multimodal provider
- `providers/google/`: Google multimodal provider
- `providers/gemini/`: Google Gemini multimodal provider
- `providers/xai/`: xAI multimodal provider
- `providers/pixtral/`: Pixtral multimodal provider
- `providers/qwen/`: Qwen multimodal provider
- `registry/`: Provider registry
- `types/`: Multimodal type definitions

**Dependencies**: `pkg/llms`, `pkg/embeddings`, `pkg/schema`, `pkg/config`, `pkg/monitoring`

---

## Package Dependency Hierarchy

```
Foundation Layer (No Dependencies)
├── pkg/schema
├── pkg/core
├── pkg/config
└── pkg/monitoring

Provider Layer (Depends on Foundation)
├── pkg/llms
├── pkg/embeddings
├── pkg/vectorstores
└── pkg/prompts

Higher-Level Layer (Depends on Providers)
├── pkg/chatmodels
├── pkg/memory
├── pkg/retrievers
├── pkg/agents
├── pkg/orchestration
└── pkg/server

Voice Layer (Depends on Foundation & Higher-Level)
├── pkg/voice/stt
├── pkg/voice/tts
├── pkg/voice/s2s
├── pkg/voice/vad
├── pkg/voice/turndetection
├── pkg/voice/transport
├── pkg/voice/noise
├── pkg/voice/session (depends on all voice packages)
└── pkg/voice/backend (depends on session)

Messaging Layer (Depends on Foundation & Higher-Level)
└── pkg/messaging

Supporting Layer (Depends on Foundation)
├── pkg/documentloaders
├── pkg/textsplitters
└── pkg/multimodal
```

---

## Package Relationships

### Voice Package Relationships

```
pkg/voice/backend
  └── Uses: pkg/voice/session, pkg/voice/stt, pkg/voice/tts, pkg/voice/s2s

pkg/voice/session
  └── Uses: pkg/voice/stt, pkg/voice/tts, pkg/voice/s2s, pkg/voice/vad,
            pkg/voice/turndetection, pkg/voice/transport, pkg/voice/noise,
            pkg/agents, pkg/memory

pkg/voice/providers/twilio
  └── Uses: pkg/voice/backend, pkg/voice/stt, pkg/voice/tts
  └── Could Use: pkg/voice/session, pkg/voice/s2s, pkg/voice/vad,
                 pkg/voice/turndetection, pkg/voice/noise, pkg/voice/transport,
                 pkg/memory, pkg/orchestration
```

### RAG Pipeline Relationships

```
pkg/documentloaders → pkg/textsplitters → pkg/embeddings → pkg/vectorstores → pkg/retrievers → pkg/agents
```

### Agent Pipeline Relationships

```
pkg/llms → pkg/chatmodels → pkg/agents → pkg/orchestration → pkg/server
```

---

## Integration Patterns

### Session Package Integration Pattern

The `pkg/voice/session` package provides a complete voice session management solution that can be used by voice backend providers:

1. **Lifecycle Management**: Automatic start/stop with state machine
2. **Audio Processing**: Complete STT → Agent → TTS pipeline
3. **Error Recovery**: Automatic retry with exponential backoff
4. **Advanced Features**: Interruption handling, preemptive generation, long utterance handling
5. **Provider Integration**: Supports STT, TTS, S2S, VAD, Turn Detection, Transport, Noise Cancellation

### S2S Package Integration Pattern

The `pkg/voice/s2s` package provides speech-to-speech capabilities that can be used as an alternative to STT+TTS:

1. **Direct Conversion**: Speech-to-speech without intermediate text
2. **Lower Latency**: Faster than STT+TTS pipeline
3. **Reasoning Modes**: Built-in or external agent reasoning
4. **Streaming Support**: Real-time bidirectional streaming

### Memory Package Integration Pattern

The `pkg/memory` package provides conversation memory that can be integrated with voice sessions:

1. **Context Preservation**: Maintain conversation history across turns
2. **Multiple Types**: Buffer, window, summary, vectorstore memory
3. **Agent Integration**: Seamless integration with agents

---

## Notes

- All packages follow Beluga AI Framework design patterns (ISP, DIP, SRP)
- All packages use OpenTelemetry for observability
- All packages use `pkg/schema` for data structures
- All packages use `pkg/config` for configuration
- All packages use `pkg/monitoring` for observability
- Multi-provider packages use global registry pattern
- All packages include comprehensive testing utilities
