---
name: learn
description: Capture a correction, discovery, or process improvement into the wiki.
---

Capture learning: $ARGUMENTS

## Workflow

### Step 1 — Structure
`@agent-coordinator` parses the correction and determines:
- Which package(s) affected
- Root cause (not just symptom)
- Prevention strategy
- Confidence (HIGH / MEDIUM / LOW)

### Step 2 — Record
Append a C-NNN entry to `.wiki/corrections.md`:

```
### C-NNN | YYYY-MM-DD | <workflow> | <package>
Symptom: ...
Root cause: ...
Correction: ...
Prevention rule: ...
Confidence: HIGH/MEDIUM/LOW
```

### Step 3 — Update rules (if applicable)
If the correction reveals a pattern to enforce, update the relevant `.claude/rules/<file>.md`.

### Step 4 — Update patterns (if applicable)
If the correction reveals a canonical implementation approach, `@agent-architect` updates `.wiki/patterns/<relevant>.md`.

### Step 5 — Propose CLAUDE.md update (if mature)
If this correction has been seen ≥3 times or is HIGH confidence: propose a `CLAUDE.md` rule addition for human approval. Never auto-write to `CLAUDE.md` — human decides.

### Step 6 — Log
Append to `.wiki/log.md`.
