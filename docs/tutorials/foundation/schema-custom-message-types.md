# Custom Message Types for Enterprise Data

In this tutorial, you'll learn how to extend Beluga AI's schema with custom message types to handle complex enterprise data structures within your conversation flows.

## Learning Objectives

- ✅ Create custom message types implementing the `Message` interface
- ✅ Serialize and deserialize custom messages
- ✅ Use custom messages with agents
- ✅ Implement validation for enterprise data

## Prerequisites

- Basic understanding of Beluga AI schema (see [Modeling Multi-turn Chats](./schema-modeling-multiturn-chats.md))
- Go 1.24+

## Why Custom Message Types?

Standard messages (`HumanMessage`, `AIMessage`, `SystemMessage`) handle text well. But enterprise applications often need to pass structured data like:
- Customer profiles
- Transaction records
- Compliance events
- IoT sensor data

Custom message types allow you to maintain type safety and validation while passing this data through your AI pipeline.

## Step 1: Define Custom Message Structure

Let's define a `TransactionMessage` that holds financial transaction details.
```go
package main

import (
    "encoding/json"
    "fmt"
    "time"

    "github.com/lookatitude/beluga-ai/pkg/schema"
)

// TransactionData holds the structured payload
type TransactionData struct {
    TransactionID string    `json:"transaction_id"`
    Amount        float64   `json:"amount"`
    Currency      string    `json:"currency"`
    Timestamp     time.Time `json:"timestamp"`
    Status        string    `json:"status"`
}

// TransactionMessage implements schema.Message
type TransactionMessage struct {
    Data TransactionData
}

// Ensure interface compliance
var _ schema.Message = (*TransactionMessage)(nil)

func NewTransactionMessage(data TransactionData) *TransactionMessage {
    return &TransactionMessage{Data: data}
}

// GetType returns a unique type identifier
func (m *TransactionMessage) GetType() schema.Role {
    return "transaction"
}

// GetContent returns a string representation (for LLMs that expect text)
func (m *TransactionMessage) GetContent() string {
    return fmt.Sprintf("Transaction %s: %s %.2f (%s)", 
        m.Data.TransactionID, m.Data.Currency, m.Data.Amount, m.Data.Status)
}

// GetRole returns the role (usually system or tool for data messages)
func (m *TransactionMessage) GetRole() schema.Role {
    return schema.RoleSystem
}

// Clone creates a deep copy
func (m *TransactionMessage) Clone() schema.Message {
    return &TransactionMessage{Data: m.Data}
}
```

## Step 2: Implement JSON Marshaling

To ensure your custom message works with the framework's serialization (e.g., for memory persistence), implement `MarshalJSON`:
```go
func (m *TransactionMessage) MarshalJSON() ([]byte, error) {
    type Alias TransactionMessage
    return json.Marshal(&struct {
        Type string `json:"type"`
        *Alias
    }{
        Type:  string(m.GetType()),
        Alias: (*Alias)(m),
    })
}
```

## Step 3: Using Custom Messages in a Flow

Now let's use this message in a chat flow.
```go
func main() {
    // Create structured data
    txData := TransactionData{
        TransactionID: "TX-998877",
        Amount:        1250.00,
        Currency:      "USD",
        Timestamp:     time.Now(),
        Status:        "Pending",
    }
    
    // Create custom message
    txMsg := NewTransactionMessage(txData)
    
    // Create standard messages
    sysMsg := schema.NewSystemMessage("You are a financial assistant. Analyze the transaction data provided.")
    userMsg := schema.NewHumanMessage("Is this transaction suspicious?")
    
    // Combine in a conversation
    messages := []schema.Message{
        sysMsg,
        txMsg, // Inject structured data
        userMsg,
    }
    
    // Simulate LLM processing
    // Note: The LLM will see the result of GetContent() for the custom message
    fmt.Println("Sending context to LLM:")
    for _, msg := range messages {
        fmt.Printf("[%s]: %s\n", msg.GetType(), msg.GetContent())
    }
}
```

## Step 4: Advanced - Content Conversion

Sometimes you want `GetContent()` to return JSON for the LLM to parse.
```go
func (m *TransactionMessage) GetContent() string {
    data, _ := json.Marshal(m.Data)
    return string(data)
}
```

## Step 5: Handling Custom Messages in Memory

If you're using `ChatHistory`, you can store these messages just like any others.
```go
func storeTransaction(history schema.ChatHistory, txData TransactionData) error {
    msg := NewTransactionMessage(txData)
    return history.AddMessage(context.Background(), msg)
}
```

## Verification

Run the example code to verify that:
1. The custom message compiles and satisfies the interface.
2. `GetContent()` formats the data correctly.
3. It fits seamlessly into a slice of `[]schema.Message`.

## Next Steps

- **[Modeling Multi-turn Chats](./schema-modeling-multiturn-chats.md)** - Learn about standard message flows
- **[Building a Custom Runnable](./core-custom-runnable.md)** - Create custom processing steps
