#!/usr/bin/env python3
"""Fix MDX code block issues in markdown files.

This script fixes common MDX compilation errors:
1. HTML entities in code (&#91; -> [, &#123; -> {, etc.)
2. JSX-like patterns outside code blocks (<, {, etc.)
3. Fragmented code blocks (merge split blocks)
"""

import re
import os
import sys

# HTML entities to decode
HTML_ENTITIES = {
    '&#91;': '[',
    '&#93;': ']',
    '&#123;': '{',
    '&#125;': '}',
    '&lt;': '<',
    '&gt;': '>',
    '&amp;': '&',
    '&quot;': '"',
    '&#39;': "'",
}


def split_frontmatter(content):
    """Split content into frontmatter and body.

    Returns (frontmatter, body) where frontmatter includes the --- markers.
    If no frontmatter, returns ('', content).
    """
    if not content.startswith('---'):
        return '', content

    # Find the closing ---
    rest = content[3:]
    end_idx = rest.find('\n---')
    if end_idx == -1:
        return '', content

    # Include the closing --- and newline
    end_idx = 3 + end_idx + 4  # '---' + position + '\n---'
    frontmatter = content[:end_idx]
    body = content[end_idx:]

    return frontmatter, body


def decode_html_entities(content):
    """Decode HTML entities back to original characters."""
    for entity, char in HTML_ENTITIES.items():
        content = content.replace(entity, char)
    # Decode numeric entities
    content = re.sub(r'&#(\d+);', lambda m: chr(int(m.group(1))), content)
    return content


def is_code_continuation(line, prev_lines):
    """Check if a line looks like a code continuation after a closing fence."""
    stripped = line.strip()
    if not stripped:
        return False

    # Strong indicators that this is orphaned code
    # Go code patterns
    if re.match(r'^\s*(//|/\*|\*/)', line):  # Comments
        return True
    if re.match(r'^\s*\}$', stripped):  # Closing brace only
        return True
    if re.match(r'^\s*(return|defer|go)\s', line):
        return True
    if re.match(r'^\s*[a-zA-Z_][a-zA-Z0-9_]*\s*:=', line):  # Go short decl
        return True
    if re.match(r'^\s*select\s*\{', line):
        return True

    # Indented code continuation (4+ spaces or tab)
    if re.match(r'^(\t|    )', line) and stripped and not stripped.startswith(('-', '*', '>')):
        return True

    return False


def is_markdown_text(line):
    """Check if a line is clearly markdown text (not code)."""
    stripped = line.strip()
    if not stripped:
        return True

    # Markdown indicators
    if stripped.startswith(('#', '>', '|', '-', '1.', '2.', '3.', '!', '[')):
        return True
    if stripped.startswith(('**', '__')):
        return True
    if re.match(r'^\d+\.\s+\*\*', stripped):  # Numbered list with bold
        return True

    # Long prose-like sentences (more than 10 words, no code chars)
    words = stripped.split()
    if len(words) > 10 and not any(c in stripped for c in ['{}', '()', ':=', '->', '{{', '}}']):
        return True

    return False


def fix_fragmented_blocks(content):
    """Fix fragmented code blocks where code continues after closing fence."""
    lines = content.split('\n')
    result = []
    i = 0
    in_code_block = False
    code_lang = None

    while i < len(lines):
        line = lines[i]
        stripped = line.strip()

        # Track code block state
        if stripped.startswith('```'):
            rest = stripped[3:].strip()

            if rest and rest[0].isalpha():
                # Opening fence with language
                in_code_block = True
                code_lang = rest.split()[0]
                result.append(line)
                i += 1
                continue
            elif rest == '' and in_code_block:
                # Closing fence - check for orphaned code immediately after
                # Look ahead for orphaned code
                orphan_code = []
                j = i + 1

                # Check for immediate code continuation (after blank lines)
                blank_count = 0
                while j < len(lines) and not lines[j].strip():
                    blank_count += 1
                    j += 1

                # Only look for orphan if there are 0-1 blank lines
                if blank_count <= 1 and j < len(lines):
                    # Check if next non-blank line looks like code continuation
                    while j < len(lines):
                        next_line = lines[j]
                        next_stripped = next_line.strip()

                        # Stop at markdown headings or new code blocks
                        if next_stripped.startswith('#') or next_stripped.startswith('```'):
                            break

                        # Stop at clear markdown text
                        if is_markdown_text(next_line):
                            break

                        # This looks like orphaned code
                        if is_code_continuation(next_line, orphan_code):
                            orphan_code.append(next_line)
                            j += 1
                        elif orphan_code and next_stripped:
                            # Continue collecting if we already have some orphans
                            orphan_code.append(next_line)
                            j += 1
                        else:
                            break

                if orphan_code:
                    # Don't close the code block yet, append orphaned code
                    for orphan in orphan_code:
                        result.append(orphan)
                    # Skip the orphaned lines
                    i = j
                    # Now close the block
                    result.append('```')
                    in_code_block = False
                    code_lang = None
                    continue
                else:
                    result.append(line)
                    in_code_block = False
                    code_lang = None
                    i += 1
                    continue
            else:
                # Opening fence without language (toggle)
                if in_code_block:
                    in_code_block = False
                    code_lang = None
                else:
                    in_code_block = True
                result.append(line)
                i += 1
                continue

        result.append(line)
        i += 1

    return '\n'.join(result)


