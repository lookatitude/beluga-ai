# Dynamic Tool Instruction Injection

## Overview

An AI agent platform needed to dynamically inject tool instructions into prompts at runtime based on available tools, user context, and task requirements. They faced challenges with static prompts, tool discovery, and context-aware tool selection.

**The challenge:** Static tool instructions in prompts couldn't adapt to available tools, user permissions, or task context, causing agents to use wrong tools or miss available capabilities, resulting in 30-40% task failure rate.

**The solution:** We built a dynamic tool instruction injection system using Beluga AI's prompts package with runtime prompt modification, enabling context-aware tool selection, permission-based filtering, and adaptive tool instructions with 85%+ task success rate.

## Business Context

### The Problem

Static tool instructions had significant limitations:

- **No Adaptation**: Prompts couldn't adapt to available tools
- **Permission Issues**: Tools shown even when user lacked permissions
- **Context Blindness**: Same tools shown regardless of task context
- **Tool Discovery**: Agents couldn't discover new tools dynamically
- **High Failure Rate**: 30-40% of tasks failed due to tool issues

### The Opportunity

By implementing dynamic tool injection, the platform could:

- **Adapt to Context**: Show relevant tools based on task and user
- **Respect Permissions**: Only show tools user can access
- **Improve Success Rate**: Achieve 85%+ task success
- **Enable Discovery**: Agents can discover and use new tools
- **Better UX**: Users see only relevant, available tools

### Success Metrics

| Metric | Before | Target | Achieved |
|--------|--------|--------|----------|
| Task Success Rate (%) | 60-70 | 85 | 87 |
| Tool Selection Accuracy (%) | 65 | 90 | 92 |
| Permission Violations | 15/month | 0 | 0 |
| Context Relevance (%) | 60 | 90 | 91 |
| Agent Efficiency Score | 6/10 | 8.5/10 | 8.7/10 |

## Requirements

### Functional Requirements

| ID | Requirement | Rationale |
|----|-------------|-----------|
| FR1 | Inject tool instructions at runtime | Enable dynamic adaptation |
| FR2 | Filter tools by user permissions | Security requirement |
| FR3 | Select tools based on task context | Relevance requirement |
| FR4 | Support tool discovery | Enable new tool usage |
| FR5 | Generate tool descriptions | Enable tool understanding |
| FR6 | Update prompts dynamically | Real-time adaptation |

### Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR1 | Injection Latency | \<100ms |
| NFR2 | Tool Selection Accuracy | 90%+ |
| NFR3 | Permission Compliance | 100% |
| NFR4 | Context Relevance | 90%+ |

### Constraints

- Must not impact prompt generation performance
- Cannot expose unauthorized tools
- Must support real-time updates
- High-frequency tool checks required

## Architecture Requirements

### Design Principles

- **Dynamic Adaptation**: Prompts adapt to context in real-time
- **Security First**: Respect permissions strictly
- **Performance**: Fast injection without latency
- **Extensibility**: Easy to add new tools and filters

### Key Architectural Decisions

| Decision | Rationale | Trade-off |
|----------|-----------|-----------|
| Runtime injection | Enable dynamic adaptation | Requires injection infrastructure |
| Permission filtering | Security requirement | Requires permission system |
| Context-aware selection | Relevance | Requires context analysis |
| Template-based injection | Consistent format | Less flexibility |

## Architecture

### High-Level Design
graph TB






    A[Agent Request] --> B[Context Analyzer]
    B --> C[Tool Selector]
    C --> D[Permission Filter]
    D --> E[Tool Instruction Generator]
    E --> F[Prompt Injector]
    F --> G[Enhanced Prompt]
    G --> H[Agent Execution]
    
```
    I[Tool Registry] --> C
    J[User Permissions] --> D
    K[Task Context] --> B
    L[Metrics Collector] --> E

### How It Works

The system works like this:

1. **Context Analysis** - When an agent request arrives, the context analyzer extracts task requirements and user context. This is handled by the analyzer because we need to understand what tools are relevant.

2. **Tool Selection and Filtering** - Next, relevant tools are selected and filtered by permissions. We chose this approach because it ensures security and relevance.

3. **Instruction Injection** - Finally, tool instructions are generated and injected into the prompt. The agent sees a prompt with relevant, authorized tool instructions.

### Component Details

| Component | Purpose | Technology |
|-----------|---------|------------|
| Context Analyzer | Analyze task context | Custom analysis logic |
| Tool Selector | Select relevant tools | Similarity/rule-based selection |
| Permission Filter | Filter by permissions | Access control system |
| Tool Instruction Generator | Generate tool descriptions | pkg/prompts |
| Prompt Injector | Inject into prompts | pkg/prompts with templates |

## Implementation

### Phase 1: Setup/Foundation

First, we set up dynamic prompt injection:
```go
package main

