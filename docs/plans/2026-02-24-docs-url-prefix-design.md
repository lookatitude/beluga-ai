# Docs URL Prefix — Design

**Date**: 2026-02-24
**Branch**: `fix/website-seo`
**Site**: `https://beluga-ai.org` (Astro + Starlight, GitHub Pages)

## Problem

All documentation pages live at the site root (`/getting-started/`, `/guides/`, `/api-reference/`, etc.), which prevents adding non-documentation pages (blog, pricing, about) in the future. Documentation should live under `/docs/*` while the homepage stays at `/`.

## Approach

Starlight has no built-in URL prefix config. The official workaround is **subdirectory nesting**: move all content directories into `src/content/docs/docs/`, which maps files to `/docs/*` URLs automatically.

---

## Part 1: File Structure

```
src/content/docs/
├── index.mdx           ← STAYS (homepage at /)
├── 404.md              ← STAYS (404 at /404)
└── docs/               ← NEW — all doc content moves here
    ├── getting-started/
    ├── guides/
    ├── tutorials/
    ├── api-reference/
    ├── providers/
    ├── architecture/
    ├── integrations/
    ├── contributing/
    ├── cookbook/
    ├── use-cases/
    └── reports/
```

No redirects from old URLs — clean break.

---

## Part 2: Cascading Updates

### Sidebar config (`src/config/sidebar.json`)
- Every `slug` value: prefix with `docs/`
- Every `autogenerate.directory` value: prefix with `docs/`
- ~50 entries to update

### Homepage links (`src/content/docs/index.mdx`)
- Hero action link: `/getting-started/overview/` → `/docs/getting-started/overview/`
- All `LinkButton` hrefs: add `/docs/` prefix
- StatBadge, FeatureCard, Card content: no links to change (text-only)

### Internal content links (all MD/MDX files)
- Root-relative links like `/guides/something/` → `/docs/guides/something/`
- Relative links (`./sibling`) continue working after the move

### Components
- `Head.astro`: Section filter path detection — add `/docs/` prefix to all `pathname.startsWith()` checks
- `SearchModal.tsx`: QUICK_LINKS URLs — add `/docs/` prefix
- OG image route: Self-correcting (uses `doc.id` which reflects filesystem path)

---

## Files Affected

**Moved (directories):**
- `src/content/docs/getting-started/` → `src/content/docs/docs/getting-started/`
- `src/content/docs/guides/` → `src/content/docs/docs/guides/`
- `src/content/docs/tutorials/` → `src/content/docs/docs/tutorials/`
- `src/content/docs/api-reference/` → `src/content/docs/docs/api-reference/`
- `src/content/docs/providers/` → `src/content/docs/docs/providers/`
- `src/content/docs/architecture/` → `src/content/docs/docs/architecture/`
- `src/content/docs/integrations/` → `src/content/docs/docs/integrations/`
- `src/content/docs/contributing/` → `src/content/docs/docs/contributing/`
- `src/content/docs/cookbook/` → `src/content/docs/docs/cookbook/`
- `src/content/docs/use-cases/` → `src/content/docs/docs/use-cases/`
- `src/content/docs/reports/` → `src/content/docs/docs/reports/`

**Stay in place:**
- `src/content/docs/index.mdx` (homepage)
- `src/content/docs/404.md` (error page)

**Modified:**
- `src/config/sidebar.json` — prefix all slugs/directories
- `src/content/docs/index.mdx` — update all internal links
- `src/components/override-components/Head.astro` — update section filter paths
- `src/components/search/SearchModal.tsx` — update QUICK_LINKS
- All MD/MDX files with root-relative links to other doc pages

---

## Out of Scope

- Redirect infrastructure (clean break, no redirects)
- Homepage conversion to standalone Astro page (stays as Starlight splash)
- Blog or other non-docs pages (future work enabled by this change)
