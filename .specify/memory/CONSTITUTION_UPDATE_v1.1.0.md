# Constitution Update v1.1.0 - Task Execution Clarification

**Date**: 2025-10-05  
**Version**: 1.0.0 → 1.1.0  
**Type**: MINOR (new principles added, material expansions)

## Problem Addressed

The previous constitution version (1.0.0) did not clearly distinguish between:
1. **Analysis tasks** that document existing code (write to `specs/` only)
2. **Implementation tasks** that create/modify actual code (write to `pkg/`, `cmd/`, `internal/`)

This led to confusion where analysis specs (like `008-for-the-embeddings`) generated tasks that only wrote documentation files instead of implementing actual fixes to the codebase.

## Key Changes

### 1. Added Task Type Classification System

Three distinct task types are now defined:

#### **NEW FEATURE IMPLEMENTATION** (`specs/NNN-feature-name/`)
- **Goal**: Create NEW code in `pkg/`, `cmd/`, `internal/` directories
- **File Targets**: Actual codebase files (`.go` files)
- **Task Verbs**: Create, Implement, Add, Build, Write
- **Example**: "Create Embedder interface in `pkg/embeddings/iface/embedder.go`"

#### **ANALYSIS/AUDIT** (`specs/NNN-for-the-{package}/`)
- **Goal**: Document findings about EXISTING code WITHOUT modifying it
- **File Targets**: `specs/` directory ONLY (`.md` files)
- **Task Verbs**: Verify, Analyze, Validate, Document, Review, Audit
- **Example**: "Analyze error handling in `specs/008-for-the-embeddings/findings/error-handling.md`"

#### **CORRECTION/ENHANCEMENT** (`specs/NNN-fix-{package}-{issue}/`)
- **Goal**: Fix/improve EXISTING code in `pkg/` based on analysis findings
- **File Targets**: Actual codebase files in `pkg/` directory
- **Task Verbs**: Fix, Update, Enhance, Refactor, Improve, Correct
- **Example**: "Fix error wrapping in `pkg/embeddings/providers/openai.go`"

### 2. Added Implementation Workflow Steps

Clear workflows for each task type:

**NEW FEATURES**: `/specify` → `/plan` → `/tasks` → Execute (Setup → Tests → Core → Integration → Polish) → Test → Commit → PR

**ANALYSIS**: `/specify` → `/plan` → `/tasks` → Execute (Setup → Verify → Analyze → Validate → Report) → Review → Create correction spec if needed

**CORRECTIONS**: Start from analysis → Create new spec → Generate correction tasks → Execute (Setup → Tests → Fixes → Verify → Document) → Test → Commit → PR

### 3. Added Critical Rules for Task Definitions

1. **EXPLICIT FILE PATHS**: Every task MUST specify exact file paths
   - ✅ "Implement Embedder interface in `pkg/embeddings/iface/embedder.go`"
   - ❌ "Implement embedder interface" (no path)

2. **CLEAR ACTION VERBS**: Use precise verbs indicating file operations
   - NEW features: Create, Implement, Add, Build, Write
   - ANALYSIS: Verify, Analyze, Validate, Document, Review
   - CORRECTIONS: Fix, Update, Enhance, Refactor, Improve

3. **PACKAGE COMPLIANCE**: All tasks must reference package design patterns (ISP, DIP, SRP, OTEL metrics, error handling)

### 4. Enhanced Post-Implementation Workflow

Added verification checklist:
- ✅ All files in `pkg/` directories created/modified
- ✅ Tests passing: `go test ./pkg/{package}/... -v -cover`
- ✅ Linter clean: `golangci-lint run ./pkg/{package}/...`
- ✅ Documentation updated

## Files Updated

### 1. `.specify/memory/constitution.md`
- **Version**: 1.0.0 → 1.1.0
- **Added**: Complete "Task Execution Requirements" section with task type classification
- **Added**: Implementation Workflow Steps for each task type
- **Added**: Critical Rules for Task Definitions
- **Enhanced**: Post-Implementation Workflow with verification checklist

