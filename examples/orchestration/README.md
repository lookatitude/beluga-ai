# Orchestration Examples

This directory contains examples demonstrating workflow and chain orchestration in the Beluga AI Framework.

## Examples

### [chain](chain/)

Simple chain creation and sequential execution.

**What you'll learn:**
- Creating chains
- Sequential step execution
- Data passing between steps
- Batch processing

**Run:**
```bash
cd chain
go run main.go
```

### [workflow](workflow/)

Workflow creation with conditional branching and parallel execution.

**What you'll learn:**
- Workflow definition
- Conditional branching
- Parallel execution patterns
- Workflow configuration

**Run:**
```bash
cd workflow
go run main.go
```

### [multi_agent](multi_agent/)

Multi-agent coordination using message bus and scheduler.

**What you'll learn:**
- Agent coordination
- Message passing
- Task distribution
- Agent communication patterns

**Run:**
```bash
cd multi_agent
go run main.go
```

## Prerequisites

- Go 1.21 or later
- Beluga AI Framework
- (Optional) OpenAI API key for LLM providers

## Learning Path

1. Start with `chain` for basic orchestration
2. Try `workflow` for complex workflows
3. Explore `multi_agent` for coordination

## Related Documentation

- [Orchestration Concepts](../../docs/concepts/orchestration.md)
- [Architecture Documentation](../../docs/architecture.md)
