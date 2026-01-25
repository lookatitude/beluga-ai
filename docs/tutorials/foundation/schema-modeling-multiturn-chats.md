# Modeling Multi-turn Chats

<!--
Persona: Pair Programmer Colleague
- Treat the reader as a competent peer
- Be helpful, direct, and slightly informal but professional
- Focus on getting results quickly
- Keep it concise and functional
- Provide runnable code examples
-->

## What you will build
In this guide, we'll explore how to model multi-turn chats using Beluga AI's `schema` package. We'll move beyond simple strings and learn how to use structured `Message` types and `ChatHistory` management to create robust conversational agents.

## Learning Objectives
By the end of this tutorial, you will understand how to:
1.  Create and differentiate between various message types (Human, AI, System).
2.  Use `ChatHistory` to manage conversation state automatically.
3.  Implement a simple conversation loop.
4.  Handle context constraints by limiting message history.

## Introduction
Welcome, colleague! If you've ever built a chatbot, you know that keeping track of "who said what" is the most fundamental part of the experience. Unlike a simple Q&A system where each query is independent, a multi-turn chat relies on **context**—the history of previous messages—to provide relevant and coherent responses.

## Why This Matters

*   **Context Awareness**: LLMs (Large Language Models) are stateless. To have a conversation, you must send the entire history (or a relevant summary) with every new request. The `schema` package automates this structure.
*   **Structured Data**: Conversations aren't just text. They contain roles (User vs. AI) and special signals (function calls, system instructions).
*   **Memory Management**: As conversations grow, they eat up tokens. Effective history management prevents you from hitting token limits or confusing the model.

## Prerequisites

Before we start, ensure you have:
*   A working Go environment.
*   The Beluga AI framework installed.
*   Basic familiarity with Go interfaces.

## Concepts

### 1. Message Roles
In Beluga AI, every piece of a conversation is a `Message`. The most important attribute of a message is its **Role**, which tells the LLM how to interpret the content.

| Role | Constant | usage |
| :--- | :--- | :--- |
| **System** | `schema.RoleSystem` | Sets the behavior, persona, or rules for the AI (e.g., "You are a helpful coding assistant"). |
| **Human** | `schema.RoleHuman` | Represents the user's input. |
| **AI** | `schema.RoleAssistant` | Represents the model's response. |
| **Tool/Function** | `schema.RoleTool` | Represents the result of an external tool called by the model. |

### 2. Chat History
`ChatHistory` is an interface that acts as the "memory" of your application. Instead of manually appending strings to a slice, you use `ChatHistory` to `AddUserMessage`, `AddAIMessage`, and retrieve the full list when needed. It handles storage and can even enforce limits (like "keep only the last 10 messages").

## Step-by-Step Implementation

### Step 1: Creating Core Messages

Let's start by creating the basic building blocks of a conversation. We'll use the factory functions provided by `pkg/schema`.
```go
package main

import (
	"fmt"
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	// 1. System Message: Define the persona
	sysMsg := schema.NewSystemMessage("You are a concise data analyst.")

	// 2. Human Message: The user's query
	humanMsg := schema.NewHumanMessage("What is the average rainfall in Seattle?")

	// 3. AI Message: The model's hypothetical response
	// (In a real app, this comes from the LLM)
	aiMsg := schema.NewAIMessage("Seattle receives an average of 38 inches of rain per year.")

	fmt.Printf("System: %s\n", sysMsg.GetContent())
	fmt.Printf("User:   %s\n", humanMsg.GetContent())
	fmt.Printf("AI:     %s\n", aiMsg.GetContent())
}
```

**Why do we do this?**
Using structured types instead of strings ensures that when we eventually send this data to an LLM provider (like OpenAI or Anthropic), the correct "role" fields are automatically populated.

### Step 2: initializing Chat History

Now, let's manage these messages using `ChatHistory`. This relieves us from managing slices manually.
```go
package main

import (
	"fmt"
	"log"
	
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	// Create a new in-memory chat history configuration
	// We pass options to configure it (more on this in Step 4)
	history, err := schema.NewBaseChatHistory()
	if err != nil {
		log.Fatalf("Failed to create history: %v", err)
	}

	// Add the system prompt first - context is key!
	history.AddMessage(schema.NewSystemMessage("You are a helpful assistant."))

	// Add a user interaction
	history.AddUserMessage("Hi, I'm defining a workflow.")
	
	// Add the AI's response
	history.AddAIMessage("That sounds great! What kind of workflow are you building?")

	// Check what we have stored
	msgs, _ := history.Messages()
	fmt.Printf("Total Messages: %d\n", len(msgs))
}
```

