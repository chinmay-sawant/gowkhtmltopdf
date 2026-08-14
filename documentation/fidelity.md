# Fidelity guide

**gowkhtmltopdf** is an HTML template engine that converts **authored HTML** to PDF
and raster images. It is **not** a browser and does **not** target full WebKit
or Chrome print parity under a pure-Go, no-cgo design.

This guide is the product-facing fidelity story. The normative per-feature
contract is the [compatibility matrix](compatibility-matrix.md). Post-MVP work
is tracked in
[`plans/0.2.0/10-canonical-post-mvp-roadmap.md`](../plans/0.2.0/10-canonical-post-mvp-roadmap.md).

---

## Product positioning

| You need… | Expectation |
|-----------|-------------|
| Invoices, certificates, storybooks, posters, statements, tables, headers/footers, TOC, outlines | **In scope** (HTML template engine) |
| Repeatable layout, static binary, no browser process | **In scope** |
| Pixel-perfect clone of an arbitrary website | **Out of scope** |
| Wikipedia / marketing “decent print” (readable title + body) | **Progressive goal** (Phase 21) — not MVP acceptance yet |
| Full CSS (flex/grid as layout, absolute/fixed/sticky positioning) | **Partial** — flex (grow/shrink/basis/order/wrap), grid lite, relative/absolute/fixed; sticky print-scoped (page = scrollport); static 2D transforms (paint CTM); not full CSS3 |
| JavaScript-driven pages | **Out of scope** (`<script>` stripped; JS CLI flags are unknown options) |
| Full Unicode / CJK typesetting | **Partial** — Type0/CID + `--font-path`; Arabic OT via `go-text/typesetting` (GSUB) + presentation-form fallback; Indic Partial; no CGO HarfBuzz; `writing-mode` vertical is parsed but lays out horizontal |

**Explicit non-milestone:** full WebKit parity under this no-cgo design is
**not** a dated goal. For open-web screenshot quality, use a headless browser
pipeline instead of claiming this engine matches it.

**Explicit non-claims (Phase 21):** this engine does **not** claim Wikipedia
visual parity (Vector/Minerva skins, pixel layout) and does **not** claim
marketing-site pixel match. The bar is **“decent print”** below — readable
primary content, not a browser clone.

---

## Tiers (what “good” means)

| Tier | Goal | Rough phases | Good means… |
|------|------|--------------|-------------|
| **Tier 1** | Solid HTML template engine | 10–16 | Controlled HTML templates look correct in PDF/PNG; bold/italic/spacing usable; image mode not blocky 5×7 text |
| **Tier 2** | Leave wkhtmltopdf for most jobs | 17–20 | Broader CSS, pagination polish, multi-font/Unicode, HF/link edges |
| **Tier 3** | Compete on the open web | 23 deferred | Not planned as a pure-Go HTML engine; Chrome/Playwright territory |

**As of 2026-08-13:** Tier 1 closed; **Tier 2 phases 17–20 core shipped**.
Phase 21 (arbitrary URL / “decent print”) is a product contract, **not** an
acceptance pass. See [deferred.md](deferred.md) and
[performance.md](performance.md).

### CSS invoices use (phase 16)

Report templates can rely on: richer selectors (`:nth-child`, attribute, siblings), **float lite** (`float`/`clear` for logo+meta chrome), real **`inline-block`**, **`box-sizing: border-box`**, simple **`text-align: justify`**, and table-cell **`vertical-align`** top/middle/bottom. Not a full CSS2 float engine — prefer clear after chrome; complex float wrap is best-effort.

### Images in PDF (phase 14)

PNG/JPEG logos and grids are a solid path (fixtures 07/20). JPEG bytes pass through as DCTDecode; PNG alpha → soft-mask. DPI/quality CLI knobs for PDF remain ignored (honest matrix). `GlobalSettings` / `web.images=false` disables painting.

---

## How to read the matrix

Status labels in [compatibility-matrix.md](compatibility-matrix.md):

| Label | Meaning |
|-------|---------|
| **Implemented** | Parsed and consumed by layout/paint in the cases documented |
| **Partial** | Parsed and used for a subset; other cases degrade silently |
| **Not implemented** | Parsed and dropped, or not parsed; never crashes |
| **Ignored / deferred** | Flags or tags accepted for CLI compatibility without full behavior |

If a row says Implemented, there should be a code path and preferably a test
or golden fixture. Prefer the matrix over marketing prose.

---

## How we prove fidelity

