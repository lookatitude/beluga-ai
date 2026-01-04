# Beluga AI Framework Website

This website is built using [Docusaurus 3](https://docusaurus.io/), a modern static website generator for documentation sites.

## Overview

The website provides comprehensive documentation for the Beluga AI Framework, including:
- Getting started guides and tutorials
- Core concepts and architecture
- API reference documentation
- Provider documentation (LLMs, Embeddings, Vector Stores)
- Cookbook with recipes and examples
- Use case examples
- Voice agents documentation

## Installation

Install dependencies using npm or yarn:

```bash
npm install
# or
yarn install
```

## Local Development

Start the local development server:

```bash
npm start
# or
yarn start
```

This command starts a local development server and opens up a browser window. Most changes are reflected live without having to restart the server.

The site will be available at `http://localhost:3000`.

## Build

Build the static site:

```bash
npm run build
# or
yarn build
```

This command generates static content into the `build` directory and can be served using any static contents hosting service.

## Serve Production Build

Preview the production build locally:

```bash
npm run serve
# or
yarn serve
```

## Deployment

Deploy to GitHub Pages:

```bash
GIT_USER=<Your GitHub username> USE_SSH=true npm run deploy
# or
GIT_USER=<Your GitHub username> USE_SSH=true yarn deploy
```

If you are using GitHub pages for hosting, this command is a convenient way to build the website and push to the `gh-pages` branch.

## Documentation Structure

- `docs/` - All documentation markdown files
- `blog/` - Blog posts (if enabled)
- `src/` - React components and custom pages
- `static/` - Static assets (images, files, etc.)
- `sidebars.js` - Sidebar navigation configuration
- `docusaurus.config.js` - Main Docusaurus configuration

## Updating Documentation

1. Edit markdown files in the `docs/` directory
2. Update `sidebars.js` if adding new sections
3. Test locally with `yarn start`
4. Build and deploy when ready

## Type Checking

Run TypeScript type checking:

```bash
npm run typecheck
# or
yarn typecheck
```
