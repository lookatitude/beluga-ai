---
description: How to develop a new feature in Beluga AI
---
# Feature Development Workflow

1.  **Sync with Main**
    ```bash
    git checkout main
    git pull origin main
    ```

2.  **Create Feature Branch**
    ```bash
    # Use format: feat/your-feature-name or fix/your-bug-fix
    git checkout -b feat/<feature-name>
    ```

3.  **Implement Changes**
    -   Follow architecture in `pkg/`.
    -   Add tests in `{package}_test.go`.
    -   Add OTEL metrics in `metrics.go`.

4.  **Run Quality Checks**
    // turbo-all
    ```bash
    make fmt
    make lint
    make test
    make security
    ```

5.  **Commit Changes**
    -   Use Conventional Commits.
    ```bash
    git add .
    git commit -m "feat(scope): description"
    ```

6.  **Push**
    ```bash
    git push -u origin feat/<feature-name>
    ```
