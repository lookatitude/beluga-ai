---
name: notion-sync
description: Standalone Notion sync. Mirrors docs/ to Notion and updates the tracking dashboard.
---

Run Notion sync.

## Workflow

### Step 1 — Sync
`@agent-notion-syncer` runs both tasks:
- Mirror `docs/` to Notion pages (using `.claude/state/notion-pages.json` mapping).
- Update the project tracking dashboard from `.wiki/log.md`, `.wiki/architecture/decisions.md`, `.wiki/corrections.md`, and `.claude/state/learnings-index.md`.

### Step 2 — Report
- Pages created (with Notion URLs)
- Pages updated
- Sync failures and causes

### Step 3 — Log
Append to `.wiki/log.md`.
