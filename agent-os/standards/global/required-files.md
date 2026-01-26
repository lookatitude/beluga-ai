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

## Wrapper Package Required Files

Wrapper/aggregation packages (e.g., `voice`, `orchestration`) that compose multiple sub-packages require:

- `iface/` — shared interfaces across sub-packages
- `config.go` — root config embedding sub-package configs
- `metrics.go` — aggregated metrics from sub-packages
- `errors.go` — custom errors (Op/Err/Code)
- `registry.go` — facade registry delegating to sub-package registries
- `{package_name}.go` — facade API entry point
- `test_utils.go` — composite mocks for all sub-packages
- `advanced_test.go` — cross-sub-package integration tests
- `README.md` — package overview including sub-package documentation

## Sub-Package Required Files

Each sub-package within a wrapper (e.g., `voice/stt/`, `voice/tts/`) requires the **full standard file set**:

- `iface/` — sub-package interfaces
- `providers/` — provider implementations (if multi-provider)
- `config.go` — sub-package specific config
- `metrics.go` — sub-package OTEL metrics
- `errors.go` — sub-package errors
- `registry.go` — sub-package registry (if multi-provider)
- `{subpackage_name}.go` — main API
- `test_utils.go` — sub-package mocks
- `advanced_test.go` — comprehensive tests
- `README.md` — sub-package documentation

Sub-packages MUST be independently importable and testable. They MUST NOT import the parent package.

**Why:** This project is a library/framework; consistent layout across packages makes onboarding, navigation, and tooling predictable.
