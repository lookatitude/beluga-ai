# Website SEO Comprehensive Overhaul — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Fix bad social previews and add comprehensive SEO improvements to the Beluga AI documentation site.

**Architecture:** Incremental four-layer approach — OG image generation via satori at build time, contextual JSON-LD structured data, technical SEO meta tags and headers, and performance optimizations. All changes are in the `docs/website/` directory.

**Tech Stack:** Astro 5 + Starlight, satori + @resvg/resvg-js for OG images, Netlify for hosting/headers, TypeScript.

---

## Task 1: Install OG Image Dependencies

**Files:**
- Modify: `docs/website/package.json`

**Step 1: Install satori and resvg-js**

Run from `docs/website/`:
```bash
cd docs/website && yarn add satori @resvg/resvg-js
```

**Step 2: Download Inter font for satori**

Satori cannot use web fonts — it needs a local `.ttf` file. Download Inter Regular and Semi-Bold:

```bash
mkdir -p docs/website/src/assets/fonts
curl -L -o docs/website/src/assets/fonts/Inter-Regular.ttf "https://github.com/rsms/inter/raw/master/fonts/truetype/Inter-Regular.ttf"
curl -L -o docs/website/src/assets/fonts/Inter-SemiBold.ttf "https://github.com/rsms/inter/raw/master/fonts/truetype/Inter-SemiBold.ttf"
```

**Step 3: Verify fonts exist**

```bash
ls -la docs/website/src/assets/fonts/
```

Expected: Two `.ttf` files present.

**Step 4: Commit**

```bash
git add docs/website/package.json docs/website/yarn.lock docs/website/src/assets/fonts/
git commit -m "feat(website): add satori and resvg-js for OG image generation"
```

---

## Task 2: Create OG Image Generator

**Files:**
- Create: `docs/website/src/lib/og-image.ts`

**Step 1: Create the OG image rendering module**

This module exports a function that takes a title and description, renders an SVG via satori, and converts it to PNG via resvg.

```typescript
// docs/website/src/lib/og-image.ts
import satori from "satori";
import { Resvg } from "@resvg/resvg-js";
import { readFileSync } from "node:fs";
import { join } from "node:path";

const interRegular = readFileSync(
  join(process.cwd(), "src/assets/fonts/Inter-Regular.ttf")
);
const interSemiBold = readFileSync(
  join(process.cwd(), "src/assets/fonts/Inter-SemiBold.ttf")
);

export async function generateOgImage(
  title: string,
  description: string
): Promise<Buffer> {
  const svg = await satori(
    {
      type: "div",
      props: {
        style: {
          width: "100%",
          height: "100%",
          display: "flex",
          flexDirection: "column",
          justifyContent: "space-between",
          padding: "60px",
          background: "linear-gradient(135deg, #0d0d0d 0%, #1a1a2e 50%, #151515 100%)",
          fontFamily: "Inter",
        },
        children: [
          {
            type: "div",
            props: {
              style: {
                display: "flex",
                flexDirection: "column",
                gap: "20px",
              },
              children: [
                {
                  type: "div",
                  props: {
                    style: {
                      fontSize: "48px",
                      fontWeight: 600,
                      color: "#ffffff",
                      lineHeight: 1.2,
                      maxWidth: "900px",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                    },
                    children: title,
                  },
                },
                {
                  type: "div",
                  props: {
                    style: {
                      fontSize: "24px",
                      fontWeight: 400,
                      color: "#999999",
                      lineHeight: 1.4,
                      maxWidth: "800px",
                      overflow: "hidden",
                      textOverflow: "ellipsis",
                    },
                    children: description,
                  },
                },
              ],
            },
          },
          {
            type: "div",
            props: {
              style: {
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
              },
              children: [
                {
                  type: "div",
                  props: {
                    style: {
                      display: "flex",
                      alignItems: "center",
                      gap: "12px",
                    },
                    children: [
                      {
                        type: "div",
                        props: {
                          style: {
                            width: "40px",
                            height: "40px",
                            borderRadius: "8px",
                            background: "#D76D77",
                          },
                          children: "",
                        },
                      },
                      {
                        type: "div",
                        props: {
                          style: {
                            fontSize: "28px",
                            fontWeight: 600,
                            color: "#ffffff",
                          },
                          children: "Beluga AI",
                        },
                      },
                    ],
                  },
                },
                {
                  type: "div",
                  props: {
                    style: {
                      fontSize: "20px",
                      color: "#666666",
                    },
                    children: "beluga-ai.dev",
                  },
                },
              ],
            },
          },
        ],
      },
    },
    {
      width: 1200,
      height: 630,
      fonts: [
        { name: "Inter", data: interRegular, weight: 400, style: "normal" },
        { name: "Inter", data: interSemiBold, weight: 600, style: "normal" },
      ],
    }
  );

  const resvg = new Resvg(svg, {
    fitTo: { mode: "width", value: 1200 },
  });

  return resvg.render().asPng();
}
```

