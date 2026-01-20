---
name: Create New Package
description: Guide to creating a new standardized package in Beluga AI.
---

# Create New Package

This skill ensures your new package follows the strict Beluga AI architecture.

## Steps

1.  **Create Directory Structure**
    ```bash
    mkdir -p pkg/<package_name>/{iface,internal,providers}
    ```

2.  **Create Required Files**
    -   `pkg/<package_name>/<package_name>.go`: Main entry point / interfaces.
    -   `pkg/<package_name>/config.go`: Configuration structs and validation.
    -   `pkg/<package_name>/metrics.go`: OTEL metrics setup.
    -   `pkg/<package_name>/errors.go`: Custom error definitions.
    -   `pkg/<package_name>/test_utils.go`: Test helpers and mocks.

3.  **Define Interfaces**
    -   Place primary interfaces in `iface/` or the root package file.

4.  **Implement OTEL Metrics**
    -   In `metrics.go`, define `NewMetrics(meter)`.
    -   Create counters/histograms for operations.

5.  **Add Documentation**
    -   Create `pkg/<package_name>/README.md`.

## Validation
-   Run `make lint` to ensure no linting errors.
-   Run `go test ./pkg/<package_name>/...` to verify tests.