def escape_jsx_patterns(content):
    """Escape patterns that MDX interprets as JSX."""
    lines = content.split('\n')
    result = []
    in_code_block = False

    for line in lines:
        stripped = line.strip()

        # Track code blocks
        if stripped.startswith('```'):
            rest = stripped[3:].strip()
            if rest and rest[0].isalpha():
                in_code_block = True
            elif rest == '':
                in_code_block = not in_code_block
            result.append(line)
            continue

        if in_code_block:
            result.append(line)
            continue

        # Outside code blocks - escape problematic patterns

        # Normalize double escapes to single escapes first
        line = line.replace('\\\\{', '\\{')
        line = line.replace('\\\\}', '\\}')
        line = line.replace('\\\\<', '\\<')

        # Escape unescaped < followed by number (like <50ms)
        line = re.sub(r'(?<!\\)<(\d)', r'\\<\1', line)

        # Escape unescaped <-chan pattern (Go channels)
        line = re.sub(r'(?<!\\)<-chan', r'\\<-chan', line)

        # Escape unescaped <- pattern (Go receive)
        line = re.sub(r'(?<!\\)<-(?![a-zA-Z])', r'\\<-', line)

        # Escape {} empty braces (but not in inline code)
        if '`' not in line:
            line = re.sub(r'(?<!\\)\{\}', r'\\{\\}', line)

        # Escape {word} patterns in mermaid-like lines (outside code blocks)
        if '-->' in line:
            line = re.sub(r'(?<!\\)\{([^{}]+)\}', r'\\{\1\\}', line)

        # Escape {{- and -}} Helm patterns (only if truly outside code)
        if '`' not in line:
            line = re.sub(r'(?<!\\)\{\{-?', r'\\{\\{', line)
            line = re.sub(r'-?(?<!\\)\}\}', r'\\}\\}', line)

        result.append(line)

    return '\n'.join(result)


def remove_empty_code_blocks(content):
    """Remove empty code blocks."""
    content = re.sub(r'```\n(\s*\n)*```', '', content)
    content = re.sub(r'```\n```', '', content)
    return content


def fix_markdown_file(filepath):
    """Fix MDX issues in a markdown file."""
    with open(filepath, 'r', encoding='utf-8') as f:
        content = f.read()

    original = content

    # Split out frontmatter - don't process it
    frontmatter, body = split_frontmatter(content)

    # Step 1: Decode HTML entities (body only)
    body = decode_html_entities(body)

    # Step 2: Fix fragmented code blocks
    body = fix_fragmented_blocks(body)

    # Step 3: Remove empty code blocks
    body = remove_empty_code_blocks(body)

    # Step 4: Escape JSX patterns (final pass)
    body = escape_jsx_patterns(body)

    # Recombine
    content = frontmatter + body

    if content != original:
        with open(filepath, 'w', encoding='utf-8') as f:
            f.write(content)
        return True
    return False


def process_path(path):
    """Process a file or directory."""
    fixed = 0

    if os.path.isfile(path):
        if path.endswith('.md'):
            if fix_markdown_file(path):
                print(f"Fixed: {path}")
                fixed = 1
    elif os.path.isdir(path):
        for root, dirs, files in os.walk(path):
            dirs[:] = [d for d in dirs if d not in ['node_modules', '.git', 'vendor', 'build']]
            for f in files:
                if f.endswith('.md'):
                    filepath = os.path.join(root, f)
                    if fix_markdown_file(filepath):
                        print(f"Fixed: {filepath}")
                        fixed += 1
    else:
        print(f"Path not found: {path}")
        sys.exit(1)

    return fixed


if __name__ == '__main__':
    if len(sys.argv) < 2:
        print("Usage: fix_mdx.py <file_or_directory>")
        sys.exit(1)

    total = 0
    for path in sys.argv[1:]:
        total += process_path(path)
    print(f"\nTotal files fixed: {total}")
