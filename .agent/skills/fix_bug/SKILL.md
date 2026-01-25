---
name: Fix Bug
description: Bug investigation and fix workflow with root cause analysis
personas:
  - backend-developer
---

# Fix Bug

This skill guides you through investigating and fixing bugs in Beluga AI with proper root cause analysis and verification.

## Prerequisites

- Bug report or description of unexpected behavior
- Access to relevant logs/error messages
- Ability to reproduce the issue

## Steps

### 1. Understand the Bug

Gather information:

1. **Symptoms**: What is the observed behavior?
2. **Expected**: What should happen instead?
3. **Reproduction**: Steps to trigger the bug
4. **Frequency**: Always, intermittent, specific conditions?
5. **Impact**: Severity and affected functionality

### 2. Reproduce the Issue

Create a minimal reproduction:

```go
func TestBugReproduction(t *testing.T) {
    // Setup that triggers the bug
    component := NewComponent(config)

    // Action that causes the issue
    result, err := component.Process(ctx, input)

    // This assertion should fail (demonstrating the bug)
    require.NoError(t, err) // Currently fails
}
```

### 3. Investigate Root Cause

#### Trace the Code Path

1. Start from the entry point
2. Follow the execution flow
3. Identify where behavior diverges from expected

#### Check Common Causes

- [ ] Nil pointer dereference
- [ ] Race condition (concurrent access)
- [ ] Resource leak (goroutine, file handle)
- [ ] Incorrect error handling
- [ ] Missing validation
- [ ] Configuration issue
- [ ] State mutation
- [ ] Context cancellation not respected

#### Use Debugging Techniques

```bash
# Run with race detection
go test -race -v ./pkg/affected/...

# Add verbose logging temporarily
// In code:
log.Printf("DEBUG: value=%v, state=%v", value, state)

# Check OTEL traces if available
```

### 4. Document Root Cause

Before fixing, document:

```markdown
## Root Cause Analysis

**Bug**: [Brief description]

**Root Cause**: [Technical explanation]

**Location**: `pkg/example/file.go:42`

**Why It Happened**: [Context]

**Fix Approach**: [How to fix]
```

### 5. Implement the Fix

#### Write Test First (TDD)

```go
func TestBugFix_IssueXYZ(t *testing.T) {
    // Setup
    component := NewComponent(config)

    // Action
    result, err := component.Process(ctx, input)

    // Assertions that should pass after fix
    require.NoError(t, err)
    assert.Equal(t, expectedResult, result)
}
```

#### Apply Minimal Fix

- Fix only what's broken
- Don't refactor unrelated code
- Preserve existing behavior for non-bug cases
- Add defensive checks if appropriate

#### Example Fix Patterns

**Nil Check**:
```go
// Before
value := obj.Field.Method()

// After
if obj == nil || obj.Field == nil {
    return nil, &Error{Op: "Process", Code: ErrCodeInvalidInput}
}
value := obj.Field.Method()
```

**Race Condition**:
```go
// Before
c.state = newState

// After
c.mu.Lock()
c.state = newState
c.mu.Unlock()
```

**Context Cancellation**:
```go
// Before
for item := range items {
    process(item)
}

// After
for item := range items {
    select {
    case <-ctx.Done():
        return ctx.Err()
    default:
        process(item)
    }
}
```

### 6. Add Regression Test

Ensure the bug cannot recur:

```go
func TestRegression_IssueXYZ(t *testing.T) {
    // Specific conditions that caused the bug
    input := createBugTriggerInput()

    component := NewComponent(config)
    result, err := component.Process(ctx, input)

    // Bug is now fixed
    require.NoError(t, err)
    assert.Equal(t, expected, result)
}
```

### 7. Run Quality Checks

```bash
# Format and lint
make fmt
make lint

# Run all tests with race detection
make test-race

# Run affected package tests
go test -v -race ./pkg/affected/...

# Run integration tests if applicable
make test-integration
```

### 8. Verify Fix

- [ ] Original bug is fixed
- [ ] Regression test passes
- [ ] No new test failures
- [ ] Race detection passes
- [ ] Related functionality still works

### 9. Document the Fix

Add to commit message:

```
fix(package): brief description of fix

Root cause: [explanation]
Fix: [what was changed]

Fixes #123
```

## Validation Checklist

- [ ] Bug is reproducible before fix
- [ ] Root cause is understood and documented
- [ ] Fix is minimal and focused
- [ ] Regression test added
- [ ] All existing tests pass
- [ ] Race detection passes
- [ ] Related functionality verified
- [ ] Commit message explains the fix

## Common Bug Categories

| Category | Symptoms | Typical Fix |
|----------|----------|-------------|
| Nil pointer | Panic | Add nil checks |
| Race condition | Intermittent failures | Add mutex/atomic |
| Resource leak | Memory growth | Add defer cleanup |
| Error handling | Silent failures | Check and propagate errors |
| Context | Hanging operations | Respect ctx.Done() |
| Validation | Invalid state | Add input validation |

## Output

A verified bug fix with:
- Root cause documentation
- Regression test
- Minimal code change
- Passing quality checks
