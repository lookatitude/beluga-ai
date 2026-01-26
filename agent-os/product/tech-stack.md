# Tech Stack

## Language & Runtime

- **Go 1.24+** — Primary language (go 1.24.1, toolchain go1.24.2)
- **Module Path**: `github.com/lookatitude/beluga-ai`
- **License**: MIT

## LLM Providers

- **OpenAI** — `github.com/sashabaranov/go-openai` v1.41.2
- **Anthropic** — `github.com/anthropics/anthropic-sdk-go` v0.2.0-beta.4
- **AWS Bedrock** — `github.com/aws/aws-sdk-go-v2` v1.41.1
- **Ollama** — `github.com/ollama/ollama` v0.14.2
- **Gemini** — Google Gemini models
- **Google** — Google AI services
- **Grok** — xAI Grok
- **Groq** — Groq cloud service

## Vector Stores

- **In-memory** — Built-in
- **PgVector** — PostgreSQL with pgvector extension
- **Pinecone** — Cloud vector database
- **Qdrant** — Qdrant vector database
- **Weaviate** — Weaviate vector database
- **Chroma** — Chroma vector database

## Embedding Providers

- **OpenAI** — OpenAI embeddings and multimodal
- **Ollama** — Local embeddings
- **Cohere** — Cohere embeddings
- **Google Multimodal** — Google multimodal embeddings

## Voice Providers

### Backends
- **LiveKit** — `github.com/livekit/protocol` v1.43.4
- **Twilio** — `github.com/twilio/twilio-go` v1.29.1
- **Cartesia**, **Pipecat**, **VAPI**, **Vocode**

### Speech-to-Text (STT)
- Azure, Deepgram, Google Cloud Speech, OpenAI Whisper

### Text-to-Speech (TTS)
- Azure, ElevenLabs, Google Cloud TTS, OpenAI TTS

### Speech-to-Speech (S2S)
- Amazon Nova, Gemini, Grok, OpenAI Realtime

### Voice Activity Detection (VAD)
- Energy-based, RNNoise, Silero, WebRTC

### Transport & Processing
- **WebRTC** — `github.com/pion/webrtc/v4` v4.2.2
- **WebSocket** — `github.com/gorilla/websocket` v1.5.3
- Noise reduction (RNNoise, Spectral, WebRTC)
- Turn detection (Heuristic, ONNX)

## Multimodal Providers

- Anthropic, DeepSeek, Gemini, Gemma, Google, OpenAI, Phi, Pixtral, Qwen, xAI

## Observability

- `go.opentelemetry.io/otel` v1.39.0 — Tracing and metrics
- OTEL exporters: Prometheus, OTLP/gRPC, stdout
- OpenTelemetry as single observability stack

## Workflow Orchestration

- `go.temporal.io/sdk` v1.39.0 — Temporal workflow engine

## Configuration & Validation

- `github.com/spf13/viper` v1.21.0 — Configuration management
- `github.com/go-playground/validator/v10` v10.30.1 — Struct validation

## HTTP & Server

- `github.com/gorilla/mux` v1.8.1 — HTTP routing
- `github.com/twitchtv/twirp` v8.1.3 — RPC framework
- REST and MCP server providers

## Data & Persistence

- `github.com/lib/pq` v1.10.9 — PostgreSQL driver
- `github.com/google/uuid` v1.6.0 — UUID generation

## Testing

- `github.com/stretchr/testify` v1.11.1 — Assertions and mocking
- Table-driven tests throughout
- Race detection enabled (`-race`)

## Build & CI

- **Make** — Build, test, lint, security, docs
- **golangci-lint** v2.6.2 — Linting
- **gosec** — Security (SAST)
- **govulncheck** — Vulnerability checking
- **gitleaks** v8.18.0 — Secret detection
- **Trivy** — Container/filesystem scan
- **GitHub Actions** — CI/CD pipelines
- **pre-commit** — Git hooks

## Documentation

- **gomarkdoc** — API docs generation
- **Docusaurus** 3.x — Documentation site
