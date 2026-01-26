# Pipeline Selection

Process() routes to reasoning or generation pipeline based on content.

## Current Auto-Selection
```go
// Checks first non-text block
for _, block := range blocks {
    if block.Type != "text" {
        hasMultimodalContent = true
        break
    }
}

if hasMultimodalContent {
    return m.reasoningPipeline(ctx, input, blocks, routing)
}
return m.generationPipeline(ctx, input, blocks)
```

## Pipeline Behaviors

| Pipeline | Input | Output |
|----------|-------|--------|
| Reasoning | Image/audio/video + text | Text analysis/description |
| Generation | Text instructions | Generated content |

## Future: Explicit Selection
Callers should be able to force pipeline via input config (not yet implemented):
```go
input.Metadata["pipeline"] = "reasoning" // Force reasoning even for text
```

## Fallback Behavior
- Reasoning without LLM returns input blocks unchanged (confidence: 0.90)
- Generation without multimodal returns prefixed text response
