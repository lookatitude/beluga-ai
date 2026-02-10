// Package mocktool provides a mock implementation of the Tool interface
// for testing.
//
// This is an internal package and is not part of the public API. It is used
// by agent tests, tool execution tests, and guard pipeline tests that need
// a controllable tool without side effects.
//
// # MockTool
//
// [MockTool] implements the tool.Tool interface with configurable name,
// description, input schema, and execution behavior. It supports canned
// results, error injection, custom execute functions, and call tracking
// for assertions.
//
// Create a mock with functional options:
//
//	t := mocktool.New("search", "Search the web",
//	    mocktool.WithResult(&schema.ToolResult{
//	        Content: []schema.ContentPart{schema.TextPart{Text: "result"}},
//	    }),
//	)
//
// Configure error injection:
//
//	t := mocktool.New("search", "Search the web",
//	    mocktool.WithError(errors.New("network timeout")),
//	)
//
// Use a custom function for dynamic behavior:
//
//	t := mocktool.New("calc", "Calculator",
//	    mocktool.WithExecuteFunc(func(ctx context.Context, input map[string]any) (*schema.ToolResult, error) {
//	        // custom execution logic
//	    }),
//	)
//
// Inspect call history:
//
//	_, _ = t.Execute(ctx, map[string]any{"q": "test"})
//	fmt.Println(t.ExecuteCalls()) // 1
//	fmt.Println(t.LastInput())    // {"q": "test"}
//
// The mock is safe for concurrent use. Call [MockTool.Reset] to clear
// all state between test cases.
package mocktool
