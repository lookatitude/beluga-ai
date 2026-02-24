import { useState, useCallback, useRef, useEffect } from "react";
import { usePagefind, type SearchResult } from "./usePagefind";
import { useKeyboardNavigation } from "./useKeyboardNavigation";
import { useRecentSearches } from "./useRecentSearches";

const QUICK_LINKS = [
  { label: "Getting Started", url: "/docs/getting-started/overview/" },
  { label: "API Reference", url: "/docs/api-reference/" },
  { label: "Providers", url: "/docs/providers/" },
  { label: "Architecture", url: "/docs/architecture/" },
  { label: "Tutorials", url: "/docs/tutorials/" },
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

  // Build flat list of navigable items
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
