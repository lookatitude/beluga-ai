# Config.Enabled and NoOpMemory

When `Config.Enabled` is false, the factory returns a **NoOpMemory** instance (non-nil), not nil. This keeps call sites consistent: they can always call Memory methods without nil checks.

- **NoOpMemory:** Implements Memory with no-op MemoryVariables (empty), LoadMemoryVariables (empty map), SaveContext (no-op), Clear (no-op). Use when `Config.Enabled == false`.
- **Why non-nil:** Callers that receive Memory from the factory can assume a valid implementation. No `if mem != nil` before SaveContext/LoadMemoryVariables/Clear when the only "disabled" path is `Enabled: false`.
- **Nil when optional:** `nil` is acceptable when memory is **optional and not supplied** (e.g. no Config, or an explicit "no memory" choice). In those cases the caller must nil-check. When Config is present and `Enabled` is false, use NoOpMemory.
