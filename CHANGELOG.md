# Changelog

All notable changes to Beluga AI are documented here.

## [2.10.0] - 2026-04-19

### Bug Fixes

- **ci**: Align go.mod and release.yml to go 1.25.7 (CI runner) [LOO-142]
- **memory**: Register "inmemory" as a Memory via blank-import init [LOO-142]

### Documentation

- Document beluga CLI (Layer 7 app) and six subcommands
- **consultations**: Capture LOO-142 security + QA review artifacts

### Features

- **cli**: T2 — add cobra root + DisableFlagParsing adapters for init/dev/test/deploy [LOO-142]
- **cli**: T3 — migrate init/dev/test/deploy to native cobra RunE + pflag [LOO-142]
- **cli**: Add beluga version subcommand with ldflags+build-info fallback
- **cli**: Curated 7-provider blank imports for beluga CLI
- **cli**: Add beluga providers subcommand with --output json
- **o11y**: Add BootstrapFromEnv skeleton for S3+ OTel wiring

### Miscellaneous

- **release**: Update CHANGELOG.md for v2.9.1 [skip ci]
- **cli**: T1 — bump Go toolchain to 1.25.9 for DX-1 S1 [LOO-142]

### Build

- **goreleaser**: Enable 5-target binary builds, archives, and checksums

### Ci

- **release**: Add smoke-install job verifying go install @tag post-release

## [2.9.1] - 2026-04-19

### Miscellaneous

- **release**: Update CHANGELOG.md for v2.9.0 [skip ci]
- **.wiki**: Remove stale artifacts (VALIDATION_REPORT, competitor stubs, empty drafts dir)

### Refactoring

- **.wiki**: Split corrections.md by category (preserves C-NNN IDs)
- **.claude**: Consolidate branch discipline — branch-discipline.md is canonical; workflow.md + CLAUDE.md point to it

## [2.9.0] - 2026-04-18

### Bug Fixes

- **ci**: Handle merge commits in release filter + fix goreleaser checkout ref
- **claude**: Address framework review findings

### Documentation

- **claude**: Document Linear-integrated branch naming in branch-discipline
- **claude**: Architect can invoke /consult for specialist escape hatch (A2)
- **claude**: Developer-go can invoke /consult for specialist escape hatch (A2)
- Document /consult command in framework CLAUDE.md (A2)

### Features

- **ci**: Wire dispatch-examples into release workflow
- **claude**: Extend /develop with Linear MCP pre-flight and label-derived branch naming
- **claude**: Add /consult command for workspace specialist escape hatch (A2)

### Miscellaneous

- **release**: Update CHANGELOG.md for v2.8.7 [skip ci]

## [2.8.7] - 2026-04-16

### Miscellaneous

- **release**: Update CHANGELOG.md for v2.8.6 [skip ci]
- **claude**: Evict website and notion-marketing orphans

## [2.8.6] - 2026-04-15

### Documentation

- **claude**: Link framework to workspace knowledge layer

### Miscellaneous

- **release**: Update CHANGELOG.md for v2.8.5 [skip ci]

## [2.8.5] - 2026-04-14

### Miscellaneous

- **claude**: Loosen over-paranoid PreToolUse safety prompts
- **release**: Update CHANGELOG.md for v2.8.3 [skip ci]
- Split docs/website into its own repo
- **deps**: Bump the go-minor-patch group with 3 updates
- **deps**: Bump actions/github-script from 8 to 9
- **deps**: Bump peter-evans/repository-dispatch from 3 to 4
- **release**: Update CHANGELOG.md for v2.8.4 [skip ci]
- **release**: Update CHANGELOG.md for v2.8.5 [skip ci]

### Release

- Ship raw report bodies in docs-bundle

## [2.8.4] - 2026-04-14

### Documentation

- **wiki**: Capture C-006 docs-audit correction

## [2.8.3] - 2026-04-14

### Documentation

- **website**: Phase 10 — embed remaining diagrams + visual polish

### Miscellaneous

- **deps**: Bump the go-minor-patch group with 6 updates
- **sonar**: Fix UserPromptSubmit hook + allow sonar CLI
- Consolidate worktree
- **release**: Update CHANGELOG.md for v2.8.2 [skip ci]
- **deps**: Bump google.golang.org/genai in the google group

### Website

- Complete redesign — new IA, typography, marketing pages, docs cutover
- Flatten docs sidebar, clean visible .md citations, fix double chevron
- Fix sidebar group labels — too big + empty-hr look
- Consolidate single header on docs pages, harmonize sidebar density
- Align header logo + docs button with docs content rails
- Tighten hero rhythm and compact footer
- Phase 7 — content + visual + diagram consistency pass
- Phase 8 — utility migration, imagery, more diagrams
- Phase 9 — 22 canonical diagrams embedded, surface tokens applied
- Align marketing frames to consistent 88rem width
- True side-by-side split layout across marketing pages
- Revert home architecture to editorial, drop sticky header
- Use split layout consistently for narrow-content sections
- Restore title-on-top + horizontal grid pattern
- Unify typography to IBM Plex Sans + fluid type scale
- Frame mermaid diagrams in centered panel
- Route every font-size through the type scale tokens

