# v0.2.7 LearnCpp Visual - Phase-Wise Checklist

> **Parent:** `plans/0.2.7/README.md`
> **Status:** complete 2026-09-16 - all phases closed with gates green; 3 deferred rows (`[~]`): 9.4c fixed-header chain offset, Phase 14.7 map-scan extension, and percent-height / pagination-clone edges noted in the V6b report
> **Estimated effort:** 4-6 days across 4 fix waves plus catalogue and closure
> **Depends on:** `01-canonical-0.2.7-learncpp.md` code wave (complete, gates green); `plans/0.2.6/catalog/` snapshot
> **Evidence:** `learncpp/evidence/2026-09-16-visual-diagnosis.md`, reference PNGs in the same folder
>
> **Artifact paths:** `learncpp.pdf` and `learncpp_noimages.pdf` moved from the repo root to `real-sites/learncpp/evidence/` on 2026-09-16; command lines below keep their historical output names.

---

## Overview

The v0.2.7 code wave made `learncpp.pdf` generate correctly at the artifact level
(19 pages, 0 covered pages, gates green). A follow-up picture diagnosis compared
our render against a real `wkhtmltopdf 0.12.6.1` render of the same URL and
against Chrome measurements at the same 718px CSS viewport. It found six
confirmed defects in the header branding block, the chapter badge, the HTML
parser print path, and pagination paint, plus one lower-priority glyph fallback.
One suspected defect (lesson rows side by side instead of stacked) was killed:
wkhtmltopdf's Qt WebKit ignores unprefixed `display:flex`, so our side-by-side
rows are correct.

This ledger is the canonical execution plan for those defects. It also carries
the mandatory catalogue JSON refresh (Phase 14) so the CSS catalogue keeps
matching the engine after the fixes.

## Executive Summary

| # | Defect | Class | Target (start) |
|---|--------|-------|----------------|
| 9 | Tagline collapses to a ~13pt column; screen header clips | engine | `internal/layout/layout_flow.go:1082-1088,1195-1231`, `internal/layout/layout.go:2013-2036` |
| 9b | Underline painted on a float (CSS 2.1 16.3.1 excludes floats) | engine | `internal/layout/inline_paint.go:781-820` |
| 9c | Painted `LEARN`/`C++` overlap 6.1pt (advance uses untransformed text) | engine | `internal/layout/layout_measure.go:283`, `inline_collect.go:835`, `inline_paint.go:431` |
| 10 | `Chapter` badge under-measured (56.2pt vs ~68pt), right edge 13.75pt past the card clip, white `0` invisible | engine | `internal/layout/layout_flow.go:1226-1230`, `internal/layout/overflow_clip.go:294-346` |
| 11 | Print appends `(url)` to 356 links because unclosed `<p>` never implicitly closes | engine | `internal/html` (implicit `</p>` per HTML5) |
| 12 | Page 19 row 21.2: number painted 41pt above its own row band | engine | `internal/layout/paint_pagination_fixpoint.go` |
| 13 | Missing icon font paints literal `()` for the menu glyph | data/media | icon fallback decision |
| 14 | Catalogue JSON (`mapping.json`, `coverage-summary.json`, `implemented-code-evidence.json`) refresh | process | `plans/0.2.6/catalog/`, `scripts/css-catalog-map.py` |

Measured baselines (2026-09-16, artifacts in `evidence/`):

- wkhtmltopdf reference: 16 pages, screen media, tagline one line, badge inside card.
- Our print run: 19 pages, `learncpp_noimages.pdf`; our screen run: 13 pages.
- Chrome at 718px (same media): tagline one line (155.4px screen, 180.9px print); badge box 96.4px inside the table; no URL text added in print.
- Confirmed engine rows reproduce with PyMuPDF to 0.1pt and with override probes.

---

## Phase 9: Header branding block

Goal: the branding block sizes to its text, the screen render stops clipping the
header, floats stop inheriting text decoration, and the title advance matches the
painted text.

