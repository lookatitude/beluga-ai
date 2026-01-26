# Doc and Godoc Validation

Generated API documentation MUST be validated in CI and locally via `make ci-local`. Validation includes both "fix by regenerating" and extra checks that regeneration alone cannot fix.

- **Where:** CI (e.g. generate-api-docs or equivalent) and local: `make ci-local` runs the doc validation steps.
- **validate-godoc-output.sh (or equivalent):**
  - Generated files exist and are non-empty.
  - Frontmatter: `---`, `title`, `sidebar_position` (or whatever the site requires).
  - "Too short" when a file has fewer than a set line count (e.g. 10) — warning.
  - `<details>` in output — warn (can break MDX); fix in generate-docs.sh (e.g. flatten to `###`) so regeneration resolves it.
- **validate-docs.sh (or equivalent):** Workflow triggers on the right paths; gomarkdoc install and generate-docs.sh; doc-generation failure fails the job.
- **Extra checks (beyond regeneration):** Include checks that cannot be fixed solely by re-running generate-docs.sh, e.g.:
  - Lint of generated markdown (links, structure, forbidden patterns).
  - Consistency (e.g. all packages under pkg/ that should be documented have a corresponding generated file).
  - Project-specific rules (e.g. no raw HTML, required sections).
- **When validation fails:** Fix via regeneration when the cause is godoc/gomarkdoc; fix via source or generator/script when the cause is extra checks or generator behavior.
