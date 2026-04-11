# Reference: Providers

Every extension point in Beluga is backed by a registry of providers. This catalog lists the built-in providers per category with their config keys. For the registration mechanism, see [Registry + Factory pattern](../patterns/registry-factory.md); for the universal template, see [Provider Template pattern](../patterns/provider-template.md).

## LLM providers — `llm`

Import the provider for its `init()` side-effect, then construct via `llm.New("name", cfg)`.

| Provider | Import path | Config keys |
|---|---|---|
| OpenAI | `github.com/lookatitude/beluga-ai/llm/providers/openai` | `model`, `api_key`, `base_url`, `organization`, `temperature`, `max_tokens` |
| Anthropic | `github.com/lookatitude/beluga-ai/llm/providers/anthropic` | `model`, `api_key`, `max_tokens`, `thinking` |
| Google Gemini | `github.com/lookatitude/beluga-ai/llm/providers/gemini` | `model`, `api_key`, `project`, `region` |
| AWS Bedrock | `github.com/lookatitude/beluga-ai/llm/providers/bedrock` | `model`, `region`, AWS standard auth |
| Azure OpenAI | `github.com/lookatitude/beluga-ai/llm/providers/azure` | `deployment`, `api_key`, `endpoint`, `api_version` |
| Ollama | `github.com/lookatitude/beluga-ai/llm/providers/ollama` | `model`, `base_url`, `timeout` |
| OpenAI-compatible | `github.com/lookatitude/beluga-ai/llm/providers/openaicompat` | `base_url`, `api_key`, `model` — for vLLM, LM Studio, LocalAI, etc. |

**Status note:** specific provider availability varies — run `llm.List()` to see what's registered in your build. To add one, see [Custom Provider guide](../guides/custom-provider.md).

## Embedding providers — `rag/embedding`

| Provider | Config keys |
|---|---|
| OpenAI embeddings | `model` (e.g. `text-embedding-3-large`), `api_key`, `dimensions` |
| Voyage | `model`, `api_key` |
| Cohere | `model`, `api_key`, `input_type` |
| BGE (self-hosted) | `base_url`, `model` |
| Ollama embeddings | `model`, `base_url` |

## Vector stores — `rag/vectorstore`

| Provider | Config keys |
|---|---|
| pgvector (Postgres) | `dsn`, `table`, `dimensions` |
| Pinecone | `api_key`, `environment`, `index` |
| Weaviate | `url`, `api_key`, `class` |
| Qdrant | `url`, `api_key`, `collection` |
| Milvus | `host`, `port`, `collection` |
| Chroma | `url`, `collection` |
| In-memory | (none — for tests) |

## Memory stores — `memory/stores`

| Provider | Stores | Config keys |
|---|---|---|
| In-memory | all tiers | (none) |
| Redis | working + recall | `addr`, `password`, `db`, `prefix` |
| Postgres | recall + archival | `dsn`, `table` |
| SQLite | single-file dev | `path` |
| Neo4j | graph tier | `uri`, `username`, `password` |

## Retriever strategies — `rag/retriever`

| Strategy | What it does |
|---|---|
| `hybrid` (default) | BM25 + vector + RRF fusion + reranker |
| `dense` | Vector search only |
| `bm25` | BM25 only |
| `crag` | Corrective RAG with query rewrite fallback |
| `hyde` | Hypothetical document embedding |
| `adaptive` | Chooses retrieval mode per query difficulty |
| `parent` | Retrieves small chunks, returns parent docs |

## Voice providers

### STT — `voice/stt`

| Provider | Config |
|---|---|
| Whisper API | `api_key`, `model`, `language` |
| Whisper.cpp (self-hosted) | `model_path` |
| Deepgram | `api_key`, `model` |
| Google STT | `api_key`, `language_code` |

### TTS — `voice/tts`

| Provider | Config |
|---|---|
| OpenAI TTS | `api_key`, `voice`, `model` |
| ElevenLabs | `api_key`, `voice_id`, `model_id` |
| Azure Speech | `api_key`, `voice`, `region` |
| Cartesia | `api_key`, `voice_id` |

### S2S — `voice/s2s`

| Provider | Config |
|---|---|
| OpenAI Realtime | `api_key`, `model`, `voice` |
| Gemini Live | `api_key`, `model` |

### Transport — `voice/transport`

| Provider | Config |
|---|---|
| LiveKit | `url`, `api_key`, `api_secret`, `room` |
| WebRTC | (peer config) |
| Local (microphone) | `device` |
| WebSocket audio | `url`, `token` |

## Tool providers — `tool/builtin`

Built-in tools ship under `tool/builtin/<name>`:

| Tool | Capability | Description |
|---|---|---|
| `web_search` | `tool.web.search` | Search via Tavily/Serper/etc. |
| `http_fetch` | `tool.http.fetch` | Fetch a URL (with allowlist) |
| `file_read` | `tool.filesystem.read` | Read file (with path allowlist) |
| `file_write` | `tool.filesystem.write` | Write file (with path allowlist) |
| `shell_exec` | `tool.shell.exec` | Execute command (with allowlist) |
| `sql_query` | `tool.sql.query` | Read-only SQL (parameterised) |
| `arxiv_search` | `tool.arxiv.search` | Query arXiv |
| `github` | `tool.github.*` | GitHub API operations |

## Guard providers — `guard/providers`

| Provider | What it checks |
|---|---|
| `promptInjection` | Known jailbreak templates |
| `piiDetect` | PII patterns in input |
| `piiRedact` | PII in output |
| `contentModeration` | OpenAI moderation API or equivalent |
| `capabilityCheck` | Tool capability requirements vs tenant grants |
| `schemaValidator` | JSON schema enforcement |
| `spotlighting` | Delimit untrusted content |

## Session service — `runtime`

| Provider | Config |
|---|---|
| `inmemory` | (none) |
| `redis` | `addr`, `ttl` |
| `postgres` | `dsn`, `table` |
| `sqlite` | `path` |

## Workflow engines — `workflow`

| Engine | Config |
|---|---|
| built-in | `backend` (`sqlite`/`postgres`), `dsn` |
| Temporal | `host_port`, `namespace`, `task_queue` |

## Authentication providers — `auth`

| Provider | Config |
|---|---|
| JWT | `public_key`, `issuer`, `audience` |
| OAuth2 | `issuer_url`, `client_id`, `client_secret` |
| API key | `header`, `source` |

## Cache backends — `cache`

| Backend | Config |
|---|---|
| `inmemory` | `max_entries` |
| `redis` | `addr`, `ttl` |
| `semantic` | `embedder`, `threshold`, `store` |
| `prompt` | (provider-native, opts out of local cache) |

## Adding a new provider

See [Custom Provider guide](../guides/custom-provider.md) and [Provider Template pattern](../patterns/provider-template.md) for a step-by-step walk-through. The five-part template applies to any category listed above:

1. Implement the interface.
2. Write a factory function.
3. Register in `init()`.
4. Write table-driven tests including registration.
5. Document in godoc.

## Related

- [Registry + Factory pattern](../patterns/registry-factory.md)
- [Provider Template pattern](../patterns/provider-template.md)
- [Custom Provider guide](../guides/custom-provider.md)
- [03 — Extensibility Patterns](../architecture/03-extensibility-patterns.md)
