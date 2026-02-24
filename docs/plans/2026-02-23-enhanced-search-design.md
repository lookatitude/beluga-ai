# Enhanced Search Modal — Design

**Date**: 2026-02-23
**Branch**: `fix/website-seo`
**Site**: `https://beluga-ai.org` (Astro + Starlight, GitHub Pages)

## Problem

The current search modal uses Pagefind's default UI which lacks keyboard navigation within results, section filtering, recent searches, and quick links. Developers can't efficiently navigate the 447-page documentation site.

## Approach

Replace Pagefind's default UI with a custom React search modal that uses Pagefind's JS API directly. Keep the existing `<site-search>` web component as the outer shell (handles Cmd+K and dialog management). The React component owns rendering, keyboard nav, filtering, and state.

---

## Architecture

```
SearchModal (React, client:idle)
├── SearchInput (auto-focused, debounced 150ms)
├── FilterBar (section pills: All, Guides, API, Tutorials, Providers...)
├── ResultsArea (conditional rendering)
│   ├── EmptyState → QuickLinks + RecentSearches
│   ├── SearchResults → grouped by section, keyboard-navigable
│   └── NoResults → helpful message + suggestions
└── Footer (keyboard hints: ↑↓ navigate, ↵ select, esc close)
```

**Integration:** The Astro `Search.astro` component renders `<site-search>` with the `<dialog>` and mounts `<SearchModal client:idle />` inside it instead of Pagefind's default UI.

**Pagefind API usage:** Import `pagefind` dynamically at runtime. Call `pagefind.search(query, { filters })` for search and `pagefind.filters()` to discover available sections.

---

## Keyboard Navigation

| Key | Action |
|-----|--------|
| `↑` / `↓` | Move selection through results (wraps around) |
| `Enter` | Open selected result (navigates and closes modal) |
| `Escape` | Close modal (or clear search if input has text) |
| `Tab` | Move between filter pills |
| `Ctrl/Cmd + K` | Toggle modal open/close (existing) |

**Implementation:** `useKeyboardNavigation` hook tracks selected index. Results use `role="listbox"` / `role="option"` with `aria-activedescendant`. Selected item auto-scrolls into view.

---

## Section Filtering

**Pagefind filter setup:** Add `<meta data-pagefind-filter="section">` to Head.astro, inferring section from URL path.

**Sections:**
- All (default, no filter)
- Getting Started (`/getting-started/`)
- Guides (`/guides/`)
- Tutorials (`/tutorials/`)
- API Reference (`/api-reference/`)
- Providers (`/providers/`)
- Architecture (`/architecture/`)

**UI:** Horizontal pill buttons above results. Active pill highlighted. Pagefind's `search(query, { filters: { section: "Guides" } })` scopes results.

---

## Recent Searches & Quick Links

**Empty state (no search input):**

1. **Recent Searches** (up to 5, stored in `localStorage` as `beluga-search-recent`):
   - Query text with clock icon
   - Click to re-run search
   - "Clear recents" link

2. **Quick Links** (static high-traffic destinations):
   - Getting Started → `/getting-started/overview/`
   - API Reference → `/api-reference/`
   - Providers → `/providers/`
   - Architecture → `/architecture/`
   - Tutorials → `/tutorials/`

**Behavior:** Sections disappear when user starts typing. Return when input is cleared.

---

## Visual Design

**Modal:**
- Dark background (`#0d0d0d` / `#151515`), subtle border, backdrop blur
- Max width ~640px centered, max height 80vh
- Fade + slide-up entrance animation

**Results:**
- Grouped by section with section headers
- Each result: title (bold), excerpt with highlighted match (muted), breadcrumb path (smallest)
- Selected result has subtle highlight background
- Result count near top ("12 results")

**No results:** "No results for [query]" with suggestion text and quick links fallback.

**Footer:** Fixed at bottom, keyboard hints: `↑↓ Navigate` `↵ Open` `Esc Close`

---

## Files Affected

**Modified:**
- `docs/website/src/components/override-components/Search.astro` — Replace PagefindUI mount with React component
- `docs/website/src/components/override-components/Head.astro` — Add pagefind section filter meta tag
- `docs/website/astro.config.mjs` — Add React integration if not present
- `docs/website/package.json` — Add `@astrojs/react`, `react`, `react-dom`

**New:**
- `docs/website/src/components/search/SearchModal.tsx` — Main React component
- `docs/website/src/components/search/useKeyboardNavigation.ts` — Keyboard nav hook
- `docs/website/src/components/search/usePagefind.ts` — Pagefind API hook (search + filters)
- `docs/website/src/components/search/useRecentSearches.ts` — localStorage hook
- `docs/website/src/components/search/SearchResults.tsx` — Results rendering
- `docs/website/src/components/search/FilterBar.tsx` — Section filter pills
- `docs/website/src/components/search/QuickLinks.tsx` — Quick links component
- `docs/website/src/components/search/SearchFooter.tsx` — Keyboard hints footer

---

## Out of Scope

- AI-powered search / natural language queries
- Search analytics
- Algolia or other external search providers
- Full-text content preview in results (beyond Pagefind excerpts)
- Search within code blocks specifically
