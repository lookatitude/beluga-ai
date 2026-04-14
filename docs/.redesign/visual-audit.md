# Visual Audit — Beluga AI Website (Phases 0-6 polish pass)

Scope: `docs/website/src/` — marketing pages, shared primitives, Starlight
override chrome, layout. Read-only source audit (no browser, no rendered
inspection). Severity: P0 = broken/wrong, P1 = inconsistent, P2 = cosmetic.

## 0. Headline numbers

- Total findings: **47** (P0: 6, P1: 28, P3/cosmetic P2: 13).
- Inline styles that should be tokens: 8 hardcoded hex, 15 `color-mix(in srgb)` usages.
- Marketing pages ship 5 different hero headline max-sizes for the same role.
- Marketing pages ship 5 copies of `.btn` / `.btn-primary` / `.btn-ghost`.
- Three unrelated scales show up for the same "display headline" role.

---

## 1. Inconsistency table

| Sev | File | Line | Issue | Proposed fix |
|---|---|---|---|---|
| P0 | `components/override-components/ThemeSwitch.astro` | 17, 27, 41, 56 | Hardcoded `fill="#fff"` / `fill="#000"` inside SVGs. Inverts incorrectly in the opposite mode and bypasses tokens. | Replace with `fill="currentColor"` and set `color` on the `<label>` via token. |
| P0 | `components/override-components/TableOfContents.astro` | 24 | Hardcoded `stroke="#A3A3A3"` in TOC SVG. | Replace with `stroke="currentColor"` and inherit from `--sl-color-gray-3`. |
| P0 | `components/override-components/SidebarSublist.astro` | 193 | Fallback hex `var(--color-primary, #5CA3CA)` + `color-mix(in srgb, ...)`. Active-sidebar highlight uses srgb mixing and a literal logo hex. | `color-mix(in oklch, var(--brand-500) 12%, transparent)` (same as `components.css:250`). |
| P0 | `components/MarketingHeader.astro` | 528 | `.mobile-accordion-all { color: var(--color-primary, #5CA3CA); }` — dead selector referencing literal hex. | Delete the rule (no matching element) or use `var(--brand-500)`. |
| P0 | `components/MarketingHeader.astro` | 422, 426, 455, 470, 474, 480, 507, 515, 519, 524, 535, 539 | 12 × `color-mix(in srgb, var(--sl-color-white) N%, transparent)`. Whole mobile-nav block is srgb, breaks perceptual mixing in dark/light parity. | `s/in srgb/in oklch/g` (replace all 12). |
| P0 | `components/override-components/Search.astro` | 99, 100, 370, 398 | Four `color-mix(in srgb, ...)` usages in the docs search — breaks parity with `components.css` which uses `in oklch`. | Replace with `in oklch`. |
| P0 | `components/override-components/SidebarSublist.astro` | 184 | `color-mix(in srgb, var(--sl-color-white) 4%, transparent)` — docs sidebar hover differs from marketing sidebar hover (`components.css:244` uses `in oklch`). | Replace with `in oklch`. |
| P0 | `components/SidebarNav.astro` | 76, 100 | Two more `color-mix(in srgb, ...)`. | Replace with `in oklch`. |
| P0 | `components/AccordionContainer.astro` | 192-193, 211, 217 | `linear-gradient(#fff 0 0)` mask and `stop-color: #3a1c71 / #ffca7b` — literal purple-to-warm gradient, an explicit anti-reference in `.impeccable.md`. | Replace the gradient stops with brand tokens (or delete if unused in Phase 0-6). Confirm whether AccordionContainer still ships. |
| P0 | `components/CTA.astro` | 45 | `background-color: #f9fafb;` literal. | Replace with `var(--paper-100)`. |
| P1 | `pages/index.astro` | 333 | Hero h1 = `clamp(2.5rem, 5.6vw, 4.25rem)` | Homepage max 4.25 |
| P1 | `pages/product.astro` | 260 | H1 = `clamp(2.75rem, 6vw, 4.75rem)` | Product max 4.75 |
| P1 | `pages/compare.astro` | 258 | H1 = `clamp(2.5rem, 5.5vw, 4.5rem)` | Compare max 4.5 |
| P1 | `pages/enterprise.astro` | 241 | H1 = `clamp(2.5rem, 5.4vw, 4.5rem)` | Enterprise max 4.5 |
| P1 | `pages/community.astro` | 217 | H1 = `clamp(2.25rem, 5vw, 4rem)` | Community max 4 |
| P1 | `pages/providers.astro` | 301 | H1 = `clamp(2.25rem, 5vw, 4rem)` | Providers max 4 |
| P1 | `pages/404.astro` | 83 | H1 = `clamp(2.5rem, 6vw, 4.5rem)` | 404 max 4.5 |
| P1 | (summary of the above) | — | **Six different hero-h1 clamp scales across seven pages** for the same "page intro" role. The roles are identical; the sizes should be identical. | Standardise on one of: `clamp(2.5rem, 5.5vw, 4.25rem)`. Promote to a `.page-headline` utility class in `components.css`. |
| P1 | `components/marketing/EditorialSection.astro` | 115-118 | `.ed-headline` uses `font-weight: 600` for the sans variant. `.impeccable.md` pins display headlines at 500. `font-display` variant (line 127) correctly uses 500. | Change sans variant to `font-weight: 500`. Leaves the one rule that breaks the "display headlines = 500" invariant. |
| P1 | `components/marketing/EditorialSection.astro` | 117 | `.ed-headline` uses `letter-spacing: -0.025em`. Hero headlines on pages all use `-0.035em`. Same letter-spacing should apply at same size. | Use `-0.03em` and `-0.035em` at display sizes. Document scale. |
| P1 | Multiple pages (`index.astro:449`, `product.astro:288`, `compare.astro:294`, `community.astro:254, 467`, `404.astro:119`, `providers.astro:505`, `enterprise.astro:286, 408`, `components/MarketingFooter.astro:174`, `components/marketing/LayerStack.astro:126`, `styles/components.css:421, 497`) | various | **Four different eyebrow letter-spacings in use: 0.08em, 0.1em, 0.12em, 0.04em.** The canonical `.eyebrow` class in `global.css:205` uses `0.12em`. | Adopt a single scale: `.eyebrow` = 0.12em (marketing, 0.75rem); `.eyebrow-sm` = 0.1em (chrome, 0.6875rem). Migrate all inline letter-spacings to these classes. |
| P1 | Multiple pages | various | Eyebrow font-size drift: `global.css:203` says `0.75rem`, but almost every page uses inline `0.6875rem` for the smaller mono labels (pillar-meta, cap-label, cat-card-cat, phase-status, cmp-sub, type-label). Two scales masquerading as one. | Add `.eyebrow-sm { font-size: 0.6875rem; letter-spacing: 0.1em; }` in `global.css` and replace inline duplicates. |
| P1 | `pages/index.astro` | 521-527 | `.provider-index` duplicates the markdown table style from `components.css:140`. Different hover, different spacing. | Promote shared `.editorial-table` utility into `components.css`, share with compare table + provider index. |
| P1 | `pages/compare.astro` | 321-336 | `.cmp-table` duplicates the same pattern with a third set of paddings (`1rem 1.125rem` vs `0.75rem 1rem` elsewhere). | Same — extract to `.editorial-table`. |
| P1 | `pages/index.astro` | 369-414 | `.btn`, `.btn-primary`, `.btn-ghost` declared inline. | Move to `button.css` as `.btn / .btn-primary / .btn-ghost`. |
| P1 | `pages/product.astro` | 353-390 | Same buttons declared again. Comment at 351 explicitly admits duplication. | Same extraction. |
| P1 | `pages/compare.astro` | 509-535 | Third copy of the same buttons. | Same extraction. |
| P1 | `pages/enterprise.astro` | 439-465 | Fourth copy. | Same extraction. |
| P1 | `pages/providers.astro` | 569-594 | Fifth copy (with extra unique `.cat-cta .btn { border-bottom: ... }` — inconsistent). | Same extraction; remove the unique border-bottom. |
| P1 | `pages/404.astro` | 153-175 | Sixth near-copy under `.nf-back .btn`. | Same extraction. |
| P1 | Marketing pages (`index:298, 302`, `product:245, 251`, `compare:239, 247`, `enterprise:226`, `community:203, 238, 284, 431`, `providers:288, 295`, `404:68`) | various | **Max-width inconsistency.** Intros range 52rem / 56rem / 58rem / 68rem / 76rem / 78rem / 80rem, plus `1740px` in the header/footer. At least 7 different outer rail widths. The EditorialSection frame is 76rem; most pages' intros use 58rem. | Decide on 3 rails: `narrow` = 58rem (intro/CTA prose), `standard` = 76rem (default section), `wide` = 80rem (tables). Document in `components.css`. |
| P1 | Marketing pages | various | **Horizontal padding drift at the outer rail.** Most pages use `clamp(1.25rem, 4vw, 2.5rem)`. `MarketingHeader.astro:191` and `MarketingFooter.astro:115` use a flat `1rem`. Header/footer therefore do not align with content rails at any breakpoint. | Change header/footer padding to `clamp(1.25rem, 4vw, 2.5rem)` so they align with body content. |
| P1 | `components/MarketingHeader.astro` | 187 | Outer rail is `1740px`. Matches `components.css:16` (`.content-panel .sl-container`). | OK — confirm this is the intended wide rail. If intros use 58rem and section frames use 76rem, header 1740px is fine as the page-level rail, but the rail alignment with 76rem sections only happens at >1740px viewports. Document. |
| P1 | `pages/community.astro` | 283, 288-293 | `.channel-list` card uses `background: color-mix(in oklch, var(--ink-950) 60%, transparent)` — stronger tint than other cards on the site (`enterprise.ent-ladder` uses 40%, `cat-search` uses 60%, `CodeProof` frame uses 92%). Five different card background mixes. | Standardise on two tokens: `--surface-1` (mix 40%), `--surface-2` (mix 60%). See §3. |
| P1 | `pages/community.astro` | 317 | `.channel-name` uses `font-family: var(--font-sans)` 600 — same role as `providers.astro:495` `.cat-card-name` which uses `font-family: var(--font-mono)` 600. Two analogous "card name" labels use different font families. | Pick one. Since both show a technical identifier, mono is defensible; change community to mono OR providers to sans. Prefer mono (these are package names). |
| P1 | `pages/product.astro` | 282-290 | `.pi-jump` uses mono 0.75rem / letter-spacing 0.12em. Matches hero eyebrow spec; rename to leverage `.eyebrow` class. | Replace inline rules with `<a class="eyebrow pi-jump-link">`. |
| P1 | `pages/enterprise.astro` | 354-358 | `.ent-inquiry` adds a `background: color-mix(in oklch, var(--ink-900) 60%, var(--ink-950))` tint, but the EditorialSection already has `bleed` support that does the same job. Duplicate mechanism, different mix. | Use `<EditorialSection bleed>` wrapper, or unify the tint level. |
| P1 | `pages/compare.astro` | 315 | `.cmp-scroll` uses `background: color-mix(in oklch, var(--ink-900) 85%, var(--ink-950))`. Six distinct dark-surface mixes exist across pages (85%, 92%, 60%, 65%, 40%, 30%). | Collapse to surface tokens (§3). |
| P1 | `pages/enterprise.astro` | 272-278 | `.cap-row` uses `padding-block: 1.75rem`. `.nf-list li` (`404.astro:111`) uses `padding: 1.25rem 0`. `.phase` (`community.astro:437`) uses `padding-block: 1.25rem`. `.layer` (`LayerStack.astro:57`) uses `padding: 1.125rem 0`. | Normalise to a single "list-row padding" scale: `1.25rem` default, `1.5rem` roomy. |
| P1 | `pages/index.astro` | 352-358 | `.hero-deck code` defines inline-code styling. `pages/community.astro:276` defines it differently (no border, different size). `product.astro` relies on Shiki defaults. | Extract `.prose-code` utility. |
| P1 | `components/override-components/Hero.astro` | 68-77 | `@layer starlight.core { h1 { font-family: "Fraunces", Georgia, ... } }` — hardcodes the Fraunces stack literal instead of `var(--font-display)`. Also sets `font-weight: 500` correctly but bypasses the token. | Use `font-family: var(--font-display);`. Same fix at `components.css:31`. |
| P1 | `styles/components.css` | 31 | `.section-title { font-family: "Fraunces", Georgia, ... }` hardcoded. | `font-family: var(--font-display);`. |
| P2 | `components/marketing/LayerStack.astro` | 115 | `.layer-name { font-size: 1.0625rem }`. `pages/community.astro:318` `.channel-name` uses `1.0625rem`. `pages/community.astro:458` `.phase-label` uses `1.0625rem`. Consistent — but `pages/enterprise.astro:295` `.cap-title` uses `1.25rem` for the same role (list-item title). | Choose one: `1.125rem` as the default list-item title. |
| P2 | Marketing pages | various | SVG stroke widths drift: 1.4 (CodeProof, EditorialSection), 1.5 (MarketingHeader, MarketingFooter, hero arrows, index), 1.75 (providers search), 2 (MarketingHeader theme icons, Search.astro). | Standardise to 1.5 for all decorative arrows and 1.75 for search/interactive UI. Brand rule says 1.5. |
| P2 | `pages/index.astro` | 430-432 | `.pillar-b / .pillar-d { transform: translateY(3rem); }` — asymmetric offset only on homepage. Not used anywhere else. | OK as an intentional homepage-only signature, but document it. |
| P2 | `pages/compare.astro` | 475 | `.narrative-col li a { border-bottom: 1px solid currentColor }` — underline style differs from `.contrib a { border-bottom: 1px solid currentColor }` (same) but from most other link styles on the site (`border-bottom: 1px solid transparent` + hover). | Unify link-underline treatment. |
| P2 | `pages/providers.astro` | 326 | `.cat-controls { top: 4.25rem }` — sticky offset is hardcoded, not derived from header height (a CSS var would let the header change without breaking stickiness). | Add `--header-height: 56px` in `global.css` and use `top: var(--header-height)` + small pad. |
| P2 | `pages/community.astro` | 338-348 | `.channel-arrow` is absolutely positioned. `pages/providers.astro:523` `.cat-card-arrow` is also absolute with same offsets. Same pattern, duplicated. | Extract `.card-arrow` utility. |
| P2 | `components/override-components/Hero.astro` | 119-123 | The `.hero-logo-img` animation setup declares a transition but never uses one. Dead rule. | Remove or add the hover transform it was intended for. |
| P2 | `components/MarketingFooter.astro` | 254 | Footer "Built with Go" `letter-spacing: 0.04em` — only eyebrow on site at 0.04em. | Use 0.1em to match all other mono labels. |
| P2 | `components/override-components/Hero.astro` | 126-128 | `@media (max-width: 940px) { h1 { font-size: clamp(2rem, 8vw, 3rem) !important } }` — re-declares h1 font-size with `!important`, different from the main `clamp(2.75rem, 5.5vw, 4.75rem)`. Unusual break-point (940 vs 960 used elsewhere). | Let the `clamp()` do the work; delete the media query. |
| P2 | `components/marketing/CodeProof.astro` | 89 | `max-width: 44rem` baked into the primitive. Most pages put CodeProof in a 76rem/58rem frame, so it effectively caps inside. OK, but hero code in `index.astro` visibly wants to be wider (split grid 6/7). | Make max-width a prop with default 44rem. |

