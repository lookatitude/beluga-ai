# Use Case 7: Distributed Workflow Orchestration System

## Overview & Objectives

### Business Problem

Enterprise applications require complex, long-running workflows that span multiple services, need fault tolerance, and must handle failures gracefully. Traditional orchestration solutions lack AI capabilities and don't integrate well with LLM-based decision making.

### Solution Approach

This use case implements a distributed workflow orchestration system that:
- Orchestrates complex workflows with Temporal integration
- Uses LLMs for intelligent decision making in workflows
- Handles failures with retries and circuit breakers
- Provides comprehensive observability
- Supports chains, graphs, and distributed workflows

### Key Benefits

- **Distributed Execution**: Workflows span multiple services and machines
- **Fault Tolerance**: Automatic retries and failure handling
- **AI-Powered Decisions**: LLM integration for intelligent workflow decisions
- **Observability**: Full tracing and metrics for workflow execution
- **Scalable**: Handles thousands of concurrent workflows

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    Workflow Clients                              │
│              (Applications, APIs, Services)                      │
└────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              Orchestration Layer (pkg/orchestration)              │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │   Chains     │  │   Graphs     │  │  Workflows   │         │
│  │  (Sequential)│  │  (DAG)       │  │  (Temporal)  │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└────────────────────────────┬────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   LLMs       │    │   Memory     │    │  Temporal    │
│  (pkg/llms)  │    │  (pkg/memory)│    │  Workflow    │
│              │    │               │    │  Engine      │
│  - OpenAI    │    │  - Buffer     │    │              │
│  - Anthropic │    │  - VectorStore│   │  - Activities │
└──────────────┘    └──────────────┘    └──────────────┘
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
                              ▼
              ┌────────────────────────┐
              │   External Services    │
              │  - APIs                │
              │  - Databases          │
              │  - Message Queues     │
              └────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    Observability Layer                           │
│  - Workflow execution metrics                                   │
│  - Distributed tracing                                          │
│  - Performance monitoring                                       │
└─────────────────────────────────────────────────────────────────┘
```

## Component Usage

### Beluga AI Packages Used

1. **pkg/orchestration**
   - Chain orchestration for sequential workflows
   - Graph orchestration for DAG-based workflows
   - Workflow orchestration with Temporal integration

2. **pkg/llms**
   - Intelligent decision making in workflows
   - Dynamic workflow routing based on LLM analysis

3. **pkg/memory**
   - Workflow state management
   - Context preservation across workflow steps

4. **pkg/monitoring**
   - Workflow execution metrics
   - Distributed tracing
   - Performance monitoring

5. **pkg/config**
   - Workflow configuration management

6. **pkg/server**
   - REST API for workflow management

## Implementation Guide

### Step 1: Create Workflow Orchestrator

```go
func createOrchestrator(ctx context.Context, cfg *config.Config) (orchestration.Orchestrator, error) {
    orch, err := orchestration.NewOrchestratorWithOptions(
        orchestration.WithChainTimeout(300*time.Second),
        orchestration.WithGraphMaxWorkers(10),
        orchestration.WithWorkflowTaskQueue("beluga-workflows"),
        orchestration.WithMetricsPrefix("workflow"),
    )
    return orch, nil
}
```

### Step 2: Define Workflow Activities

```go
// Activity: Process payment
func ProcessPaymentActivity(ctx context.Context, orderID string) (string, error) {
    // Payment processing logic
    paymentID := processPayment(orderID)
    return paymentID, nil
}

// Activity: Check inventory
func CheckInventoryActivity(ctx context.Context, productID string) (bool, error) {
    // Inventory check logic
    available := checkInventory(productID)
    return available, nil
}

