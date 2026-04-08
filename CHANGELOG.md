# Changelog

All notable changes to Beluga AI are documented here.

## [2.4.0] - 2026-04-07

### Bug Fixes

- **ci**: Release workflow not triggered after auto-tagging (#185)

### Features

- Self-improving agent team infrastructure (#187)

## [2.3.0] - 2026-04-07

### Features

- Implement all stubs, restructure agent team and workflow (#184)

## [2.2.5] - 2026-04-07

### Bug Fixes

- SonarCloud integration and issue resolution (#183)

## [2.2.4] - 2026-04-07

### Build

- **deps**: Bump actions/deploy-pages from 4 to 5 (#175)
- **deps**: Bump actions/upload-artifact from 4 to 7 (#160)
- **deps**: Bump securego/gosec from 2.23.0 to 2.25.0 (#172)
- **deps**: Bump the aws-sdk group with 4 updates (#178)
- **deps**: Bump the otel group with 5 updates (#179)
- **deps**: Bump the google group with 2 updates (#180)
- **deps**: Bump github.com/mattn/go-sqlite3 (#181)

## [2.2.3] - 2026-04-05

### Build

- **deps**: Bump devalue from 5.6.2 to 5.6.3 in /docs/website (#156)
- **deps**: Bump securego/gosec from 2.21.4 to 2.23.0 (#154)
- **deps**: Bump goreleaser/goreleaser-action from 6 to 7 (#155)
- **deps**: Bump actions/download-artifact from 4 to 8 (#159)
- **deps**: Bump marked from 17.0.3 to 17.0.4 in /docs/website (#164)
- **deps**: Bump satori from 0.19.2 to 0.25.0 in /docs/website (#165)
- **deps**: Bump @astrojs/sitemap from 3.7.0 to 3.7.1 in /docs/website (#168)
- **deps**: Bump astro from 5.18.0 to 5.18.1 in /docs/website (#169)
- **deps**: Bump the google group across 1 directory with 2 updates (#143)
- **deps**: Bump the otel group with 5 updates (#161)
- **deps**: Bump the aws-sdk group with 4 updates (#167)
- **deps**: Bump @astrojs/starlight in /docs/website (#170)
- **deps**: Bump the go-minor-patch group across 1 directory with 15 updates (#176)

## [2.2.2] - 2026-02-26

### Bug Fixes

- **website**: Regenerate yarn.lock for yarn classic v1 (#152)

### Security

- Add govulncheck, gosec, and Greptile to CI/CD pipeline (#153)

## [2.2.1] - 2026-02-26

### Bug Fixes

- Resolve SonarQube findings and improve website SEO (#151)

## [2.2.0] - 2026-02-24

### Bug Fixes

- **website**: Add yarnrc for node-modules linker and update gitignore
- **website**: Address SEO review findings
- **website**: Correct domain to beluga-ai.org and remove Netlify artifacts
- **website**: Center hero text, constrain tagline width
- **website**: Center section descriptions, fix light-text inline display
- **website**: Align stats row badges vertically with items-end
- **website**: Update navigation, footer, and CTA links with /docs/ prefix
- **website**: Remove placeholder contributors section and update roadmap

### Documentation

- Add enhanced search modal implementation plan
- Add gradient colors and homepage layout design
- Add gradient and homepage layout implementation plan
- Add docs URL prefix design
- Add implementation plan for docs URL prefix restructuring

### Features

- **website**: Add satori and resvg-js for OG image generation
- **website**: Add OG image rendering module with satori
- **website**: Add OG image generation endpoint for all doc pages
- **website**: Dynamic OG images with frontmatter override and proper meta tags
- **website**: Add comprehensive structured data (WebSite, Organization, SoftwareApplication, HowTo)
- **website**: Add 404 noindex, redirect infrastructure, security headers, and cache rules
- **website**: Add font preconnect hints and og:type differentiation
- **website**: Add React integration for custom search modal
- **website**: Add Pagefind section filter meta tag
- **website**: Add Pagefind API hook for custom search
- **website**: Add keyboard navigation hook for search
- **website**: Add recent searches localStorage hook
- **website**: Create SearchModal React component
- **website**: Replace Pagefind UI with custom React search modal
- **website**: Update primary color to teal (#5CA3CA) matching logo
- **website**: Update gradient to navy-teal-cyan palette
- **website**: Update logo glow and shadow to teal
- **website**: Update OG image to teal accent
- **website**: Add marketing pages, fix images/SEO, and unify navigation
- **website**: Unify docs and marketing code block styling

### Refactoring

- **website**: Move content directories under docs/ subdirectory
- **website**: Update homepage links with /docs/ prefix
- **website**: Prefix all sidebar slugs and directories with docs/
- **website**: Update Head.astro section filter paths with /docs/ prefix
- **website**: Update search QUICK_LINKS with /docs/ prefix
- **website**: Update all internal links in content with /docs/ prefix

### Build

- **deps**: Bump actions/github-script from 7 to 8
- **deps**: Bump peter-evans/create-pull-request from 7 to 8
- **deps**: Bump actions/stale from 9 to 10
- **deps**: Bump diff from 5.2.2 to 8.0.3 in /docs/website
- **deps**: Bump astro-embed from 0.9.2 to 0.12.0 in /docs/website
- **deps**: Bump marked from 17.0.1 to 17.0.2 in /docs/website
- **deps**: Bump astro from 5.17.1 to 5.17.2 in /docs/website
- **deps**: Bump astro from 5.17.2 to 5.17.3 in /docs/website
- **deps**: Bump marked from 17.0.2 to 17.0.3 in /docs/website
- **deps-dev**: Bump tailwindcss from 4.1.18 to 4.2.0 in /docs/website
- **deps**: Bump go.temporal.io/sdk from 1.39.0 to 1.40.0 (#134)
- **deps**: Bump the aws-sdk group with 4 updates (#144)
- **deps**: Bump github.com/anthropics/anthropic-sdk-go (#149)
- **deps**: Bump github.com/mattn/go-sqlite3 from 1.14.33 to 1.14.34 (#136)
- **deps**: Bump @tailwindcss/vite in /docs/website (#141)
- **deps**: Bump mermaid from 11.12.2 to 11.12.3 in /docs/website (#142)
- **deps**: Bump github.com/a2aproject/a2a-go from 0.3.6 to 0.3.7 (#145)
- **deps**: Bump github.com/nats-io/nats.go from 1.48.0 to 1.49.0 (#147)

## [2.1.1] - 2026-02-12

### Bug Fixes

- **ci**: Add semver auto-release from conventional commits and manual trigger

### Documentation

- Expand guides, cookbook, and tutorials with pattern reasoning and design context
- Expand providers, integrations, use-cases, getting-started, architecture, and contributing with pattern reasoning
- Update copyright year to 2026 and website URL
- Update logo assets and site config

### Features

- **website**: Modernize UI with new components, glassmorphism, and homepage redesign
- **website**: Comprehensive SEO optimization across 447 pages

### Testing

- Increase coverage to 90%+ across 155 packages (+909 tests)

## [2.1.0] - 2026-02-11

### Bug Fixes

- **ci**: Resolve Trivy vulnerabilities and CI workflow failures
- **security**: Resolve 7 Trivy vulnerabilities in Go and npm deps
- **security**: Resolve remaining Trivy findings in package-lock.json
- **ci**: Pin create-pull-request to full SHA for supply chain security
- **ci**: Scope write permissions to job level per least privilege
- **docs**: Fix website build and godoc artifact upload
- **ci**: Update report automation to open PRs

### Documentation

- Improve README, migrate architecture docs to website

### Build

- **deps**: Bump modernc.org/sqlite from 1.44.3 to 1.45.0
- **deps**: Bump github.com/modelcontextprotocol/go-sdk
- **deps**: Bump google.golang.org/genai in the google group
- **deps**: Bump SonarSource/sonarqube-scan-action from 5 to 7
- **deps**: Bump stefanzweifel/git-auto-commit-action from 5 to 7
- **deps**: Bump actions/setup-node from 4 to 6
- **deps**: Bump actions/upload-pages-artifact from 3 to 4
- **deps**: Bump actions/github-script from 7 to 8
- **deps**: Bump @astrojs/starlight in /docs/website
- **deps-dev**: Bump astro-vtbot from 2.1.9 to 2.1.11 in /docs/website
- **deps**: Bump astro from 5.16.4 to 5.17.1 in /docs/website
- **deps**: Bump the aws-sdk group with 2 updates
- **deps**: Bump @tailwindcss/vite in /docs/website
- **deps**: Bump vite from 7.2.6 to 7.3.1 in /docs/website

## [2.0.0] - 2026-02-09

### Documentation

- Add comprehensive documentation site with 390 pages
- Generate API reference from godoc with cmd/docgen tool
- Restructure site navigation with sidebar filtering and categorized sections

### Features

- **ci**: Add complete GitHub Actions CI/CD pipeline and docs site

