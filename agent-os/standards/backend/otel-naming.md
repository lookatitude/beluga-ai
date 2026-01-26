# OTEL Naming (Metrics and Spans)

**Metrics:** Use `package.thing` with dots. Examples: `llm.requests.total`, `orchestration.chain.executions.total`. Add `.total`, `.seconds`, etc. as needed. Be consistent within a package.

**Spans:** Use `package.Operation` or `package.sub.Operation`. Operation is CamelCase. Examples: `orchestrator.CreateChain`, `textsplitters.markdown.SplitText`.

**Attributes:** Use `attribute.String`, `attribute.Int`, `attribute.Bool`, etc. with `attribute.Key` in snake_case when custom (e.g. `attribute.String("chain_name", chainName)`).
