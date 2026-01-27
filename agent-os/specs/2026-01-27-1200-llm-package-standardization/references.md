# Reference Files

## Reference Implementation: pkg/tools

### pkg/tools/registry.go
Full registry implementation with:
- `ToolRegistry` struct
- `sync.RWMutex` for thread safety
- Global `globalToolRegistry` variable (no sync.Once, direct initialization)
- `GetRegistry()` function
- Global convenience functions
- Interface compliance check

### pkg/tools/errors.go
Error pattern with:
- `ErrorCode` type
- Error code constants
- Static base errors for wrapping
- `ToolError` struct with Op/Err/Code/Message
- Constructor functions
- Helper functions (`IsToolError`, `AsToolError`, `GetErrorCode`)

## Files to Modify

### pkg/embeddings

**Delete:**
- `pkg/embeddings/iface/errors.go`
- `pkg/embeddings/internal/registry/registry.go`
- `pkg/embeddings/internal/registry/` (directory)
- `pkg/embeddings/factory.go`
- `pkg/embeddings/embeddings_mock.go`
- `pkg/embeddings/advanced_mock.go`
- `pkg/embeddings/testutils/helpers.go`
- `pkg/embeddings/testutils/` (directory)

**Modify:**
- `pkg/embeddings/errors.go` - Add missing error codes from iface/errors.go
- `pkg/embeddings/registry.go` - Replace with full implementation
- `pkg/embeddings/test_utils.go` - Merge mock files
- `pkg/embeddings/iface/registry.go` - Keep interface only
- `pkg/embeddings/providers/*/init.go` - Update to use root registry

**Move:**
- `pkg/embeddings/iface/iface_test.go` -> `pkg/embeddings/errors_test.go`

### pkg/chatmodels

**Delete:**
- `pkg/chatmodels/advanced_mock.go`
- `pkg/chatmodels/internal/` (empty directory, if exists)

**Modify:**
- `pkg/chatmodels/registry.go` - Add full implementation
- `pkg/chatmodels/iface/registry.go` - Keep interface + types only
- `pkg/chatmodels/test_utils.go` - Merge advanced_mock.go

## Provider init.go Files to Update

### pkg/embeddings/providers/*/init.go
6 files total:
- `cohere/init.go`
- `google_multimodal/init.go`
- `mock/init.go`
- `ollama/init.go`
- `openai/init.go`
- `openai_multimodal/init.go`

Change from:
```go
import "github.com/lookatitude/beluga-ai/pkg/embeddings/internal/registry"
registry.GetRegistry().Register(...)
```

To:
```go
import "github.com/lookatitude/beluga-ai/pkg/embeddings"
embeddings.Register(...)
```
