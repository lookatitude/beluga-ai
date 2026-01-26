# Registry Delegation for Provider Creation

Factory delegates to registry instead of direct provider imports.

```go
func NewChatModel(model string, config *Config, opts ...iface.Option) (iface.ChatModel, error) {
    // Use registry to create provider to avoid import cycles
    registry := registry.GetRegistry()
    if registry.IsRegistered(config.DefaultProvider) {
        return registry.CreateProvider(model, config, options)
    }

    // Provider not registered - return helpful error
    return nil, NewChatModelError("creation", model, config.DefaultProvider,
        ErrCodeProviderNotSupported,
        fmt.Errorf("provider '%s' not registered (import the provider package to register it, "+
            "e.g., _ \"github.com/lookatitude/beluga-ai/pkg/chatmodels/providers/openai\")"))
}
```

## How Providers Register
```go
// In providers/openai/init.go
func init() {
    registry.GetRegistry().Register("openai", &OpenAIProviderCreator{})
}
```

## User's Responsibility
```go
import (
    "github.com/lookatitude/beluga-ai/pkg/chatmodels"
    _ "github.com/lookatitude/beluga-ai/pkg/chatmodels/providers/openai"  // Register provider
)
```

## Error Message
Helpful error tells user exactly what import to add:
```
provider 'openai' not registered (import the provider package to register it,
e.g., _ "github.com/lookatitude/beluga-ai/pkg/chatmodels/providers/openai")
```

## Note
This is the same pattern as llms package - dual registries exist.