### 2. `.specify/templates/tasks-template.md`
- **Added**: "Task Type Classification (CRITICAL - Read First!)" section at top
- **Updated**: Execution Flow to include task type identification as step 1
- **Added**: "Path Conventions by Task Type" section with examples
- **Added**: "Quick Task Type Reference" table at bottom
- **Updated**: Constitution version reference to v1.1.0

### 3. `.cursor/commands/implement.md`
- **Added**: Task type identification as critical step
- **Added**: File target verification based on task type
- **Added**: Type-specific execution rules for NEW FEATURES, ANALYSIS, and CORRECTIONS
- **Enhanced**: Validation requirements per task type

## Impact on Existing Specs

### Spec 008 (Embeddings Analysis) - ✅ CORRECT AS-IS
This spec is correctly classified as **ANALYSIS**:
- All tasks write to `specs/008-for-the-embeddings/` (findings, analysis, validation, reports)
- NO tasks modify `pkg/embeddings/` files (correct for analysis type)
- Phase names match analysis workflow (Verification, Analysis, Validation, Reporting)

**Action**: No changes needed. This spec follows the new classification correctly.

### Future Corrections Needed
If issues were found in the analysis, create a NEW spec:
- **Name**: `specs/009-fix-embeddings-error-handling/` (or similar)
- **Type**: CORRECTION
- **Tasks**: All tasks modify files in `pkg/embeddings/`
- **Workflow**: Setup → Tests → Fixes → Verification → Documentation

## Benefits

1. **Clear Separation of Concerns**: Analysis documents, implementation modifies code
2. **Explicit File Targets**: Every task knows exactly what files to create/modify
3. **Proper Workflows**: Different processes for analysis vs implementation vs corrections
4. **Better Accountability**: File paths required, making tasks trackable and verifiable
5. **Reduced Confusion**: AI agents now know when to document vs when to implement

## Migration Guide

For existing specs:

1. **Identify current type**:
   - Name contains "for-the-{package}"? → ANALYSIS
   - Name is just feature name? → NEW FEATURE
   - Name contains "fix-{package}"? → CORRECTION

2. **Verify task targets**:
   - ANALYSIS: All tasks should write to `specs/`
   - NEW FEATURE/CORRECTION: All tasks should write to `pkg/`, `cmd/`, `internal/`

3. **Update if mismatched**:
   - If analysis spec has tasks modifying `pkg/` → Create separate correction spec
   - If implementation spec has tasks writing to `specs/` → Move to correct type

## Constitutional Compliance

This update maintains all core principles:
- ✅ ISP: Small, focused interfaces (unchanged)
- ✅ DIP: Dependency injection (unchanged)
- ✅ SRP: Single responsibility (enhanced with task type focus)
- ✅ Composition: Functional options (unchanged)

Added clarity aligns with package design patterns in `/docs/package_design_patterns.md`:
- Clear structure requirements
- Explicit file paths matching package conventions
- Testing requirements per task type
- Documentation standards

## Next Steps

1. ✅ Constitution updated to v1.1.0
2. ✅ Tasks template updated with task type classification
3. ✅ Implement command updated with type-specific execution rules
4. ⏳ Review all existing specs for proper task type classification
5. ⏳ Create correction specs for any analysis findings that need fixes

## Conclusion

Constitution v1.1.0 provides **crystal-clear guidance** on when to:
- **Document** (analysis tasks → `specs/` files)
- **Implement** (feature tasks → `pkg/` files)
- **Fix** (correction tasks → `pkg/` files)

This eliminates confusion and ensures consistent, high-quality implementation across the Beluga AI Framework.

---

**Ratified**: 2025-10-05  
**Effective**: Immediately  
**Review**: Next constitutional audit

