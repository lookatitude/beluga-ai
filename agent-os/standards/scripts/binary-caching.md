# Binary Caching with Hierarchical Naming

Cache test binaries in `.cache/` with nested package naming.

```bash
#!/bin/bash
set -e

PACKAGE_PATH="$1"

# Get package name from path
PACKAGE_NAME=$(basename "$PACKAGE_PATH")

# For nested packages, use parent_child format
if [ "$(dirname "$PACKAGE_PATH")" != "." ] && \
   [ "$(dirname "$PACKAGE_PATH")" != "./pkg" ]; then
    PARENT_DIR=$(basename "$(dirname "$PACKAGE_PATH")")
    PACKAGE_NAME="${PARENT_DIR}_${PACKAGE_NAME}"
fi

# Create cache directory
mkdir -p .cache/test-binaries

# Build test binary to cache
go test -c -o ".cache/test-binaries/$PACKAGE_NAME.test" "$PACKAGE_PATH"

echo "✅ Built: .cache/test-binaries/$PACKAGE_NAME.test"
```

## Naming Examples

| Package Path | Binary Name |
|--------------|-------------|
| `./pkg/agents` | `agents.test` |
| `./pkg/llms/providers/openai` | `providers_openai.test` |
| `./tests/integration/voice` | `integration_voice.test` |

## Why This Pattern
- Prevents test binaries from cluttering source directories
- Hierarchical naming avoids collisions
- `.cache/` can be gitignored
- Easy to clean: `rm -rf .cache/test-binaries`

## Cache Location
- Use `.cache/` at repo root
- Add `.cache/` to `.gitignore`
- Subdirectories: `test-binaries/`, `build/`, etc.
