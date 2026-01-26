# Workflow and Activity

**Workflow:** `Execute(ctx, input)→(workflowID, runID)`, `GetResult`, `Signal`, `Query`, `Cancel`, `Terminate`. For distributed, durable execution (e.g. Temporal).

**Activity:** `Execute(ctx, input) (any, error)`. May differ from `Runnable.Invoke`. Use to bridge Beluga components to workflow engines.

**CreateWorkflow(workflowFn any, opts):** `workflowFn` is `any` to support multiple signatures required by different backends (e.g. Temporal). Apply `WorkflowOption` to build `*WorkflowConfig`; pass `workflowFn` and config to the provider.

**WorkflowConfig:** Name, TaskQueue, Timeout, Retries, Container, Metadata. **Container** (e.g. `core.Container`) is optional by default; a backend may require it.

**Temporal:** `TemporalWorkflow` implements Workflow; needs `client.Client` and `workflowFn`. When the workflow backend is not configured, `CreateWorkflow` returns `ErrInvalidState`.
