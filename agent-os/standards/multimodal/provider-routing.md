# Provider Routing

Router selects providers for each content block based on modality.

## Routing Strategies

```go
input.Routing = map[string]any{
    "strategy": "auto",   // Default: heuristic detection
    "strategy": "manual", // Explicit provider per modality
    "strategy": "fallback", // Try provider, fallback to text
}
```

## Manual Routing
```go
input.Routing = map[string]any{
    "strategy": "manual",
    "image_provider": "openai",
    "audio_provider": "google",
    "fallback_to_text": true,
}
```

## Capability Detection (Heuristics)

Known multimodal providers (hardcoded): `openai`, `google`, `anthropic`, `xai`
- These are assumed to support all modalities
- Other providers assumed text-only
- Heuristics are acceptable; explicit capability queries not required

## Fallback Behavior
- If no provider found for modality, falls back to first available text provider
- `fallback_to_text: false` makes missing provider an error