**Step 2: Verify the file compiles**

```bash
cd docs/website && npx tsc --noEmit src/lib/og-image.ts 2>&1 || echo "Type check done (warnings ok for standalone file)"
```

**Step 3: Commit**

```bash
git add docs/website/src/lib/og-image.ts
git commit -m "feat(website): add OG image rendering module with satori"
```

---

## Task 3: Create OG Image Endpoint

**Files:**
- Create: `docs/website/src/pages/og/[...slug].png.ts`

**Step 1: Create the Astro static endpoint**

This endpoint generates OG images at build time for every documentation page.

```typescript
// docs/website/src/pages/og/[...slug].png.ts
import type { APIRoute, GetStaticPaths } from "astro";
import { getCollection } from "astro:content";
import { generateOgImage } from "../../lib/og-image";

export const getStaticPaths: GetStaticPaths = async () => {
  const docs = await getCollection("docs");
  return docs.map((doc) => ({
    params: { slug: doc.id === "index" ? undefined : doc.id },
    props: {
      title: doc.data.title,
      description:
        doc.data.description || `${doc.data.title} — Beluga AI documentation`,
    },
  }));
};

export const GET: APIRoute = async ({ props }) => {
  const { title, description } = props as { title: string; description: string };
  const png = await generateOgImage(title, description);

  return new Response(png, {
    headers: {
      "Content-Type": "image/png",
      "Cache-Control": "public, max-age=31536000, immutable",
    },
  });
};
```

**Step 2: Test the build generates OG images**

```bash
cd docs/website && yarn build 2>&1 | head -50
```

Expected: Build succeeds. Check that `dist/og/` directory contains `.png` files:

```bash
ls docs/website/dist/og/ 2>/dev/null | head -10
```

**Step 3: Commit**

```bash
git add docs/website/src/pages/og/
git commit -m "feat(website): add OG image generation endpoint for all doc pages"
```

---

## Task 4: Update Head.astro for Dynamic OG Images

**Files:**
- Modify: `docs/website/src/components/override-components/Head.astro`

**Step 1: Update the OG image URL logic**

Replace the static `ogImageURL` with dynamic per-page URLs. Add support for frontmatter `ogImage` override. Add `og:image:width`, `og:image:height`, `og:image:type`. Fix `twitter:card` to `summary_large_image`.

In `Head.astro`, replace lines 17-18:
```typescript
// Default OG image
const ogImageURL = new URL("/hero-icon.svg", Astro.site || "https://beluga-ai.dev").href;
```

With:
```typescript
// Dynamic OG image: frontmatter override > auto-generated > fallback
const customOgImage = (route.entry.data as Record<string, unknown>).ogImage as string | undefined;
const ogSlug = Astro.url.pathname.replace(/^\/|\/$/g, "") || "index";
const defaultOgPath = `/og/${ogSlug === "" ? "index" : ogSlug}.png`;
const ogImageURL = customOgImage
  ? new URL(customOgImage, Astro.site || "https://beluga-ai.dev").href
  : new URL(defaultOgPath, Astro.site || "https://beluga-ai.dev").href;
```

**Step 2: Add OG image dimension and type meta tags**

After line 101 (`<meta property="og:image" content={ogImageURL} />`), add:
```html
<meta property="og:image:width" content="1200" />
<meta property="og:image:height" content="630" />
<meta property="og:image:type" content="image/png" />
```

**Step 3: Fix twitter:card to summary_large_image**

Before the existing twitter meta tags (line 104), add:
```html
<meta name="twitter:card" content="summary_large_image" />
```

**Step 4: Build and verify meta tags in output HTML**

```bash
cd docs/website && yarn build
grep -r "og:image" dist/index.html | head -5
grep -r "twitter:card" dist/index.html | head -5
```

Expected: `og:image` points to `/og/index.png`, `twitter:card` is `summary_large_image`, dimensions present.

**Step 5: Commit**

```bash
git add docs/website/src/components/override-components/Head.astro
git commit -m "feat(website): dynamic OG images with frontmatter override and proper meta tags"
```

---

## Task 5: Add Comprehensive Structured Data

**Files:**
- Modify: `docs/website/src/components/override-components/Head.astro`

**Step 1: Add WebSite schema with SearchAction (global)**

Add this after the existing `articleJsonLd` constant (after line 72 in the original file):

```typescript
// WebSite schema with SearchAction (global, for sitelinks search box)
const siteUrl = (Astro.site || new URL("https://beluga-ai.dev")).href;
const webSiteJsonLd = {
  "@context": "https://schema.org",
  "@type": "WebSite",
  name: "Beluga AI",
  url: siteUrl,
  potentialAction: {
    "@type": "SearchAction",
    target: {
      "@type": "EntryPoint",
      urlTemplate: `${siteUrl}?q={search_term_string}`,
    },
    "query-input": "required name=search_term_string",
  },
};
```

