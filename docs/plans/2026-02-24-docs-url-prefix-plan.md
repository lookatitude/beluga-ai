# Docs URL Prefix — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Move all documentation pages under `/docs/*` while keeping the homepage at `/`, enabling future non-docs pages at the root.

**Architecture:** Starlight maps `src/content/docs/` to URLs. Moving content directories into `src/content/docs/docs/` makes them serve from `/docs/*`. The homepage (`index.mdx`) and 404 stay at root. All sidebar slugs, component path checks, and internal links get the `/docs/` prefix.

**Tech Stack:** Astro + Starlight, JSON config, Markdown/MDX content

---

### Task 1: Move content directories into `docs/` subdirectory

**Files:**
- Create: `docs/website/src/content/docs/docs/` (directory)
- Move: 11 content directories into it

**Step 1: Create the target directory and move all content directories**

```bash
cd docs/website/src/content/docs
mkdir -p docs
mv getting-started guides tutorials api-reference providers architecture integrations contributing cookbook use-cases reports docs/
```

This moves all 11 directories. `index.mdx` and `404.md` stay in place.

**Step 2: Verify file structure**

Run:
```bash
ls docs/website/src/content/docs/
```
Expected: `404.md  docs/  index.mdx`

Run:
```bash
ls docs/website/src/content/docs/docs/
```
Expected: All 11 directories listed.

**Step 3: Commit**

```bash
git add -A docs/website/src/content/docs/
git commit -m "refactor(website): move content directories under docs/ subdirectory"
```

---

### Task 2: Update sidebar.json — prefix all slugs and directories

**Files:**
- Modify: `docs/website/src/config/sidebar.json`

**Step 1: Prefix all `slug` values with `docs/`**

Every `"slug": "X"` entry becomes `"slug": "docs/X"`. The entries to update:

- `"slug": "getting-started/overview"` → `"slug": "docs/getting-started/overview"`
- `"slug": "getting-started/installation"` → `"slug": "docs/getting-started/installation"`
- `"slug": "getting-started/quick-start"` → `"slug": "docs/getting-started/quick-start"`
- `"slug": "guides"` → `"slug": "docs/guides"`
- `"slug": "tutorials"` → `"slug": "docs/tutorials"`
- `"slug": "cookbook"` → `"slug": "docs/cookbook"`
- `"slug": "integrations"` → `"slug": "docs/integrations"`
- `"slug": "use-cases"` → `"slug": "docs/use-cases"`
- `"slug": "architecture"` → `"slug": "docs/architecture"`
- `"slug": "architecture/concepts"` → `"slug": "docs/architecture/concepts"`
- `"slug": "architecture/packages"` → `"slug": "docs/architecture/packages"`
- `"slug": "architecture/architecture"` → `"slug": "docs/architecture/architecture"`
- `"slug": "architecture/providers"` → `"slug": "docs/architecture/providers"`
- `"slug": "providers"` → `"slug": "docs/providers"`
- `"slug": "api-reference"` → `"slug": "docs/api-reference"`
- `"slug": "reports"` → `"slug": "docs/reports"`
- `"slug": "reports/changelog"` → `"slug": "docs/reports/changelog"`
- `"slug": "reports/security"` → `"slug": "docs/reports/security"`
- `"slug": "reports/code-quality"` → `"slug": "docs/reports/code-quality"`
- `"slug": "contributing"` → `"slug": "docs/contributing"`
- `"slug": "contributing/development-setup"` → `"slug": "docs/contributing/development-setup"`
- `"slug": "contributing/code-style"` → `"slug": "docs/contributing/code-style"`
- `"slug": "contributing/testing"` → `"slug": "docs/contributing/testing"`
- `"slug": "contributing/pull-requests"` → `"slug": "docs/contributing/pull-requests"`
- `"slug": "contributing/releases"` → `"slug": "docs/contributing/releases"`

**Step 2: Prefix all `autogenerate.directory` values with `docs/`**

Every `"directory": "X"` entry becomes `"directory": "docs/X"`. The entries to update:

