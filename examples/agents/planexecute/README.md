# PlanExecute Agent Example

## Description

This example shows you how to use PlanExecute agents in Beluga AI. You'll learn:

- How PlanExecute agents separate planning from execution
- How to configure plan complexity and iteration limits
- How to use different LLMs for planning vs execution
- Best practices for OTEL instrumentation and error handling

## Prerequisites

Before running this example, you need:

| Requirement | Version | Notes |
|-------------|---------|-------|
| Go | 1.24+ | [Install Go](https://go.dev/doc/install) |
| Beluga AI | latest | `go get github.com/lookatitude/beluga-ai` |
| OpenAI API Key | - | Get one at [OpenAI](https://platform.openai.com/api-keys) |

### Environment Setup

Set these environment variables:

```bash
export OPENAI_API_KEY="your-api-key-here"

# Optional: Enable debug logging
export BELUGA_LOG_LEVEL=debug
```

## Usage

### Running the Example

1. **Navigate to the example directory**:

```bash
cd examples/agents/planexecute
```

2. **Run the example**:

```bash
go run planexecute_agent.go
```

### Expected Output

When successful, you'll see:

```
=== PlanExecute Agent Example ===

Task: Research the benefits of renewable energy and calculate the potential savings for a household switching to solar panels

=== Execution Plan ===
Goal: Research the benefits of renewable energy and calculate the potential savings for a household switching to solar panels
Total Steps: 4

Step 1: Search for renewable energy benefits
  Tool: web_search
  Input: renewable energy benefits
  Reasoning: Need to gather information about renewable energy advantages

Step 2: Search for solar panel household savings
  Tool: web_search
  Input: solar panel household savings average
  Reasoning: Need specific data on savings

Step 3: Calculate potential annual savings
  Tool: calculator
  Input: average_electricity_cost * solar_reduction_percentage
  Reasoning: Compute actual savings figures

Step 4: Summarize findings
  Tool: summarize
  Input: all gathered information
  Reasoning: Create a coherent final report

=== Execution Results ===
Steps executed: 4
Duration: 3.245s

Final Output:
Goal: Research the benefits of renewable energy...
Steps completed: 4/4

Results:
  step_1: {"query": "renewable energy benefits", "results": [...]}
  step_2: {"query": "solar panel household savings average", "results": [...]}
  step_3: {"expression": "...", "result": "calculated"}
  step_4: {"summary": "Summary of: all gathered information"}
```

### Using Different LLMs for Planning and Execution

For better results, use a more capable model for planning:

```go
// Use GPT-4 for planning (better reasoning)
plannerLLM, _ := llms.NewOpenAIChat(
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    llms.WithModel("gpt-4"),
)

// Use GPT-3.5 for execution (faster, cheaper)
executorLLM, _ := llms.NewOpenAIChat(
    llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    llms.WithModel("gpt-3.5-turbo"),
)

example, _ := NewPlanExecuteExample(
    "optimized-agent",
    plannerLLM,
    tools,
    WithPlannerLLM(plannerLLM),
    WithExecutorLLM(executorLLM),
)
```

## Code Structure

```
planexecute/
├── README.md                   # This file
├── planexecute_agent.go        # Main example implementation
└── planexecute_agent_test.go   # Comprehensive test suite
```

### Key Components

| File | Purpose |
|------|---------|
| `planexecute_agent.go` | Main implementation showing PlanExecute pattern |
| `planexecute_agent_test.go` | Tests covering creation, execution, and edge cases |

### Design Decisions

This example demonstrates these Beluga AI patterns:

- **Composition over inheritance**: `PlanExecuteExample` wraps the agent rather than extending it
- **Functional options**: `WithMaxPlanSteps()`, `WithPlannerLLM()` etc. for flexible configuration
- **OTEL Instrumentation**: Spans for plan generation, execution, and synthesis
- **Error handling**: Partial results returned even on partial failures

## Testing

### Running Tests

```bash
# Run all tests
go test -v ./...

# Run with race detection
go test -race ./...

# Run with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...
```

### Test Structure

| Test | What it verifies |
|------|-----------------|
| `TestNewPlanExecuteExample` | Example creation with various configurations |
| `TestPlanExecuteExample_Run` | Main run method behavior |
| `TestPlanExecuteExample_ContextCancellation` | Proper context handling |
| `TestDisplayPlan` | Plan display formatting |
| `TestCreateResearchTools` | Tool creation and naming |
| `TestToolExecution` | Individual tool execution |
| `TestPlanExecuteOptions` | Configuration options work correctly |
| `BenchmarkPlanGeneration` | Performance of plan generation |

### Expected Test Output

```
=== RUN   TestNewPlanExecuteExample
=== RUN   TestNewPlanExecuteExample/creates_example_with_defaults
=== RUN   TestNewPlanExecuteExample/creates_example_with_options
=== RUN   TestNewPlanExecuteExample/creates_example_with_no_tools
--- PASS: TestNewPlanExecuteExample (0.01s)
    --- PASS: TestNewPlanExecuteExample/creates_example_with_defaults (0.00s)
    --- PASS: TestNewPlanExecuteExample/creates_example_with_options (0.00s)
    --- PASS: TestNewPlanExecuteExample/creates_example_with_no_tools (0.00s)
PASS
coverage: 82.5% of statements
```

## Troubleshooting

### Common Issues

<details>
<summary>❌ Error: "OPENAI_API_KEY environment variable is required"</summary>

**Cause:** The `OPENAI_API_KEY` environment variable is not set.

**Solution:**
```bash
export OPENAI_API_KEY="sk-..."
# Then run the example again
```
</details>

<details>
<summary>❌ Plans are empty or have no steps</summary>

**Cause:** The LLM didn't generate a valid JSON plan.

**Solution:**
1. Use a more capable model (GPT-4 instead of GPT-3.5)
2. Make the task description more specific
3. Ensure tools have clear, descriptive names and descriptions
</details>

<details>
<summary>❌ Plan execution stops early</summary>

**Cause:** A step failed and didn't produce output for subsequent steps.

**Solution:**
1. Check tool implementations for errors
2. Increase `WithMaxIterations()` if needed
3. Add error handling in tool functions
</details>

<details>
<summary>❌ Performance is slow</summary>

**Cause:** Each step makes a separate LLM call.

**Solution:**
1. Use separate LLMs: GPT-4 for planning, GPT-3.5 for execution
2. Reduce `WithMaxPlanSteps()` for simpler tasks
3. Consider caching tool results if inputs repeat
</details>

## When to Use PlanExecute vs ReAct

| Use PlanExecute When... | Use ReAct When... |
|------------------------|-------------------|
| Task has clear, sequential steps | Task requires adaptive exploration |
| You need to review the plan before execution | Each step depends on previous discoveries |
| Debugging is important | Speed is critical |
| Steps can fail and you want to know which | The path isn't clear upfront |

## Related Examples

After completing this example, you might want to explore:

- **[ReAct Agent](../react/README.md)** - Agents that reason and act in a tight loop
- **[Agent with Tools](../with_tools/README.md)** - Basic tool integration
- **[Streaming LLM](/examples/llms/streaming/README.md)** - Add streaming to agent responses

## Learn More

- **[Agent Types Guide](/docs/guides/agent-types.md)** - In-depth guide comparing agent types
- **[Custom Agent Extension](/docs/cookbook/custom-agent.md)** - Extend agent behavior
- **[Batch Processing](/docs/use-cases/batch-processing.md)** - Using agents at scale
