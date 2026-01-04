# Integration Examples

This directory contains complete integration examples combining multiple Beluga AI Framework components.

## Examples

### [full_stack](full_stack/)

Complete application example combining agents, RAG, memory, and orchestration.

**What you'll learn:**
- Full-stack integration
- Component coordination
- Production-ready patterns
- Best practices

**Run:**
```bash
cd full_stack
go run main.go
```

### [observability](observability/)

OTEL integration example with metrics and tracing.

**What you'll learn:**
- OpenTelemetry setup
- Metrics collection
- Distributed tracing
- Monitoring patterns

**Run:**
```bash
cd observability
go run main.go
```

## Prerequisites

- Go 1.21 or later
- Beluga AI Framework
- (Optional) OpenAI API key
- (Optional) OpenTelemetry collector for production

## Related Documentation

- [Best Practices](../../docs/BEST_PRACTICES.md)
- [Production Deployment](../../docs/getting-started/07-production-deployment.md)
- [Monitoring](../../pkg/monitoring/README.md)
