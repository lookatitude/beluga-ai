---
name: Compare Approaches
description: Trade-off analysis for implementation decisions
personas:
  - researcher
---

# Compare Approaches

This skill guides you through systematically comparing different implementation approaches to inform decision-making.

## Prerequisites

- Clear decision to be made
- 2 or more alternatives to compare
- Understanding of requirements and constraints

## Steps

### 1. Define the Decision

```markdown
## Decision: [What needs to be decided]

### Context
[Background and why this decision matters]

### Requirements
1. [Must-have requirement 1]
2. [Must-have requirement 2]
3. [Nice-to-have requirement]

### Constraints
- [Technical constraint]
- [Resource constraint]
- [Timeline constraint]

### Success Criteria
[How we'll know we made the right choice]
```

### 2. Identify Alternatives

List all viable options:

```markdown
## Alternatives

### Option A: [Name]
**Description**: [Brief explanation]
**Precedent**: [Where this is used in codebase or industry]

### Option B: [Name]
**Description**: [Brief explanation]
**Precedent**: [Where this is used]

### Option C: [Name]
**Description**: [Brief explanation]
**Precedent**: [Where this is used]
```

### 3. Define Evaluation Criteria

Select relevant criteria:

| Category | Criteria | Weight |
|----------|----------|--------|
| **Performance** | Latency | High |
| | Throughput | Medium |
| | Memory usage | Low |
| **Maintainability** | Code complexity | High |
| | Testability | High |
| | Documentation | Medium |
| **Compatibility** | Existing patterns | High |
| | Future extensibility | Medium |
| | Breaking changes | High |
| **Operational** | Monitoring | Medium |
| | Debugging | Medium |
| | Deployment | Low |

### 4. Analyze Each Option

For each alternative, evaluate against criteria:

```markdown
## Analysis: Option A - [Name]

### Implementation Approach
```go
// Example implementation
type ComponentA struct {
    // ...
}

func NewComponentA() *ComponentA {
    // ...
}
```

### Performance
- **Latency**: [Assessment with evidence]
- **Throughput**: [Assessment]
- **Memory**: [Assessment]

### Maintainability
- **Complexity**: [Assessment]
  - Lines of code: ~X
  - Cyclomatic complexity: Low/Medium/High
- **Testability**: [Assessment]
- **Documentation**: [Assessment]

### Compatibility
- **Existing patterns**: [How it fits]
  - Similar to: `pkg/example/` implementation
- **Extensibility**: [Future changes]
- **Breaking changes**: [Impact]

### Operational
- **Monitoring**: [OTEL integration]
- **Debugging**: [Error visibility]
- **Deployment**: [Rollout considerations]

### Pros
1. [Advantage 1]
2. [Advantage 2]

### Cons
1. [Disadvantage 1]
2. [Disadvantage 2]

### Risk Assessment
- [Risk 1]: [Likelihood] / [Impact]
- [Risk 2]: [Likelihood] / [Impact]
```

### 5. Create Comparison Matrix

```markdown
## Comparison Matrix

| Criteria | Weight | Option A | Option B | Option C |
|----------|--------|----------|----------|----------|
| **Performance** |
| Latency | High | ⭐⭐⭐ | ⭐⭐ | ⭐ |
| Throughput | Medium | ⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| Memory | Low | ⭐⭐ | ⭐ | ⭐⭐⭐ |
| **Maintainability** |
| Complexity | High | ⭐⭐⭐ | ⭐⭐ | ⭐ |
| Testability | High | ⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐ |
| **Compatibility** |
| Existing patterns | High | ⭐⭐⭐ | ⭐⭐ | ⭐ |
| Extensibility | Medium | ⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐ |
| **Weighted Score** | | **X.X** | **X.X** | **X.X** |

Legend: ⭐ = Poor, ⭐⭐ = Acceptable, ⭐⭐⭐ = Excellent
```

### 6. Analyze Trade-offs

```markdown
## Trade-off Analysis

### Option A vs Option B
- **A is better when**: [Conditions]
- **B is better when**: [Conditions]
- **Key trade-off**: [Performance vs Flexibility, etc.]

### Critical Trade-offs
1. **[Trade-off 1]**: Choosing X means accepting Y
   - Impact: [Description]
   - Mitigation: [How to address]

2. **[Trade-off 2]**: ...
```

### 7. Make Recommendation

```markdown
## Recommendation

### Primary Recommendation: Option [X]

**Rationale**:
1. [Reason aligned with highest-weight criteria]
2. [Reason aligned with requirements]
3. [Reason considering constraints]

**Conditions for this recommendation**:
- [Assumption 1]
- [Assumption 2]

### Alternative: Option [Y]

**When to choose this instead**:
- [Condition 1]
- [Condition 2]

### Not Recommended: Option [Z]

**Why not**:
- [Critical issue 1]
- [Critical issue 2]

### Implementation Notes
1. [Key consideration for implementing chosen option]
2. [Potential pitfall to avoid]
3. [Migration path if changing later]
```

### 8. Document Decision

```markdown
## Decision Record

**Date**: [YYYY-MM-DD]
**Decision**: [Chosen option]
**Status**: [Proposed/Accepted/Deprecated]

### Context
[Brief background]

### Decision
We will use [Option X] because:
1. [Primary reason]
2. [Secondary reason]

### Consequences
**Positive**:
- [Benefit 1]

**Negative**:
- [Trade-off accepted]

**Neutral**:
- [Neither good nor bad]

### Follow-up Actions
- [ ] [Action 1]
- [ ] [Action 2]
```

## Comparison Templates

### Quick Comparison (2 options)
```markdown
| Aspect | Option A | Option B |
|--------|----------|----------|
| [Criterion 1] | [Value] | [Value] |
| [Criterion 2] | [Value] | [Value] |

**Recommendation**: [Option] because [one-sentence reason]
```

### Detailed Comparison (3+ options)
Use full template above with all sections.

## Quality Checklist

- [ ] All options are viable (no strawmen)
- [ ] Criteria are weighted by importance
- [ ] Evidence supports assessments
- [ ] Trade-offs are explicit
- [ ] Recommendation has clear rationale
- [ ] Conditions for alternative are defined
- [ ] Implementation notes provided

## Output

A comparison document with:
- Clear problem statement
- Comprehensive option analysis
- Weighted comparison matrix
- Trade-off analysis
- Actionable recommendation
- Decision record for future reference
