# v0.2.7 LearnCpp - Phase-Wise Checklist

> **Parent:** `plans/0.2.7/README.md`
> **Status:** code wave complete 2026-09-16 - all phases closed, gates green (run 2); docs/frontend wave pending
> **Estimated effort:** 5-8 days across 11 fix agents in 4 waves plus closure
> **Depends on:** v0.2.6 tree; no new third-party dependency lands without explicit user sign-off (AGENTS.md dependency policy)
> **Unblocks:** docs and frontend follow-up wave (separate ledger, not in this file)

---

## Overview

The CLI conversion of `https://www.learncpp.com/` is the first real-world site drill for
the engine. The drill produced one fatal error, one visual regression that hides nearly
all content, and four secondary defect classes. This ledger turns the 2026-09-16
reconnaissance into ordered, atomic work. Code changes only; docs and the product site
are explicitly out of scope here.

Baseline artifacts (repo root, untracked by design):

- `learncpp_conversion.log` - failed run, exit 1, 89s, 0-byte `learncpp.pdf`
- `learncpp_noimages.log`, `learncpp_noimages.pdf` - `--no-images` run, exit 0, 37 pages
- Page renders: `/tmp/opencode/learncpp/shots/`; contact sheet `/tmp/opencode/learncpp/agentE/contact.png`
- Five reconnaissance reports: `/tmp/opencode/learncpp/agent{A,B,C,D,E}.md` (ephemeral, summarized below)
- Canonical before/after measurement: `scripts/pdf_page_forensics.py` (new 2026-09-16) writes the per-page report and JSON; baseline at `plans/0.2.7/learncpp/evidence/2026-09-16-baseline-noimages.{md,json}` (37 pages, 3096 words, 1119 link annots, LiberationSans only, 34 low-ink pages with a text layer)
- After run (2026-09-16): `learncpp.pdf` (269,184 bytes, 19 pages), `learncpp_v027.log`, `evidence/2026-09-16-v027-learncpp.{md,json}`, contact sheet `evidence/2026-09-16-v027-contact.png`

## Executive Summary

| # | Defect class | Symptom | Primary code sites |
|---|--------------|---------|--------------------|
| 1 | Image payload safety | Fatal `embed png I0: image: config: png: invalid format`; 0-byte output | `internal/layout/style_paint_props.go:339-356`, `internal/layout/background_image.go:466-491`, `internal/layout/layout_flow.go:76-112`, `internal/layout/paint.go:1568-1592`, `internal/pdf/images.go:293-314` |
| 2 | Paint z-order | 34/37 pages visually blank; text painted then covered by ancestor background | `internal/layout/paint_order.go:38-64`, `internal/layout/layout.go:1043-1052`, `internal/layout/layout_chrome.go:471-481` |
| 3 | Stylesheet base URLs | 23 font fetches 404; background images resolve to doc-relative paths | `internal/convert/prepare/styles.go:190-205,444-489`, `internal/css/css.go:63-73` |
| 4 | Font pipeline | Open Sans (woff2-only) skipped; Liberation fallback; icon fonts absent; 42KB log dump | `internal/convert/prepare/styles.go:466-489` |
| 5 | Width / flex / calc | TOC rows wrap in ~130pt columns; `calc(100% - 200px)` resolves against viewport | `internal/layout/flex.go`, `internal/layout/style_properties.go:745`, `internal/layout/style_values.go:894-948` |
| 6 | Media and page count | Print default appends `attr(href)` to 356 links (37 pages vs 12); `--no-print-media-type` is a no-op for PDF | `internal/settings/settings.go:217-235`, `internal/layout` generated content |
| 7 | PDF artifact polish | One image error loses the whole document; 1119 annots ~48% of file; empty `/Title`, no `/ID`, `/Lang` | `internal/convert/pdf_pipeline.go:285`, `internal/app/pdf.go:88-97`, `internal/pdf` |

## Evidence baseline (2026-09-16)

- Fatal chain: `.lessontable-row:nth-child(odd){background:#f4f6fd}` (custom CSS `18062.css`) stores the color as `BackgroundImage`; the fetch of `#f4f6fd` returns the 250KB homepage HTML; 187 of 189 image ops carried those bytes; `png.DecodeConfig` aborts the pipeline.
- Paint order: `main#main{z-index:1;background:#fff;border-radius:15px;box-shadow:...}` plus `article.hentry{transform: translateY(0) scale(1,1)}` (`css/10_style.css:2029-2037`) reset descendants to z=0; the flat sort paints descendant text before ancestor chrome. Verified minimal repro (Agent B): removing `z-index:1` or the transform makes text visible.
- URL base: real fonts exist at stylesheet-relative URLs (`/blog/wp-includes/fonts/dashicons.eot` -> 200) while the engine requests doc-relative paths (`/fonts/dashicons.eot` -> 404). A local two-file probe embedded the doc-relative background image instead of the sheet-relative one.
- Fonts in `learncpp_noimages.pdf`: LiberationSans (37/37 pages) and LiberationSans-Bold (29/37); no Open Sans, no icon fonts. All 356 TOC rows and 35 chapter headers are present in the text layer.
- Media: PDF defaults to print (`internal/settings/settings.go:217-235`). `--media-type screen` yields 12 pages and no `attr(href)` text; `--no-print-media-type` produced identical 37 pages (no-op).
- Width: `calc(100% - 200px)` in a 400px parent measured 518px (viewport-relative). Smart shrink ignores text ops; measured text overflow up to 14.1pt into the right margin.

