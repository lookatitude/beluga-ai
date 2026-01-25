---
title: Production Agent Platform
sidebar_position: 1
---

# Use Case 10: Production-Grade AI Agent Platform

## Overview & Objectives

### Business Problem

Organizations need a comprehensive AI agent platform that can handle diverse use cases, integrate with multiple systems, scale to enterprise needs, and provide full observability. Building such a platform from scratch is complex and time-consuming.

### Solution Approach

This use case implements a complete, production-ready AI agent platform that:
- Integrates ALL Beluga AI framework components
- Supports multiple agent types and use cases
- Provides comprehensive tool ecosystem
- Implements advanced memory management
- Offers full observability and monitoring
- Scales to enterprise requirements

### Key Benefits

- **Complete Platform**: All framework capabilities in one system
- **Flexible Architecture**: Supports diverse use cases
- **Enterprise-Grade**: Production-ready with full observability
- **Extensible**: Easy to add new agents, tools, and capabilities
- **Scalable**: Handles high-volume agent operations

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────┐
│                    Platform API Layer                            │
│  - REST API  - MCP Server  - WebSocket  - gRPC                 │
└────────────────────────────┬────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│              Agent Management Layer                             │
│                                                                  │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐         │
│  │  Agent       │  │  Agent       │  │  Agent       │         │
│  │  Registry    │  │  Executor    │  │  Factory     │         │
│  └──────────────┘  └──────────────┘  └──────────────┘         │
└────────────────────────────┬────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   Agents     │    │   Tools      │    │   Memory     │
│  (pkg/agents)│    │  (pkg/agents/│    │  (pkg/memory)│
│              │    │   tools)     │    │              │
│  - ReAct     │    │  - API       │    │  - Buffer     │
│  - Base      │    │  - Shell     │    │  - Summary   │
│              │    │  - Calculator│    │  - VectorStore│
└──────────────┘    │  - GoFunc    │    └──────────────┘
        │           │  - MCP       │              │
        │           └──────────────┘              │
        │                     │                   │
        └─────────────────────┼───────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        │                     │                     │
        ▼                     ▼                     ▼
┌──────────────┐    ┌──────────────┐    ┌──────────────┐
│   LLMs       │    │ Orchestration│    │ VectorStores │
│  (pkg/llms)  │    │  (pkg/       │    │  (pkg/       │
│              │    │  orchestration)│   │  vectorstores)│
│  - OpenAI    │    │              │    │              │
│  - Anthropic │    │  - Chains    │    │  - PgVector  │
│  - Bedrock   │    │  - Graphs    │    │  - Pinecone  │
│  - Ollama    │    │  - Workflows │    │  - InMemory  │
└──────────────┘    └──────────────┘    └──────────────┘

        │                     │                     │
        └─────────────────────┼─────────────────────┘
                              │
                              ▼
              ┌────────────────────────┐
              │   Observability        │
              │  (pkg/monitoring)     │
              │  - Metrics            │
              │  - Tracing            │
              │  - Logging            │
              └────────────────────────┘
```

## Component Usage

### ALL Beluga AI Packages Used

1. **pkg/core** - Runnable interface, DI container
2. **pkg/schema** - Message, Document, ToolCall types
3. **pkg/config** - Configuration management
4. **pkg/llms** - Multiple LLM providers
5. **pkg/chatmodels** - ChatModel interface
6. **pkg/embeddings** - Embedding generation
7. **pkg/vectorstores** - Vector storage
8. **pkg/retrievers** - Document retrieval
9. **pkg/memory** - All memory types
10. **pkg/prompts** - Prompt templates
11. **pkg/agents** - Agent framework
12. **pkg/agents/tools** - All tool types
13. **pkg/orchestration** - Chains, graphs, workflows
14. **pkg/monitoring** - Full observability
15. **pkg/server** - REST and MCP servers

## Implementation Guide

### Step 1: Platform Initialization

```go
type AgentPlatform struct {
    config        *config.Config
    agentRegistry *agents.Registry
    toolRegistry  *tools.Registry
    orchestrator  orchestration.Orchestrator
    vectorStores  map[string]vectorstores.VectorStore
    llmProviders  map[string]llms.ChatModel
    memoryManager *MemoryManager
    server        server.Server
    monitor       *monitoring.MetricsCollector
}

