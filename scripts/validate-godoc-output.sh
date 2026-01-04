#!/bin/bash
# Script to validate generated godoc output files
# Checks: all expected files exist, files have frontmatter, files are non-empty, files are valid markdown

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUTPUT_DIR="${ROOT_DIR}/website/docs/api/packages"

echo -e "${GREEN}Validating generated godoc output...${NC}"

# Track validation results
ERRORS=0
WARNINGS=0

# Expected files (based on script package list)
EXPECTED_MAIN_FILES=(
    "agents.md"
    "chatmodels.md"
    "config.md"
    "core.md"
    "embeddings.md"
    "llms.md"
    "memory.md"
    "monitoring.md"
    "orchestration.md"
    "prompts.md"
    "retrievers.md"
    "schema.md"
    "server.md"
    "vectorstores.md"
    "tools.md"
)

EXPECTED_LLM_FILES=(
    "llms/anthropic.md"
    "llms/bedrock.md"
    "llms/ollama.md"
    "llms/openai.md"
)

EXPECTED_VOICE_FILES=(
    "voice/stt.md"
    "voice/tts.md"
    "voice/vad.md"
    "voice/turndetection.md"
    "voice/transport.md"
    "voice/noise.md"
    "voice/session.md"
)

# Function to check if file exists
check_file_exists() {
    local file="$1"
    if [ ! -f "${OUTPUT_DIR}/${file}" ]; then
        echo -e "${RED}✗ Missing file: ${file}${NC}"
        ((ERRORS++))
        return 1
    else
        echo -e "${GREEN}✓ File exists: ${file}${NC}"
        return 0
    fi
}

# Function to check frontmatter
check_frontmatter() {
    local file="$1"
    local full_path="${OUTPUT_DIR}/${file}"
    
    if [ ! -f "$full_path" ]; then
        return 1
    fi
    
    if ! head -10 "$full_path" | grep -q "^---"; then
        echo -e "${RED}✗ Missing frontmatter start (---): ${file}${NC}"
        ((ERRORS++))
        return 1
    fi
    
    if ! head -10 "$full_path" | grep -q "title:"; then
        echo -e "${RED}✗ Missing title in frontmatter: ${file}${NC}"
        ((ERRORS++))
        return 1
    fi
    
    if ! head -10 "$full_path" | grep -q "sidebar_position:"; then
        echo -e "${YELLOW}⚠ Missing sidebar_position in frontmatter: ${file}${NC}"
        ((WARNINGS++))
    fi
    
    echo -e "${GREEN}✓ Frontmatter valid: ${file}${NC}"
    return 0
}

# Function to check file is non-empty
check_file_non_empty() {
    local file="$1"
    local full_path="${OUTPUT_DIR}/${file}"
    
    if [ ! -f "$full_path" ]; then
        return 1
    fi
    
    local line_count=$(wc -l < "$full_path" 2>/dev/null || echo "0")
    if [ "$line_count" -lt 10 ]; then
        echo -e "${YELLOW}⚠ File appears too short (< 10 lines): ${file}${NC}"
        ((WARNINGS++))
    fi
    
    if [ ! -s "$full_path" ]; then
        echo -e "${RED}✗ File is empty: ${file}${NC}"
        ((ERRORS++))
        return 1
    fi
    
    echo -e "${GREEN}✓ File is non-empty: ${file}${NC}"
    return 0
}

# Function to check markdown validity (basic checks)
check_markdown_validity() {
    local file="$1"
    local full_path="${OUTPUT_DIR}/${file}"
    
    if [ ! -f "$full_path" ]; then
        return 1
    fi
    
    # Check for common markdown issues
    if grep -q "<details>" "$full_path" 2>/dev/null; then
        echo -e "${YELLOW}⚠ File contains <details> tags (may cause MDX issues): ${file}${NC}"
        ((WARNINGS++))
    fi
    
    # Check for unclosed HTML tags (basic check)
    local open_tags=$(grep -o "<[^/>]*>" "$full_path" 2>/dev/null | grep -v "<!--" | wc -l || echo "0")
    local close_tags=$(grep -o "</[^>]*>" "$full_path" 2>/dev/null | wc -l || echo "0")
    
    # This is a rough check - not perfect but catches obvious issues
    if [ "$open_tags" -gt 0 ] && [ "$close_tags" -eq 0 ]; then
        echo -e "${YELLOW}⚠ Possible unclosed HTML tags in: ${file}${NC}"
        ((WARNINGS++))
    fi
    
    echo -e "${GREEN}✓ Markdown validity check passed: ${file}${NC}"
    return 0
}

# Validate main package files
echo -e "\n${GREEN}Validating main package files...${NC}"
for file in "${EXPECTED_MAIN_FILES[@]}"; do
    if check_file_exists "$file"; then
        check_frontmatter "$file"
        check_file_non_empty "$file"
        check_markdown_validity "$file"
    fi
done

# Validate LLM provider files
echo -e "\n${GREEN}Validating LLM provider files...${NC}"
for file in "${EXPECTED_LLM_FILES[@]}"; do
    if check_file_exists "$file"; then
        check_frontmatter "$file"
        check_file_non_empty "$file"
        check_markdown_validity "$file"
    fi
done

# Validate voice package files
echo -e "\n${GREEN}Validating voice package files...${NC}"
for file in "${EXPECTED_VOICE_FILES[@]}"; do
    if check_file_exists "$file"; then
        check_frontmatter "$file"
        check_file_non_empty "$file"
        check_markdown_validity "$file"
    fi
done

# Summary
echo -e "\n${GREEN}Validation Summary:${NC}"
echo -e "Errors: ${ERRORS}"
echo -e "Warnings: ${WARNINGS}"

if [ $ERRORS -eq 0 ] && [ $WARNINGS -eq 0 ]; then
    echo -e "${GREEN}✓ All validation checks passed!${NC}"
    exit 0
elif [ $ERRORS -eq 0 ]; then
    echo -e "${YELLOW}⚠ Validation passed with warnings${NC}"
    exit 0
else
    echo -e "${RED}✗ Validation failed with ${ERRORS} error(s)${NC}"
    exit 1
fi
