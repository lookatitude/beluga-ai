# DevSecOps Policy Auditor

## Overview

A financial services company needed to automate security and compliance policy auditing across their infrastructure, codebase, and deployments. They faced challenges with manual audits, inconsistent policy enforcement, and inability to detect violations in real-time.

**The challenge:** Manual policy audits took 2-3 weeks, missed 20-30% of violations, and couldn't keep up with rapid infrastructure changes, causing compliance risks and security vulnerabilities.

**The solution:** We built a DevSecOps policy auditor using Beluga AI's agents package with specialized tools for infrastructure scanning, code analysis, and compliance checking, enabling automated, real-time policy auditing with 95%+ violation detection and continuous compliance monitoring.

## Business Context

### The Problem

Policy auditing had significant gaps:

- **Manual Process**: 2-3 week audit cycles
- **Missed Violations**: 20-30% of violations undetected
- **Slow Response**: Violations discovered weeks after introduction
- **Inconsistent Enforcement**: Different standards across teams
- **Compliance Risk**: Regulatory violations due to missed issues

### The Opportunity

By implementing automated policy auditing, the company could:

- **Automate Audits**: Reduce audit time from weeks to hours
- **Improve Detection**: Achieve 95%+ violation detection
- **Real-time Monitoring**: Detect violations immediately
- **Consistent Enforcement**: Uniform policy application
- **Reduce Risk**: Prevent compliance violations proactively

### Success Metrics

| Metric | Before | Target | Achieved |
|--------|--------|--------|----------|
| Audit Cycle Time (weeks) | 2-3 | \<0.5 | 0.3 |
| Violation Detection Rate (%) | 70-80 | 95 | 96 |
| Time to Detect Violations | Weeks | \<1 hour | 0.5 hours |
| Compliance Score | 75 | 95 | 96 |
| Security Incidents Prevented | N/A | 10+/month | 12/month |
| Policy Enforcement Consistency (%) | 60 | 95 | 97 |

## Requirements

### Functional Requirements

| ID | Requirement | Rationale |
|----|-------------|-----------|
| FR1 | Scan infrastructure for policy violations | Detect infrastructure issues |
| FR2 | Analyze code for security vulnerabilities | Detect code-level issues |
| FR3 | Check deployment configurations | Ensure deployment compliance |
| FR4 | Generate compliance reports | Enable reporting and tracking |
| FR5 | Alert on violations | Immediate notification |
| FR6 | Track violation remediation | Monitor fix progress |

### Non-Functional Requirements

| ID | Requirement | Target |
|----|-------------|--------|
| NFR1 | Audit Execution Time | \<1 hour for full audit |
| NFR2 | Violation Detection Rate | 95%+ |
| NFR3 | Real-time Detection | \<1 hour from violation |
| NFR4 | System Availability | 99.9% uptime |

### Constraints

- Must not impact production systems
- Cannot modify infrastructure or code
- Must comply with security standards
- High-frequency scanning required

## Architecture Requirements

### Design Principles

- **Automation First**: Minimize manual intervention
- **Security**: Secure scanning and reporting
- **Performance**: Fast audit execution
- **Comprehensiveness**: Cover all policy areas

### Key Architectural Decisions

| Decision | Rationale | Trade-off |
|----------|-----------|-----------|
| Agent-based auditing | Reasoning and action capability | Requires tool integration |
| Specialized tools | Domain-specific scanning | Requires tool development |
| Continuous monitoring | Real-time detection | Requires monitoring infrastructure |
| Policy as code | Version-controlled policies | Requires policy management |

## Architecture

### High-Level Design

```mermaid
graph TB
    A[Policy Definitions] --> B[Policy Auditor Agent]
    B --> C[Infrastructure Scanner]
    B --> D[Code Analyzer]
    B --> E[Config Checker]
    C --> F[Violation Detector]
    D --> F
    E --> F
    F --> G{Violations Found?}
    G -->|Yes| H[Alert System]
    G -->|No| I[Compliance Report]
    H --> J[Remediation Tracker]
    K[Tool Registry] --> B
    L[Metrics Collector] --> B
    M[Policy Engine] --> B
```

### How It Works

The system works like this:

1. **Policy Loading** - When an audit starts, policies are loaded and the agent plans the audit. This is handled by the agent because we need reasoning for complex audits.

2. **Tool Execution** - Next, the agent uses specialized tools to scan infrastructure, analyze code, and check configurations. We chose this approach because tools enable comprehensive scanning.

