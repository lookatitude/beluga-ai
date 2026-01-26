# Provider Subpackage Layout

Each provider lives in `providers/<name>/` with:

- **init.go** — `func init()` that registers with the package registry. Enables discoverability (auto-register on import) and keeps wiring in one place.

```go
func init() {
    llms.GetRegistry().Register("openai", NewOpenAIProviderFactory())
}
```

- **provider.go** — Common functionality and wiring: factory, shared setup, config handling. Keeps provider implementations consistent.

- **{name}.go** — Provider-specific implementation (e.g. `openai.go`, `anthropic.go`).

- **{name}_mock.go** — Test mocks for this provider.

**Rule:** The string passed to `Register` matches the directory name (e.g. `"openai"` for `providers/openai/`).
