#!/usr/bin/env python3
"""Fix broken documentation links in Beluga AI docs."""
import os
import re
from pathlib import Path

ROOT_DIR = Path(__file__).parent.parent
DOCS_DIR = ROOT_DIR / "docs"

# Counter for changes
changes_made = 0
files_modified = set()


def fix_file(file_path: Path) -> int:
    """Fix all broken links in a single file."""
    global changes_made, files_modified

    try:
        content = file_path.read_text(encoding='utf-8')
    except Exception as e:
        print(f"  Error reading {file_path}: {e}")
        return 0

    original = content
    local_changes = 0

    # 1. Fix API path references
    # /api/packages/ -> /api-docs/packages/
    if '/api/packages/' in content:
        content = content.replace('/api/packages/', '/api-docs/packages/')
        local_changes += 1

    # 2. Fix use-case filename references
    # use-cases/batch-processing.md -> use-cases/11-batch-processing.md
    if 'use-cases/batch-processing.md' in content:
        content = content.replace('use-cases/batch-processing.md', 'use-cases/11-batch-processing.md')
        local_changes += 1

    # 3. Convert external /pkg/ links to GitHub URLs
    pkg_pattern = r'\]\(/pkg/([^)]+)\)'
    if re.search(pkg_pattern, content):
        content = re.sub(pkg_pattern, r'](https://github.com/lookatitude/beluga-ai/blob/main/pkg/\1)', content)
        local_changes += 1

    # 4. Convert external /examples/ links to GitHub URLs
    examples_pattern = r'\]\(/examples/([^)]+)\)'
    if re.search(examples_pattern, content):
        content = re.sub(examples_pattern, r'](https://github.com/lookatitude/beluga-ai/blob/main/examples/\1)', content)
        local_changes += 1

    # 5. Convert external /specs/ links to GitHub URLs
    specs_pattern = r'\]\(/specs/([^)]+)\)'
    if re.search(specs_pattern, content):
        content = re.sub(specs_pattern, r'](https://github.com/lookatitude/beluga-ai/blob/main/specs/\1)', content)
        local_changes += 1

    # 6. Fix links to missing guides/config-providers.md
    # Replace with link to implementing-providers guide
    config_providers_pattern = r'\[([^\]]+)\]\([^)]*guides/config-providers\.md[^)]*\)'
    if re.search(config_providers_pattern, content):
        content = re.sub(config_providers_pattern, r'[\1](../guides/implementing-providers.md)', content)
        local_changes += 1

    # 7. Fix links to non-existent use-cases
    # ./event-driven-agents.md - convert to valid reference
    event_driven_pattern = r'\[([^\]]+)\]\(\./event-driven-agents\.md\)'
    if re.search(event_driven_pattern, content):
        content = re.sub(event_driven_pattern, r'[\1](./09-multi-model-llm-gateway.md)', content)
        local_changes += 1

    # ./distributed-orchestration.md - convert to valid reference
    distributed_pattern = r'\[([^\]]+)\]\(\./distributed-orchestration\.md\)'
    if re.search(distributed_pattern, content):
        content = re.sub(distributed_pattern, r'[\1](./07-distributed-workflow-orchestration.md)', content)
        local_changes += 1

    # 8. Remove ./README.md links (keep the text)
    # These integration docs link to README.md that doesn't exist
    readme_pattern = r'\[([^\]]+)\]\(\./README\.md\)'
    if re.search(readme_pattern, content):
        content = re.sub(readme_pattern, r'\1', content)
        local_changes += 1

    # 9. Fix ../guides/configuration.md links (file doesn't exist)
    guides_config_pattern = r'\[([^\]]+)\]\(\.\./guides/configuration\.md\)'
    if re.search(guides_config_pattern, content):
        content = re.sub(guides_config_pattern, r'[\1](../guides/implementing-providers.md)', content)
        local_changes += 1

    # 10. Fix voice/s2s.md links - point to session.md instead
    voice_s2s_pattern = r'\]\([^)]*api-docs/packages/voice/s2s\.md[^)]*\)'
    if re.search(voice_s2s_pattern, content):
        content = re.sub(voice_s2s_pattern, r'](../../api-docs/packages/voice/session.md)', content)
        local_changes += 1

    # 11. Fix missing API docs references - remove links, keep text
    missing_api_docs = [
        'messaging', 'documentloaders', 'safety', 'multimodal', 'textsplitters'
    ]
    for pkg in missing_api_docs:
        pattern = rf'\[([^\]]+)\]\([^)]*api-docs/packages/{pkg}\.md[^)]*\)'
        if re.search(pattern, content):
            content = re.sub(pattern, r'\1', content)
            local_changes += 1

    # 12. Fix getting-started tutorial paths
    # ../getting-started/tutorials/working-with-tools.md -> ../getting-started/04-working-with-tools.md
    tutorial_pattern = r'\]\(\.\./getting-started/tutorials/working-with-tools\.md\)'
    if re.search(tutorial_pattern, content):
        content = re.sub(tutorial_pattern, r'](../getting-started/04-working-with-tools.md)', content)
        local_changes += 1

    # 13. Fix /docs/providers/embeddings/selection.md - file doesn't exist
    providers_pattern = r'\[([^\]]+)\]\(/docs/providers/embeddings/selection\.md\)'
    if re.search(providers_pattern, content):
        content = re.sub(providers_pattern, r'[\1](../guides/rag-multimodal.md)', content)
        local_changes += 1

    # 14. Fix ../../../tutorials paths in integrations
    tutorials_pattern = r'\]\(\.\./\.\./\.\./tutorials/([^)]+)\)'
    if re.search(tutorials_pattern, content):
        content = re.sub(tutorials_pattern, r'](../../tutorials/\1)', content)
        local_changes += 1

    # 15. Fix ../../../api-docs/packages/server.md paths
    server_pattern = r'\]\(\.\./\.\./\.\./api-docs/packages/server\.md\)'
    if re.search(server_pattern, content):
        content = re.sub(server_pattern, r'](../../api-docs/packages/server.md)', content)
        local_changes += 1

    # 16. Fix ../../api-docs/packages links in integrations (need one more ../)
    rel_path = file_path.relative_to(DOCS_DIR)
    depth = len(rel_path.parts) - 1  # -1 for the file itself

    # For integration docs at depth 3 (integrations/xxx/file.md), fix paths
    if depth >= 2 and 'integrations' in str(rel_path):
        if depth == 3:
            if '../../api-docs' in content and '../../../api-docs' not in content:
                content = content.replace('../../api-docs', '../../../api-docs')
                local_changes += 1

    # 17. Convert ../README.md, ../CONTRIBUTING.md, ../CHANGELOG.md to GitHub URLs
    for doc in ['README', 'CONTRIBUTING', 'CHANGELOG']:
        pattern = rf'\]\(\.\./({doc})\.md\)'
        if re.search(pattern, content):
            content = re.sub(pattern, rf'](https://github.com/lookatitude/beluga-ai/blob/main/{doc}.md)', content)
            local_changes += 1

    # 18. Convert ../../CONTRIBUTING.md etc to GitHub URLs
    for doc in ['README', 'CONTRIBUTING', 'CHANGELOG']:
        pattern = rf'\]\(\.\./\.\./{doc}\.md\)'
        if re.search(pattern, content):
            content = re.sub(pattern, rf'](https://github.com/lookatitude/beluga-ai/blob/main/{doc}.md)', content)
            local_changes += 1

    # 19. Convert ../examples/* links to GitHub URLs
    pattern = r'\]\(\.\./examples/([^)]*)\)'
    if re.search(pattern, content):
        content = re.sub(pattern, r'](https://github.com/lookatitude/beluga-ai/tree/main/examples/\1)', content)
        local_changes += 1

    # 20. Convert ../../examples/* links to GitHub URLs
    pattern = r'\]\(\.\./\.\./examples/([^)]*)\)'
    if re.search(pattern, content):
        content = re.sub(pattern, r'](https://github.com/lookatitude/beluga-ai/tree/main/examples/\1)', content)
        local_changes += 1

    # 21. Convert ../../pkg/* links to GitHub URLs
    pattern = r'\]\(\.\./\.\./pkg/([^)]*)\)'
    if re.search(pattern, content):
        content = re.sub(pattern, r'](https://github.com/lookatitude/beluga-ai/tree/main/pkg/\1)', content)
        local_changes += 1

    # 22. Convert ../../../pkg/* links to GitHub URLs
    pattern = r'\]\(\.\./\.\./\.\./pkg/([^)]*)\)'
    if re.search(pattern, content):
        content = re.sub(pattern, r'](https://github.com/lookatitude/beluga-ai/tree/main/pkg/\1)', content)
        local_changes += 1

    # 23. Fix ../website/docs/api/ links
    pattern = r'\]\([^)]*website/docs/api/[^)]*\)'
    if re.search(pattern, content):
        content = re.sub(pattern, r'](./api-reference.md)', content)
        local_changes += 1

    # 24. Fix ./patterns/ directory links
    pattern = r'\]\(\./patterns/\)'
    if re.search(pattern, content):
        content = re.sub(pattern, r'](./patterns/README.md)', content)
        local_changes += 1

    # 25. Fix ./guides/ directory links
    pattern = r'\]\(\./guides/\)'
    if re.search(pattern, content):
        content = re.sub(pattern, r'](./guides/llm-providers.md)', content)
        local_changes += 1

    # 26. Fix ./providers/xxx/ directory links to GitHub
    pattern = r'\]\(\./providers/([^)]+)/\)'
    if re.search(pattern, content):
        content = re.sub(pattern, r'](https://github.com/lookatitude/beluga-ai/tree/main/pkg/\1)', content)
        local_changes += 1

    # 27. Fix broken ./TEST_ISSUES_SUMMARY.md link
    pattern = r'\[([^\]]+)\]\(\./TEST_ISSUES_SUMMARY\.md\)'
    if re.search(pattern, content):
        content = re.sub(pattern, r'\1', content)
        local_changes += 1

    # 28. Fix ../../integrations links with wrong path depth
    pattern = r'\]\(\.\./\.\./integrations/([^)]+)\)'
    if re.search(pattern, content):
        content = re.sub(pattern, r'](../integrations/\1)', content)
        local_changes += 1

    # 29. Fix ../voice-providers.md in examples (wrong path)
    pattern = r'\]\(\.\./voice-providers\.md\)'
    if re.search(pattern, content):
        content = re.sub(pattern, r'](../../guides/voice-providers.md)', content)
        local_changes += 1

    # 30. Fix links in examples directory that have wrong depth
    if 'examples' in str(rel_path):
        # ../../guides/ from examples/voice/ should be ../../guides/
        pattern = r'\]\(\.\./\.\./guides/([^)]+)\)'
        if re.search(pattern, content):
            # Check actual depth
            if depth >= 2:  # examples/xxx/file.md
                # Already correct, but let's ensure it stays
                pass

        # ../../use-cases/ - fix wrong filename references
        if 'voice-sessions.md' in content:
            content = content.replace('use-cases/voice-sessions.md', 'use-cases/voice-sessions.md')
            # Actually check if this file exists
            # For now, point to existing voice use case
            pattern = r'\]\([^)]*use-cases/voice-sessions\.md\)'
            if re.search(pattern, content):
                content = re.sub(pattern, r'](../../use-cases/voice-session-multi-turn-forms.md)', content)
                local_changes += 1

    # 31. Fix ../../../examples/* links to GitHub URLs
    pattern = r'\]\(\.\./\.\./\.\./examples/([^)]*)\)'
    if re.search(pattern, content):
        content = re.sub(pattern, r'](https://github.com/lookatitude/beluga-ai/tree/main/examples/\1)', content)
        local_changes += 1

    # Write back if changed
    if content != original:
        file_path.write_text(content, encoding='utf-8')
        files_modified.add(str(file_path))
        changes_made += local_changes
        print(f"  Fixed: {file_path.relative_to(ROOT_DIR)} ({local_changes} patterns)")
        return local_changes

    return 0


def main():
    """Fix all broken links in docs."""
    print("Fixing broken documentation links (pass 2)...")
    print(f"Scanning: {DOCS_DIR}")
    print()

    # Find all markdown files
    md_files = list(DOCS_DIR.rglob("*.md"))
    print(f"Found {len(md_files)} markdown files")
    print()

    # Process each file
    for md_file in sorted(md_files):
        fix_file(md_file)

    print()
    print(f"Done! Modified {len(files_modified)} files with {changes_made} pattern fixes.")


if __name__ == "__main__":
    main()