---

## Phase 1: Image pipeline safety (unblocks any PDF at all)

Goal: one bad or unsupported image can never abort the document, and a color-only
`background` shorthand never becomes an image fetch.

### 1.1 Shorthand parser

- [x] `applyBackgroundImageValue` must not store a value with no image token (color-only `background`, position/size keywords only) as `BackgroundImage` (`internal/layout/style_paint_props.go:339-356`); `applyBackgroundShorthand` still extracts the color (`:331-337`).
- [x] Unit test: `.x{background:#f4f6fd}` -> BGColor set, `BackgroundImage == ""`, no image op emitted.
  - Evidence 2026-09-16 (F1): stores only image tokens (`url(`, `image-set(`, gradients) or bare non-color/keyword/length paths; explicitly rejects colors via `css.ParseColor`. Exact learncpp rule covered by a layout test (BGColor set, no fetch, no op).

### 1.2 Fetch-time validation

- [x] `resolveImage` drops payloads that are neither rasterized SVG nor PNG/JPEG (and not a format the painter supports) and warns once per src (`internal/layout/layout_flow.go:76-112`, `internal/layout/layout_images.go:528-539`).
- [x] Non-image responses (HTML error pages, `text/*`) are treated as a miss, not as zero-size images.
  - Evidence: HTML/WebP/empty -> nil ref with exactly one warning per src (cache dedupe proven by a second call); PNG kept with intrinsic size; GIF now decodes and re-encodes to PNG (`gifToPNG`) so GIF `<img>`/backgrounds work; SVG raster path unchanged.

### 1.3 Emit-site guards

- [x] Every `OpImage` emit site skips ops with no usable payload: `internal/layout/background_image.go`, `internal/layout/layout_images.go`, `internal/layout/inline_paint.go`, `internal/layout/border_image.go`, `internal/layout/layout_svg.go`.
- [x] Warn summary reports skipped image srcs without spamming one line per occurrence.
  - Evidence: 4 repeated broken backgrounds -> 1 warning, 0 ops; GIF border-image still emits its 8 cropped PNG slices.

### 1.4 Error context

- [x] Image ops carry their `src`; embed errors name the src instead of only the `I0` counter (`internal/layout/op_extra.go`, `internal/layout/layout.go`, `internal/layout/paint.go:1568-1592`).
  - Evidence: `opExtra.Src` + `Op.ImageSrc()` on all six emit sites; embed errors read `layout: embed png <src>` and `inline` for synthetic rasters; `TestDrawImageEmbedErrorNamesSrc` asserts `I0` never appears. `TestOpSizePacked` still passes (Op stays 256 bytes).

### 1.5 Tests

- [x] Tests: color-only shorthand; HTML payload skipped with warning; unsupported magic (e.g. WebP) skipped; error text contains the src.
  - Evidence: new `internal/layout/image_payload_safety_test.go` (8 tests) plus background-src cases; `go test ./internal/layout -run 'Image|Background|Border|Chrome|Paint|SVG|Filter|List|Figure|Logo|Layout' -count=1` -> ok, 161 top-level tests.

### 1.6 Closure gates

- [x] `go build ./...` exit 0; `go vet ./internal/layout` exit 0; `gofmt -l internal/layout/*.go` clean.
- [x] Conversion no longer reaches the `embed png I0` fatal on the baseline CSS fixture (targeted test; full URL rerun in Phase 8).
- [x] Size regression resolved (F5): `internal/layout/layout.go` 2501 -> 2342 lines after extracting the stacking code to `internal/layout/layout_stacking.go`; the `scripts/file-size-allowlist.txt` recorded count was updated as a deliberate change; `scripts/check-file-size.sh` exit 0.

### 1.7 Warning sink wiring (added 2026-09-16)

- [x] `layout.Options.Warnf` is wired at the `internal/convert` call sites (`bodyLayoutOpts`, TOC, HF) so skipped-image warnings reach the CLI log; test asserts one warning line per skipped src in a conversion log. Assigned to F6 with Phase 4 (different package from F5). Evidence recorded under 4.4 above (`TestSkippedImagePayloadWarnsOnceThroughRunLog`, falsification removing the body sink yielded 0 warnings).

---

## Phase 2: Stacking-context-aware paint order

Goal: ancestor chrome (background, border, shadow) always paints before descendant
content, and z ordering respects stacking-context nesting instead of a flat z sort.

### 2.1 Ordering model

