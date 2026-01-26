# Graph: DAG, Entry/Exit

**AddNode:** Reject duplicate names (`ErrInvalidConfig` or `ErrNotFound` as appropriate). **AddEdge:** Require both source and target to exist; otherwise `ErrNotFound`.

**Entry and exit:** By default infer: entry = nodes with no incoming edges; exit = nodes with no outgoing edges. Allow explicit `SetEntryPoint` and `SetFinishPoint` to override. Validate that all named nodes exist.

**Invoke:** Execute in dependency order (topological). Use a max-iterations guard for cycle detection; on exceed return `ErrExecutionFailed` with a message like "maximum iterations exceeded, possible cycle". Single exit → return that node's output; multiple exits → `map[exitNode]output`.

**EnableParallelExecution:** When true, prefer real parallel/DAG execution (e.g. run ready nodes concurrently). When that's not possible, fall back to sequential and document the behavior.
