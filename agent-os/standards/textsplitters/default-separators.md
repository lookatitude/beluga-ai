# Default Separator Hierarchy

Text splitting uses a semantic hierarchy of separators.

```go
if len(config.Separators) == 0 {
    config.Separators = []string{"\n\n", "\n", " ", ""}
}
```

## Hierarchy (in order)
| Separator | Purpose | Semantic Level |
|-----------|---------|----------------|
| `"\n\n"` | Paragraph boundaries | Highest |
| `"\n"` | Line boundaries | High |
| `" "` | Word boundaries | Medium |
| `""` | Character-level (fallback) | Lowest |

## How It Works
1. Try first separator; if text doesn't split, try next
2. If part still exceeds ChunkSize after splitting, recursively try remaining separators
3. Empty string `""` triggers character-by-character fallback

## Customization
```go
// Markdown-optimized separators
config.Separators = []string{
    "\n## ", "\n### ", "\n\n", "\n", " ", "",
}
```

## Note
Empty string `""` should always be last - it's the ultimate fallback.
