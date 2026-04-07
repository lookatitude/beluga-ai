---
description: Security constraints for all Beluga AI v2 code. Enforced on every code change.
globs: "*.go"
alwaysApply: false
---

# Security Rules

## Input Validation

- Validate and sanitize ALL external input at system boundaries.
- Parameterized queries only — no string concatenation for SQL.
- Never pass user input directly to `os/exec.Command()`.
- Use `filepath.Clean()` and reject paths containing `..` for any file path from external input.
- Validate JSON schemas at API boundaries before processing.

## Secrets Management

- NEVER hardcode API keys, passwords, tokens, or credentials in source code.
- Secrets come from environment variables or config files — never from code.
- NEVER log, trace, or include secrets in error messages.
- NEVER include secrets in test files — use environment variables or test fixtures.
- Redact sensitive fields before serialization (logs, traces, API responses).

## Authentication & Authorization

- Every HTTP handler and RPC endpoint must enforce auth checks.
- Use capability-based access control — deny by default, explicitly grant.
- Multi-tenancy: isolate data by tenant. Use `core.WithTenant(ctx)` for namespace scoping.
- Validate JWT/token signatures — never trust unverified claims.

## Cryptography

- No MD5 or SHA1 for security purposes (hashing passwords, HMAC, signatures).
- Use `crypto/rand` for random values — never `math/rand` for security.
- No ECB mode for block ciphers.
- TLS 1.2+ for all external connections.

## Concurrency Safety

- No goroutine leaks: every goroutine must be cancellable via context.
- No race conditions: protect shared state with proper synchronization.
- Bounded concurrency: use worker pools and semaphores, not unbounded goroutine spawning.
- Always `defer` resource cleanup (connections, files, locks, transactions).

## Error Handling

- Errors must not leak internal details (stack traces, file paths, SQL queries) to external callers.
- Use typed errors with appropriate codes — not raw error strings.
- Log detailed errors internally, return sanitized errors externally.
- Never swallow errors silently — at minimum, log them.

## Resource Protection

- No unbounded allocations from external input (limit request body size, array lengths, string lengths).
- Set timeouts on all external calls (HTTP, database, LLM providers).
- Use `context.WithTimeout()` or `context.WithDeadline()` for all blocking operations.
- Rate limit external-facing endpoints.

## Dependencies

- Zero external deps in `core/` and `schema/` beyond stdlib + otel.
- Review new dependencies for known vulnerabilities before adding.
- Pin dependency versions in `go.mod` — no floating versions.
- Run `go mod tidy` to remove unused dependencies.

## Prompt Injection (LLM-specific)

- Apply input guards before sending user content to LLM.
- Use spotlighting (data delimiters) to separate instructions from user data.
- Apply output guards to LLM responses before returning to users.
- Apply tool guards before executing any tool call from LLM output.
- The guard pipeline is always 3-stage: Input → Output → Tool.

## Unsafe Patterns (NEVER use)

- `unsafe` package — unless there is an extraordinary, documented reason.
- `reflect` for security-sensitive operations.
- `//go:nosplit`, `//go:noescape` without justification.
- Global mutable state beyond `init()` registrations.
- `panic()` for recoverable errors — return errors instead.
