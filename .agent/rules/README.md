# Beluga AI Framework Rules

This directory contains the **framework-wide rules** that apply to ALL development work on Beluga AI, regardless of which persona is active.

## Rule Files

| File | Purpose | Always Apply |
|------|---------|--------------|
| `framework-architecture.mdc` | Architecture, design patterns, package structure | Yes |
| `framework-quality.mdc` | Quality standards, testing requirements, CI/CD | Yes |
| `framework-observability.mdc` | OTEL metrics, tracing, logging (MANDATORY) | Yes |

## How Rules Are Loaded

### Framework Rules (Always Loaded)
These rules in `.agent/rules/` are **always applied** regardless of active persona:
- Architecture compliance
- Quality standards
- Observability requirements

### Persona Rules (Per Activation)
When a persona is activated via `activate-persona.sh`, additional role-specific rules are loaded from `.agent/personas/{name}/rules/main.mdc`.

## Rule Hierarchy

```
Framework Rules (.agent/rules/)          ← Always loaded
         ↓
Persona Rules (.agent/personas/*/rules/) ← Per activation
```

## Activating Personas

```bash
# Activate a specific persona
./.agent/scripts/activate-persona.sh backend-developer

# This creates a symlink:
# .cursor/rules/active-persona.mdc → .agent/personas/backend-developer/rules/main.mdc
```

## Available Personas

| Persona | Focus | Location |
|---------|-------|----------|
| backend-developer | Implementation | `.agent/personas/backend-developer/` |
| architect | Design & Review | `.agent/personas/architect/` |
| researcher | Analysis | `.agent/personas/researcher/` |
| qa | Testing | `.agent/personas/qa/` |
| documentation-writer | Documentation | `.agent/personas/documentation-writer/` |

## Creating New Rules

Framework rules should:
1. Use `.mdc` extension (Markdown with YAML frontmatter)
2. Include `globs` to specify which files they apply to
3. Set `alwaysApply: true` for mandatory rules
4. Be focused on a single concern

```yaml
---
description: Brief description of what this rule enforces
globs: "**/*.go"  # Files this rule applies to
alwaysApply: true # Always loaded, not persona-specific
---

# Rule Title

[Rule content in markdown]
```

## References

- `.agent/agents.md` - Master persona registry
- `.agent/personas/` - Persona definitions
- `.cursorrules` - Entry point (points here)
- `CLAUDE.md` - AI assistant guide