## [2.8.2] - 2026-04-12

### Bug Fixes

- **docs**: Address 5 Greptile comments in DOC-13
- **docs**: Address Greptile review on doc.go files

### Documentation

- **arch**: Expand DOC-13 with agentic guards, memory safety, degradation
- Add missing doc.go for undocumented packages

### Miscellaneous

- **release**: Update CHANGELOG.md for v2.8.1 [skip ci]

## [2.8.1] - 2026-04-12

### Bug Fixes

- **docs**: Correct prompt examples — ApplyMiddleware type mismatch
- **docs**: Correct code examples in DOC-20 — use config.ProviderConfig

### Documentation

- Add feature-status page and reconcile README with main
- Amend feature-status accuracy and capture two doc-writer corrections
- Include full rewritten README with all planned-feature fixes applied
- **arch**: Correct dependency tables for 7 packages in DOC-18
- Correct Hooks field names in DOC-03 and hooks-lifecycle
- **reference**: Regenerate provider catalog from the tree
- Add honest competitive comparison page
- **arch**: Add DOC-21 — Human-in-the-Loop
- **arch**: Fix hitl dependency statement — core+o11y not schema
- Add CONTRIBUTING.md contributor on-ramp
- Add production-readiness framing to docs hub pages
- **arch**: Add DOC-19 — Prompt Management
- **arch**: Add DOC-20 — Evaluation Framework
- **guides**: Rewrite first-agent against the real API
- Add production checklist for enterprise deployment
- **arch**: Expand DOC-10 with advanced RAG retrievers and loaders
- **wiki**: Capture C-010 registry-vs-constructor footgun for advanced RAG retrievers
- **arch**: Expand DOC-09 with advanced memory sub-packages
- **arch**: Expand DOC-05/06 with cognitive architectures and missing strategies

### Miscellaneous

- **release**: Update CHANGELOG.md for v2.7.0 [skip ci]
- **release**: Update CHANGELOG.md for v2.8.0 [skip ci]

### Wiki

- Add C-009 (feature-presence-invariant) and index routing entries

## [2.8.0] - 2026-04-12

### Bug Fixes

- **eval/judge**: Address review feedback on rubric, batch, consistency
- **ci**: Truncate coverage float to integer for GITHUB_OUTPUT
- **ci**: Harden test-parse step against multi-line GITHUB_OUTPUT

### Features

- **eval/judge**: LLM-as-Judge evaluation with rubrics (LOO-39)

### Testing

- **eval/judge**: Add coverage for review-comment fixes

## [2.7.0] - 2026-04-12

### Bug Fixes

- **computeruse**: Address Greptile review comments
- **computeruse**: Extract duplicated guard error code literal

### Features

- **tool/computeruse**: Computer use and browser automation (LOO-29)

### Miscellaneous

- **release**: Update CHANGELOG.md for v2.6.0 [skip ci]

### Testing

- **computeruse**: Expand coverage to 95.6% to pass SonarCloud gate

## [2.6.0] - 2026-04-12

### Bug Fixes

- Address Greptile review for codeact agent
- **codeact**: Restore NewProcessExecutor body and add gosec annotation
- Address Greptile review and security findings
- **security**: Use #nosec G204 directive so GHAS honours the suppression
- Address Greptile review and security findings
- **playground**: Extract Content-Type constant to resolve Sonar duplication

### Features

- **agent**: Add CodeAct (code-as-action) agent pattern
- **cmd**: Add beluga CLI with init/dev/test/deploy subcommands (LOO-45)
- **server**: Add playground chat UI package (LOO-43)

### Miscellaneous

- **release**: Update CHANGELOG.md for v2.5.0 [skip ci]
- **release**: Update CHANGELOG.md for v2.5.1 [skip ci]

### Refactoring

- **codeact**: Reduce cognitive complexity for SonarCloud gate

### Testing

- **codeact**: Add coverage for planner, agent, and executor paths
- **cmd/beluga**: Raise coverage and harden go binary lookup
- **playground**: Raise coverage to 98.7% to pass SonarCloud gate

## [2.5.1] - 2026-04-11

### Miscellaneous

- **rules**: Enforce pre-commit security gate and branch-PR discipline

## [2.5.0] - 2026-04-11

### Bug Fixes

