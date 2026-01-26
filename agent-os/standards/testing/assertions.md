# Assertions

**Library:** Prefer `testify/assert` and `testify/require` (e.g. `require.NoError`, `assert.Equal`) over raw `t.Error`/`t.Fatal` in new tests.

**t.Helper():** In every custom `validate` or helper that calls `t.Error`, `t.Fatal`, `require`, or `assert`, add `t.Helper()` at the top so failures are reported at the caller's line.
