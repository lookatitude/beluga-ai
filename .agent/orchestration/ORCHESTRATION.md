# Workflow Orchestration System

This document describes how to use multi-persona workflows to orchestrate complex tasks across different agent personas.

## Overview

Workflow orchestration allows you to chain multiple personas together to accomplish complex tasks that require different perspectives and skills. For example:
- **Verify Architecture**: Researcher scans → Architect validates → Report or Fix
- **Fix and Validate**: Backend Developer fixes → Re-validate
- **Comprehensive Review**: Full codebase quality review across all dimensions

## Directory Structure

```
.agent/orchestration/
├── ORCHESTRATION.md          # This file
├── workflows/                # Workflow definitions (YAML)
│   ├── verify-architecture.yaml
│   ├── fix-and-validate.yaml
│   └── comprehensive-review.yaml
└── reports/                  # Generated reports output
    └── .gitkeep
```

## Running Workflows

Execute a workflow using the runner script:

```bash
# Run a workflow
./.agent/scripts/run-workflow.sh <workflow-name>

# Dry run (show steps without executing)
./.agent/scripts/run-workflow.sh <workflow-name> --dry-run

# With verbose output
./.agent/scripts/run-workflow.sh <workflow-name> --verbose
```

### Examples

```bash
# Verify architecture compliance
./.agent/scripts/run-workflow.sh verify-architecture

# Run comprehensive review
./.agent/scripts/run-workflow.sh comprehensive-review

# Dry run to see what will happen
./.agent/scripts/run-workflow.sh verify-architecture --dry-run
```

## Available Workflows

| Workflow | Description | Personas Used |
|----------|-------------|---------------|
| `verify-architecture` | Scan codebase, validate patterns, generate report | researcher → architect |
| `fix-and-validate` | Fix issues and re-validate | backend-developer → architect |
| `comprehensive-review` | Full quality review | researcher → architect → qa |

## Workflow Definition Format

Workflows are defined in YAML format:

```yaml
name: Workflow Name
description: What this workflow does

steps:
  - name: step-name
    persona: persona-name       # Which persona to activate
    skill: skill-name           # Which skill to execute
    input:                      # Input parameters for the skill
      key: value
    output: output-file.md      # Where to save output
    on_failure: step-id         # Optional: step to run on failure
    condition: expression       # Optional: condition to run this step
    then: step-id               # Optional: next step (for loops)
```

### Step Properties

| Property | Required | Description |
|----------|----------|-------------|
| `name` | Yes | Unique step identifier |
| `persona` | Yes | Persona to activate for this step |
| `skill` | Yes | Skill to execute |
| `input` | No | Input parameters (can reference previous outputs with `${output-file}`) |
| `output` | No | Output file path (relative to reports/) |
| `on_failure` | No | Step ID to execute if this step fails |
| `condition` | No | Boolean expression to determine if step should run |
| `then` | No | Step ID to execute next (enables loops) |

### Variable References

Reference outputs from previous steps using `${filename}`:

```yaml
steps:
  - name: scan
    output: scan-report.md

  - name: validate
    input:
      source: ${scan-report.md}  # References output from scan step
```

## Workflow Execution Model

```
┌─────────────────────────────────────────────────────────────┐
│                    WORKFLOW EXECUTION                        │
├─────────────────────────────────────────────────────────────┤
│ 1. Parse workflow YAML                                       │
│ 2. Initialize shared context                                 │
│ 3. For each step:                                           │
│    a. Activate persona                                       │
│    b. Check condition (skip if false)                        │
│    c. Execute skill with inputs                              │
│    d. Capture output                                         │
│    e. Check on_failure (branch if error)                     │
│    f. Follow then pointer (or next step)                     │
│ 4. Generate final report                                     │
└─────────────────────────────────────────────────────────────┘
```

### Loop Protection

To prevent infinite loops, workflows have a maximum iteration count (default: 3) for any step that loops back via `then`. After max iterations, the workflow fails with an error.

## Creating Custom Workflows

1. Create a new YAML file in `.agent/orchestration/workflows/`:

```yaml
# .agent/orchestration/workflows/my-workflow.yaml
name: My Custom Workflow
description: Description of what it does

steps:
  - name: first-step
    persona: researcher
    skill: research_topic
    input:
      question: "What to research"
    output: research.md

  - name: second-step
    persona: backend-developer
    skill: implement_feature
    input:
      specification: ${research.md}
    output: implementation.md
```

2. Run it:

```bash
./.agent/scripts/run-workflow.sh my-workflow
```

## Integration with Personas

Workflows leverage the existing persona system:
- Each step activates a persona via `activate-persona.sh`
- Persona rules are loaded for the duration of the step
- Skills defined in the persona are available

See `.agent/agents.md` for available personas and their skills.

## Reports

All workflow outputs are saved to `.agent/orchestration/reports/`:

```
reports/
├── verify-architecture-2024-01-15T10-30-00/
│   ├── scan-report.md
│   ├── validation-report.md
│   └── final-report.md
└── comprehensive-review-2024-01-15T11-00-00/
    └── ...
```

Each workflow run creates a timestamped directory containing all step outputs.

## Best Practices

1. **Small, Focused Steps**: Each step should do one thing well
2. **Clear Outputs**: Name outputs descriptively
3. **Handle Failures**: Define `on_failure` paths for critical steps
4. **Avoid Deep Loops**: Keep fix→validate loops to 1-2 iterations
5. **Document Custom Workflows**: Add descriptions and comments

## Troubleshooting

### Workflow Not Found
```
Error: Workflow 'name' not found
```
Check that the YAML file exists in `.agent/orchestration/workflows/`.

### Persona Activation Failed
```
Error: Failed to activate persona 'name'
```
Verify the persona exists in `.agent/personas/`.

### Step Timeout
Steps have a default timeout of 5 minutes. Long-running tasks may need adjustment in the workflow YAML.

## References

- `.agent/agents.md` - Available personas and skills
- `.agent/personas/` - Persona definitions
- `.agent/skills/` - Skill definitions
- `.agent/rules/` - Framework rules (always loaded)
