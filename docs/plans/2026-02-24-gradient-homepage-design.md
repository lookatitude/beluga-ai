# Gradient Colors & Homepage Layout — Design

**Date**: 2026-02-24
**Branch**: `fix/website-seo`
**Site**: `https://beluga-ai.org` (Astro + Starlight, GitHub Pages)

## Problem

The site's primary gradient (purple → rose → gold: #3a1c71 → #D76D77 → #ffca7b) is visually disconnected from the logo which uses teal/cyan/navy tones (#C4E8F1, #5CA3CA, #0F3353). The logo glow and drop shadow also use rose tones. Additionally, the homepage has text centering and alignment issues across the hero, section titles, stats row, and card grids.

## Approach

Two-part fix: (1) shift the entire color system to match the logo's teal palette, (2) fix homepage layout alignment issues.

---

## Part 1: Color System — Match Site to Logo

### New Color Values

| Element | Old Value | New Value |
|---------|-----------|-----------|
| Primary color (`theme.json`) | `#D76D77` (rose) | `#5CA3CA` (teal) |
| Gradient start | `#3a1c71` (purple) | `#0F3353` (navy) |
| Gradient mid | `#D76D77` (rose) | `#5CA3CA` (teal) |
| Gradient end | `#ffca7b` (gold) | `#81D4E8` (cyan) |
| Logo glow primary | `rgba(215, 109, 119, 0.35)` | `rgba(92, 163, 202, 0.35)` |
| Logo glow secondary | `rgba(58, 28, 113, 0.15)` | `rgba(15, 51, 83, 0.15)` |
| Logo drop shadow | `rgba(215, 109, 119, 0.3)` | `rgba(92, 163, 202, 0.3)` |
| OG image accent | `#D76D77` | `#5CA3CA` |
| OG image gradient bg | `#1a1a2e` | `#0F2535` |

### New Primary Gradient

```css
--color-primary-gradient: linear-gradient(
  35.65deg,
  #0F3353 -10.94%,
  #5CA3CA 61.04%,
  #81D4E8 133.01%
);
```

### What Uses the Gradient (all auto-update via CSS variable)

- `.gradient-text` — stat values, section title accents
- `.btn-primary` — primary button backgrounds
- `.btn-outline-primary` — outlined button borders
- `.feature-card::before` — feature card hover borders
- `.card::before` — architecture card hover borders
- `blockquote` borders
- Navigation active state borders
- `.section-title.gradient-text` — section heading accents

### Feature Card Accent Colors (unchanged)

These individual colors complement teal and should stay:
- Streaming First: `#7BE1A4` (green)
- Protocol Interop: `#979BFF` (lavender)
- Pluggable Everything: `#FF8585` (red)
- Voice Pipeline: `#FFD97B` (golden)

---

## Part 2: Homepage Layout Fixes

### Hero Section (`Hero.astro`)
- Ensure hero title and tagline are properly centered with `text-center` on the container
- Add `max-width` constraint on tagline for readable line length
- Ensure search container is centered with consistent width

### Section Titles (`Section.astro` + CSS)
- Verify `.section-description` has `max-width` and `margin: 0 auto` for centered readable text
- Ensure no CSS overrides defeat `text-center`

### Stats Row (StatBadge)
- Add consistent `min-width` on StatBadge cards to prevent uneven sizing at different viewports

### Card Grids (Grid + FeatureCard + NewCard)
- Ensure equal card heights within each row (grid `align-items: stretch` or flex equivalent)
- Verify consistent padding and gap across 4-column and 2-column layouts

### Code Block
- Ensure the "Get started in minutes" code block is properly centered within its section

---

## Files Affected

**Modified**:
- `docs/website/src/config/theme.json` — primary color
- `docs/website/src/styles/global.css` — gradient CSS variable
- `docs/website/src/components/override-components/Hero.astro` — logo glow, drop shadow, hero layout
- `docs/website/src/lib/og-image.ts` — OG image accent color and background
- `docs/website/src/components/Section.astro` — section description centering
- `docs/website/src/components/user-components/StatBadge.astro` — min-width for consistency
- `docs/website/src/components/user-components/Grid.astro` — equal height alignment
- `docs/website/src/styles/components.css` — section description max-width

---

## Out of Scope

- Logo redesign (keeping current logo as-is)
- Light mode color adjustments (focus on dark mode first)
- Feature card individual accent colors
- Font changes
- Content rewrites