3. **Violation Detection and Reporting** - Finally, violations are detected, alerts are sent, and reports are generated. The user sees comprehensive compliance status and violation details.

### Component Details

| Component | Purpose | Technology |
|-----------|---------|------------|
| Policy Auditor Agent | Orchestrate audits | pkg/agents (ReAct) |
| Infrastructure Scanner | Scan infrastructure | Custom tool |
| Code Analyzer | Analyze code | Custom tool with static analysis |
| Config Checker | Check configurations | Custom tool |
| Violation Detector | Detect policy violations | Custom logic |
| Alert System | Send violation alerts | pkg/monitoring |

## Implementation

### Phase 1: Setup/Foundation

First, we set up the policy auditor agent:
```go
package main

import (
    "context"
    "fmt"
    
    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
    "github.com/lookatitude/beluga-ai/pkg/llms"
)

// PolicyAuditorAgent implements automated policy auditing
type PolicyAuditorAgent struct {
    agent        agents.Agent
    tools        tools.Registry
    policyEngine *PolicyEngine
    tracer       trace.Tracer
    meter        metric.Meter
}

// NewPolicyAuditorAgent creates a new policy auditor
func NewPolicyAuditorAgent(ctx context.Context, llm llms.ChatModel) (*PolicyAuditorAgent, error) {
    toolRegistry := tools.NewInMemoryToolRegistry()
    
    // Infrastructure scanning tool
    infraTool, _ := tools.NewGoFunctionTool("scan_infrastructure", "Scan infrastructure for policy violations", "", scanInfrastructure)
    toolRegistry.RegisterTool(infraTool)
    
    // Code analysis tool
    codeTool, _ := tools.NewGoFunctionTool("analyze_code", "Analyze code for security issues", "", analyzeCode)
    toolRegistry.RegisterTool(codeTool)
    
    // Config checking tool
    configTool, _ := tools.NewGoFunctionTool("check_config", "Check deployment configurations", "", checkConfig)
    toolRegistry.RegisterTool(configTool)
    
    // Create ReAct agent
    agent, err := agents.NewReActAgent(
        agents.WithName("policy-auditor"),
        agents.WithDescription("DevSecOps policy auditor that scans infrastructure, code, and configurations for policy violations"),
        agents.WithLLM(llm),
        agents.WithTools(toolRegistry),
        agents.WithMaxIterations(15),
    )
    if err != nil {
        return nil, fmt.Errorf("failed to create agent: %w", err)
    }

    
    return &PolicyAuditorAgent{
        agent:        agent,
        tools:        toolRegistry,
        policyEngine: NewPolicyEngine(),
    }, nil
}
```

**Key decisions:**
- We chose ReAct agents for complex audit reasoning
- Specialized tools enable comprehensive scanning

For detailed setup instructions, see the [Agents Package Guide](../guides/agent-types.md).

### Phase 2: Core Implementation

Next, we implemented audit execution:
```go
// RunAudit executes a comprehensive policy audit
func (p *PolicyAuditorAgent) RunAudit(ctx context.Context, scope AuditScope) (*AuditReport, error) {
    ctx, span := p.tracer.Start(ctx, "policy_audit.run")
    defer span.End()
    
    span.SetAttributes(
        attribute.String("scope", scope.String()),
    )
    
    // Load policies
    policies, err := p.policyEngine.LoadPolicies(ctx, scope)
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("failed to load policies: %w", err)
    }
    
    // Build audit plan
    auditPlan := p.buildAuditPlan(ctx, policies, scope)
    
    // Execute audit using agent
    prompt := fmt.Sprintf("Perform a comprehensive policy audit for the following scope: %s\n\nPolicies to check:\n%s\n\nUse available tools to scan infrastructure, analyze code, and check configurations.\nReport all violations found.", scope.String(), formatPolicies(policies))

    result, err := p.agent.Execute(ctx, map[string]any{
        "task":  prompt,
        "scope": scope,
    })
    if err != nil {
        span.RecordError(err)
        return nil, fmt.Errorf("audit execution failed: %w", err)
    }

    // Parse violations from result
    violations := p.parseViolations(result)

    // Generate report
    report := &AuditReport{
        Scope:      scope,
        Violations: violations,
        Compliance: p.calculateCompliance(violations, policies),
        Timestamp:  time.Now(),
    }

    // Send alerts for critical violations
    p.alertViolations(ctx, violations)

    return report, nil
}
```

**Challenges encountered:**
- Policy complexity: Solved by implementing policy engine with rule evaluation
- Tool reliability: Addressed by implementing tool health checks and retries