- [x] Replace the flat z comparison with stacking-context-aware ordering that keeps ancestor chrome before descendant content (`internal/layout/paint_order.go:38-64`, `internal/layout/layout.go:1011-1052`, `internal/layout/layout_chrome.go:471-481`).
- [x] Keep CSS-correct stacking-context creation: a non-`none` `transform` creates a context even when it resolves to identity; do not fix this by ignoring identity transforms.
  - Evidence 2026-09-16 (F5): `paintCtxFrame` parent chain plus `paintStackingBefore` split chrome from content on the first divergent frame; flat comparison only inside one chain. Identity transform still pushes a context (`ZIndexSet`, `ZIndex==0`, chain depth 2 asserted). `blendScope`/`pushZ`/`enterStackingContext` extracted verbatim to `internal/layout/layout_stacking.go`; `layout.go` 2501 -> 2345 lines and `scripts/file-size-allowlist.txt` recorded count updated deliberately.

### 2.2 Repro coverage

- [x] Unit test replicating Agent B's minimal repro: `main` with `z-index:1;background:#fff` plus `article` with `transform: translateY(0) scale(1,1)` and 400 paragraphs; all text ops must paint after the chrome fill.
- [x] Regression test that the same document without `z-index`/transform still paints chrome first.
  - Evidence: `TestTransformedDescendantTextPaintsAfterAncestorChrome` asserts max chrome order < min text order; control document asserts nil chains and chrome first. Falsification: neutering `paintStackingBefore` fails with `chrome order 65 >= first text order 0` (the real bug); restored green. `go test ./internal/layout -count=1` ok (3.771s).

### 2.3 Fixtures and diagnostics

- [x] Record chrome nesting depth on ops for diagnosis (`internal/layout/layout_chrome.go`).
- [~] Golden fixture deliberately not added this wave (F5 instruction): the Phase 8 real-site rerun plus contact sheet is the integration proof. Deep equal-z cross-context ties still fall back to flat order as a deliberate minimal change.

### 2.4 Closure gates

- [x] Targeted `go test ./internal/layout -count=1 -parallel 2 -timeout 20m` ok (3.771s); `go build ./...`, `go vet ./internal/layout`, `gofmt -l` clean.
- [x] `bash scripts/check-file-size.sh` exit 0 (`internal/layout/layout.go` 2345, new `layout_stacking.go` 192).
- [x] Contact-sheet check on the Phase 8 rerun shows content above the pane, not hidden beneath it. Result 2026-09-16: 0 low-ink pages (was 34); pages 1, 2, and 6 inspected; sheet at `evidence/2026-09-16-v027-contact.png`.

---

## Phase 3: Stylesheet-relative URL resolution

Goal: `url()` values inside an external stylesheet resolve against that stylesheet's
URL, not the document URL.

### 3.1 Carry the base

- [x] `css.Stylesheet` carries its source base URL (`internal/css/css.go:63-73`), set when the sheet is parsed/collected (`internal/convert/prepare/styles.go:178-205`).
  - Evidence 2026-09-16 (F2): new `Stylesheet.Base` field; `addWithImports` calls `sheet.ResolveURLs(base)` (link sheets get `resourceBase(resource)`, imported sheets their own base, inline `<style>` the document base). Tests `TestFontFaceResolvesAgainstSheetBaseAcrossOrigins`, `TestCollectSheetsIgnoreBaseHref`.
- [x] The collection path keeps the base instead of using it for `@import` only (`internal/convert/prepare/styles.go:197-206`).
  - Evidence: `internal/css/url_resolve.go` (new) rewrites relative `url()` refs in rule declarations and `@font-face` src against the sheet base; skips `data:`, fragment-only, absolute URLs; `@import` stays per-importer.

### 3.2 Consumers

- [x] `@font-face` `src` URLs (`internal/convert/prepare/styles.go:444-489`) and `background-image`, `list-style-image`, `border-image-source` consumers resolve against the owning sheet base.
  - Evidence: `TestResolveURLsRewritesRelativeRefs`, `TestCollectSheetsAbsolutizesURLValuesAgainstSheetBase` (fetches the resolved value). LearnCpp shape covered: `/blog/wp-includes/css/dashicons.min.css` + `../fonts/dashicons.ttf?v=1` -> `/blog/wp-includes/fonts/dashicons.ttf`.
- [x] Inline `<style>` and `style=""` keep the document base; `<base href>` behavior is documented and tested.

### 3.3 Tests

- [x] Two-origin fixture where doc-relative and sheet-relative paths differ; assert the sheet-relative resource is fetched for both a font and a background image.
  - Evidence: doc-origin decoys get zero hits in `internal/convert/prepare/url_base_test.go`.
- [x] `@import` chains still resolve under the importer's base (existing tests stay green plus `TestImportedSheetURLsResolveAgainstImportedBase`).
- [x] Falsification: with `sheet.ResolveURLs(base)` disabled the three regression tests fail; restored and green.

### 3.4 Closure gates

- [x] `go test ./internal/css ./internal/convert/prepare -count=1` green; `go build ./...` exit 0. `internal/layout` was not touched by this phase; layout package tests run in the layout waves and full gates run in 8.3.
- [x] learncpp font URLs resolve in a local two-origin fixture (`file://` and ACL paths also re-tested via the convert suite subset).
- [x] Targeted `golangci-lint` on the two touched packages exit 0 (4 findings fixed).

---

## Phase 4: Font pipeline

