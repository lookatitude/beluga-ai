#!/bin/bash
# Build an example binary to .cache/bin instead of the current directory
# Usage: ./scripts/build-example.sh examples/deployment/single_binary

set -e

if [ $# -eq 0 ]; then
    echo "Usage: $0 <example-directory>"
    echo "Example: $0 examples/deployment/single_binary"
    exit 1
fi

EXAMPLE_DIR="$1"
if [ ! -d "$EXAMPLE_DIR" ]; then
    echo "Error: Directory $EXAMPLE_DIR does not exist"
    exit 1
fi

if [ ! -f "$EXAMPLE_DIR/main.go" ]; then
    echo "Error: $EXAMPLE_DIR/main.go does not exist"
    exit 1
fi

# Get the binary name from the directory path
BINARY_NAME=$(basename "$EXAMPLE_DIR")
# If it's a nested example, use parent_dir_child_dir format
if [ "$(dirname "$EXAMPLE_DIR")" != "examples" ]; then
    PARENT_DIR=$(basename "$(dirname "$EXAMPLE_DIR")")
    BINARY_NAME="${PARENT_DIR}_${BINARY_NAME}"
fi

# Create .cache/bin directory
mkdir -p .cache/bin

# Build to .cache/bin
echo "Building $EXAMPLE_DIR -> .cache/bin/$BINARY_NAME"
go build -o ".cache/bin/$BINARY_NAME" "$EXAMPLE_DIR"

echo "✅ Binary built: .cache/bin/$BINARY_NAME"