- **security**: Address gosec findings from CI (PR #248)
- **quality**: Address SonarCloud findings on arch-validate-sweep
- **security**: Address residual gosec findings on touched files
- **security**: Suppress G118 false positives in newWorkflowContext
- **quality**: Nudge coverage >=80%, cover tracing passthroughs, last gosec

### Documentation

- **specs**: Add unified agent system design
- **wiki**: Populate from codebase via /wiki-learn all
- **specs**: Add docs architecture redesign spec
- Replace monolithic docs with 29-document architecture set
- **wiki**: /arch-validate all — capture findings
- **architecture**: Arch-update for arch-validate sweep

### Features

- **.claude**: Unified self-evolving multi-agent system

### Refactoring

- **agent, voice/s2s**: Split 6-method interfaces via composition
- Add core.Errorf helper; migrate llm + FuncTool to typed errors
- **tool, guard**: Migrate error handling to core.Errorf
- **memory**: Fix arch-validate violations (errors, streaming, tracing)
- Arch-validate sweep across framework (errors + tracing)
- Chan->iter.Seq2 cross-package cascades (3 of 5)
- **voice**: FrameProcessor + Transport chan->iter.Seq2 cascade

## [2.4.3] - 2026-04-11

### Bug Fixes

- **website**: Bump react-dom to 19.2.5 to match react version (#246)

### LOO-10

- Agent Trajectory Evaluation (#200)

### LOO-11

- System 1/System 2 Dual-Process Cognitive Architecture (#201)

### LOO-12

- Metacognitive Agents (Self-Model Improvement) (#202)

### LOO-13

- Role-Based Dynamic Team Formation (#203)

### LOO-14

- Multi-Agent Debate and Generator-Evaluator Pattern (#204)

### LOO-15

- Shared Blackboard / Shared Context Layer (#205)

### LOO-16

- Semantic Contracts Between Agents (#206)

### LOO-17

- Procedural Memory (How-To Knowledge) (#207)

### LOO-18

- Memory Consolidation with Intentional Forgetting (#208)

### LOO-19

- Sleep-Time Compute (#209)

### LOO-20

- Cross-Agent Shared Memory with Access Control (#210)

### LOO-21

- RL-Optimized Memory Operations (Memory-R1) (#211)

### LOO-22

- Zettelkasten-Style Associative Memory (A-MEM) (#212)

### LOO-23

- Late Interaction Retrieval Models (ColBERT/ColPali) (#213)

### LOO-24

- RAPTOR Hierarchical Tree-Based Retrieval (#214)

### LOO-25

- Agentic RAG Orchestration (#215)

### LOO-26

- Structured Data RAG (Text2Cypher/Text2SQL) (#216)

### LOO-28

- Sandboxed Code Execution (E2B-Style) (#217)

### LOO-30

- MCP November 2025 Spec Features (#219)

### LOO-31

- Tool Learning and Creation by Agents (#220)

### LOO-32

- OWASP Top 10 for Agentic Applications Compliance (#221)

### LOO-33

- Memory Poisoning Detection and Cascading Failure Prevention (#222)

### LOO-34

- Temporal Least Privilege and Agentic Identity (#223)

### LOO-35

- Automated Red Teaming (#224)

### LOO-37

- Simulation-Based Testing Environments (#226)

### LOO-38

- Cost-Aware Evaluation (#227)

### LOO-40

- Agent Replay and Time-Travel Debugging (#229)

### LOO-41

- Conversation Clustering and Pattern Analysis (#230)

### LOO-42

- Agent Execution Visualization (#231)

### LOO-44

- Declarative Agent Definition (YAML/JSON) (#233)

### LOO-46

- Speaker Diarization (#235)

### LOO-47

- Telephony Integration (SIP/PSTN) (#236)

### LOO-48

- AG-UI and AGENTS.md (#237)

### LOO-49

- Intelligent Model Routing for Cost Optimization (#238)

### LOO-50

- Self-Evolving / Self-Improving Agents (#239)

### LOO-51

- Agent File Format / Portable Serialization (#240)

### LOO-52

- Context Engineering as Explicit Discipline (#241)

### LOO-55

- Mind-Map Agent / Structured Reasoning Context (#244)

### LOO-7

- Reasoning Model Integration (o3/o4, Claude thinking) (#197)

### LOO-8

- Agentic Plan Caching (#198)

### LOO-9

- Temporal Knowledge Graph Memory (#199)

### Security

- Graceful Degradation Under Security Events (#225)

## [2.4.2] - 2026-04-11

### Documentation

- Fix architecture, concepts, packages, and API reference to match current codebase
- Tier 2 sweep — fix code examples across all documentation pages

### LOO-53

- Speculative Execution for Agents (#242)

### Miscellaneous

- **release**: Update CHANGELOG.md for v2.4.1 [skip ci] (#190)

### Build

- **deps**: Bump @tailwindcss/vite from 4.2.1 to 4.2.2 in /docs/website (#192)
- **deps**: Bump devalue from 5.6.3 to 5.7.1 in /docs/website (#195)
- **deps**: Bump react from 19.2.4 to 19.2.5 in /docs/website (#194)
- **deps**: Bump lodash-es from 4.17.23 to 4.18.1 in /docs/website (#196)

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

