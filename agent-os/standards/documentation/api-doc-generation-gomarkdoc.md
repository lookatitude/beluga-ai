# API Doc Generation (gomarkdoc)

API documentation is generated from godoc via gomarkdoc into markdown for the Docusaurus website. `scripts/generate-docs.sh` is the source of truth; it discovers packages under `pkg/` recursively instead of a hardcoded list.

- **Tool:** gomarkdoc with `--format github`. Converts godoc into markdown for Docusaurus to render.
- **Output:** `docs/api-docs/packages/` (or the path the site consumes). One `.md` per package; subpackages (e.g. `llms/providers/openai`) in subdirs when applicable.
- **Discovery:** Recursively search `pkg/` and collect existing Go packages. New packages under `pkg/` are included automatically; no manual PACKAGES list to update.
- **Frontmatter:** Each generated file gets `title`, `sidebar_position` (and any other fields the site needs).
- **MDX/Docusaurus:** Convert gomarkdoc's `<details><summary>...</summary>...</details>` to `###` headings or similar so the output is valid for MDX. Avoid raw `<details>` in the final markdown when it breaks rendering.
- **CI:** `.github/workflows/generate-api-docs.yml` runs `scripts/generate-docs.sh` on changes to `pkg/**`, the script, and the workflow. Run `bash scripts/generate-docs.sh` locally to refresh docs.