### Phase 3: Integration/Polish

Finally, we integrated monitoring and reporting:

```go
// RunAuditWithMonitoring executes audit with comprehensive tracking
func (p *PolicyAuditorAgent) RunAuditWithMonitoring(ctx context.Context, scope AuditScope) (*AuditReport, error) {
    ctx, span := p.tracer.Start(ctx, "policy_audit.run.monitored",
        trace.WithAttributes(
            attribute.String("scope", scope.String()),
        ),
    )
    defer span.End()
    
    startTime := time.Now()
    report, err := p.RunAudit(ctx, scope)
    duration := time.Since(startTime)

    

    if err != nil {
        span.RecordError(err)
        return nil, err
    }
    
    span.SetAttributes(
        attribute.Int("violations_count", len(report.Violations)),
        attribute.Float64("compliance_score", report.Compliance),
        attribute.Float64("duration_ms", float64(duration.Nanoseconds())/1e6),
    )
    
    p.meter.Histogram("policy_audit_duration_ms").Record(ctx, float64(duration.Nanoseconds())/1e6)
    p.meter.Counter("policy_audits_total").Add(ctx, 1)
    p.meter.Counter("policy_violations_total").Add(ctx, int64(len(report.Violations)))
    
    return report, nil
}
```

## Results

### Performance Metrics

| Metric | Before | After | Improvement |
|--------|--------|-------|-------------|
| Audit Cycle Time (weeks) | 2-3 | 0.3 | 90-93% reduction |
| Violation Detection Rate (%) | 70-80 | 96 | 20-37% improvement |
| Time to Detect Violations | Weeks | 0.5 hours | 99%+ reduction |
| Compliance Score | 75 | 96 | 28% improvement |
| Security Incidents Prevented | N/A | 12/month | New capability |
| Policy Enforcement Consistency (%) | 60 | 97 | 62% improvement |

### Qualitative Outcomes

- **Efficiency**: 90-93% reduction in audit time improved responsiveness
- **Detection**: 96% violation detection rate improved security
- **Real-time**: 0.5 hour detection time enabled proactive remediation
- **Compliance**: 96% compliance score reduced regulatory risk

### Trade-offs

| Trade-off | Benefit | Cost |
|-----------|---------|------|
| Agent-based auditing | Reasoning capability | Requires tool integration |
| Specialized tools | Comprehensive scanning | Requires tool development |
| Continuous monitoring | Real-time detection | Requires monitoring infrastructure |

## Lessons Learned

### What Worked Well

✅ **ReAct Agents** - Using Beluga AI's agents package with ReAct provided reasoning capability for complex audits. Recommendation: Always use ReAct agents for complex automation tasks.

✅ **Specialized Tools** - Domain-specific tools enabled comprehensive scanning. Tool quality is critical.

### What We'd Do Differently

⚠️ **Policy Engine** - In hindsight, we would implement the policy engine earlier. Initial hardcoded policies were inflexible.

⚠️ **Tool Development** - We initially used generic tools. Specialized tools significantly improved detection accuracy.

### Recommendations for Similar Projects

1. **Start with ReAct Agents** - Use ReAct agents from the beginning. They provide reasoning capability for complex tasks.

2. **Invest in Tools** - Specialized tools are critical. Invest in domain-specific tool development.

3. **Don't underestimate Policy Management** - Policy as code enables version control and consistency. Implement policy engine early.

## Production Readiness Checklist

- [x] **Observability**: OpenTelemetry metrics configured for auditing
- [x] **Error Handling**: Comprehensive error handling for audit failures
- [x] **Security**: Secure scanning and access controls in place
- [x] **Performance**: Audit optimized - \<1 hour execution time
- [x] **Scalability**: System handles large-scale audits
- [x] **Monitoring**: Dashboards configured for audit metrics
- [x] **Documentation**: API documentation and runbooks updated
- [x] **Testing**: Unit, integration, and security tests passing
- [x] **Configuration**: Policy and tool configs validated
- [x] **Disaster Recovery**: Audit data backup procedures tested

## Related Use Cases

If you're working on a similar project, you might also find these helpful:

- **[Autonomous Customer Support](./agents-autonomous-support.md)** - Agent-based automation patterns
- **[Automated Code Review System](./06-automated-code-review-system.md)** - Code analysis patterns
- **[Agents Package Guide](../guides/agent-types.md)** - Deep dive into agent patterns
- **[Tools Package Guide](../package_design_patterns.md)** - Tool integration patterns
