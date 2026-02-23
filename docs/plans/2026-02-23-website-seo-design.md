# Website SEO Comprehensive Overhaul — Design

**Date**: 2026-02-23
**Branch**: `fix/website-seo`
**Site**: `https://beluga-ai.dev` (Astro + Starlight, Netlify)

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
| Tutorials | `HowTo` with steps | `/tutorials/**` |
| API Reference | `TechArticle` | `/api-reference/**` |
| Guides | `TechArticle` | `/guides/**` |
| FAQ content | `FAQPage` | Frontmatter `schemaType: faq` or Q&A heading detection |
| Code examples | `CodeSample` embedded in Article | Fenced code block detection |
| All pages | `WebSite` with `SearchAction` | Global |

**Implementation**:
- Schema selection function in `Head.astro` mapping URL patterns to types
- Optional frontmatter fields: `schemaType`, `steps` (for HowTo)
- Global `WebSite` schema with `SearchAction` for sitelinks search box
- `Organization` schema for Lookatitude on homepage

---

## Layer 3: Technical SEO Polish

### Meta Tag Completeness
- `og:image:width` (1200), `og:image:height` (630), `og:image:type` (image/png)
- `twitter:card` as `summary_large_image`
- `og:type`: `article` for docs, `website` for homepage

### Performance Hints
- `<link rel="preconnect">` for Google Fonts
- `<link rel="dns-prefetch">` for external resources
- Verify font `display: swap`

### Redirect Infrastructure
- Create `public/_redirects` with Netlify format and comment template
- No specific redirects needed yet, just the structure

### 404 Improvements
- Verify HTTP 404 status code
- Add `<meta name="robots" content="noindex">` to 404 page

### Security Headers (via `netlify.toml`)
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`

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

### Caching (via Netlify headers)
- Long `Cache-Control` + `immutable` for static assets (images, fonts, JS/CSS)
- Shorter cache for HTML pages

---

## Files Affected

**Modified**:
- `docs/website/src/components/override-components/Head.astro` — OG image refs, meta tags, structured data
- `docs/website/astro.config.mjs` — integration config if needed
- `docs/website/netlify.toml` — security headers, cache rules
- `docs/website/src/content/docs/404.md` — noindex meta
- `docs/website/src/components/ImageMod.astro` — width/height attributes

**New**:
- `docs/website/src/pages/og/[...slug].png.ts` — OG image generation endpoint
- `docs/website/src/lib/og-image.ts` — OG image rendering template
- `docs/website/public/_redirects` — Netlify redirect rules (empty template)
- `docs/website/src/assets/fonts/` — Bundled font for satori

---

## Out of Scope

- Internationalization / multi-language support
- Content rewrites for SEO (keyword optimization in copy)
- Google Search Console setup / verification
- Analytics integration
- Link building strategy
