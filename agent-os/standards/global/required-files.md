# Required File Set

Every `pkg/` package MUST include:

- `iface/` — interfaces and shared types
- `config.go` — configuration and validation
- `metrics.go` — OTEL metrics (and tracing where used)
- `errors.go` — custom errors (Op/Err/Code)
- `test_utils.go` — test helpers and mocks
- `advanced_test.go` — broader/edge-case tests
- `README.md` — package overview and usage

Main API: `{package_name}.go` (e.g. `llms.go`, `embeddings.go`).

**Multi-provider packages** (with `providers/`): also need either `registry.go` or `factory.go` for registration and creation. No exceptions.

**Why:** This project is a library/framework; consistent layout across packages makes onboarding, navigation, and tooling predictable.
