---
description: How to release a new version of Beluga AI
---
# Release Process Workflow

1.  **Verify Main Branch**
    Ensure `main` is stable and all CI checks are passing.

2.  **Tag Release**
    ```bash
    # e.g., v1.0.0
    git tag -a <version> -m "Release <version>"
    ```

3.  **Push Tag**
    ```bash
    git push origin <version>
    ```

4.  **Monitor Release**
    -   Check the `.github/workflows/release.yml` action.
    -   Verify the GitHub Release is created with artifacts.

5.  **Verify Documentation**
    -   Ensure documentation version references are updated if necessary.
