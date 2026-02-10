// Package mockmemory provides a mock implementation of the memory.Memory
// interface for testing.
//
// This is an internal package and is not part of the public API. It is used
// by agent tests and memory integration tests that need an in-memory store
// without external dependencies.
//
// # MockMemory
//
// [MockMemory] stores messages in-memory and returns them from Load. It
// supports Save, Load, Search, and Clear operations, with configurable
// error injection and call tracking for assertions.
//
// Create a mock with functional options:
//
//	mem := mockmemory.New(
//	    mockmemory.WithMessages([]schema.Message{
//	        schema.NewHumanMessage("hello"),
//	    }),
//	)
//
// Pre-load documents for Search:
//
//	mem := mockmemory.New(
//	    mockmemory.WithDocuments([]schema.Document{
//	        {ID: "1", Content: "relevant info"},
//	    }),
//	)
//
// Configure error injection:
//
//	mem := mockmemory.New(
//	    mockmemory.WithError(errors.New("storage unavailable")),
//	)
//
// Inspect call history:
//
//	_ = mem.Save(ctx, input, output)
//	fmt.Println(mem.SaveCalls()) // 1
//	fmt.Println(mem.Messages())  // all stored messages
//
// The mock is safe for concurrent use. Call [MockMemory.Reset] to clear
// all state between test cases.
package mockmemory