### 9.1 Red probe

- [x] Minimal fixture with the real `#branding`/`#site-text`/`#site-title`/`#site-description` subtree and the `18062.css` header rules, served over HTTP or inline; assert `#site-text` used width.
- [x] Probe is red before the fix: `#site-text` ~13pt vs Chrome 180.9pt print / 155.4px screen. Record the number.
  - Evidence 2026-09-16 (V2): `go test ./internal/layout -run TestLearnCppBrandingTextFloatNotCappedByLogo -count=1 -v` -> before: tagline 12 line bands, painted width 13.1pt; after: 1 line, painted width 120.8pt (minimal fixture). Live print: tagline ink 125.65pt + 10.5pt margin = 136.15pt box (181.5px) vs Chrome print 180.9px (0.3% off), x0=63.35 vs Chrome 62.1.

### 9.2 Float intrinsic width

- [x] A float with text content uses text max-content when the largest descendant image is small/skipped (logo 32px); the image branch must not cap the text float (`internal/layout/layout_flow.go:1082-1088`, `:1195-1231`; `internal/layout/layout.go:2013-2036`).
- [x] Expected after: tagline on one line at 718px in both media; `#site-text` width within 5 percent of the Chrome numbers.
  - Evidence: `floatIntrinsicAvail` reserves both text max-content and descendant float image width; new `measureLargestFloatImageWidth` in `layout_measure.go`; in-flow image floats and logo-only floats keep image sizing. Four tests in new `internal/layout/learncpp_header_test.go`.

### 9.3 Regressions

- [x] Genuine image-driven floats keep their image widths: `fixture-22-float-invoice-chrome`, `fixture-29-float-beside-table`, logo fixtures.
- [x] Targeted `go test ./internal/layout -count=1` green.
  - Evidence: layout suite ok (`-parallel 2`), `go vet` + `gofmt` clean, `go test ./internal/convert -count=1` ok (golden corpus included), fixture-07/22/29 pass, no needles changed, `scripts/check-file-size.sh` exit 0.

### 9.4 Screen header

- [~] `--media-type screen` render has no clipping: title at y ~ 69pt, tagline one line; no fragments at the page top. No clipping is achieved (title y=35.7, tagline y=51.8 one line); the absolute y ~ 69pt target depends on the deferred 9.4c fixed-header chain offset. Sub-rows: 9.4a [x], 9.4b [x], 9.4c [~].
  - Status 2026-09-16 (V2): tagline is one line and the collapse is fixed, but the title still clips at y=2.35 because the relative `top:50%` on `#site-text` is not applied (the code-wave Phase 5 deferral: relative `%` insets in a definite-height containing block). Assigned to V6.
  - [x] 9.4a Apply relative percentage `top`/`left` against the containing block's definite height/width in a post-measure pass; unit test `#site-text{top:50%}` inside a 200pt header lands at 100pt shift.
    - Evidence 2026-09-16 (V6b): new `internal/layout/relative_percent.go` plus an engine field and calls in `finalizeResult`/`layout_section.go`; `applyRelativeOffset` splits length and percent insets. Focused test `TestLearnCppRelativePercentTopCentersTitle` red `shift=0.00pt, want 100pt` -> green; `go test ./internal/layout -count=1 -parallel 2` ok. Percent `top`/`bottom` resolve against the parent content box when its height is definite; `left`/`right` against content width; auto-height containing blocks keep the static position.
  - [x] 9.4b Screen render check: title near y ~ 69pt, no clipping; record before/after numbers.
    - Root cause found: the site rule carries `-webkit-transform: translateY(-50%)` and `transform: translateY(-50%)`; the engine applied both and accumulated to -100%. Fixed 2026-09-16 (orchestrator): the cascade's rest-longhand loop now skips a vendor alias when its canonical property is present (`internal/layout/style_cascade.go`), with `TestWebkitTransformAliasDoesNotAccumulateWithCanonical` (prefixed + canonical == canonical; webkit-only == canonical). Final measured screen render (fresh build, 2026-09-16): `LEARN` at y=35.7, tagline `Skill` at y=51.8 on one line (was clipped at y=2.35); print unchanged. No clipping remains.
  - [~] 9.4c New row (discovered by V6b): the absolute y ~ 69pt target also depends on the fixed-header chain offset (`#masthead #site-header-main{position:fixed}` flow); the engine renders `#branding` at y=0 while Chrome offsets that chain. Deferred as a separate defect; evidence in the V6b report and `evidence/2026-09-16-visual-diagnosis.md`.

