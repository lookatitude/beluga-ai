#!/bin/bash

# Simple script to ONLY analyze errors, no fixing
# This is safer and won't make changes

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$PROJECT_ROOT"

echo "══════════════════════════════════════════════════════════"
echo "          🔍 ERROR ANALYSIS (READ-ONLY)"
echo "══════════════════════════════════════════════════════════"
echo ""

# Run linter and capture errors
echo "Running golangci-lint..."
ERRORS_FILE=$(mktemp)
timeout 180 ~/go/bin/golangci-lint run --timeout=10m ./pkg/... ./tests/... 2>&1 | \
    grep -E "(undefined: ctx|syntax error|no new variables)" > "$ERRORS_FILE" || true

ERROR_COUNT=$(wc -l < "$ERRORS_FILE" | tr -d ' ')
echo "Found $ERROR_COUNT errors"
echo ""

if [ "$ERROR_COUNT" -eq 0 ]; then
    echo "🎉 No errors found!"
    rm -f "$ERRORS_FILE"
    exit 0
fi

# Group by file
echo "══════════════════════════════════════════════════════════"
echo "          📊 ERRORS BY FILE"
echo "══════════════════════════════════════════════════════════"
echo ""

ERRORS_FILE_PATH="$ERRORS_FILE" python3 << 'PYTHON_EOF'
import re
import os
from collections import defaultdict

errors_file = os.environ.get('ERRORS_FILE_PATH')
if not errors_file:
    print("Error: ERRORS_FILE_PATH environment variable not set")
    sys.exit(1)

# Parse errors
file_errors = defaultdict(lambda: {'undefined': [], 'syntax': [], 'duplicate': []})

try:
    with open(errors_file, 'r') as f:
        for line_num_in_file, line in enumerate(f, 1):
            if not line.strip():
                continue
            try:
                # Match format: pkg/file.go:line:col: error message
                match = re.search(r'(pkg/[^:]+\.go):(\d+):', line)
                if match:
                    filepath = match.group(1)
                    line_num = int(match.group(2))
                    # Get error message - everything after the second colon
                    # Format is: file:line:col: message
                    colons = [m.start() for m in re.finditer(r':', line)]
                    if len(colons) >= 2:
                        # Get text after second colon
                        error_msg = line[colons[1]+1:].strip()
                    else:
                        error_msg = line.strip()
                    
                    if 'undefined: ctx' in error_msg:
                        file_errors[filepath]['undefined'].append((line_num, error_msg))
                    elif 'syntax error' in error_msg:
                        file_errors[filepath]['syntax'].append((line_num, error_msg))
                    elif 'no new variables' in error_msg:
                        file_errors[filepath]['duplicate'].append((line_num, error_msg))
            except Exception as parse_err:
                # Skip lines that can't be parsed
                continue
except Exception as e:
    print(f"Error reading errors file: {e}")
    sys.exit(1)

# Sort by total errors
sorted_files = sorted(file_errors.items(), key=lambda x: sum(len(v) for v in x[1].values()), reverse=True)

if not sorted_files:
    print("No errors found in parsed data.")
    sys.exit(0)

print("Files with most errors first:\n")
for filepath, errors in sorted_files:
    total = sum(len(v) for v in errors.values())
    undefined = len(errors['undefined'])
    syntax = len(errors['syntax'])
    duplicate = len(errors['duplicate'])
    
    print(f"📄 {filepath}")
    print(f"   Total: {total} (undefined: {undefined}, syntax: {syntax}, duplicate: {duplicate})")
    
    # Show first few errors of each type
    if errors['undefined']:
        print(f"   Undefined ctx (showing first 3):")
        for ln, msg in sorted(errors['undefined'])[:3]:
            msg_display = msg[:60] if len(msg) > 60 else msg
            print(f"      Line {ln}: {msg_display}")
        if len(errors['undefined']) > 3:
            print(f"      ... and {len(errors['undefined']) - 3} more")
    
    if errors['syntax']:
        print(f"   Syntax errors (showing first 3):")
        for ln, msg in sorted(errors['syntax'])[:3]:
            msg_display = msg[:60] if len(msg) > 60 else msg
            print(f"      Line {ln}: {msg_display}")
        if len(errors['syntax']) > 3:
            print(f"      ... and {len(errors['syntax']) - 3} more")
    
    if errors['duplicate']:
        print(f"   Duplicates (showing first 3):")
        for ln, msg in sorted(errors['duplicate'])[:3]:
            msg_display = msg[:60] if len(msg) > 60 else msg
            print(f"      Line {ln}: {msg_display}")
        if len(errors['duplicate']) > 3:
            print(f"      ... and {len(errors['duplicate']) - 3} more")
    
    print()

print(f"\nTotal: {len(sorted_files)} files with errors")
PYTHON_EOF

# Cleanup
rm -f "$ERRORS_FILE"

echo ""
echo "✅ Analysis complete (no changes made)"

