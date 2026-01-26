# Blank Imports for Provider Registration

Register providers via init() using blank imports.

```go
import (
    // Standard imports
    "context"
    "fmt"

    // Package APIs
    "github.com/lookatitude/beluga-ai/pkg/embeddings"
    "github.com/lookatitude/beluga-ai/pkg/vectorstores"

    // Blank imports trigger init() for provider registration
    _ "github.com/lookatitude/beluga-ai/pkg/chatmodels/providers/openai"
    _ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/mock"
    _ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/openai"
    _ "github.com/lookatitude/beluga-ai/pkg/vectorstores/providers/inmemory"
)
```

## How It Works
1. Blank import (`_ "pkg/path"`) executes package's `init()` function
2. Provider's `init()` calls `Register()` on global registry
3. Example can now use `NewProvider(ctx, "openai", config)`

## Import Organization
```go
import (
    // 1. Standard library
    "context"

    // 2. Third-party packages
    "github.com/stretchr/testify/assert"

    // 3. Package APIs (direct imports)
    "github.com/lookatitude/beluga-ai/pkg/embeddings"

    // 4. Blank imports for side effects (grouped, commented)
    _ "github.com/lookatitude/beluga-ai/pkg/embeddings/providers/openai"
)
```

## Why This Pattern
- Example controls which providers are available
- No global provider pollution
- Clear visibility of dependencies