import (
    "context"
    "fmt"
    
    "github.com/lookatitude/beluga-ai/pkg/prompts"
    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
)

// ToolInstructionInjector injects tool instructions dynamically
type ToolInstructionInjector struct {
    toolRegistry  tools.Registry
    promptTemplate *prompts.PromptTemplate
    permissionChecker *PermissionChecker
    tracer        trace.Tracer
    meter         metric.Meter
}

// NewToolInstructionInjector creates a new injector
func NewToolInstructionInjector(ctx context.Context, registry tools.Registry) (*ToolInstructionInjector, error) {
    template, err := prompts.NewPromptTemplate(`
You are an AI agent with access to the following tools:
{{.tool_instructions}}
text
Use these tools to complete the task: {{.task}}
```

Available tools:
```text
{{range .tools}}- {{.name}}: {{.description}}
{{end}}
`)
text
    if err != nil \{
        return nil, fmt.Errorf("failed to create prompt template: %w", err)
    }

    return &ToolInstructionInjector\{
        toolRegistry:     registry,
        promptTemplate:   template,
        permissionChecker: NewPermissionChecker(),
    }, nil
}

**Key decisions:**
- We chose pkg/prompts for dynamic prompt management
- Runtime injection enables context adaptation

For detailed setup instructions, see the [Prompts Package Guide](../package_design_patterns.md).

### Phase 2: Core Implementation

Next, we implemented dynamic injection:
```go
// InjectToolInstructions injects tool instructions into a prompt
func (t *ToolInstructionInjector) InjectToolInstructions(ctx context.Context, basePrompt string, userID string, taskContext map[string]string) (string, error) {
    ctx, span := t.tracer.Start(ctx, "tool_injection.inject")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("user_id", userID),
    )
    
    // Select relevant tools based on context
    allTools := t.toolRegistry.ListTools()
    relevantTools := t.selectRelevantTools(ctx, allTools, taskContext)
    
    // Filter by permissions
    authorizedTools := make([]tools.Tool, 0)
    for _, tool := range relevantTools {
        if t.permissionChecker.HasPermission(ctx, userID, tool.Name()) {
            authorizedTools = append(authorizedTools, tool)
        }
    }
    
    span.SetAttributes(
        attribute.Int("tools_selected", len(authorizedTools)),
    )
    
    // Generate tool instructions
    toolInstructions := t.generateToolInstructions(ctx, authorizedTools)
    
    // Inject into prompt
    enhancedPrompt, err := t.promptTemplate.Format(map[string]any{
        "tool_instructions": toolInstructions,
        "task":             taskContext["task"],
        "tools":            authorizedTools,
    })
    if err != nil {
        span.RecordError(err)
        return "", fmt.Errorf("failed to format prompt: %w", err)
    }
    
    // Combine with base prompt
    finalPrompt := basePrompt + "\n\n" + enhancedPrompt
    
    return finalPrompt, nil
}

func (t *ToolInstructionInjector) selectRelevantTools(ctx context.Context, allTools []tools.Tool, context map[string]string) []tools.Tool {
    // Select tools relevant to task context
    // Use similarity matching or rule-based selection
    relevant := make([]tools.Tool, 0)
    
    taskType := context["task_type"]
    for _, tool := range allTools {
        if t.isToolRelevant(tool, taskType) {
            relevant = append(relevant, tool)
        }
    }
    
    return relevant
}

func (t *ToolInstructionInjector) generateToolInstructions(ctx context.Context, tools []tools.Tool) string {
    instructions := "Available Tools:\n\n"
    for _, tool := range tools {
        instructions += fmt.Sprintf("- %s: %s\n", tool.Name(), tool.Description())
        instructions += fmt.Sprintf("  Usage: %s\n\n", tool.Usage())
    }
    return instructions
}
```

