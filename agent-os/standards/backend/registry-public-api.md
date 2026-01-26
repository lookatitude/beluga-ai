# Registry Public Entry and Helpers

**NewXxx(ctx, name, config):** Main public entry. Apply options, validate config, then call the registry's `Get`/`Create(name, config)`. The registry (or the factory it calls) must also validate config at creation time.

**Required:** Every provider package with a registry MUST expose:

- `ListProviders()` or `ListXxx()` — returns registered provider names.
- `IsRegistered(name string) bool` — reports whether a name is registered.

Optional: additional helpers (e.g. `ListLLMs`) when the package has more than one registry or surface.
