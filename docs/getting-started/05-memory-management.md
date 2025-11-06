# Part 5: Memory Management

In this tutorial, you'll learn how to add conversation memory to your agents and applications. Memory enables agents to remember previous conversations and maintain context across interactions.

## Learning Objectives

- ✅ Understand different memory types
- ✅ Implement buffer memory
- ✅ Use window memory for size limits
- ✅ Add memory to agents
- ✅ Manage conversation history

## Prerequisites

- Completed [Part 3: Creating Your First Agent](./03-first-agent.md)
- Basic understanding of conversation context
- API key for an LLM provider

## What is Memory?

Memory in Beluga AI allows agents to:
- Remember previous conversations
- Maintain context across interactions
- Store and retrieve conversation history
- Manage conversation state

## Memory Types

Beluga AI supports several memory types:

1. **Buffer Memory** - Stores all conversation messages
2. **Window Memory** - Stores only the last N messages
3. **Summary Memory** - Summarizes old messages, keeps recent ones
4. **Vector Store Memory** - Semantic search over conversation history

## Step 1: Buffer Memory

Buffer memory stores all conversation messages:

```go
package main

import (
    "context"
    "fmt"

    "github.com/lookatitude/beluga-ai/pkg/memory"
)

func main() {
    ctx := context.Background()

    // Create buffer memory
    mem, err := memory.NewMemory(memory.MemoryTypeBuffer)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    // Save conversation context
    inputs := map[string]any{
        "input": "Hello, my name is Alice",
    }
    outputs := map[string]any{
        "output": "Nice to meet you, Alice!",
    }
    mem.SaveContext(ctx, inputs, outputs)

    // Load memory variables
    vars, err := mem.LoadMemoryVariables(ctx, map[string]any{})
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Memory: %v\n", vars["history"])
}
```

## Step 2: Window Memory

Window memory limits the number of stored messages:

```go
// Create window memory with size limit
mem, err := memory.NewMemory(
    memory.MemoryTypeWindow,
    memory.WithWindowSize(10), // Keep last 10 messages
)
```

## Step 3: Using Memory with Agents

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/memory"
)

func main() {
    ctx := context.Background()

    // Setup LLM
    config := llms.NewConfig(
        llms.WithProvider("openai"),
        llms.WithModelName("gpt-3.5-turbo"),
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )
    factory := llms.NewFactory()
    llm, _ := factory.CreateProvider("openai", config)

    // Create memory
    mem, _ := memory.NewMemory(memory.MemoryTypeBuffer)

    // Create agent with memory
    tools := []tools.Tool{
        tools.NewCalculatorTool(),
    }
    agent, _ := agents.NewBaseAgent("assistant", llm, tools)

    // Initialize agent
    agent.Initialize(map[string]interface{}{
        "memory": mem, // Attach memory
    })

    // First interaction
    input1 := map[string]interface{}{
        "input": "My name is Bob",
    }
    result1, _ := agent.Invoke(ctx, input1)
    fmt.Printf("Response 1: %v\n", result1)

    // Second interaction (agent remembers name)
    input2 := map[string]interface{}{
        "input": "What's my name?",
    }
    result2, _ := agent.Invoke(ctx, input2)
    fmt.Printf("Response 2: %v\n", result2) // Should remember "Bob"
}
```

## Step 4: Chat Message History

For more control, use ChatMessageHistory directly:

```go
import "github.com/lookatitude/beluga-ai/pkg/memory"
import "github.com/lookatitude/beluga-ai/pkg/schema"

// Create message history
history := memory.NewChatMessageHistory()

// Add messages
history.AddUserMessage(ctx, "Hello!")
history.AddAIMessage(ctx, "Hi there! How can I help?")

