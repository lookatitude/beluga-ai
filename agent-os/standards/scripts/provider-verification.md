# Provider Verification via Init Triggering

Verify provider registration by importing and checking registries.

```go
//go:build ignore

package main

import (
    "fmt"
    "os"

    // Blank imports trigger init() which calls Register()
    _ "github.com/lookatitude/beluga-ai/pkg/llms/providers/gemini"
    _ "github.com/lookatitude/beluga-ai/pkg/llms/providers/grok"

    "github.com/lookatitude/beluga-ai/pkg/llms"
)

func main() {
    fmt.Println("Verifying provider registrations...")

    registry := llms.GetRegistry()
    providers := registry.ListProviders()

    // List all registered providers
    fmt.Println("Registered providers:")
    for _, name := range providers {
        fmt.Printf("  ✓ %s\n", name)
    }

    // Verify specific new providers
    newProviders := []string{"grok", "gemini"}
    for _, name := range newProviders {
        if registry.IsRegistered(name) {
            fmt.Printf("  ✓ NEW: %s\n", name)
        } else {
            fmt.Printf("  ✗ MISSING: %s\n", name)
            os.Exit(1)
        }
    }

    fmt.Println("✓ All provider verifications complete!")
}
```

## Output Convention
- `✓` for registered providers
- `✓ NEW:` for newly added providers being verified
- `✗ MISSING:` for expected but unregistered providers

## When to Use
- After adding new providers
- In CI to verify provider registration
- Before releases to check all providers work

## Exit Codes
- `0`: All providers verified
- `1`: One or more providers missing
