#!/bin/bash
# Build a test binary to .cache/test-binaries instead of the current directory
# Usage: ./scripts/build-test.sh ./pkg/agents

set -e

if [ $# -eq 0 ]; then
    echo "Usage: $0 <package-path>"
    echo "Example: $0 ./pkg/agents"
    exit 1
fi

PACKAGE_PATH="$1"
if [ ! -d "$PACKAGE_PATH" ]; then
    echo "Error: Directory $PACKAGE_PATH does not exist"
    exit 1
fi

# Get package name from path
PACKAGE_NAME=$(basename "$PACKAGE_PATH")
# If it's a nested package, use parent_dir_child_dir format
if [ "$(dirname "$PACKAGE_PATH")" != "." ] && [ "$(dirname "$PACKAGE_PATH")" != "./pkg" ] && [ "$(dirname "$PACKAGE_PATH")" != "./tests" ]; then
    PARENT_DIR=$(basename "$(dirname "$PACKAGE_PATH")")
    PACKAGE_NAME="${PARENT_DIR}_${PACKAGE_NAME}"
fi

# Create .cache/test-binaries directory
mkdir -p .cache/test-binaries

# Build test binary to .cache/test-binaries
echo "Building test binary for $PACKAGE_PATH -> .cache/test-binaries/$PACKAGE_NAME.test"
go test -c -o ".cache/test-binaries/$PACKAGE_NAME.test" "$PACKAGE_PATH"

echo "✅ Test binary built: .cache/test-binaries/$PACKAGE_NAME.test"