func NewAgentPlatform(ctx context.Context, cfg *config.Config) (*AgentPlatform, error) {
    platform := &AgentPlatform{
        config:        cfg,
        agentRegistry: agents.NewRegistry(),
        toolRegistry:  tools.NewInMemoryToolRegistry(),
        vectorStores:  make(map[string]vectorstores.VectorStore),
        llmProviders:  make(map[string]llms.ChatModel),
    }
    
    // Initialize all components
    if err := platform.initializeLLMs(ctx); err != nil {
        return nil, err
    }
    
    if err := platform.initializeVectorStores(ctx); err != nil {
        return nil, err
    }
    
    if err := platform.initializeTools(ctx); err != nil {
        return nil, err
    }
    
    if err := platform.initializeOrchestration(ctx); err != nil {
        return nil, err
    }
    
    if err := platform.initializeMemory(ctx); err != nil {
        return nil, err
    }
    
    if err := platform.initializeMonitoring(ctx); err != nil {
        return nil, err
    }
    
    if err := platform.initializeServer(ctx); err != nil {
        return nil, err
    }
    
    return platform, nil
}
```

### Step 2: Agent Creation

```go
func (p *AgentPlatform) CreateAgent(ctx context.Context, config AgentConfig) (agents.Agent, error) {
    // Get LLM
    llm, ok := p.llmProviders[config.LLMProvider]
    if !ok {
        return nil, fmt.Errorf("LLM provider not found: %s", config.LLMProvider)
    }
    
    // Get memory
    memory := p.memoryManager.GetMemory(config.MemoryType, config.MemoryConfig)
    
    // Get tools
    agentTools := p.getToolsForAgent(config.Tools)
    
    // Create agent
    agent, err := agents.NewReActAgent(
        agents.WithName(config.Name),
        agents.WithDescription(config.Description),
        agents.WithLLM(llm),
        agents.WithMemory(memory),
        agents.WithTools(agentTools),
        agents.WithMaxIterations(config.MaxIterations),
    )
    if err != nil {
        return nil, err
    }
    
    // Register agent
    p.agentRegistry.RegisterAgent(config.Name, agent)
    
    return agent, nil
}
```

### Step 3: Tool Integration

```go
func (p *AgentPlatform) initializeTools(ctx context.Context) error {
    // API tools
    for _, apiCfg := range p.config.GetAPIConfigs() {
        apiTool := tools.NewAPITool(
            apiCfg.Name,
            apiCfg.Description,
            apiCfg.Endpoint,
            apiCfg.APIKey,
        )
        p.toolRegistry.RegisterTool(apiTool)
    }
    
    // Shell tool
    shellTool, _ := tools.NewShellTool(60 * time.Second)
    p.toolRegistry.RegisterTool(shellTool)
    
    // Calculator tool
    calcTool, _ := tools.NewCalculatorTool()
    p.toolRegistry.RegisterTool(calcTool)
    
    // MCP tools
    mcpTool := tools.NewMCPTool("mcp-server", "localhost:8081")
    p.toolRegistry.RegisterTool(mcpTool)
    
    // Custom Go functions
    for _, funcCfg := range p.config.GetFunctionConfigs() {
        goFunc := tools.NewGoFunctionTool(
            funcCfg.Name,
            funcCfg.Description,
            funcCfg.Schema,
            funcCfg.Function,
        )
        p.toolRegistry.RegisterTool(goFunc)
    }
    
    return nil
}
```

### Step 4: Orchestration Setup

```go
func (p *AgentPlatform) initializeOrchestration(ctx context.Context) error {
    orch, err := orchestration.NewOrchestratorWithOptions(
        orchestration.WithChainTimeout(300*time.Second),
        orchestration.WithGraphMaxWorkers(10),
        orchestration.WithWorkflowTaskQueue("agent-workflows"),
        orchestration.WithMetricsPrefix("platform"),
    )
    if err != nil {
        return err
    }
    
    p.orchestrator = orch
    return nil
}

