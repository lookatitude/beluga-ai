# Corrections Log

This file used to contain all corrections. It has been split into per-category files under `.wiki/corrections/`:

- [corrections/architecture.md](./corrections/architecture.md) — arch-validate findings
- [corrections/docs-drift.md](./corrections/docs-drift.md) — docs-writer / docs-audit / marketeer findings
- [corrections/workflow.md](./corrections/workflow.md) — coordinator + agent-workflow findings

See [corrections/README.md](./corrections/README.md) for the format, promotion pipeline, and ID-to-category lookup table.

**All existing `C-NNN` IDs are preserved across the split.** References elsewhere (log.md, PR descriptions, agent prompts) continue to resolve.

---

## Adding a new correction

1. Determine the discovering workflow (arch-validate / docs-writer / docs-audit / marketeer / coordinator).
2. Choose the category file accordingly (architecture / docs-drift / workflow).
3. Append a new `### C-NNN | YYYY-MM-DD | <workflow> | <package>` section at the bottom of the chosen file.
4. Use the next unused `C-NNN` ID — check `corrections/README.md`'s lookup table for the current maximum.
5. Update the lookup table in `corrections/README.md`.