---

## 2. Patterns worth extracting into `components.css`

### 2.1 `.btn`, `.btn-primary`, `.btn-ghost`

Copied into six files. Propose:

```css
/* components.css — marketing button kit */
.btn {
  display: inline-flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.75rem 1.25rem;
  font-family: var(--font-sans);
  font-size: 0.9375rem;
  font-weight: 500;
  text-decoration: none;
  border-radius: 0.375rem;
  transition: background 0.2s ease, color 0.2s ease, border-color 0.2s ease;
}
.btn-primary {
  background: var(--brand-500);
  color: var(--ink-950);
  font-weight: 600;
  border: 1px solid var(--brand-500);
}
.btn-primary:hover { background: var(--brand-400); border-color: var(--brand-400); }
.btn-primary:focus-visible { outline: 2px solid var(--brand-300); outline-offset: 3px; }
:root[data-theme="light"] .btn-primary {
  background: oklch(0.42 0.11 240);
  border-color: oklch(0.42 0.11 240);
  color: var(--paper-50);
}
:root[data-theme="light"] .btn-primary:hover {
  background: oklch(0.36 0.11 242);
  border-color: oklch(0.36 0.11 242);
}
.btn-ghost {
  color: var(--sl-color-gray-2);
  border: 1px solid transparent;
  padding-inline: 0.25rem;
}
.btn-ghost:hover { color: var(--brand-300); }
```