func (p *AgentPlatform) CreateAgentWorkflow(ctx context.Context, workflowDef WorkflowDefinition) (orchestration.Workflow, error) {
    // Create workflow from definition
    workflowFn := p.buildWorkflowFunction(workflowDef)
    
    workflow, err := p.orchestrator.CreateWorkflow(workflowFn,
        orchestration.WithWorkflowName(workflowDef.Name),
        orchestration.WithWorkflowTimeout(workflowDef.Timeout),
    )
    return workflow, err
}
```

### Step 5: Memory Management

```go
type MemoryManager struct {
    bufferMemories    map[string]memory.Memory
    summaryMemories   map[string]memory.Memory
    vectorMemories    map[string]memory.Memory
    vectorStores      map[string]vectorstores.VectorStore
    embedders         map[string]embeddings.Embedder
}

func (mm *MemoryManager) GetMemory(memoryType string, config MemoryConfig) memory.Memory {
    switch memoryType {
    case "buffer":
        return memory.NewBufferMemory()
    case "summary":
        llm := mm.getLLM(config.LLMProvider)
        return memory.NewSummaryMemory(llm)
    case "vectorstore":
        vs := mm.vectorStores[config.VectorStore]
        emb := mm.embedders[config.Embedder]
        return memory.NewVectorStoreMemory(vs, emb)
    default:
        return memory.NewBufferMemory()
    }
}
```

### Step 6: Server Setup

```go
func (p *AgentPlatform) initializeServer(ctx context.Context) error {
    // REST server
    restServer, err := server.NewRESTServer(
        server.WithRESTConfig(server.RESTConfig{
            Config: server.Config{
                Host: p.config.GetString("server.host"),
                Port: p.config.GetInt("server.port"),
            },
        }),
    )
    if err != nil {
        return err
    }
    
    // Register handlers
    restServer.RegisterHandler("POST", "/api/v1/agents", p.handleCreateAgent)
    restServer.RegisterHandler("POST", "/api/v1/agents/:name/execute", p.handleExecuteAgent)
    restServer.RegisterHandler("GET", "/api/v1/agents", p.handleListAgents)
    
    // MCP server
    mcpServer, err := server.NewMCPServer(
        server.WithMCPConfig(server.MCPConfig{
            Config: server.Config{
                Host: p.config.GetString("server.mcp.host"),
                Port: p.config.GetInt("server.mcp.port"),
            },
        }),
    )
    if err != nil {
        return err
    }
    
    // Register MCP tools
    for _, tool := range p.toolRegistry.ListTools() {
        mcpServer.RegisterTool(tool)
    }
    
    p.server = restServer
    return nil
}
```

## Workflow & Data Flow

### End-to-End Process Flow

1. **Platform Initialization**
   ```
   Config → Initialize All Components → Ready State
   ```

2. **Agent Creation**
   ```
   Agent Config → Create Agent → Register → Available
   ```

3. **Agent Execution**
   ```
   Request → Agent → Use Tools → LLM → Memory → Response
   ```

4. **Orchestration**
   ```
   Workflow → Execute Steps → Coordinate Agents → Result
   ```

5. **Observability**
   ```
   All Operations → Metrics → Tracing → Logging
   ```

## Observability Setup

### Comprehensive Metrics

- **Agent Metrics**: Execution count, duration, success rate
- **Tool Metrics**: Usage count, duration, error rate
- **LLM Metrics**: Request count, latency, token usage
- **Memory Metrics**: Operations, size, retrieval time
- **Orchestration Metrics**: Workflow execution, step duration
- **Platform Metrics**: Overall health, resource usage

## Configuration Examples

### Complete YAML Configuration

```yaml
# config.yaml
app:
  name: "agent-platform"
  version: "1.0.0"

