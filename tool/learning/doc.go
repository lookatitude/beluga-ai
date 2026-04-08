// Package learning provides tool learning and creation capabilities for agents.
//
// It enables agents to dynamically create, validate, version, and register tools
// at runtime. Unlike FuncTool which requires compile-time generics, DynamicTool
// supports runtime-defined schemas and code execution.
//
// Key components:
//
//   - DynamicTool: A tool.Tool implementation with runtime-defined input schema
//     and a pluggable CodeExecutor for execution.
//
//   - CodeExecutor: Interface for executing tool code. Includes ASTValidator
//     for static analysis (go/ast allowlist checking) and NoopExecutor for testing.
//
//   - ToolGenerator: Uses an llm.ChatModel to generate tool code from natural
//     language descriptions, with function signature templates and few-shot prompts.
//
//   - VersionedRegistry: Wraps tool.Registry with immutable version tracking,
//     supporting Upsert, Activate, Rollback, and History operations.
//
//   - ToolTester: Validates generated tools by running test inputs before
//     registration, ensuring tools produce expected outputs.
//
//   - Hooks: Lifecycle callbacks (OnToolCreated, OnToolTested, OnVersionActivated)
//     for observability and integration.
package learning