### 9.5 Decoration on floats

- [x] Floats are excluded from `text-decoration` propagation per CSS 2.1 16.3.1 (`internal/layout/inline_paint.go:781-820`); A/B probe: print tagline strokes 12 -> 0, no underline on the tagline in either media.
  - Evidence: `decorationBlocked` veto in `style_cascade.go` `inheritProps`; decoration no longer copies to floats, absolute, inline-block, or inline-table boxes; CSS rule evaluated only when the parent paints a decoration. Live print underline strokes 12 -> 0.

### 9.6 Title advance

- [x] Red probe for the 6.1pt `LEARN`/`C++` overlap; align measured advance with the painted (uppercase) text (`layout_measure.go:283`, `inline_collect.go:835`, `inline_paint.go:431`).
- [x] After: no glyph overlap; title width matches Chrome within 1pt.
  - Evidence: trigger isolated to one title item split into face runs whose width was measured pre-transform; `emitInlineTextRun` now returns the painted advance and `emitInlineFaceRuns` advances by it (`inline.go` measures transformed rune advances). Live print: `LEARN` x1 84.81 / `C++` x0 78.72 (6.09pt overlap) -> x1 108.81 / x0 112.24 (3.43pt gap). Unit probe red at x=32.02 vs painted 44.67, green at 44.67.

### 9.7 Closure gates

- [x] `go build ./...`, targeted layout tests green, `bash scripts/check-file-size.sh` exit 0.
- [x] Header PNGs archived next to the diagnosis evidence: after-fix renders produced in Phase 15.2. Archived as `evidence/2026-09-16-v027-print-p01.png` and `evidence/2026-09-16-v027-screen-p01.png`.

---

## Phase 10: Chapter badge width and clip

Goal: the `Chapter N` pill measures to its full NBSP run and paints inside the
card clip, with the number visible.

### 10.1 Red probe

- [x] Minimal fixture: `float:right` pill with text `Chapter\xa00` inside a card with `overflow:hidden`; assert badge box width >= text + padding (expected ~68pt) and right edge inside the clip.
  - Evidence 2026-09-16 (V3): `go test ./internal/layout -run 'TestLearnCppChapterBadge' -count=1 -v` -> before: badge box right 513.75 vs clip 500.00 and a second probe `w=48.00 below text+padding 67.33`; after: right 469.75 inside clip 500.00, `w=67.33`. Root-cause correction: the 56.2pt diagnosis reading was the clipped paint rect, not the float box; the real driver was `box-sizing:inherit` being dropped by the cascade, leaving `.main` content-box and 44pt too wide.

### 10.2 Intrinsic width

- [x] The unbreakable run is honored; the pill box is not capped below max-content (`internal/layout/layout_flow.go:1226-1230`).
  - Evidence: `box-sizing:inherit` now reads the parent value (`style_properties.go`); text-branch floats use min/max content and floor at min-content per CSS 2.1 10.3.5 when `avail` is below it. Live mirror (home.html + site CSS, print, 538.5pt): badge `x=438.28 w=69.97 right=508.25`, text right 502.25; Chrome reference badge right 508.28pt (0.03pt). Image/size-containment branches unchanged.

### 10.3 Clip semantics

