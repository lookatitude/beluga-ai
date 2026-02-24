import { useState, useMemo } from "react";
import { integrations, categories } from "@/data/integrations";
import type { Integration } from "@/data/integrations";

const TOTAL = integrations.length;

const priorityConfig: Record<
  Integration["priority"],
  { bg: string; text: string; darkBg: string; darkText: string }
> = {
  Core: {
    bg: "rgba(34, 197, 94, 0.12)",
    text: "#22c55e",
    darkBg: "rgba(34, 197, 94, 0.15)",
    darkText: "#4ade80",
  },
  Extended: {
    bg: "rgba(92, 163, 202, 0.12)",
    text: "#5CA3CA",
    darkBg: "rgba(92, 163, 202, 0.15)",
    darkText: "#7bbde0",
  },
  Community: {
    bg: "rgba(156, 163, 175, 0.12)",
    text: "#6b7280",
    darkBg: "rgba(156, 163, 175, 0.15)",
    darkText: "#9ca3af",
  },
};

export default function IntegrationsCatalog() {
  const [search, setSearch] = useState("");
  const [activeCategory, setActiveCategory] = useState<string>("All");

  const filtered = useMemo(() => {
    const query = search.toLowerCase().trim();
    return integrations.filter((i) => {
      const matchesCategory =
        activeCategory === "All" || i.category === activeCategory;
      const matchesSearch = !query || i.name.toLowerCase().includes(query);
      return matchesCategory && matchesSearch;
    });
  }, [search, activeCategory]);

  return (
    <div>
      {/* Search */}
      <div style={{ maxWidth: 480, margin: "0 auto 2rem" }}>
        <div style={{ position: "relative" }}>
          <svg
            style={{
              position: "absolute",
              left: 16,
              top: "50%",
              transform: "translateY(-50%)",
              width: 18,
              height: 18,
              opacity: 0.4,
            }}
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            strokeLinejoin="round"
          >
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            type="text"
            placeholder="Search integrations..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            style={{
              width: "100%",
              padding: "0.75rem 1rem 0.75rem 2.75rem",
              fontSize: "0.9375rem",
              borderRadius: "0.75rem",
              border: "1px solid var(--search-border)",
              background: "var(--search-bg)",
              color: "var(--search-text)",
              outline: "none",
              fontFamily: "inherit",
              boxSizing: "border-box",
              transition: "border-color 0.2s ease",
            }}
            onFocus={(e) => {
              e.currentTarget.style.borderColor = "#5CA3CA";
            }}
            onBlur={(e) => {
              e.currentTarget.style.borderColor = "var(--search-border)";
            }}
          />
        </div>
      </div>

      {/* Category tabs */}
      <div
        style={{
          display: "flex",
          gap: "0.5rem",
          overflowX: "auto",
          paddingBottom: "0.5rem",
          marginBottom: "1.5rem",
          scrollbarWidth: "none",
          msOverflowStyle: "none",
          WebkitOverflowScrolling: "touch",
        }}
      >
        {["All", ...categories].map((cat) => {
          const isActive = activeCategory === cat;
          const count =
            cat === "All"
              ? TOTAL
              : integrations.filter((i) => i.category === cat).length;
          return (
            <button
              key={cat}
              onClick={() => setActiveCategory(cat)}
              style={{
                flexShrink: 0,
                padding: "0.5rem 1rem",
                fontSize: "0.8125rem",
                fontWeight: 500,
                borderRadius: "9999px",
                border: isActive
                  ? "1px solid #5CA3CA"
                  : "1px solid var(--tab-border)",
                background: isActive
                  ? "rgba(92, 163, 202, 0.15)"
                  : "transparent",
                color: isActive ? "#5CA3CA" : "var(--tab-text)",
                cursor: "pointer",
                transition:
                  "background 0.2s ease, color 0.2s ease, border-color 0.2s ease",
                fontFamily: "inherit",
                whiteSpace: "nowrap",
              }}
            >
              {cat}
              <span
                style={{
                  marginLeft: "0.375rem",
                  opacity: 0.6,
                  fontSize: "0.75rem",
                }}
              >
                {count}
              </span>
            </button>
          );
        })}
      </div>

      {/* Count display */}
      <p
        style={{
          fontSize: "0.875rem",
          color: "var(--count-text)",
          marginBottom: "1.5rem",
        }}
      >
        Showing {filtered.length} of {TOTAL} integrations
      </p>

      {/* Grid */}
      <div
        style={{
          display: "grid",
          gridTemplateColumns: "repeat(auto-fill, minmax(280px, 1fr))",
          gap: "1rem",
        }}
      >
        {filtered.map((integration) => (
          <IntegrationCard key={integration.name} integration={integration} />
        ))}
      </div>

      {filtered.length === 0 && (
        <div
          style={{
            textAlign: "center",
            padding: "4rem 1rem",
            color: "var(--count-text)",
          }}
        >
          <p style={{ fontSize: "1.125rem", marginBottom: "0.5rem" }}>
            No integrations found
          </p>
          <p style={{ fontSize: "0.875rem", opacity: 0.7 }}>
            Try a different search term or category
          </p>
        </div>
      )}

      {/* CSS variables for theming */}
      <style>{`
        :root {
          --search-bg: rgba(255, 255, 255, 0.03);
          --search-border: rgba(246, 246, 246, 0.1);
          --search-text: #f6f6f6;
          --tab-border: rgba(246, 246, 246, 0.1);
          --tab-text: rgba(246, 246, 246, 0.6);
          --count-text: rgba(246, 246, 246, 0.5);
          --card-bg: #0a0a0a;
          --card-border: rgba(246, 246, 246, 0.08);
          --card-border-hover: rgba(92, 163, 202, 0.3);
          --card-title: #f6f6f6;
          --card-desc: rgba(246, 246, 246, 0.55);
          --card-category: rgba(246, 246, 246, 0.4);
        }
        :root[data-theme="light"] {
          --search-bg: rgba(0, 0, 0, 0.03);
          --search-border: rgba(24, 24, 24, 0.12);
          --search-text: #181818;
          --tab-border: rgba(24, 24, 24, 0.12);
          --tab-text: rgba(24, 24, 24, 0.55);
          --count-text: rgba(24, 24, 24, 0.5);
          --card-bg: #ffffff;
          --card-border: rgba(24, 24, 24, 0.08);
          --card-border-hover: rgba(92, 163, 202, 0.4);
          --card-title: #181818;
          --card-desc: rgba(24, 24, 24, 0.6);
          --card-category: rgba(24, 24, 24, 0.45);
        }
      `}</style>
    </div>
  );
}