- `"directory": "guides/foundations"` → `"directory": "docs/guides/foundations"`
- `"directory": "guides/capabilities"` → `"directory": "docs/guides/capabilities"`
- `"directory": "guides/production"` → `"directory": "docs/guides/production"`
- `"directory": "tutorials/foundation"` → `"directory": "docs/tutorials/foundation"`
- `"directory": "tutorials/providers"` → `"directory": "docs/tutorials/providers"`
- `"directory": "tutorials/agents"` → `"directory": "docs/tutorials/agents"`
- `"directory": "tutorials/memory"` → `"directory": "docs/tutorials/memory"`
- `"directory": "tutorials/rag"` → `"directory": "docs/tutorials/rag"`
- `"directory": "tutorials/orchestration"` → `"directory": "docs/tutorials/orchestration"`
- `"directory": "tutorials/safety"` → `"directory": "docs/tutorials/safety"`
- `"directory": "tutorials/server"` → `"directory": "docs/tutorials/server"`
- `"directory": "tutorials/messaging"` → `"directory": "docs/tutorials/messaging"`
- `"directory": "tutorials/multimodal"` → `"directory": "docs/tutorials/multimodal"`
- `"directory": "tutorials/documents"` → `"directory": "docs/tutorials/documents"`
- `"directory": "tutorials/voice"` → `"directory": "docs/tutorials/voice"`
- `"directory": "cookbook/agents"` → `"directory": "docs/cookbook/agents"`
- `"directory": "cookbook/llm"` → `"directory": "docs/cookbook/llm"`
- `"directory": "cookbook/rag"` → `"directory": "docs/cookbook/rag"`
- `"directory": "cookbook/memory"` → `"directory": "docs/cookbook/memory"`
- `"directory": "cookbook/voice"` → `"directory": "docs/cookbook/voice"`
- `"directory": "cookbook/multimodal"` → `"directory": "docs/cookbook/multimodal"`
- `"directory": "cookbook/prompts"` → `"directory": "docs/cookbook/prompts"`
- `"directory": "cookbook/infrastructure"` → `"directory": "docs/cookbook/infrastructure"`
- `"directory": "integrations/agents"` → `"directory": "docs/integrations/agents"`
- `"directory": "integrations/llm"` → `"directory": "docs/integrations/llm"`
- `"directory": "integrations/embeddings"` → `"directory": "docs/integrations/embeddings"`
- `"directory": "integrations/data"` → `"directory": "docs/integrations/data"`
- `"directory": "integrations/voice"` → `"directory": "docs/integrations/voice"`
- `"directory": "integrations/infrastructure"` → `"directory": "docs/integrations/infrastructure"`
- `"directory": "integrations/observability"` → `"directory": "docs/integrations/observability"`
- `"directory": "integrations/messaging"` → `"directory": "docs/integrations/messaging"`
- `"directory": "integrations/prompts"` → `"directory": "docs/integrations/prompts"`
- `"directory": "integrations/safety"` → `"directory": "docs/integrations/safety"`
- `"directory": "use-cases/search"` → `"directory": "docs/use-cases/search"`
- `"directory": "use-cases/agents"` → `"directory": "docs/use-cases/agents"`
- `"directory": "use-cases/voice"` → `"directory": "docs/use-cases/voice"`
- `"directory": "use-cases/documents"` → `"directory": "docs/use-cases/documents"`
- `"directory": "use-cases/analytics"` → `"directory": "docs/use-cases/analytics"`
- `"directory": "use-cases/safety"` → `"directory": "docs/use-cases/safety"`
- `"directory": "use-cases/infrastructure"` → `"directory": "docs/use-cases/infrastructure"`
- `"directory": "use-cases/messaging"` → `"directory": "docs/use-cases/messaging"`
- `"directory": "providers/llm"` → `"directory": "docs/providers/llm"`
- `"directory": "providers/embedding"` → `"directory": "docs/providers/embedding"`
- `"directory": "providers/vectorstore"` → `"directory": "docs/providers/vectorstore"`
- `"directory": "providers/voice"` → `"directory": "docs/providers/voice"`
- `"directory": "providers/loader"` → `"directory": "docs/providers/loader"`
- `"directory": "providers/guard"` → `"directory": "docs/providers/guard"`
- `"directory": "providers/eval"` → `"directory": "docs/providers/eval"`
- `"directory": "providers/observability"` → `"directory": "docs/providers/observability"`
- `"directory": "providers/workflow"` → `"directory": "docs/providers/workflow"`
- `"directory": "providers/vad"` → `"directory": "docs/providers/vad"`
- `"directory": "providers/transport"` → `"directory": "docs/providers/transport"`
- `"directory": "providers/cache"` → `"directory": "docs/providers/cache"`
- `"directory": "providers/state"` → `"directory": "docs/providers/state"`
- `"directory": "providers/prompt"` → `"directory": "docs/providers/prompt"`
- `"directory": "providers/mcp"` → `"directory": "docs/providers/mcp"`
- `"directory": "api-reference/foundation"` → `"directory": "docs/api-reference/foundation"`
- `"directory": "api-reference/llm-agents"` → `"directory": "docs/api-reference/llm-agents"`
- `"directory": "api-reference/memory-rag"` → `"directory": "docs/api-reference/memory-rag"`
- `"directory": "api-reference/voice"` → `"directory": "docs/api-reference/voice"`
- `"directory": "api-reference/infrastructure"` → `"directory": "docs/api-reference/infrastructure"`
- `"directory": "api-reference/protocol"` → `"directory": "docs/api-reference/protocol"`

