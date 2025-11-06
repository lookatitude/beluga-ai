# Use Case 2: Multi-Agent Customer Support System

## Overview & Objectives

### Business Problem

Customer support teams face challenges handling diverse inquiries requiring different expertise levels. Simple queries get escalated unnecessarily, while complex issues require coordination between multiple specialists. Traditional ticketing systems lack intelligence to route, analyze, and resolve issues automatically.

### Solution Approach

This use case implements an intelligent multi-agent customer support system that:
- Routes inquiries to specialized agents based on problem type
- Coordinates multiple agents working on complex issues
- Maintains conversation context across agent interactions
- Integrates with external APIs for real-time information
- Provides automated resolution for common issues

### Key Benefits

- **Intelligent Routing**: Automatically routes tickets to appropriate agents
- **Multi-Agent Collaboration**: Agents work together on complex issues
- **Context Preservation**: Maintains conversation history across agents
- **Tool Integration**: Agents use tools to fetch data, perform calculations, and execute actions
- **Scalable Architecture**: Handles high-volume support requests efficiently

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    Customer Support Portal                       │
│              (Web, Mobile, Email, Chat)                         │
└────────────────────────────┬────────────────────────────────────┘
                              │
                              │ HTTP/REST
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    REST Server (pkg/server)                     │
│  - Request routing                                              │
│  - Authentication                                              │
│  - Rate limiting                                               │
└────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              Orchestration Graph (pkg/orchestration)              │
│                                                                  │
│  ┌──────────────┐                                              │
│  │  Router      │ ── Determines issue type and routes          │
│  │  Agent       │    to appropriate specialist                 │
│  └──────┬───────┘                                              │
│         │                                                       │
│    ┌────┴────┬──────────┬──────────┐                          │
│    │         │          │          │                          │
│    ▼         ▼          ▼          ▼                          │
│  ┌────┐  ┌────┐    ┌────┐    ┌────┐                          │
│  │Tech│  │Billing│  │Sales│  │General│                        │
│  │Agent│  │Agent │  │Agent│  │Agent │                        │
│  └────┘  └────┘    └────┘    └────┘                          │
│    │         │          │          │                          │
│    └─────────┴──────────┴──────────┘                          │
│              │                                                 │
│              ▼                                                 │
│  ┌──────────────────────┐                                     │
│  │  Resolution          │ ── Final response generation         │
│  │  Agent               │                                     │
│  └──────────────────────┘                                     │
└────────────────────────────┬────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Tools      │    │   Memory     │    │     LLMs     │
│  (pkg/agents/│    │  (pkg/memory) │    │  (pkg/llms)  │
│   tools)     │    │               │    │              │
│  - API       │    │  - Buffer     │    │  - OpenAI    │
│  - Shell     │    │  - VectorStore│   │  - Anthropic │
│  - Calculator│    │               │    │              │
└──────────────┘    └──────────────┘    └──────────────┘
        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
                              ▼
              ┌────────────────────────┐
              │   External Systems      │
              │  - CRM                  │
              │  - Billing System       │
              │  - Knowledge Base       │
              └────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                    Observability Layer                           │
│  - Agent execution metrics                                       │
│  - Tool usage tracking                                           │
│  - Response time monitoring                                      │
│  - Resolution rate tracking                                      │
└─────────────────────────────────────────────────────────────────┘
```

## Component Usage

### Beluga AI Packages Used

1. **pkg/agents**
   - ReAct agents for reasoning and action
   - Agent executor for plan execution
   - Multiple specialized agents

2. **pkg/agents/tools**
   - API tool for external system integration
   - Shell tool for system operations
   - Calculator tool for computations
   - GoFunc tool for custom functions

3. **pkg/memory**
   - BufferMemory for conversation history
   - VectorStoreMemory for semantic retrieval of past issues

4. **pkg/llms**
   - Multiple LLM providers for agent reasoning
   - ChatModel interface for conversations

5. **pkg/orchestration**
   - Graph orchestration for multi-agent coordination
   - Dependency management between agents

6. **pkg/monitoring**
   - Agent execution metrics
   - Tool usage tracking
   - Performance monitoring

7. **pkg/server**
   - REST API for customer interactions
   - MCP server for tool integration

8. **pkg/config**
   - Configuration management
   - Agent-specific settings

## Implementation Guide

### Step 1: Define Agent Types

```go
type AgentType string