- [x] Text must not paint past an `overflow:hidden` clip when its background is clipped (`internal/layout/overflow_clip.go:294-346`); choose clamp for text ops or full clip parity and document it.
  - Decision: clamp-text-ops. `clipTextOp` drops whole runes whose advance boxes fall outside the horizontal clip; transformed or vertically-overlapping ops keep whole-op behavior. No ink paints past a clip; no rune is ever cut. Full clip parity (shared PDF clip path with `internal/imageout`) is out of scope. Deferred: vertical partial overlap still paints whole lines.

### 10.4 Tests

- [x] No ink past the card edge; `Chapter N` fully visible on page 1 in print and screen; regression test in `internal/layout`.
  - Evidence: new `learncpp_chapter_badge_test.go` (2 probes) and `overflow_clip_test.go` `TestOverflowClipTextOps` (hidden clamps, visible does not). `go test ./internal/layout -count=1 -parallel 2` ok; `go test ./internal/convert -count=1` ok (both golden suites, no needles changed); `go vet`/`gofmt` clean; size-check exit 0.

---

## Phase 11: HTML implicit `</p>` close

Goal: the print rule `.cryout p a::after` stops matching anchors that live inside
divs, matching browser parsing.

### 11.1 Red probe

- [x] `<p>text<div><a href=x>link</a></div>` must not match `p a::after`; count URL strings before and after.
  - Evidence 2026-09-16 (V1): `go test ./internal/html -run TestParseImplicitCloseParagraphAtBlockStart -count=1` FAIL on the current tree (`<div>` nested in `<p>`), ok after the fix.

### 11.2 Parser fix

- [x] Implement the HTML5 rule: a block-level start tag closes an open `<p>` (`internal/html`), matching browser behavior; verify against the unclosed-paragraph shape in `home.html` (6 opens, 0 closes).
  - Evidence: `internal/html/html.go` `shouldAutoClose` returns true when the open element is `p` and the start tag is in the new `closesOpenParagraph` list (address, article, aside, blockquote, details, div, dl, fieldset, figcaption, figure, footer, form, h1-h6, header, hgroup, hr, main, menu, nav, ol, p, pre, section, table, ul). New `internal/convert/paragraph_print_url_test.go`: unclosed `<p>` + lesson rows appends no URLs, a closed `<p><a>` still appends `/inside/`. Known narrowing: closes only when `<p>` is top of stack (matches the existing auto-close idiom); `<p><b><div>` keeps old nesting.

### 11.3 Print diff

- [x] learncpp print URL-text occurrences drop to the Chrome behavior (0 in the lesson table); record the page-count change (baseline 19).
  - Evidence: pre-fix `learncpp_noimages.pdf` 19 pages / 350 `(url)` suffixes; post-fix run 0 suffixes / 14 pages. The A/B convert fixture isolates the fix; the live measurement matches the direction.

### 11.4 Golden corpus

- [x] `make golden` green; any fixture that relied on the old nesting behavior is reviewed, not silently re-baselined.
  - Evidence: scan of all 63 fixtures found 0 with an unclosed `<p>` before a block tag; `go test ./internal/convert -count=1` green (includes both golden suites); no needles changed. One html test expectation updated (`TestParseVoidElements`, `<hr>` now closes `<p>` per browsers). Full `make golden` recorded in 15.3.

---

## Phase 12: Pagination paint split

Goal: a print row that moves across a page boundary keeps its badge background,
number, and title together.

### 12.1 Red probe

- [x] Minimal print document reproducing the page 19 row 21.2 shape: number at y=178.2 between bands, empty pill, title at y=222.4 (`internal/layout/paint_pagination_fixpoint.go`).
  - Evidence 2026-09-16 (V4): `go test ./internal/layout -run TestLearnCppDeprecatedRowNumberStaysWithRow -count=1 -v` (4-row table, 24 filler paragraphs, site row CSS inlined, A4/10mm). Before: number op=50 `Y=769.10` in the page gap, title op=51 `Y=809.63` on page 1, pill fill `Y=797.14 H=15.0` on page 1 (empty). After: number `Y=809.22` inside the pill band `[797.14, 812.14]` on the title's page.

