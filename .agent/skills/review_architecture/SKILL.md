---
name: Review Architecture
description: Architecture review checklist and pattern compliance verification
personas:
  - architect
---

# Review Architecture

This skill provides a systematic approach to reviewing code for architectural compliance with Beluga AI patterns and principles.

## Prerequisites

- Code or design to review
- Understanding of target functionality
- Access to relevant packages

## Review Process

### 1. Identify Scope

Determine what's being reviewed:

- [ ] New package
- [ ] New provider
- [ ] Feature addition
- [ ] Refactoring
- [ ] Bug fix

### 2. Layer Compliance Check

Verify dependencies follow the architecture:

```
Application Layer (cmd/, examples/)
       ↓
Agent Layer (pkg/agents/, pkg/orchestration/)
       ↓
LLM Layer (pkg/llms/, pkg/chatmodels/)
       ↓
RAG Layer (pkg/retrievers/, pkg/vectorstores/, pkg/embeddings/)
       ↓
Memory Layer (pkg/memory/)
       ↓
Infrastructure Layer (pkg/core/, pkg/config/, pkg/monitoring/, pkg/schema/)
```

**Check for violations**:
```bash
# Look for upward dependencies
grep -r "import.*pkg/agents" pkg/llms/       # Should be empty
grep -r "import.*pkg/llms" pkg/schema/       # Should be empty
```

### 3. Interface Segregation (ISP) Check

| Criteria | Pass | Fail |
|----------|------|------|
| Interfaces have ≤5 methods | [ ] | [ ] |
| Single-method interfaces use `-er` suffix | [ ] | [ ] |
| No "god interfaces" | [ ] | [ ] |
| Clients use only what they need | [ ] | [ ] |

**Red Flags**:
```go
// BAD: Too many methods
type Store interface {
    Get()
    Set()
    Delete()
    List()
    Watch()
    Backup()
    Restore()
    Compact()
    // ... more
}
```

### 4. Dependency Inversion (DIP) Check

| Criteria | Pass | Fail |
|----------|------|------|
| Dependencies are interfaces | [ ] | [ ] |
| Constructor injection used | [ ] | [ ] |
| No `init()` with side effects | [ ] | [ ] |
| No package-level mutable state | [ ] | [ ] |

**Red Flags**:
```go
// BAD: Concrete dependency
type Agent struct {
    client *openai.Client  // Should be interface
}

// BAD: Global state
var defaultClient *http.Client
```

### 5. Single Responsibility (SRP) Check

| Criteria | Pass | Fail |
|----------|------|------|
| Package has one clear purpose | [ ] | [ ] |
| Types have one reason to change | [ ] | [ ] |
| Methods do one thing | [ ] | [ ] |

**Red Flags**:
- Package name ends in "utils" or "helpers"
- Type has both business logic and I/O
- Method longer than 50 lines

### 6. Package Structure Check

Required files for each package:

| File | Required | Present |
|------|----------|---------|
| `iface/` directory | Multi-type | [ ] |
| `config.go` | Yes | [ ] |
| `metrics.go` | Yes | [ ] |
| `errors.go` | Yes | [ ] |
| `test_utils.go` | Yes | [ ] |
| `advanced_test.go` | Yes | [ ] |
| `README.md` | Yes | [ ] |

### 7. Configuration Check

| Criteria | Pass | Fail |
|----------|------|------|
| Config struct in `config.go` | [ ] | [ ] |
| `mapstructure` tags present | [ ] | [ ] |
| `validate` tags for required fields | [ ] | [ ] |
| `Validate()` method exists | [ ] | [ ] |
| Functional options for runtime config | [ ] | [ ] |

### 8. Error Handling Check

| Criteria | Pass | Fail |
|----------|------|------|
| Custom Error type with Op/Err/Code | [ ] | [ ] |
| Error codes defined as constants | [ ] | [ ] |
| Errors wrap underlying errors | [ ] | [ ] |
| Context cancellation respected | [ ] | [ ] |

### 9. Observability Check

| Criteria | Pass | Fail |
|----------|------|------|
| `metrics.go` exists | [ ] | [ ] |
| `NewMetrics(meter, tracer)` function | [ ] | [ ] |
| `{pkg}_operations_total` counter | [ ] | [ ] |
| `{pkg}_operation_duration_seconds` histogram | [ ] | [ ] |
| `{pkg}_errors_total` counter | [ ] | [ ] |
| Tracing spans on public methods | [ ] | [ ] |
| Structured logging with trace context | [ ] | [ ] |

### 10. Provider Pattern Check (Multi-Provider Packages)

| Criteria | Pass | Fail |
|----------|------|------|
| `registry.go` exists | [ ] | [ ] |
| `RegisterGlobal()` function | [ ] | [ ] |
| `NewProvider()` function | [ ] | [ ] |
| Providers in `providers/` subdirectory | [ ] | [ ] |
| Each provider has `config.go` | [ ] | [ ] |
| Each provider has tests | [ ] | [ ] |

## Review Output Template

```markdown
## Architecture Review: [Component/PR Name]

### Summary
[Overall assessment: APPROVED / NEEDS CHANGES / REJECTED]

### Compliance Matrix

| Principle | Status | Notes |
|-----------|--------|-------|
| Layer Compliance | ✅/❌ | |
| ISP | ✅/❌ | |
| DIP | ✅/❌ | |
| SRP | ✅/❌ | |
| Package Structure | ✅/❌ | |
| Configuration | ✅/❌ | |
| Error Handling | ✅/❌ | |
| Observability | ✅/❌ | |

### Issues Found

#### Critical
1. [Issue]: [Location] - [Fix required]

#### Major
1. [Issue]: [Location] - [Recommendation]

#### Minor
1. [Issue]: [Location] - [Suggestion]

### Recommendations
1. [Specific recommendation]

### Positive Highlights
- [Good pattern usage]
```

## Severity Definitions

| Severity | Definition | Action |
|----------|------------|--------|
| Critical | Violates core architecture | Must fix before merge |
| Major | Deviates from patterns | Should fix before merge |
| Minor | Style or optimization | Can fix later |

## Quick Reference: Anti-Patterns

1. **Circular Dependencies**: A imports B imports A
2. **God Objects**: Single type with many responsibilities
3. **Service Locator**: Runtime dependency lookup
4. **Leaky Abstractions**: Implementation details in interfaces
5. **Global State**: Mutable package-level variables
6. **Deep Inheritance**: Use composition instead
7. **Missing Observability**: No metrics or tracing

## Output

A comprehensive review document with:
- Compliance assessment
- Issues categorized by severity
- Specific fix recommendations
- Positive pattern highlights
