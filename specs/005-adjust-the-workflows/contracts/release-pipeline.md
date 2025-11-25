# Contract: Release Pipeline

**Feature**: 005-adjust-the-workflows  
**Date**: 2025-01-27  
**Contract ID**: C005

## Purpose
Ensure release pipeline uses GoReleaser to generate releases, generates godocs, updates website, tags, and publishes releases.

## Input
- Release trigger: tag push, workflow_dispatch, or automated (release-please)
- Version tag (e.g., "v1.0.0")

## Validation Rules

### Rule 1: GoReleaser Must Generate Release
- GoReleaser MUST run as part of release workflow
- GoReleaser MUST use `.goreleaser.yml` configuration
- Release artifacts MUST be generated (archives, checksums)
- Release MUST be created on GitHub

### Rule 2: API Documentation Must Be Generated
- `scripts/generate-docs.sh` MUST run during release
- API documentation MUST be generated using gomarkdoc
- Documentation MUST be generated before website update
- Documentation generation failure MUST fail the release

### Rule 3: Website Must Be Updated
- Docusaurus website MUST be built with generated docs
- Website MUST be deployed to GitHub Pages
- Website update MUST happen after documentation generation
- Website deployment failure MUST fail the release

### Rule 4: Release Must Be Tagged and Published
- Release MUST be tagged with version (handled by GoReleaser)
- Release MUST be published to GitHub repository
- Release notes MUST be included (from CHANGELOG or git)
- Release MUST be marked as published (not draft)

### Rule 5: Release Pipeline Must Support Multiple Triggers
- Automated releases via release-please MUST be supported
- Manual releases via workflow_dispatch MUST be supported
- Tag-based releases MUST be supported
- Only one release process MUST run at a time (concurrency control)

## Success Criteria
- GoReleaser successfully creates release artifacts
- API documentation is generated and included
- Website is updated with new documentation
- Release is tagged and published on GitHub
- Release notes are included
- Release process supports all trigger types

## Failure Modes
- GoReleaser not configured → Error: "GoReleaser must be used for releases"
- Documentation generation missing → Error: "API documentation must be generated"
- Website update missing → Error: "Website must be updated during release"
- Release not published → Error: "Release must be published"
- Concurrent releases → Error: "Concurrency control must prevent simultaneous releases"

## Test Validation
```bash
# Validate GoReleaser usage
grep -q "goreleaser" .github/workflows/release.yml || echo "ERROR: GoReleaser not found"

# Validate documentation generation
grep -q "generate-docs\|docs-generate\|gomarkdoc" .github/workflows/release.yml || echo "ERROR: Documentation generation not found"

# Validate website update
grep -q "docusaurus\|website\|github-pages" .github/workflows/release.yml || echo "ERROR: Website update not found"

# Validate concurrency control
grep -q "concurrency:" .github/workflows/release.yml || echo "WARNING: Concurrency control not found"

# Validate multiple triggers
grep -q "workflow_dispatch" .github/workflows/release.yml || echo "ERROR: Manual trigger not found"
grep -q "tags:" .github/workflows/release.yml || echo "ERROR: Tag trigger not found"
```

## Implementation Notes
- Release workflow sequence:
  1. Pre-release checks (tests, build)
  2. Run GoReleaser (`goreleaser/goreleaser-action@v6`)
  3. Generate API docs (`make docs-generate` or `scripts/generate-docs.sh`)
  4. Build Docusaurus website (`yarn build` in website/)
  5. Deploy website (GitHub Pages or trigger website_deploy workflow)
  6. Tag and publish (handled by GoReleaser)
- Use concurrency group to prevent simultaneous releases
- Support both automated (release-please) and manual releases
- Version validation: `^v[0-9]+\.[0-9]+\.[0-9]+$`