**Approach:** Use two `sed` commands for reliable bulk replacement:

```bash
cd docs/website
# Prefix all slug values
sed -i 's/"slug": "/"slug": "docs\//g' src/config/sidebar.json

# Prefix all directory values
sed -i 's/"directory": "/"directory": "docs\//g' src/config/sidebar.json
```

**Step 3: Verify the file is valid JSON**

Run:
```bash
cd docs/website && node -e "JSON.parse(require('fs').readFileSync('src/config/sidebar.json','utf8')); console.log('Valid JSON')"
```
Expected: `Valid JSON`

**Step 4: Spot-check a few entries**

Run:
```bash
grep -c '"docs/' docs/website/src/config/sidebar.json
```
Expected: ~80+ occurrences (25 slugs + 57 directories).

**Step 5: Commit**

```bash
git add docs/website/src/config/sidebar.json
git commit -m "refactor(website): prefix all sidebar slugs and directories with docs/"
```

---

### Task 3: Update homepage links in index.mdx

**Files:**
- Modify: `docs/website/src/content/docs/index.mdx`

**Step 1: Update hero action link**

Line 16:
```
link: /getting-started/overview/
```
→
```
link: /docs/getting-started/overview/
```

**Step 2: Update LinkButton hrefs**

Line 86:
```html
<LinkButton href="/architecture/packages/" ...>View packages</LinkButton>
```
→
```html
<LinkButton href="/docs/architecture/packages/" ...>View packages</LinkButton>
```

Line 97:
```html
<LinkButton href="/providers/" ...>View providers</LinkButton>
```
→
```html
<LinkButton href="/docs/providers/" ...>View providers</LinkButton>
```

Line 108:
```html
<LinkButton href="/architecture/architecture/" ...>View architecture</LinkButton>
```
→
```html
<LinkButton href="/docs/architecture/architecture/" ...>View architecture</LinkButton>
```

**Step 3: Verify build**

Run:
```bash
cd docs/website && yarn build 2>&1 | tail -5
```
Expected: Build succeeds.

**Step 4: Commit**

```bash
git add docs/website/src/content/docs/index.mdx
git commit -m "refactor(website): update homepage links with /docs/ prefix"
```

---

### Task 4: Update Head.astro section filter paths

**Files:**
- Modify: `docs/website/src/components/override-components/Head.astro:55-68`

**Step 1: Update the `getPageSection()` function and `isApiRef` check**

Old (line 55):
```typescript
const isApiRef = pathname.startsWith("/api-reference/");
```

New:
```typescript
const isApiRef = pathname.startsWith("/docs/api-reference/");
```

