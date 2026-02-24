# Enhanced Search Modal — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace Pagefind's default UI with a custom React search modal featuring keyboard navigation, section filtering, recent searches, and quick links.

**Architecture:** Custom React component (`SearchModal`) mounted inside the existing `<site-search>` web component's `<dialog>`. Uses Pagefind's JS API directly for search and filters. Keeps the existing `SiteSearch` web component for dialog/shortcut management. Deployed on GitHub Pages via `beluga-ai.org`.

**Tech Stack:** React 19, `@astrojs/react`, Pagefind JS API, localStorage, Astro `client:idle` hydration.

---

### Task 1: Install React and @astrojs/react

**Files:**
- Modify: `docs/website/package.json`
- Modify: `docs/website/astro.config.mjs`

**Step 1: Install dependencies**

Run:
```bash
cd docs/website && yarn add @astrojs/react react react-dom
```

**Step 2: Add React integration to Astro config**

In `docs/website/astro.config.mjs`, add the import and integration:

```js
import react from "@astrojs/react";
```

Add `react()` to the `integrations` array (before `sitemap()`):

```js
integrations: [
    starlight({ ... }),
    react(),
    sitemap(),
],
```

**Step 3: Verify build**

Run:
```bash
cd docs/website && yarn build
```
Expected: Build succeeds with React integration active.

**Step 4: Commit**

```bash
git add docs/website/package.json docs/website/yarn.lock docs/website/astro.config.mjs
git commit -m "feat(website): add React integration for custom search modal"
```

---

### Task 2: Add Pagefind section filter meta tag to Head.astro

**Files:**
- Modify: `docs/website/src/components/override-components/Head.astro:1-157` (frontmatter section)

**Step 1: Compute the section from URL path**

In the frontmatter of `Head.astro`, after line 56 (after `customSchemaType`), add:

```typescript
// Section for Pagefind filtering (inferred from URL path)
function getPageSection(): string {
  if (isSplash) return "Home";
  if (pathname.startsWith("/getting-started/")) return "Getting Started";
  if (pathname.startsWith("/guides/")) return "Guides";
  if (pathname.startsWith("/tutorials/")) return "Tutorials";
  if (pathname.startsWith("/api-reference/")) return "API Reference";
  if (pathname.startsWith("/providers/")) return "Providers";
  if (pathname.startsWith("/architecture/")) return "Architecture";
  return "Docs";
}
const pageSection = getPageSection();
```

**Step 2: Add meta tag in template**

After the `<!-- Author -->` meta tag (line 175), add:

```html
<!-- Pagefind section filter -->
<meta data-pagefind-filter="section" content={pageSection} />
```

**Step 3: Verify build**

Run:
```bash
cd docs/website && yarn build
```
Expected: Build succeeds. Check a few built HTML files contain `data-pagefind-filter="section"`.

**Step 4: Commit**

```bash
git add docs/website/src/components/override-components/Head.astro
git commit -m "feat(website): add Pagefind section filter meta tag"
```

---

### Task 3: Create the Pagefind API hook (usePagefind)

**Files:**
- Create: `docs/website/src/components/search/usePagefind.ts`

**Step 1: Create the hook**

