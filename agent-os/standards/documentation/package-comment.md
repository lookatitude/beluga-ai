# Package Comment

Every package MUST have a single `// Package <name> provides ...` block. Prefer full blocks and complete paragraphs so it works for go doc, gomarkdoc, onboarding, and search.

- **First sentence:** `// Package X provides <concise description>.` — required; one line.
- **Optional "The package provides:"** — bullet list of main capabilities when the package does several things.
- **Optional "Example usage:"** — indented code (one tab) showing typical use. Keep short; point to `examples/` for full examples when helpful.
- **One comment per package** — no duplicate `// Package` in other files in the package.
- **Complete paragraphs** — prefer full blocks over one-liners so gomarkdoc and go doc produce useful output.