### 12.2 Fix

- [x] Ops belonging to one measured row band split or shift at the same boundary; no op is left in the previous page's gap.
  - Evidence: `snapOpForward` moved only the snapped run following flow strictly below its Y plus fill chrome from `rowChromeAbove`; the badge number sits 0.4pt above the title baseline and fell between them. `rowChromeBandCandidate` now also accepts `OpText/OpBullet/OpImage/OpLinkURI` on the same band; the snap lead (`minY`) still comes only from fill/stroke chrome, so snap distance is unchanged (`paint_pagination_fixpoint.go`).

### 12.3 Tests

- [x] Unit or fixture assertion that a split row never paints its number in a different band from its background; print page 19 of learncpp re-verified in Phase 15.
  - Evidence: live `./bin/gowkhtmltopdf --media-type print --no-images --url https://www.learncpp.com/` -> 13 pages, 0 numbers without a same-line title (pre-fix same-day artifact: page 11 number at y=795.1 with chrome/title on page 12). Layout, vet, gofmt, convert golden (no needle changes), size-check all green.

---

## Phase 13: Missing icon glyph fallback

Goal: a missing icon face does not paint a literal `()` placeholder.

### 13.1 Decision

- [x] Decide and document: suppress the glyph, or keep a neutral placeholder when the icon font is absent (the no-images run has no icon font data).
  - Root cause corrected 2026-09-16 (V4): the `()` was never a glyph fallback. It was the print rule `.cryout p a::after{content:" (" attr(href) ")"}` matching href-less `#nav-toggle`/search anchors nested in an unclosed `<p>`; empty `attr(href)` paints `()`. Phase 11's parser fix removes the nesting, and the live run now has 0 `()` runs. A genuinely missing icon face keeps its rune in the display list and the PDF writer folds it to `?` (`internal/pdf/content.go:763-784` `textShowSimple`); the engine never substitutes `()`.
  - Decision: keep the rune; no layout change. A "paint nothing when the face lacks the codepoint" behavior would live at `internal/pdf/content.go` (`textShowSimple`, routing via `splitType0Runs` `:606-649`); reported, not hacked.

### 13.2 Test

- [x] A conversion without the icon font renders no literal `()`; header still readable.
  - Evidence: `learncpp_missing_icon_test.go` plus the live re-render check above (0 `()` runs).

---

## Phase 14: Catalogue JSON update (bottom of this ledger)

Goal: after Phases 9-13, the CSS catalogue still matches the engine and every
affected row carries refreshed evidence. This phase is mandatory; the ledger is
not complete without it.

### 14.1 Check

- [x] `python3 scripts/css-catalog-map.py --check` exit 0; record output.
  - Evidence 2026-09-16 (V5): `css-catalog-map: check ok (240 apply arms mapped)`, exit 0, before and after the refresh.

### 14.2 Mapping rows

- [x] Review and update the affected rows in `plans/0.2.6/catalog/mapping.json`: `float`, `clear`, `text-decoration`, `overflow`, `position` (this wave), plus the code-wave consumers whose behavior changed (`width`/`calc`, `flex-*`, `inset`/`top`, `z-index`, sheet-relative `url()` resolution), and any property whose consumer evidence moved with the layout/html changes.
  - Evidence: 20 rows refreshed; `engine_status` stayed `implemented` everywhere. Examples: `float` code_path `style_properties.go:36 -> :66` plus a min/max-content note; `text-decoration` note for the float veto (`style_cascade.go:375,398`); `overflow-x/y` note for rune-wise `clipTextOp` (`overflow_clip.go:432`); `flex*` code paths moved to `style_flex_props.go`; `width` note for containing-block `calc()`; `z-index` note for stacking-aware paint order; `box-sizing` note for `inherit`.
