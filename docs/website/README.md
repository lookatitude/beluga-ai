# DocKit

[![Built with Starlight](https://astro.badg.es/v2/built-with-starlight/tiny.svg)](https://starlight.astro.build)

A modern, feature-rich documentation theme built on top of **Astro Starlight** with custom components, enhanced UI, and multilingual support.

## 🚀 Step-by-Step Getting Started Guide

### Step 1: Clone and Install

First, clone the repository and install dependencies:

```bash
# Clone the repository
git clone https://github.com/yourusername/dockit-astro.git
cd dockit-astro

# Install dependencies
yarn install

# Start development server
yarn dev
```

Your site will be available at `http://localhost:4321`

### Step 2: Configure Your Site

Configure your site settings by editing the configuration files in `src/config/`:

#### Basic Site Configuration

Edit `src/config/config.json`:

```json
{
  "site": {
    "title": "Your Documentation Site",
    "description": "Your site description",
    "author": "Your Name",
    "email": "your.email@example.com",
    "base_url": "https://yourdomain.com"
  }
}
```

#### Theme Customization

Edit `src/config/theme.json`:

```json
{
  "theme": {
    "primary_color": "#2563eb"
  }
}
```

### Step 3: Add Your First Documentation Page

Create your first documentation page:

```bash
# Create a new markdown file
touch src/content/docs/getting-started.md
```

Add content to your file:

```markdown
---
title: Getting Started
description: Welcome to your documentation site
---

# Getting Started

Your first documentation page content goes here...
```

### Step 4: Configure Sidebar Navigation with Icons

Edit `src/config/sidebar.json` to add navigation with icons:

```json
{
  "main": [
    {
      "label": "[seti:vite] Getting Started",
      "translations": {
        "fr": "[seti:vite] Aan de slag"
      },
      "slug": "getting-started"
    },
    {
      "label": "[document] API Reference",
      "autogenerate": { "directory": "api" }
    }
  ]
}
```

**Supported Icon Formats:**

- `[seti:vite]` - Seti UI icons (vite, typescript, react, etc.)
- `[setting]` - Settings/configuration
- `[document]` - Documentation
- `[pencil]` - Editing/writing

### Step 5: Use Custom Components in Your Documentation

Create rich documentation using DocKit's custom components:

```mdx
---
title: Features Overview
---

import Grid from "~/components/user-components/Grid.astro";
import NewCard from "~/components/user-components/NewCard.astro";
import Button from "~/components/user-components/Button.astro";
import Accordion from "~/components/user-components/Accordion.astro";

# Features Overview

<Grid columns={3}>
  <NewCard title="Fast Setup" icon="rocket">
    Get started in minutes with our pre-configured setup
  </NewCard>

{" "}
<NewCard title="Custom Components" icon="document">
  Rich set of components for beautiful documentation
</NewCard>

  <NewCard title="Multilingual" icon="setting">
    Built-in support for multiple languages
  </NewCard>
</Grid>

<Accordion
  question="How do I add more pages?"
  answer="Simply create new .md or .mdx files in the src/content/docs/ directory"
/>

<Button
  label="View Full Documentation"
  link="/docs/components/using-components"
  variant="primary"
/>
```

### Step 6: Add Multilingual Support (Optional)

To add Dutch (or other language) translations:

1. **Create language-specific content:**

```bash
mkdir src/content/docs/de
touch src/content/docs/de/getting-started.md
```

2. **Add translated content:**

```markdown
---
title: Aan de slag
description: Welkom bij je documentatiesite
---

# Aan de slag

Je eerste documentatiepagina inhoud komt hier...
```

3. **Configure language settings in `src/config/locals.json`:**

```json
{
  "defaultLocale": "en",
  "locales": {
    "en": {
      "label": "English",
      "lang": "en"
    },
    "fr": {
      "label": "Français",
      "lang": "fr"
    }
  }
}
```

### Step 7: Customize Styling (Optional)

Add custom styles to `src/styles/global.css`:

```css
/* Custom theme variables */
:root {
  --custom-primary: #your-color;
  --custom-accent: #your-accent;
}

/* Custom component styles */
.custom-hero {
  background: linear-gradient(
    45deg,
    var(--custom-primary),
    var(--custom-accent)
  );
}
```

### Step 8: Build and Deploy

When ready to deploy:

```bash
# Build for production
yarn build

# Preview the build
yarn preview

# Deploy to your hosting platform
# (Netlify, Vercel, GitHub Pages, etc.)
```

---

## 📚 Available Components Reference

### Custom User Components

| Component         | Description                   | Example Usage                |
| ----------------- | ----------------------------- | ---------------------------- |
| `Accordion.astro` | Collapsible Q&A sections      | FAQ pages, help sections     |
| `Button.astro`    | Styled buttons with variants  | CTAs, navigation links       |
| `Grid.astro`      | Responsive grid layouts       | Organizing cards and content |
| `ListCard.astro`  | Cards with icons and counters | Feature listings, navigation |
| `NewCard.astro`   | Modern gradient cards         | Showcasing features          |

### Enhanced Starlight Overrides

| Component               | Enhancement                                   |
| ----------------------- | --------------------------------------------- |
| `Sidebar.astro`         | Custom icon support with `[icon-name]` syntax |
| `Header.astro`          | Improved mobile navigation and design         |
| `Footer.astro`          | Configurable footer sections                  |
| `Hero.astro`            | Enhanced hero styling and layout              |
| `TableOfContents.astro` | Better navigation and UX                      |

## 🎯 Icon Reference for Sidebar

### Use Starlight Built-in Icons in Sidebar (use `[icon-name]`)

- `[seti:vite]` - Vite
- `[seti:typescript]` - TypeScript
- `[seti:javascript]` - JavaScript
- `[seti:react]` - React
- `[seti:json]` - JSON files
- `[seti:config]` - Configuration
- `[seti:npm]` - NPM/packages

### Starlight Built-in Icons

- `[document]` - Documentation pages
- `[setting]` - Settings/configuration
- `[pencil]` - Editing/writing
- `[rocket]` - Getting started/launch
- `[github]` - GitHub integration

## 🌐 Project Structure

```
.
├── public/                     # Static assets
├── src/
│   ├── assets/                # Images and media
│   ├── components/
│   │   ├── override-components/    # Enhanced Starlight components
│   │   └── user-components/        # Custom DocKit components
│   ├── config/                # Configuration files
│   │   ├── config.json        # Site settings
│   │   ├── theme.json         # Theme customization
│   │   ├── sidebar.json       # Navigation with icons
│   │   ├── menu.en.json       # English menu
│   │   ├── menu.de.json       # Dutch menu
│   │   └── locals.json        # Language settings
│   ├── content/
│   │   ├── docs/              # English documentation
│   │   │   └── de/            # Dutch translations
│   │   └── sections/          # Page sections
│   └── styles/                # Custom CSS
├── astro.config.mjs           # Astro configuration
└── package.json
```

---

## � Advanced Usage

### Creating Custom Themes

1. **Modify theme configuration:**

```json
// src/config/theme.json
{
  "theme": {
    "primary_color": "#your-brand-color"
  }
}
```

2. **Add custom CSS:**

```css
/* src/styles/global.css */
:root {
  --sl-color-accent: #your-accent-color;
}
```

### Working with Images

1. **Add images to `src/assets/`:**

```
src/assets/
├── logo.svg
├── hero-image.png
└── screenshots/
    └── feature.jpg
```

2. **Reference in markdown:**

```markdown
![Alt text](../../../../assets/overview.png)
```

## 🧞 Commands

All commands are run from the root of the project, from a terminal:

| Command                | Action                                           |
| :--------------------- | :----------------------------------------------- |
| `yarn install`         | Installs dependencies                            |
| `yarn dev`             | Starts local dev server at `localhost:4321`      |
| `yarn build`           | Build your production site to `./dist/`          |
| `yarn preview`         | Preview your build locally, before deploying     |
| `yarn astro ...`       | Run CLI commands like `astro add`, `astro check` |
| `yarn astro -- --help` | Get help using the Astro CLI                     |

## 👀 Want to learn more?

Check out [Starlight’s docs](https://starlight.astro.build/), read [the Astro documentation](https://docs.astro.build), or jump into the [Astro Discord server](https://astro.build/chat).
