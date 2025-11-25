# Contract: Changelog Generation and Release Notes

**Feature**: 005-adjust-the-workflows  
**Date**: 2025-01-27  
**Contract ID**: C008

## Purpose
Ensure changelog is generated before releases and included in release notes, following common Go library patterns (changie or git-based).

## Input
- Release trigger (tag, workflow_dispatch, or automated)
- Changelog source (changie files, git commits, or manual)

## Validation Rules

### Rule 1: Changelog Must Be Generated Before Release
- Changelog generation MUST run before GoReleaser executes
- Changelog MUST be generated from changie files (if `.changie.yaml` exists) or git commits
- Changelog generation failure MUST mark release as incomplete but allow release to continue
- Generated changelog MUST be validated before use

### Rule 2: Changelog Integration with GoReleaser
- GoReleaser MUST use generated changelog for release notes
- Changelog MUST be included in GitHub release description
- Changelog format MUST be compatible with GoReleaser expectations
- Changelog MUST follow semantic versioning format

### Rule 3: Changelog Source Priority
- If `.changie.yaml` exists, use changie CLI for changelog generation
- If changie not configured, use git-based changelog (git log, conventional commits)
- Manual changelog files MUST be supported as fallback
- Changelog generation MUST be optional (can skip if not configured)

### Rule 4: Changelog Content Requirements
- Changelog MUST include version number
- Changelog MUST include release date
- Changelog MUST categorize changes (Added, Changed, Deprecated, Removed, Fixed, Security)
- Changelog MUST be formatted as Markdown

## Success Criteria
- Changelog is generated before release artifacts are created
- Changelog is included in GitHub release notes
- Changelog generation supports multiple sources (changie, git, manual)
- Changelog generation failure doesn't block release (marks as incomplete)
- Changelog format is consistent and readable

## Failure Modes
- Changelog generation missing → Error: "Changelog generation must run before release"
- Changelog not included in release → Error: "Changelog must be included in release notes"
- Changelog generation blocks release → Error: "Changelog generation failure must not block release"
- Changelog format invalid → Warning: "Changelog format may be invalid"

## Test Validation
```bash
# Validate changelog generation step exists
grep -q "changie\|changelog\|CHANGELOG" .github/workflows/release.yml || echo "ERROR: Changelog generation not found"

# Validate changelog runs before GoReleaser
# Check job order: changelog job must run before release job

# Test changelog generation locally
changie batch || echo "WARNING: Changie not configured, using git-based changelog"

# Validate changelog in release notes
# Check GoReleaser configuration includes changelog
grep -q "changelog:" .goreleaser.yml || echo "WARNING: GoReleaser changelog not configured"
```

## Implementation Notes
- Add changelog generation step before GoReleaser in release workflow
- Support changie: `changie batch` → `changie merge` → Include in release
- Support git-based: `git log --pretty=format:"- %s" vX.Y.Z..HEAD` → Format as changelog
- Pass changelog to GoReleaser via configuration or file
- Make changelog generation optional via workflow input or configuration check
- Follow patterns from changie project: file-based changelog management

