# Error Predicates and Extraction

For every main error type, provide:

- **IsXxxError(err)** — `errors.As(err, &xxxErr)`; use for type checks.
- **GetXxxError(err)** — returns `*XxxError` or `nil`; use when you need the struct.
- **GetXxxErrorCode(err)** — returns the `Code` string or `""`; use when branching on codes.

**IsRetryable(err):** Only in packages where retry vs not is part of the contract (e.g. LLM calls, orchestration). Implement with `errors.As` and a switch on `Code` (and/or domain-specific error types). Do not add in packages where retry is not part of the design.