Goal: supported font formats are used when available; logs stay small and honest.

### 4.1 Source selection

- [x] Extension gate parses the URL path, ignoring query/fragment, so `woff2?v=...` is skipped by policy and `ttf?x` is not misclassified (`internal/convert/prepare/styles.go:466-471`).
- [x] Multi-source `src` iteration proven: woff2 first plus ttf fallback registers the ttf (test).
  - Evidence 2026-09-16 (F6): new `fontURIPath` (url.Parse + lowercased Path); `TestFontURIPathIgnoresQueryAndFragment`; `TestFontFaceWOFF2SkippedWithTTFFallback` plain + `?v=9` subtests. Falsification: raw-suffix gate fetched `Custom.woff2?v=9`, parsed path restored green.

### 4.2 Data URIs and log hygiene

- [x] `data:` font sources decode and register when the embedded format is supported (WOFF1/TTF/OTF); unsupported ones log scheme and length only.
- [x] Replace the `styles.go:474` format/arg mismatch; add a test asserting no `%!(EXTRA` output and no log line over 1KB for a data-URI font.
  - Evidence: `TestFontFaceDataURIRegistersSupportedPayload` (woff1, ttf), `TestFontFaceUndecodableDataSkipped`, `TestFontFaceDataURILogHygiene` (no `%!(EXTRA`, no base64, every line <= 1KB), `TestDataURIMetaStopsAtPayload`. Falsification: blanket skip and the old format made the new tests red.

### 4.3 WOFF2 decoding (approved 2026-09-16)

- [x] Promote `github.com/tdewolff/font` to a third direct dependency and decode WOFF2 in the `@font-face` path so woff2-only faces register; update the `Makefile` allowlist comment and `TestDirectModuleAllowlist`; add a woff2 fixture test; re-verify learncpp Open Sans selection in Phase 8.
  - Evidence 2026-09-16 (F11): `internal/pdf/woff.go` `wOF2` branch calls `DecodeWOFF2` (`tdewolff/font.ParseWOFF2`) then the existing `ParseTTF`; `errWOFF2Unsupported` removed; `internal/convert/prepare/styles.go` now skips only `.eot`. Fixture `testdata/fonts/woff2/LiberationSans-Regular-latin.woff2` (30,576 B, sha256 `14088dcd...`, generated from the bundled Liberation TTF with `pyftsubset` + fontTools `flavor="woff2"`, commands recorded in `testdata/fonts/README.md`). Tests: fixture decode, garbage/truncated errors, prepare registration for plain/`?v=`/data-URI WOFF2; `go test ./internal/pdf ./internal/convert/prepare ./internal/convert -count=1` ok; size-check exit 0.
  - Real-font proof: learncpp Open Sans woff2 downloaded to /tmp, converted with `--allow-local-files`; `scripts/inspect_pdf_fonts.py` prints `OpenSans` and PyMuPDF sees `MTEHER+OpenSans` (3580 B subset). Full learncpp selection is re-verified in Phase 8.
  - [~] Dependency-version finding: the previously pinned `font@...-20260424...` cannot decode real WOFF2 (its own `TestParseWOFF2` fails: parse/v2 exact-EOF change). F11 pinned the upstream-fixed `font@...-20260809...`, and `go mod tidy` bumped 9 indirect modules (`parse/v2 2.8.15`, `brotli 1.2.2`, `minify 2.24.16`, `goldmark 1.8.5`, `x/image 0.44.0`, `x/net 0.57.0`, `x/text 0.40.0`, `fpdf 0.12.0`, `go-latex 0.3.0`). The alternative (old font + parse 2.8.15) keeps known decoder bugs; rejected.
  - Docs wave done 2026-09-16 (V7): AGENTS.md dependency policy now lists three direct modules (`go-text/typesetting`, `tdewolff/canvas`, `tdewolff/font`); WOFF2 claims corrected in `documentation/` (compatibility-matrix, fonts, fidelity, deferred, THREAT-MODEL, integration-security, architecture 04/08/09, README, overview, getting-started) and `frontend/src/data/content/` (page-fonts, page-compatibility, page-dossier, page-about, page-overview, page-getting-started); `make claim-scan` clean, frontend lint clean, frontend build regenerated `docs/` (13 path changes).

### 4.4 Closure gates

- [x] Targeted `go test ./internal/convert/prepare ./internal/convert -count=1` green (0.016s / 8.809s); `go build ./...`, `go vet`, `gofmt` clean.
- [x] Font selection for learncpp verified in Phase 8: Open Sans registered via WOFF2 (`BRGTJZ+OpenSans`, `SPTOVV+OpenSans` in `learncpp.pdf`); Liberation subsets remain for fallback glyphs.
- [x] Row 1.7: `bodyLayoutOpts`, TOC, and both HF layout sites wire `layout.Options.Warnf` through `line.Emit` (`internal/convert/convert.go`, `toc.go`, `hf.go`); `TestSkippedImagePayloadWarnsOnceThroughRunLog` asserts exactly one warning naming `bad.bin`, `TestSkippedImagePayloadWarnsThroughHTMLLog` covers the header path. Falsification: removing the body sink yielded 0 warnings.

