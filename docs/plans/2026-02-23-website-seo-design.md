# Website SEO Comprehensive Overhaul — Design

**Date**: 2026-02-23
**Branch**: `fix/website-seo`
**Site**: `https://beluga-ai.org` (Astro + Starlight, GitHub Pages)

## Problem

Social previews are generic (all pages share the same OG image), structured data is minimal, and several technical SEO best practices are missing. The site needs a comprehensive SEO overhaul covering social previews, structured data, technical hygiene, and performance.

## Approach

Incremental enhancement in four layers, each independently valuable:

1. Social preview / OG image system (highest impact, fixes immediate pain)
2. Comprehensive structured data
3. Technical SEO polish
4. Performance SEO

---

## Layer 1: Social Preview / OG Image System

### Current State

All pages use `/hero-icon.svg` as their OG image. Social shares on Twitter, LinkedIn, Discord, and Slack all look identical.

### Design

Auto-generate unique OG images per page at build time using `satori` + `@resvg/resvg-js`. Allow frontmatter override for key pages.

**Build-time generation**:
- Astro API endpoint at `/og/[...slug].png` generates 1200x630px PNG images
- Each image renders: page title (large), description (smaller), Beluga AI logo, site URL
- Visual style: dark background matching site theme, gradient accent, branded layout

**Head.astro changes**:
- Reference `/og/{slug}.png` instead of static SVG
- Add `og:image:width`, `og:image:height`, `og:image:type` meta tags
- Set `twitter:card` to `summary_large_image`

**Frontmatter override**:
```yaml
ogImage: /custom-og-image.png  # optional, overrides auto-generated
```

**Dependencies**: `satori`, `@resvg/resvg-js`, bundled `.ttf` font file (satori requires local fonts).

---

## Layer 2: Comprehensive Structured Data

### Current State

BreadcrumbList + TechArticle on all doc pages. No differentiation by content type.

### Design

Contextually apply schema types based on URL pattern and optional frontmatter:

| Content Area | Schema Type | Detection |
|---|---|---|
| Homepage | `SoftwareApplication` + `Organization` | `index.mdx` |
| API Reference | `TechArticle` | `/api-reference/**` |
| Guides | `TechArticle` | `/guides/**` |
| Tutorials | `TechArticle` | `/tutorials/**` |
| FAQ content | `FAQPage` | Frontmatter `schemaType: faq` |
| Homepage only | `WebSite` | Splash template |

**Implementation**:
- Schema selection function in `Head.astro` mapping URL patterns to types
- Optional frontmatter field: `schemaType`
- `WebSite` schema on homepage only
- `Organization` schema for Lookatitude on homepage

---

## Layer 3: Technical SEO Polish

### Meta Tag Completeness
- `og:image:width` (1200), `og:image:height` (630), `og:image:type` (image/png)
- `og:image:alt` for accessibility
- `twitter:card` as `summary_large_image`

### Performance Hints
- `<link rel="preconnect">` for Google Fonts
- `<link rel="dns-prefetch">` for external resources
- Verify font `display: swap`

### 404 Improvements
- Add `<meta name="robots" content="noindex">` to 404 page

Note: Security headers and caching are managed by GitHub Pages and cannot be customized via config files.

---

## Layer 4: Performance SEO

### Image Optimization
- Explicit `width`/`height` on `<img>` tags to prevent CLS
- `fetchpriority="high"` on above-fold hero images
- `loading="lazy"` on below-fold images

### Resource Loading
- Audit font loading order (critical fonts first)
- Verify CSS inlining/preloading for critical path
- `modulepreload` for critical JS if applicable

---

## Files Affected

**Modified**:
- `docs/website/src/components/override-components/Head.astro` — OG image refs, meta tags, structured data
- `docs/website/astro.config.mjs` — site URL
- `docs/website/src/content/docs/404.md` — noindex meta

**New**:
- `docs/website/src/pages/og/[...slug].png.ts` — OG image generation endpoint
- `docs/website/src/lib/og-image.ts` — OG image rendering template
- `docs/website/src/assets/fonts/` — Bundled font for satori

---

## Out of Scope

- Internationalization / multi-language support
- Content rewrites for SEO (keyword optimization in copy)
- Google Search Console setup / verification
- Analytics integration
- Link building strategy