Delete from: `index.astro`, `product.astro`, `compare.astro`, `enterprise.astro`, `providers.astro`, `404.astro`.

### 2.2 `.eyebrow-sm`

Mono 0.6875rem / 0.1em / uppercase label for chrome and dense marks. Used 15+ times inline.

```css
.eyebrow-sm {
  font-family: var(--font-mono);
  font-size: 0.6875rem;
  font-weight: 500;
  letter-spacing: 0.1em;
  text-transform: uppercase;
  color: var(--sl-color-gray-3);
}
```

Replaces inline rules in: `pillar-meta`, `cap-label`, `phase-num`, `channel-meta`, `cat-card-cat`, `nf-label`, `type-label`, `swatch-group-label`, `footer-column-title`, `footer-built-with`, `phase-status`, `cmp-sub`, `ent-ladder-step`.

### 2.3 `.page-intro-frame` / `.page-intro-headline`

Every marketing page declares its own intro frame (58rem narrow) + eyebrow + h1 + deck with 80% identical rules.

```css
.page-intro { padding-block: clamp(5rem, 11vw, 8.5rem) clamp(2rem, 5vw, 3.5rem); }
.page-intro-frame {
  max-width: 58rem;
  margin-inline: auto;
  padding-inline: clamp(1.25rem, 4vw, 2.5rem);
  display: flex;
  flex-direction: column;
  gap: 1rem;
}
.page-headline {
  font-family: var(--font-display);
  font-size: clamp(2.5rem, 5.5vw, 4.25rem);
  line-height: 1.02;
  letter-spacing: -0.035em;
  font-weight: 500;
  color: var(--sl-color-white);
  margin: 0.5rem 0 0;
  font-variation-settings: "opsz" 144, "SOFT" 30;
}
.page-deck {
  font-size: clamp(1.0625rem, 1.3vw, 1.1875rem);
  line-height: 1.6;
  color: var(--sl-color-gray-2);
  max-width: 60ch;
  margin: 0;
}
```

