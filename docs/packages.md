# Beluga AI v2 â€” Package Architecture

## 1. Module / Package Layout

```
beluga-ai/
â”œâ”€â”€ go.mod
â”‚
â”œâ”€â”€ core/                    # Foundation â€” zero external deps beyond stdlib + otel
â”‚   â”œâ”€â”€ stream.go            # Event[T], Stream[T], Fan-in/Fan-out, Pipe()
â”‚   â”œâ”€â”€ runnable.go          # Runnable interface (Invoke, Stream, Batch)
â”‚   â”œâ”€â”€ batch.go             # BatchInvoke[I,O] with concurrency control
â”‚   â”œâ”€â”€ context.go           # Session context, cancel propagation, tenant
â”‚   â”œâ”€â”€ tenant.go            # TenantID, WithTenant(), GetTenant()
â”‚   â”œâ”€â”€ lifecycle.go         # Lifecycle interface (Start, Stop, Health), App struct
â”‚   â”œâ”€â”€ errors.go            # Typed error codes, Is/As helpers, IsRetryable()
â”‚   â””â”€â”€ option.go            # Functional options helpers
â”‚
â”œâ”€â”€ schema/                  # Shared types â€” no business logic
â”‚   â”œâ”€â”€ message.go           # Message interface, HumanMsg, AIMsg, SystemMsg, ToolMsg
â”‚   â”œâ”€â”€ content.go           # ContentPart: Text, Image, Audio, Video, File
â”‚   â”œâ”€â”€ tool.go              # ToolCall, ToolResult, ToolDefinition
â”‚   â”œâ”€â”€ document.go          # Document with metadata
â”‚   â”œâ”€â”€ event.go             # AgentEvent, StreamEvent, LifecycleEvent
â”‚   â””â”€â”€ session.go           # Session, Turn, ConversationState
â”‚
â”œâ”€â”€ config/                  # Configuration loading
â”‚   â”œâ”€â”€ config.go            # Load[T], Validate, env + file + struct tags
â”‚   â”œâ”€â”€ provider.go          # ProviderConfig base type
â”‚   â””â”€â”€ watch.go             # Watcher interface, hot-reload (fsnotify, Consul, etcd, K8s)
â”‚
â”œâ”€â”€ o11y/                    # Observability
â”‚   â”œâ”€â”€ tracer.go            # OTel tracer wrapper
â”‚   â”œâ”€â”€ meter.go             # OTel meter wrapper
â”‚   â”œâ”€â”€ logger.go            # Structured logging (slog)
â”‚   â”œâ”€â”€ health.go            # Health checks
â”‚   â”œâ”€â”€ exporter.go          # TraceExporter interface (LLM-specific: prompts, tokens, cost)
â”‚   â”œâ”€â”€ dashboard/           # Built-in dev telemetry dashboard
â”‚   â”‚   â””â”€â”€ dashboard.go     # Embedded web UI: traces, costs, tool calls, prompt playground
â”‚   â””â”€â”€ adapters/
â”‚       â”œâ”€â”€ langfuse/        # Langfuse trace exporter
â”‚       â””â”€â”€ phoenix/         # Arize Phoenix trace exporter
â”‚
â”œâ”€â”€ llm/                     # LLM abstraction
â”‚   â”œâ”€â”€ llm.go               # ChatModel interface
â”‚   â”œâ”€â”€ options.go           # GenerateOptions (temp, max_tokens, tools, response_format, etc.)
â”‚   â”œâ”€â”€ registry.go          # Register(), New(), List() â€” provider registry
â”‚   â”œâ”€â”€ hooks.go             # LLM-level hooks (BeforeGenerate, AfterGenerate, etc.)
â”‚   â”œâ”€â”€ middleware.go         # Retry, rate-limit, cache, logging, guardrail, fallback middleware
â”‚   â”œâ”€â”€ router.go            # LLM Router: routes across backends with pluggable strategies
â”‚   â”œâ”€â”€ structured.go        # StructuredOutput[T]: JSON Schema from Go structs, parse + validate + retry
â”‚   â”œâ”€â”€ context.go           # ContextManager: fit messages within token budget (Truncate, Summarize, Semantic, Adaptive)
â”‚   â”œâ”€â”€ tokenizer.go         # Tokenizer interface: Count, CountMessages, Encode, Decode (tiktoken, SentencePiece)
â”‚   â”œâ”€â”€ ratelimit.go         # ProviderLimits: RPM, TPM, MaxConcurrent, CooldownOnRetry
â”‚   â””â”€â”€ providers/
â”‚       â”œâ”€â”€ openai/           # OpenAI + Azure OpenAI (GPT-4o, GPT-4.1, o1, o3)
â”‚       â”œâ”€â”€ anthropic/        # Claude models (Opus 4, Sonnet 4.5, Haiku 4.5)
â”‚       â”œâ”€â”€ google/           # Gemini + Vertex AI (2.5 Pro, 2.5 Flash, Gemini 3)
â”‚       â”œâ”€â”€ ollama/           # Local models
â”‚       â”œâ”€â”€ bedrock/          # AWS Bedrock (multi-model gateway)
â”‚       â”œâ”€â”€ groq/             # Ultra-fast inference (LPU)
â”‚       â”œâ”€â”€ mistral/          # Mistral AI (Large, Medium, Codestral)
â”‚       â”œâ”€â”€ deepseek/         # DeepSeek (V3, R1, Coder V2)
â”‚       â”œâ”€â”€ xai/              # xAI Grok (3, 3.5)
â”‚       â”œâ”€â”€ cohere/           # Cohere (Command R+)
â”‚       â”œâ”€â”€ together/         # Together AI (200+ models)
â”‚       â””â”€â”€ fireworks/        # Fireworks AI (FireAttention)
â”‚
â”œâ”€â”€ tool/                    # Tool system
â”‚   â”œâ”€â”€ tool.go              # Tool interface: Name, Description, Schema, Execute
â”‚   â”œâ”€â”€ functool.go          # NewFuncTool() â€” wrap any Go function as a Tool
â”‚   â”œâ”€â”€ registry.go          # ToolRegistry: Add, Get, List, Remove
â”‚   â”œâ”€â”€ hooks.go             # Tool hooks (BeforeExecute, AfterExecute, OnError)
â”‚   â”œâ”€â”€ mcp.go               # MCP client â€” discovers & wraps MCP servers as Tools
â”‚   â”œâ”€â”€ mcp_registry.go      # MCPRegistry: Search, Discover MCP servers from registries
â”‚   â”œâ”€â”€ middleware.go         # Auth, rate-limit, timeout wrappers
â”‚   â””â”€â”€ builtin/             # Calculator, HTTP, Shell, Code execution
â”‚
â”œâ”€â”€ memory/                  # Conversation & knowledge memory
â”‚   â”œâ”€â”€ memory.go            # Memory interface: Save, Load, Search, Clear
â”‚   â”œâ”€â”€ registry.go          # Register(), New(), List()
â”‚   â”œâ”€â”€ store.go             # MessageStore interface (backend abstraction)
â”‚   â”œâ”€â”€ hooks.go             # Memory hooks (BeforeSave, AfterLoad, etc.)
â”‚   â”œâ”€â”€ middleware.go         # Memory middleware (trace, cache, TTL)
â”‚   â”œâ”€â”€ buffer.go            # Full-history buffer
â”‚   â”œâ”€â”€ window.go            # Sliding window (last N turns)
â”‚   â”œâ”€â”€ summary.go           # LLM-summarised history
â”‚   â”œâ”€â”€ entity.go            # Entity extraction & tracking
â”‚   â”œâ”€â”€ semantic.go          # Vector-backed semantic memory
â”‚   â”œâ”€â”€ graph.go             # Knowledge graph memory (Neo4j/Memgraph/Cayley)
â”‚   â”œâ”€â”€ composite.go         # CompositeMemory: Working + Episodic + Semantic + Graph
â”‚   â””â”€â”€ stores/
â”‚       â”œâ”€â”€ inmemory/
â”‚       â”œâ”€â”€ redis/
â”‚       â”œâ”€â”€ postgres/
â”‚       â”œâ”€â”€ sqlite/
â”‚       â”œâ”€â”€ neo4j/            # Graph memory backend
â”‚       â”œâ”€â”€ memgraph/         # In-memory graph database
â”‚       â””â”€â”€ dragonfly/        # Redis-compatible, 25x faster
â”‚
â”œâ”€â”€ rag/                     # RAG pipeline
â”‚   â”œâ”€â”€ embedding/
â”‚   â”‚   â”œâ”€â”€ embedder.go      # Embedder interface
â”‚   â”‚   â”œâ”€â”€ registry.go      # Register(), New(), List()
â”‚   â”‚   â”œâ”€â”€ hooks.go         # BeforeEmbed, AfterEmbed
â”‚   â”‚   â””â”€â”€ providers/
â”‚   â”‚       â”œâ”€â”€ openai/       # text-embedding-3-large
â”‚   â”‚       â”œâ”€â”€ google/       # text-embedding-005
â”‚   â”‚       â”œâ”€â”€ ollama/       # nomic-embed, mxbai
â”‚   â”‚       â”œâ”€â”€ cohere/       # Embed V4, V3
â”‚   â”‚       â”œâ”€â”€ voyage/       # voyage-3-large (top MTEB)
â”‚   â”‚       â””â”€â”€ jina/         # jina-embeddings-v3
â”‚   â”œâ”€â”€ vectorstore/
â”‚   â”‚   â”œâ”€â”€ store.go          # VectorStore interface: Add, Search, Delete
â”‚   â”‚   â”œâ”€â”€ registry.go       # Register(), New(), List()
â”‚   â”‚   â”œâ”€â”€ hooks.go          # BeforeAdd, AfterSearch
â”‚   â”‚   â””â”€â”€ providers/
â”‚   â”‚       â”œâ”€â”€ inmemory/
â”‚   â”‚       â”œâ”€â”€ pgvector/     # PostgreSQL (471 QPS at 99% recall on 50M vectors)
â”‚   â”‚       â”œâ”€â”€ qdrant/       # Rust-based, best real-time perf
â”‚   â”‚       â”œâ”€â”€ pinecone/     # Fully managed, serverless
â”‚   â”‚       â”œâ”€â”€ chroma/       # Developer-friendly
â”‚   â”‚       â”œâ”€â”€ weaviate/     # Best hybrid search (vector + BM25 + metadata)
â”‚   â”‚       â”œâ”€â”€ milvus/       # Enterprise scale, GPU-accelerated
â”‚   â”‚       â”œâ”€â”€ turbopuffer/  # S3-based serverless, 10x cheaper
â”‚   â”‚       â”œâ”€â”€ redis/        # RediSearch â€” ultra-low latency
â”‚   â”‚       â”œâ”€â”€ elasticsearch/ # Mature text + vector hybrid
â”‚   â”‚       â””â”€â”€ sqlitevec/    # Embedded, edge/mobile/CLI
â”‚   â”œâ”€â”€ retriever/
â”‚   â”‚   â”œâ”€â”€ retriever.go      # Retriever interface
â”‚   â”‚   â”œâ”€â”€ registry.go       # Register(), New(), List()
â”‚   â”‚   â”œâ”€â”€ hooks.go          # BeforeRetrieve, AfterRetrieve, OnRerank
â”‚   â”‚   â”œâ”€â”€ middleware.go     # Retriever middleware (cache, trace, etc.)
â”‚   â”‚   â”œâ”€â”€ vector.go         # VectorStoreRetriever
â”‚   â”‚   â”œâ”€â”€ multiquery.go     # Multi-query expansion
â”‚   â”‚   â”œâ”€â”€ rerank.go         # Re-ranking retriever
â”‚   â”‚   â”œâ”€â”€ ensemble.go       # Ensemble retriever
â”‚   â”‚   â””â”€â”€ hyde.go           # Hypothetical Document Embedding retriever
â”‚   â”œâ”€â”€ loader/
â”‚   â”‚   â”œâ”€â”€ loader.go         # DocumentLoader interface
â”‚   â”‚   â”œâ”€â”€ pipeline.go       # LoaderPipeline: chain loaders + transformers
â”‚   â”‚   â”œâ”€â”€ text.go
â”‚   â”‚   â”œâ”€â”€ pdf.go            # PDF with table extraction
â”‚   â”‚   â”œâ”€â”€ html.go           # HTML with boilerplate removal
â”‚   â”‚   â”œâ”€â”€ web.go            # Web crawler (Firecrawl, Colly)
â”‚   â”‚   â”œâ”€â”€ csv.go
â”‚   â”‚   â”œâ”€â”€ json.go
â”‚   â”‚   â”œâ”€â”€ docx.go           # Word documents
â”‚   â”‚   â”œâ”€â”€ pptx.go           # PowerPoint
â”‚   â”‚   â”œâ”€â”€ xlsx.go           # Excel
â”‚   â”‚   â”œâ”€â”€ markdown.go
â”‚   â”‚   â”œâ”€â”€ code.go           # Source code with AST awareness
â”‚   â”‚   â”œâ”€â”€ confluence.go     # Confluence wiki pages
â”‚   â”‚   â”œâ”€â”€ notion.go         # Notion databases/pages
â”‚   â”‚   â”œâ”€â”€ github.go         # Repos, issues, PRs
â”‚   â”‚   â”œâ”€â”€ s3.go             # AWS S3
â”‚   â”‚   â”œâ”€â”€ gcs.go            # Google Cloud Storage
â”‚   â”‚   â””â”€â”€ ocr.go            # OCR via Tesseract or Google Vision
â”‚   â””â”€â”€ splitter/
â”‚       â”œâ”€â”€ splitter.go       # TextSplitter interface
â”‚       â”œâ”€â”€ recursive.go
â”‚       â””â”€â”€ markdown.go
â”‚
â”œâ”€â”€ agent/                   # Agent runtime
â”‚   â”œâ”€â”€ agent.go             # Agent interface + BaseAgent embeddable struct
â”‚   â”œâ”€â”€ base.go              # BaseAgent: ID, Persona, Tools, Children, Card
â”‚   â”œâ”€â”€ persona.go           # Role, Goal, Backstory (RGB framework)
â”‚   â”œâ”€â”€ executor.go          # Reasoning loop: delegates to Planner
â”‚   â”œâ”€â”€ planner.go           # Planner interface + PlannerState + Action types
â”‚   â”œâ”€â”€ registry.go          # RegisterPlanner(), NewPlanner(), ListPlanners()
â”‚   â”œâ”€â”€ hooks.go             # Hooks struct + ComposeHooks()
â”‚   â”œâ”€â”€ middleware.go         # Agent middleware (retry, trace, etc.)
â”‚   â”œâ”€â”€ bus.go               # EventBus: agent-to-agent async messaging (in-memory, NATS, Redis)
â”‚   â”œâ”€â”€ react.go             # ReAct planner implementation
â”‚   â”œâ”€â”€ planexecute.go       # Plan-and-Execute planner implementation
â”‚   â”œâ”€â”€ reflection.go        # Reflection planner implementation
â”‚   â”œâ”€â”€ structured.go        # Structured-output agent
â”‚   â”œâ”€â”€ conversational.go    # Optimised for multi-turn chat
â”‚   â”œâ”€â”€ handoff.go           # Agent-to-agent handoff within a session
â”‚   â”œâ”€â”€ card.go              # A2A AgentCard (capability advertisement)
â”‚   â””â”€â”€ workflow/             # Built-in workflow agents
â”‚       â”œâ”€â”€ sequential.go    # SequentialAgent
â”‚       â”œâ”€â”€ parallel.go      # ParallelAgent
â”‚       â””â”€â”€ loop.go          # LoopAgent
â”‚
â”œâ”€â”€ voice/                   # Voice / multimodal pipeline
â”‚   â”œâ”€â”€ pipeline.go          # VoicePipeline: STT â†’ LLM â†’ TTS (cascading)
â”‚   â”œâ”€â”€ hybrid.go            # HybridPipeline: S2S + cascade with switch policy
â”‚   â”œâ”€â”€ session.go           # VoiceSession: manages audio state & turns
â”‚   â”œâ”€â”€ vad.go               # VAD interface
â”‚   â”œâ”€â”€ stt/
â”‚   â”‚   â”œâ”€â”€ stt.go           # STT interface (streaming)
â”‚   â”‚   â””â”€â”€ providers/
â”‚   â”‚       â”œâ”€â”€ deepgram/     # Nova-3, Nova-3 Medical, Flux
â”‚   â”‚       â”œâ”€â”€ assemblyai/   # Slam-1 SLM
â”‚   â”‚       â”œâ”€â”€ whisper/      # OpenAI Whisper / GPT-4o Transcribe
â”‚   â”‚       â”œâ”€â”€ elevenlabs/   # Scribe v2
â”‚   â”‚       â”œâ”€â”€ groq/         # Whisper on LPU (200x realtime)
â”‚   â”‚       â””â”€â”€ gladia/       # Solaria-1 (100 languages)
â”‚   â”œâ”€â”€ tts/
â”‚   â”‚   â”œâ”€â”€ tts.go           # TTS interface (streaming)
â”‚   â”‚   â””â”€â”€ providers/
â”‚   â”‚       â”œâ”€â”€ elevenlabs/   # Highest quality TTS
â”‚   â”‚       â”œâ”€â”€ cartesia/     # Sonic â€” ultra-low latency
â”‚   â”‚       â”œâ”€â”€ openai/       # OpenAI TTS
â”‚   â”‚       â”œâ”€â”€ playht/       # PlayDialog, Play3.0-mini
â”‚   â”‚       â”œâ”€â”€ groq/         # PlayAI TTS on LPU
â”‚   â”‚       â”œâ”€â”€ fish/         # Fish-Speech (multilingual)
â”‚   â”‚       â”œâ”€â”€ lmnt/         # Low-latency conversational
â”‚   â”‚       â””â”€â”€ smallest/     # Ultra-low cost, voice cloning
â”‚   â”œâ”€â”€ s2s/
â”‚   â”‚   â”œâ”€â”€ s2s.go           # S2S interface (bidirectional)
â”‚   â”‚   â””â”€â”€ providers/
â”‚   â”‚       â”œâ”€â”€ openai_realtime/ # GPT-4o native voice-to-voice
â”‚   â”‚       â”œâ”€â”€ gemini_live/    # Google bidirectional voice
â”‚   â”‚       â””â”€â”€ nova/           # Amazon Nova S2S
â”‚   â””â”€â”€ transport/
â”‚       â”œâ”€â”€ transport.go      # AudioTransport interface
â”‚       â”œâ”€â”€ websocket.go
â”‚       â”œâ”€â”€ livekit.go        # LiveKit room integration
â”‚       â””â”€â”€ daily.go          # Daily.co transport
â”‚
â”œâ”€â”€ orchestration/           # Workflow composition
â”‚   â”œâ”€â”€ node.go              # Node interface (extension point for custom steps)
â”‚   â”œâ”€â”€ hooks.go             # BeforeStep, AfterStep, OnBranch
â”‚   â”œâ”€â”€ chain.go             # Sequential pipeline
â”‚   â”œâ”€â”€ graph.go             # DAG with conditional edges
â”‚   â”œâ”€â”€ router.go            # Conditional routing (LLM or rule-based)
â”‚   â”œâ”€â”€ parallel.go          # Fan-out / fan-in
â”‚   â”œâ”€â”€ workflow.go          # In-process workflow (non-durable)
â”‚   â””â”€â”€ supervisor.go        # Multi-agent supervisor pattern
â”‚
â”œâ”€â”€ workflow/                # Durable workflow execution
â”‚   â”œâ”€â”€ executor.go          # DurableExecutor interface
â”‚   â”œâ”€â”€ activity.go          # LLMActivity, ToolActivity, HumanActivity wrappers
â”‚   â”œâ”€â”€ state.go             # WorkflowState: checkpoint, metadata, history
â”‚   â”œâ”€â”€ signal.go            # Signal types for HITL, inter-workflow communication
â”‚   â”œâ”€â”€ registry.go          # Register(), New(), List()
â”‚   â”œâ”€â”€ hooks.go             # BeforeActivity, AfterActivity, OnSignal, OnRetry
â”‚   â”œâ”€â”€ middleware.go         # Workflow-level middleware
â”‚   â”œâ”€â”€ config.go            # RetryPolicy, Timeouts, TaskQueues
â”‚   â”œâ”€â”€ patterns/            # Pre-built workflow patterns
â”‚   â”‚   â”œâ”€â”€ agent_loop.go    # ReAct/Plan-Execute as durable workflow
â”‚   â”‚   â”œâ”€â”€ research.go      # Multi-source research pipeline
â”‚   â”‚   â”œâ”€â”€ approval.go      # Human approval workflow
â”‚   â”‚   â”œâ”€â”€ scheduled.go     # Periodic/ambient agent
â”‚   â”‚   â””â”€â”€ saga.go          # Saga pattern with compensating actions
â”‚   â””â”€â”€ providers/
â”‚       â”œâ”€â”€ temporal/         # Temporal Go SDK integration
â”‚       â”‚   â”œâ”€â”€ executor.go   # TemporalExecutor implements DurableExecutor
â”‚       â”‚   â”œâ”€â”€ worker.go     # Activity worker setup
â”‚       â”‚   â”œâ”€â”€ activities.go # Activity implementations
â”‚       â”‚   â””â”€â”€ converter.go  # Payload codec (encryption, compression)
â”‚       â”œâ”€â”€ inmemory/         # In-process (dev/test, no durability)
â”‚       â””â”€â”€ nats/             # NATS JetStream (lightweight alternative)
â”‚
â”œâ”€â”€ protocol/                # External protocol support
â”‚   â”œâ”€â”€ mcp/
â”‚   â”‚   â”œâ”€â”€ server.go        # Expose tools as MCP server
â”‚   â”‚   â””â”€â”€ client.go        # Connect to MCP servers
â”‚   â”œâ”€â”€ a2a/
â”‚   â”‚   â”œâ”€â”€ server.go        # Expose agent as A2A remote agent
â”‚   â”‚   â”œâ”€â”€ client.go        # Call remote A2A agents
â”‚   â”‚   â””â”€â”€ card.go          # AgentCard JSON generation
â”‚   â””â”€â”€ rest/
â”‚       â””â”€â”€ server.go        # REST/SSE API for agents
â”‚
â”œâ”€â”€ guard/                   # Safety & guardrails
â”‚   â”œâ”€â”€ guard.go             # Guard interface: Check(input) â†’ (ok, reason)
â”‚   â”œâ”€â”€ registry.go          # Register(), New(), List()
â”‚   â”œâ”€â”€ content.go           # Content moderation
â”‚   â”œâ”€â”€ pii.go               # PII detection
â”‚   â”œâ”€â”€ injection.go         # Prompt injection detection
â”‚   â””â”€â”€ adapters/
â”‚       â”œâ”€â”€ nemo/            # NVIDIA NeMo Guardrails integration
â”‚       â”œâ”€â”€ guardrailsai/    # Guardrails AI Validator Hub
â”‚       â”œâ”€â”€ llmguard/        # LLM Guard scanners
â”‚       â””â”€â”€ lakera/          # Lakera prompt injection API
â”‚
â”œâ”€â”€ eval/                    # Evaluation framework
â”‚   â”œâ”€â”€ eval.go              # Metric interface, EvalSample, EvalReport
â”‚   â”œâ”€â”€ runner.go            # EvalRunner: parallel execution, reporting
â”‚   â”œâ”€â”€ dataset.go           # Dataset management (load, save, augment)
â”‚   â””â”€â”€ metrics/
â”‚       â”œâ”€â”€ faithfulness.go   # Output grounded in retrieved context?
â”‚       â”œâ”€â”€ relevance.go      # Output answers the question?
â”‚       â”œâ”€â”€ hallucination.go  # Output contains fabricated facts?
â”‚       â”œâ”€â”€ toxicity.go       # Output contains harmful content?
â”‚       â”œâ”€â”€ latency.go        # End-to-end timing
â”‚       â””â”€â”€ cost.go           # Token cost per interaction
â”‚
â”œâ”€â”€ cache/                   # Caching layer
â”‚   â”œâ”€â”€ cache.go             # Cache interface: Get, Set, GetSemantic
â”‚   â”œâ”€â”€ semantic.go          # Embedding-based similarity cache
â”‚   â””â”€â”€ providers/
â”‚       â”œâ”€â”€ inmemory/         # LRU cache
â”‚       â”œâ”€â”€ redis/
â”‚       â””â”€â”€ dragonfly/        # Redis-compatible, 25x faster
â”‚
â”œâ”€â”€ hitl/                    # Human-in-the-loop
â”‚   â”œâ”€â”€ hitl.go              # InteractionRequest, InteractionResponse, InteractionType
â”‚   â”œâ”€â”€ approval.go          # Approval workflow primitives
â”‚   â”œâ”€â”€ feedback.go          # Feedback collection
â”‚   â””â”€â”€ notification.go      # Notification dispatching (Slack, email, webhook)
â”‚
â”œâ”€â”€ auth/                    # Agent authentication & authorization
â”‚   â”œâ”€â”€ auth.go              # Permission, Policy interface
â”‚   â”œâ”€â”€ rbac.go              # Role-based access control
â”‚   â”œâ”€â”€ abac.go              # Attribute-based access control
â”‚   â””â”€â”€ opa.go               # Open Policy Agent integration
â”‚
â”œâ”€â”€ resilience/              # Production resilience patterns
â”‚   â”œâ”€â”€ circuitbreaker.go    # Stop calling failing provider after N failures
â”‚   â”œâ”€â”€ hedge.go             # Send to multiple providers, use first response
â”‚   â”œâ”€â”€ retry.go             # Exponential backoff with jitter, retryable errors
â”‚   â””â”€â”€ ratelimit.go         # Provider-aware: RPM, TPM, MaxConcurrent
â”‚
â”œâ”€â”€ state/                   # Shared agent state
â”‚   â”œâ”€â”€ state.go             # Store interface: Get, Set, Delete, Watch
â”‚   â””â”€â”€ providers/
â”‚       â”œâ”€â”€ inmemory/
â”‚       â”œâ”€â”€ redis/
â”‚       â””â”€â”€ postgres/
â”‚
â”œâ”€â”€ prompt/                  # Prompt management & versioning
â”‚   â”œâ”€â”€ template.go          # Template: Name, Version, Content, Variables
â”‚   â”œâ”€â”€ manager.go           # PromptManager interface: Get, Render, List
â”‚   â””â”€â”€ providers/
â”‚       â”œâ”€â”€ file/             # YAML-based prompt templates
â”‚       â”œâ”€â”€ db/               # Database-backed
â”‚       â””â”€â”€ langfuse/         # Langfuse-synced prompt management
â”‚
â”œâ”€â”€ server/                  # HTTP framework integration
â”‚   â”œâ”€â”€ adapter.go           # ServerAdapter interface
â”‚   â”œâ”€â”€ registry.go          # Register(), New(), List()
â”‚   â”œâ”€â”€ handler.go           # Standard http.Handler adapter
â”‚   â”œâ”€â”€ sse.go               # SSE streaming helper
â”‚   â””â”€â”€ adapters/
â”‚       â”œâ”€â”€ gin/              # Gin integration
â”‚       â”œâ”€â”€ fiber/            # Fiber integration
â”‚       â”œâ”€â”€ echo/             # Echo integration
â”‚       â”œâ”€â”€ chi/              # Chi integration
â”‚       â”œâ”€â”€ grpc/             # gRPC service definitions
â”‚       â”œâ”€â”€ connectgo/        # Connect-Go (gRPC-compatible with HTTP/1.1)
â”‚       â””â”€â”€ huma/             # Huma (auto-generated OpenAPI)
â”‚
â””â”€â”€ internal/                # Shared internal utilities
    â”œâ”€â”€ syncutil/             # sync primitives, worker pools
    â”œâ”€â”€ jsonutil/             # JSON schema generation from Go types
    â””â”€â”€ testutil/             # Mock providers for every interface, test helpers
        â”œâ”€â”€ mockllm/          # Mock ChatModel
        â”œâ”€â”€ mocktool/         # Mock Tool
        â”œâ”€â”€ mockmemory/       # Mock Memory + MessageStore
        â”œâ”€â”€ mockembedder/     # Mock Embedder
        â”œâ”€â”€ mockstore/        # Mock VectorStore
        â”œâ”€â”€ mockworkflow/     # Mock DurableExecutor
        â””â”€â”€ helpers.go        # Assertion helpers, stream builders
```

