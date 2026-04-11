# Pattern: Registry + Factory

## What it is

A runtime-discoverable registry where providers self-register during `init()` and consumers look them up by name. Three functions make up the contract: `Register(name, factory)`, `New(name, cfg)`, and `List()`.

## Why we use it

You need runtime discovery of pluggable components without introducing import cycles. A user's code should be able to say `llm.New("openai", cfg)` without the `llm` package having any knowledge of the OpenAI provider, and without editing a central configuration file.

**Alternatives considered:**
- **Config files.** Every framework that tried it ended up with stale registrations, typos that fail at runtime, and files that had to be updated alongside code changes. Rejected.
- **Compile-time registration via build tags.** Works but requires users to know the right tags. Rejected for usability.
- **Reflection-based discovery.** Fragile, slow to start, and makes tooling hard. Rejected.

`init()`-based registration is the Go-idiomatic answer: the import graph is the source of truth. If you import `_ "github.com/lookatitude/beluga-ai/llm/providers/openai"`, the provider is registered. No config, no ceremony.

## How it works

Canonical code from `llm/registry.go:19-27` (see [`.wiki/patterns/provider-registration.md`](../../.wiki/patterns/provider-registration.md)):

```go
// llm/registry.go
package llm

import (
    "fmt"
    "sync"
)

type Factory func(cfg Config) (Provider, error)

var (
    mu        sync.RWMutex
    providers = make(map[string]Factory)
)

func Register(name string, factory Factory) error {
    mu.Lock()
    defer mu.Unlock()
    if _, exists := providers[name]; exists {
        return fmt.Errorf("provider %q already registered", name)
    }
    providers[name] = factory
    return nil
}

func New(name string, cfg Config) (Provider, error) {
    mu.RLock()
    defer mu.RUnlock()
    factory, ok := providers[name]
    if !ok {
        return nil, fmt.Errorf("provider %q not found", name)
    }
    return factory(cfg)
}

func List() []string {
    mu.RLock()
    defer mu.RUnlock()
    names := make([]string, 0, len(providers))
    for name := range providers {
        names = append(names, name)
    }
    return names
}
```

Provider registration from `llm/providers/anthropic/anthropic.go:19-23`:

```go
package anthropic

import "github.com/lookatitude/beluga-ai/llm"

func init() {
    if err := llm.Register("anthropic", newFactory()); err != nil {
        panic(err) // duplicate registration is a programming error
    }
}

func newFactory() llm.Factory {
    return func(cfg llm.Config) (llm.Provider, error) {
        return &anthropicProvider{cfg: cfg}, nil
    }
}
```

## Where it's used

| Package | Registry type | Global or instance |
|---|---|---|
| `llm` | provider registry | global |
| `tool` | tool registry | instance (passed explicitly) |
| `memory` | store registry | global |
| `memory/stores` | message store registry | global |
| `rag/embedding` | embedder registry | global |
| `rag/vectorstore` | vector store registry | global |
| `rag/retriever` | retriever strategy registry | global |
| `voice/stt`, `voice/tts`, `voice/s2s` | voice provider registries | global |
| `voice/transport` | transport registry | global |
| `guard` | guard registry | global |
| `workflow` | workflow engine registry | global |
| `server` | server framework registry | global |
| `cache` | cache backend registry | global |
| `auth` | auth provider registry | global |
| `state` | state store registry | global |
| `agent` | planner registry | global |

See [`.wiki/architecture/package-map.md`](../../.wiki/architecture/package-map.md) for the current live list.

## Common mistakes

From [`.wiki/corrections.md`](../../.wiki/corrections.md) and field experience:

- **Registering outside `init()`.** The registry is append-only at startup. Runtime registration is a race condition waiting to happen.
- **Silent overwrites on duplicate registration.** If two providers claim the same name, the second one silently wins — and you spend hours debugging. Always return an error on duplicates.
- **Panicking from `init()` without justification.** Panic is appropriate only for "this is a programming error that must be fixed before the program can run" — duplicate registration fits. Unrecoverable config errors don't; return an error instead.
- **Unprotected map access.** Without `sync.RWMutex`, concurrent `Register` and `New` races. Use `RWMutex` for read-heavy access.
- **Empty registry at `New` time.** If you forgot the import, you'll get "provider not found" with zero context. Consider logging `List()` when an error occurs.

## Example: implementing your own

Let's create a fictional `cipher` package with a registry of encryption algorithms:

```go
// cipher/cipher.go
package cipher

import (
    "fmt"
    "sync"
)

type Cipher interface {
    Encrypt(plaintext []byte) ([]byte, error)
    Decrypt(ciphertext []byte) ([]byte, error)
}

type Factory func(key []byte) (Cipher, error)

var (
    mu      sync.RWMutex
    ciphers = make(map[string]Factory)
)

func Register(name string, factory Factory) error {
    mu.Lock()
    defer mu.Unlock()
    if _, exists := ciphers[name]; exists {
        return fmt.Errorf("cipher %q already registered", name)
    }
    ciphers[name] = factory
    return nil
}

func New(name string, key []byte) (Cipher, error) {
    mu.RLock()
    defer mu.RUnlock()
    factory, ok := ciphers[name]
    if !ok {
        return nil, fmt.Errorf("cipher %q not found", name)
    }
    return factory(key)
}

func List() []string {
    mu.RLock()
    defer mu.RUnlock()
    names := make([]string, 0, len(ciphers))
    for name := range ciphers {
        names = append(names, name)
    }
    return names
}
```

And an implementation:

```go
// cipher/providers/aesgcm/aesgcm.go
package aesgcm

import "github.com/lookatitude/beluga-ai/cipher"

func init() {
    if err := cipher.Register("aes-gcm", newFactory()); err != nil {
        panic(err)
    }
}

func newFactory() cipher.Factory {
    return func(key []byte) (cipher.Cipher, error) {
        if len(key) != 32 {
            return nil, fmt.Errorf("aes-gcm requires a 32-byte key")
        }
        return &aesGCMCipher{key: key}, nil
    }
}
```

User code:

```go
import _ "github.com/lookatitude/beluga-ai/cipher/providers/aesgcm"

c, err := cipher.New("aes-gcm", key)
if err != nil {
    return err
}
ct, err := c.Encrypt(plaintext)
```

That's the whole pattern. Three functions, one init, and the user is unaware of how the provider got there.

## Related

- [03 — Extensibility Patterns](../architecture/03-extensibility-patterns.md#ring-2--registry)
- [`patterns/provider-template.md`](./provider-template.md) — full provider implementation walk-through.
- [`.wiki/patterns/provider-registration.md`](../../.wiki/patterns/provider-registration.md) — canonical code references.
