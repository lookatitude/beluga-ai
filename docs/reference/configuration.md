# Reference: Configuration

Beluga packages take configuration in two forms: functional options for constructors (`agent.New(id, agent.WithLLM(m))`), and a `config.ProviderConfig` struct for registry lookups (`llm.New("openai", cfg)`). The library itself reads very few environment variables; this page lists every one it reads and where. Provider-specific options live in [`providers.md`](./providers.md).

## Convention

- Constructors take variadic options: `NewX(required..., opts ...Option)`. See [Functional Options pattern](../patterns/functional-options.md).
- Registry constructors take `config.ProviderConfig` (`Provider`, `APIKey`, `Model`, `BaseURL`, `Timeout`, `Options map[string]any`). Read typed extras with `config.GetOption[T](cfg, "key")`.
- Your application decides where values come from (env vars, a JSON file, flags). Use the `config` package helpers below to load them.
- Secrets never appear in Go code. Pass them from the environment or a secret mount.

## Environment variables

These are the only environment variables read by code in this repository.

### Read by the library

| Env var | Read by | Effect |
|---|---|---|
| `OTEL_SDK_DISABLED` | `o11y.BootstrapFromEnv` | Truthy (`1`, `true`, `yes`): no exporter, spans are no-ops. Checked first. |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `o11y.BootstrapFromEnv` | When set, installs the OTLP/HTTP trace exporter. The OTel SDK then reads its own `OTEL_EXPORTER_OTLP_*` variables (headers, timeout, ...). |
| `BELUGA_OTEL_STDOUT` | `o11y.BootstrapFromEnv` | Truthy: pretty-printed stdout trace exporter, when no OTLP endpoint is set. For local debugging only. |
| `BELUGA_MOCK_FIXTURES` | `llm/providers/mock` | Path to a fixtures file, used when no fixtures are passed in code or in `ProviderConfig.Options`. |

`o11y.BootstrapFromEnv` only runs if your program calls it. With none of the variables set, it installs no exporter.

### Read by the `beluga` CLI and scaffolded projects