---

## Phase 5: Width and flex resolution

Goal: text columns use the space the page gives them; percentages inside `calc()`
resolve against the containing block.

### 5.1 Flex free space

- [x] Flex rows give the flexible item the remaining width: `.lessontable-row` title/anchor no longer shrinks to ~130pt inside a ~508pt card (`internal/layout/flex.go`); focused test.
  - Evidence 2026-09-16 (F7): `measureCellMinMaxMode`/`measurePseudo` in `internal/layout/layout_measure.go` and the flex base/`min-width:auto` floor in `flex.go` now include `::after`. Unit row 141.25pt -> 310.50pt; on the real page title boxes 141.25pt -> 487.59pt. New `flex_lesson_row_test.go`.

### 5.2 Calc percentages

- [x] Percentages inside `calc()` resolve against the containing block, not the viewport (`internal/layout/style_properties.go:745`, `internal/layout/style_values.go:894-948`); probe case `calc(100% - 200px)` in a 400px parent must be 200px.
  - Evidence: `calc()` with `%` is deferred at cascade (`setWidthValue`) and resolved at layout (`calcLengthParts`/`calcUsedWidth`); probe 518.1px -> 150pt (200px) in a 400px parent; `calc(50% - 10px)` = 142.5pt; mixed units unchanged. `style_calc_test.go`; flex/gap parsing extracted to `style_flex_props.go` so `style_properties.go` returned under the size limit (allowlist entry removed).

### 5.3 Overflow diagnostics

- [x] Smart shrink accounts for text ops or warns when text overflows the content box by more than a tolerance (`internal/layout/layout.go:1306`, `internal/convert/convert_helpers.go:247-287`); test for the measured 7.8-14.1pt overflow.
  - Evidence: new `internal/layout/layout_census.go` `warnTextOverflow` (warning-only; `MaxContentX` unchanged, no behavior change). Real page warns `text overflows content box by 14.1pt (text right edge 552.7, content width 538.6)`, matching agent D's measured max. Tests `text_overflow_test.go` + `internal/convert/layout_warn_test.go`.

### 5.4 Closure gates

- [x] Targeted `go test ./internal/layout ./internal/convert -count=1 -parallel 2` ok (6.68s / 10.68s), `go test ./internal/html` ok; `go build ./...`, `go vet`, `gofmt -l` clean; `bash scripts/check-file-size.sh` exit 0.
- [x] Phase 8 recheck: TOC row title boxes 487.59pt (was 141.25pt) and title-then-URL order; hero `#site-text` y=37.5 static; pages 1, 2, and 6 inspected.
- [x] Hero offset fix: `#site-text` y=37.5 static (was ~458.45, the 392.6pt shift); unit tests: relative `top:50%` zero shift, absolute `top:25%` = 25pt of a 100pt CB. Deferred: `calc()` percentages for height/margin/padding and calc insets; relative `%` insets inside a definite-height CB stay static (needs a post-pass).

---

## Phase 6: Media selection and page inflation

Goal: the PDF media default and the `--no-print-media-type` flag do what the CLI
documents, and generated `::after` content paints in the correct order.

### 6.1 Flag semantics

- [x] `--no-print-media-type` selects screen media for PDF instead of being a no-op (`internal/settings/settings.go:217-235`, CLI mapping in `internal/cli`); test asserts distinct page counts on the baseline fixture.
  - Evidence 2026-09-16 (F3): `Web.PrintMediaType`/`LoadPage.PrintMediaType` became tri-state `MediaOverride` (unset/screen/print); `setMediaOverride` maps `false` -> screen; `ResolveMedia` lets an explicit override win over `media-type` in both directions (object home beats load home). Tests: `TestResolvePDFMediaFlagMatrix`, `TestPrintMediaTypeSetStoresTriState`, `TestPrintMediaTypeFlagsSelectPDFMedia`; `go test -count=1 ./internal/settings ./internal/cli` ok; binary probe: default `Hello (PRINT)`, `--no-print-media-type` and `--media-type screen` -> `Hello (SCREEN)`, `--media-type screen --print-media-type` -> `Hello (PRINT)`. Behavior change: `--print-media-type=false` now means screen (previously ignored).

### 6.2 Default decision (resolved 2026-09-16)

- [x] User decision: keep both modes first-class, no default change. PDF default stays print; screen stays selectable via `--media-type screen` and the now-working `--no-print-media-type`. Measured learncpp impact stays documented (37 pages print vs 12 screen). Both paths are covered by the F3 tests.

### 6.3 Generated content order

- [x] Print `::after { content: " (" attr(href) ")" }` paints after the element text, not before it (currently URL text precedes the title); generated-content ops ordered relative to the element's own runs; test asserting title-before-URL.
  - Evidence 2026-09-16 (F7): root cause was not paint order. `internal/html.parseTag` treated `<a href=.../>` (unquoted value ending in `/`) as self-closing, orphaning the anchor text so the URL painted first. New `selfClosingSlash` detection; spec-correct. Tests in `internal/html/html_test.go` and `TestPrintAfterURLOrderOnUnquotedHref`.

