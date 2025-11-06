#!/bin/bash
# Helper script to migrate documentation files with Docusaurus frontmatter

SOURCE_FILE=$1
DEST_FILE=$2
TITLE=$3
POSITION=${4:-1}

if [ -z "$SOURCE_FILE" ] || [ -z "$DEST_FILE" ] || [ -z "$TITLE" ]; then
    echo "Usage: $0 <source_file> <dest_file> <title> [position]"
    exit 1
fi

# Create directory if it doesn't exist
mkdir -p "$(dirname "$DEST_FILE")"

# Add frontmatter and copy content
{
    echo "---"
    echo "title: ${TITLE}"
    echo "sidebar_position: ${POSITION}"
    echo "---"
    echo ""
    cat "$SOURCE_FILE"
} > "$DEST_FILE"

echo "Migrated: $SOURCE_FILE -> $DEST_FILE"

