# Dimension Lookup

Hardcoded model-to-dimension mapping.

```go
func (e *OpenAIEmbedder) GetDimension(ctx context.Context) (int, error) {
    switch e.config.Model {
    case "text-embedding-ada-002":
        return 1536, nil
    case "text-embedding-3-small":
        return 1536, nil
    case "text-embedding-3-large":
        return 3072, nil
    default:
        return 1536, nil  // Silent fallback!
    }
}
```

## Why Hardcoded?
- Avoids API call to determine dimension
- Faster initialization
- Works offline

## Limitations
- New models require code changes
- Unknown models silently default to 1536
- No warning logged for unknown models

## Adding New Models
Update the switch statement in each provider's `GetDimension()` method.
