// Package testutil provides test helpers and assertion utilities for the
// Beluga AI framework.
//
// This is an internal package and is not part of the public API. It is used
// across the framework's test suites to reduce boilerplate and provide
// consistent assertion patterns.
//
// # Assertion Helpers
//
// The package provides lightweight assertion functions that fail the test
// immediately on mismatch:
//
//   - [AssertNoError] — fails if err is non-nil
//   - [AssertError] — fails if err is nil
//   - [AssertEqual] — performs deep equality comparison
//   - [AssertContains] — checks string containment
//
// Example:
//
//	result, err := agent.Run(ctx, "hello")
//	testutil.AssertNoError(t, err)
//	testutil.AssertContains(t, result.Text(), "world")
//
// # Stream Collector
//
// [CollectStream] drains an iter.Seq2[T, error] iterator into a slice,
// stopping on the first error. This is useful for testing streaming
// interfaces:
//
//	chunks, err := testutil.CollectStream(model.Stream(ctx, msgs))
//	testutil.AssertNoError(t, err)
//	testutil.AssertEqual(t, 3, len(chunks))
//
// # Mock Packages
//
// Dedicated mock implementations for core interfaces are available in
// sub-packages:
//
//   - [github.com/lookatitude/beluga-ai/internal/testutil/mockllm] — mock ChatModel
//   - [github.com/lookatitude/beluga-ai/internal/testutil/mocktool] — mock Tool
//   - [github.com/lookatitude/beluga-ai/internal/testutil/mockmemory] — mock Memory
//   - [github.com/lookatitude/beluga-ai/internal/testutil/mockembedder] — mock Embedder
//   - [github.com/lookatitude/beluga-ai/internal/testutil/mockstore] — mock VectorStore
//   - [github.com/lookatitude/beluga-ai/internal/testutil/mockworkflow] — mock WorkflowStore
package testutil