Old (lines 59-68):
```typescript
function getPageSection(): string {
  if (isSplash) return "Home";
  if (pathname.startsWith("/getting-started/")) return "Getting Started";
  if (pathname.startsWith("/guides/")) return "Guides";
  if (pathname.startsWith("/tutorials/")) return "Tutorials";
  if (pathname.startsWith("/api-reference/")) return "API Reference";
  if (pathname.startsWith("/providers/")) return "Providers";
  if (pathname.startsWith("/architecture/")) return "Architecture";
  return "Docs";
}
```

New:
```typescript
function getPageSection(): string {
  if (isSplash) return "Home";
  if (pathname.startsWith("/docs/getting-started/")) return "Getting Started";
  if (pathname.startsWith("/docs/guides/")) return "Guides";
  if (pathname.startsWith("/docs/tutorials/")) return "Tutorials";
  if (pathname.startsWith("/docs/api-reference/")) return "API Reference";
  if (pathname.startsWith("/docs/providers/")) return "Providers";
  if (pathname.startsWith("/docs/architecture/")) return "Architecture";
  return "Docs";
}
```

**Step 2: Verify build**

Run:
```bash
cd docs/website && yarn build 2>&1 | tail -5
```
Expected: Build succeeds.

**Step 3: Commit**

```bash
git add docs/website/src/components/override-components/Head.astro
git commit -m "refactor(website): update Head.astro section filter paths with /docs/ prefix"
```

---

### Task 5: Update SearchModal.tsx QUICK_LINKS

**Files:**
- Modify: `docs/website/src/components/search/SearchModal.tsx:6-12`

**Step 1: Update all QUICK_LINKS URLs**

Old:
```typescript
const QUICK_LINKS = [
  { label: "Getting Started", url: "/getting-started/overview/" },
  { label: "API Reference", url: "/api-reference/" },
  { label: "Providers", url: "/providers/" },
  { label: "Architecture", url: "/architecture/" },
  { label: "Tutorials", url: "/tutorials/" },
];
```

New:
```typescript
const QUICK_LINKS = [
  { label: "Getting Started", url: "/docs/getting-started/overview/" },
  { label: "API Reference", url: "/docs/api-reference/" },
  { label: "Providers", url: "/docs/providers/" },
  { label: "Architecture", url: "/docs/architecture/" },
  { label: "Tutorials", url: "/docs/tutorials/" },
];
```

**Step 2: Verify build**

Run:
```bash
cd docs/website && yarn build 2>&1 | tail -5
```
Expected: Build succeeds.

**Step 3: Commit**

```bash
git add docs/website/src/components/search/SearchModal.tsx
git commit -m "refactor(website): update search QUICK_LINKS with /docs/ prefix"
```

---

### Task 6: Update internal links in all MD/MDX content files

**Files:**
- Modify: All `.md` and `.mdx` files under `docs/website/src/content/docs/docs/` that contain root-relative links

**Step 1: Find all root-relative links in content files**

Run:
```bash
cd docs/website/src/content/docs/docs
grep -rn '\](/[a-z]' --include='*.md' --include='*.mdx' | wc -l
```

This tells us how many links need updating.

**Step 2: Bulk update root-relative markdown links**

All links like `](/getting-started/...)`, `](/guides/...)`, `](/tutorials/...)`, `](/api-reference/...)`, `](/providers/...)`, `](/architecture/...)`, `](/integrations/...)`, `](/contributing/...)`, `](/cookbook/...)`, `](/use-cases/...)`, `](/reports/...)` need `/docs/` prefix.

Run:
```bash
cd docs/website/src/content/docs/docs
# Update markdown-style links: ](/section/...) → ](/docs/section/...)
find . -name '*.md' -o -name '*.mdx' | xargs sed -i \
  -e 's|](/getting-started/|](/docs/getting-started/|g' \
  -e 's|](/guides/|](/docs/guides/|g' \
  -e 's|](/tutorials/|](/docs/tutorials/|g' \
  -e 's|](/api-reference/|](/docs/api-reference/|g' \
  -e 's|](/providers/|](/docs/providers/|g' \
  -e 's|](/architecture/|](/docs/architecture/|g' \
  -e 's|](/integrations/|](/docs/integrations/|g' \
  -e 's|](/contributing/|](/docs/contributing/|g' \
  -e 's|](/cookbook/|](/docs/cookbook/|g' \
  -e 's|](/use-cases/|](/docs/use-cases/|g' \
  -e 's|](/reports/|](/docs/reports/|g'
```

