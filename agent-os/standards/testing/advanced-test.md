# advanced_test.go

**Required.** Every package must have an `advanced_test.go` (see also `global/required-files`).

**Role:** Broader, edge-case, and integration-style tests: table-driven, concurrency, error paths, timeouts, context cancellation. Complements `{package}_test.go` and `{thing}_test.go`, which focus on unit tests of a single type or function.
