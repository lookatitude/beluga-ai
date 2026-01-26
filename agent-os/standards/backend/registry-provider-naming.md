# Registry Provider Naming

**Case:** The `name` in `Register(name, factory)` and in `Get`/`Create(name, config)` MUST be lowercase (e.g. `"openai"`, `"anthropic"`).

**Directory vs key:** The register key can differ from the provider directory. Prefer matching `providers/<name>/` when there is a single provider per dir (e.g. `"openai"` for `providers/openai/`). A different key (e.g. `"openai-gpt4"`) is allowed when one implementation exposes multiple logical providers.
