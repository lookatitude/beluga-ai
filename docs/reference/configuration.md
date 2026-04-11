# Reference: Configuration

Every Beluga package accepts configuration via functional options or typed `Config` values. This document catalogues the common options and environment variables. Provider-specific options live in [`providers.md`](./providers.md).

## Convention

- Constructors take variadic `Option` values: `NewX(opts ...Option)`. See [Functional Options pattern](../patterns/functional-options.md).
- Config values come from **env vars**, **config files**, or **in-code options** — never hardcoded.
- Every option has a sensible default. `NewX()` with zero arguments always works (when the package permits zero config).
- Secrets **never** appear in Go code. Always via env or secret mount.

## Environment variables

### Observability

| Env var | Default | Description |
|---|---|---|
| `OTEL_EXPORTER_OTLP_ENDPOINT` | _unset_ | OTLP collector endpoint. Enables OTel export when set. |
| `OTEL_SERVICE_NAME` | _binary name_ | Service name in traces. |
| `OTEL_TRACES_SAMPLER` | `parentbased_always_on` | Sampling strategy. |
| `BELUGA_LOG_LEVEL` | `info` | `debug`, `info`, `warn`, `error`. |
| `BELUGA_LOG_FORMAT` | `json` | `json` or `text`. |

### Multi-tenancy

| Env var | Default | Description |
|---|---|---|
| `BELUGA_DEFAULT_TENANT` | `default` | Fallback tenant when none in context. |
| `BELUGA_TENANT_HEADER` | `X-Tenant-ID` | HTTP header protocol servers read to populate context. |

### Session store

| Env var | Default | Description |
|---|---|---|
| `BELUGA_SESSION_BACKEND` | `inmemory` | `inmemory`, `redis`, `postgres`, `sqlite`. |
| `BELUGA_SESSION_TTL` | `24h` | Session expiry. |
| `REDIS_ADDR` | _unset_ | Redis address when backend is `redis`. |
| `POSTGRES_DSN` | _unset_ | Postgres DSN when backend is `postgres`. |

### Memory store

| Env var | Default | Description |
|---|---|---|
| `BELUGA_MEMORY_BACKEND` | `inmemory` | `inmemory`, `redis`, `postgres`, `sqlite`, `neo4j`. |
| `BELUGA_MEMORY_MAX_WORKING_MESSAGES` | `50` | Cap on working memory. |

### Cost / budget

| Env var | Default | Description |
|---|---|---|
| `BELUGA_COST_BACKEND` | `inmemory` | Where cost counters are stored. |
| `BELUGA_COST_BUDGET_DEFAULT` | _unset_ | Default per-tenant budget in cents. |

### Security / guards

| Env var | Default | Description |
|---|---|---|
| `BELUGA_GUARDS_ENABLED` | `true` | Master switch for the guard pipeline. Disable only in dev. |
| `BELUGA_HITL_ENABLED` | `false` | Enable human-in-the-loop for high-risk tools. |

## Common functional options

### Agent

```go
agent.NewLLMAgent(
    agent.WithID("research-assistant"),
    agent.WithPersona(agent.Persona{...}),
    agent.WithLLM(model),
    agent.WithMemory(mem),
    agent.WithPlanner(planner),
    agent.WithTools(tools...),
    agent.WithHooks(hooks),
    agent.WithMiddleware(...),
    agent.WithHandoffs(otherAgents...),
    agent.WithMaxIterations(10), // default 20
)
```

### Runner

```go
runtime.NewRunner(agent,
    runtime.WithSessionService(sessions),
    runtime.WithArtifactService(artifacts),
    runtime.WithPlugin(plugin),
    runtime.WithGuards(guards),
    runtime.WithRESTEndpoint("/api/chat"),
    runtime.WithA2A("/.well-known/agent.json"),
    runtime.WithMCPServer("/mcp"),
    runtime.WithWebSocketEndpoint("/ws"),
    runtime.WithGRPCEndpoint(":9090"),
    runtime.WithDrainTimeout(30 * time.Second),
    runtime.WithDurableExecution(engine),
    runtime.WithDurableWorkflowTimeout(24 * time.Hour),
    runtime.WithWorkerPoolSize(50),
)
```

### LLM

```go
llm.New("openai", llm.Config{
    "model":   "gpt-4o",
    "api_key": os.Getenv("OPENAI_API_KEY"),
    "temperature": 0.7,
    "max_tokens":  4096,
})

// middleware
llm.ApplyMiddleware(model,
    llm.WithGuardrails(guards),
    llm.WithLogging(logger),
    llm.WithMetrics(),
    llm.WithCircuitBreaker(llm.CircuitBreakerConfig{
        FailureThreshold: 5,
        CooldownPeriod:   30 * time.Second,
    }),
    llm.WithRateLimit(llm.RateLimitConfig{
        RPM:           600,
        TPM:           150_000,
        MaxConcurrent: 20,
    }),
    llm.WithRetry(llm.RetryConfig{
        MaxAttempts: 3,
        BaseDelay:   100 * time.Millisecond,
        MaxDelay:    5 * time.Second,
        Jitter:      0.2,
    }),
    llm.WithHedge(llm.HedgeConfig{
        HedgeDelay: 500 * time.Millisecond,
    }),
)
```

### Memory

```go
memory.NewComposite(
    memory.WithWorking(memory.Window(50)),
    memory.WithRecall(recallStore),
    memory.WithArchival(archivalStore),
    memory.WithGraph(graphStore),
)
```

### RAG

```go
retriever, _ := rag.NewRetriever("hybrid", rag.Config{
    "bm25_index":     bm25Index,
    "vector_store":   vectorStore,
    "embedder":       embedder,
    "reranker":       reranker,
    "rrf_k":          60,
    "top_k":          10,
    "bm25_candidates": 200,
    "vector_candidates": 100,
})
```

### Guard pipeline

```go
guards := guard.NewPipeline(
    guard.WithInputGuards(
        guard.PromptInjectionDetector(),
        guard.PIIDetector(),
    ),
    guard.WithOutputGuards(
        guard.PIIRedactor(),
        guard.ContentModeration(),
    ),
    guard.WithToolGuards(
        guard.CapabilityChecker(capMatrix),
        guard.SchemaValidator(),
    ),
)
```

## Config loading

The `config` package provides generic loading:

```go
type AppConfig struct {
    LLMModel string `yaml:"llm_model" env:"LLM_MODEL" default:"gpt-4o"`
    Budget   int    `yaml:"budget_cents" env:"BUDGET_CENTS" default:"1000"`
}

cfg, err := config.Load[AppConfig](ctx,
    config.FromFile("/etc/app/config.yaml"),
    config.FromEnv(),
    config.WithValidator(validateAppConfig),
)
```

Precedence (highest to lowest): env var → file → default struct tag.

## Hot reload

`config.Watch` re-parses the config file on change and emits the new value through a channel:

```go
cfgChan, stop := config.Watch[AppConfig](ctx, "/etc/app/config.yaml")
defer stop()
for cfg := range cfgChan {
    // apply new config
}
```

Used sparingly — most production config is immutable per deployment.

## Related

- [Functional Options pattern](../patterns/functional-options.md)
- [`providers.md`](./providers.md) — provider-specific config keys.
- [17 — Deployment Modes](../architecture/17-deployment-modes.md) — which env vars apply per mode.