---

## 2. Core Interfaces

### 2.1 Event Stream

Everything flows as typed events through a unified `Stream` abstraction. Replaces v1's `<-chan any` with type safety and backpressure.

```go
// core/stream.go

// Event is the unit of data flowing through the system.
type Event[T any] struct {
    Type    EventType
    Payload T
    Err     error
    Meta    map[string]any // trace ID, latency, token usage, etc.
}

type EventType string

const (
    EventData       EventType = "data"
    EventToolCall   EventType = "tool_call"
    EventToolResult EventType = "tool_result"
    EventDone       EventType = "done"
    EventError      EventType = "error"
)

// Stream is a pull-based iterator over events.
type Stream[T any] interface {
    Next(ctx context.Context) bool
    Event() Event[T]
    Close() error
}

// FlowController provides backpressure signalling.
type FlowController interface {
    RequestPause(ctx context.Context) error
    RequestResume(ctx context.Context) error
    BufferSize() int
}

// BufferedStream wraps a stream with backpressure support.
type BufferedStream[T any] struct {
    inner  Stream[T]
    buffer chan Event[T]
    flow   FlowController
}
```

### 2.2 Runnable

```go
// core/runnable.go

// Runnable is the universal execution interface.
type Runnable interface {
    Invoke(ctx context.Context, input any, opts ...Option) (any, error)
    Stream(ctx context.Context, input any, opts ...Option) (Stream[any], error)
}

// Pipe composes two Runnables sequentially.
func Pipe(a, b Runnable) Runnable { ... }

// Parallel fans out to multiple Runnables and collects results.
func Parallel(runnables ...Runnable) Runnable { ... }
```

