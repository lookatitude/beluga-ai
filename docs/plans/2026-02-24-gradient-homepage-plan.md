# Gradient Colors & Homepage Layout — Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Shift the site color system from purple-rose-gold to teal/cyan/navy (matching the logo) and fix homepage layout alignment issues.

**Architecture:** Change the CSS `--color-primary-gradient` variable and `theme.json` primary color, then update the Hero.astro glow/shadow and OG image accent. Fix homepage centering with CSS adjustments to Section, Grid, and Hero components.

**Tech Stack:** Astro, CSS custom properties, Tailwind CSS

---

### Task 1: Update theme.json primary color

**Files:**
- Modify: `docs/website/src/config/theme.json:5,16`

**Step 1: Change primary color in both dark and light modes**

In `docs/website/src/config/theme.json`, change line 5:

```json
"primary": "#5CA3CA",
```

And change line 16:

```json
"primary": "#5CA3CA",
```

Both dark mode and light mode `primary` values change from `#D76D77` to `#5CA3CA`.

**Step 2: Verify build**

Run:
```bash
cd docs/website && yarn build 2>&1 | tail -5
```
Expected: Build succeeds.

**Step 3: Commit**

```bash
git add docs/website/src/config/theme.json
git commit -m "feat(website): update primary color to teal (#5CA3CA) matching logo"
```

---

### Task 2: Update the primary gradient CSS variable

**Files:**
- Modify: `docs/website/src/styles/global.css:114-119`

**Step 1: Change the gradient**

In `docs/website/src/styles/global.css`, replace lines 114-119:

Old:
```css
@theme {
  --color-primary-gradient: linear-gradient(
    35.65deg,
    #3a1c71 -10.94%,
    var(--color-primary) 61.04%,
    #ffca7b 133.01%
  );
}
```

New:
```css
@theme {
  --color-primary-gradient: linear-gradient(
    35.65deg,
    #0F3353 -10.94%,
    var(--color-primary) 61.04%,
    #81D4E8 133.01%
  );
}
```

