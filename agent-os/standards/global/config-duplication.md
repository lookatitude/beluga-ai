# Config Duplication for Import Cycles

Providers may duplicate parent config structs to break circular imports.

## Problem
```
pkg/textsplitters/config.go       imports providers/recursive
providers/recursive/splitter.go   wants to import textsplitters.SplitterConfig
```
This creates an import cycle.

## Solution
```go
// In providers/recursive/splitter.go
// RecursiveConfig duplicated here to avoid import cycle
type RecursiveConfig struct {
    LengthFunction func(string) int
    Separators     []string
    ChunkSize      int
    ChunkOverlap   int
}
```

## Where This Appears
- `pkg/textsplitters/providers/recursive/splitter.go` - `RecursiveConfig`
- `pkg/documentloaders/providers/text/loader.go` - `LoaderConfig`
- `pkg/vectorstores/providers/inmemory/inmemory_vectorstore.go` - `Config`

## Guidelines
- Add comment explaining why duplication exists
- Keep duplicated struct in sync with parent
- Consider if provider really needs full config or just subset
- Alternative: Extract config to separate package (e.g., `types/`)
