---
description: How to run local quality and security checks
---
# Quality Check Workflow

1.  **Format Code**
    ```bash
    make fmt
    ```

2.  **Run Linter**
    ```bash
    make lint
    ```

3.  **Run Tests**
    ```bash
    # Run unit tests
    make test

    # Run race detection
    make test-race
    ```

4.  **Run Security Scans**
    ```bash
    make security
    ```

5.  **Run All Checks**
    // turbo
    ```bash
    make all
    ```
