# Go Scripts with Build Ignore Tag

Use `//go:build ignore` for standalone Go scripts.

```go
//go:build ignore
// +build ignore

// This script verifies that all providers are properly registered.
// Run with: go run scripts/verify-providers.go
package main

import (
    "fmt"
    "os"

    // Import providers to trigger init() for registration
    _ "github.com/lookatitude/beluga-ai/pkg/llms/providers/gemini"
    _ "github.com/lookatitude/beluga-ai/pkg/llms/providers/grok"

    "github.com/lookatitude/beluga-ai/pkg/llms"
)

func main() {
    registry := llms.GetRegistry()

    // Verify expected providers are registered
    expected := []string{"grok", "gemini"}
    for _, name := range expected {
        if registry.IsRegistered(name) {
            fmt.Printf("  ✓ %s\n", name)
        } else {
            fmt.Printf("  ✗ MISSING: %s\n", name)
            os.Exit(1)
        }
    }

    fmt.Println("✓ All verifications complete!")
}
```

## Build Tag Format
```go
//go:build ignore
// +build ignore
```
- Line 1: New format (Go 1.17+)
- Line 2: Old format (backwards compatibility)
- Blank line required before package declaration

## Why This Pattern
- Script excluded from `go build ./...`
- Can import any package without affecting main binary
- Runs with `go run scripts/script.go`

## Use Cases
- Provider verification scripts
- Code generation tools
- One-off migration scripts
- Development utilities
