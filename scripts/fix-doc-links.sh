#!/bin/bash
# Script to fix internal links in migrated documentation files

DOCS_DIR="website/docs"

# Fix common link patterns
find "$DOCS_DIR" -name "*.md" -type f | while read file; do
    # Fix links to architecture.md
    sed -i 's|\.\/architecture\.md|../guides/architecture|g' "$file"
    sed -i 's|\.\.\/architecture\.md|../guides/architecture|g' "$file"
    
    # Fix links to best-practices
    sed -i 's|\.\/BEST_PRACTICES\.md|../guides/best-practices|g' "$file"
    sed -i 's|\.\/best-practices\.md|../guides/best-practices|g' "$file"
    
    # Fix links to troubleshooting
    sed -i 's|\.\/TROUBLESHOOTING\.md|../guides/troubleshooting|g' "$file"
    sed -i 's|\.\.\/TROUBLESHOOTING\.md|../guides/troubleshooting|g' "$file"
    
    # Fix links to migration
    sed -i 's|\.\/MIGRATION\.md|../guides/migration|g' "$file"
    
    # Fix links to package design patterns
    sed -i 's|\.\/package_design_patterns\.md|../guides/package-design-patterns|g' "$file"
    
    # Fix links to QUICKSTART.md
    sed -i 's|\.\/QUICKSTART\.md|./quickstart|g' "$file"
    sed -i 's|\.\.\/QUICKSTART\.md|../getting-started/quickstart|g' "$file"
    
    # Fix links to INSTALLATION.md
    sed -i 's|\.\/INSTALLATION\.md|./installation|g' "$file"
    sed -i 's|\.\.\/INSTALLATION\.md|../getting-started/installation|g' "$file"
    
    # Fix links to use-cases
    sed -i 's|\.\/use-cases\/|../use-cases/|g' "$file"
    
    # Fix links to getting-started
    sed -i 's|\.\/getting-started\/|../getting-started/|g' "$file"
    sed -i 's|getting-started\/|../getting-started/|g' "$file"
    
    # Fix links to concepts
    sed -i 's|\.\/concepts\/|../concepts/|g' "$file"
    
    # Fix links to providers
    sed -i 's|\.\/providers\/|../providers/|g' "$file"
    
    # Fix links to cookbook
    sed -i 's|\.\/cookbook\/|../cookbook/|g' "$file"
    
    # Fix links to FRAMEWORK_COMPARISON.md
    sed -i 's|\.\/FRAMEWORK_COMPARISON\.md|../reference/framework-comparison|g' "$file"
    
    # Fix links to API docs (remove .md extension for Docusaurus)
    sed -i 's|\.md)|)|g' "$file"
    sed -i 's|\.md#|#|g' "$file"
    
    # Fix relative paths that go up too many levels from getting-started
    if [[ "$file" == *"getting-started"* ]]; then
        sed -i 's|\.\.\/\.\.\/guides|../guides|g' "$file"
        sed -i 's|\.\.\/\.\.\/use-cases|../use-cases|g' "$file"
    fi
    
    # Fix links to CONTRIBUTING.md (external, keep as is but fix path if needed)
    sed -i 's|\.\.\/CONTRIBUTING\.md|https://github.com/lookatitude/beluga-ai/blob/main/CONTRIBUTING.md|g' "$file"
    
    # Fix links to README.md in root
    sed -i 's|\.\.\/README\.md|https://github.com/lookatitude/beluga-ai/blob/main/README.md|g' "$file"
done

echo "Link fixing complete!"