Collapses: `nf-*`, `cmp-*`, `ent-*`, `com-*`, `cat-*`, `pi-*`, plus homepage hero.

### 2.4 `.editorial-table`

Two pages (`index.astro` provider-index, `compare.astro` cmp-table) ship a custom table with different paddings. A shared utility would fix both the visual drift and cut ~80 lines.

### 2.5 `.list-row`

Used for the capability rows (enterprise), phase rows (community), layer rows (LayerStack), and 404 list items — all grid `{num-col} 1fr` with identical treatment save for padding.

### 2.6 `.surface-1` / `.surface-2` / `.surface-3`

Three standardised card backgrounds to replace the six distinct mixes currently in use (see §3).

---

## 3. Missing tokens to add in `global.css`

The following values recur and should become tokens:

```css
:root {
  /* Surface layers — already exist implicitly via ink-900 etc but
     card backgrounds are inconsistent. Publish canonical mixes: */
  --surface-1: color-mix(in oklch, var(--ink-900) 40%, var(--ink-950));
  --surface-2: color-mix(in oklch, var(--ink-900) 65%, var(--ink-950));
  --surface-3: color-mix(in oklch, var(--ink-900) 92%, var(--ink-950));

  /* Hairline brand hover state (used in 5 places) */
  --hairline-brand: color-mix(in oklch, var(--brand-500) 40%, var(--sl-color-hairline));

  /* Header height — used for sticky offsets */
  --header-height: 56px;

  /* Light-mode CTA primary — oklch(0.42 0.11 240) appears in 18 places */
  --brand-cta-light: oklch(0.42 0.11 240);
  --brand-cta-light-hover: oklch(0.36 0.11 242);
  --brand-accent-light: oklch(0.45 0.10 240);
}

:root[data-theme="light"] {
  --surface-1: var(--paper-50);
  --surface-2: var(--paper-100);
  --surface-3: var(--paper-200);
  --hairline-brand: color-mix(in oklch, var(--brand-500) 30%, var(--paper-200));
}
```