### 6.4 Closure gates

- [x] Targeted `go test ./internal/settings ./internal/cli ./internal/layout` green; `go build ./...` green. Full gates run 2: `make test`, `make lint`, `make golden` all exit 0.

---

## Phase 7: PDF artifact polish

Goal: failures never leave a 0-byte file and the writer stops shipping avoidable bulk.

### 7.1 Failure behavior

- [x] An image or layout error no longer loses the whole document: either skip-and-continue (Phase 1) or write the rendered pages and report the error afterward (`internal/convert/pdf_pipeline.go:285`, `internal/app/pdf.go:88-97`).
- [x] The output path is not truncated before conversion succeeds.
  - Evidence 2026-09-16 (F4): `internal/app/pdf.go` no longer calls `cli.OpenOutput` up front; a `lazyFileWriter` creates/truncates on first write and `checkOutputPath` still fails fast for unwritable paths or missing parents. Failure demo: the old binary truncated a 31-byte `clobber.pdf` to 0 bytes on a failed run; the new binary left it byte-identical and created no file when the path did not exist. Tests in `internal/app/pdf_test.go`.

### 7.2 Artifact size and metadata

- [x] Repeated link annotations merge or dedupe (baseline: 1119 annots, about 48% of a 432KB file) (`internal/pdf`).
  - Evidence: new `internal/pdf/annots.go` merges only identical targets with overlapping rects (never for tagged/PDF-UA documents); unit test 60 -> 20 annots and 13326 -> 4887 bytes; CLI 30-link page 89 -> 31 annots and 42996 -> 29411 bytes, annotation bytes 38.8% -> 19.7%. Cross-page links cannot merge and stay separate.
- [x] `/Title` populated from `<title>`; `/Lang` emitted when set; `/ID` and XMP follow the existing version policy (1.7+ only, pinned absent on 1.4 for legacy byte compatibility); `BaseFont` subset tags deferred.
  - Evidence: `TestTrailerIDAndXMPFollowVersionPolicy`, `internal/pdf/catalog_lang_test.go`, `internal/convert/metadata_test.go`. Explicit `Global.Title` still wins.
  - `[~]` BaseFont subset tags: promoted to row 7.5 and assigned to F9. XMP and `/ID` on PDF 1.4: user decision 2026-09-16 is to keep 1.4 output as-is (version-gated metadata stays a 1.7+/profile feature).

### 7.3 Closure gates

- [x] Targeted `go test ./internal/pdf ./internal/convert ./internal/app` green (3.461s / 10.771s / 0.021s); `go build ./...` exit 0.
- [x] `internal/app/image.go` promoted to row 7.4 and fixed by F8; no longer truncates before rendering.

### 7.4 Image-mode output safety (promoted 2026-09-16)

- [x] Apply the same lazy-open contract to `internal/app/image.go` that Phase 7.1 gave the PDF path: a failed `gowkhtmltoimage` run must not truncate an existing output file, and a missing output path must still fail fast. Mirror `lazyFileWriter` (`internal/app/pdf.go`) and add the equivalent tests. Evidence: `openLazyOutput` shared by both adapters; 4 tests plus a binary probe (seeded 36-byte file byte-identical after a failed run, no file created when absent, bad path fails fast with `app: open image output:`).

### 7.5 BaseFont subset tags (promoted 2026-09-16)

- [x] Prefix embedded subset fonts with a deterministic 6-letter subset tag (`/ABCDEF+FamilyName`) in simple fonts, Type0 dicts, CIDFont dicts, and FontDescriptor names; keep one tag per font everywhere; assert tag presence and cross-reference consistency in tests. Assigned to F9 (`internal/pdf`).
  - Evidence: tag = `'A' + sha256(subset program bytes)[i] % 26` (no time/random; per-subset); one `pdfName` string feeds simple `/BaseFont`, CIDFontType2 `/BaseFont`, Type0 `/BaseFont`, and `/FontName`. Tests prove six uppercase letters plus `+`, descriptor equality, Type0 -> CIDFont -> descriptor chain equality, and identical names across two conversions. PyMuPDF inspection shows `BDBRRV+LiberationSans` and `CYSVAY+...`; two independent generations produced byte-identical name lists. `go test ./internal/pdf -count=1` ok (3.652s). veraPDF parse PASS; the PDF/A-4 and PDF/UA-2 flavor failures are pre-existing unrelated rules (1.4 header, XMP, `/ID`, StructTreeRoot) with no font/subset findings.

---

## Phase 8: Closure - regenerate and re-verify on learncpp.com

Goal: prove the fix wave on the real URL, re-run the forensics, and close the ledger.

### 8.1 Regenerate

- [x] `make build`; then the baseline command via the harness. Result 2026-09-16: conversion exit 0, elapsed 24s (was exit 1 after 89s), `learncpp.pdf` 269,184 bytes (was 0), 19 pages, forensics exit 0. Log: `learncpp_v027.log`.
- [x] The old `learncpp_conversion.log` and the new log are both kept for before/after evidence.

### 8.2 Structural and visual re-verification