**Step 3: Also update `href="/"` style links in MDX files**

Run:
```bash
cd docs/website/src/content/docs/docs
find . -name '*.mdx' | xargs sed -i \
  -e 's|href="/getting-started/|href="/docs/getting-started/|g' \
  -e 's|href="/guides/|href="/docs/guides/|g' \
  -e 's|href="/tutorials/|href="/docs/tutorials/|g' \
  -e 's|href="/api-reference/|href="/docs/api-reference/|g' \
  -e 's|href="/providers/|href="/docs/providers/|g' \
  -e 's|href="/architecture/|href="/docs/architecture/|g' \
  -e 's|href="/integrations/|href="/docs/integrations/|g' \
  -e 's|href="/contributing/|href="/docs/contributing/|g' \
  -e 's|href="/cookbook/|href="/docs/cookbook/|g' \
  -e 's|href="/use-cases/|href="/docs/use-cases/|g' \
  -e 's|href="/reports/|href="/docs/reports/|g'
```

**Step 4: Verify no old links remain**

Run:
```bash
cd docs/website/src/content/docs/docs
grep -rn '](/getting-started/\|](/guides/\|](/tutorials/\|](/api-reference/\|](/providers/\|](/architecture/\|](/integrations/\|](/contributing/\|](/cookbook/\|](/use-cases/\|](/reports/' --include='*.md' --include='*.mdx' | grep -v '/docs/' | wc -l
```
Expected: `0`

**Step 5: Verify build**

Run:
```bash
cd docs/website && yarn build 2>&1 | tail -5
```
Expected: Build succeeds.

**Step 6: Commit**

```bash
git add docs/website/src/content/docs/docs/
git commit -m "refactor(website): update all internal links in content with /docs/ prefix"
```

---

### Task 7: Build verification and smoke test

**Files:** None (verification only)

**Step 1: Full build**

Run:
```bash
cd docs/website && yarn build
```
Expected: Clean build, no errors.

**Step 2: Verify URL structure**

Run:
```bash
# Check that docs pages exist under /docs/
ls docs/website/dist/docs/getting-started/overview/index.html
ls docs/website/dist/docs/guides/index.html
ls docs/website/dist/docs/api-reference/index.html

# Check that homepage is at root
ls docs/website/dist/index.html

# Check that old paths DON'T exist
ls docs/website/dist/getting-started/ 2>&1
```
Expected: First 4 succeed, last one fails with "No such file or directory".

**Step 3: Preview and verify key pages**

Run:
```bash
cd docs/website && yarn preview --port 4327
```

Manual checklist:
1. Homepage at `http://localhost:4327/` loads correctly
2. "Get Started" button navigates to `/docs/getting-started/overview/`
3. Sidebar navigation works — all links go to `/docs/*` paths
4. Search returns results and quick links point to `/docs/*`
5. Section filter pills work correctly
6. "View packages", "View providers", "View architecture" links on homepage work
7. Internal links within doc pages work (e.g., cross-references in integration pages)

---

## Summary

| Task | Description | Dependencies |
|------|-------------|-------------|
| 1 | Move content directories into `docs/` subdirectory | None |
| 2 | Update sidebar.json slugs and directories | Task 1 |
| 3 | Update homepage links in index.mdx | None |
| 4 | Update Head.astro section filter paths | None |
| 5 | Update SearchModal.tsx QUICK_LINKS | None |
| 6 | Update internal links in all MD/MDX files | Task 1 |
| 7 | Build verification and smoke test | Tasks 1-6 |

Tasks 3, 4, 5 are independent of each other. Tasks 2 and 6 depend on Task 1 (files must be moved first). Task 7 depends on all.
