# Technical Architecture — Milestone 3: Static Documentation Website

**Date:** 2026-02-28
**Status:** Accepted
**Branch:** `milestone/3-website`
**Module:** `github.com/mrlm-net/simconnect-mcp`

## Design Direction

**Aesthetic**: Cool technocratic — dark background (GitHub Dark palette), monospace code fonts (JetBrains Mono), terminal-style accents, precise grid layouts, aviation/data instrumentation visual language. Think flight deck meets developer tooling.

**Web Vitals**: All pages must meet Google Core Web Vitals thresholds — LCP < 2.5 s, INP < 200 ms, CLS < 0.1. Achieved via: static prerendering (adapter-static), no layout shift from font loading (`font-display: swap` + `size-adjust`), minimal JS hydration (Svelte compiles to lean output), image dimensions declared upfront.

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [ADR-010: Monorepo vs Separate Repository](#2-adr-010-monorepo-vs-separate-repository)
3. [ADR-011: Shared Component Library vs Direct Copy](#3-adr-011-shared-component-library-vs-direct-copy)
4. [ADR-012: Content Source Location](#4-adr-012-content-source-location)
5. [ADR-013: Dependency Version Alignment with Sibling](#5-adr-013-dependency-version-alignment-with-sibling)
6. [Component Architecture](#6-component-architecture)
7. [Content Pipeline Design](#7-content-pipeline-design)
8. [Navigation Structure](#8-navigation-structure)
9. [Route Map](#9-route-map)
10. [GitHub Actions Workflow Design](#10-github-actions-workflow-design)
11. [Cross-Milestone Notes (Milestone 4)](#11-cross-milestone-notes-milestone-4)
12. [Directory Layout](#12-directory-layout)
13. [Interface Definitions](#13-interface-definitions)
14. [Implementation Guidelines](#14-implementation-guidelines)

---

## 1. System Overview

Milestone 3 delivers a static documentation website at `simconnect-mcp.mrlm.net`,
hosted on GitHub Pages. The site is a SvelteKit application in the `website/`
subfolder of the existing monorepo. It mirrors the structure and stack of the sibling
project at `simconnect.mrlm.net` (`github.com/mrlm-net/simconnect`), with adaptations
specific to the MCP server audience.

The site consumes markdown files from the repo-root `docs/` folder (excluding
ADR records), renders them as static HTML at build time, and deploys via a
GitHub Actions workflow triggered on push to `main`.

### C4 Context Diagram

```
+---------------------------------------------------------------+
|  User (developer, flight sim enthusiast)                      |
|  Browser -> HTTPS simconnect-mcp.mrlm.net                     |
+-----------------------------+---------------------------------+
                              |
                              v
+---------------------------------------------------------------+
|  GitHub Pages CDN                                             |
|  Serves pre-built static HTML/CSS/JS from gh-pages branch    |
+-----------------------------+---------------------------------+
                              |
                              v (build time only)
+---------------------------------------------------------------+
|  GitHub Actions: website workflow                             |
|  Runner: ubuntu-latest                                        |
|  1. Checkout repo                                             |
|  2. Node.js setup + pnpm install (website/)                   |
|  3. vite build (reads docs/ via ../docs/ relative path)       |
|  4. Deploy build/ to gh-pages branch                         |
+-----------------------------+---------------------------------+
                              |
                              v (source)
+---------------------------------------------------------------+
|  Monorepo: mrlm-net/simconnect-mcp                            |
|  website/        SvelteKit app                                |
|  docs/           Markdown content source                      |
|  cmd/ internal/  Go server (unchanged)                        |
+---------------------------------------------------------------+
```

### Build-Time Data Flow

```
docs/
  getting-started.md          --+
  configuration.md              |   pipeline.server.ts
  mcp-tools-docs.md             |   (Node.js, build-time only)
  mcp-tools-simconnect.md       +--> gray-matter (frontmatter)
  claude-desktop.md             |    mdsvex compile (markdown -> HTML)
  examples.md                   |    Prism (syntax highlighting)
  changelog.md                --+    rehype-slug / rehype-rewrite-links
                                            |
                                            v
                              SvelteKit SSG (adapter-static)
                                            |
                                            v
                              website/build/   (static HTML/CSS/JS)
                                            |
                                            v
                              GitHub Pages  simconnect-mcp.mrlm.net
```

---

## 2. ADR-010: Monorepo vs Separate Repository

**Status:** Accepted

### Context

The documentation website references content from `docs/`, links to source code in
the same repo, and must stay in sync with server releases. Two structural options:

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| A: Monorepo subfolder (`website/`) | Website lives in `mrlm-net/simconnect-mcp` at `website/` | Single PR touches code and docs together; content path `../docs/` is direct; consistent with sibling project pattern; no cross-repo deploy tokens | Website CI runs on every Go push (mitigated by path filters) |
| B: Separate repo | New `mrlm-net/simconnect-mcp-docs` repo | Complete isolation; independent deploy | Cross-repo sync required; content duplication or git submodule complexity; breaks pattern established by sibling |

### Decision

**Option A: Monorepo subfolder `website/`.**

### Rationale

The sibling project (`mrlm-net/simconnect`) uses an identical monorepo layout — the
website is a `website/` subfolder reading `../docs/` for content. This is a proven,
working pattern in the organisation. Maintaining a single repository means a PR that
adds a new env variable can update `docs/configuration.md` and the site source in
the same diff. Content and code stay in lockstep without synchronisation overhead.

The concern about CI noise on Go pushes is resolved by path filters on the website
workflow trigger (`paths: ['docs/**', 'website/**', '.github/workflows/website.yml']`),
which ensures the website build only runs when relevant files change.

### Consequences

- `website/` is created at the repository root alongside existing Go packages.
- The Go `go build ./...` commands must never descend into `website/`; Go tooling
  already ignores non-`.go` directories so no explicit exclusion is needed.
- The `website/` package.json uses `"private": true` to prevent accidental npm publish.
- `.gitignore` entries are added for `website/node_modules/` and `website/build/`.

---

## 3. ADR-011: Shared Component Library vs Direct Copy

**Status:** Accepted

### Context

The sibling project has six layout components (`Header`, `Footer`, `Sidebar`,
`TableOfContents`, `SeoHead`, `JsonLd`) and three rehype plugins. Two integration
options:

| Option | Description | Pros | Cons |
|--------|-------------|------|------|
| A: Copy and adapt | Duplicate component files into `website/src/lib/` | No cross-repo dependency; modify freely; no versioning overhead | Divergence risk over time; bug fixes must be applied in two places |
| B: Shared npm package | Extract components into `@mrlm-net/docs-components` npm package | Single source of truth for shared UI | Requires publishing an npm package; adds release overhead; premature abstraction at two sites |

### Decision

**Option A: Copy and adapt components from the sibling project.**

### Rationale

There are two sites in the organisation at this time. Extracting a shared component
library requires publishing and versioning an npm package, which is significant
infrastructure overhead for a two-consumer abstraction. The sibling project's
components are small (each under 150 lines) and the customisation surface (site name,
colour accents for brand differentiation, navigation structure) is large enough that
the components will diverge anyway.

The practical risk of not propagating bug fixes is low: both sites are primarily
read-only documentation; the component bugs that matter are visual regressions easily
caught in the website's own PR review.

If a third documentation site is added to the organisation, this decision should be
revisited as ADR-011-rev-1 and the threshold for extracting a shared package is three
consumers.

### Consequences

- All six layout components are copied from `mrlm-net/simconnect/website/src/lib/components/`
  into `website/src/lib/components/` and adapted for simconnect-mcp identity.
- The three rehype plugins (`rehype-slug`, `rehype-rewrite-links`, `rehype-table-wrap`)
  are copied verbatim — they contain no project-specific logic.
- `rehype-rewrite-links` requires adaptation: the GitHub URL is changed from
  `mrlm-net/simconnect` to `mrlm-net/simconnect-mcp` and the `/docs/` base path is
  updated to match this site's route structure.
- `app.css` (GitHub Dark theme, Prism token colours, prose overrides) is copied verbatim
  with only the custom property for the accent colour changed to a distinct hue to
  visually differentiate the two sites (`--color-accent: #3fb950` -> keep green as a
  flight-data colour; adjust as brand preference dictates during implementation).

---

## 4. ADR-012: Content Source Location

**Status:** Accepted

### Context

Markdown content can live in two places:

| Option | Path | Pros | Cons |
|--------|------|------|------|
| A: Repo root `docs/` | `docs/*.md` | Consumed directly by the pipeline via `../docs/`; single source of truth; consistent with sibling project | All public-facing and contributor-only content must coexist in the same folder |
| B: `website/content/` | `website/content/*.md` | Website content fully isolated from contributor docs | Content duplication — some docs already exist at `docs/`; breaks sibling pattern |

### Decision

**Option A: `docs/` as the content source, with filename conventions to separate
public pages from contributor/ADR documents.**

### Rationale

The sibling project reads `../docs/` from the pipeline. Adopting the same convention
means the pipeline code is copied without modification. The existing `docs/` folder
already has the right content: `docs/api/mcp-tools.md` covers both modes; the
architecture documents are excluded from the public site by frontmatter convention.

ADR documents (`docs/decisions/`) and architecture notes (`docs/architecture/`) are
excluded via a `public: false` frontmatter flag or by directory exclusion in the
pipeline loader function. The pipeline only reads files from `docs/` root level
(not subdirectories), which naturally excludes `decisions/` and `architecture/`
without requiring any frontmatter changes on existing files.

New public content files (`getting-started.md`, `configuration.md`,
`mcp-tools-docs.md`, `mcp-tools-simconnect.md`, `claude-desktop.md`, `examples.md`,
`changelog.md`) are authored directly in `docs/` with frontmatter.

### Consequences

- The pipeline `docsDir()` function returns `path.resolve(process.cwd(), '..', 'docs')`
  and reads only `*.md` files at the top level of that directory (no recursive walk).
- Files in `docs/api/`, `docs/architecture/`, `docs/decisions/`, `docs/security/`
  subdirectories are ignored by the pipeline because the loader only scans the flat
  `docs/` root.
- Existing files (`docs/api/mcp-tools.md`) are migrated to flat `docs/` root level
  as new filenames (`docs/mcp-tools-docs.md`, `docs/mcp-tools-simconnect.md`) with
  appropriate frontmatter added. The originals in `docs/api/` are retained as-is for
  contributor reference.
- Each public markdown file carries YAML frontmatter with `title`, `description`,
  `order`, and `section` fields.

---

## 5. ADR-013: Dependency Version Alignment with Sibling

**Status:** Accepted

### Context

The sibling site's `package.json` declares specific version ranges. Matching those
ranges versus using latest:

| Option | Description |
|--------|-------------|
| A: Match sibling versions exactly | Lowest risk; confirmed working configuration |
| B: Use latest compatible versions | May pick up fixes/features; increases divergence |

### Decision

**Match the sibling site's dependency version ranges exactly, with one intentional
deviation: add `prismjs/components/prism-go` import for Go syntax highlighting and
confirm `unist-util-visit` is present for the rehype plugins.**

### Confirmed Stack

| Dependency | Version Range | Role |
|------------|--------------|------|
| `svelte` | `^5.20.0` | Component framework |
| `@sveltejs/kit` | `^2.16.0` | Meta-framework, SSG |
| `@sveltejs/adapter-static` | `^3.0.8` | Emit static HTML |
| `@sveltejs/vite-plugin-svelte` | `^5.0.3` | Svelte/Vite integration |
| `tailwindcss` | `^4.1.3` | Utility CSS |
| `@tailwindcss/vite` | `^4.1.3` | Tailwind Vite plugin |
| `@tailwindcss/typography` | `^0.5.16` | Prose styling |
| `mdsvex` | `^0.12.5` | Markdown-in-Svelte compiler |
| `gray-matter` | `^4.0.3` | YAML frontmatter parser |
| `prismjs` | `^1.30.0` | Syntax highlighting |
| `vite` | `^6.2.4` | Build toolchain |
| `unist-util-visit` | `^5.0.0` | AST traversal for rehype plugins |
| `typescript` | `^5.7.3` | Type checking |
| `svelte-check` | `^4.1.5` | Svelte type checking |
| `@types/node` | `^25.3.0` | Node.js types |
| `@types/prismjs` | `^1.26.6` | Prism types |

**Deviation from sibling**: The sibling imports `prism-go.js` in its pipeline; this
site does too — no deviation required. The `rehype-slug` plugin is a local copy, not
an npm dependency, matching sibling.

**Package manager**: Use `npm` (consistent with organisation tooling; avoids adding
a pnpm/yarn installation step to CI). The sibling does not specify a package manager
in `package.json`; `npm` is the safe default.

### Consequences

- `website/package.json` is authored with the versions above.
- No additional npm dependencies are required beyond what the sibling uses.
- `website/.npmrc` is created with `engine-strict=false` to prevent node version
  mismatch failures in CI.

---

## 6. Component Architecture

### Porting Classification

| Component | Source in Sibling | Action for simconnect-mcp | Changes Required |
|-----------|-------------------|--------------------------|-----------------|
| `Header` | `components/layout/Header.svelte` | Port verbatim, adapt branding | Site name, repo URL, nav links |
| `Footer` | `components/layout/Footer.svelte` | Port verbatim, adapt branding | Repo URL, copyright owner |
| `Sidebar` | `components/layout/Sidebar.svelte` | Port verbatim | None — nav sections driven by config |
| `TableOfContents` | `components/layout/TableOfContents.svelte` | Port verbatim | None |
| `SeoHead` | `components/layout/SeoHead.svelte` | Port verbatim | None |
| `JsonLd` | `components/layout/JsonLd.svelte` | Port verbatim | None |
| `rehype-slug` | `lib/plugins/rehype-slug.js` | Copy verbatim | None |
| `rehype-rewrite-links` | `lib/plugins/rehype-rewrite-links.js` | Copy, adapt URLs | GitHub repo path: `mrlm-net/simconnect-mcp`; base path: `/docs/` |
| `rehype-table-wrap` | `lib/plugins/rehype-table-wrap.js` | Copy verbatim | None |
| `pipeline.server.ts` | `lib/content/pipeline.server.ts` | Copy, adapt Prism imports | Keep `prism-go`, `prism-bash`, `prism-json`; add `prism-yaml`; same `docsDir()` logic |
| `toc.ts` | `lib/content/toc.ts` | Copy verbatim | None |
| `navigation.ts` | `lib/config/navigation.ts` | New — simconnect-mcp sections | Sections: `getting-started`, `reference`, `integration`, `examples` |
| `site.ts` | `lib/config/site.ts` | New — simconnect-mcp metadata | Different title, URL, repo URL |
| `types/index.ts` | `lib/types/index.ts` | Copy verbatim | Interfaces are project-agnostic |
| `app.css` | `src/app.css` | Copy, adapt accent colour | CSS variable for accent/brand hue |
| `app.html` | `src/app.html` | Copy, adapt title/fonts | Page title, og tags |
| Marketing landing page | `routes/(marketing)/+page.svelte` | New — simconnect-mcp hero | Different headline, feature grid |
| Docs layout | `routes/(docs)/+layout.svelte` | Port verbatim | None |
| Docs slug page | `routes/(docs)/[slug]/+page.svelte` | Port verbatim | None |
| `sitemap.xml` route | `routes/sitemap.xml/+server.ts` | Port verbatim | Update base URL |
| `llm.txt` route | `routes/llm.txt/+server.ts` | Port verbatim | Update site identity string |
| `getting-started` route | N/A in sibling (it's a docs page) | Not needed — rendered via `[slug]` | The content file handles this |
| `changelog` route | `routes/changelog/+page.svelte` | Port, adapt for hand-authored | Remove GitHub API call; read from `docs/changelog.md` via pipeline |

### New Components (simconnect-mcp-specific)

None — the marketing landing page hero and feature grid are new Svelte markup in
`routes/(marketing)/+page.svelte`, not a new reusable component.

---

## 7. Content Pipeline Design

### Pipeline Flow

```
docs/*.md (flat directory, build-time)
    |
    +-- fs.readdirSync() -> filter *.md
    |
    +-- gray-matter -> { data: frontmatter, content: markdown body }
    |
    +-- mdsvex compile() -> Svelte component string
    |       |
    |       +-- highlight: Prism.highlight (go, bash, json, yaml)
    |       +-- rehypePlugins:
    |             rehype-slug     (add id= to all h2/h3/h4)
    |             rehype-rewrite-links  (rewrite .md -> /docs/ paths)
    |             rehype-table-wrap     (wrap <table> in .table-wrapper div)
    |
    +-- resolveHtmlDirectives() -> strip Svelte {@html `...`} wrapper
    +-- strip <script> and <style> tags
    |
    +-- extractToc(content) -> TocEntry[] (h2 headings only)
    |
    v
DocPage { slug, title, description, renderedContent, headings }
    |
    v
SvelteKit SSG (adapter-static)
    -> website/build/<slug>/index.html
```

### Frontmatter Schema

Every public content file in `docs/` carries this YAML frontmatter:

```yaml
---
title: "Getting Started"
description: "Install and run SimConnect MCP in docs mode or simconnect mode."
order: 1
section: getting-started
---
```

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `title` | string | Yes | Page title (used in `<title>`, sidebar link, `<h1>`) |
| `description` | string | Yes | Page description for `<meta description>` and OG tags |
| `order` | integer | Yes | Sort order within the section for sidebar rendering |
| `section` | string | Yes | Section identifier; maps to `navigation.ts` section definitions |

### Content Files to Author

The following files are new, authored for Milestone 3. Existing files from `docs/api/`
are the primary source material.

| File | Section | Order | Source Material |
|------|---------|-------|----------------|
| `docs/getting-started.md` | `getting-started` | 1 | README.md Quick Start, Prerequisites |
| `docs/configuration.md` | `getting-started` | 2 | README.md Environment Variables table |
| `docs/mcp-tools-docs.md` | `reference` | 1 | `docs/api/mcp-tools.md` (docs mode section) |
| `docs/mcp-tools-simconnect.md` | `reference` | 2 | `docs/api/mcp-tools.md` (simconnect mode section) |
| `docs/claude-desktop.md` | `integration` | 1 | README.md MCP Client Configuration section |
| `docs/examples.md` | `integration` | 2 | New — three worked scenarios (see §8) |
| `docs/changelog.md` | `changelog` | 1 | Hand-authored for M3 release |

### Changelog Handling

The sibling site fetches the changelog from GitHub Releases API at build time. This
site uses a simpler approach: `docs/changelog.md` is hand-authored markdown, rendered
by the same pipeline as all other docs pages. No GitHub API token or API call is
needed at build time.

This is a deliberate simplification. If release cadence increases after Milestone 4,
the changelog route can be upgraded to fetch GitHub Releases in a follow-up.

---

## 8. Navigation Structure

### Sections and Sidebar Order

```
Getting Started
  |-- Getting Started                /docs/getting-started
  |-- Configuration                  /docs/configuration

Reference
  |-- MCP Tools — Docs Mode          /docs/mcp-tools-docs
  |-- MCP Tools — SimConnect Mode    /docs/mcp-tools-simconnect

Integration
  |-- Claude Desktop Setup           /docs/claude-desktop
  |-- Examples                       /docs/examples

Changelog
  |-- Changelog                      /docs/changelog
```

### navigation.ts Section Definitions

```typescript
const sectionMeta: Record<string, { title: string; defaultOpen: boolean }> = {
  'getting-started': { title: 'Getting Started', defaultOpen: true },
  'reference':       { title: 'Reference',        defaultOpen: true },
  'integration':     { title: 'Integration',      defaultOpen: true },
  'changelog':       { title: 'Changelog',        defaultOpen: false }
};

const sectionOrder = ['getting-started', 'reference', 'integration', 'changelog'];
```

### Top-Bar Navigation Links

| Label | URL | Type |
|-------|-----|------|
| Docs | `/docs/getting-started` | Internal |
| Examples | `/docs/examples` | Internal |
| Changelog | `/docs/changelog` | Internal |
| GitHub | `https://github.com/mrlm-net/simconnect-mcp` | External |

---

## 9. Route Map

Full SvelteKit route tree with source for each leaf:

```
website/src/routes/
  +layout.server.ts          Loads nav data (loadDocIndex) server-side, passes to layout
  +layout.svelte             Root layout: <SeoHead>, minimal shell

  (marketing)/
    +layout.svelte           Marketing layout: <Header>, <Footer>
    +page.svelte             Landing page — hero, feature grid, quick-start snippet
    +page.ts                 No server load needed (static content only)

  (docs)/
    +layout.svelte           Docs layout: <Header>, <Sidebar>, <TableOfContents>, <Footer>
    +layout.server.ts        Passes nav sections from root layout data to Sidebar
    [slug]/
      +page.svelte           Renders DocPage.renderedContent + heading
      +page.server.ts        Calls loadDocPage(slug); throws 404 if null
      +page.ts               export const prerender = true

  sitemap.xml/
    +server.ts               GET handler: emits sitemap XML for all doc slugs + home

  llm.txt/
    +server.ts               GET handler: emits plain-text project description for LLMs
```

### Static Route Entries (prerendered)

| Route | Slug | Output Path |
|-------|------|-------------|
| `/` | — | `build/index.html` |
| `/docs/getting-started` | `getting-started` | `build/docs/getting-started/index.html` |
| `/docs/configuration` | `configuration` | `build/docs/configuration/index.html` |
| `/docs/mcp-tools-docs` | `mcp-tools-docs` | `build/docs/mcp-tools-docs/index.html` |
| `/docs/mcp-tools-simconnect` | `mcp-tools-simconnect` | `build/docs/mcp-tools-simconnect/index.html` |
| `/docs/claude-desktop` | `claude-desktop` | `build/docs/claude-desktop/index.html` |
| `/docs/examples` | `examples` | `build/docs/examples/index.html` |
| `/docs/changelog` | `changelog` | `build/docs/changelog/index.html` |
| `/sitemap.xml` | — | `build/sitemap.xml` |
| `/llm.txt` | — | `build/llm.txt` |

### svelte.config.js Adapter Options

```javascript
adapter({
  pages: 'build',
  assets: 'build',
  fallback: '404.html',
  precompress: false,
  strict: true
})
```

`BASE_PATH` env var support is included for subdirectory deployment compatibility,
matching the sibling pattern.

---

## 10. GitHub Actions Workflow Design

### Workflow: `website.yml`

**File**: `.github/workflows/website.yml`

**Triggers**:
- `push` to `main` — branch: build + deploy
- `pull_request` targeting `main` — build only (no deploy)
- Path filters (both triggers): `['docs/**', 'website/**', '.github/workflows/website.yml']`

**Permissions** (required for GitHub Pages deployment):
```yaml
permissions:
  contents: read
  pages: write
  id-token: write
```

**Concurrency** (prevents in-flight deploys racing):
```yaml
concurrency:
  group: pages
  cancel-in-progress: true
```

### Job: `build`

```
Runs on: ubuntu-latest
Steps:
  1. actions/checkout@v4
  2. actions/setup-node@v4  (node-version: '20', cache: 'npm', cache-dependency-path: website/package-lock.json)
  3. npm ci  (working-directory: website)
  4. npm run check  (working-directory: website)
  5. npm run build  (working-directory: website, env: BASE_PATH='')
  6. actions/upload-pages-artifact@v3  (path: website/build)
```

### Job: `deploy` (push to main only)

```
Runs on: ubuntu-latest
Needs: build
Environment: github-pages
Steps:
  1. actions/deploy-pages@v4
```

### Isolation from Go CI Workflow

The existing `ci.yml` runs on paths matching Go source files. The `website.yml`
workflow runs on separate path filters. They share no jobs and have no dependencies
on each other. Both can run concurrently on the same push without conflict because
they operate on independent artifact stores.

The `website.yml` workflow does NOT call `go build` or `go test`. The `ci.yml`
workflow does NOT call `npm` or `vite`. No shared cache keys are used.

### Full workflow skeleton

```yaml
name: Website

on:
  push:
    branches: [main]
    paths:
      - 'docs/**'
      - 'website/**'
      - '.github/workflows/website.yml'
  pull_request:
    branches: [main]
    paths:
      - 'docs/**'
      - 'website/**'
      - '.github/workflows/website.yml'

permissions:
  contents: read
  pages: write
  id-token: write

concurrency:
  group: pages
  cancel-in-progress: true

jobs:
  build:
    name: Build website
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: '20'
          cache: 'npm'
          cache-dependency-path: website/package-lock.json
      - name: Install dependencies
        run: npm ci
        working-directory: website
      - name: Type-check
        run: npm run check
        working-directory: website
      - name: Build
        run: npm run build
        working-directory: website
        env:
          BASE_PATH: ''
      - name: Upload Pages artifact
        uses: actions/upload-pages-artifact@v3
        with:
          path: website/build

  deploy:
    name: Deploy to GitHub Pages
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    needs: build
    runs-on: ubuntu-latest
    environment:
      name: github-pages
      url: ${{ steps.deployment.outputs.page_url }}
    steps:
      - name: Deploy
        id: deployment
        uses: actions/deploy-pages@v4
```

### GitHub Repository Settings Required

- Source: "GitHub Actions" (not "Deploy from branch") in Settings > Pages
- Custom domain: `simconnect-mcp.mrlm.net`
- CNAME record: `simconnect-mcp.mrlm.net CNAME mrlm-net.github.io`
- `website/static/CNAME` file containing `simconnect-mcp.mrlm.net`
- Enforce HTTPS: enabled after DNS propagation

---

## 11. Cross-Milestone Notes (Milestone 4)

Milestone 4 delivers a pre-built Windows binary in GitHub Releases and a Docker image
for docs mode. The following interactions between Milestone 3 and Milestone 4 CI/CD
must be managed:

### Concern 1: GitHub Pages deployment on main vs Release tagging

Milestone 4 will introduce a release workflow that creates a GitHub Release (and git
tag `v*`) on push to `main` or on manual trigger. The website workflow runs on push
to `main` and on path filters. Two considerations:

- The release workflow MUST also set path filters or a `workflow_dispatch` trigger so
  it does not run on every docs-only push. Alternatively, gate it on tag creation
  only (`on: push: tags: ['v*']`) — this is the recommended pattern.
- The website workflow MUST NOT be blocked by the release workflow's `pages` concurrency
  group. Because the release workflow does not deploy to GitHub Pages, there is no
  concurrency conflict. Verify this when Milestone 4 is designed.

### Concern 2: Changelog content

Milestone 4 will produce versioned releases. The `docs/changelog.md` file must be
updated as part of the Milestone 4 release process (either hand-authored or generated
from GitHub Releases). The website workflow's path filter on `docs/**` ensures that
updating `docs/changelog.md` as part of a release PR automatically triggers a site
rebuild and deploy.

Recommended Milestone 4 release sequence:
1. Update `docs/changelog.md` with M4 release notes in the release PR.
2. Merge PR to `main` — triggers both the release workflow (tag + GitHub Release)
   and the website workflow (site rebuild with updated changelog).

### Concern 3: Docker image and binary — no website interaction

The Docker image and Windows binary artifacts are uploaded to GitHub Container Registry
and GitHub Releases respectively. Neither interacts with GitHub Pages. No conflicts.

### Concern 4: `ci.yml` branch trigger updates for Milestone 4

The existing `ci.yml` triggers on `milestone/1-docs`, `milestone/2-simconnect`, and
`main`. The Milestone 3 branch (`milestone/3-website`) should be added. The Milestone 4
branch (`milestone/4-packaging`) should also be added when that milestone begins.
Update: add both branches to `ci.yml` triggers as part of the branch creation for
each milestone. The website workflow only needs to target `main` since static sites
deploy from main.

---

## 12. Directory Layout

```
website/
  src/
    app.css                          GitHub Dark theme + Prism token colours
    app.html                         HTML shell with Inter/JetBrains Mono fonts

    lib/
      components/
        layout/
          Header.svelte              Site header: logo, nav links, GitHub link
          Footer.svelte              Site footer: copyright, repo link
          Sidebar.svelte             Doc navigation sidebar with collapsible sections
          TableOfContents.svelte     In-page TOC from h2 headings
          SeoHead.svelte             <svelte:head> SEO meta tags + OG tags
          JsonLd.svelte              JSON-LD structured data (WebSite + BreadcrumbList)

      config/
        site.ts                      siteConfig: title, url, repoUrl, basePath, ogImage
        navigation.ts                buildNavigation(): section definitions + ordering

      content/
        pipeline.server.ts           loadDocIndex(), loadDocPage(), loadAllSlugs()
        toc.ts                       extractToc(), slugify()
        types.ts                     DocPage interface (slug, title, renderedContent, headings)

      plugins/
        rehype-slug.js               Add id= attributes to headings
        rehype-rewrite-links.js      Rewrite .md links to /docs/ paths
        rehype-table-wrap.js         Wrap <table> in .table-wrapper <div>

      types/
        index.ts                     NavItem, NavSection, TocEntry, SiteConfig,
                                     DocMeta, ChangelogRelease interfaces

    routes/
      +layout.server.ts              Loads nav data (loadDocIndex + buildNavigation)
      +layout.svelte                 Root layout shell

      (marketing)/
        +layout.svelte               Marketing layout with Header + Footer
        +page.svelte                 Landing page: hero, feature cards, quick-start

      (docs)/
        +layout.svelte               Docs layout: Header + Sidebar + ToC + Footer
        +layout.server.ts            Passes nav from parent layout
        [slug]/
          +page.server.ts            Loads DocPage via loadDocPage(slug)
          +page.svelte               Renders doc content + heading
          +page.ts                   export const prerender = true

      sitemap.xml/
        +server.ts                   Emits sitemap.xml for all routes

      llm.txt/
        +server.ts                   Emits plain-text project description

  static/
    CNAME                            simconnect-mcp.mrlm.net
    favicon.ico
    favicon.svg
    og-image.png                     1200x630 Open Graph image

  package.json
  svelte.config.js
  tsconfig.json
  vite.config.ts
  .npmrc
```

---

## 13. Interface Definitions

### site.ts (simconnect-mcp values)

```typescript
// website/src/lib/config/site.ts
import type { SiteConfig } from '$lib/types/index.js';

export const siteConfig: SiteConfig = {
  title: 'SimConnect MCP',
  description: 'Model Context Protocol server for Microsoft Flight Simulator — read SimConnect docs and live sim data from your AI assistant.',
  repoUrl: 'https://github.com/mrlm-net/simconnect-mcp',
  basePath: '',
  url: 'https://simconnect-mcp.mrlm.net',
  ogImage: {
    width: 1200,
    height: 630
  },
  locale: 'en_US',
  license: 'MIT'
};
```

### navigation.ts (simconnect-mcp sections)

```typescript
// website/src/lib/config/navigation.ts
import type { DocMeta, NavSection } from '$lib/types/index.js';

const sectionMeta: Record<string, { title: string; defaultOpen: boolean }> = {
  'getting-started': { title: 'Getting Started', defaultOpen: true },
  'reference':       { title: 'Reference',        defaultOpen: true },
  'integration':     { title: 'Integration',      defaultOpen: true },
  'changelog':       { title: 'Changelog',        defaultOpen: false }
};

const sectionOrder = ['getting-started', 'reference', 'integration', 'changelog'];

export function buildNavigation(docs: DocMeta[], basePath: string): NavSection[] {
  // identical algorithm to sibling — groups by section, sorts by order
}
```

### Frontmatter examples for each content file

```markdown
<!-- docs/getting-started.md -->
---
title: "Getting Started"
description: "Install and run SimConnect MCP in docs or simconnect mode."
order: 1
section: getting-started
---

<!-- docs/configuration.md -->
---
title: "Configuration"
description: "Environment variables for MCP_MODE, PORT, DOCS_MSFS_VERSION, and more."
order: 2
section: getting-started
---

<!-- docs/mcp-tools-docs.md -->
---
title: "MCP Tools — Docs Mode"
description: "Reference for all 11 MCP tools exposed in docs mode."
order: 1
section: reference
---

<!-- docs/mcp-tools-simconnect.md -->
---
title: "MCP Tools — SimConnect Mode"
description: "Reference for the 4 live-data MCP tools in simconnect mode."
order: 2
section: reference
---

<!-- docs/claude-desktop.md -->
---
title: "Claude Desktop Setup"
description: "Connect Claude Desktop to SimConnect MCP in docs or live mode."
order: 1
section: integration
---

<!-- docs/examples.md -->
---
title: "Examples"
description: "Worked scenarios: query altitude, toggle landing lights, check sim state."
order: 2
section: integration
---

<!-- docs/changelog.md -->
---
title: "Changelog"
description: "Release history for SimConnect MCP."
order: 1
section: changelog
---
```

### Examples content structure (three required scenarios)

The `docs/examples.md` file must document at minimum:

1. **Query current altitude** — using `get_simvar_value` with `name: "PLANE ALTITUDE"`, `unit: "feet"` in a Claude Desktop conversation; includes the JSON request/response from `docs/api/mcp-tools.md`.
2. **Toggle landing lights** — using `transmit_event` with `name: "LANDING_LIGHTS_TOGGLE"`; shows Claude Desktop prompt + tool call flow.
3. **Check simulator state** — using `get_sim_state`; shows both connected and disconnected response examples.

---

## 14. Implementation Guidelines

### File Creation Order

Implement in this order to keep the build runnable at each step:

1. `website/package.json`, `website/svelte.config.js`, `website/tsconfig.json`, `website/vite.config.ts`, `website/.npmrc`
2. `website/src/lib/types/index.ts`
3. `website/src/lib/plugins/` (three rehype plugins — copy and adapt)
4. `website/src/lib/content/toc.ts`, `website/src/lib/content/types.ts`
5. `website/src/lib/config/site.ts`, `website/src/lib/config/navigation.ts`
6. `website/src/app.css`, `website/src/app.html`
7. `website/src/lib/components/layout/` (all six components)
8. `website/src/lib/content/pipeline.server.ts`
9. `website/src/routes/` (root layout, then marketing, then docs, then sitemap/llm.txt)
10. `docs/*.md` — author the seven content files with frontmatter
11. `website/static/CNAME` and static assets
12. `.github/workflows/website.yml`
13. `.github/workflows/ci.yml` — add `milestone/3-website` to branch triggers

### Build Verification Checklist

Before merging to `main`:
- `npm run check` passes with zero errors
- `npm run build` completes and emits all 10 routes to `website/build/`
- All internal links resolve (SvelteKit strict mode will fail the build on broken slug references)
- `website/build/CNAME` is present (copied from `static/CNAME` by adapter-static)
- `website/build/sitemap.xml` lists all 8 doc pages + home page
- Go CI (`ci.yml`) still passes — no Go files were modified

### Testing Strategy

- No unit tests for the website (server-rendered markdown pipeline is tested implicitly
  by `npm run build` completing without errors and all pages prerendering).
- PR builds on `pull_request` events serve as the integration test gate — if the build
  fails, the PR is blocked.
- Visual regression testing is out of scope for Milestone 3; add to Milestone 5 backlog
  if needed.

### Naming Conventions

- SvelteKit files: SvelteKit conventions (`+page.svelte`, `+layout.server.ts`)
- Component files: PascalCase (`Header.svelte`, `Sidebar.svelte`)
- Config/lib files: camelCase (`site.ts`, `navigation.ts`, `pipeline.server.ts`)
- CSS custom properties: `--color-*`, `--bg-*` (inherited from sibling)
- Content file slugs: kebab-case matching the filename without `.md`
  (`getting-started`, `mcp-tools-docs`, `claude-desktop`)

### .gitignore Additions Required

```
website/node_modules/
website/build/
website/.svelte-kit/
```

### Go Tooling Isolation

The `website/` directory contains no `.go` files. Go's module resolution does not
descend into it. No `go.work` file is needed. `go build ./...` from the repo root
continues to work unchanged.