function IntegrationCard({ integration }: { integration: Integration }) {
  const [hovered, setHovered] = useState(false);
  const isDark =
    typeof document !== "undefined"
      ? document.documentElement.getAttribute("data-theme") !== "light"
      : true;
  const priority = priorityConfig[integration.priority];
  const badgeBg = isDark ? priority.darkBg : priority.bg;
  const badgeText = isDark ? priority.darkText : priority.text;

  const card = (
    <div
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        padding: "1.25rem",
        borderRadius: "0.875rem",
        background: "var(--card-bg)",
        border: `1px solid ${hovered ? "var(--card-border-hover)" : "var(--card-border)"}`,
        transition:
          "border-color 0.25s ease, transform 0.25s ease, box-shadow 0.25s ease",
        transform: hovered ? "translateY(-2px)" : "translateY(0)",
        boxShadow: hovered
          ? "0 4px 20px rgba(92, 163, 202, 0.08)"
          : "none",
        cursor: integration.docLink ? "pointer" : "default",
        display: "flex",
        flexDirection: "column" as const,
        gap: "0.625rem",
      }}
    >
      {/* Header row */}
      <div
        style={{
          display: "flex",
          justifyContent: "space-between",
          alignItems: "flex-start",
          gap: "0.5rem",
        }}
      >
        <h3
          style={{
            fontSize: "0.9375rem",
            fontWeight: 600,
            color: "var(--card-title)",
            margin: 0,
            lineHeight: 1.3,
          }}
        >
          {integration.name}
        </h3>
        <span
          style={{
            flexShrink: 0,
            fontSize: "0.6875rem",
            fontWeight: 600,
            padding: "0.15rem 0.5rem",
            borderRadius: "9999px",
            background: badgeBg,
            color: badgeText,
            lineHeight: 1.5,
            textTransform: "uppercase",
            letterSpacing: "0.03em",
          }}
        >
          {integration.priority}
        </span>
      </div>

      {/* Description */}
      <p
        style={{
          fontSize: "0.8125rem",
          color: "var(--card-desc)",
          margin: 0,
          lineHeight: 1.55,
          flex: 1,
        }}
      >
        {integration.description}
      </p>

      {/* Category tag */}
      <span
        style={{
          fontSize: "0.6875rem",
          color: "var(--card-category)",
          fontWeight: 500,
        }}
      >
        {integration.category}
      </span>
    </div>
  );

  if (integration.docLink) {
    return (
      <a
        href={integration.docLink}
        style={{ textDecoration: "none", color: "inherit" }}
      >
        {card}
      </a>
    );
  }

  return card;
}
