---
derived_from_template: guild.skill_template.v1
name: doc-writing
description: "Beluga AI v2 technical-documentation conventions for docs/ and godoc: package-doc structure (header, quick start, core interface, usage, configuration, extension guide), compilable examples with full github.com/lookatitude/beluga-ai/v2 imports and explicit error handling, no marketing tone, link don't duplicate architecture docs, point at docs/reference/providers.md instead of listing providers. Pulled by doc-writer. TRIGGER: \"document the X package\", \"write a guide for Y\", \"update docs/architecture after this change\", \"add a tutorial\", \"fix the code example in the docs\", \"godoc for this package\". DO NOT TRIGGER for: blog posts, landing pages or marketing copy (website repo); runnable example programs (examples repo); release notes and CHANGELOG (release flow); wiki pattern extraction under .guild/wiki/ (the /wiki-learn workflow); designing the API being documented (architect)."
when_to_use: The doc-writer specialist (or the docs-writer Claude agent) is writing or verifying framework docs under docs/ or package godoc.
type: specialist
---

# doc-writing

Canonical body (structure, rules, don'ts): `.claude/skills/doc-writing/SKILL.md`. Also binding: `.claude/rules/documentation.md`. Read both first; this file is the Guild routing wrapper.

# When to use it

- Pages under `docs/architecture/`, `docs/guides/`, `docs/patterns/`, `docs/reference/` (`docs/reference/providers.md` is generated — never hand-edit it).
- Package-level godoc and doc comments on exported identifiers.
- `/document`, `/doc-check`, and the doc step of `/arch-update`.

# When not to use it

- Website content, blog, tutorials hosted on the site → `beluga-website` repo.
- Runnable example programs → `beluga-examples` repo.
- `.guild/wiki/` pattern extraction → `/wiki-learn`.

# Required inputs

The source being documented, `.guild/wiki/architecture/package-map.md` entry, relevant `.guild/wiki/patterns/*.md`, and `.guild/wiki/corrections/docs-drift.md` for "common mistakes".

# Output format

Markdown under `docs/` (or godoc in source) with: concept overview, quick start (<20 lines), API reference, full example, common mistakes, related packages.

# Workflow steps

1. Read `.claude/skills/doc-writing/SKILL.md` and `.claude/rules/documentation.md`.
2. Read the code; confirm every documented signature with `go doc <pkg>.<Symbol>`.
3. Write; link to `docs/architecture/*` instead of duplicating it.
4. Compile every example (`go build` on a throwaway file); handle every error.

# Evidence requirements

Each example's build command and result; a list of exported identifiers checked against `go doc`.

# Escalation rules

Docs that reveal a contract gap or an undecided design point → `architect`; never decide it in prose.

# Safety constraints

No marketing language, no emojis, no phase/timeline language; no secrets or real keys in examples.

# Eval cases

See `evals.json` next to this file.