### 2.3 Batch Execution

```go
// core/batch.go

type BatchOptions struct {
    MaxConcurrency int            // max parallel executions
    BatchSize      int            // max items per batch API call
    Timeout        time.Duration  // per-item timeout
    RetryPolicy    *RetryPolicy
}

func BatchInvoke[I, O any](ctx context.Context, fn func(context.Context, I) (O, error), inputs []I, opts BatchOptions) ([]O, []error)
```

### 2.4 Lifecycle

```go
// core/lifecycle.go

type Lifecycle interface {
    Start(ctx context.Context) error
    Stop(ctx context.Context) error
    Health() HealthStatus
}

type App struct {
    components []Lifecycle
}

func (a *App) Register(components ...Lifecycle)
func (a *App) Shutdown(ctx context.Context) error // reverse order, with timeout
```

### 2.5 Errors

```go
// core/errors.go

type Error struct {
    Op      string    // "llm.generate", "tool.execute", etc.
    Code    ErrorCode
    Message string
    Err     error     // wrapped cause
}

type ErrorCode string
const (
    ErrRateLimit    ErrorCode = "rate_limit"
    ErrAuth         ErrorCode = "auth_error"
    ErrTimeout      ErrorCode = "timeout"
    ErrInvalidInput ErrorCode = "invalid_input"
    ErrToolFailed   ErrorCode = "tool_failed"
    ErrProviderDown ErrorCode = "provider_unavailable"
)

func (e *Error) Error() string   { ... }
func (e *Error) Unwrap() error   { return e.Err }
func (e *Error) Is(target error) bool { ... }
func IsRetryable(err error) bool { ... }
```

