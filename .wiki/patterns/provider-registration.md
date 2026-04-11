# Pattern: Provider Registration

**Status:** stub — populate with `/wiki-learn`

## Contract

Every extensible package exposes three functions and a registry:

```go
var registry = make(map[string]Factory)

func Register(name string, f Factory) { registry[name] = f } // init()-only
func New(name string, cfg Config) (Interface, error)          // factory lookup
func List() []string                                           // discovery
```

Providers call `Register()` from `init()`. No runtime mutation. `New()` returns a typed error if the name is unknown.

## Canonical example

(populate via `/wiki-learn` — scan for `func Register` in registry-pattern packages)

## Variations

(populated by /wiki-learn)

## Anti-patterns

- Registering outside `init()` (introduces race conditions).
- Mutating the registry map without the registry's own mutex.
- Silent overwrite on duplicate Register calls — use an error or panic in init.
- Global state that isn't the registry map itself.

## Related

- `patterns/hooks.md`
- `patterns/middleware.md`
- `architecture/invariants.md#7-registry-pattern-everywhere`
