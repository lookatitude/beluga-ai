# Memory as Single Interface

Use a single **Memory** interface as the public contract. Callers and higher-level APIs depend on **Memory** only, not on ChatMessageHistory.

- **Memory:** `MemoryVariables()`, `LoadMemoryVariables(ctx, inputs)`, `SaveContext(ctx, inputs, outputs)`, `Clear(ctx)`. This is the only interface exposed to external code and higher-level components.
- **ChatMessageHistory (if present):** Treated as an internal or storage concern. Memory implementations may use it as a backend, but it is not part of the public API. New code must not depend on ChatMessageHistory.
- **Always use Memory** in agents, chains, and application code. Do not accept or return ChatMessageHistory in public functions.
