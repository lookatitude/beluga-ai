# Safety as Opt-In Middleware

**Opt-in:** `WithEnableSafety(true)`. After constructing the agent, if `options.EnableSafety` then wrap with `newSafetyMiddleware(agent)`. Applied in `NewBaseAgent`, `NewReActAgent`, and `AgentFactory` `Create*` methods.

- **Why opt-in:** Performance, custom checkers, or environments where safety is not needed; use of this package is not mandatory.
- **Scope:** Middleware may wrap the whole agent. The design allows running safety on **specific operations** (e.g. only `Execute`, only `Plan`, or specific tool calls) when implemented. Prefer operation-specific hooks over a single global wrap when fine-grained control is required.