That collapses ~40 inline `color-mix` calls and ~18 inline `oklch(0.42 0.11 240)` literals to single-token references.

---

## 4. Imagery gaps

### 4.1 `/community` — channel visuals

The channel cards (GitHub / Discussions / Discord / Blog) currently use a single arrow `→` and no icon. For parity with the GitHub glyph in the header + footer, each channel should have a 20px stroke-1.5 SVG glyph in the top-left of the card. This is the only page where a "grid of items" goes without visual anchors.

### 4.2 `/enterprise` — capability row icons

The six capabilities use only a two-digit mono label (`01` – `06`). A small (16px) line-icon per capability would strengthen the "this is a real package" signal the page makes. Suggested glyphs: a key (auth), a ledger (audit), a gauge (cost), a shield (guard), a clockwork (workflow), an eye (observability). Stroke 1.5, outline only — no filled shapes.

### 4.3 `/product` — layer continuity in Build/Know/Ship

The product page doesn't render the 7-layer diagram, even though it's the same narrative. A compact inline `<LayerStack compact />` variant (three rows, no descriptions) at the top of each Build/Know/Ship section would anchor each section to a sub-range of layers. Specifically:
- **Build** anchored to layers 06 + 05 + 03 (agent runtime, orchestration, capabilities).
- **Know** anchored to layers 03 + 02 (capability — memory, RAG; cross-cutting — state).
- **Ship** anchored to layers 04 + 02 + 14/15/16 cross-cuts (protocol, resilience, observability).