- [x] Page count, embedded fonts, text-layer integrity, and per-page visible-content coverage measured the same way as the baseline.
  - Result 2026-09-16: 19 pages (was 37); 2,531 words extracted; 371 link annots (was 1119); fonts `BRGTJZ+OpenSans`, `SPTOVV+OpenSans`, plus tagged Liberation subsets for fallback glyphs; text integrity unchanged (55 chapter headings and 310 lesson rows in both runs; the extra 120 `cpp-tutorial` occurrences are the previously orphaned anchor URLs). Reports: `evidence/2026-09-16-v027-learncpp.{md,json}`.
- [x] The 34/37 blank-page defect is gone: low-ink pages with a text layer 34 -> 0; coverage min/median/max 0.00491/0.00491/0.82021 -> 0.49192/0.80921/0.81379. Pages 1, 2, and 6 inspected directly: chrome, intro text, chapter cards, and lesson rows all visible. Contact sheet: `evidence/2026-09-16-v027-contact.png`.
- [x] `embed png I0` fatal is absent (0 matches in `learncpp_v027.log`); no `%!(EXTRA` log lines (0 matches). Remaining warnings are 12 policy/diagnostic lines: EOT and SVG-font skips, genuine site 404s, and one 12.7pt text-overflow warning.

### 8.3 Full gates (single pass, session end)

- [x] `make test` - second run TEST_EXIT=0 (first run failed only on the deliberate +86-byte pin, updated with reason).
- [x] `make lint` - second run LINT_EXIT=0: golangci-lint clean across six packages (78/78 findings fixed by six lint agents), `size-check: clean (2 allowlisted over-limit files)`, frontend ESLint + data lint clean.
- [x] `make golden` - GOLDEN_EXIT=0; all fixtures and page envelopes green (also green on the first run).
- [x] `make claim-scan` not run: no documentation surface changed in this wave (docs/frontend wave is separate).

### 8.4 Ledger and knowledge base

- [x] Close every phase row above with the command and result recorded beside it.
- [x] Update `plans/README.md` status row and this ledger's status line.
- [x] Update `knowledge-base/wiki/` in the same session: summary `summaries/learncpp-2026-09-16.md` (baseline + outcome + final gates) and a log entry.

### 8.5 Deferred items

- [x] `[~]` Docs and frontend updates for v0.2.7: completed 2026-09-16 (V7), see the Phase 4.3 evidence above; 20 source files plus the regenerated `docs/`.
- [x] WOFF2 dependency decision: approved 2026-09-16; implemented by F11 under Phase 4.3.
- [x] Media default decision: keep print default with screen selectable (Phase 6.2).
- [x] PDF 1.4 metadata decision: keep output as-is; version-gated metadata unchanged.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 1 (image safety) | Phase 8 conversion succeeds at all |
| Phase 1 (op src) | Phase 7 error reporting |
| Phase 3 (sheet base) | Phase 4 font resolution verification |
| Phase 2 (paint order) | Phase 8 visible content |
| Phase 5 (width) | Phase 8 readable line lengths |
| Phase 1-7 | Phase 8 closure gates |

Dependency policy amendment (2026-09-16): the direct-module allowlist grows from two to three modules with `github.com/tdewolff/font` (WOFF2/Brotli decoding), user-approved. Enforced by `TestDirectModuleAllowlist` (`internal/pdf/shape_test.go`) and documented in the Makefile header. Phase 4.3 records the version pin (`font@...-20260809...`) and the 9 indirect module bumps that came with it. The docs wave updates the AGENTS.md wording.

## Agent allocation (11 fix agents, one package owner at a time)

| Wave | Agent | Phase | Owned packages |
|------|-------|-------|----------------|
| 1 | F1 | Phase 1 | `internal/layout` (image path) |
| 1 | F2 | Phase 3 | `internal/css`, `internal/convert/prepare` |
| 1 | F3 | Phase 6 | `internal/settings`, `internal/cli` |
| 1 | F4 | Phase 7 | `internal/pdf`, `internal/app` |
| 2 | F5 | Phase 2 | `internal/layout` (paint order) |
| 2 | F6 | Phase 4 | `internal/convert/prepare` (after F2), `internal/convert` |
| 2 | F8 | Phase 7.4 | `internal/app` (image-mode lazy open) |
| 2 | F9 | Phase 7.5 | `internal/pdf` (BaseFont subset tags) |
| 2 | F10 | Phase 8 harness | `scripts/` only |
| 3 | F7 | Phase 5 + Phase 6.3 | `internal/layout` (width/flex and generated content order), `internal/convert` (smart-shrink helper if needed) |
| 4 | F11 | Phase 4.3 | `internal/pdf` + `internal/convert/prepare` + `go.mod`/`Makefile` (after F6 and F9) |

No two agents own the same package in the same wave. Agents run targeted tests only;
`make test`, `make lint`, and `make golden` run once in Phase 8. No git commands in
this session.

## Out of scope

- Documentation, README, `documentation/`, and frontend updates (next wave)
- WOFF2 dependency addition without sign-off
- Changing the PDF media default without sign-off
- Visual parity work beyond the defect classes above

