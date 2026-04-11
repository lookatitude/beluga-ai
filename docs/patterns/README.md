# Patterns

Reusable patterns every Beluga package uses. If you can apply these eight patterns, you can extend any package in the framework.

## The eight patterns

| Pattern | Problem it solves | Where it's used |
|---|---|---|
| [Registry + Factory](./registry-factory.md) | Runtime discovery without import cycles | llm · tool · memory · rag/* · voice/* · guard · workflow · server · cache · auth · state |
| [Middleware Chain](./middleware-chain.md) | Cross-cutting concerns that compose | llm · tool · memory · rag/retriever · agent |
| [Lifecycle Hooks](./hooks-lifecycle.md) | Interception at specific execution points | tool · llm · memory · agent |
| [Streaming with iter.Seq2](./streaming-iter-seq2.md) | Type-safe streaming with backpressure | core · llm · agent · voice · tool |
| [Functional Options](./functional-options.md) | Configuration with defaults and growth | every constructor |
| [Provider Template](./provider-template.md) | Implementing a new provider | llm/providers · rag/embedding · rag/vectorstore · memory/stores · tool/builtin · voice/* |
| [Error Handling](./error-handling.md) | Typed errors with retry semantics | core · all providers |
| [Context Propagation](./context-propagation.md) | Tenant, session, auth, tracing | every public function |

## Pattern document structure

Each pattern document follows the same shape:

- **What it is** — one paragraph.
- **Why we use it** — the problem, what goes wrong without it, alternatives considered.
- **How it works** — canonical Go code with a `file:line` reference.
- **Where it's used** — table of packages that use this pattern.
- **Common mistakes** — sourced from [`../../.wiki/corrections.md`](../../.wiki/corrections.md).
- **Example: implementing your own** — compilable code showing how to apply the pattern.

## Cross-references

- Every pattern has a canonical example documented in [`.wiki/patterns/`](../../.wiki/patterns/) with `file:line` pointers.
- Architecture rationale lives in [`../architecture/03-extensibility-patterns.md`](../architecture/03-extensibility-patterns.md).
