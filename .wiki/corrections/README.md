# Corrections Log — Index

Every correction is an opportunity to prevent future mistakes. Agents search this directory for their target package before starting work.

## Structure (post-B split)

Corrections are grouped into three category files, each chronological within its category. The split preserves every `C-NNN` ID verbatim — references elsewhere (logs, PR descriptions, agent prompts) continue to resolve.

- **[architecture.md](./architecture.md)** — arch-validate findings: interface shape, invariants, layering, Go idiom violations
- **[docs-drift.md](./docs-drift.md)** — docs-writer / docs-audit / marketeer findings: docs↔code divergence, wiki drift, website/mermaid issues
- **[workflow.md](./workflow.md)** — coordinator findings: retrieval protocol, agent-workflow mechanics, worktree awareness

## Format

```
### C-NNN | YYYY-MM-DD | <workflow> | <package>
**Symptom:** what went wrong
**Root cause:** why it happened
**Correction:** what's right
**Prevention rule:** where the rule was added
**Confidence:** HIGH / MEDIUM / LOW
```

## Promotion pipeline

Per-agent `rules/` → this directory → `.claude/rules/<file>.md` → (human-approved) `CLAUDE.md`.
Entries reach `.claude/rules/` when seen ≥3 times or HIGH confidence.

## ID → category lookup

| ID | Category |
|---|---|
| C-001 | architecture.md |
| C-002 | architecture.md |
| C-003 | architecture.md |
| C-004 | architecture.md |
| C-005 | architecture.md |
| C-006 | docs-drift.md |
| C-007 | docs-drift.md |
| C-008 | workflow.md |
| C-009 | docs-drift.md |
| C-010 | docs-drift.md (two entries — prompt and rag/retriever; duplicate numbering preserved from source) |
| C-011 | docs-drift.md |
| C-012 | docs-drift.md |
| C-013 | docs-drift.md |
| C-014 | docs-drift.md |
| C-015 | docs-drift.md |
| C-016 | workflow.md |
| C-017 | docs-drift.md |

### Note on C-010 duplicate

The source `corrections.md` contains two `### C-010 ...` headers (2026-04-12 `docs-writer | prompt` + 2026-04-12 `docs-writer | rag/retriever`). The split preserves both as-is under `docs-drift.md`. Fix the numbering in a follow-up: renumber the later entry to `C-018` (or the next unused number) and update any references.
