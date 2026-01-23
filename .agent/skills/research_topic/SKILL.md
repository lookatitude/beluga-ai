---
name: Research Topic
description: Codebase exploration and analysis with pattern discovery
personas:
  - researcher
---

# Research Topic

This skill guides you through researching a topic within the Beluga AI codebase, documenting patterns, and providing evidence-based recommendations.

## Prerequisites

- Clear research question or topic
- Understanding of what decision the research will inform
- Access to the codebase

## Steps

### 1. Define Research Question

Frame the research clearly:

```markdown
## Research Question

**Topic**: [What are we investigating?]

**Context**: [Why is this important? What decision does it inform?]

**Scope**: [What areas of the codebase are relevant?]

**Deliverable**: [What output is expected?]
```

### 2. Map Relevant Areas

Identify codebase locations to explore:

| Area | Location | Relevance |
|------|----------|-----------|
| Core implementation | `pkg/<package>/` | Primary |
| Interfaces | `pkg/<package>/iface/` | High |
| Providers | `pkg/<package>/providers/` | High |
| Tests | `pkg/<package>/*_test.go` | Usage patterns |
| Examples | `examples/<category>/` | Usage patterns |
| Documentation | `docs/` | Intended behavior |

### 3. Explore and Document

For each relevant file:

```markdown
### File: `pkg/example/example.go`

**Purpose**: [What this file does]

**Key Types**:
- `Component` (line 15): Main implementation
- `Config` (line 42): Configuration struct

**Key Functions**:
- `NewComponent()` (line 58): Factory function
- `Process()` (line 85): Main processing logic

**Patterns Observed**:
- Functional options at line 60-75
- OTEL tracing at line 87
- Error wrapping at line 102

**Notable Code**:
```go
// From example.go:85-95
func (c *Component) Process(ctx context.Context) error {
    ctx, span := c.tracer.Start(ctx, "component.process")
    defer span.End()
    // ...
}
```
```

### 4. Trace Code Paths

Follow execution from entry to completion:

```markdown
## Code Trace: [Flow Name]

### Entry Point
`pkg/agents/agent.go:NewAgent()` (line 42)

### Execution Flow

1. **NewAgent** (agent.go:42)
   - Validates configuration
   - Creates internal components
   - Returns agent instance

2. **agent.Run** (agent.go:85)
   - Starts main loop
   - Calls `processMessage()` for each input

3. **agent.processMessage** (agent.go:120)
   - Invokes LLM via `llm.Generate()`
   - Processes tools if needed
   - Returns response

### Key Decision Points

| Location | Decision | Options |
|----------|----------|---------|
| agent.go:95 | LLM selection | Based on config.Provider |
| agent.go:130 | Tool execution | If LLM returns tool call |

### Dependencies Called

- `pkg/llms` - LLM generation
- `pkg/memory` - Conversation storage
- `pkg/agents/tools` - Tool execution
```

### 5. Identify Patterns

Document recurring patterns:

```markdown
## Pattern: [Name]

### Description
[What this pattern does and why it's used]

### Locations Found
1. `pkg/llms/llms.go:45`
2. `pkg/embeddings/embeddings.go:38`
3. `pkg/vectorstores/vectorstores.go:52`

### Implementation

```go
// Common pattern structure
type Registry struct {
    mu       sync.RWMutex
    creators map[string]CreatorFunc
}

func RegisterGlobal(name string, creator CreatorFunc) {
    registry.mu.Lock()
    defer registry.mu.Unlock()
    registry.creators[name] = creator
}
```

### Usage
[When and how to apply this pattern]

### Variations
[Different implementations observed]
```

### 6. Create Comparison (if applicable)

For comparing approaches:

```markdown
## Comparison: [Topic]

### Options Evaluated

| Aspect | Option A | Option B | Option C |
|--------|----------|----------|----------|
| Performance | High | Medium | Low |
| Complexity | Low | Medium | High |
| Flexibility | Limited | Good | Excellent |
| Testing | Easy | Moderate | Complex |
| Existing Usage | 5 places | 2 places | 0 places |

### Detailed Analysis

#### Option A: [Name]
**Pros**:
- [Benefit 1]
- [Benefit 2]

**Cons**:
- [Drawback 1]

**Example**: `pkg/example/implementation.go:42`

#### Option B: [Name]
...

### Recommendation
[Which option and detailed rationale]
```

### 7. Summarize Findings

Create executive summary:

```markdown
## Research Summary: [Topic]

### Key Findings

1. **Finding 1**: [Summary]
   - Evidence: `file.go:line`
   - Implication: [What this means]

2. **Finding 2**: [Summary]
   - Evidence: `file.go:line`
   - Implication: [What this means]

### Patterns Identified

| Pattern | Occurrences | Recommended |
|---------|-------------|-------------|
| [Pattern A] | 5 | Yes |
| [Pattern B] | 2 | Conditionally |

### Recommendations

1. **Primary**: [Main recommendation with rationale]
2. **Alternative**: [Backup option if primary doesn't fit]

### Open Questions
- [Unresolved question 1]
- [Unresolved question 2]

### References
- `pkg/relevant/file.go` - [Why relevant]
- `docs/relevant.md` - [Why relevant]
```

## Output Formats

### Quick Research (< 1 hour)
- 2-3 sentence summary
- Key finding with code reference
- Recommendation

### Standard Research (1-4 hours)
- Executive summary
- Findings with evidence
- Comparison table (if applicable)
- Recommendation with rationale

### Deep Research (> 4 hours)
- Full report with all sections
- Multiple code traces
- Comprehensive comparison
- Detailed recommendations
- Open questions for follow-up

## Quality Checklist

- [ ] All code references include `file:line`
- [ ] Actual code snippets (not paraphrases)
- [ ] Assumptions documented
- [ ] Multiple perspectives considered
- [ ] Edge cases noted
- [ ] Test files reviewed for usage patterns
- [ ] Recommendations have clear rationale

## Output

A research document with:
- Clear findings with evidence
- Code references and snippets
- Pattern documentation
- Comparison tables (if applicable)
- Actionable recommendations
