# Orchestrator and Feature Flags

**Orchestrator:** `CreateChain(steps, opts)`, `CreateGraph(opts)`, `CreateWorkflow(workflowFn, opts)`, `GetMetrics`.

**Config.Enabled:** Chains, Graphs, Workflows, Scheduler, MessageBus. Use to ensure consistency and to enable or disable patterns and integrations (e.g. Temporal). When a feature is disabled, the corresponding `Create*` returns `ErrInvalidState(op, "chains_disabled", "chains_enabled")` (or `"graphs_disabled"`, `"workflows_disabled"`). When the workflow backend (e.g. Temporal) is not configured, `CreateWorkflow` returns `ErrInvalidState(op, "temporal_client_not_configured", "temporal_client_required")`.

**Options:** Apply `ChainOption`/`GraphOption`/`WorkflowOption` to build `*ChainConfig`, `*GraphConfig`, `*WorkflowConfig`; then delegate to providers (e.g. `chain.NewSimpleChain`, `graph.NewBasicGraph`, Temporal or error).