- [x] Move `engine_status` only where the evidence changed; do not close or demote from intent. Record per-row before/after.
  - Evidence: no status moved. `clear`, `content`, `height`, margin/padding longhands, `margin-break`, and `flex-line-count` were reviewed and intentionally untouched.
  - [x] `inset` family follow-up completed 2026-09-16 (V5b) with the refreshed code paths: `inset` -> `style_properties.go:1014`; `inset-block/-start/-end` -> `:949/:963/:965`; `inset-inline/-start/-end` -> `:956/:967/:969`; `top/right/bottom/left` -> `:234/:236/:238/:240`. Shared note records fixed lengths at build (`flex.go:1631`) and percentages post-build against the containing block (`relative_percent.go:33`). The cascade dedupe shifted 27 `-webkit-*` rows by +10 and all were re-derived with 0 bad rows. `--check` exit 0; counts unchanged at 354/0/464/0.

### 14.3 Summary counts

- [x] If any status moved, run `python3 scripts/css-catalog-map.py --write` and record the diff.
- [x] If no status moves, state that explicitly with the `--check` evidence.
  - Evidence: no status moved, `--write` never run, `coverage-summary.json` untouched at 354 implemented / 0 partial / 464 unsupported / 0 ignored.

### 14.4 Code evidence

- [x] Refresh `plans/0.2.6/catalog/implemented-code-evidence.json` rows whose `file:line` consumer evidence moved (float/layout, html parser, pagination).
  - Evidence: 332 of 354 VERIFIED rows refreshed against the current tree (flex rows re-pointed to `style_flex_props.go`, consumer reads re-derived, `font-language-override` corrected to the case-string read). Counts unchanged {VERIFIED:354, DEMOTED:3}; `updated` 2026-09-16. Six rows flagged and left as-is with the reason (`bookmark-label/level/state`, `footnote-display/policy`, `string-set`; consumed outside the layout-only scope). Refresh procedure documented in `implemented-code-evidence.md` `## Refresh procedure (2026-09-16)`.

### 14.5 Frontend catalogue pointer

- [x] `frontend/src/data/content/page-fonts.json` updated 2026-09-16 (V7 docs wave): WOFF2-supported wording, `make claim-scan` clean, frontend lint clean, `docs/` rebuilt (13 path changes). Pointer retained; no catalogue-status change.

### 14.6 Record

- [x] Every Phase 14 command and outcome recorded in the validation table below.
  - Also recorded in `plans/0.2.6/catalog/README.md`: the map script's apply-arm scan now covers 240 of 354 implemented rows because the v0.2.7 extractions moved arms out of the two scanned files.

### 14.7 Known scan limitation

- [~] `scripts/css-catalog-map.py` scans only `style_properties.go` and `style_cascade.go`; after the v0.2.7 extractions it sees 240 of 354 implemented arms. Documented in the catalog README; extending the scan set is a follow-up, not part of this wave.

---

## Phase 15: Closure - re-verify against wkhtmltopdf

### 15.1 Regenerate

- [x] `make build`; regenerate `learncpp.pdf` and `learncpp_noimages.pdf` with logs (harness `scripts/real_site_drill.sh` when usable, else the documented commands).
  - Result 2026-09-16: print run exit 0 in 7s, `learncpp.pdf` 226,441 bytes / 13 pages (was 19 pages before this wave); no-images 224,780 bytes / 13 pages; screen render 399,984 bytes / 13 pages. Logs: `learncpp_v027.log`, `learncpp_noimages.log`.

### 15.2 Picture check

- [x] Page 1 print and screen: tagline one line, no clipping, badge inside the card with `Chapter N` visible, no `(url)` spam in print, no literal `()`.
  - Result: screen header `LEARN` y=35.7 / `Skill` y=51.8 on one line (was clipped at y=2.35); badge `Chapter 0`/`Chapter 1` inside the cards; 0 URL suffixes and 0 literal `()` across print, no-images, and screen; forensics low-ink pages 0 of 13. Pages inspected directly.
