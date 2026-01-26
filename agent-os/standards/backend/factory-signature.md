# Factory Signature

Registry factories MUST have one of:

- `func(*Config) (Interface, error)` — when the constructor does not need `context` (e.g. no I/O, no cancellation).
- `func(ctx context.Context, config *Config) (Interface, error)` — when the constructor does I/O, calls other services, or must respect cancellation.

Config is always `*Config` (pointer). Return the package's public interface, not a concrete type.
