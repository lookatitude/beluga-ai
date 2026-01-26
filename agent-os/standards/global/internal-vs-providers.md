# internal/ vs providers/

**internal/** — Private abstractions used only inside the package. Not exported. Use for:

- Schedulers, message buses, worker pools, shared helpers
- Implementation details that are not swappable by name

**providers/** — All swappable implementations that are registered and selected by name. Use for:

- Every provider that appears in the registry/factory
- Each provider in its own `providers/<name>/` subpackage

**Rule:** No providers under `internal/`. If it is registered and chosen by name, it lives in `providers/`.
