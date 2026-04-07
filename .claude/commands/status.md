---
name: status
description: Check package health and structure for Beluga AI v2.
---

Check health and structure of Beluga AI v2 packages.

## Steps

1. List existing top-level packages.
2. For each, check: file count, test count, registry (Register func), hooks (Hooks struct), middleware (Middleware func), compiles (`go build`).
3. Present as status table: Package | Files | Tests | Registry | Hooks | Middleware | Compiles.
