#!/bin/bash
# Validate that all examples compile and can be run (dry-run mode)

set -e

echo "=== Validating Document Loaders Examples ==="

# Basic example
echo "Checking examples/documentloaders/basic/main.go..."
cd examples/documentloaders/basic
go build -o /dev/null main.go 2>&1 || { echo "FAILED: basic/main.go"; exit 1; }
echo "✓ basic/main.go compiles"

# Directory example
echo "Checking examples/documentloaders/directory/main.go..."
cd ../directory
go build -o /dev/null main.go 2>&1 || { echo "FAILED: directory/main.go"; exit 1; }
echo "✓ directory/main.go compiles"

cd ../../..

echo ""
echo "=== Validating Text Splitters Examples ==="

# Basic example
echo "Checking examples/textsplitters/basic/main.go..."
cd examples/textsplitters/basic
go build -o /dev/null main.go 2>&1 || { echo "FAILED: basic/main.go"; exit 1; }
echo "✓ basic/main.go compiles"

# Token-based example
echo "Checking examples/textsplitters/token_based/main.go..."
cd ../token_based
go build -o /dev/null main.go 2>&1 || { echo "FAILED: token_based/main.go"; exit 1; }
echo "✓ token_based/main.go compiles"

cd ../../..

echo ""
echo "=== Validating RAG Examples ==="

# With loaders example
echo "Checking examples/rag/with_loaders/main.go..."
cd examples/rag/with_loaders
go build -o /dev/null main.go 2>&1 || { echo "FAILED: with_loaders/main.go"; exit 1; }
echo "✓ with_loaders/main.go compiles"

cd ../../..

echo ""
echo "✅ All examples compile successfully!"
