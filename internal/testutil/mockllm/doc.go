// Package mockllm provides a mock implementation of the ChatModel interface
// for testing.
//
// This is an internal package and is not part of the public API. It is used
// by agent tests, tool execution tests, and any test that requires an LLM
// without calling a real provider.
//
// # MockChatModel
//
// [MockChatModel] provides configurable Generate and Stream methods with
// canned responses, error injection, streaming chunks, and call tracking.
//
// Note: MockChatModel.BindTools returns *MockChatModel (not llm.ChatModel),
// so tests that need the llm.ChatModel interface should define a local mock
// or use a type assertion.
//
// Create a mock with functional options:
//
//	model := mockllm.New(
//	    mockllm.WithResponse(&schema.AIMessage{
//	        Parts: []schema.ContentPart{schema.TextPart{Text: "Hello!"}},
//	    }),
//	)
//
// Configure streaming:
//
//	model := mockllm.New(
//	    mockllm.WithStreamChunks([]schema.StreamChunk{
//	        {Delta: "Hello"},
//	        {Delta: " World"},
//	    }),
//	)
//
// Configure error injection:
//
//	model := mockllm.New(
//	    mockllm.WithError(errors.New("rate limit exceeded")),
//	)
//
// Inspect call history:
//
//	_, _ = model.Generate(ctx, msgs)
//	fmt.Println(model.GenerateCalls()) // 1
//	fmt.Println(model.LastMessages())  // the messages passed to Generate
//
// The mock is safe for concurrent use. Call [MockChatModel.Reset] to clear
// all state between test cases.
package mockllm
