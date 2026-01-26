# Character-Level Fallback Guarantee

When no separator works, split by individual characters.

```go
func (s *Splitter) splitTextRecursive(text string, separators []string, lengthFn func(string) int) []string {
    if len(separators) == 0 {
        // Fallback: split character by character
        return s.splitByCharacters(text, lengthFn)
    }
    // ... try separators
}

func (s *Splitter) splitByCharacters(text string, lengthFn func(string) int) []string {
    var chunks []string
    var currentChunk strings.Builder

    for _, char := range text {  // Iterates runes, not bytes
        testChunk := currentChunk.String() + string(char)
        if lengthFn(testChunk) > s.config.ChunkSize && currentChunk.Len() > 0 {
            chunks = append(chunks, currentChunk.String())
            currentChunk.Reset()
            // Add overlap...
        }
        currentChunk.WriteRune(char)
    }
    // ...
}
```

## Separator Hierarchy
Default: `["\n\n", "\n", " ", ""]`
- `"\n\n"` - Paragraph boundaries
- `"\n"` - Line boundaries
- `" "` - Word boundaries
- `""` - Character-by-character (empty separator triggers fallback)

## Guarantee
Chunks will never exceed ChunkSize (assuming single runes fit).

## Edge Case
If a single rune exceeds ChunkSize, current implementation returns it anyway. Consider returning an error for this edge case.