```typescript
import { useState, useCallback, useRef, useEffect } from "react";

export interface SearchResult {
  id: string;
  url: string;
  title: string;
  excerpt: string;
  section: string;
  subResults: Array<{
    url: string;
    title: string;
    excerpt: string;
  }>;
}

interface PagefindInstance {
  search: (
    query: string,
    options?: { filters?: Record<string, string> }
  ) => Promise<{
    results: Array<{
      id: string;
      data: () => Promise<{
        url: string;
        meta: { title?: string };
        excerpt: string;
        filters: Record<string, string[]>;
        sub_results: Array<{
          url: string;
          title: string;
          excerpt: string;
        }>;
      }>;
    }>;
  }>;
  filters: () => Promise<Record<string, Record<string, number>>>;
}

export function usePagefind() {
  const [results, setResults] = useState<SearchResult[]>([]);
  const [sections, setSections] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [resultCount, setResultCount] = useState(0);
  const pagefindRef = useRef<PagefindInstance | null>(null);
  const abortRef = useRef(0);

  // Load Pagefind on first use
  const getPagefind = useCallback(async (): Promise<PagefindInstance | null> => {
    if (pagefindRef.current) return pagefindRef.current;
    try {
      const basePath =
        import.meta.env.BASE_URL.replace(/\/$/, "") + "/pagefind/pagefind.js";
      const pf = await import(/* @vite-ignore */ basePath);
      await pf.init?.();
      pagefindRef.current = pf;
      return pf;
    } catch {
      console.error("Failed to load Pagefind");
      return null;
    }
  }, []);

  // Load available sections
  useEffect(() => {
    (async () => {
      const pf = await getPagefind();
      if (!pf) return;
      const filters = await pf.filters();
      if (filters.section) {
        setSections(Object.keys(filters.section).sort());
      }
    })();
  }, [getPagefind]);

  // Search function
  const search = useCallback(
    async (query: string, section?: string) => {
      if (!query.trim()) {
        setResults([]);
        setResultCount(0);
        setLoading(false);
        return;
      }

      setLoading(true);
      const searchId = ++abortRef.current;

      const pf = await getPagefind();
      if (!pf || searchId !== abortRef.current) return;

      const filters = section ? { filters: { section } } : undefined;
      const response = await pf.search(query, filters);

      if (searchId !== abortRef.current) return;

      setResultCount(response.results.length);

      // Load first 10 results (lazy-load more on scroll if needed)
      const loaded = await Promise.all(
        response.results.slice(0, 10).map(async (r) => {
          const data = await r.data();
          return {
            id: r.id,
            url: data.url,
            title: data.meta?.title || data.url,
            excerpt: data.excerpt,
            section: data.filters?.section?.[0] || "Docs",
            subResults: data.sub_results || [],
          };
        })
      );

      if (searchId !== abortRef.current) return;

      setResults(loaded);
      setLoading(false);
    },
    [getPagefind]
  );

  return { results, sections, loading, resultCount, search };
}
```

**Step 2: Verify TypeScript compiles**

Run:
```bash
cd docs/website && npx tsc --noEmit src/components/search/usePagefind.ts 2>&1 || true
```

Note: TypeScript checking may need the full project context. The real validation is the build in a later task. For now, ensure no obvious syntax errors.

**Step 3: Commit**

```bash
git add docs/website/src/components/search/usePagefind.ts
git commit -m "feat(website): add Pagefind API hook for custom search"
```

---

### Task 4: Create the keyboard navigation hook (useKeyboardNavigation)

**Files:**
- Create: `docs/website/src/components/search/useKeyboardNavigation.ts`

**Step 1: Create the hook**

```typescript
import { useState, useCallback, useEffect, useRef } from "react";

interface UseKeyboardNavigationOptions {
  itemCount: number;
  onSelect: (index: number) => void;
  onEscape: () => void;
  enabled?: boolean;
}

export function useKeyboardNavigation({
  itemCount,
  onSelect,
  onEscape,
  enabled = true,
}: UseKeyboardNavigationOptions) {
  const [selectedIndex, setSelectedIndex] = useState(-1);
  const containerRef = useRef<HTMLDivElement>(null);

  // Reset selection when item count changes
  useEffect(() => {
    setSelectedIndex(-1);
  }, [itemCount]);

  const handleKeyDown = useCallback(
    (e: KeyboardEvent) => {
      if (!enabled || itemCount === 0) return;

      switch (e.key) {
        case "ArrowDown":
          e.preventDefault();
          setSelectedIndex((prev) =>
            prev < itemCount - 1 ? prev + 1 : 0
          );
          break;
        case "ArrowUp":
          e.preventDefault();
          setSelectedIndex((prev) =>
            prev > 0 ? prev - 1 : itemCount - 1
          );
          break;
        case "Enter":
          e.preventDefault();
          if (selectedIndex >= 0) {
            onSelect(selectedIndex);
          }
          break;
        case "Escape":
          e.preventDefault();
          onEscape();
          break;
      }
    },
    [enabled, itemCount, selectedIndex, onSelect, onEscape]
  );

  useEffect(() => {
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [handleKeyDown]);

  // Auto-scroll selected item into view
  useEffect(() => {
    if (selectedIndex < 0 || !containerRef.current) return;
    const items = containerRef.current.querySelectorAll('[role="option"]');
    items[selectedIndex]?.scrollIntoView({ block: "nearest" });
  }, [selectedIndex]);

  return { selectedIndex, setSelectedIndex, containerRef };
}
```