### 2.6 Tenant

```go
// core/tenant.go

type TenantID string

func WithTenant(ctx context.Context, id TenantID) context.Context
func GetTenant(ctx context.Context) TenantID
```

---

## 3. ChatModel (LLM Interface)

```go
// llm/llm.go

type ChatModel interface {
    Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error)
    Stream(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (core.Stream[schema.StreamChunk], error)
    BindTools(tools []tool.Tool) ChatModel
    ModelID() string
}
```

### 3.1 Router

```go
// llm/router.go

type RouterStrategy interface {
    Select(ctx context.Context, models []ChatModel, msgs []schema.Message) (ChatModel, error)
}

type Router struct {
    models    []ChatModel
    strategy  RouterStrategy
    fallbacks []ChatModel
}

// Router implements ChatModel â€” use anywhere a model is expected
func (r *Router) Generate(ctx context.Context, msgs []schema.Message, opts ...GenerateOption) (*schema.AIMessage, error) { ... }
```

### 3.2 Structured Output

```go
// llm/structured.go

type StructuredOutput[T any] struct {
    model   ChatModel
    schema  map[string]any  // auto-generated from T via json/struct tags
    retries int
}

func NewStructured[T any](model ChatModel, opts ...StructuredOption) *StructuredOutput[T]
func (s *StructuredOutput[T]) Generate(ctx context.Context, msgs []schema.Message) (T, error)
```

