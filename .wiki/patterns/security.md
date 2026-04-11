# Pattern: Security

**Status:** stub — populate with `/wiki-learn`

## Contract

All external input validated at system boundaries. Parameterized queries only. No command injection via `os/exec`. Secrets only from env/config — never in code, logs, errors, or spans. TLS 1.2+. Guard pipeline (Input → Output → Tool) enforced on any LLM-touching code path.

See `.claude/rules/security.md` for the auto-loaded full checklist.

## Canonical example

(populate via `/wiki-learn`)

## Anti-patterns

- String concatenation in SQL.
- `os/exec.Command(userInput)` without argument lists and allowlist.
- `filepath.Join` without `filepath.Clean` + containment check.
- Logging request bodies that may contain PII or secrets.
- `math/rand` for security randomness.

## Related

- `.claude/rules/security.md`
- `architecture/invariants.md#4-guard-pipeline-is-3-stage`