platform:
  max_agents: 1000
  max_concurrent_executions: 100

agents:
  default:
    type: "react"
    max_iterations: 20
    memory_type: "buffer"

llm:
  providers:
    openai:
      api_key: "${OPENAI_API_KEY}"
      models: ["gpt-4", "gpt-3.5-turbo"]
    anthropic:
      api_key: "${ANTHROPIC_API_KEY}"
      models: ["claude-3-opus", "claude-3-sonnet"]

tools:
  api:
    - name: "github_api"
      endpoint: "https://api.github.com"
      api_key: "${GITHUB_TOKEN}"
  shell:
    enabled: true
    timeout: 60s
  calculator:
    enabled: true
  mcp:
    enabled: true
    endpoint: "localhost:8081"

memory:
  buffer:
    enabled: true
    max_messages: 100
  summary:
    enabled: true
    summarize_interval: 20
  vectorstore:
    enabled: true
    provider: "pgvector"
    connection_string: "${POSTGRES_CONNECTION_STRING}"

vectorstores:
  default:
    provider: "pgvector"
    connection_string: "${POSTGRES_CONNECTION_STRING}"
  pinecone:
    provider: "pinecone"
    api_key: "${PINECONE_API_KEY}"

orchestration:
  chain:
    timeout: 300s
  graph:
    max_workers: 10
  workflow:
    task_queue: "agent-workflows"

server:
  rest:
    host: "0.0.0.0"
    port: 8080
  mcp:
    host: "0.0.0.0"
    port: 8081

monitoring:
  otel:
    endpoint: "localhost:4317"
  metrics:
    enabled: true
    prefix: "agent_platform"
  tracing:
    enabled: true
    sample_rate: 1.0
```

## Deployment Considerations

### Production Requirements

- **High Availability**: Multiple platform instances
- **Load Balancing**: Distribute agent executions
- **Database**: PostgreSQL with pgvector
- **Message Queue**: For workflow orchestration
- **Monitoring**: Full observability stack

### Scaling Strategies

1. **Horizontal Scaling**: Multiple platform instances
2. **Agent Pooling**: Reuse agent instances
3. **Caching**: Cache frequent operations
4. **Async Processing**: Queue-based agent execution

## Testing Strategy

### Comprehensive Test Suite

```go
func TestAgentPlatform(t *testing.T) {
    platform := createTestPlatform(t)
    
    // Test agent creation
    agent, err := platform.CreateAgent(context.Background(), AgentConfig{
        Name: "test-agent",
        LLMProvider: "openai",
        Tools: []string{"calculator"},
    })
    require.NoError(t, err)
    
    // Test agent execution
    result, err := agent.Execute(context.Background(), map[string]any{
        "task": "Calculate 2+2",
    })
    require.NoError(t, err)
    assert.Contains(t, result.(string), "4")
}
```

## Troubleshooting Guide

### Common Issues

1. **Agent Execution Failures**
   - Check LLM provider health
   - Verify tool availability
   - Review memory configuration

2. **Performance Issues**
   - Optimize agent configurations
   - Scale platform instances
   - Implement caching

3. **Memory Issues**
   - Monitor memory usage
   - Adjust buffer sizes
   - Use summary memory for long conversations

## Conclusion

This Production-Grade AI Agent Platform demonstrates the full power of Beluga AI Framework by integrating ALL components into a unified, enterprise-ready system. The architecture showcases:

- **Complete Integration**: All framework components working together
- **Flexible Architecture**: Supports diverse use cases
- **Enterprise-Grade**: Production-ready with full observability
- **Extensible Design**: Easy to add new capabilities
- **Scalable**: Handles enterprise-scale operations

The platform serves as a foundation for building:
- Multi-agent systems
- Complex AI workflows
- Enterprise AI applications
- Custom AI solutions
- Research and experimentation platforms

This use case represents the pinnacle of what's possible with Beluga AI Framework, demonstrating that all components can work together seamlessly to create powerful, production-ready AI systems.

