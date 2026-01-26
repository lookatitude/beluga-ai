# Product Roadmap

## Phase 1: MVP (Complete)

Core framework is production-ready:

- **LLM Integration** — OpenAI, Anthropic, Bedrock, Ollama, Gemini, Grok, Groq providers
- **Agent Framework** — ReAct agents, tools, executors, planners
- **RAG Pipeline** — Document loaders, text splitters, embeddings, vector stores, retrievers
- **Memory Systems** — Buffer, summary, and vector-based conversation memory
- **Voice Processing** — STT, TTS, VAD, transport, S2S pipelines
- **Orchestration** — Chains, graphs, workflows with Temporal integration
- **Observability** — OpenTelemetry tracing and metrics throughout

## Phase 2: Provider Expansion

- Additional LLM providers (Cohere, Mistral, local models)
- More embedding providers (Cohere, Voyage, local)
- Vector store expansion (Milvus, Chroma, Elasticsearch)
- Messaging providers beyond Twilio

## Phase 3: Advanced Voice

- Enhanced speech-to-speech pipelines
- Multi-modal voice processing (voice + vision)
- Real-time voice activity detection improvements
- Voice agent specializations

## Phase 4: Agent Orchestration

- Multi-agent workflows and collaboration
- Agent-to-agent communication patterns
- Hierarchical agent structures
- Agent memory sharing and coordination

## Phase 5: Evaluation Framework

- Built-in evaluation metrics for LLM outputs
- RAG quality benchmarking
- Agent performance scoring
- Automated regression testing for AI behavior

## Phase 6: Developer Tooling

- CLI tools for scaffolding and code generation
- Project templates and starter kits
- Development server with hot reload
- Visual debugging for agent traces
