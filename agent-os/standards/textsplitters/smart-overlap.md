# Smart Overlap with Break-Point Heuristics

Overlap extraction tries semantic boundaries before byte fallback.

```go
func (s *Splitter) getOverlapText(chunk string, overlapSize int, lengthFn func(string) int) string {
    if lengthFn(chunk) <= overlapSize {
        return chunk
    }

    // Try to find a good break point (space, newline)
    for i := len(chunk) - overlapSize; i < len(chunk); i++ {
        if i > 0 && (chunk[i] == ' ' || chunk[i] == '\n') {
            return chunk[i+1:]  // Start after the break
        }
    }

    // Fallback: just take the last N characters (UTF-8 safe)
    runes := []rune(chunk)
    if len(runes) <= overlapSize {
        return chunk
    }
    return string(runes[len(runes)-overlapSize:])
}
```

## Why Break-Points?
- **Semantic coherence**: Word boundaries keep meaning intact
- **Embedding quality**: Overlaps at word boundaries produce better vector representations

## Fallback Chain
1. Try to start overlap at space or newline
2. If no break-point found, use exact character count
3. UTF-8 safe: uses `[]rune` for multi-byte characters

## Note
Default separators: `["\n\n", "\n", " ", ""]` already favor semantic breaks.
