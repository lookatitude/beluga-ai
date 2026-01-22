# Long-running Workflows with Temporal

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this tutorial, we'll integrate Beluga AI with Temporal to create durable, long-running agent workflows. You'll build a workflow that can survive process crashes, handle multi-day waits, and incorporate human-in-the-loop approvals.

## Learning Objectives
- ✅ Understand Durable Execution
- ✅ Setup Temporal Worker
- ✅ Create a Beluga Workflow Activity
- ✅ Handle long-running human-in-the-loop steps

## Introduction
Welcome, colleague! Standard agents run in memory—if your server reboots, the agent's state is gone. For workflows that take hours or days, we need something more robust. Let's use Temporal to give our agents "save game" capabilities.

## Prerequisites

- [Orchestration Basics](../../getting-started/06-orchestration-basics.md)
- Temporal Server running (local or cloud)

## Why Temporal?

Standard agents run in memory. If you restart the server, the agent dies. Temporal persists the state of execution.
- **Sleep for days**: "Remind user in 3 days".
- **Reliable retries**: "API down? Retry in 1h".
- **Human signals**: "Wait for manager approval".

## Step 1: Define Activities

Wrap Beluga agents/chains as Temporal Activities.
```go
func AgentActivity(ctx context.Context, input string) (string, error) {
    // Standard Beluga code inside
    agent := createAgent()
    res, err := agent.Invoke(context.Background(), input) // Don't use Temporal context for HTTP
    return res.(string), err
}

## Step 2: Define Workflow
func ResearchWorkflow(ctx workflow.Context, topic string) (string, error) {
    options := workflow.ActivityOptions{
        StartToCloseTimeout: time.Minute * 10,
    }
    ctx = workflow.WithActivityOptions(ctx, options)

    var result string
    err := workflow.ExecuteActivity(ctx, AgentActivity, topic).Get(ctx, &result)

    
    return result, err
}
```

## Step 3: Human-in-the-Loop
```go
// Wait for signal
var approval bool
signalChan := workflow.GetSignalChannel(ctx, "approval_signal")
selector := workflow.NewSelector(ctx)
selector.AddReceive(signalChan, func(c workflow.ReceiveChannel, _ bool) {
    c.Receive(ctx, &approval)
```
})






// Block until signal received
selector.Select(ctx)

```
if approval \{
    // Continue...
}

## Step 4: The Worker
```go
func main() {
    c, _ := client.Dial(client.Options{})
    w := worker.New(c, "beluga-tasks", worker.Options{})

    

    w.RegisterWorkflow(ResearchWorkflow)
    w.RegisterActivity(AgentActivity)
    
    w.Run(worker.InterruptCh())
}
```

## Verification

1. Start the worker.
2. Start a workflow via CLI or UI.
3. Kill the worker process halfway through.
4. Restart the worker.
5. Verify the workflow resumes exactly where it left off.

## Next Steps

- **[DAG Agents](./orchestration-dag-agents.md)** - Complex flows inside activities
- **[Production Deployment](../../getting-started/07-production-deployment.md)** - Scaling workers
