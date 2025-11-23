# Data Model: Fix Corrupted Mock Files

## Overview
This fix involves correcting Go source files, not data entities. The "data model" here represents the file structure and validation rules.

## File Entities

### MockFile
Represents a Go mock file that must conform to Go syntax rules.

**Attributes**:
- `path`: string - Relative path from repository root
- `package`: string - Package name (must match directory structure)
- `hasPackageDeclaration`: boolean - Whether file starts with `package` declaration
- `content`: string - File contents

**Validation Rules**:
- MUST start with `package <name>` declaration
- Package name MUST match directory structure:
  - `pkg/core/*.go` → `package core`
  - `pkg/prompts/*.go` → `package prompts`
  - `pkg/memory/*.go` → `package memory`
  - `pkg/vectorstores/*.go` → `package vectorstores`
  - `pkg/vectorstores/iface/*.go` → `package vectorstores`

**State Transitions**:
- `corrupted` (missing package) → `fixed` (has package declaration)

## Affected Files

| Path | Current State | Target Package | Status |
|------|---------------|----------------|--------|
| `pkg/core/di_mock.go` | corrupted | `core` | To fix |
| `pkg/prompts/advanced_mock.go` | corrupted | `prompts` | To fix |
| `pkg/memory/advanced_mock.go` | corrupted | `memory` | To fix |
| `pkg/vectorstores/advanced_mock.go` | corrupted | `vectorstores` | To fix |
| `pkg/vectorstores/iface/iface_mock.go` | corrupted | `vectorstores` | To fix |

## Relationships
- Each file belongs to exactly one package
- Package name is determined by directory structure
- Files in subdirectories (like `iface/`) inherit parent package name unless explicitly declared otherwise

## Constraints
- All `.go` files MUST start with `package` declaration (Go language requirement)
- Package name MUST be valid Go identifier
- No breaking changes to existing API
- Files MUST compile with `go build`

