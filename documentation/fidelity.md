# Fidelity guide

**gowkhtmltopdf** converts **controlled, server-generated report HTML** to PDF
and raster images. It is **not** a browser and does **not** target full WebKit
or Chrome print parity under a pure-stdlib, no-cgo design.

This guide is the product-facing fidelity story. The normative per-feature
contract is the [compatibility matrix](compatibility-matrix.md). Post-MVP work
is tracked in
[`plans/10-canonical-post-mvp-roadmap.md`](../plans/10-canonical-post-mvp-roadmap.md).

---

## Product positioning

| You need… | Expectation |
|-----------|-------------|
| Invoices, statements, multi-page tables, headers/footers, TOC, outlines | **In scope** (report engine) |
| Deterministic PDF bytes, static binary, zero third-party modules | **In scope** |
| Pixel-perfect clone of an arbitrary website | **Out of scope** |
| Full CSS (flex/grid as layout, absolute/fixed positioning) | **Partial** — flex (grow/shrink/basis/order/wrap), grid lite, relative/absolute/fixed; not full CSS3 |
| JavaScript-driven pages | **Out of scope** (`<script>` stripped; flags warn only) |
| Full Unicode / CJK typesetting | **Partial** — Type0/CID + `--font-path`; Arabic joining best-effort; no HarfBuzz (stdlib) |

**Explicit non-milestone:** full WebKit parity under pure Go stdlib only is
**not** a dated goal. For open-web screenshot quality, use a headless browser
pipeline instead of claiming this engine matches it.

---

## Tiers (what “good” means)

| Tier | Goal | Rough phases | Good means… |
|------|------|--------------|-------------|
| **Tier 1** | Solid report engine | 10–16 | Controlled HTML templates look correct in PDF/PNG; bold/italic/spacing usable; image mode not blocky 5×7 text |
| **Tier 2** | Leave wkhtmltopdf for most jobs | 17–20 | Broader CSS, pagination polish, multi-font/Unicode, HF/link edges |
| **Tier 3** | Compete on the open web | 23 deferred | Not planned as pure-stdlib product; Chrome/Playwright territory |

**As of 2026-08-04:** Tier 1 remainder (float lite, PDF image docs/`web.images`, library install + `ConvertHTML`) is closed. See the post-MVP roadmap status index.

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
**smoke only**: the file may open and paginate, but layout and non-Latin fonts
are **not** a pass criterion.

---

## Feature fidelity map (goals → status → phase)

| User-facing goal | Status (2026-08-04) | Primary phase(s) |
|------------------|---------------------|------------------|
| Typography bold/italic | **Shipped** (Liberation R/B/I/BI) | 12 |
| Typography spacing | **Shipped** (coalesce + shared advances) | 13 |
| Image mode text quality | **Shipped** (TTF outline AA, 2× supersample) | 15 |
| Invoice CSS (boxes/tables) | Implemented subset | 4, 16 expands |
| Selectors (`:nth-child`, attr, siblings) | **Shipped** | 16.1 |
| Floats / flex / position / grid | No floats; no flex/grid layout | 16–17 (grid deferred) |
| PDF images (logos/grids) | PNG/JPEG path + golden fixtures solid | 14 (docs polish remain) |
| Pagination / thead repeat | Partial (breaks yes; thead repeat no) | 5, 18 |
| Fonts / CJK / discovery | Latin family only | 12 done; 19 next for CJK |
| HF / links edges | Mostly done; known gaps | 6, 20 |
| Arbitrary URL print | Smoke only | 21 |
| JavaScript | Stripped | 22 staged |
| Open-web competition | Not planned | 23 |

---

## Failure modes (graceful degrade)

The engine should **not crash** on unsupported input:

| Input | Behavior |
|-------|----------|
| Unknown CSS property / value | Declaration ignored |
| Unsupported display (flex/grid) | Element keeps initial/default display |
| `<script>` | Stripped at load; JS flags warn only |
| Missing font family name | Falls back to embedded Liberation Sans |
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
parity, “paste any URL and get Chrome-quality print” (until tier 3 is
explicitly reopened with different constraints).

---

## Related docs

| Doc | Role |
|-----|------|
| [compatibility-matrix.md](compatibility-matrix.md) | Normative support contract |
| [samples.md](samples.md) | Fixtures and sample outputs |
| [overview.md](overview.md) | Product overview |
| [architecture.md](architecture.md) | Pipeline packages |
| [../plans/10-canonical-post-mvp-roadmap.md](../plans/10-canonical-post-mvp-roadmap.md) | Active post-MVP ledger |
| [../README.md](../README.md#deferred--not-planned) | Deferred table |