### 3.3 Context Manager

```go
// llm/context.go

type ContextManager interface {
    Fit(ctx context.Context, msgs []schema.Message, budget int) ([]schema.Message, error)
}

// Strategies: Truncate, Summarize, Semantic, Sliding, Adaptive
```

### 3.4 Tokenizer

```go
// llm/tokenizer.go

type Tokenizer interface {
    Count(text string) int
    CountMessages(msgs []schema.Message) int
    Encode(text string) []int
    Decode(tokens []int) string
}
```

---

## 4. Tool System

```go
// tool/tool.go

type Tool interface {
    Name() string
    Description() string
    InputSchema() map[string]any
    Execute(ctx context.Context, input map[string]any) (*ToolResult, error)
}

type ToolResult struct {
    Content []schema.ContentPart
    Err     error
}
```

### 4.1 FuncTool (wrap any Go function)

```go
// tool/functool.go

// Automatic JSON Schema generation from struct tags
type WeatherInput struct {
    City  string `json:"city" description:"City name" required:"true"`
    Units string `json:"units" description:"celsius or fahrenheit" default:"celsius"`
}

weatherTool := tool.NewFuncTool("get_weather", "Get current weather",
    func(ctx context.Context, input WeatherInput) (*tool.ToolResult, error) {
        return tool.TextResult(fmt.Sprintf("72Â°F in %s", input.City)), nil
    },
)
```