| Mechanism | What it proves |
|-----------|----------------|
| Golden HTML corpus `testdata/golden/fixture-*.html` | Deterministic structure, page envelopes, feature flags (`make golden`) |
| Committed samples under `output/` | Viewer smoke artifacts (`make samples`) - **not** byte baselines |
| Unit/integration tests under `internal/*` | Layout, CSS match, PDF structure, library API |
| Visual open of PDF/PNG | Letter-spacing, table density, image-mode text quality |

Details: [samples.md](samples.md), [testdata/golden/README.md](../testdata/golden/README.md).

**Wikipedia / arbitrary URL** smoke (e.g. `output/wiki-ana-de-armas.pdf`) is
**smoke only** until Phase 21 acceptance against **vendored** fixtures: the
file may open and paginate, but layout quality and non-Latin fonts are **not**
yet a product pass criterion. See [Arbitrary websites](#arbitrary-websites-phase-21).

---

## Arbitrary websites (Phase 21)

Product goal: paste a URL (or feed static HTML from a public page) and get a
**decent print** PDF — not Chrome/WebKit parity. Work is tracked in
[`plans/0.2.0/phases/phase-21-arbitrary-websites.md`](../plans/0.2.0/phases/phase-21-arbitrary-websites.md).
Until vendored-fixture acceptance lands, treat live URL output as exploratory.
Vendored HTML and optional live smoke notes: [samples.md](samples.md).

### CSS-faithful / site-agnostic default

The engine’s **default** convert path must honor the page’s cascaded CSS
(UA → author sheets → inline), including print `@media` and `var()` custom
properties. Wikipedia and other named sites are **canaries and recipes**, not
hardwired style sources.

| Intentional policy (operator flags) | Not allowed in default cascade |
|-------------------------------------|--------------------------------|
| `--zoom`, smart-shrinking | Inventing font sizes for skin tokens (e.g. forced `8pt` for `--font-size-medium`) |
| `--simplify-dom` (landmarks) + optional `--simplify-dom-profile=mediawiki` | Forcing link underlines when CSS computes `inherit`/`none` (unless `--print-link-underline`) |
| `--use-system-fonts` / `--font-path` | Rewriting named `font-family` entries before trying the author’s stack |
| `--print-link-underline` (opt-in, default off) | Encoding `#mw-*` / `.infobox` / `.vector-body` into layout |

Cleanup ledger: [`plans/0.2.0/phases/pending-phase-items/12-css-faithful-engine.md`](../plans/0.2.0/phases/pending-phase-items/12-css-faithful-engine.md).

### “Decent print” criteria (acceptance bar)

A page meets the bar when **all** of the following hold (vendored fixtures in
CI; live Wikipedia remains optional manual smoke):

1. **Primary title** is visible early — not buried under many pages of nav /
   chrome before the article heading.
2. **Main body text** is readable across pages (multi-page OK).
3. **Reduced useless chrome** (search, appearance menus, site chrome) when
   print/simplify heuristics are enabled — heuristics are **opt-in** and
   default **off** for authored HTML templates (not shipped as of this docs
   contract).
4. **Non-Latin text:** tofu/boxes only when the configured font set is missing
   glyphs (Phase 19 fonts); missing fonts are not a layout failure by themselves.
5. **Allowed gaps:** JS widgets, sticky headers, complex grids, SPA hydration,
   and full skin CSS may still be wrong or absent.

### Explicit non-claims

| Claim | Status |
|-------|--------|
| Wikipedia visual / skin parity | **Not claimed** |
| Marketing landing pixel match | **Not claimed** |
| Full CSS / browser replacement | **Not claimed** (matrix remains Partial / Not implemented for many properties) |
| “Paste any URL → Chrome-quality print” | **Banned** until Tier 3 is explicitly reopened |

CLI URL fetch is a supported **input path** (`http`/`https`); it is **not** the
same as meeting the decent-print acceptance bar. Security when fetching URLs
or converting untrusted HTML: [cli.md](cli.md#remote-url-security),
[THREAT-MODEL.md](THREAT-MODEL.md), [integration-security.md](integration-security.md).

---

## Feature fidelity map (goals → status → phase)

| User-facing goal | Status (2026-08-13) | Primary phase(s) |
|------------------|---------------------|------------------|
| Typography bold/italic | **Shipped** (Liberation Sans/Serif/Mono R/B/I/BI + DejaVu fallback) | 12, 19 |
| Typography spacing | **Shipped** (coalesce + shared advances) | 13 |
| `text-transform` | **Shipped** (`uppercase` / `lowercase` / `capitalize`) | 16+ |
| Image mode text quality | **Shipped** (TTF outline AA, 2× supersample — **not** 5×7 as the primary path) | 15 |
| Invoice CSS (boxes/tables) | Implemented subset | 4, 16 expands |
| Lists (`ol` / `ul`) | **Shipped** — `decimal` / alpha / roman markers (not always `•`) | 4, 16 |
| Table `rowspan` | **Shipped** | 16+ |
| Table `<caption>` | **Shipped** | 16+ |
| `border-collapse` | **Shipped (lite)** — collapse sets spacing 0 and uses the grid emitter | 16+ |
| `::before` / `::after` | **Shipped** (string / `attr()` generated content) | 16+ |
| Selectors (`:nth-child`, attr, siblings) | **Shipped** | 16.1 |
| Floats / flex / position / grid | **Partial** — float lite; flex subset; grid lite; relative/absolute/fixed lite; sticky print scrollport (page content box) | 16–17 |
| PDF images (logos/grids) | PNG/JPEG path + golden fixtures solid | 14 (docs polish remain) |
| SVG-as-`<img>` | **Shipped** — rasterized via `internal/svg` then painted as PNG | 14+ |
| Pagination / thead repeat | **Shipped** breaks + thead repeat; CSS `orphans`/`widows` parsed + Rule 3 (heuristic fallback) | 5, 18 |
| Fonts / CJK / discovery | **Partial** — Type0/CID + `--font-path` / registry; Arabic OT (`go-text/typesetting`); `@font-face` **https** TTF/OTF/WOFF1 fetched via `FetchSub` (same ACL as other subresources). `.woff2` / `.eot` / `data:` skipped | 12, 19 |
| `writing-mode` vertical | **Not implemented** — `vertical-rl` / `vertical-lr` parsed but lay out **horizontal** only | 19 |
| HF / links edges | Body GoTo + HF URI + HF fragment GoTo (copies-aware) | 6, 20 |
| Arbitrary URL / “decent print” | **In progress** — product contract + docs; **acceptance not met** | 21 |
| JavaScript | Stripped | 22 staged |
| Open-web competition | Not planned | 23 |

---

## Failure modes (graceful degrade)

The engine should **not crash** on unsupported input:

| Input | Behavior |
|-------|----------|
| Unknown CSS property / value | Declaration ignored |
| Unsupported display (full Grid / unknown values) | Unknown/`display` values ignored; print-CSS-subset flex/grid are Partial (see matrix) |
| `<script>` | Stripped at load; JS CLI flags are unknown options |
| Missing font family name | Falls back through the author’s stack, then Liberation Sans (DejaVu for uncovered glyphs) |
| Missing bold face | Fake stroke bold only if face missing |
| Missing/corrupt image | Skip paint; no process crash |
| Local file without ACL opt-in | Load denied (secure default) |
| Oversized HTTP body / timeouts | Loader limits; error returned |

Security defaults: [THREAT-MODEL.md](THREAT-MODEL.md),
[integration-security.md](integration-security.md).

---

## Claims language (allowed vs banned)

**Allowed:** report-oriented, controlled HTML, wkhtmltopdf-compatible CLI
surface, pure Go, deterministic PDF, print-quality raster (relative to 5×7).

**Banned / over-claim:** pixel perfect, full CSS, browser replacement, WebKit
parity, Wikipedia visual parity, marketing pixel match, “paste any URL and get
Chrome-quality print” (until tier 3 is explicitly reopened with different
constraints). “Decent print” is allowed only with the criteria above and only
after Phase 21 acceptance against vendored fixtures.

---

## Related docs

| Doc | Role |
|-----|------|
| [compatibility-matrix.md](compatibility-matrix.md) | Normative support contract |
| [fonts.md](fonts.md) | Bundled faces, discovery, `@font-face`, shaping limits |
| [samples.md](samples.md) | Fixtures and sample outputs |
| [overview.md](overview.md) | Product overview |
| [architecture.md](architecture.md) | Pipeline packages |
| [deferred.md](deferred.md) | Deferred features and workload priority |
| [performance.md](performance.md) | Benchmarks and how to measure |
| [../plans/0.2.0/10-canonical-post-mvp-roadmap.md](../plans/0.2.0/10-canonical-post-mvp-roadmap.md) | Active post-MVP ledger |
| [../README.md](../README.md#deferred--not-planned) | Deferred table |
