// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import react from "@astrojs/react";
import sitemap from "@astrojs/sitemap";
import { viewTransitions } from "astro-vtbot/starlight-view-transitions";
import rehypeMermaid from "./src/lib/rehype-mermaid.mjs";

import tailwindcss from "@tailwindcss/vite";
import config from "./src/config/config.json" assert { type: "json" };
import social from "./src/config/social.json";
import locals from "./src/config/locals.json";
import sidebar from "./src/config/sidebar.json";

import { fileURLToPath } from "url";

const { site } = config;
const { title } = site;

export const locales = locals


// https://astro.build/config
export default defineConfig({
  site: "https://beluga-ai.org",
  image: {
    service: { entrypoint: "astro/assets/services/noop" },
  },
  // Permanent redirects for the IA cutover. Marketing URLs from the
  // pre-rebuild site map to their new destinations. Docs redirects
  // land in a later phase when the content tree is migrated.
  redirects: {
    // ── Marketing redirects (Phase 3) ─────────────────────────
    "/about": "/community/",
    "/integrations": "/providers/",
    "/features": "/product/",
    "/features/agents": "/product/#build",
    "/features/llm": "/product/#build",
    "/features/tools": "/product/#build",
    "/features/orchestration": "/product/#build",
    "/features/rag": "/product/#know",
    "/features/memory": "/product/#know",
    "/features/voice": "/product/#voice",
    "/features/guardrails": "/product/#ship",
    "/features/observability": "/product/#ship",
    "/features/protocols": "/product/#protocols",

    // ── Docs IA cutover (Phase 4) ─────────────────────────────
    // getting-started → start
    "/docs/getting-started": "/docs/start/",
    "/docs/getting-started/overview": "/docs/start/",
    "/docs/getting-started/installation": "/docs/start/installation/",
    "/docs/getting-started/quick-start": "/docs/start/quick-start/",

    // cookbook → recipes
    "/docs/cookbook": "/docs/recipes/",

    // architecture split between concepts and reference
    "/docs/architecture": "/docs/reference/architecture/overview/",
    "/docs/architecture/concepts": "/docs/concepts/",
    "/docs/architecture/packages": "/docs/reference/architecture/packages/",
    "/docs/architecture/providers": "/docs/reference/architecture/providers/",
    "/docs/architecture/architecture": "/docs/reference/architecture/overview/",

    // providers → reference/providers
    "/docs/providers": "/docs/reference/providers/",

    // api-reference → reference/api
    "/docs/api-reference": "/docs/reference/api/",

    // reports → contributing/project-reports
    "/docs/reports": "/docs/contributing/",
    "/docs/reports/changelog": "/docs/contributing/project-reports/changelog/",
    "/docs/reports/security": "/docs/contributing/project-reports/security/",
    "/docs/reports/code-quality": "/docs/contributing/project-reports/code-quality/",

    // tutorials dissolved into guides
    "/docs/tutorials": "/docs/guides/",

    // use-cases dissolved into guides
    "/docs/use-cases": "/docs/guides/",

    // integrations consolidated under guides/production/integrations
    "/docs/integrations": "/docs/guides/production/integrations/",
  },
  markdown: {
    rehypePlugins: [rehypeMermaid],
  },
  integrations: [
    starlight({
      title,
      // @ts-ignore
      social: social.main || [],
      locales,
      sidebar: sidebar.main || [],
      // Use our marketing-layout /404 (src/pages/404.astro) instead of
      // Starlight's built-in one, so a 404 keeps the same chrome as
      // the rest of the site.
      disable404Route: true,
      customCss: ["./src/styles/global.css"],
      expressiveCode: {
        themes: ['github-dark-default', 'github-light-default'],
        styleOverrides: {
          borderRadius: '0.5rem',
          borderColor: 'var(--sl-color-hairline)',
          codeFontFamily: 'var(--font-mono)',
          codeFontSize: '0.8125rem',
          codeLineHeight: '1.7',
          codePaddingBlock: '1.25rem',
          codePaddingInline: '1.5rem',
          codeBackground: ({ theme }) =>
            theme.type === 'dark'
              ? 'color-mix(in oklch, var(--ink-900) 92%, var(--ink-950))'
              : 'var(--paper-100)',
          frames: {
            shadowColor: 'transparent',
          },
        },
      },
      components: {
        Head: "./src/components/override-components/Head.astro",
        Search: "./src/components/override-components/Search.astro",
        Header: "./src/components/override-components/Header.astro",
        Hero: "./src/components/override-components/Hero.astro",
        PageFrame: "./src/components/override-components/PageFrame.astro",
        PageSidebar: "./src/components/override-components/PageSidebar.astro",
        TwoColumnContent: "./src/components/override-components/TwoColumnContent.astro",
        ContentPanel: "./src/components/override-components/ContentPanel.astro",
        Pagination: "./src/components/override-components/Pagination.astro",
        Sidebar: "./src/components/override-components/Sidebar.astro",


      },

    }),
    react(),
    sitemap(),
  ],
  vite: {
    plugins: [tailwindcss(),viewTransitions()],
    resolve: {
      alias: {
        "@": fileURLToPath(new URL("./src", import.meta.url)),
        "~": fileURLToPath(new URL("./src", import.meta.url)),
      },
    },
  },
});
