# pkg/agents/tools - DEPRECATED

> **Warning**: This package is deprecated and will be removed in v2.0.
> Please migrate to the new top-level `pkg/tools` package.

## Migration Guide

Update your imports as follows:

| Old Import | New Import |
|------------|------------|
| `github.com/lookatitude/beluga-ai/pkg/agents/tools` | `github.com/lookatitude/beluga-ai/pkg/tools` |
| `github.com/lookatitude/beluga-ai/pkg/agents/tools/api` | `github.com/lookatitude/beluga-ai/pkg/tools/api` |
| `github.com/lookatitude/beluga-ai/pkg/agents/tools/mcp` | `github.com/lookatitude/beluga-ai/pkg/tools/mcp` |
| `github.com/lookatitude/beluga-ai/pkg/agents/tools/gofunc` | `github.com/lookatitude/beluga-ai/pkg/tools/gofunc` |
| `github.com/lookatitude/beluga-ai/pkg/agents/tools/shell` | `github.com/lookatitude/beluga-ai/pkg/tools/shell` |
| `github.com/lookatitude/beluga-ai/pkg/agents/tools/providers` | `github.com/lookatitude/beluga-ai/pkg/tools/providers` |

## Why the change?

The tools package has been promoted to a top-level package (`pkg/tools/`) to:

1. **Better reusability** - Tools can be imported independently without pulling in the entire agents framework
2. **Clearer organization** - Top-level packages align with Go standard library patterns
3. **Reduced coupling** - Tools no longer depend on the agents package structure

## Backward Compatibility

This package will continue to work until v2.0. Both the old and new import paths point to identical functionality.