**Step 2: Commit**

```bash
git add docs/website/src/components/search/useKeyboardNavigation.ts
git commit -m "feat(website): add keyboard navigation hook for search"
```

---

### Task 5: Create the recent searches hook (useRecentSearches)

**Files:**
- Create: `docs/website/src/components/search/useRecentSearches.ts`

**Step 1: Create the hook**

```typescript
import { useState, useCallback } from "react";

const STORAGE_KEY = "beluga-search-recent";
const MAX_RECENT = 5;

export function useRecentSearches() {
  const [recentSearches, setRecentSearches] = useState<string[]>(() => {
    try {
      const stored = localStorage.getItem(STORAGE_KEY);
      return stored ? JSON.parse(stored) : [];
    } catch {
      return [];
    }
  });

  const addRecent = useCallback((query: string) => {
    const trimmed = query.trim();
    if (!trimmed) return;

    setRecentSearches((prev) => {
      const filtered = prev.filter((q) => q !== trimmed);
      const updated = [trimmed, ...filtered].slice(0, MAX_RECENT);
      try {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(updated));
      } catch {
        // localStorage full or unavailable
      }
      return updated;
    });
  }, []);

  const clearRecent = useCallback(() => {
    setRecentSearches([]);
    try {
      localStorage.removeItem(STORAGE_KEY);
    } catch {
      // ignore
    }
  }, []);

  return { recentSearches, addRecent, clearRecent };
}
```

**Step 2: Commit**

```bash
git add docs/website/src/components/search/useRecentSearches.ts
git commit -m "feat(website): add recent searches localStorage hook"
```

---

### Task 6: Create the SearchModal React component

This is the main component. It integrates all three hooks and renders the full search UI.

**Files:**
- Create: `docs/website/src/components/search/SearchModal.tsx`

**Step 1: Create the component**