### 4.2 MCP Client

```go
// tool/mcp.go

func FromMCP(ctx context.Context, serverURL string, opts ...MCPOption) ([]Tool, error)
```

### 4.3 MCP Registry

```go
// tool/mcp_registry.go

type MCPRegistry interface {
    Search(ctx context.Context, query string) ([]MCPServerInfo, error)
    Discover(ctx context.Context) ([]MCPServerInfo, error)
}

type MCPServerInfo struct {
    Name         string
    URL          string
    Tools        []ToolDefinition
    Transport    TransportType  // stdio, http, sse
    AuthRequired bool
}
```

---

## 5. Memory System

```go
// memory/memory.go

type Memory interface {
    Save(ctx context.Context, input schema.Message, output schema.Message) error
    Load(ctx context.Context, query string) ([]schema.Message, error)
    Search(ctx context.Context, query string, k int) ([]schema.Document, error)
    Clear(ctx context.Context) error
}
```

### 5.1 Graph Memory

```go
// memory/graph.go

type GraphStore interface {
    AddEntity(ctx context.Context, entity Entity) error
    AddRelation(ctx context.Context, from, to string, relation string, props map[string]any) error
    Query(ctx context.Context, cypher string) ([]GraphResult, error)
    Neighbors(ctx context.Context, entityID string, depth int) ([]Entity, []Relation, error)
}
```

### 5.2 Composite Memory

```go
// memory/composite.go

mem := memory.NewComposite(
    memory.WithWorking(memory.NewWindow(20)),
    memory.WithEpisodic(memory.NewEntity(llm, store)),
    memory.WithSemantic(memory.NewSemantic(vectorstore, embedder)),
    memory.WithGraph(memory.NewGraph(neo4jDriver, llm)),
)
```

---

## 6. Agent Runtime

```go
// agent/agent.go

type Agent interface {
    core.Runnable
    ID() string
    Persona() Persona
    Tools() []tool.Tool
    Children() []Agent
    Card() a2a.AgentCard
}

// agent/base.go

type BaseAgent struct {
    id       string
    persona  Persona
    tools    []tool.Tool
    hooks    Hooks
    children []Agent
    metadata map[string]any
}

// agent/persona.go

type Persona struct {
    Role      string   // "Senior Go Engineer"
    Goal      string   // "Review PRs for security issues"
    Backstory string   // "You are paranoid about injection attacks..."
    Traits    []string // additional behavioural hints
}
```

### 6.1 Planner Interface

```go
// agent/planner.go

type Planner interface {
    Plan(ctx context.Context, state PlannerState) ([]Action, error)
    Replan(ctx context.Context, state PlannerState) ([]Action, error)
}

type PlannerState struct {
    Input        string
    Messages     []schema.Message
    Tools        []tool.Tool
    Observations []Observation
    Iteration    int
    Metadata     map[string]any
}

type Action struct {
    Type     ActionType
    ToolCall *schema.ToolCall
    Message  *schema.AIMessage
    Metadata map[string]any
}

type ActionType string
const (
    ActionTypeTool    ActionType = "tool"
    ActionTypeRespond ActionType = "respond"
    ActionTypeFinish  ActionType = "finish"
    ActionTypeHandoff ActionType = "handoff"
)

type Observation struct {
    Action  Action
    Result  *tool.ToolResult
    Error   error
    Latency time.Duration
}
```

### 6.2 Hooks

```go
// agent/hooks.go

type Hooks struct {
    // Agent lifecycle
    OnStart    func(ctx context.Context, input any) error
    OnEnd      func(ctx context.Context, result any, err error)
    OnError    func(ctx context.Context, err error) error

    // Reasoning loop
    BeforePlan   func(ctx context.Context, state PlannerState) (PlannerState, error)
    AfterPlan    func(ctx context.Context, actions []Action) ([]Action, error)
    BeforeReplan func(ctx context.Context, state PlannerState) (PlannerState, error)
    AfterReplan  func(ctx context.Context, actions []Action) ([]Action, error)
    OnIteration  func(ctx context.Context, iteration int, state PlannerState)

    // Action execution
    BeforeAct func(ctx context.Context, action Action) (Action, error)
    AfterAct  func(ctx context.Context, action Action, result *Observation) (*Observation, error)

    // Tool calls
    OnToolCall   func(ctx context.Context, name string, input map[string]any) (map[string]any, error)
    OnToolResult func(ctx context.Context, name string, result *tool.ToolResult) (*tool.ToolResult, error)

    // Agent delegation
    OnHandoff func(ctx context.Context, from, to string) error

    // LLM interaction
    BeforeGenerate func(ctx context.Context, msgs []schema.Message) ([]schema.Message, error)
    AfterGenerate  func(ctx context.Context, resp *schema.AIMessage) (*schema.AIMessage, error)
}

func ComposeHooks(hooks ...Hooks) Hooks { ... }
```

### 6.3 Event Bus

```go
// agent/bus.go

type EventBus interface {
    Publish(ctx context.Context, topic string, event AgentEvent) error
    Subscribe(ctx context.Context, topic string, handler EventHandler) (Subscription, error)
}

type AgentEvent struct {
    SourceAgent string
    TargetAgent string
    Type        string
    Payload     any
    Timestamp   time.Time
}
```

---

## 7. Durable Workflow System