This changes: purple (#3a1c71) → navy (#0F3353), gold (#ffca7b) → bright cyan (#81D4E8). The middle stop already uses `var(--color-primary)` which now resolves to #5CA3CA from Task 1.

**Step 2: Verify build**

Run:
```bash
cd docs/website && yarn build 2>&1 | tail -5
```
Expected: Build succeeds.

**Step 3: Commit**

```bash
git add docs/website/src/styles/global.css
git commit -m "feat(website): update gradient to navy-teal-cyan palette"
```

---

### Task 3: Update Hero.astro logo glow and drop shadow

**Files:**
- Modify: `docs/website/src/components/override-components/Hero.astro:119-124,135`

**Step 1: Update the radial gradient glow**

In `docs/website/src/components/override-components/Hero.astro`, find the `.hero-logo::before` block (line 119-124). Replace the three color stops:

Old:
```css
background: radial-gradient(
  circle,
  rgba(215, 109, 119, 0.35) 0%,
  rgba(58, 28, 113, 0.15) 50%,
  transparent 70%
);
```

New:
```css
background: radial-gradient(
  circle,
  rgba(92, 163, 202, 0.35) 0%,
  rgba(15, 51, 83, 0.15) 50%,
  transparent 70%
);
```

**Step 2: Update the drop shadow**

On line 135, change the drop-shadow filter:

Old:
```css
filter: drop-shadow(0 0 30px rgba(215, 109, 119, 0.3));
```

New:
```css
filter: drop-shadow(0 0 30px rgba(92, 163, 202, 0.3));
```

**Step 3: Verify build**

Run:
```bash
cd docs/website && yarn build 2>&1 | tail -5
```
Expected: Build succeeds.

**Step 4: Commit**

```bash
git add docs/website/src/components/override-components/Hero.astro
git commit -m "feat(website): update logo glow and shadow to teal"
```

---

### Task 4: Update OG image accent color

**Files:**
- Modify: `docs/website/src/lib/og-image.ts:28,99`

**Step 1: Update the background gradient**

In `docs/website/src/lib/og-image.ts`, line 28, change the background:

Old:
```js
background: "linear-gradient(135deg, #0d0d0d 0%, #1a1a2e 50%, #151515 100%)",
```

New:
```js
background: "linear-gradient(135deg, #0d0d0d 0%, #0F2535 50%, #151515 100%)",
```

**Step 2: Update the accent square color**

On line 99, change the logo accent square:

Old:
```js
background: "#D76D77",
```

New:
```js
background: "#5CA3CA",
```

**Step 3: Verify build**

Run:
```bash
cd docs/website && yarn build 2>&1 | tail -5
```
Expected: Build succeeds.

**Step 4: Commit**

```bash
git add docs/website/src/lib/og-image.ts
git commit -m "feat(website): update OG image to teal accent"
```

---

### Task 5: Fix hero text centering and tagline layout

**Files:**
- Modify: `docs/website/src/components/override-components/Hero.astro:22,29,33`

**Step 1: Ensure hero title is centered**

The hero container (line 22) already has `items-center` but the `text-center` class is missing from the container. Add it:

Old (line 22):
```html
<div
  class="hero flex flex-col items-center mx-auto px-4 sm:px-6 lg:px-8 pt-20 pb-35"
>
```

New:
```html
<div
  class="hero flex flex-col items-center text-center mx-auto px-4 sm:px-6 lg:px-8 pt-20 pb-35"
>
```

**Step 2: Constrain tagline width**

On line 29, the tagline `<p>` should have max-width for readable line length:

Old:
```html
{tagline && <p set:html={tagline} class="text-xl text-center" />}
```

New:
```html
{tagline && <p set:html={tagline} class="text-xl text-center max-w-2xl" />}
```

**Step 3: Center the search and badges wrapper**

On line 33, add `mx-auto` and `w-full` to ensure the container is centered:

Old:
```html
<div class="mt-12 max-w-4xl">
```

New:
```html
<div class="mt-12 max-w-4xl w-full mx-auto">
```

**Step 4: Verify build**

Run:
```bash
cd docs/website && yarn build 2>&1 | tail -5
```
Expected: Build succeeds.

**Step 5: Commit**

```bash
git add docs/website/src/components/override-components/Hero.astro
git commit -m "fix(website): center hero text, constrain tagline width"
```

---

### Task 6: Fix section description centering

**Files:**
- Modify: `docs/website/src/styles/components.css:70-74`

**Step 1: Add max-width and auto margin to section description**

In `docs/website/src/styles/components.css`, replace lines 70-74:

Old:
```css
.section-description {
  font-size: 1.125rem;
  line-height: 1.75rem;
  letter-spacing: -0.2px;
}
```

New:
```css
.section-description {
  font-size: 1.125rem;
  line-height: 1.75rem;
  letter-spacing: -0.2px;
  max-width: 48rem;
  margin-left: auto;
  margin-right: auto;
}
```

**Step 2: Fix the `.light-text` display property**

On line 56, the `.light-text` class uses `display: block` which forces line breaks within section titles. Change to `inline`:

Old:
```css
.light-text {
  @apply text-lightmode-text/50 dark:text-text/50 block;
}
```

New:
```css
.light-text {
  @apply text-lightmode-text/50 dark:text-text/50 inline;
}
```

**Step 3: Verify build**

Run:
```bash
cd docs/website && yarn build 2>&1 | tail -5
```
Expected: Build succeeds.

**Step 4: Commit**

```bash
git add docs/website/src/styles/components.css
git commit -m "fix(website): center section descriptions, fix light-text inline display"
```

---

### Task 7: Verify end-to-end

**Files:** None (verification only)

**Step 1: Build the site**

Run:
```bash
cd docs/website && yarn build
```
Expected: Clean build, no errors.

**Step 2: Preview the site**

Run:
```bash
cd docs/website && yarn preview
```

**Step 3: Manual verification checklist**

Open the preview URL and verify:

1. **Logo glow**: Teal-tinted glow behind the logo (not rose)
2. **Gradient text**: Stat badge values show navy→teal→cyan gradient (not purple→rose→gold)
3. **Primary buttons**: "Get Started" button uses teal gradient
4. **Card hover borders**: Feature cards and architecture cards show teal gradient border on hover
5. **Blockquote borders**: Use teal gradient
6. **Hero text**: Title "Beluga AI" is centered
7. **Tagline**: Centered with readable max-width
8. **Section titles**: "Built for production AI", etc. are centered with light-text inline (not on separate line)
9. **Section descriptions**: Centered with max-width constraint
10. **Search bar**: Centered within hero
11. **Stats row**: Evenly spaced and centered
12. **Nav active border**: Teal gradient

**Step 4: Commit any polish if needed**

```bash
git add -A
git commit -m "fix(website): final polish after visual review"
```

---

## Summary

| Task | Description | Dependencies |
|------|-------------|-------------|
| 1 | Update theme.json primary color | None |
| 2 | Update gradient CSS variable | None |
| 3 | Update Hero.astro logo glow/shadow | None |
| 4 | Update OG image accent | None |
| 5 | Fix hero text centering | None |
| 6 | Fix section description centering | None |
| 7 | End-to-end verification | Tasks 1-6 |

Tasks 1-6 are all independent and can be done in any order. Task 7 depends on all of them.
