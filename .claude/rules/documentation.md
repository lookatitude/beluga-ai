---
description: Documentation rules for Beluga AI v2. Auto-loaded when editing docs.
globs: "docs/**/*.md, **/*.md"
alwaysApply: false
---

# Documentation Rules

## Every doc includes

1. **Concept overview** — what and why, 2-3 paragraphs.
2. **Quick start** — working code example, <20 lines, copy-paste ready.
3. **API reference** — every exported type and function documented.
4. **Full example** — realistic use case, compilable.
5. **Common mistakes** — sourced from `.wiki/corrections.md`.
6. **Related packages** — cross-references.

## Code example requirements

- All Go examples must compile. Verify with `go build` on a throwaway file.
- Full imports with module path `github.com/lookatitude/beluga-ai/v2/...`.
- Handle errors explicitly — never `_` for error returns.
- No pseudocode in reference docs.

## Writing style

- No marketing language. Technical precision.
- Show, don't tell — every concept needs a code example.
- Lead with the problem, not the solution.
- Cross-reference related packages and concepts.

## Sources to consult before writing

- `.wiki/architecture/package-map.md` — what the package does and what it depends on.
- `.wiki/patterns/*.md` — canonical implementation snippets.
- `.wiki/corrections.md` — real mistakes users hit.
- `.wiki/architecture/decisions.md` — design rationale for the "why" sections.

## Anti-rationalization

| Excuse | Counter |
|---|---|
| "The example is trivial, imports aren't important" | Full imports every time. |
| "Users will understand without error handling" | Show errors. That's how the library is actually used. |
| "I'll fix the code example later" | Broken examples erode trust. Fix before commit. |
| "This concept is too simple to need a code example" | Every concept gets a code example. |