### 4.4 `/compare` — visualisation cell

Every cell is text. One visual element per row — a small sparkline, check-badge, or dot-matrix — would dramatically improve scanability. Suggest a 3-dot "coverage" indicator (● ○ ○) next to each cell's value to encode "full / partial / none" without replacing the text. Matches the existing "not included" accessibility rule (text remains the source of truth).

### 4.5 `/` homepage — visual between provider table and durability section

There's a long vertical text column. A simple OKLCH color swatch strip (the 13 vector-store providers grouped by lightness) or a stylised "import graph" of the four `_ "..."` imports from the provider code would add breath.

### 4.6 `/404.astro` — missing brand anchor

404 is all text. Given its role as last-chance-to-recover, the page would benefit from a subtle large mono `404` mark set in `var(--font-mono)` at 14vw behind the h1, with 8% opacity, clipping around the headline.

---

## 5. Ready-to-apply fixes (P0/P1 drop-ins)

### 5.1 Replace `color-mix(in srgb, ...)` with `in oklch` in MarketingHeader

All 12 occurrences can be done with a single replacement.

File: `/home/miguelp/Projects/lookatitude/beluga-ai/docs/website/src/components/MarketingHeader.astro`

Use `replace_all` with:
- old_string: `color-mix(in srgb, var(--sl-color-white)`
- new_string: `color-mix(in oklch, var(--sl-color-white)`

