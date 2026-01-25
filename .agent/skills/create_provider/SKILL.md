---
name: Create New Provider
description: Step-by-step guide to adding a new provider (LLM, VectorStore, etc.) to Beluga AI.
---

# Create New Provider

This skill guides you through adding a new provider implementation to an existing package.

## Prerequisites
-   Ensure the package supports the provider pattern (e.g., `pkg/llms`, `pkg/vectorstores`).
-   Identify the interface to implement (usually in `pkg/<package>/iface` or `pkg/<package>/<package>.go`).

## Steps

1.  **Create Provider Directory**
    -   Create `pkg/<package>/providers/<provider_name>/`.
    -   Create `pkg/<package>/providers/<provider_name>/<provider_name>.go`.

2.  **Implement Interface**
    -   Define a struct `Provider` (or similar) that implements the interface.
    -   Implement the `New(options ...)` factory function.

3.  **Add Configuration**
    -   Define a config struct.
    -   Use `mapstructure` tags for decoding.

4.  **Register Global Factory**
    -   In `pkg/<package>/registry.go` (or `factory.go`), add the init block or manual registration for your new provider.

5.  **Add Testing**
    -   Create `pkg/<package>/providers/<provider_name>/<provider_name>_test.go`.
    -   Use `test_utils` for mocks.

## Example File Structure
```
pkg/llms/providers/anthropic/
├── anthropic.go       # Implementation
├── config.go          # Config struct
└── anthropic_test.go  # Tests
```
