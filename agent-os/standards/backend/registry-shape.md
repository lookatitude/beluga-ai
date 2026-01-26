# Registry Shape

**Location:** Prefer `registry.go` at the package root. Use a `registry/` subpackage only when the registry has enough logic or deps to justify it.

**Singleton:** `GetRegistry()` returning the global instance, built with `sync.Once`. It's also OK to have `NewRegistry()` and pass it around (e.g. in tests).

**Methods:** `Register(name, factory)` and `Get`/`Create(name, config)`. Store factories in a `map[string]func(...)`. Protect with `sync.RWMutex` (Lock in `Register`, RLock in `Get`/`Create`).

**Factory type:** `func(Config) (Interface, error)` or `func(ctx context.Context, config Config) (Interface, error)` depending on the package. Config is the package's `Config` type.

## Nested/Hierarchical Registries (Wrapper Packages)

Wrapper packages use facade registries that delegate to sub-package registries:

```go
// pkg/voice/registry.go
type Registry struct {
    STT       *stt.Registry
    TTS       *tts.Registry
    VAD       *vad.Registry
    Transport *transport.Registry
}

var (
    globalRegistry *Registry
    registryOnce   sync.Once
)

func GetRegistry() *Registry {
    registryOnce.Do(func() {
        globalRegistry = &Registry{
            STT:       stt.GetRegistry(),
            TTS:       tts.GetRegistry(),
            VAD:       vad.GetRegistry(),
            Transport: transport.GetRegistry(),
        }
    })
    return globalRegistry
}
```

### Delegation Pattern

Facade registries delegate to sub-package registries rather than duplicating functionality:

```go
// Delegate to sub-package registry
func (r *Registry) GetSTTProvider(name string, cfg *stt.Config) (stt.iface.Transcriber, error) {
    return r.STT.GetProvider(name, cfg)
}

// Aggregate providers from all sub-packages
func (r *Registry) ListAllProviders() map[string][]string {
    return map[string][]string{
        "stt":       r.STT.ListProviders(),
        "tts":       r.TTS.ListProviders(),
        "vad":       r.VAD.ListProviders(),
        "transport": r.Transport.ListProviders(),
    }
}
```

### Sub-Package Registry Independence

Each sub-package maintains its own registry that can be used directly:

```go
// Direct use of sub-package registry
stt.GetRegistry().Register("custom", NewCustomTranscriber)
transcriber, err := stt.GetRegistry().GetProvider("custom", cfg)
```
