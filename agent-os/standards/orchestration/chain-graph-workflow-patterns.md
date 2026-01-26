# Chain, Graph, Workflow as Patterns

**Chain** and **Graph** implement `core.Runnable` (Invoke, Batch, Stream). **Workflow** does not: it is long-running and async; it has `Execute`, `GetResult`, `Signal`, `Query`, `Cancel`, `Terminate`.

**Chain:** `GetInputKeys`, `GetOutputKeys`, `GetMemory`. Steps are `core.Runnable`; output of one is input to the next.

**Graph:** `AddNode(name, runnable)`, `AddEdge(source, target)`, `SetEntryPoint`, `SetFinishPoint`. Nodes are `core.Runnable`; edges form a DAG.

**Workflow:** `Execute(ctx, input)→(workflowID, runID)`, `GetResult`, `Signal`, `Query`, `Cancel`, `Terminate`. For distributed, durable execution (e.g. Temporal).

**Activity:** `Execute(ctx, input) (any, error)`. May differ from `Runnable.Invoke` (e.g. input/output types). Use to bridge Beluga components to workflow systems.