**Step 2: Add Organization schema for homepage**

```typescript
// Organization schema (homepage only)
const organizationJsonLd = isSplash
  ? {
      "@context": "https://schema.org",
      "@type": "Organization",
      name: "Lookatitude",
      url: "https://lookatitude.com",
      logo: ogImageURL,
      sameAs: ["https://github.com/lookatitude"],
    }
  : null;
```

**Step 3: Add SoftwareApplication schema for homepage**

```typescript
// SoftwareApplication schema (homepage only)
const softwareJsonLd = isSplash
  ? {
      "@context": "https://schema.org",
      "@type": "SoftwareApplication",
      name: "Beluga AI",
      applicationCategory: "DeveloperApplication",
      operatingSystem: "Cross-platform",
      programmingLanguage: "Go",
      description:
        "Go-native agentic AI framework with streaming-first design, 22+ LLM providers, RAG, voice AI, and enterprise-grade infrastructure.",
      url: siteUrl,
      author: {
        "@type": "Organization",
        name: "Lookatitude",
        url: "https://lookatitude.com",
      },
      license: "https://opensource.org/licenses/MIT",
    }
  : null;
```

**Step 4: Update articleJsonLd to use contextual schema types**

Replace the existing `articleJsonLd` block (lines 48-72) with:

```typescript
// Contextual article schema based on URL pattern
const pathname = Astro.url.pathname;
const isTutorial = pathname.startsWith("/tutorials/");
const isApiRef = pathname.startsWith("/api-reference/");
const customSchemaType = (route.entry.data as Record<string, unknown>).schemaType as string | undefined;

function getArticleType(): string {
  if (isSplash) return "WebPage";
  if (customSchemaType === "faq") return "FAQPage";
  if (isTutorial) return "HowTo";
  if (isApiRef) return "TechArticle";
  return "TechArticle";
}

const articleType = getArticleType();
const articleJsonLd: Record<string, unknown> = {
  "@context": "https://schema.org",
  "@type": articleType,
  headline: pageTitle,
  description: pageDescription,
  url: canonicalURL.href,
  author: {
    "@type": "Organization",
    name: "Lookatitude",
    url: "https://lookatitude.com",
  },
  publisher: {
    "@type": "Organization",
    name: "Lookatitude",
    url: "https://lookatitude.com",
    logo: {
      "@type": "ImageObject",
      url: ogImageURL,
    },
  },
  image: ogImageURL,
  inLanguage: pageLang,
  ...(route.lastUpdated && { dateModified: route.lastUpdated.toISOString() }),
};

// Add HowTo-specific fields for tutorials
if (articleType === "HowTo") {
  articleJsonLd.step = [
    {
      "@type": "HowToStep",
      name: pageTitle,
      text: pageDescription,
      url: canonicalURL.href,
    },
  ];
}
```

**Step 5: Add the new JSON-LD script tags in the HTML template**

After the existing JSON-LD script tags (after line 122), add:

```html
<!-- JSON-LD Structured Data: WebSite -->
<script type="application/ld+json" set:html={JSON.stringify(webSiteJsonLd)} />

<!-- JSON-LD Structured Data: Organization (homepage only) -->
{organizationJsonLd && (
  <script type="application/ld+json" set:html={JSON.stringify(organizationJsonLd)} />
)}

<!-- JSON-LD Structured Data: SoftwareApplication (homepage only) -->
{softwareJsonLd && (
  <script type="application/ld+json" set:html={JSON.stringify(softwareJsonLd)} />
)}
```

**Step 6: Build and verify structured data**

```bash
cd docs/website && yarn build
grep -c "application/ld+json" dist/index.html
```

Expected: 5 JSON-LD blocks on homepage (BreadcrumbList, Article/WebPage, WebSite, Organization, SoftwareApplication).

```bash
grep -c "application/ld+json" dist/tutorials/index.html 2>/dev/null || grep -c "application/ld+json" dist/getting-started/overview/index.html
```

Expected: 3 blocks on non-homepage pages (BreadcrumbList, Article, WebSite).

**Step 7: Commit**

```bash
git add docs/website/src/components/override-components/Head.astro
git commit -m "feat(website): add comprehensive structured data (WebSite, Organization, SoftwareApplication, HowTo)"
```

---

## Task 6: Technical SEO — 404, Redirects, and Security Headers

**Files:**
- Modify: `docs/website/src/content/docs/404.md`
- Create: `docs/website/public/_redirects`
- Modify: `docs/website/netlify.toml`

**Step 1: Add noindex to 404 page**

Update `docs/website/src/content/docs/404.md` frontmatter to include a robots noindex tag:

