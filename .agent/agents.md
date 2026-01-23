# Beluga AI Agent Personas

This document describes the available agent personas for working with the Beluga AI codebase. Each persona is optimized for a specific role with tailored skills, rules, and permissions.

## Quick Reference

| Persona | Focus | Permissions |
|---------|-------|-------------|
| [Backend Developer](#backend-developer) | Implementation | Build, Test, Lint, Commit |
| [Architect](#architect) | Design | Read-only |
| [Researcher](#researcher) | Analysis | Read-only |
| [QA Engineer](#qa-engineer) | Testing | Test, Lint, Security |
| [Documentation Writer](#documentation-writer) | Documentation | Docs only |

## Activation

To activate a persona, run:

```bash
./.agent/scripts/activate-persona.sh <persona-name>
```

This creates a symlink to the persona's rules in `.cursor/rules/active-persona.mdc`.

### Example

```bash
# Activate backend developer persona
./.agent/scripts/activate-persona.sh backend-developer

# Activate QA persona
./.agent/scripts/activate-persona.sh qa
```

---

## Available Personas

### Backend Developer

**Location**: `.agent/personas/backend-developer/`

**Description**: Senior Go backend developer implementing production code for Beluga AI.

**Skills**:
- `create_package` - Create new standardized packages
- `create_provider` - Add new provider implementations
- `add_agent` - Create new agent types
- `implement_feature` - End-to-end feature implementation
- `fix_bug` - Bug investigation and fixing

**Workflows**:
- `feature_development` - Standard feature development process
- `run_quality_checks` - Quality assurance pipeline

**Permissions**:
- Build: Yes (`make build`)
- Test: Yes (`make test`, `make test-unit`)
- Lint: Yes (`make lint`, `make lint-fix`)
- Commit: Yes

**Activation**:
```bash
./.agent/scripts/activate-persona.sh backend-developer
```

---

### Architect

**Location**: `.agent/personas/architect/`

**Description**: Software architect ensuring design integrity and pattern compliance.

**Skills**:
- `design_component` - Component design following ISP, DIP, SRP
- `review_architecture` - Architecture review and compliance verification
- `create_package` - Package structure design

**Permissions**:
- Read-only (analysis mode)
- No code modifications

**Activation**:
```bash
./.agent/scripts/activate-persona.sh architect
```

---

### Researcher

**Location**: `.agent/personas/researcher/`

**Description**: Technical researcher exploring the codebase and documenting patterns.

**Skills**:
- `research_topic` - Codebase exploration and analysis
- `compare_approaches` - Trade-off analysis for decisions

**Permissions**:
- Read-only
- No code modifications

**Activation**:
```bash
./.agent/scripts/activate-persona.sh researcher
```

---

### QA Engineer

**Location**: `.agent/personas/qa/`

**Description**: QA engineer ensuring quality standards through testing and validation.

**Skills**:
- `create_test_suite` - Comprehensive test suite creation
- `run_quality_checks` - Quality pipeline execution

**Workflows**:
- `run_quality_checks` - Full quality assurance pipeline

**Permissions**:
- Test: Yes (`make test`, `make test-unit`, `make test-integration`)
- Lint: Yes (`make lint`)
- Security: Yes (`make security`)
- No feature code modifications

**Activation**:
```bash
./.agent/scripts/activate-persona.sh qa
```

---

### Documentation Writer

**Location**: `.agent/personas/documentation-writer/`

**Description**: Technical writer creating and maintaining documentation.

**Skills**:
- `write_guide` - Guide creation ("The Teacher" persona)
- `write_api_docs` - API reference documentation

**Permissions**:
- Edit documentation only
- Allowed paths: `docs/**/*.md`, `website/**/*.md`, `pkg/**/README.md`

**Activation**:
```bash
./.agent/scripts/activate-persona.sh documentation-writer
```

---

## Persona Structure

Each persona directory contains:

```
.agent/personas/<persona-name>/
├── PERSONA.md        # Role definition, context, skills, permissions
└── rules/
    └── main.mdc      # Isolated rule set for this persona
```

### PERSONA.md Format

```yaml
---
name: [Persona Name]
description: [Brief description]
skills:
  - skill_1
  - skill_2
workflows:
  - workflow_1
permissions:
  permission1: true
  permission2: false
---

# [Persona Name] Agent

[Detailed instructions and context for the persona]
```

### rules/main.mdc Format

```yaml
---
description: [Rule set description]
globs: [file patterns]
alwaysApply: false
---

# [Persona] Rules

[Persona-specific rules and guidelines]
```

---

## Creating Custom Personas

To create a new persona:

1. Create directory structure:
   ```bash
   mkdir -p .agent/personas/<name>/rules
   ```

2. Create `PERSONA.md` with YAML frontmatter defining:
   - name, description
   - skills, workflows
   - permissions

3. Create `rules/main.mdc` with persona-specific rules

4. Activate with:
   ```bash
   ./.agent/scripts/activate-persona.sh <name>
   ```

---

## Workflow Orchestration

Multi-persona workflows allow chaining different personas to accomplish complex tasks.

### Running Workflows

```bash
# Run a workflow
./.agent/scripts/run-workflow.sh <workflow-name>

# Dry run (preview steps)
./.agent/scripts/run-workflow.sh <workflow-name> --dry-run
```

### Available Workflows

| Workflow | Description | Personas |
|----------|-------------|----------|
| `verify-architecture` | Scan, validate patterns, report | researcher -> architect |
| `fix-and-validate` | Fix issues and re-validate | backend-developer -> architect |
| `comprehensive-review` | Full quality review | researcher -> architect -> qa |

See [Orchestration Guide](./orchestration/ORCHESTRATION.md) for details.

---

## Related Documentation

- [Framework Rules](./rules/) - Framework-wide rules (always loaded)
- [Skills](./skills/) - Available skills
- [Workflows](./workflows/) - Simple workflows
- [Orchestration](./orchestration/ORCHESTRATION.md) - Multi-persona workflow orchestration
- [CLAUDE.md](../CLAUDE.md) - AI assistant guide