const (
    AgentTypeRouter    AgentType = "router"
    AgentTypeTech      AgentType = "tech"
    AgentTypeBilling   AgentType = "billing"
    AgentTypeSales     AgentType = "sales"
    AgentTypeGeneral   AgentType = "general"
    AgentTypeResolution AgentType = "resolution"
)

type AgentConfig struct {
    Type        AgentType
    Name        string
    Description string
    Tools       []string
    LLMProvider string
    LLMModel    string
}
```

### Step 2: Create Specialized Agents

```go
func createTechAgent(ctx context.Context, cfg *config.Config, tools tools.Registry) (agents.Agent, error) {
    llm, err := llms.NewChatModel(ctx, "openai",
        llms.WithAPIKey(cfg.GetString("llm.openai.api_key")),
        llms.WithModel("gpt-4"),
    )
    if err != nil {
        return nil, err
    }

    memory := memory.NewBufferMemory()

    agent, err := agents.NewReActAgent(
        agents.WithName("tech-support-agent"),
        agents.WithDescription("Specialized in technical issues and troubleshooting"),
        agents.WithLLM(llm),
        agents.WithMemory(memory),
        agents.WithTools(tools),
        agents.WithMaxIterations(10),
    )
    if err != nil {
        return nil, err
    }

    return agent, nil
}

func createBillingAgent(ctx context.Context, cfg *config.Config, tools tools.Registry) (agents.Agent, error) {
    llm, err := llms.NewChatModel(ctx, "openai",
        llms.WithAPIKey(cfg.GetString("llm.openai.api_key")),
        llms.WithModel("gpt-4"),
    )
    if err != nil {
        return nil, err
    }

    memory := memory.NewBufferMemory()

    // Billing agent needs API tool for billing system
    apiTool, _ := tools.GetTool("billing_api")
    
    agent, err := agents.NewReActAgent(
        agents.WithName("billing-agent"),
        agents.WithDescription("Handles billing inquiries, refunds, and payment issues"),
        agents.WithLLM(llm),
        agents.WithMemory(memory),
        agents.WithTools(tools),
        agents.WithMaxIterations(8),
    )
    return agent, nil
}
```

### Step 3: Create Router Agent

```go
func createRouterAgent(ctx context.Context, cfg *config.Config) (agents.Agent, error) {
    llm, err := llms.NewChatModel(ctx, "openai",
        llms.WithAPIKey(cfg.GetString("llm.openai.api_key")),
        llms.WithModel("gpt-4"),
    )
    if err != nil {
        return nil, err
    }

    routerAgent, err := agents.NewReActAgent(
        agents.WithName("router-agent"),
        agents.WithDescription("Routes customer inquiries to appropriate specialist agents"),
        agents.WithLLM(llm),
        agents.WithMaxIterations(3),
    )
    return routerAgent, nil
}