// Activity: LLM-based decision
func LLMDecisionActivity(ctx context.Context, context string) (string, error) {
    llm, _ := llms.NewChatModel(ctx, "openai",
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )
    
    prompt := fmt.Sprintf("Based on this context, make a decision: %s", context)
    response, err := llm.Generate(ctx, []schema.Message{
        schema.NewHumanMessage(prompt),
    })
    if err != nil {
        return "", err
    }
    
    return response.GetContent(), nil
}
```

### Step 3: Create Temporal Workflow

```go
func OrderProcessingWorkflow(ctx workflow.Context, order Order) (OrderResult, error) {
    // Step 1: Check inventory
    var inventoryAvailable bool
    err := workflow.ExecuteActivity(ctx, CheckInventoryActivity, order.ProductID).Get(ctx, &inventoryAvailable)
    if err != nil {
        return OrderResult{}, err
    }
    
    if !inventoryAvailable {
        return OrderResult{Status: "out_of_stock"}, nil
    }
    
    // Step 2: LLM-based decision for special handling
    var decision string
    err = workflow.ExecuteActivity(ctx, LLMDecisionActivity, 
        fmt.Sprintf("Order value: %f, Customer tier: %s", order.Value, order.CustomerTier)).Get(ctx, &decision)
    if err != nil {
        return OrderResult{}, err
    }
    
    // Step 3: Process payment
    var paymentID string
    err = workflow.ExecuteActivity(ctx, ProcessPaymentActivity, order.ID).Get(ctx, &paymentID)
    if err != nil {
        return OrderResult{}, err
    }
    
    // Step 4: Fulfill order
    var fulfillmentID string
    err = workflow.ExecuteActivity(ctx, FulfillOrderActivity, order.ID).Get(ctx, &fulfillmentID)
    if err != nil {
        return OrderResult{}, err
    }
    
    return OrderResult{
        Status:        "completed",
        PaymentID:     paymentID,
        FulfillmentID: fulfillmentID,
    }, nil
}
```

### Step 4: Create Graph Workflow

```go
func createOrderGraph(ctx context.Context, orch orchestration.Orchestrator) (orchestration.Graph, error) {
    graph, err := orch.CreateGraph(
        orchestration.WithGraphName("order-processing"),
        orchestration.WithGraphMaxWorkers(5),
        orchestration.WithGraphParallelExecution(true),
    )
    if err != nil {
        return nil, err
    }
    
    // Add nodes
    graph.AddNode("validate", &ValidateOrderNode{})
    graph.AddNode("check_inventory", &CheckInventoryNode{})
    graph.AddNode("process_payment", &ProcessPaymentNode{})
    graph.AddNode("fulfill", &FulfillOrderNode{})
    graph.AddNode("notify", &NotifyCustomerNode{})
    
    // Define dependencies
    graph.AddEdge("validate", "check_inventory")
    graph.AddEdge("check_inventory", "process_payment")
    graph.AddEdge("process_payment", "fulfill")
    graph.AddEdge("fulfill", "notify")
    
    // Set entry and exit points
    graph.SetEntryPoint([]string{"validate"})
    graph.SetFinishPoint([]string{"notify"})
    
    return graph, nil
}
```

## Workflow & Data Flow

### End-to-End Process Flow

1. **Workflow Initiation**
   ```
   Client Request → Create Workflow → Start Execution
   ```

2. **Activity Execution**
   ```
   Workflow → Execute Activities → Collect Results
   ```

3. **Decision Making**
   ```
   Context → LLM Analysis → Route to Next Step
   ```

4. **Completion**
   ```
   All Steps Complete → Return Result → Update State
   ```

## Observability Setup

### Metrics to Monitor

- `workflows_started_total`: Total workflows started
- `workflow_duration_seconds`: Workflow execution time
- `workflow_activities_total`: Activity execution count
- `workflow_failures_total`: Failure count by type
- `workflow_retries_total`: Retry count

## Configuration Examples

### Complete YAML Configuration

```yaml
# config.yaml
app:
  name: "workflow-orchestration"
  version: "1.0.0"

orchestration:
  chain:
    timeout: 300s
    max_retries: 3
  graph:
    max_workers: 10
    parallel_execution: true
    timeout: 600s
  workflow:
    task_queue: "beluga-workflows"
    timeout: 3600s
    max_retries: 5

temporal:
  address: "localhost:7233"
  namespace: "default"

llm:
  provider: "openai"
  openai:
    api_key: "${OPENAI_API_KEY}"
    model: "gpt-4"

monitoring:
  otel:
    endpoint: "localhost:4317"
  metrics:
    enabled: true
    prefix: "workflow"
```

## Deployment Considerations

### Production Requirements

- **Temporal Server**: Temporal cluster for workflow execution
- **Compute**: Sufficient resources for concurrent workflows
- **Storage**: Persistent storage for workflow state
- **Network**: Reliable connectivity between services

## Testing Strategy

### Unit Tests

```go
func TestWorkflowGraph(t *testing.T) {
    graph := createTestOrderGraph(t)
    
    input := map[string]any{
        "order_id": "123",
        "product_id": "prod-1",
    }
    
    result, err := graph.Invoke(context.Background(), input)
    require.NoError(t, err)
    assert.Equal(t, "completed", result.(map[string]any)["status"])
}
```

## Troubleshooting Guide

### Common Issues

1. **Workflow Timeouts**
   - Increase timeout settings
   - Optimize activity execution
   - Break down large workflows

2. **Activity Failures**
   - Implement retry logic
   - Add circuit breakers
   - Monitor activity health

3. **Temporal Connection Issues**
   - Check Temporal server health
   - Verify network connectivity
   - Review connection configuration

## Conclusion

This Distributed Workflow Orchestration System demonstrates Beluga AI's capabilities in building enterprise-grade workflow systems. The architecture showcases:

- **Multiple Orchestration Patterns**: Chains, graphs, and workflows
- **Temporal Integration**: Distributed workflow execution
- **AI-Powered Decisions**: LLM integration for intelligent routing
- **Comprehensive Observability**: Full tracing and metrics

The system can be extended with:
- More workflow patterns
- Advanced error recovery
- Workflow versioning
- Dynamic workflow generation
- Multi-tenant support

