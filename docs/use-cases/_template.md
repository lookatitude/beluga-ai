# {Use Case Title}

<!--
Template Guidelines for Use Cases:
- Tell a story - describe a real business problem
- Be honest about challenges and trade-offs
- Include concrete implementation details
- Make it feel like a real-world case study
- Write like you're presenting to peers
-->

## Overview

<!--
Set the scene. Describe the business problem in relatable terms.
Use concrete examples.
Example: "A customer support team needs to..." rather than "An organization requires..."
-->

{Organization type} needed to {business objective}. They faced challenges with {pain points}, and required a solution that could {key requirements}.

**The challenge:** {One sentence describing the core problem}

**The solution:** {One sentence describing what we built}

## Business Context

<!--
Explain why this use case matters.
What pain does it solve? What value does it deliver?
Be specific about outcomes.
-->

### The Problem

{Describe the current state and its problems}

- {Pain point 1 with quantifiable impact if possible}
- {Pain point 2}
- {Pain point 3}

### The Opportunity

By implementing {solution}, the team could:

- {Benefit 1 with expected improvement}
- {Benefit 2}
- {Benefit 3}

### Success Metrics

| Metric | Before | Target | Achieved |
|--------|--------|--------|----------|
| {Metric 1} | {Value} | {Target} | {Result} |
| {Metric 2} | {Value} | {Target} | {Result} |

## Requirements

<!--
List functional and non-functional requirements clearly.
Explain the reasoning behind each requirement.
-->

### Functional Requirements

| ID | Requirement | Rationale |
|----|-------------|-----------|
| FR1 | {Requirement} | {Why this is needed} |
| FR2 | {Requirement} | {Why this is needed} |
| FR3 | {Requirement} | {Why this is needed} |

### Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR1 | {Performance/reliability/scalability} | {Specific target} |
| NFR2 | {Requirement} | {Target} |

### Constraints

- {Technical constraint}
- {Business constraint}
- {Time/resource constraint}

## Architecture

<!--
Describe the solution architecture.
Include a diagram (ASCII or Mermaid).
Use a narrative style to explain how components interact.
-->

### High-Level Design

```
┌─────────────────┐       ┌─────────────────┐       ┌─────────────────┐
│                 │       │                 │       │                 │
│   {Component}   │──────▶│   {Component}   │──────▶│   {Component}   │
│                 │       │                 │       │                 │
└─────────────────┘       └─────────────────┘       └─────────────────┘
         │                         │
         │                         │
         ▼                         ▼
┌─────────────────┐       ┌─────────────────┐
│   {Component}   │       │   {Component}   │
└─────────────────┘       └─────────────────┘
```

### How It Works

The system works like this:

1. **{Step 1}** - When {trigger}, the system {action}. This is handled by {component} because {reason}.

2. **{Step 2}** - Next, {component} {action}. We chose this approach because {rationale}.

3. **{Step 3}** - Finally, {outcome}. The user sees {result}.

### Component Details

| Component | Purpose | Technology |
|-----------|---------|------------|
| {Name} | {What it does} | {Beluga AI package/feature} |
| {Name} | {What it does} | {Technology} |

## Implementation

<!--
Step-by-step implementation guide.
Reference specific guides and examples.
Include code snippets with explanations.
-->

### Phase 1: {Setup/Foundation}

First, we set up the basic infrastructure:

```go
package main

import (
    "context"
    
    "github.com/lookatitude/beluga-ai/pkg/{package}"
)

// {Comment explaining the setup}
func setup(ctx context.Context) (*Component, error) {
    // ...
}
```

**Key decisions:**
- We chose {option} because {reason}
- {Another decision} enables {benefit}

For detailed setup instructions, see the [{Guide Name}](../guides/{guide}.md).

### Phase 2: {Core Implementation}

Next, we implemented the core logic:

```go
// {Comment explaining the implementation}
func processRequest(ctx context.Context, input Input) (Output, error) {
    // Step 1: {Action}
    // ...
    
    // Step 2: {Action}
    // ...
    
    return output, nil
}
```

**Challenges encountered:**
- {Challenge 1}: Solved by {solution}
- {Challenge 2}: Addressed using {approach}

### Phase 3: {Integration/Polish}

Finally, we integrated monitoring and error handling:

```go
// Production-ready with OTEL instrumentation
func processRequestWithMonitoring(ctx context.Context, input Input) (Output, error) {
    ctx, span := tracer.Start(ctx, "process.request")
    defer span.End()
    
    // ... implementation with error handling
}
```

## Results

<!--
Describe actual or expected outcomes.
Include metrics if available.
Be honest about trade-offs.
-->

### Performance Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| {Metric 1} | {Value} | {Value} | {%} |
| {Metric 2} | {Value} | {Value} | {%} |
| {Metric 3} | {Value} | {Value} | {%} |

### Qualitative Outcomes

- **{Outcome 1}**: {Description of impact}
- **{Outcome 2}**: {Description}

### Trade-offs

| Trade-off | Benefit | Cost |
|-----------|---------|------|
| {Decision} | {What we gained} | {What we sacrificed} |

## Lessons Learned

<!--
Share insights, gotchas, and recommendations.
Write like a post-mortem: honest and constructive.
-->

### What Worked Well

✅ **{Success 1}** - {Why it worked and recommendation}

✅ **{Success 2}** - {Description}

### What We'd Do Differently

⚠️ **{Lesson 1}** - In hindsight, we would {alternative approach} because {reason}.

⚠️ **{Lesson 2}** - {Description}

### Recommendations for Similar Projects

1. **Start with {recommendation}** - This saves time because {reason}
2. **Consider {factor} early** - We discovered {insight} late in the project
3. **Don't underestimate {aspect}** - {Explanation}

## Related Use Cases

<!--
Link to related use cases, guides, and examples.
Explain relationships clearly.
-->

If you're working on a similar project, you might also find these helpful:

- **[{Related Use Case}](./related-use-case.md)** - Similar scenario focusing on {different aspect}
- **[{Guide}](../guides/{guide}.md)** - Deep dive into {technology used}
- **[{Example}](/examples/{example}/README.md)** - Runnable code demonstrating {feature}
- **[{Cookbook Recipe}](../cookbook/{recipe}.md)** - Quick solution for {specific problem}
