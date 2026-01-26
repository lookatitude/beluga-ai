# Registry Shape

**Location:** Prefer `registry.go` at the package root. Use a `registry/` subpackage only when the registry has enough logic or deps to justify it.

**Singleton:** `GetRegistry()` returning the global instance, built with `sync.Once`. It's also OK to have `NewRegistry()` and pass it around (e.g. in tests).

**Methods:** `Register(name, factory)` and `Get`/`Create(name, config)`. Store factories in a `map[string]func(...)`. Protect with `sync.RWMutex` (Lock in `Register`, RLock in `Get`/`Create`).

**Factory type:** `func(Config) (Interface, error)` or `func(ctx context.Context, config Config) (Interface, error)` depending on the package. Config is the package's `Config` type.