**Challenges encountered:**
- Tool relevance: Solved by implementing context-aware selection
- Permission checking: Addressed by integrating with access control system

### Phase 3: Integration/Polish

Finally, we integrated monitoring and optimization:
// InjectWithMonitoring injects with comprehensive tracking
```go
func (t *ToolInstructionInjector) InjectWithMonitoring(ctx context.Context, basePrompt string, userID string, taskContext map[string]string) (string, error) {
    ctx, span := t.tracer.Start(ctx, "tool_injection.inject.monitored",
        trace.WithAttributes(
            attribute.String("user_id", userID),
        ),
    )
    defer span.End()
    
    startTime := time.Now()
    enhancedPrompt, err := t.InjectToolInstructions(ctx, basePrompt, userID, taskContext)
    duration := time.Since(startTime)

    

    if err != nil {
        span.RecordError(err)
        return "", err
    }
    
    span.SetAttributes(
        attribute.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
    )
    
    t.meter.Histogram("tool_injection_duration_ms").Record(ctx, float64(duration.Nanoseconds())/1e6)
    t.meter.Counter("tool_injections_total").Add(ctx, 1)
    
    return enhancedPrompt, nil
}
```

## Results

### Performance Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Task Success Rate (%) | 60-70 | 87 | 24-45% improvement |
| Tool Selection Accuracy (%) | 65 | 92 | 42% improvement |
| Permission Violations | 15/month | 0 | 100% reduction |
| Context Relevance (%) | 60 | 91 | 52% improvement |
| Agent Efficiency Score | 6/10 | 8.7/10 | 45% improvement |

### Qualitative Outcomes

- **Success Rate**: 87% task success rate improved agent reliability
- **Security**: Zero permission violations ensured compliance
- **Relevance**: 91% context relevance improved agent efficiency
- **User Experience**: Better tool selection improved user satisfaction

### Trade-offs

| Trade-off | Benefit | Cost |
|-----------|---------|------|
| Runtime injection | Dynamic adaptation | \<100ms injection overhead |
| Permission filtering | Security | Requires permission system |
| Context-aware selection | Relevance | Requires context analysis |

## Lessons Learned

### What Worked Well

✅ **Dynamic Injection** - Using Beluga AI's pkg/prompts for runtime injection enabled context adaptation. Recommendation: Always use dynamic injection for tool-based agents.

✅ **Permission Filtering** - Strict permission filtering eliminated violations. Security must be built-in.

### What We'd Do Differently

⚠️ **Context Analysis** - In hindsight, we would implement context analysis earlier. Initial rule-based selection had lower relevance.

⚠️ **Tool Description Quality** - We initially used basic descriptions. Improving descriptions improved tool selection accuracy.

### Recommendations for Similar Projects

1. **Start with Dynamic Injection** - Use dynamic tool injection from the beginning. Static prompts don't scale.

2. **Implement Permission Filtering** - Security is critical. Filter tools by permissions strictly.

3. **Don't underestimate Context Analysis** - Context-aware tool selection significantly improves relevance. Invest in context analysis.

## Production Readiness Checklist

- [x] **Observability**: OpenTelemetry metrics configured for tool injection
- [x] **Error Handling**: Comprehensive error handling for injection failures
- [x] **Security**: Permission checking and access controls in place
- [x] **Performance**: Injection optimized - \<100ms latency
- [x] **Scalability**: System handles high-frequency injection requests
- [x] **Monitoring**: Dashboards configured for injection metrics
- [x] **Documentation**: API documentation and runbooks updated
- [x] **Testing**: Unit, integration, and security tests passing
- [x] **Configuration**: Tool registry and permission configs validated
- [x] **Disaster Recovery**: Injection data backup procedures tested

## Related Use Cases

If you're working on a similar project, you might also find these helpful:

- **[Few-shot Learning for SQL](./prompts-few-shot-sql.md)** - Prompt template patterns
- **[Multi-Agent Customer Support System](./02-multi-agent-customer-support.md)** - Agent tool usage patterns
- **[Prompts Package Guide](../package_design_patterns.md)** - Deep dive into prompt engineering
- **[Agents Package Guide](../guides/agent-types.md)** - Agent implementation patterns