// Get all messages
messages, _ := history.GetMessages(ctx)
for _, msg := range messages {
    fmt.Printf("%s: %s\n", msg.GetType(), msg.GetContent())
}
```

## Step 5: Memory Configuration

```go
// Buffer memory with custom return messages key
mem, err := memory.NewMemory(
    memory.MemoryTypeBuffer,
    memory.WithReturnMessages(true),
    memory.WithInputKey("user_input"),
    memory.WithOutputKey("assistant_output"),
)
```

## Step 6: Complete Example with Memory

```go
package main

import (
    "context"
    "fmt"
    "os"

    "github.com/lookatitude/beluga-ai/pkg/agents"
    "github.com/lookatitude/beluga-ai/pkg/agents/tools"
    "github.com/lookatitude/beluga-ai/pkg/llms"
    "github.com/lookatitude/beluga-ai/pkg/memory"
)

func main() {
    ctx := context.Background()

    // Setup
    config := llms.NewConfig(
        llms.WithProvider("openai"),
        llms.WithModelName("gpt-3.5-turbo"),
        llms.WithAPIKey(os.Getenv("OPENAI_API_KEY")),
    )
    factory := llms.NewFactory()
    llm, _ := factory.CreateProvider("openai", config)

    // Create memory
    mem, _ := memory.NewMemory(memory.MemoryTypeBuffer)

    // Create agent
    agentTools := []tools.Tool{
        tools.NewCalculatorTool(),
    }
    agent, _ := agents.NewBaseAgent("assistant", llm, agentTools)
    agent.Initialize(map[string]interface{}{
        "memory": mem,
    })

    // Conversation
    conversations := []string{
        "I like pizza",
        "What food do I like?",
        "Calculate 10 + 5",
        "What was the result of the calculation?",
    }

    for _, userInput := range conversations {
        input := map[string]interface{}{
            "input": userInput,
        }
        result, _ := agent.Invoke(ctx, input)
        fmt.Printf("User: %s\n", userInput)
        fmt.Printf("Agent: %v\n\n", result)
    }
}
```

## Step 7: Memory Persistence

For production, you may want to persist memory:

```go
// Example: Save memory to file (simplified)
func saveMemory(mem memory.Memory, filename string) error {
    vars, _ := mem.LoadMemoryVariables(context.Background(), map[string]any{})
    data, _ := json.Marshal(vars)
    return os.WriteFile(filename, data, 0644)
}

// Load memory from file
func loadMemory(filename string) (map[string]any, error) {
    data, err := os.ReadFile(filename)
    if err != nil {
        return nil, err
    }
    var vars map[string]any
    json.Unmarshal(data, &vars)
    return vars, nil
}
```

## Step 8: Advanced: Vector Store Memory

For semantic search over conversation history:

```go
import "github.com/lookatitude/beluga-ai/pkg/memory"
import "github.com/lookatitude/beluga-ai/pkg/vectorstores"
import "github.com/lookatitude/beluga-ai/pkg/embeddings"

// Setup embedder and vector store
embedder, _ := setupEmbedder(ctx)
vectorStore, _ := vectorstores.NewInMemoryStore(ctx, 
    vectorstores.WithEmbedder(embedder))

// Create vector store memory
mem, err := memory.NewMemory(
    memory.MemoryTypeVectorStore,
    memory.WithVectorStore(vectorStore),
    memory.WithEmbedder(embedder),
)
```

## Exercises

1. **Conversation tracking**: Build a chatbot that remembers user preferences
2. **Context management**: Implement a system that maintains context across sessions
3. **Memory limits**: Use window memory to manage long conversations
4. **Memory persistence**: Save and load conversation history
5. **Semantic search**: Use vector store memory for conversation search

## Next Steps

Congratulations! You've learned memory management. Next, learn how to:

- **[Part 6: Orchestration Basics](./06-orchestration-basics.md)** - Build complex workflows
- **[Part 7: Production Deployment](./07-production-deployment.md)** - Deploy your applications
- **[Concepts: Memory](../concepts/memory.md)** - Deep dive into memory concepts

---

**Ready for the next step?** Continue to [Part 6: Orchestration Basics](./06-orchestration-basics.md)!