```go
// workflow/executor.go

type DurableExecutor interface {
    Execute(ctx context.Context, opts WorkflowOptions, input any) (*WorkflowHandle, error)
    Signal(ctx context.Context, workflowID string, signal Signal) error
    Query(ctx context.Context, workflowID string, queryType string) (any, error)
    Cancel(ctx context.Context, workflowID string) error
    Terminate(ctx context.Context, workflowID string, reason string) error
}

type WorkflowOptions struct {
    ID               string
    TaskQueue        string
    RetryPolicy      RetryPolicy
    ExecutionTimeout time.Duration
    IdempotencyKey   string
    SearchAttributes map[string]any
}

type WorkflowHandle struct {
    ID     string
    RunID  string
    Status WorkflowStatus
    Result func() (any, error)
}
```

### 7.1 Activity Wrappers

```go
// workflow/activity.go

type LLMActivity struct {
    Model llm.ChatModel
}

func (a *LLMActivity) Generate(ctx context.Context, msgs []schema.Message, opts llm.GenerateOptions) (*schema.AIMessage, error)

type ToolActivity struct {
    Registry *tool.Registry
}

func (a *ToolActivity) Execute(ctx context.Context, toolName string, input map[string]any) (*tool.ToolResult, error)

type HumanActivity struct{}

func (a *HumanActivity) RequestApproval(ctx context.Context, req hitl.InteractionRequest) (*hitl.InteractionResponse, error)
```

### 7.2 Durable Agent Loop Pattern

```go
// workflow/patterns/agent_loop.go

func DurableAgentLoop(ctx workflow.Context, input AgentLoopInput) (*AgentLoopOutput, error) {
    state := agent.PlannerState{Input: input.Query, Messages: input.Messages, Tools: input.Tools}

    for iteration := 0; iteration < input.MaxIterations; iteration++ {
        // Durable activity: call LLM to plan
        var actions []agent.Action
        err := workflow.ExecuteActivity(ctx, activities.PlanActivity, state).Get(ctx, &actions)

        if hasFinishAction(actions) {
            return extractResult(actions), nil
        }

        // Durable activity: execute each action
        for _, action := range actions {
            switch action.Type {
            case agent.ActionTypeTool:
                var result tool.ToolResult
                workflow.ExecuteActivity(ctx, activities.ToolActivity, action.ToolCall).Get(ctx, &result)
                state.Observations = append(state.Observations, agent.Observation{Action: action, Result: &result})
            case agent.ActionTypeHandoff:
                child := workflow.ExecuteChildWorkflow(ctx, DurableAgentLoop, childInput)
                var childResult AgentLoopOutput
                child.Get(ctx, &childResult)
            }
        }
        state.Iteration = iteration + 1
    }
    return nil, fmt.Errorf("max iterations exceeded")
}
```

---

## 8. Evaluation Framework

```go
// eval/eval.go

type Metric interface {
    Name() string
    Score(ctx context.Context, sample EvalSample) (float64, error)
}

type EvalSample struct {
    Input          string
    Output         string
    ExpectedOutput string
    RetrievedDocs  []schema.Document
    Metadata       map[string]any
}

// eval/runner.go

type EvalRunner struct {
    metrics  []Metric
    dataset  []EvalSample
    parallel int
}

func (r *EvalRunner) Run(ctx context.Context) (*EvalReport, error)
```

---

## 9. Resilience Package

```go
// resilience/circuitbreaker.go

type CircuitBreaker struct {
    failureThreshold int
    resetTimeout     time.Duration
    state            State  // closed, open, half-open
}

// resilience/hedge.go

func Hedge(primary, secondary ChatModel, delay time.Duration) ChatModel

// resilience/retry.go

type RetryPolicy struct {
    MaxAttempts     int
    InitialBackoff  time.Duration
    MaxBackoff      time.Duration
    BackoffFactor   float64
    Jitter          bool
    RetryableErrors []ErrorCode
}

// resilience/ratelimit.go

type ProviderLimits struct {
    RPM             int
    TPM             int
    MaxConcurrent   int
    CooldownOnRetry time.Duration
}

func WithProviderLimits(limits ProviderLimits) llm.Middleware
```

---

## 10. Cache Package

```go
// cache/cache.go

type Cache interface {
    Get(ctx context.Context, key string) (any, bool, error)
    Set(ctx context.Context, key string, value any, ttl time.Duration) error
    GetSemantic(ctx context.Context, embedding []float32, threshold float64) (any, bool, error)
}
```

---

## 11. Auth Package

```go
// auth/auth.go

type Permission string
const (
    PermToolExec      Permission = "tool:execute"
    PermMemoryRead    Permission = "memory:read"
    PermMemoryWrite   Permission = "memory:write"
    PermAgentDelegate Permission = "agent:delegate"
    PermExternalAPI   Permission = "external:api"
)

type Policy interface {
    Authorize(ctx context.Context, agent string, permission Permission, resource string) (bool, error)
}
```

---

## 12. State Package

```go
// state/state.go

type Store interface {
    Get(ctx context.Context, key string) (any, error)
    Set(ctx context.Context, key string, value any) error
    Delete(ctx context.Context, key string) error
    Watch(ctx context.Context, key string) (<-chan StateChange, error)
}

// Scopes: AgentScope, SessionScope, GlobalScope
```

---

## 13. HITL Package

```go
// hitl/hitl.go

type InteractionType string
const (
    InteractionApproval   InteractionType = "approval"
    InteractionFeedback   InteractionType = "feedback"
    InteractionInput      InteractionType = "input"
    InteractionAnnotation InteractionType = "annotation"
)

type InteractionRequest struct {
    ID       string
    Type     InteractionType
    AgentID  string
    Context  any
    Options  []string
    Timeout  time.Duration
    Callback InteractionCallback
}

type InteractionResponse struct {
    Decision string
    Reviewer string
    Reason   string
    Data     any
}
```

---

## 14. Prompt Management

```go
// prompt/template.go

type Template struct {
    Name      string
    Version   string
    Content   string            // Go template syntax
    Variables map[string]string // defaults
    Metadata  map[string]any
}

// prompt/manager.go

type PromptManager interface {
    Get(name string, version string) (*Template, error)
    Render(name string, vars map[string]any) ([]schema.Message, error)
    List() []TemplateInfo
}
```

---

## 15. Extension Examples (Application Code)

### 15.1 Custom LLM Provider

```go
package fireworks

import "github.com/lookatitude/beluga-ai/llm"

type FireworksModel struct { /* ... */ }

func (f *FireworksModel) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) { /* ... */ }
func (f *FireworksModel) Stream(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (core.Stream[schema.StreamChunk], error) { /* ... */ }
func (f *FireworksModel) BindTools(tools []tool.Tool) llm.ChatModel { /* ... */ }
func (f *FireworksModel) ModelID() string { return "fireworks/" + f.model }

func init() {
    llm.Register("fireworks", func(cfg llm.ProviderConfig) (llm.ChatModel, error) {
        return &FireworksModel{apiKey: cfg.APIKey, model: cfg.Model}, nil
    })
}
```

