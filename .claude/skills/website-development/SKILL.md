---
name: website-development
description: Website development patterns for the Beluga AI v2 documentation site. Use when building, styling, or adjusting website components, layouts, pages, and interactive elements — NOT for writing documentation content.
---

# Website Development Patterns

## Tech Stack

- **Framework**: Astro 5 with Starlight documentation theme
- **Styling**: Tailwind CSS 4 (via `@tailwindcss/vite` plugin)
- **Components**: Astro components (`.astro`) — use React (`.tsx`) only when client-side interactivity is required
- **Build**: Vite 7, TypeScript 5
- **Extras**: astro-vtbot (view transitions), astro-embed, astro-font, Mermaid diagram rendering, Sharp for image optimization

## Project Structure

```
docs/website/
├── astro.config.mjs          # Astro + Starlight + Tailwind config
├── src/
│   ├── components/            # Shared Astro components
│   │   ├── override-components/  # Starlight component overrides
│   │   └── user-components/      # Custom reusable components
│   ├── config/                # Site config JSON files (sidebar, social, theme, locals)
│   ├── content/               # Markdown content (managed by doc-writer, NOT this persona)
│   ├── content.config.ts      # Content collection schema
│   ├── lib/                   # Utilities and rehype plugins
│   ├── styles/                # CSS files (global, base, components, navigation, button)
│   └── tailwind-plugin/       # Custom Tailwind plugins (grid, theme)
├── public/                    # Static assets (logos, favicons)
├── tsconfig.json
└── package.json
```

## Scope & Boundaries

### This persona handles:
- Astro component development (`.astro`, `.tsx`)
- Starlight theme customization and component overrides
- Tailwind CSS styling and custom plugins
- Page layouts, navigation, sidebar, header, footer
- Interactive UI elements (tabs, accordions, search, theme switching)
- Responsive design and accessibility
- Build configuration (Astro, Vite, Tailwind)
- Performance optimization (image handling, view transitions, bundle size)
- Custom rehype/remark plugins for content rendering

### This persona does NOT handle:
- Writing or editing Markdown documentation content — delegate to `doc-writer`
- Go framework code — delegate to appropriate developer personas
- Architecture decisions for the Go framework — delegate to `architect`

## Conventions

### Component Rules
- Astro components for static/server-rendered UI (default choice)
- React components only when `client:*` directives are needed (interactive widgets)
- Component files use PascalCase: `HeroTabs.astro`, `LinkButton.astro`
- Override components mirror Starlight's naming in `override-components/`
- User-facing reusable components go in `user-components/`

### Styling Rules
- Use Tailwind utility classes as the primary styling approach
- Custom CSS goes in `src/styles/` — split by concern (base, components, navigation, button)
- Custom Tailwind plugins in `src/tailwind-plugin/` for project-specific utilities
- Respect dark/light mode — always provide both variants
- Use CSS custom properties from theme config for brand colors

### Path Aliases
- `@/` and `~/` both resolve to `src/` (configured in `astro.config.mjs`)
- Use these aliases in all imports: `import X from "@/components/X.astro"`

### Starlight Overrides
- Override Starlight components by placing replacements in `src/components/override-components/`
- Register overrides in `astro.config.mjs` under `starlight({ components: { ... } })`
- Keep overrides minimal — extend rather than replace when possible

### Configuration
- Site config: `src/config/config.json`
- Sidebar: `src/config/sidebar.json`
- Social links: `src/config/social.json`
- Theme: `src/config/theme.json`
- Locales: `src/config/locals.json`
- Menus: `src/config/menu.{locale}.json`

## Development Commands

```bash
cd docs/website
yarn dev        # Start dev server
yarn build      # Production build
yarn preview    # Preview production build
```

## Don'ts

- Don't edit Markdown content files — that's the doc-writer's job
- Don't introduce new CSS frameworks or UI libraries without Architect approval
- Don't break Starlight's content collection schema
- Don't hardcode text strings — use config files or Starlight's i18n system
- Don't add client-side JavaScript when Astro's server-rendering suffices
- Don't commit `node_modules/` or build artifacts
