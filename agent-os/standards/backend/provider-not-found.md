# Provider-Not-Found

When `Get`/`Create` is called with an unknown `name`, return the package's error type, not a raw `fmt.Errorf` or `errors.New`.

- **Code:** `ErrCodeUnsupportedProvider` in every package.
- **Wrapping:** Wrap a descriptive error in the package error, e.g. `NewXxxError("GetProvider", ErrCodeUnsupportedProvider, fmt.Errorf("provider '%s' not registered", name))`.