### 15.2 Custom Planner (Tree-of-Thought)

```go
package planners

import "github.com/lookatitude/beluga-ai/agent"

type TreeOfThoughtPlanner struct {
    llm       llm.ChatModel
    branches  int
    evaluator llm.ChatModel
    maxDepth  int
}

func (p *TreeOfThoughtPlanner) Plan(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) {
    branches := p.generateBranches(ctx, state, p.branches)
    scored := p.evaluateBranches(ctx, branches)
    best := selectBest(scored)
    state.Metadata["branches"] = scored
    state.Metadata["active_branch"] = best.ID
    return best.Actions, nil
}

func (p *TreeOfThoughtPlanner) Replan(ctx context.Context, state agent.PlannerState) ([]agent.Action, error) { /* ... */ }

func init() {
    agent.RegisterPlanner("tree-of-thought", func(cfg agent.PlannerConfig) (agent.Planner, error) {
        return &TreeOfThoughtPlanner{
            llm:       cfg.LLM,
            branches:  cfg.GetIntOr("branches", 3),
            evaluator: cfg.GetLLMOr("evaluator", cfg.LLM),
            maxDepth:  cfg.GetIntOr("max_depth", 5),
        }, nil
    })
}
```

### 15.3 Custom Agent (Research Pipeline)

```go
package agents

import "github.com/lookatitude/beluga-ai/agent"

type ResearchPipelineAgent struct {
    agent.BaseAgent
    searchers   []Agent
    extractor   Agent
    verifier    Agent
    synthesizer Agent
    llm         llm.ChatModel
    minCredibility float64
}

func (r *ResearchPipelineAgent) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
    query := input.(string)

    // Phase 1: Parallel source gathering
    var sources []schema.Document
    results := make(chan []schema.Document, len(r.searchers))
    for _, s := range r.searchers {
        go func(s Agent) {
            res, _ := s.Invoke(ctx, query)
            results <- res.([]schema.Document)
        }(s)
    }
    for range r.searchers { sources = append(sources, <-results...) }

    // Phase 2: Extract and verify
    claims, _ := r.extractor.Invoke(ctx, sources)
    verified, _ := r.verifier.Invoke(ctx, claims)

    // Phase 3: Conditional retry
    if verified.(VerifiedClaims).Score < r.minCredibility {
        expanded, _ := r.expandSearch(ctx, query, sources)
        sources = append(sources, expanded...)
        claims, _ = r.extractor.Invoke(ctx, sources)
        verified, _ = r.verifier.Invoke(ctx, claims)
    }

    // Phase 4: Synthesize
    return r.synthesizer.Invoke(ctx, verified)
}
```

### 15.4 Custom Middleware (Token Budget)

```go
package middleware

import "github.com/lookatitude/beluga-ai/llm"

func WithTokenBudget(maxTokens int) llm.Middleware {
    return func(next llm.ChatModel) llm.ChatModel {
        return &tokenBudgetModel{inner: next, remaining: maxTokens}
    }
}

type tokenBudgetModel struct {
    inner     llm.ChatModel
    remaining int
    mu        sync.Mutex
}

func (m *tokenBudgetModel) Generate(ctx context.Context, msgs []schema.Message, opts ...llm.GenerateOption) (*schema.AIMessage, error) {
    m.mu.Lock()
    if m.remaining <= 0 { m.mu.Unlock(); return nil, fmt.Errorf("token budget exhausted") }
    m.mu.Unlock()

    resp, err := m.inner.Generate(ctx, msgs, opts...)
    if err == nil {
        m.mu.Lock(); m.remaining -= resp.Usage.TotalTokens; m.mu.Unlock()
    }
    return resp, err
}
```

### 15.5 Custom Vector Store

```go
package milvus

import "github.com/lookatitude/beluga-ai/rag/vectorstore"

type MilvusStore struct { /* ... */ }

func (m *MilvusStore) Add(ctx context.Context, docs []schema.Document, embeddings [][]float32) error { /* ... */ }
func (m *MilvusStore) Search(ctx context.Context, query []float32, k int, opts ...vectorstore.SearchOption) ([]schema.Document, error) { /* ... */ }
func (m *MilvusStore) Delete(ctx context.Context, ids []string) error { /* ... */ }

func init() {
    vectorstore.Register("milvus", func(cfg vectorstore.ProviderConfig) (vectorstore.VectorStore, error) {
        return &MilvusStore{/* ... */}, nil
    })
}
```

### 15.6 Custom Retriever (HyDE)

```go
package retrievers

import "github.com/lookatitude/beluga-ai/rag/retriever"

type HyDERetriever struct {
    llm      llm.ChatModel
    embedder embedding.Embedder
    store    vectorstore.VectorStore
}

func (h *HyDERetriever) Retrieve(ctx context.Context, query string, opts ...retriever.Option) ([]schema.Document, error) {
    hypoDoc, _ := h.llm.Generate(ctx, []schema.Message{
        schema.SystemMessage("Generate a hypothetical document that answers this query"),
        schema.HumanMessage(query),
    })
    vec, _ := h.embedder.Embed(ctx, hypoDoc.Text())
    return h.store.Search(ctx, vec, 10)
}

func init() {
    retriever.Register("hyde", func(cfg retriever.Config) (retriever.Retriever, error) {
        return &HyDERetriever{/* ... */}, nil
    })
}
```

### 15.7 Custom Memory (Graph Memory)

```go
package memory

type GraphMemory struct {
    graph neo4j.Driver
    llm   llm.ChatModel
}

func (g *GraphMemory) Save(ctx context.Context, input, output schema.Message) error { /* extract entities + relationships via LLM, persist to Neo4j */ }
func (g *GraphMemory) Load(ctx context.Context, query string) ([]schema.Message, error) { /* query graph for relevant context */ }
func (g *GraphMemory) Search(ctx context.Context, query string, k int) ([]schema.Document, error) { /* graph traversal + vector hybrid search */ }
func (g *GraphMemory) Clear(ctx context.Context) error { /* ... */ }

func init() {
    memory.Register("graph", func(cfg memory.Config) (memory.Memory, error) {
        return &GraphMemory{/* ... */}, nil
    })
}
```

### 15.8 Custom Orchestration Node (Approval Gate)

```go
package nodes

import "github.com/lookatitude/beluga-ai/orchestration"

type ApprovalGate struct {
    notifier NotificationService
    timeout  time.Duration
}

func (a *ApprovalGate) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
    approval := a.notifier.RequestApproval(ctx, input, a.timeout)
    if !approval.Approved {
        return nil, fmt.Errorf("rejected by %s: %s", approval.Reviewer, approval.Reason)
    }
    return input, nil
}
```