| Env var | Read by | Effect |
|---|---|---|
| `BELUGA_ENV` | scaffolded `main.go` | `eval` switches the binary to the `beluga eval` row protocol. `beluga test` sets it to `test`, which the scaffold does not currently branch on. |
| `BELUGA_LLM_PROVIDER` | scaffolded `main.go` | `mock` routes the scaffold's `llm.New` call to the mock provider. Set by `beluga test` and `make test-ci`/`eval-ci`. |
| `BELUGA_EVAL_SAMPLE_JSON` | scaffolded `main.go` | The dataset row, set per child process by `beluga eval`. |
| `BELUGA_EVAL_RUN_ID`, `BELUGA_EVAL_ROW_ID` | set by `beluga eval` | Run and row IDs for the child; see the [CLI reference](./cli.md#beluga-eval). |
| `BELUGA_PLAYGROUND_URL` | set by `beluga dev` | URL of the dev playground, exported to the child. |
| `OPENAI_API_KEY` | scaffolded `main.go` | API key for the default `openai` provider. |

The scaffolded `Makefile` also exports `BELUGA_DETERMINISTIC=1` and `BELUGA_SEED=42`. Nothing reads them yet.

### Set by the Kubernetes reconciler

`k8s/operator`'s reconciler puts `BELUGA_AGENT_NAME`, `BELUGA_PLANNER`, `BELUGA_MODEL_REF` and `BELUGA_API_KEY` into the Deployment it computes. The library does not read them; your agent binary can.

## Common options and defaults

### Agent

```go
a := agent.New("research-assistant",
	agent.WithPersona(agent.Persona{Role: "research assistant"}),
	agent.WithLLM(model),
	agent.WithTools(tools),
	agent.WithPlannerName("react"), // default "react"
	agent.WithMaxIterations(10),    // default 10
	agent.WithTimeout(5*time.Minute), // default 5m
	agent.WithHooks(hooks),
	agent.WithHandoffs(handoffs),
)
```

Other options: `WithPlanner` (a planner instance instead of a name), `WithMemory`, `WithChildren`, `WithMetadata`, `WithContract`. Wrap the agent with `agent.ApplyMiddleware(a, agent.WithTracing())` for spans.

### Runner

```go
r := runtime.NewRunner(a,
	runtime.WithSessionService(runtime.NewInMemorySessionService()), // default: in-memory
	runtime.WithPlugins(plugins.NewAuditPlugin(audit.NewInMemoryStore())),
	runtime.WithRunnerConfig(runtime.RunnerConfig{
		SessionTTL:              time.Hour,        // default 0: sessions never expire
		WorkerPoolSize:          10,               // default 10
		GracefulShutdownTimeout: 30 * time.Second, // default 30s
	}),
)
```

The Runner does not serve HTTP. Expose an agent with `protocol/rest`, `protocol/a2a`, `server.NewAgentHandler`, or your own handler around `Runner.Run`.

### LLM

```go
model, err := llm.New("openai", config.ProviderConfig{
	Provider: "openai",
	APIKey:   os.Getenv("OPENAI_API_KEY"),
	Model:    "gpt-4o",
	Timeout:  30 * time.Second, // ProviderConfig default tag: 30s
})
if err != nil {
	return err
}

model = llm.ApplyMiddleware(model,
	llm.WithTracing(),
	llm.WithLogging(slog.Default()),
	llm.WithProviderLimits(llm.ProviderLimits{RPM: 600, MaxConcurrent: 20}),
	llm.WithFallback(backupModel),
)
```

`llm.ProviderLimits` also has a `TPM` field, but the `llm` middleware does not enforce it yet. Per-call settings (temperature, max tokens, tool choice) are `llm.GenerateOption`s passed to `Generate`/`Stream`, for example `llm.WithTemperature(0.7)`.

### Resilience

Retry, circuit breaking, hedging and token-aware rate limiting are generic helpers in `resilience/`:

```go
policy := resilience.RetryPolicy{
	MaxAttempts:    3,                      // default 3
	InitialBackoff: 500 * time.Millisecond, // default 500ms
	MaxBackoff:     30 * time.Second,       // default 30s
	BackoffFactor:  2.0,                    // default 2.0
	Jitter:         true,                   // default true
}
breaker := resilience.NewCircuitBreaker(5, 30*time.Second) // failure threshold, reset timeout
limiter := resilience.NewRateLimiter(resilience.ProviderLimits{RPM: 600, TPM: 150_000, MaxConcurrent: 20})

resp, err := resilience.Retry(ctx, policy, func(ctx context.Context) (*schema.AIMessage, error) {
	if err := limiter.Allow(ctx); err != nil {
		return nil, err
	}
	defer limiter.Release()
	out, err := breaker.Execute(ctx, func(ctx context.Context) (any, error) {
		return model.Generate(ctx, msgs)
	})
	if err != nil {
		return nil, err
	}
	return out.(*schema.AIMessage), nil
})
```

`resilience.Hedge(ctx, primary, secondary, delay)` starts `secondary` if `primary` has not returned after `delay`.

### Memory

```go
mem := memory.NewComposite(
	memory.WithCore(memory.NewCore(memory.CoreConfig{})), // persona/human blocks, 2000 chars each by default
	memory.WithRecall(memory.NewRecall(messageStore)),
	memory.WithArchival(archival),
	memory.WithGraph(graphStore),
)
```

Stores come from `memory/stores/*`; for a registered provider use `memory.New(name, config.ProviderConfig{...})`.

### RAG

```go
r := retriever.NewHybridRetriever(vectorStore, embedder, bm25,
	retriever.WithHybridRRFK(60),
)
docs, err := r.Retrieve(ctx, "how do I rotate keys?", retriever.WithTopK(10))
```

Registered strategies are also available through `retriever.New(name, config.ProviderConfig{...})`.

### Guard pipeline

```go
guards := guard.NewPipeline(
	guard.Input(guard.NewPromptInjectionDetector()),
	guard.Output(guard.NewPIIRedactor(guard.DefaultPIIPatterns...)),
	guard.Tool(guard.NewContentFilter(guard.WithKeywords("rm -rf"))),
)
res, err := guards.ValidateInput(ctx, userText)
```

## Config loading

The `config` package loads typed configuration from JSON files and environment variables:

```go
type AppConfig struct {
	LLMModel    string `json:"llm_model" default:"gpt-4o"`
	BudgetCents int    `json:"budget_cents" default:"1000" min:"0"`
	APIKey      string `json:"api_key" required:"true"`
}

// From a JSON file. Only .json is supported. Defaults fill fields that are
// absent from the file, then required/min/max tags are validated.
cfg, err := config.Load[AppConfig]("/etc/app/config.json")
if err != nil {
	return err
}

// Override fields from APP_LLM_MODEL, APP_BUDGET_CENTS, APP_API_KEY when set.
if err := config.MergeEnv(&cfg, "APP"); err != nil {
	return err
}
```

`config.LoadFromEnv[T](prefix)` builds the struct from environment variables alone (`PREFIX_FIELD_NAME`, nested structs extend the prefix), applying `default` tags and validating. `config.Validate(&cfg)` runs the `required`, `min` and `max` checks on any struct.

Precedence with `Load` followed by `MergeEnv`: env var, then file, then `default` tag.

## Hot reload

`config.NewFileWatcher(path, interval)` polls a file (minimum interval 100ms) and calls your callback with the new raw bytes when the content hash changes:

```go
w := config.NewFileWatcher("/etc/app/config.json", 5*time.Second)
defer w.Close()
go func() {
	_ = w.Watch(ctx, func(raw any) {
		var next AppConfig
		if err := json.Unmarshal(raw.([]byte), &next); err != nil {
			return // keep the old config
		}
		apply(next)
	})
}()
```

`Watch` blocks until `ctx` is cancelled or `Close` is called. There is no Consul, etcd or Kubernetes watcher. Use reload sparingly; most production config is immutable per deployment.

## Related

- [Functional Options pattern](../patterns/functional-options.md)
- [`providers.md`](./providers.md) — provider catalog.
- [CLI reference](./cli.md) — the environment the CLI passes to scaffolded projects.
- [17 — Deployment Modes](../architecture/17-deployment-modes.md)