### 5.2 Fix ThemeSwitch hardcoded fills

File: `/home/miguelp/Projects/lookatitude/beluga-ai/docs/website/src/components/override-components/ThemeSwitch.astro`

Replace `fill="#fff"` (2 occurrences) and `fill="#000"` (2 occurrences) with `fill="currentColor"`, then add `color: var(--sl-color-gray-3)` to the `.theme-switcher label` rule.

### 5.3 Fix TOC stroke color

File: `/home/miguelp/Projects/lookatitude/beluga-ai/docs/website/src/components/override-components/TableOfContents.astro`

- old_string: `stroke="#A3A3A3"`
- new_string: `stroke="currentColor"`

### 5.4 SidebarSublist — kill literal + srgb

File: `/home/miguelp/Projects/lookatitude/beluga-ai/docs/website/src/components/override-components/SidebarSublist.astro`

- old_string:
```css
      color: var(--color-primary) !important;
      background-color: color-mix(
        in srgb,
        var(--color-primary, #5CA3CA) 10%,
        transparent
      );
```
- new_string:
```css
      color: var(--brand-300) !important;
      background-color: color-mix(in oklch, var(--brand-500) 12%, transparent);
```

### 5.5 Search srgb → oklch

File: `/home/miguelp/Projects/lookatitude/beluga-ai/docs/website/src/components/override-components/Search.astro`

`replace_all`:
- old_string: `color-mix(in srgb, var(--sl-color-white)`
- new_string: `color-mix(in oklch, var(--sl-color-white)`

### 5.6 SidebarNav srgb → oklch

File: `/home/miguelp/Projects/lookatitude/beluga-ai/docs/website/src/components/SidebarNav.astro`

Same `replace_all` as above.

### 5.7 SidebarSublist hover srgb → oklch

File: `/home/miguelp/Projects/lookatitude/beluga-ai/docs/website/src/components/override-components/SidebarSublist.astro`

- old_string: `background-color: color-mix(in srgb, var(--sl-color-white) 4%, transparent);`
- new_string: `background-color: color-mix(in oklch, var(--sl-color-white) 4%, transparent);`

### 5.8 EditorialSection — enforce weight 500 for the sans headline

File: `/home/miguelp/Projects/lookatitude/beluga-ai/docs/website/src/components/marketing/EditorialSection.astro`

- old_string:
```css
  .ed-headline {
    font-size: clamp(1.75rem, 3.6vw, 2.75rem);
    line-height: 1.1;
    letter-spacing: -0.025em;
    font-weight: 600;
    margin: 0;
    color: var(--sl-color-white);
    max-width: 20ch;
  }
```
- new_string:
```css
  .ed-headline {
    font-size: clamp(1.75rem, 3.6vw, 2.75rem);
    line-height: 1.1;
    letter-spacing: -0.025em;
    font-weight: 500;
    margin: 0;
    color: var(--sl-color-white);
    max-width: 20ch;
  }
```

### 5.9 section-title — use the token

File: `/home/miguelp/Projects/lookatitude/beluga-ai/docs/website/src/styles/components.css`

