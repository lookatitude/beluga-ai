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
      customCss: ["./src/styles/global.css"],
      expressiveCode: {
        themes: ['github-dark-default', 'github-light-default'],
        styleOverrides: {
          borderRadius: '1rem',
          borderColor: 'color-mix(in srgb, var(--sl-color-white) 8%, transparent)',
          codeFontFamily: '"SF Mono", "Fira Code", "Fira Mono", "Roboto Mono", monospace',
          codeFontSize: '0.8125rem',
          codeLineHeight: '1.75',
          codePaddingBlock: '1.25rem',
          codePaddingInline: '1.5rem',
          codeBackground: ({ theme }) =>
            theme.type === 'dark'
              ? 'color-mix(in srgb, #151515 95%, transparent)'
              : 'color-mix(in srgb, #f8f8f8 95%, white)',
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
