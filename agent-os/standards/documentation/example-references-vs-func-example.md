# Example References vs func Example

Examples live either in `examples/` (runnable, with their own tests) or as `func ExampleX()` in the package's `_test.go` next to the code they demonstrate. In godoc, reference `examples/` only when that is where the example is actually implemented.

- **examples/** — Runnable examples (e.g. `examples/llms/basic/main.go`). Tests in `examples/` unit-test the example itself. Follow example naming conventions. In godoc, use "Example usage can be found in examples/.../main.go" **only when** that file is the canonical implementation of the example.
- **func ExampleX()** — Lives **next to the code it demonstrates**: in the same package's `_test.go` (e.g. `pkg/llms/examples_test.go`). Use when the example is implemented there rather than in `examples/`. `go doc` and `godoc -play` can run these.
- **Do not** reference `examples/` in godoc when the example is actually implemented as `func ExampleX()` in the package.