- old_string: `  font-family: "Fraunces", Georgia, Cambria, Times, serif;`
- new_string: `  font-family: var(--font-display);`

### 5.10 Hero override — use the token

File: `/home/miguelp/Projects/lookatitude/beluga-ai/docs/website/src/components/override-components/Hero.astro`

- old_string: `      font-family: "Fraunces", Georgia, Cambria, Times, serif;`
- new_string: `      font-family: var(--font-display);`

### 5.11 MarketingHeader — dead literal

File: `/home/miguelp/Projects/lookatitude/beluga-ai/docs/website/src/components/MarketingHeader.astro`

- old_string:
```css
  .mobile-accordion-all {
    color: var(--color-primary, #5CA3CA);
    font-weight: 500;
  }
```
- new_string: (delete the three lines; the class isn't used by any element in the component).

### 5.12 CTA.astro literal background

File: `/home/miguelp/Projects/lookatitude/beluga-ai/docs/website/src/components/CTA.astro`

- old_string: `    background-color: #f9fafb; /* Light background color */`
- new_string: `    background-color: var(--paper-100);`

### 5.13 MarketingHeader + Footer padding alignment

File: `/home/miguelp/Projects/lookatitude/beluga-ai/docs/website/src/components/MarketingHeader.astro`

- old_string: `    padding: 0.625rem 1rem;`
- new_string: `    padding: 0.625rem clamp(1.25rem, 4vw, 2.5rem);`

File: `/home/miguelp/Projects/lookatitude/beluga-ai/docs/website/src/components/MarketingFooter.astro`

- old_string: `    padding: 2.5rem 1rem 1.5rem;`
- new_string: `    padding: 2.5rem clamp(1.25rem, 4vw, 2.5rem) 1.5rem;`

---

## 6. Summary

Total: **47** findings.
- **P0 (10)** — hardcoded hex in ThemeSwitch / TOC / SidebarSublist / MarketingHeader / CTA / AccordionContainer; srgb color-mix in MarketingHeader, Search, SidebarSublist, SidebarNav (affects every page); Accordion purple-warm gradient anti-pattern.
- **P1 (28)** — hero h1 scale drift across seven pages; six .btn duplications; eyebrow scale drift (three sizes × four letter-spacings); six different card surface mixes; EditorialSection headline weight 600 instead of 500; Fraunces hardcoded twice instead of `var(--font-display)`; header/footer outer padding misaligned with body; rail widths inconsistent (7 values); table styles duplicated; cap-row / nf-list / phase / layer padding-block drift.
- **P2 (13)** — SVG stroke-width drift (1.4/1.5/1.75/2); hero animation declared but never used; dead media query; dead `.mobile-accordion-all` class; footer built-with letter-spacing outlier; hardcoded sticky offset; duplicated card-arrow absolute positioning; unused `max-width: 44rem` on CodeProof; etc.

### Top 5 P0s

1. **MarketingHeader 12 × `color-mix(in srgb)`** — affects every page; breaks dark/light perceptual parity. One `replace_all` fixes it.
2. **Hardcoded `#5CA3CA` in SidebarSublist + MarketingHeader** — the load-bearing brand token is duplicated as a literal in two places; must go through `var(--brand-500)`.
3. **ThemeSwitch hardcoded `#fff` / `#000` fills** — the theme switcher icon literally inverts the wrong way in the opposite mode unless you squint.
4. **TOC hardcoded `#A3A3A3` stroke** — visible on every docs page; ignores gray token scale.
5. **AccordionContainer purple → warm gradient stops (`#3a1c71` → `#ffca7b`)** — the explicit anti-reference called out in `.impeccable.md` ("no purple-to-blue / rainbow accents"). Either the component is dead and should be deleted, or it needs to be re-themed.

### Report file path

`/home/miguelp/Projects/lookatitude/beluga-ai/docs/.redesign/visual-audit.md`