- [x] Page 19 (or wherever Appendix D lands): row 21.2 number inside its pill.
  - Result (V9): print p13 `21.2` rect y 553.7-567.3 (center 560.5) inside fill band y 550.6-570.4 with the title; screen p12 center 545.1 inside band 536.0-554.1. Sweep: all 310 print + 310 screen row numbers sit inside a row band, 0 misses. Old defect was number y=178.2 vs band start 219.3.
- [x] Re-render the wkhtmltopdf reference and record the same measurements as the baseline (tagline width, badge rect, page count).
  - Result (V8): tagline one line 125.65pt vs the council's predicted 125.7pt (0.05pt); badge right edge 536.68 print / 539.78 screen, digit 6.0pt inside the pill, versus the diagnosed clip at 566.9 with the digit 0.4pt outside. The wk `/tmp` reference files were no longer on disk, so the archived numbers and page-1 PNG in `evidence/` were used. One cosmetic leftover: an 18pt underline stroke under the screen menu icon.

### 15.3 Gates

- [x] `make test`, `make lint`, `make golden` all exit 0; record exact outcomes.
  - Result 2026-09-16 (post-lint tree): TEST_EXIT=0, LINT_EXIT=0 (golangci clean, size-check clean with 2 allowlisted files, frontend eslint + data lint clean), GOLDEN_EXIT=0. Lint cleanup itself: 38/38 layout findings (L7) + 3/3 html (L8); one bookkeeping flag: `internal/layout/layout.go` allowlist count moved 2342 -> 2349 for V6b's in-flight growth, deliberate and reported.
- [x] `python3 scripts/css-catalog-map.py --check` exit 0 (Phase 14 rows).
  - Result: exit 0 (`check ok (240 apply arms mapped)`) after the V5 refresh; inset/top follow-up (V5b) in flight.
- [x] Artifact-to-code alignment after lint cleanup: fresh render from the final tree is content-identical (8 bytes differ overall, no byte-determinism claim exists); text, pages, fonts, and links match the verified artifacts.

### 15.4 Ledger and knowledge base

- [x] Close every row above with command + result; update `plans/0.2.7/README.md` and `plans/README.md` status.
- [x] Update `knowledge-base/wiki/` in the same session (summary + log).
- [x] No git commands unless the user asks: none were run in this wave.

---

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 9 (float width) | Phase 10 (same measurement path), Phase 15 |
| Phase 11 (parser) | Phase 15 print page count |
| Phases 9-13 | Phase 14 catalogue refresh |
| Phase 14 | Phase 15 closure gates |

## Agent allocation (one package owner per wave)

| Wave | Agent | Phase | Owned packages |
|------|-------|-------|----------------|
| A | V1 | Phase 11 (done) | `internal/html` |
| A | V2 | Phase 9 (done except 9.4) | `internal/layout` (header/float/decoration) |
| B | V3 | Phase 10 | `internal/layout` (badge/clip) |
| C | V4 | Phase 12 + Phase 13 | `internal/layout` (pagination, glyph fallback) |
| C | V6 | Phase 9.4 | `internal/layout` (relative % insets, after V4) |
| D | V5 | Phase 14 | `plans/0.2.6/catalog/` + `scripts/css-catalog-map.py` |
| E | orchestrator + 2 verify agents | Phase 15 | build, regeneration, gates, ledger |

`make test`, `make lint`, and `make golden` run once in Phase 15. Targeted
package tests only during waves. No git commands in this session.

## Out of scope

- Docs/frontend wave (WOFF2 claims, dependency wording) except the Phase 14.5 pointer.
- The killed claim: lesson rows side by side. wkhtmltopdf's Qt WebKit lacks unprefixed `display:flex`; our side-by-side rows match Chrome. Do not "fix" this.
- wkhtmltopdf reference artifacts stay in `/tmp` unless archived as evidence.