### Step 3: Building a Conversation Loop

Let's simulate a real multi-turn conversation. We'll create a loop where we add user input and simulate an AI response.
```go
package main

import (
	"fmt"
	"log"
	
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	history, _ := schema.NewBaseChatHistory()
	
	// Seed the conversation
	history.AddUserMessage("Hello!")
	history.AddAIMessage("Hi there! How can I help you today?")
	
	// Simulate a second turn
	newQuery := "I need to calculate the fibonacci sequence."
	fmt.Printf("User input: %s\n", newQuery)
	history.AddUserMessage(newQuery)
	
	// In a real app, you would pass 'history.Messages()' to the LLM here.
	// We'll simulate the LLM's response:
	mockResponse := "Sure! Here is a Go function to calculate Fibonacci numbers..."
	history.AddAIMessage(mockResponse)

	// Review the full context that would be sent to the model next time
	messages, err := history.Messages()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println("\n--- Current Conversation State ---")
	for _, m := range messages {
		// print based on role
		switch m.GetType() {
		case schema.RoleHuman:
			fmt.Printf("User: %s\n", m.GetContent())
		case schema.RoleAssistant:
			fmt.Printf("AI:   %s\n", m.GetContent())
		}
	}
}
```

### Step 4: Managing Context Windows

One of the biggest challenges in LLM apps is the **Context Window**. You can't send infinite history. `ChatHistoryConfig` helps us manage this by automatically maintaining a sliding window of messages.

Let's configure a history that only keeps the last 4 messages.
```go
package main

import (
	"fmt"
	"log"
	
	"github.com/lookatitude/beluga-ai/pkg/schema"
)

func main() {
	// Configure history to keep only the last 4 messages using functional options
	history, err := schema.NewBaseChatHistory(
		schema.WithMaxMessages(4),
	)
	if err != nil {
		log.Fatalf("Failed to create history: %v", err)
	}

	// Let's add 6 messages
	for i := 1; i <= 6; i++ {
		msg := fmt.Sprintf("Message #%d", i)
		history.AddUserMessage(msg)
		fmt.Printf("Added: %s\n", msg)
	}

	// Retrieve messages
	msgs, _ := history.Messages()
	
	fmt.Printf("\nTotal messages stored: %d\n", len(msgs))
	fmt.Println("--- Stored Messages ---")
	for _, m := range msgs {
		fmt.Println(m.GetContent())
	}
}
```

> **Note**: When `MaxMessages` is exceeded, `BaseChatHistory` automatically drops the *oldest* messages to make room for new ones, acting like a FIFO (First-In-First-Out) queue.

## Pro-Tips

*   **System Prompt Persistence**: When using `MaxMessages` (trimming), be careful! You usually want your `SystemMessage` (instruction) to *never* be deleted. A common pattern is to keep the System Message separate and append it to the chat history only when sending the request to the LLM.
*   **Token Counting**: While `MaxMessages` is simple, advanced implementations often limit by *token count* rather than message count to better optimize for LLM context limits.
*   **Persisting History**: Since `BaseChatHistory` is in-memory, you lose data when the program restarts. For real apps, consider implementing a `RedisChatHistory` or `PostgresChatHistory` that implements the `schema.ChatHistory` interface.

## Troubleshooting

### "My history isn't saving?"
Recall that `BaseChatHistory` is ephemeral (in-memory). If you re-initialize the variable `history := ...` inside a loop or request handler, you are creating a fresh history every time. Ensure your history object persists across the scope of the full conversation session.

### "I see empty messages."
Ensure you are using the factory methods (`NewHumanMessage`, etc.) which handle initialization correctly. Directly creating structs might leave internal fields uninitialized.

## Conclusion

You've successfully modeled a multi-turn chat! You moved from simple strings to a structured conversation model.

Here is what we achieved:
1.  Created structured **Messages** with Roles.
2.  Built a **ChatHistory** to track the conversation.
3.  Simulated a distinct **User -> AI -> User** loop.
4.  Learned how to **limit history** to manage context.

This pattern is the foundation of every advanced agent you will build in Beluga AI. As you move to `pkg/core` and `pkg/agents`, you'll see these exact same structures used to drive complex reasoning engines!