// Router agent determines issue type
func (r *RouterAgent) RouteIssue(ctx context.Context, inquiry string) (AgentType, error) {
    prompt := fmt.Sprintf(`Analyze this customer inquiry and determine which specialist agent should handle it:
    
    Inquiry: %s
    
    Available agents:
    - tech: Technical issues, bugs, feature requests
    - billing: Payment, refunds, subscription issues
    - sales: Product questions, pricing, demos
    - general: General questions, account management
    
    Respond with only the agent type (tech, billing, sales, or general).`, inquiry)

    messages := []schema.Message{
        schema.NewSystemMessage("You are a routing agent. Analyze inquiries and route them to the correct specialist."),
        schema.NewHumanMessage(prompt),
    }

    response, err := r.llm.Generate(ctx, messages)
    if err != nil {
        return AgentTypeGeneral, err
    }

    agentType := parseAgentType(response.GetContent())
    return agentType, nil
}
```

### Step 4: Set Up Tools

```go
func setupTools(cfg *config.Config) (tools.Registry, error) {
    registry := tools.NewInMemoryToolRegistry()

    // API tool for external systems
    apiTool, err := tools.NewAPITool(
        "billing_api",
        "Access billing system to check accounts, process refunds",
        "https://api.billing.example.com",
        cfg.GetString("tools.billing_api.key"),
    )
    if err != nil {
        return nil, err
    }
    registry.RegisterTool(apiTool)

    // Calculator tool
    calcTool, err := tools.NewCalculatorTool()
    if err != nil {
        return nil, err
    }
    registry.RegisterTool(calcTool)

    // Shell tool (with restrictions)
    shellTool, err := tools.NewShellTool(30 * time.Second)
    if err != nil {
        return nil, err
    }
    registry.RegisterTool(shellTool)

    // Custom Go function tool
    knowledgeBaseTool := tools.NewGoFunctionTool(
        "search_knowledge_base",
        "Search internal knowledge base for solutions",
        `{"type": "object", "properties": {"query": {"type": "string"}}}`,
        searchKnowledgeBase,
    )
    registry.RegisterTool(knowledgeBaseTool)

    return registry, nil
}

func searchKnowledgeBase(ctx context.Context, args map[string]any) (string, error) {
    query := args["query"].(string)
    // Implementation to search knowledge base
    return fmt.Sprintf("Found solutions for: %s", query), nil
}
```

### Step 5: Create Orchestration Graph

```go
func createSupportGraph(ctx context.Context, agents map[AgentType]agents.Agent) (orchestration.Graph, error) {
    graph, err := orchestration.NewGraph(
        orchestration.WithGraphName("customer-support"),
        orchestration.WithGraphMaxWorkers(5),
        orchestration.WithGraphParallelExecution(true),
    )
    if err != nil {
        return nil, err
    }

    // Add router node
    routerNode := &RouterNode{agent: agents[AgentTypeRouter]}
    graph.AddNode("router", routerNode)

    // Add specialist agent nodes
    graph.AddNode("tech", &AgentNode{agent: agents[AgentTypeTech]})
    graph.AddNode("billing", &AgentNode{agent: agents[AgentTypeBilling]})
    graph.AddNode("sales", &AgentNode{agent: agents[AgentTypeSales]})
    graph.AddNode("general", &AgentNode{agent: agents[AgentTypeGeneral]})
    graph.AddNode("resolution", &ResolutionNode{})

    // Define edges: router routes to specialists
    graph.AddEdge("router", "tech")
    graph.AddEdge("router", "billing")
    graph.AddEdge("router", "sales")
    graph.AddEdge("router", "general")

    // All specialists feed into resolution
    graph.AddEdge("tech", "resolution")
    graph.AddEdge("billing", "resolution")
    graph.AddEdge("sales", "resolution")
    graph.AddEdge("general", "resolution")

    // Set entry and exit points
    graph.SetEntryPoint([]string{"router"})
    graph.SetFinishPoint([]string{"resolution"})

    return graph, nil
}

// RouterNode implements core.Runnable
type RouterNode struct {
    agent agents.Agent
}

func (r *RouterNode) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
    inquiry := input.(map[string]any)["inquiry"].(string)
    
    // Router agent determines routing
    agentType, err := r.agent.Execute(ctx, map[string]any{
        "task": fmt.Sprintf("Route this inquiry: %s", inquiry),
    })
    if err != nil {
        return nil, err
    }

    return map[string]any{
        "inquiry":   inquiry,
        "agent_type": agentType,
    }, nil
}

// AgentNode implements core.Runnable
type AgentNode struct {
    agent agents.Agent
}