```yaml
---
title: '404'
description: "Page not found. Return to the Beluga AI documentation to find guides, API reference, and tutorials for building AI agents in Go."
editUrl: false
head:
  - tag: meta
    attrs:
      name: keywords
      content: "Beluga AI, Go AI framework, documentation, AI agents Go"
  - tag: meta
    attrs:
      name: robots
      content: "noindex, nofollow"
hero:
  title: '404'
  tagline: Page not found. Check the URL or try using the search bar.
---
```

**Step 2: Create redirect infrastructure**

Create `docs/website/public/_redirects`:

```
# Netlify redirect rules for Beluga AI docs
# Format: /old-path /new-path STATUS
# See: https://docs.netlify.com/routing/redirects/
#
# Example:
# /old-page /new-page 301
#
# No redirects needed yet — add entries above this line as URLs change.
```

**Step 3: Add security headers and cache rules to netlify.toml**

Replace `docs/website/netlify.toml`:

```toml
[build]
publish = "dist"
command = "yarn build"

[build.environment]
NODE_VERSION = "20"

# Security headers for all pages
[[headers]]
for = "/*"
[headers.values]
X-Content-Type-Options = "nosniff"
X-Frame-Options = "DENY"
Referrer-Policy = "strict-origin-when-cross-origin"
Permissions-Policy = "camera=(), microphone=(), geolocation=()"

# Cache static assets aggressively
[[headers]]
for = "/_astro/*"
[headers.values]
Cache-Control = "public, max-age=31536000, immutable"

[[headers]]
for = "/og/*"
[headers.values]
Cache-Control = "public, max-age=31536000, immutable"

[[headers]]
for = "*.svg"
[headers.values]
Cache-Control = "public, max-age=604800"

[[headers]]
for = "*.ico"
[headers.values]
Cache-Control = "public, max-age=604800"
```

**Step 4: Build and verify**

```bash
cd docs/website && yarn build
grep "noindex" dist/404/index.html 2>/dev/null || grep "noindex" dist/404.html 2>/dev/null
```

Expected: `<meta name="robots" content="noindex, nofollow">` present.

**Step 5: Commit**

```bash
git add docs/website/src/content/docs/404.md docs/website/public/_redirects docs/website/netlify.toml
git commit -m "feat(website): add 404 noindex, redirect infrastructure, security headers, and cache rules"
```

---

## Task 7: Technical SEO — Performance Hints and OG Type

**Files:**
- Modify: `docs/website/src/components/override-components/Head.astro`

**Step 1: Add preconnect for Google Fonts**

In `Head.astro`, before the `<AstroFont>` block, add:

```html
<!-- Preconnect to Google Fonts for faster font loading -->
<link rel="preconnect" href="https://fonts.googleapis.com" />
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin />
```

**Step 2: Add og:type differentiation**

After the existing `og:image` meta tags, add:

```html
<!-- OG type: website for homepage, article for doc pages -->
<meta property="og:type" content={isSplash ? "website" : "article"} />
```

**Step 3: Build and verify**

```bash
cd docs/website && yarn build
grep "preconnect" dist/index.html | head -3
grep "og:type" dist/index.html
grep "og:type" dist/getting-started/overview/index.html
```

Expected: preconnect links present, og:type is "website" on homepage, "article" on doc pages.

**Step 4: Commit**

```bash
git add docs/website/src/components/override-components/Head.astro
git commit -m "feat(website): add font preconnect hints and og:type differentiation"
```

---

## Task 8: Final Verification

**Files:** None (verification only)

**Step 1: Full build**

```bash
cd docs/website && yarn build
```

Expected: Clean build with no errors.

**Step 2: Verify OG images generated**

```bash
ls docs/website/dist/og/ | wc -l
ls docs/website/dist/og/ | head -10
```

Expected: One PNG per documentation page.

**Step 3: Spot-check homepage HTML**

```bash
grep -E "(og:image|og:type|twitter:card|application/ld\+json|preconnect|nosniff)" docs/website/dist/index.html
```

Expected: All SEO elements present.

**Step 4: Spot-check a doc page**

```bash
grep -E "(og:image|og:type|twitter:card|application/ld\+json)" docs/website/dist/getting-started/overview/index.html
```

Expected: Dynamic OG image URL, article og:type, structured data blocks.

**Step 5: Spot-check 404 page**

```bash
grep "noindex" docs/website/dist/404/index.html 2>/dev/null || grep "noindex" dist/404.html
```

Expected: noindex meta tag present.

**Step 6: Preview the site locally**

```bash
cd docs/website && yarn preview
```

Open `http://localhost:4321` and verify pages load. Check an OG image URL like `http://localhost:4321/og/getting-started/overview.png` renders a branded PNG.

**Step 7: Commit any final fixes if needed, then confirm done**

If all checks pass, the implementation is complete.
