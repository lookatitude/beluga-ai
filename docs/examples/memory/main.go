// docs/examples/memory/main.go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/lookatitude/beluga-ai/memory"
	"github.com/lookatitude/beluga-ai/schema"
	// Assuming a mock LLM or a simple implementation for summary memory if needed
)

func main() {
	 ctx := context.Background()

	 // --- Buffer Memory Example --- 
	 fmt.Println("--- Buffer Memory Example ---")
	 bufferMem := memory.NewBufferMemory()

	 // Add messages
	 bufferMem.SaveContext(ctx, map[string]any{"input": "Hello there!"}, map[string]any{"output": "Hi! How can I help you?"})
	 bufferMem.SaveContext(ctx, map[string]any{"input": "What is Beluga-ai?"}, map[string]any{"output": "It is a Go framework for AI agents."}) 

	 // Load messages
	 memVars, err := bufferMem.LoadMemoryVariables(ctx, nil) // No specific input needed for buffer
	 if err != nil {
	 	 log.Printf("BufferMemory Load failed: %v", err)
	 } else {
	 	 history, _ := memVars[bufferMem.MemoryKey()].([]schema.Message)
	 	 fmt.Println("Buffer Memory History:")
	 	 for _, msg := range history {
	 	 	 fmt.Printf(" - %s: %s\n", msg.GetType(), msg.GetContent())
	 	 }
	 }

	 // --- Window Buffer Memory Example --- 
	 fmt.Println("\n--- Window Buffer Memory Example ---")
	 windowMem := memory.NewWindowBufferMemory(memory.WithWindowK(1)) // Keep only last 1 interaction (2 messages)

	 windowMem.SaveContext(ctx, map[string]any{"input": "First input"}, map[string]any{"output": "First output"})
	 windowMem.SaveContext(ctx, map[string]any{"input": "Second input"}, map[string]any{"output": "Second output"})
	 windowMem.SaveContext(ctx, map[string]any{"input": "Third input"}, map[string]any{"output": "Third output"})

	 memVarsWindow, err := windowMem.LoadMemoryVariables(ctx, nil)
	 if err != nil {
	 	 log.Printf("WindowBufferMemory Load failed: %v", err)
	 } else {
	 	 history, _ := memVarsWindow[windowMem.MemoryKey()].([]schema.Message)
	 	 fmt.Println("Window Memory History (k=1):")
	 	 for _, msg := range history {
	 	 	 fmt.Printf(" - %s: %s\n", msg.GetType(), msg.GetContent())
	 	 }
	 }

	 // --- Clearing Memory --- 
	 fmt.Println("\n--- Clearing Memory ---")
	 bufferMem.Clear(ctx)
	 memVarsAfterClear, _ := bufferMem.LoadMemoryVariables(ctx, nil)
	 historyAfterClear, _ := memVarsAfterClear[bufferMem.MemoryKey()].([]schema.Message)
	 fmt.Printf("Buffer Memory History after clear (should be empty): %d messages\n", len(historyAfterClear))

	 // Note: SummaryMemory and VectorStoreMemory examples would require
	 // an LLM instance and a VectorStore implementation respectively.
}