```tsx
import { useState, useCallback, useRef, useEffect } from "react";
import { usePagefind, type SearchResult } from "./usePagefind";
import { useKeyboardNavigation } from "./useKeyboardNavigation";
import { useRecentSearches } from "./useRecentSearches";

const QUICK_LINKS = [
  { label: "Getting Started", url: "/getting-started/overview/" },
  { label: "API Reference", url: "/api-reference/" },
  { label: "Providers", url: "/providers/" },
  { label: "Architecture", url: "/architecture/" },
  { label: "Tutorials", url: "/tutorials/" },
];

export default function SearchModal() {
  const [query, setQuery] = useState("");
  const [activeSection, setActiveSection] = useState<string | undefined>(
    undefined
  );
  const inputRef = useRef<HTMLInputElement>(null);
  const debounceRef = useRef<ReturnType<typeof setTimeout>>();

  const { results, sections, loading, resultCount, search } = usePagefind();
  const { recentSearches, addRecent, clearRecent } = useRecentSearches();

  // Build flat list of navigable items (results + sub-results)
  const flatItems: Array<{ url: string; title: string; excerpt: string }> = [];
  for (const result of results) {
    flatItems.push({
      url: result.url,
      title: result.title,
      excerpt: result.excerpt,
    });
  }

  const navigate = useCallback((url: string) => {
    window.location.href = url;
  }, []);

  const handleSelect = useCallback(
    (index: number) => {
      const item = flatItems[index];
      if (item) {
        addRecent(query);
        navigate(item.url);
      }
    },
    [flatItems, query, addRecent, navigate]
  );

  const handleEscape = useCallback(() => {
    if (query) {
      setQuery("");
      search("", activeSection);
    } else {
      // Close dialog via the web component
      const dialog = document.querySelector("site-search dialog") as HTMLDialogElement;
      dialog?.close();
    }
  }, [query, activeSection, search]);

  const { selectedIndex, setSelectedIndex, containerRef } =
    useKeyboardNavigation({
      itemCount: flatItems.length,
      onSelect: handleSelect,
      onEscape: handleEscape,
      enabled: true,
    });

  // Debounced search
  useEffect(() => {
    clearTimeout(debounceRef.current);
    debounceRef.current = setTimeout(() => {
      search(query, activeSection);
    }, 150);
    return () => clearTimeout(debounceRef.current);
  }, [query, activeSection, search]);

  // Focus input when modal opens
  useEffect(() => {
    const observer = new MutationObserver(() => {
      const dialog = document.querySelector(
        "site-search dialog"
      ) as HTMLDialogElement;
      if (dialog?.open) {
        setTimeout(() => inputRef.current?.focus(), 10);
      }
    });
    const dialog = document.querySelector("site-search dialog");
    if (dialog) {
      observer.observe(dialog, { attributes: true });
    }
    return () => observer.disconnect();
  }, []);

  const hasQuery = query.trim().length > 0;

  return (
    <div className="search-modal" ref={containerRef}>
      {/* Search Input */}
      <div className="search-input-wrapper">
        <svg
          className="search-icon"
          width="20"
          height="20"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          strokeWidth="2"
        >
          <circle cx="11" cy="11" r="8" />
          <path d="m21 21-4.3-4.3" />
        </svg>
        <input
          ref={inputRef}
          type="text"
          className="search-input"
          placeholder="Search documentation..."
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          aria-label="Search documentation"
          aria-controls="search-results"
          aria-activedescendant={
            selectedIndex >= 0 ? `search-result-${selectedIndex}` : undefined
          }
        />
        {hasQuery && (
          <button
            className="search-clear"
            onClick={() => {
              setQuery("");
              inputRef.current?.focus();
            }}
            aria-label="Clear search"
          >
            <svg
              width="16"
              height="16"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              strokeWidth="2"
            >
              <path d="M18 6 6 18M6 6l12 12" />
            </svg>
          </button>
        )}
      </div>

      {/* Section Filter Pills */}
      {sections.length > 0 && hasQuery && (
        <div className="filter-bar" role="tablist" aria-label="Filter by section">
          <button
            role="tab"
            className={`filter-pill ${activeSection === undefined ? "active" : ""}`}
            onClick={() => setActiveSection(undefined)}
            aria-selected={activeSection === undefined}
          >
            All
          </button>
          {sections.map((section) => (
            <button
              key={section}
              role="tab"
              className={`filter-pill ${activeSection === section ? "active" : ""}`}
              onClick={() => setActiveSection(section)}
              aria-selected={activeSection === section}
            >
              {section}
            </button>
          ))}
        </div>
      )}

      {/* Results Area */}
      <div className="results-area" id="search-results" role="listbox">
        {/* Empty state: recent searches + quick links */}
        {!hasQuery && (
          <div className="empty-state">
            {recentSearches.length > 0 && (
              <div className="recent-section">
                <div className="section-header">
                  <span className="section-title">Recent Searches</span>
                  <button className="clear-recent" onClick={clearRecent}>
                    Clear
                  </button>
                </div>
                {recentSearches.map((recent) => (
                  <button
                    key={recent}
                    className="recent-item"
                    onClick={() => {
                      setQuery(recent);
                    }}
                  >
                    <svg
                      width="16"
                      height="16"
                      viewBox="0 0 24 24"
                      fill="none"
                      stroke="currentColor"
                      strokeWidth="2"
                    >
                      <circle cx="12" cy="12" r="10" />
                      <polyline points="12 6 12 12 16 14" />
                    </svg>
                    <span>{recent}</span>
                  </button>
                ))}
              </div>
            )}
            <div className="quick-links-section">
              <span className="section-title">Quick Links</span>
              {QUICK_LINKS.map((link) => (
                <a key={link.url} href={link.url} className="quick-link">
                  <svg
                    width="16"
                    height="16"
                    viewBox="0 0 24 24"
                    fill="none"
                    stroke="currentColor"
                    strokeWidth="2"
                  >
                    <path d="M9 10h1a1 1 0 1 0 0-2H9a1 1 0 0 0 0 2Zm0 2a1 1 0 0 0 0 2h6a1 1 0 0 0 0-2H9Z" />
                    <path d="M20 9V8l-6-6a1 1 0 0 0-1 0H7a3 3 0 0 0-3 3v14a3 3 0 0 0 3 3h10a3 3 0 0 0 3-3V9Z" />
                  </svg>
                  <span>{link.label}</span>
                </a>
              ))}
            </div>
          </div>
        )}

        {/* Loading */}
        {hasQuery && loading && (
          <div className="search-loading">Searching...</div>
        )}

        {/* Results */}
        {hasQuery && !loading && results.length > 0 && (
          <>
            <div className="result-count">{resultCount} results</div>
            {results.map((result, i) => (
              <a
                key={result.id}
                id={`search-result-${i}`}
                role="option"
                href={result.url}
                className={`search-result ${selectedIndex === i ? "selected" : ""}`}
                aria-selected={selectedIndex === i}
                onClick={() => addRecent(query)}
                onMouseEnter={() => setSelectedIndex(i)}
              >
                <div className="result-title">{result.title}</div>
                <div
                  className="result-excerpt"
                  dangerouslySetInnerHTML={{ __html: result.excerpt }}
                />
                <div className="result-section">{result.section}</div>
              </a>
            ))}
          </>
        )}

        {/* No results */}
        {hasQuery && !loading && results.length === 0 && (
          <div className="no-results">
            <p>No results for &ldquo;{query}&rdquo;</p>
            <p className="no-results-hint">
              Try a different search term or browse:
            </p>
            <div className="quick-links-section">
              {QUICK_LINKS.map((link) => (
                <a key={link.url} href={link.url} className="quick-link">
                  <span>{link.label}</span>
                </a>
              ))}
            </div>
          </div>
        )}
      </div>

      {/* Footer */}
      <div className="search-footer">
        <span className="kbd-hint">
          <kbd>↑↓</kbd> Navigate
        </span>
        <span className="kbd-hint">
          <kbd>↵</kbd> Open
        </span>
        <span className="kbd-hint">
          <kbd>Esc</kbd> Close
        </span>
      </div>
    </div>
  );
}
```

