# Architecture Decision Records

ADR log for Beluga AI v2. Each decision is immutable — supersede with a new entry rather than editing old ones.

## Format

```
### ADR-NNN | YYYY-MM-DD | <title>
**Status:** Accepted / Superseded by ADR-XXX / Deprecated
**Context:** what problem needs solving
**Decision:** what we decided
**Rationale:** why this over alternatives
**Consequences:** positive + negative impacts
**Alternatives considered:** brief
```

---

### ADR-001 | 2026-04-11 | Adopt unified self-evolving multi-agent system

**Status:** Accepted
**Context:** The v1 agent setup had monolithic CLAUDE.md, single-file learnings, and no enforcement layer. A parallel `.claude/teams/` system existed with better learning infrastructure but was bound to a one-shot migration workflow.
**Decision:** Fuse both systems into a 5-layer design: deterministic hooks (L1), two-tier knowledge (L2), 10 lean agents (L3), 15 standalone-composable workflow commands (L4), and an evolution pipeline (L5).
**Rationale:** Keeps the proven learning hooks from teams/ (cross-pollinating per-agent rules) while adopting the plan's architectural boundaries (file-scoped auto-loaded rules, retrieval-on-demand wiki, <2500-token CLAUDE.md).
**Consequences:**
- Positive: learning is both fast (automatic per-agent) and curated (wiki-promoted). Every workflow is independently triggerable. Enforcement is deterministic.
- Negative: two learning stores means some duplication; coordinator must periodically promote per-agent findings to wiki.
**Alternatives considered:** Full replacement (lost the existing hook infrastructure); additive migration (left duplication without integration).