## Validation record

| Date | Phase | Command | Outcome |
|------|-------|---------|---------|
| 2026-09-16 | baseline | `bin/gowkhtmltopdf --url https://www.learncpp.com/ -o learncpp.pdf` | exit 1, 0-byte PDF, `embed png I0` fatal, 89s |
| 2026-09-16 | baseline | `bin/gowkhtmltopdf --no-images --url https://www.learncpp.com/ -o learncpp_noimages.pdf` | exit 0, 37 pages, 34/37 visually blank, 82s |
| 2026-09-16 | 6.1 | `go test -count=1 ./internal/settings ./internal/cli` | ok; flag probe default PRINT, no-print SCREEN, media-type screen SCREEN, print-media-type beats media-type screen |
| 2026-09-16 | 3 | `go test ./internal/css ./internal/convert/prepare -count=1` + `go build ./...` | ok; falsification run fails without `ResolveURLs`, passes with it |
| 2026-09-16 | 7 | `go test ./internal/pdf ./internal/convert ./internal/app` | ok; annots 89 -> 31, 42996 -> 29411 bytes on a 30-link page; failed run no longer clobbers the output file |
| 2026-09-16 | 1 | `go test ./internal/layout -run 'Image|Background|Border|Chrome|Paint|SVG|Filter|List|Figure|Logo|Layout' -count=1` | ok, 161 tests; HTML/WebP skipped with 1 warning per src; embed errors name the src; GIF -> PNG |
| 2026-09-16 | 1.6 | `bash scripts/check-file-size.sh` | red on `internal/layout/layout.go` 2501 vs 2497; assigned to F5 |
| 2026-09-16 | baseline | `python3 scripts/pdf_page_forensics.py learncpp_noimages.pdf` | 37 pages, 3096 words, 1119 links, LiberationSans only, 34 low-ink text pages; report at `evidence/2026-09-16-baseline-noimages.md` |
| 2026-09-16 | 7.4 | `go test ./internal/app -count=1` + binary probe | ok; failed `gowkhtmltoimage` run leaves a seeded 36-byte file byte-identical, creates no file, bad path fails fast with `app: open image output:` |
| 2026-09-16 | 8 harness (F10) | `scripts/pdf_page_forensics.py ... --compare` baseline vs itself; local fixture drill | zero deltas (37/3096/1119/34, fonts identical); `real_site_drill.sh` runs a local fixture to a 1-page 14082-byte PDF, clean failure paths, `bash -n` clean |
| 2026-09-16 | 4.1-4.2 + 1.7 (F6) | `go test ./internal/convert/prepare ./internal/convert -count=1` | ok (0.016s / 8.809s); woff2 `?v=` falls back to ttf; data-URI woff1/ttf register; no `%!(EXTRA`; one warning per skipped image reaches the run log |
| 2026-09-16 | 7.5 (F9) | `go test ./internal/pdf -count=1` + PyMuPDF inspection + veraPDF | ok; all BaseFont/FontName tagged and equal, stable across runs; veraPDF parse PASS |
| 2026-09-16 | 2 (F5) | `go test ./internal/layout -count=1 -parallel 2 -timeout 20m` + `bash scripts/check-file-size.sh` | ok (3.771s); falsification fails the repro without `paintStackingBefore`; size-check exit 0, `layout.go` 2345 |
| 2026-09-16 | 4.3 (F11) | `go test ./internal/pdf ./internal/convert/prepare ./internal/convert -count=1` + Open Sans probe | ok; WOFF2 fixture + data-URI tests pass; probe embeds `MTEHER+OpenSans`; allowlist now three modules; 9 indirect bumps from the fixed font pin |
| 2026-09-16 | 5 + 6.3 (F7) | `go test ./internal/layout ./internal/convert ./internal/html -count=1` + size-check | ok; calc probe 518.1px -> 200px; flex row 141.25pt -> 310.5pt; 14.1pt overflow warning; unquoted-href self-closing parse fix restores title-then-URL order |
| 2026-09-16 | 8.1-8.2 | `scripts/real_site_drill.sh ... --baseline ...` | exit 0, 24s, 269184 bytes, 19 pages, 0 low-ink pages (was 34), OpenSans embedded, no fatal, no `%!(EXTRA` |
| 2026-09-16 | 8.3 run 1 `make golden` | `make golden` | GOLDEN_EXIT=0, all fixtures pass |
| 2026-09-16 | 8.3 run 1 `make test` | `make test` | FAIL `TestPerf3OutputBytesPin` 1420623 vs 1420537; pin updated (+86 bytes: subset tags, /Title, /Lang), page count and needles unchanged |
| 2026-09-16 | 8.3 run 1 `make lint` | `make lint` | FAIL: 78 golangci-lint findings across layout (29), convert (13), settings (11), prepare (7), app (5), pdf (13); delegated to six lint agents; frontend lint and size-check not reached |
| 2026-09-16 | 8.3 run 2 | `make test`, `make lint`, `make golden` | TEST_EXIT=0, LINT_EXIT=0 (78/78 fixed; size-check clean; frontend lint clean), GOLDEN_EXIT=0 |