func (a *AgentNode) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
    inquiry := input.(map[string]any)["inquiry"].(string)
    
    result, err := a.agent.Execute(ctx, map[string]any{
        "task": inquiry,
    })
    if err != nil {
        return nil, err
    }

    return map[string]any{
        "inquiry": inquiry,
        "response": result,
    }, nil
}
```

### Step 6: Set Up REST Server

```go
func setupSupportServer(graph orchestration.Graph, cfg *config.Config) error {
    restProvider, err := server.NewRESTServer(
        server.WithRESTConfig(server.RESTConfig{
            Config: server.Config{
                Host: cfg.GetString("server.host"),
                Port: cfg.GetInt("server.port"),
            },
            APIBasePath: "/api/v1",
        }),
    )
    if err != nil {
        return err
    }

    handler := &SupportHandler{graph: graph}
    restProvider.RegisterHandler("POST", "/api/v1/support/inquiry", handler.HandleInquiry)
    restProvider.RegisterHandler("GET", "/api/v1/support/status/:id", handler.HandleStatus)

    return restProvider.Start(context.Background())
}

type SupportHandler struct {
    graph orchestration.Graph
}

func (h *SupportHandler) HandleInquiry(w http.ResponseWriter, r *http.Request) {
    var req struct {
        Inquiry string `json:"inquiry"`
        CustomerID string `json:"customer_id"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request", http.StatusBadRequest)
        return
    }

    input := map[string]any{
        "inquiry":    req.Inquiry,
        "customer_id": req.CustomerID,
    }

    result, err := h.graph.Invoke(r.Context(), input)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    response := map[string]any{
        "response": result.(map[string]any)["response"],
        "status":   "resolved",
    }
    json.NewEncoder(w).Encode(response)
}
```

## Workflow & Data Flow

### End-to-End Process Flow

1. **Inquiry Reception**
   ```
   Customer Inquiry → REST Server → Router Agent
   ```

2. **Routing Decision**
   ```
   Router Agent → Analyzes Inquiry → Determines Agent Type
   ```

3. **Specialist Processing**
   ```
   Specialist Agent → Uses Tools → Generates Response
   ```

4. **Resolution**
   ```
   Resolution Agent → Compiles Response → Returns to Customer
   ```

### Component Interactions

- **Router Agent ↔ Specialist Agents**: Routes based on issue type
- **Agents ↔ Tools**: Agents use tools to fetch data and perform actions
- **Agents ↔ Memory**: Maintains conversation context
- **Graph ↔ All Agents**: Orchestrates multi-agent workflow

### Error Handling Strategies

```go
func (h *SupportHandler) HandleInquiry(w http.ResponseWriter, r *http.Request) {
    ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
    defer cancel()

    // Retry logic for transient failures
    var result any
    var err error
    for i := 0; i < 3; i++ {
        result, err = h.graph.Invoke(ctx, input)
        if err == nil {
            break
        }
        if !isRetryable(err) {
            break
        }
        time.Sleep(time.Duration(i+1) * time.Second)
    }

    if err != nil {
        // Fallback to general agent
        result, err = h.fallbackToGeneralAgent(ctx, input)
        if err != nil {
            http.Error(w, "service unavailable", http.StatusServiceUnavailable)
            return
        }
    }

    // Return response
    json.NewEncoder(w).Encode(result)
}
```

## Observability Setup

### Metrics to Monitor

- `support_inquiries_total`: Total inquiries processed
- `support_routing_duration_seconds`: Time to route inquiry
- `support_agent_execution_duration_seconds`: Agent processing time
- `support_tool_usage_total`: Tool usage by type
- `support_resolution_rate`: Percentage of resolved inquiries
- `support_agent_errors_total`: Errors by agent type

### Tracing Setup

```go
func (a *AgentNode) Invoke(ctx context.Context, input any, opts ...core.Option) (any, error) {
    tracer := otel.Tracer("support-system")
    ctx, span := tracer.Start(ctx, "agent.execute",
        trace.WithAttributes(
            attribute.String("agent.type", a.agent.Name()),
            attribute.String("inquiry", input.(map[string]any)["inquiry"].(string)),
        ),
    )
    defer span.End()

    result, err := a.agent.Execute(ctx, input)
    if err != nil {
        span.RecordError(err)
        return nil, err
    }

    span.SetAttributes(
        attribute.String("response.length", fmt.Sprintf("%d", len(result.(string)))),
    )

    return result, nil
}
```

## Configuration Examples

### Complete YAML Configuration

```yaml
# config.yaml
app:
  name: "customer-support-system"
  version: "1.0.0"

server:
  host: "0.0.0.0"
  port: 8080

agents:
  router:
    type: "react"
    llm:
      provider: "openai"
      model: "gpt-4"
      temperature: 0.3
    max_iterations: 3

  tech:
    type: "react"
    llm:
      provider: "openai"
      model: "gpt-4"
      temperature: 0.7
    tools:
      - "shell"
      - "calculator"
      - "knowledge_base"
    max_iterations: 10

  billing:
    type: "react"
    llm:
      provider: "openai"
      model: "gpt-4"
    tools:
      - "billing_api"
      - "calculator"
    max_iterations: 8

  sales:
    type: "react"
    llm:
      provider: "anthropic"
      model: "claude-3-sonnet"
    tools:
      - "knowledge_base"
    max_iterations: 6

  general:
    type: "react"
    llm:
      provider: "openai"
      model: "gpt-3.5-turbo"
    max_iterations: 5

tools:
  billing_api:
    endpoint: "https://api.billing.example.com"
    api_key: "${BILLING_API_KEY}"
    timeout: 30s

  knowledge_base:
    endpoint: "https://kb.example.com/api"
    api_key: "${KB_API_KEY}"

llm:
  providers:
    openai:
      api_key: "${OPENAI_API_KEY}"
    anthropic:
      api_key: "${ANTHROPIC_API_KEY}"

memory:
  type: "buffer"
  buffer:
    return_messages: true

orchestration:
  graph:
    max_workers: 5
    parallel_execution: true
    timeout: 120s

monitoring:
  otel:
    endpoint: "localhost:4317"
  metrics:
    enabled: true
    prefix: "support"
```

## Deployment Considerations

### Production Requirements

- **Compute**: 8+ CPU cores, 16GB+ RAM for concurrent agent execution
- **LLM Access**: Low-latency connection to LLM providers
- **External APIs**: Reliable connectivity to CRM, billing systems
- **Database**: For conversation history and metrics storage

### Scaling Strategies

1. **Agent Pooling**: Maintain pools of agent instances
2. **Load Balancing**: Distribute inquiries across multiple instances
3. **Caching**: Cache common responses and routing decisions
4. **Async Processing**: Process complex inquiries asynchronously

## Testing Strategy

### Unit Tests

```go
func TestRouterAgent(t *testing.T) {
    router := createTestRouterAgent(t)
    
    agentType, err := router.RouteIssue(context.Background(), "I can't log in")
    assert.NoError(t, err)
    assert.Equal(t, AgentTypeTech, agentType)
}

func TestTechAgent(t *testing.T) {
    agent := createTestTechAgent(t)
    
    result, err := agent.Execute(context.Background(), map[string]any{
        "task": "Help me reset my password",
    })
    assert.NoError(t, err)
    assert.NotEmpty(t, result)
}
```

### Integration Tests

```go
func TestSupportGraph(t *testing.T) {
    graph := createTestSupportGraph(t)
    
    input := map[string]any{
        "inquiry": "I need help with my billing",
    }
    
    result, err := graph.Invoke(context.Background(), input)
    assert.NoError(t, err)
    assert.Contains(t, result.(map[string]any)["response"].(string), "billing")
}
```

## Troubleshooting Guide

### Common Issues

1. **Incorrect Routing**
   - Improve router agent prompts
   - Add more training examples
   - Adjust routing confidence thresholds

2. **Agent Timeout**
   - Increase max_iterations
   - Optimize tool calls
   - Use faster LLM models

3. **Tool Failures**
   - Implement retry logic
   - Add fallback mechanisms
   - Monitor tool health

## Conclusion

This Multi-Agent Customer Support System demonstrates Beluga AI's capabilities in building intelligent, collaborative agent systems. The architecture showcases:

- **Specialized Agents**: Different agents for different expertise areas
- **Intelligent Routing**: Automatic inquiry routing
- **Tool Integration**: Agents using tools for real-world actions
- **Observability**: Comprehensive monitoring of agent behavior

The system can be extended with:
- Learning from resolved tickets
- Multi-language support
- Sentiment analysis
- Escalation to human agents
- Customer satisfaction tracking

