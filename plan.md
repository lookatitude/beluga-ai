# Beluga AI Project Cleanup Plan

This plan outlines the tasks to clean up the Beluga AI project, aligning it with the refactored architecture described in `Beluga_Refactored_Architecture.md`. Tasks are grouped by category, prioritized (High, Medium, Low), and include detailed steps. We'll execute them iteratively, using Git for versioning (e.g., create a new branch like `cleanup-refactor` before starting).

## 1. Structural Consolidation and Deduplication (High Priority)
These tasks focus on merging top-level directories into `pkg/` to eliminate redundancies and match the modular structure.

### Task 1.1: Merge or delete duplicate top-level directories into `pkg/` counterparts
- **Steps:**
  1. Create a Git branch: `git checkout -b cleanup-refactor`.
  2. For each top-level dir (agents/, core/, schema/, llms/, memory/, config/, prompts/, tools/, communication/):
     - List contents and compare with `pkg/` equivalent (e.g., use `diff` or manual review).
     - Move unique files to `pkg/<dir>/` (e.g., `mv agents/base.go pkg/agents/base/` if not duplicate).
     - Update imports in moved files to use correct paths (e.g., `github.com/lookatitude/beluga-ai/pkg/...`).
     - Delete the top-level dir if fully merged.
  3. Commit changes: `git commit -m "Merged top-level dirs into pkg/"`.

### Task 1.2: Rename and integrate `orchestrator/` into `pkg/orchestration/`
- **Steps:**
  1. Move files: `mv orchestrator/* pkg/orchestration/`.
  2. If needed, create subdirs (e.g., `mkdir pkg/orchestration/temporal` and move `temporal.go` there).
  3. Update imports and references.
  4. Delete empty `orchestrator/`.
  5. Commit: `git commit -m "Integrated orchestrator into pkg/orchestration"`.

### Task 1.3: Incorporate `rag/` components into relevant `pkg/` subpackages
- **Steps:**
  1. Create target subdirs if missing (e.g., `mkdir -p pkg/dataconnection/loaders`).
  2. Move subdirs: e.g., `mv rag/embedders/ pkg/embeddings/`, `mv rag/loaders/ pkg/dataconnection/loaders/`, etc.
  3. Update imports.
  4. Delete `rag/`.
  5. Commit: `git commit -m "Integrated rag components into pkg/"`.

### Task 1.4: Verify and clean up other top-level dirs
- **Steps:**
  1. Audit `assets/`, `website/`, `.github/`, `docs/` for code-related items.
  2. Move any misplaced code (e.g., from `assets/`) to appropriate `pkg/` locations or `docs/`.
  3. Delete unnecessary dirs/files (e.g., old `coverage.out`).
  4. Commit: `git commit -m "Cleaned up miscellaneous top-level dirs"`.

## 2. Package and Code Refinements (High Priority)
Ensure packages align with the architecture's substructures and principles.

### Task 2.1: Add missing subpackages and files in `pkg/agents/`
- **Steps:**
  1. Create missing subdirs: e.g., `mkdir pkg/agents/adapter` if not present.
  2. Add/verify files: e.g., ensure `tools/tool.go` exists; add `Plan` and `Execute` to `base/base_agent.go`.
  3. Update code for interface compliance.
  4. Commit: `git commit -m "Refined pkg/agents structure"`.

### Task 2.2: Standardize `pkg/orchestration/`
- **Steps:**
  1. Move loose files to subdirs: e.g., `mv pkg/orchestration/scheduler.go pkg/orchestration/scheduler/`.
  2. Enhance with async features (e.g., add goroutines to `InMemoryScheduler`).
  3. Commit: `git commit -m "Standardized pkg/orchestration"`.

### Task 2.3: Create or enhance potential subpackages
- **Steps:**
  1. Create dirs: e.g., `mkdir pkg/chatmodels`, `pkg/chains`, `pkg/server`, `pkg/ui`.
  2. Move any related code from other locations.
  3. Implement basic stubs if needed.
  4. Commit: `git commit -m "Added potential subpackages"`.

### Task 2.4: Update imports and remove unused code
- **Steps:**
  1. Run `go mod tidy`.
  2. Use linter (e.g., `golangci-lint run`) to fix imports and remove unused code.
  3. Manually correct paths to `github.com/lookatitude/beluga-ai/pkg/...`.
  4. Commit: `git commit -m "Updated imports and removed unused code"`.

## 3. Quality and Maintainability Improvements (Medium Priority)
Address testing, errors, and production features.

### Task 3.1: Add comprehensive testing
- **Steps:**
  1. Create test files: e.g., `pkg/agents/base/base_agent_test.go`.
  2. Write unit/integration tests for key interfaces.
  3. Run `go test ./...` and aim for high coverage.
  4. Commit: `git commit -m "Added tests"`.

### Task 3.2: Standardize error handling
- **Steps:**
  1. Create `pkg/core/utils/errors.go` with custom types.
  2. Update methods to use these errors consistently.
  3. Commit: `git commit -m "Standardized error handling"`.

### Task 3.3: Enhance configuration management
- **Steps:**
  1. Expand `pkg/config/config.go` with validation and env support.
  2. Test configurations.
  3. Commit: `git commit -m "Enhanced config management"`.

### Task 3.4: Implement dependency injection (DI)
- **Steps:**
  1. Add functional options or a DI container in relevant factories.
  2. Update usage in agents/LLMs.
  3. Commit: `git commit -m "Implemented DI"`.

### Task 3.5: Improve asynchronous and concurrent features
- **Steps:**
  1. Enhance `pkg/orchestration/` with worker pools and retries.
  2. Ensure safe concurrency with channels/sync.
  3. Commit: `git commit -m "Improved async features"`.

### Task 3.6: Propagate context and add observability
- **Steps:**
  1. Add `context.Context` to method signatures.
  2. Standardize logging in `pkg/monitoring/`.
  3. Commit: `git commit -m "Added context and observability"`.

## 4. Documentation and Project Hygiene (Low Priority)
Finalize docs and module cleanliness.

### Task 4.1: Update project documentation
- **Steps:**
  1. Edit `README.md`, `CONTRIBUTING.md` to reflect changes.
  2. Add sections on structure and testing.
  3. Commit: `git commit -m "Updated documentation"`.

### Task 4.2: Clean up Go module files
- **Steps:**
  1. Run `go mod tidy` and update deps.
  2. Commit: `git commit -m "Cleaned up go.mod/go.sum"`.

### Task 4.3: Ethical and best practices review
- **Steps:**
  1. Audit for biases, add human-in-loop hooks.
  2. Fix any globals/concurrency issues.
  3. Commit: `git commit -m "Ethical review and best practices"`.

## Execution Notes
- After each task, run `go build ./...` and `go test ./...` to verify.
- Once all tasks are complete, merge the branch: `git checkout main; git merge cleanup-refactor`.
- Track progress by checking off tasks in this file.
