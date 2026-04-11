---
description: Website development rules for Beluga AI v2. Auto-loaded when editing website files.
globs: "website/**/*"
alwaysApply: false
---

# Website Rules (Astro + Starlight)

## Stack

- Astro 4+ with Starlight theme
- TypeScript for interactive components
- Tailwind for styling
- MDX for documentation pages
- gomarkdoc for Go API reference generation

## Content rules

- All Go code examples must compile — verify with `go build` before committing.
- Examples include full imports (`github.com/lookatitude/beluga-ai/...`).
- Errors in examples must be handled explicitly — never `_` for error returns.
- No placeholder Lorem ipsum — real content from `docs/`.

## Design rules

- Responsive at 320px / 768px / 1024px / 1440px.
- WCAG AA accessibility.
- Navigation matches the blueprint's mega-menu structure.
- Reuse existing component patterns — don't invent new primitives.

## Before editing

- Read `.wiki/index.md` retrieval routing table for website tasks.
- Read `docs/beluga-ai-website-blueprint-v2.md` for the target architecture.
- Explore current `website/src/` structure — match existing conventions.

## Anti-rationalization

| Excuse | Counter |
|---|---|
| "The example is obvious, no imports needed" | Full imports every time. Copy-paste ready. |
| "I'll test responsive later" | Test at all four breakpoints now. |
| "This new component is just for this page" | Use existing patterns unless there's a documented reason. |