**Step 2: Commit**

```bash
git add docs/website/src/components/search/SearchModal.tsx
git commit -m "feat(website): create SearchModal React component"
```

---

### Task 7: Create Search.astro override (replace Pagefind UI with React modal)

Replace the current Pagefind default UI mount with the React `SearchModal` component. Keep the `<site-search>` web component as the outer shell for dialog management and Cmd+K shortcut.

**Files:**
- Create: `docs/website/src/components/override-components/Search.astro` — **This file already exists**; we will **replace** it entirely
- Modify: `docs/website/astro.config.mjs` — Add `Search` to component overrides

**Step 1: Rewrite Search.astro**

Replace the entire content of `docs/website/src/components/override-components/Search.astro` with:

```astro
---
import SearchModal from "@/components/search/SearchModal";
---

<site-search>
  <button
    data-open-modal
    disabled
    aria-label="Search"
    aria-keyshortcuts="Control+K"
  >
    <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
      <circle cx="11" cy="11" r="8" />
      <path d="m21 21-4.3-4.3" />
    </svg>
    <span class="sl-block" aria-hidden="true">Search</span>
    <kbd class="sl-flex">
      <kbd>Ctrl</kbd><kbd>K</kbd>
    </kbd>
  </button>

  <dialog style="padding:0" aria-label="Search documentation">
    <div class="dialog-frame">
      <SearchModal client:idle />
    </div>
  </dialog>
</site-search>

<script is:inline>
  (() => {
    const openBtn = document.querySelector("button[data-open-modal]");
    const shortcut = openBtn?.querySelector("kbd");
    if (!openBtn || !(shortcut instanceof HTMLElement)) return;
    const platformKey = shortcut.querySelector("kbd");
    if (platformKey && /(Mac|iPhone|iPod|iPad)/i.test(navigator.platform)) {
      platformKey.textContent = "⌘";
      openBtn.setAttribute("aria-keyshortcuts", "Meta+K");
    }
    shortcut.style.display = "";
  })();
</script>

<script>
  class SiteSearch extends HTMLElement {
    constructor() {
      super();
      const openBtn = this.querySelector<HTMLButtonElement>("button[data-open-modal]")!;
      const dialog = this.querySelector("dialog")!;
      const dialogFrame = this.querySelector(".dialog-frame")!;

      const onClick = (event: MouseEvent) => {
        const isLink = "href" in (event.target || {});
        if (
          isLink ||
          (document.body.contains(event.target as Node) &&
            !dialogFrame.contains(event.target as Node))
        ) {
          closeModal();
        }
      };

      const openModal = (event?: MouseEvent) => {
        dialog.showModal();
        document.body.toggleAttribute("data-search-modal-open", true);
        // Focus is handled by the React component
        event?.stopPropagation();
        window.addEventListener("click", onClick);
      };

      const closeModal = () => dialog.close();

      openBtn.addEventListener("click", openModal);
      openBtn.disabled = false;

      dialog.addEventListener("close", () => {
        document.body.toggleAttribute("data-search-modal-open", false);
        window.removeEventListener("click", onClick);
      });

      window.addEventListener("keydown", (e) => {
        if ((e.metaKey === true || e.ctrlKey === true) && e.key === "k") {
          dialog.open ? closeModal() : openModal();
          e.preventDefault();
        }
      });
    }
  }
  customElements.define("site-search", SiteSearch);
</script>

<style>
  @layer starlight.core {
    site-search {
      display: contents;
    }
    button[data-open-modal] {
      display: flex;
      align-items: center;
      gap: 0.5rem;
      background-color: color-mix(in srgb, var(--sl-color-white) 7%, transparent);
      border: 1px solid color-mix(in srgb, var(--sl-color-white) 10%, transparent);
      border-radius: 0.5rem;
      padding-inline-start: 1rem;
      padding-inline-end: 0.8rem;
      cursor: pointer;
      height: 3rem;
      font-size: var(--sl-text-sm);
      width: 100%;
      color: var(--sl-color-gray-2);
    }
    button[data-open-modal] kbd {
      margin-inline-start: auto;
    }
    @media (min-width: 50rem) {
      button[data-open-modal] {
        max-width: 100%;
      }
      button[data-open-modal]:hover {
        border-color: var(--sl-color-gray-2);
        color: var(--sl-color-white);
      }
      button[data-open-modal] > :last-child {
        margin-inline-start: auto;
      }
    }
    button > kbd {
      border-radius: 0.25rem;
      font-size: var(--sl-text-sm);
      padding-inline: 0.375rem;
      background-color: var(--sl-color-gray-6);
    }
    kbd {
      font-family: var(--__sl-font);
    }
    dialog {
      margin: 0;
      background-color: var(--sl-color-gray-6);
      border: 1px solid var(--sl-color-gray-5);
      width: 100%;
      max-width: 100%;
      height: 100%;
      max-height: 100%;
      box-shadow: var(--sl-shadow-lg);
    }
    dialog[open] {
      display: flex;
    }
    dialog::backdrop {
      background-color: var(--sl-color-backdrop-overlay);
      backdrop-filter: blur(0.25rem);
    }
    .dialog-frame {
      position: relative;
      overflow: auto;
      display: flex;
      flex-direction: column;
      flex-grow: 1;
      padding: 1rem;
    }
    @media (min-width: 50rem) {
      dialog {
        margin: 4rem auto auto;
        border-radius: 0.5rem;
        width: 90%;
        max-width: 40rem;
        height: max-content;
        min-height: 15rem;
        max-height: calc(100% - 8rem);
      }
      .dialog-frame {
        padding: 1.5rem;
      }
    }
  }
</style>

<style is:global>
  @layer starlight.core {
    [data-search-modal-open] {
      overflow: hidden;
    }

    /* Search modal component styles */
    .search-modal {
      display: flex;
      flex-direction: column;
      height: 100%;
      gap: 0.75rem;
    }

    .search-input-wrapper {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.75rem 1rem;
      background: var(--sl-color-black);
      border: 1px solid var(--sl-color-gray-5);
      border-radius: 0.5rem;
    }
    .search-input-wrapper:focus-within {
      border-color: var(--sl-color-accent);
    }

    .search-icon {
      color: var(--sl-color-gray-3);
      flex-shrink: 0;
    }

    .search-input {
      flex: 1;
      background: transparent;
      border: none;
      outline: none;
      color: var(--sl-color-white);
      font-size: 1rem;
      font-family: var(--__sl-font);
    }
    .search-input::placeholder {
      color: var(--sl-color-gray-3);
    }

    .search-clear {
      background: transparent;
      border: none;
      cursor: pointer;
      color: var(--sl-color-gray-3);
      padding: 0.25rem;
      border-radius: 0.25rem;
      display: flex;
      align-items: center;
    }
    .search-clear:hover {
      color: var(--sl-color-white);
    }

    /* Filter pills */
    .filter-bar {
      display: flex;
      gap: 0.375rem;
      overflow-x: auto;
      padding-bottom: 0.25rem;
      scrollbar-width: thin;
    }

    .filter-pill {
      white-space: nowrap;
      padding: 0.25rem 0.75rem;
      border-radius: 9999px;
      font-size: 0.8125rem;
      border: 1px solid var(--sl-color-gray-5);
      background: transparent;
      color: var(--sl-color-gray-2);
      cursor: pointer;
      font-family: var(--__sl-font);
      transition: all 0.15s;
    }
    .filter-pill:hover {
      border-color: var(--sl-color-gray-3);
      color: var(--sl-color-white);
    }
    .filter-pill.active {
      background: var(--sl-color-accent-low);
      border-color: var(--sl-color-accent);
      color: var(--sl-color-white);
    }

    /* Results area */
    .results-area {
      flex: 1;
      overflow-y: auto;
      min-height: 0;
    }

    .result-count {
      font-size: 0.8125rem;
      color: var(--sl-color-gray-3);
      margin-bottom: 0.5rem;
    }

    .search-result {
      display: block;
      padding: 0.75rem 1rem;
      border-radius: 0.375rem;
      text-decoration: none;
      color: var(--sl-color-white);
      transition: background 0.1s;
    }
    .search-result:hover,
    .search-result.selected {
      background: var(--sl-color-accent-low);
    }

    .result-title {
      font-weight: 600;
      font-size: 0.9375rem;
      margin-bottom: 0.25rem;
    }

    .result-excerpt {
      font-size: 0.8125rem;
      color: var(--sl-color-gray-2);
      line-height: 1.4;
      overflow: hidden;
      display: -webkit-box;
      -webkit-line-clamp: 2;
      -webkit-box-orient: vertical;
    }
    .result-excerpt mark {
      color: var(--sl-color-white);
      background: transparent;
      font-weight: 600;
    }

    .result-section {
      font-size: 0.75rem;
      color: var(--sl-color-gray-4);
      margin-top: 0.25rem;
    }

    /* Empty state */
    .empty-state {
      display: flex;
      flex-direction: column;
      gap: 1.5rem;
      padding-top: 0.5rem;
    }

    .section-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 0.5rem;
    }

    .section-title {
      font-size: 0.75rem;
      font-weight: 600;
      color: var(--sl-color-gray-3);
      text-transform: uppercase;
      letter-spacing: 0.05em;
    }

    .clear-recent {
      background: transparent;
      border: none;
      color: var(--sl-color-gray-4);
      font-size: 0.75rem;
      cursor: pointer;
      font-family: var(--__sl-font);
    }
    .clear-recent:hover {
      color: var(--sl-color-white);
    }

    .recent-item {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.5rem 0.75rem;
      width: 100%;
      background: transparent;
      border: none;
      color: var(--sl-color-gray-2);
      cursor: pointer;
      border-radius: 0.375rem;
      font-size: 0.875rem;
      font-family: var(--__sl-font);
      text-align: left;
    }
    .recent-item:hover {
      background: color-mix(in srgb, var(--sl-color-white) 5%, transparent);
      color: var(--sl-color-white);
    }
    .recent-item svg {
      color: var(--sl-color-gray-4);
      flex-shrink: 0;
    }

    .quick-links-section {
      display: flex;
      flex-direction: column;
      gap: 0.25rem;
    }
    .quick-links-section .section-title {
      margin-bottom: 0.5rem;
    }

    .quick-link {
      display: flex;
      align-items: center;
      gap: 0.75rem;
      padding: 0.5rem 0.75rem;
      color: var(--sl-color-gray-2);
      text-decoration: none;
      border-radius: 0.375rem;
      font-size: 0.875rem;
    }
    .quick-link:hover {
      background: color-mix(in srgb, var(--sl-color-white) 5%, transparent);
      color: var(--sl-color-white);
    }
    .quick-link svg {
      color: var(--sl-color-gray-4);
      flex-shrink: 0;
    }

    /* Loading */
    .search-loading {
      text-align: center;
      color: var(--sl-color-gray-3);
      padding: 2rem;
    }

    /* No results */
    .no-results {
      text-align: center;
      padding: 2rem 1rem;
      color: var(--sl-color-gray-2);
    }
    .no-results p:first-child {
      font-size: 1rem;
      font-weight: 600;
      margin-bottom: 0.5rem;
    }
    .no-results-hint {
      font-size: 0.875rem;
      color: var(--sl-color-gray-3);
      margin-bottom: 1rem;
    }
    .no-results .quick-links-section {
      align-items: center;
    }

    /* Footer */
    .search-footer {
      display: flex;
      gap: 1rem;
      justify-content: center;
      padding-top: 0.75rem;
      border-top: 1px solid var(--sl-color-gray-5);
    }

    .kbd-hint {
      font-size: 0.75rem;
      color: var(--sl-color-gray-4);
      display: flex;
      align-items: center;
      gap: 0.375rem;
    }
    .kbd-hint kbd {
      background: var(--sl-color-gray-5);
      border-radius: 0.25rem;
      padding: 0.125rem 0.375rem;
      font-size: 0.6875rem;
      font-family: var(--__sl-font);
    }
  }
</style>
```

