# Pattern: Registry + Factory

## What it is

A runtime-discoverable registry where providers self-register during `init()` and consumers look them up by name. Three functions make up the contract: `Register(name, factory)`, `New(name, cfg)`, and `List()`.

## Why we use it

You need runtime discovery of pluggable components without introducing import cycles. A user's code should be able to say `llm.New("openai", cfg)` without the `llm` package having any knowledge of the OpenAI provider, and without editing a central configuration file.

**Alternatives considered:**
- **Config files.** Every framework that tried it ended up with stale registrations, typos that fail at runtime, and files that had to be updated alongside code changes. Rejected.
- **Compile-time registration via build tags.** Works but requires users to know the right tags. Rejected for usability.
- **Reflection-based discovery.** Fragile, slow to start, and makes tooling hard. Rejected.

`init()`-based registration is the Go-idiomatic answer: the import graph is the source of truth. If you import `_ "github.com/lookatitude/beluga-ai/v2/llm/providers/openai"`, the provider is registered. No config, no ceremony.

## How it works

Canonical code from `llm/registry.go` (see [`.guild/wiki/patterns/provider-registration.md`](../../.guild/wiki/patterns/provider-registration.md)):

```go
// llm/registry.go (excerpt, doc comments trimmed)
type Factory func(cfg config.ProviderConfig) (ChatModel, error)

var (
    registryMu sync.RWMutex
    registry   = make(map[string]Factory)
)

// Register adds a provider factory. Duplicate registrations for the same
// name silently overwrite the previous factory.
func Register(name string, f Factory) {
    registryMu.Lock()
    defer registryMu.Unlock()
    registry[name] = f
}

func New(name string, cfg config.ProviderConfig) (ChatModel, error) {
    registryMu.RLock()
    f, ok := registry[name]
    registryMu.RUnlock()
    if !ok {
        return nil, core.Errorf(core.ErrNotFound, "llm: unknown provider %q (registered: %v)", name, List())
    }
    return f(cfg)
}

func List() []string {
    registryMu.RLock()
    defer registryMu.RUnlock()
    names := make([]string, 0, len(registry))
    for name := range registry {
        names = append(names, name)
    }
    sort.Strings(names)
    return names
}
```

Provider registration from `llm/providers/anthropic/anthropic.go`:

```go
func init() {
    llm.Register("anthropic", func(cfg config.ProviderConfig) (llm.ChatModel, error) {
        return New(cfg)
    })
}
```

## Where it's used

| Package | Registry of | Global or instance |
|---|---|---|
| `llm` | chat model providers | global |
| `tool` | tools (`tool.NewRegistry`) | instance (passed explicitly) |
| `memory` | memory providers | global |
| `rag/embedding` | embedders | global |
| `rag/vectorstore` | vector stores | global |
| `rag/retriever` | retriever strategies | global |
| `rag/loader`, `rag/splitter` | loaders, splitters | global |
| `voice/stt`, `voice/tts`, `voice/s2s`, `voice/transport` | voice providers | global |
| `guard` | guards | global |
| `workflow` | durable executors | global |
| `hitl` | HITL managers | global |
| `server` | HTTP framework adapters | global |
| `cache` | cache backends | global |
| `auth` | auth policies | global |
| `audit`, `cost`, `state` | stores and trackers | global |
| `agent` | planners (`RegisterPlanner`) | global |

Duplicate handling is not uniform yet: `workflow`, `hitl` and `guard` panic on a duplicate or empty name; the others overwrite silently.

See [`.guild/wiki/architecture/package-map.md`](../../.guild/wiki/architecture/package-map.md) for the current live list.

## Common mistakes

From [`.guild/wiki/corrections/`](../../.guild/wiki/corrections/) and field experience:

- **Registering outside `init()`.** Registries are meant to be filled at startup. Registering later is mutex-safe but makes which factory wins depend on timing.
- **Silent overwrites on duplicate registration.** In registries that overwrite, if two providers claim the same name the second one silently wins and you spend hours debugging. In a new registry, panic on a duplicate name in `Register` so the mistake fails at startup.
- **Panicking from `init()` without justification.** Panic is appropriate only for "this is a programming error that must be fixed before the program can run" — duplicate registration fits. Unrecoverable config errors don't; return an error instead.
- **Unprotected map access.** Without `sync.RWMutex`, concurrent `Register` and `New` races. Use `RWMutex` for read-heavy access.
- **Empty registry at `New` time.** If you forgot the import, `New` fails. `llm.New` includes `List()` in its error message; do the same in your own registries.

## Example: implementing your own

Let's create a fictional `cipher` package, in your own module, with a registry of encryption algorithms:

```go
// cipher/cipher.go
package cipher

import (
    "fmt"
    "sort"
    "sync"
)

// Cipher encrypts and decrypts byte slices.
type Cipher interface {
    Encrypt(plaintext []byte) ([]byte, error)
    Decrypt(ciphertext []byte) ([]byte, error)
}

// Factory builds a Cipher from a key.
type Factory func(key []byte) (Cipher, error)

var (
    mu      sync.RWMutex
    ciphers = make(map[string]Factory)
)

// Register adds a factory. It panics on a duplicate name, so a clash fails at
// program start.
func Register(name string, factory Factory) {
    mu.Lock()
    defer mu.Unlock()
    if _, exists := ciphers[name]; exists {
        panic(fmt.Sprintf("cipher: %q already registered", name))
    }
    ciphers[name] = factory
}

// New builds the named cipher.
func New(name string, key []byte) (Cipher, error) {
    mu.RLock()
    factory, ok := ciphers[name]
    mu.RUnlock()
    if !ok {
        return nil, fmt.Errorf("cipher %q not found (registered: %v)", name, List())
    }
    return factory(key)
}

// List returns the registered names, sorted.
func List() []string {
    mu.RLock()
    defer mu.RUnlock()
    names := make([]string, 0, len(ciphers))
    for name := range ciphers {
        names = append(names, name)
    }
    sort.Strings(names)
    return names
}
```

And an implementation. It lives in the same package here to keep the example short; in a real module it would be a subpackage that imports `cipher` and is blank-imported by users.

```go
// cipher/aesgcm.go
package cipher

import (
    "crypto/aes"
    gocipher "crypto/cipher"
    "crypto/rand"
    "errors"
)

func init() {
    Register("aes-gcm", func(key []byte) (Cipher, error) {
        if len(key) != 32 {
            return nil, errors.New("aes-gcm requires a 32-byte key")
        }
        block, err := aes.NewCipher(key)
        if err != nil {
            return nil, err
        }
        aead, err := gocipher.NewGCM(block)
        if err != nil {
            return nil, err
        }
        return &aesGCM{aead: aead}, nil
    })
}

type aesGCM struct{ aead gocipher.AEAD }

func (c *aesGCM) Encrypt(plaintext []byte) ([]byte, error) {
    nonce := make([]byte, c.aead.NonceSize())
    if _, err := rand.Read(nonce); err != nil {
        return nil, err
    }
    return c.aead.Seal(nonce, nonce, plaintext, nil), nil
}

func (c *aesGCM) Decrypt(ciphertext []byte) ([]byte, error) {
    n := c.aead.NonceSize()
    if len(ciphertext) < n {
        return nil, errors.New("aes-gcm: ciphertext too short")
    }
    return c.aead.Open(nil, ciphertext[:n], ciphertext[n:], nil)
}
```

User code:

```go
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
- [`.guild/wiki/patterns/provider-registration.md`](../../.guild/wiki/patterns/provider-registration.md) — canonical code references.