## Validation record

| Date | Phase | Command | Outcome |
|------|-------|---------|---------|
| 2026-09-16 | diagnosis | `wkhtmltopdf https://www.learncpp.com/ learncpp_wk.pdf` | exit 0, 7s, 16 pages (screen media reference) |
| 2026-09-16 | diagnosis | `bin/gowkhtmltopdf --media-type screen --url ...` | exit 0, 7s, 13 pages, Open Sans embedded |
| 2026-09-16 | diagnosis | council Analyst/Interpreter/Critic | 6 confirmed defects, 1 claim killed, 1 needs probe; archived in `evidence/2026-09-16-visual-diagnosis.md` |
| 2026-09-16 | 11 (V1) | `go test ./internal/html ./internal/convert -count=1` | ok; red probe red -> green; print URL suffixes 350 -> 0, pages 19 -> 14 |
| 2026-09-16 | 9 (V2) | `go test ./internal/layout ./internal/convert -count=1` | ok; tagline 12 lines -> 1 (181.5px vs Chrome 180.9px); title overlap 6.09pt -> 3.43pt gap; underline strokes 12 -> 0; print pages 14 -> 13; 9.4 screen clip still open (assigned V6) |
| 2026-09-16 | 10 (V3) | `go test ./internal/layout -run 'TestLearnCppChapterBadge'` | red -> green; badge right 513.75 -> 469.75 inside clip 500; live mirror right 508.25 vs Chrome 508.28; root cause `box-sizing:inherit` dropped + float min-content floor |
| 2026-09-16 | 14 (V5) | `python3 scripts/css-catalog-map.py --check` + `python3 -m json.tool` | ok; 20 mapping rows + 332 evidence rows refreshed, counts unchanged (354/0/464/0); inset/top follow-up pending |
| 2026-09-16 | 12+13 (V4) | `go test ./internal/layout -run TestLearnCppDeprecatedRowNumberStaysWithRow` | red -> green; number Y=769.1 gap -> Y=809.22 inside pill band; live 13 pages, 0 stray numbers, 0 `()` runs; Phase 13 root cause corrected (empty `attr(href)`, not glyph fallback) |
| 2026-09-16 | 9.4 (V6b) | `go test ./internal/layout -run 'TestLearnCppRelativePercentTopCentersTitle'` + package run | red `shift=0.00 want 100pt` -> green; layout package ok 5.08s; vendor alias accumulation fixed with `TestWebkitTransformAliasDoesNotAccumulateWithCanonical`; 9.4c fixed-header offset deferred |
| 2026-09-16 | 15.2 verify (V8) | PyMuPDF page-1 checks on print / no-images / screen | tagline 1 line 125.65pt (predicted 125.7); title gap +3.43pt; `Chapter 0` digit 6.0pt inside; 0 URL/`()`; no new artifacts |
| 2026-09-16 | 15.2 verify (V9) | forensics + row-band sweep + text integrity on 3 PDFs | 0 covered pages; all 310 row numbers inside a band (21.2 fixed); 55 headings / 310 rows in all three; fonts subset-tagged with FontFile2; print vs no-images text identical |
| 2026-09-16 | 15.3 lint cleanup (L7/L8) | scoped golangci-lint + package tests + size-check | 38/38 layout + 3/3 html fixed; layout ok 7.0s, html ok 0.012s; `layout.go` allowlist 2342 -> 2349 for V6b growth (deliberate, flagged) |
| 2026-09-16 | 15.3 gates (final tree) | `make test`, `make lint`, `make golden` | TEST_EXIT=0, LINT_EXIT=0, GOLDEN_EXIT=0 |
| 2026-09-16 | 14 follow-up (V5b) | `python3 scripts/css-catalog-map.py --check` + `python3 -m json.tool` | ok; 11 inset/top rows + 27 cascade-alias rows re-pointed (0 bad), counts unchanged 354/0/464/0 |
