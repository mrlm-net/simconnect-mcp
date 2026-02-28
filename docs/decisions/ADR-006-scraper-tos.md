# ADR-006: Scraper ToS and robots.txt Review

**Date**: 2026-02-23
**Status**: Accepted
**Verdict**: Scraping permitted with rate limiting and per-document attribution

---

## Context

Before implementing the build-time scraper (Phase 9, issues #25–#27), this spike reviews whether automated scraping of `https://docs.flightsimulator.com` for offline embedding in an open-source developer tool is permitted.

---

## Findings

### 1. robots.txt

`GET https://docs.flightsimulator.com/robots.txt` returns **HTTP 404**. No `robots.txt` exists. There are no `Disallow` directives and no `Crawl-delay` directives. Automated access is not restricted at the crawler-hint level.

### 2. SDK EULA

The EULA at `https://docs.flightsimulator.com/html/Introduction/SDK_EULA.htm` governs the **SDK software** (binaries, tools, simulator components). Its key clauses:

| Clause | Text (summary) | Applicability |
|--------|----------------|---------------|
| §1(b) sample content | Prohibits external distribution of sample content (audio, models, images) | Covers game assets, not text documentation |
| §1(c) add-ons | Microsoft must not be represented as endorsing add-ons | Attribution requirement |
| §2(e) redistribution | Prohibits distributing the Software itself | Covers the SDK binaries, not the documentation website |

**The EULA does not cover the documentation website.** The prohibition on redistribution refers to the SDK software, not the publicly hosted HTML reference pages. No clause explicitly restricts automated fetching of the documentation.

### 3. Copyright

The documentation is copyrighted by Microsoft Corporation and Asobo Studio. Scraping and embedding **factual technical reference material** (API names, parameter types, SimConnect variable descriptions) for use in a developer tool falls within accepted norms — analogous to a search engine cache or IDE inline documentation. The content is functional/factual rather than creative.

Community precedent: `github.com/Stephanvs/msfs-sdk-docs` hosts the SDK documentation corpus publicly without known objection from Microsoft.

---

## Decision

**Scraping is permitted** subject to the following constraints:

1. **Rate limit**: maximum **1 request/second** with a 500 ms jitter to avoid hammering the server.
2. **Attribution**: every scraped document stored in the corpus must carry a `source_url` field pointing to the canonical `docs.flightsimulator.com` page it was derived from. This attribution must be surfaced in MCP tool responses.
3. **No binary assets**: the scraper fetches text content only (HTML → plain text); no images, audio, or downloadable files.
4. **Respectful crawl**: honour any `Retry-After` headers; abort on repeated 429/503 responses.
5. **No endorsement claim**: responses must not imply Microsoft or Asobo endorses this tool (per EULA §1(c)).

---

## Consequences

- Phase 9 scraper work (#25–#27) proceeds as planned.
- The corpus schema (issues #5–#7) must include a `SourceURL string` field.
- MCP tool responses should include a `source` or `reference` field with the canonical URL.
- No manual corpus fallback is needed; the build-time scraper remains the production data source.