**Step 2: Add Search override to astro.config.mjs**

In `docs/website/astro.config.mjs`, add `Search` to the `components` object inside the `starlight()` config:

```js
Search: "./src/components/override-components/Search.astro",
```

Add it after the `Head` entry.

**Step 3: Verify build**

Run:
```bash
cd docs/website && yarn build
```
Expected: Build succeeds. The search modal should render with the React component.

**Step 4: Commit**

```bash
git add docs/website/src/components/override-components/Search.astro docs/website/astro.config.mjs
git commit -m "feat(website): replace Pagefind UI with custom React search modal"
```

---

### Task 8: Verify end-to-end functionality

**Files:** None (verification only)

**Step 1: Build the site**

Run:
```bash
cd docs/website && yarn build
```
Expected: Clean build, no errors.

**Step 2: Preview the site**

Run:
```bash
cd docs/website && yarn preview
```

**Step 3: Manual verification checklist**

Open the preview URL and verify:

1. **Cmd/Ctrl+K** opens the search modal
2. **Search input** is auto-focused when modal opens
3. **Empty state** shows recent searches (after first search) and quick links
4. **Typing a query** shows search results after 150ms debounce
5. **Section filter pills** appear above results (All, Getting Started, Guides, etc.)
6. **Clicking a filter pill** scopes results to that section
7. **Arrow keys** navigate through results (highlighted row moves)
8. **Enter** opens the selected result
9. **Escape** clears query (if present) or closes modal (if query empty)
10. **Recent searches** are saved to localStorage and appear on re-open
11. **Quick links** navigate to correct pages
12. **Footer** shows keyboard hints (↑↓ Navigate, ↵ Open, Esc Close)
13. **No results** state shows helpful message with quick links fallback

**Step 4: Final commit (if any adjustments needed)**

If any tweaks were needed during verification, commit them:
```bash
git add -A
git commit -m "fix(website): search modal polish after manual testing"
```

---

## Summary

| Task | Description | Dependencies |
|------|-------------|-------------|
| 1 | Install React + @astrojs/react | None |
| 2 | Add Pagefind section filter meta tag | None |
| 3 | Create usePagefind hook | Task 1 |
| 4 | Create useKeyboardNavigation hook | Task 1 |
| 5 | Create useRecentSearches hook | Task 1 |
| 6 | Create SearchModal component | Tasks 3, 4, 5 |
| 7 | Rewrite Search.astro + config override | Tasks 1, 6 |
| 8 | End-to-end verification | Task 7 |

Tasks 1 and 2 can run in parallel. Tasks 3, 4, and 5 can run in parallel after Task 1. Task 6 depends on 3+4+5. Task 7 depends on 1+2+6. Task 8 depends on 7.
