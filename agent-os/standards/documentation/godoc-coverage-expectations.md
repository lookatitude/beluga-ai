# Godoc Coverage Expectations

All exported symbols MUST have godoc. New*, factories, interface methods, and helpers follow the same rules. Complex logic MUST be explained in both godoc and inline `//` where non-obvious.

- **All exported functions and types:** Godoc required. First-sentence summary; add Parameters, Returns, and Example when applicable (see function-and-type-godoc). Document error conditions for functions that return errors.
- **New* and factories:** Include Parameters, Returns, and at least an Example or "Example usage can be found in ..." (or func ExampleX()).
- **Interface methods:** Document purpose, parameters, return values, and when errors occur. Implementations may reference the interface's godoc.
- **Complex logic:** Explain in godoc (e.g. in the function's main comment or type overview). Also add inline `//` in the body for non-obvious steps, branches, or algorithms.
- **Provider checklist:** For provider packages, "Public functions documented" and "Complex logic explained" apply to all exported API used by callers.
