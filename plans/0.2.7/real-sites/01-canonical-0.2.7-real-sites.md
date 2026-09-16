# v0.2.7 Real-sites - Phase-Wise Checklist

> **Parent:** `plans/0.2.7/README.md`
> **Status:** fix wave complete 2026-09-16. Findings: 60 (35 engine defects, 8 probes, 17 reference artifacts). Fixes landed: 30 engine defects and 3 probes, plus the cross-site empty-cmap blocker; 8 rows deferred with reasons (`[~]`); reference artifacts registered (no action). Gates: `make lint` exit 0, `make test` exit 0, `make golden` exit 0 (71 pass), catalogue check ok, claim-scan clean.
> **Estimated effort:** delivered in one day with 10 agents across 5 fix waves plus 3 gate agents
> **Depends on:** [`../learncpp/02-canonical-0.2.7-learncpp-visual.md`](../learncpp/02-canonical-0.2.7-learncpp-visual.md) (method), `scripts/pdf_page_forensics.py`, `wkhtmltopdf 0.12.6.1`, Chrome headless
> **Companion:** [`README.md`](README.md) (wave index, method, validation record)
> **Evidence:** per-site folders in this directory. `evidence/` paths below are relative to `real-sites/`. Post-fix artifacts: `evidence/2026-09-16-forensics-gowk-after.{json,md}` in each site folder.

---

## Overview

On 2026-09-16 seven real sites were converted with `bin/gowkhtmltopdf` and with
`wkhtmltopdf 0.12.6.1`, then compared by one read-only sub-agent per site using the
LearnCpp picture-diagnosis method. The wave produced 60 findings. A fix wave then
landed the engine defects across `internal/layout`, `internal/pdf`, and
`internal/svg`, re-rendered all seven sites plus `w3schools.com/cpp`, and closed
the gates. Reference artifacts stay registered in Phase 9 so nobody chases them.

Artifacts: every site PDF (gowk output and wk reference) lives in that site's
`evidence/` folder, moved there from the repo root on 2026-09-16; the earlier
LearnCpp PDFs live in `real-sites/learncpp/evidence/`. Root `*.pdf` and `*.log`
plus everything under `real-sites/*/evidence/` (PDF, JSON, PNG, Markdown, probe
HTML/CSS, logs, text dumps) are ignored in `.gitignore`; probe fixtures live in
each site's `evidence/2026-09-16-probes/` folder and must be generated there only.

Known comparison caveats that shaped Phase 0:

- `wkhtmltopdf` defaults to **screen** media (`--no-print-media-type` is its default),
  while `bin/gowkhtmltopdf` defaults to print. Matched print-media references were
  rendered for all six convertible sites and live in
  `/tmp/opencode/real-sites/wk-print/`.
- `wkhtmltopdf 0.12.6.1` segfaults (exit 139) on the Wikipedia page in all five
  configurations tried; Chrome print was the fallback reference there.
- GeeksforGeeks is JavaScript/bot walled: both engines produce shells. After the
  fix wave the gowk shell is blank, which matches the JS-off wk artifact.

## Executive Summary

**Fix wave outcome (2026-09-16).** New artifacts regenerated with the fixed binary;
old numbers from `evidence/2026-09-16-forensics-gowk.json`, new from
`evidence/2026-09-16-forensics-gowk-after.json`.

| Site | Pages | Words | Links | Result highlights |
|------|-------|-------|-------|-------------------|
| ana-de-armas | 22 -> 22 | 7477 -> 7479 | 954 -> 978 | 0 corrupted URIs (was 31), 32 percent-encoded en dashes, quotes painted, citation superscripts raised, image/logo links restored |
| gobyexample | 3 -> 3 | 231 -> 231 | 95 -> 95 | h2 regular (Bold face gone), intro keeps `upgrade to` on one line |
| learn-cpp-org | 2 -> 2 | 177 -> 181 | 23 -> 23 | button whitespace widths fixed, fixed-dock label kept on every page |
| cplusplus-tutorial | 4 -> 2 | 152 -> 127 | 30 -> 30 | hidden sidebar no longer flows or paints (2 phantom pages gone, -25 marker words), breadcrumb visible |
| programiz-cpp | 15 -> 10 | 1387 -> 1010 | 196 -> 31 | accordion collapses, placeholder paints, page-1 annots 0 -> 8 |
| tutorialspoint-cpp | 9 -> 8 | 1629 -> 1629 | 78 -> 79 | 1219 stray 1pt borders -> 104 drawings, bold chars 59 -> 393, 102 U+2022 discs, Tutorix annot restored |
| geeksforgeeks-cpp | 6 -> 6 | 78 -> 0 | 0 -> 0 | shell now blank like wk (site-side JS wall); no engine action |
| w3schools (cross-site) | none -> 7 | 0 -> 729 | 0 -> 342 | empty-cmap fallback: was `embed font F1` fatal, now a 469 KB PDF |

Findings ledger:

| Site | Findings | Fixed | Deferred | Reference/data |
|------|----------|-------|----------|----------------|
| ana-de-armas (Wikipedia) | 12 | 6 | 4 probes | 2 |
| cplusplus-tutorial (cplusplus.com) | 7 | 7 | 0 | 0 |
| geeksforgeeks-cpp | 7 | 0 | 2 probes | 4 artifacts |
| gobyexample | 6 | 2 | 0 | 4 |
| learn-cpp-org | 10 | 4 | 2 + 1 probe | 3 |
| programiz-cpp | 10 | 8 | 1 partial + 1 probe | 0 |
| tutorialspoint-cpp | 8 | 4 | 0 | 4 |
| w3schools (cross-site) | 1 | 1 | 0 | 0 |
| **Total** | **61** | **32** | **1 partial, 8 probes/items** | **17** |

| Phase | Area | Status |
|-------|------|--------|
| 0 | Matched-media re-baseline | renders done; print refs available |
| 1 | Hidden and zero-size boxes flow and paint | fixed |
| 2 | Width and line measurement | fixed except flex/grid percentage columns (deferred) |
| 3 | Borders, quotes, control glyphs | fixed |
| 4 | Fonts, weights, font-face failure | fixed; two probes deferred |
| 5 | List markers and pseudo content | fixed |
| 6 | Replaced content and SVG | fixed; sprites/object-fit/ICO deferred |
| 7 | Link annotations and URI encoding | fixed; hidden-anchor policy probe deferred |
| 8 | Pagination, fixed chrome, form controls, tables | fixed; Programiz header block partial, two probes deferred |
| 9 | Reference-artifact registry (no action) | registered |
| 10 | Closure gates | green |

## Phase 0: Matched-media re-baseline

- [x] Re-rendered the six non-Wikipedia wk references with print media: `wkhtmltopdf --quiet --print-media-type <url> <slug>_wk_print.pdf`; all six exit 0 (gobyexample 36,149 bytes; learn-cpp-org 177,071; cplusplus-tutorial 39,161; programiz-cpp 126,622; tutorialspoint-cpp 200,232; GeeksforGeeks 34,704 with JS off).
- [~] Re-ran the media-sensitive verdicts against the print references only where the fix wave touched them. Reason: the fix wave verified behavior with targeted tests and post-fix artifact checks rather than a second full verdict pass. Next gate: run the full per-site verdict diff against `/tmp/opencode/real-sites/wk-print/` before the next fidelity push.
- [~] Wikipedia Chrome A4 re-render. Reason: not needed for the fixes; the Letter render plus per-page checks sufficed. Next gate: rerun with a page-size flag when pagination parity becomes a target.
- [x] GeeksforGeeks same-UA/JS-state rerun: JS-off wk print ref rendered (34,704 bytes); post-fix gowk shell is blank, matching. `GFG-CPP-04` and `GFG-CPP-07` remain probes in Phase 7 and Phase 9.
- [x] Recorded the re-baseline commands in [`README.md`](README.md) validation record.

## Phase 1: Hidden and zero-size boxes

- [x] `visibility:hidden`/`height:0` subtrees contribute zero used height. `TestVisibilityHiddenHeightZeroSidebarCollapses` (sidebar 50.60 -> 0.00pt; AFTER-LEFT 75.62 -> 25.02 == control); live cplusplus.com 4 -> 2 pages.
- [x] Hidden boxes emit no fill/stroke and cannot cover text. `TestHiddenSidebarDoesNotCoverBreadcrumb`; live page 1 starts `Tutorials : C++ Language` with ink.
- [x] `height:0` + `overflow:hidden` collapses. `TestHeightZeroOverflowHiddenCollapses` (40.00 -> 0.00pt, child ink gone); live Programiz 15 -> 10 pages.
- [x] Live re-check: regenerated artifacts under `evidence/2026-09-16-forensics-gowk-after.*`.
  - Implementation: definite heights cap content per CSS 2.1 10.6.3 (`applyHeightConstraintsWithCB`); zero-area clips deactivate ops; hidden chrome/background/image/SVG/table/multicol paint guards.

## Phase 2: Width and line measurement

- [x] Collapse/trim whitespace in inline-level form controls: `TestInlineBlockNowrapWhitespaceDoesNotInflateWidth` (69.451 -> 34.014pt); learn-cpp.org words 177 -> 181.
- [x] Percentage columns in flex/grid rows: NOT fixed. Deferred `learn-cpp-org-3` (`.col-3` collapses to an 11-15pt column). Next probe: fixture `.row > .col-3 + .col-9` asserting 25 percent of the row; suspects `internal/layout/flex.go`, `layout_flow.go`.
- [x] Breadcrumb bar shrink-to-fit (`cplusplus-3`): resolved by the Phase 1 hidden-sidebar fix; live page 1 renders the breadcrumb on one line and the phantom pages are gone.
- [x] Line fit excludes the trailing space: `TestLineFitExcludesTrailingSpace`; live gobyexample keeps `upgrade to` on one line (both words y=239.2). Probes: `gobyexample/evidence/2026-09-16-probes/probe-wrap-420.html` (keeps `to`) through `probe-wrap-424.html` (sweep).
- [x] Pseudo-element trailing space: `TestGeneratedContentTrailingSpaceSeparatesInlineBlocks` (gap -86.074 -> +3.334pt).
- [x] `vertical-align: super` raises the run: `TestSupElementRaisesRunBaseline` (23.360 -> 19.362pt origin, 12.0 -> 9.996pt size). Residual: the raise uses 0.4em of the run size vs Chrome's ~0.48em parent-em; acceptable, next probe is parent-em parity.
- [x] Trailing citation cluster no longer overflows: `TestSupCitationClusterDoesNotOverflowContentBox` (28.858pt overflow -> 0).

## Phase 3: Borders, quotes, control glyphs

- [x] Explicit `border: 0` stays zero: `TestBorderZeroEmitsNoStrokes` (1.00 -> 0.00pt, active strokes 12 -> 4); live TutorialsPoint drawings 1219 -> 104.
- [x] Missing double quotes paint: `TestUAQuoteContentForQ` plus the UA `q::before/after` open-quote/close-quote funnel; live Wikipedia text contains the quote glyphs. Probes: `ana-de-armas/evidence/2026-09-16-probes/q-probe-{a,b,c}.html`.
- [x] `\r` is a line break, never U+FFFD: `TestPreCarriageReturnIsLineBreak`.

## Phase 4: Fonts, weights, font-face failure

- [x] `font: inherit` expands to weight/style/size/line-height/family inherit: `TestFontShorthandInheritExpandsLonghands`, `TestFontShorthandInheritBeatsUABold`, `TestFontShorthandFormsStillExpand`; live gobyexample h2 is regular and the Bold face is gone from the artifact. Probe: `gobyexample/evidence/2026-09-16-probes/probe-font-inherit.html`.
- [x] `font-weight: bolder` resolves per CSS Fonts 3: `TestResolveFontWeightRelativeKeywords`; live TutorialsPoint bold chars 59 -> 393.
- [x] Weight-matching face selection: `TestRegistryLookupSelectsFaceByWeight`, `TestWeightClassParsesShortOS2Table` (86-byte version-1 OS/2 tables were read as 400); one subset at 700 -> three at 400/500/700.
- [x] Empty-cmap face no longer aborts the write: `TestFontEmptyCmapEmbedFallsBackToLiberation`; live `w3schools.com/cpp` converts (7 pages, 469,491 bytes, 729 words) and its PDF is archived in `w3schools/evidence/`.
- [~] Droid Sans Mono `@font-face` probe (`programiz-9`). Reason: not attempted in this wave; code remains on LiberationMono. Next gate: fixture with the exact face, then bisect fetch vs registry.
- [~] FontAwesome pseudo-element glyphs (`learn-cpp-org-5`). Reason: not attempted; requires pseudo content plus face alias resolution. Next gate: `.fa-play::before{content:"\f04b"; font-family:"Font Awesome 5 Free"; font-weight:900}` fixture.
- [~] Deferred cosmetic seam: `internal/convert/prepare/styles.go:458` overwrites each `@font-face` PostScript name with the family name; subsets are fingerprint-distinct but share a BaseFont stem. Next gate: assert `/BaseFont` carries the real PS name per weight.

## Phase 5: List markers and pseudo content

- [x] Bullet fold: U+2022-family markers map to the WinAnsi disc glyph (0x95), not U+00B7. `TestWinAnsiFoldBulletsKeepDisc`, `TestWinAnsiDecodePunctuationBlock`, `TestBulletPaintsDiscAndToUnicode`; live TutorialsPoint 102 U+2022 discs and 0 middots, learn-cpp.org 20 discs and 0 middots. Probe: `tutorialspoint-cpp/evidence/2026-09-16-probes/bullet.html`.
- [x] Extra visible middots before cplusplus.com content links: gone (words 152 -> 127, exactly the 25 markers); `TestOutsideListMarkerClippedAtPageEdge` covers the clip edge. Probe: `cplusplus-tutorial/evidence/2026-09-16-probes/marker-probe.html`.
- [~] Wikipedia See-also bullet size/position probe (`ana-de-armas-8`). Reason: marker glyph changed by the fold fix; residual size/position difference unmeasured. Next gate: measure the U+25AA ink box vs the Chrome crop in `ana-de-armas/evidence/`.

## Phase 6: Replaced content and SVG

- [x] SVG gradient paint servers resolve: `TestRasterizeResolvesForwardReferencedLinearGradient` (objectBoundingBox and userSpaceOnUse), `TestRasterizeResolvesLooseGradientOutsideDefs`, `TestRasterizeResolvesForwardReferencedRadialGradient`, `TestRasterizeResolvesLowercasedInlineGradient`; Programiz logo pixels `(0,0,0)` -> `(101,1,229)` / `(4,152,236)`.
- [x] Inline SVG wordmark and burger (`cplusplus-5`, `cplusplus-6`): `TestRasterizeNormalizesLowercasedRootViewbox` (the HTML pipeline lowercases `viewBox`) and `TestRasterizeFallsBackForUnresolvableFontFamily`; burger 96x64 white blob -> 96x96 bars (dark fraction 0.615), wordmark raster 480x144.
- [~] External sprite `<use>` and `object-fit`/`object-position` (`programiz-8`, `GFG-CPP-05`). Reason: needs layout-side sprite resolution plus object-fit support; deferred. Next gate: HTTP-served `sprite.svg#search` fixture plus an object-fit sprite fixture.
- [~] ICO favicon (`learn-cpp-org-7`). Reason: not attempted; decode support only. Next gate: small ICO with embedded PNG payload.
- [~] Stray 1x1 spacer image (`ana-de-armas-11`). Reason: low visual impact; not traced. Next gate: grep the HTML for data-URI/1x1 images and assert hidden or zero-area images do not paint.

## Phase 7: Link annotations and URI encoding

- [x] URI percent-encoding: `TestURIStringPercentEncoding` (5 rows), `TestLinkURIAnnotPercentEncoded`; live Wikipedia 0 corrupted URIs (was 31), 0 raw non-ASCII, 32 percent-encoded en dashes. Bonus: `pdfDocEncodingFold` dash bytes corrected to em/en dash (0x84/0x85).
- [x] Image-only anchors emit annotations: `TestImageOnlyAnchorEmitsLinkOp`; root cause was blockified anchors (`display:block/inline-block`, flex items, floats) losing hrefs, fixed via `enclosingAnchorHref` stamping. Live: Wikipedia File: annots 2 -> 7 plus the logo `Main_Page` link; TutorialsPoint Tutorix 0 -> 1.
- [x] Page-1 annotations: `TestFirstPageAnchorEmitsURIAnnot`; live Programiz page-1 annots 0 -> 8 (was 7 in the earlier agent probe against the site capture).
- [~] Hidden-anchor policy probe (`GFG-CPP-04`). Reason: GeeksforGeeks shell has no content anchors; policy undecided. Next gate: `visibility:hidden` vs `display:none` anchor fixtures and a documented decision.
- [x] Face-keyed rune unions (bonus, from the first pdf agent's handoff): `TestFontRuneUnionKeyedByFace`; two-page A/B probe no longer mixes runes across faces sharing a resource name.

## Phase 8: Pagination, fixed chrome, form controls, tables

- [x] `page-break-inside: avoid` on `pre`: `TestPreAvoidInsideMovesWholeToNextPage`.
- [x] Fixed-chrome labels on every page: `TestFixedFooterLabelPaintsOnEveryPage`.
- [~] Fixed dock repetition/overlap probe (`learn-cpp-org-9`). Reason: repeat semantics unchanged; label fix landed, overlap probe not run. Next gate: Chrome A4 print of the live page vs the post-fix artifact.
- [x] Form controls: `TestInputPlaceholderPaintsTextOnce`, `TestInputValueBeatsPlaceholder`, `TestInputPlaceholderInsideContentBox`, `TestInputNonTextTypesSkipPlaceholder`; live Programiz header paints `Search` and field backgrounds. `programiz-1` is PARTIAL: a 369x26pt `#0556f3` fill remains at the header (wk print has no wide blue fill). Next gate: A/B the real header subtree with `.get-app-link-wrapper{display:none}` and an explicit field background; archived variants: `programiz-cpp/evidence/2026-09-16-probes/programiz-{hide,mark}.html`.
- [~] Wikipedia infobox line-gap probe (`ana-de-armas-12`). Reason: not attempted. Next gate: infobox-style cell fixture with `<br>`, second line within 1pt of Chrome.
- [~] Wikipedia anniversary globe probe (`ana-de-armas-7`). Reason: not attempted; may be a print `display:none` or `<picture>`/srcset selection. Next gate: fetch the logo markup and print rules.
- [x] Golden corpus follows the fixed pagination: `fixture-61-implemented-props-b.html` envelope [5, 8] -> [5, 9] (fresh conversion verified 9 pages; repeated-thead continuation moved the `left` row to page 8 and the footer to page 9); `fixture-60` moved 8 -> 7 within its [7, 9] envelope. No needles changed.

## Phase 9: Reference-artifact registry (no action)

Kept from the findings wave; no change. These are `[~]` by design: the reference
or setup is the outlier and matching it would be wrong.

- [~] `gobyexample-3/4/5/6`: wk smart-shrink 0.8 page fit, NimbusSans faces, dropped h2 margin, annotation shape merging.
- [~] `tutorialspoint-cpp-5/6/7/8`: print-vs-screen page count and body size, print CSS hiding site chrome, wk background rasters, wk text-extraction spacing.
- [~] `learn-cpp-org-1/8/10`: screen-vs-print setup, wk href double-encoding and 10pt banner hotspot, font substitution.
- [~] `GFG-CPP-01/02/03/06/07`: JS/bot wall on both sides, literal site text, blank JS-off wk artifact, page-count noise, UA/JS confound.
- [~] `ana-de-armas-9/10`: wkhtmltopdf 0.12.6.1 segfault (exit 139, five configs) and A4-vs-Letter page count.

## Phase 10: Closure gates

- [x] Fix rows closed with named tests and regenerated artifacts; the 8 deferred rows carry reasons and next gates above.
- [x] Catalogue JSON untouched and consistent: `python3 scripts/css-catalog-map.py --check` exit 0 (240 apply arms mapped); the wave changed no CSS property coverage, so no refresh was required.
- [x] `make lint` exit 0 (golangci-lint, size-check with 2 allowlisted files, frontend eslint + data lint). `make test` exit 0. `make golden` exit 0 (71 pass, 0 fail). `make claim-scan` clean.
- [x] `plans/0.2.7/README.md`, `plans/README.md`, the wave README, and `knowledge-base/` updated in the same session; KB fixture summary corrected for `fixture-61`.
- [x] No git commands run by the orchestrator. Disclosure: one test-gate agent ran a read-only `git status` by mistake; it changed nothing and no other git command ran.
- [~] `make samples` regeneration deferred: committed `output/fixture-61-implemented-props-b.pdf` is still the 2026-09-12 8-page sample while the fixture now renders 9 pages; other samples may have moved within envelopes. Next gate: run `make samples` and review the sample diff in a dedicated housekeeping change.

## Dependencies

| Depends on | Provides to |
|------------|-------------|
| Phase 1 (hidden boxes) | Phase 2 (breadcrumb), Phase 5 (markers) |
| Phase 2 (inline measurement) | Phase 3 (glyph runs), Phase 4 (font runs) |
| Phase 4 (font selection) | Phase 5 (marker faces) |
| Phases 1-8 | Phase 10 closure gates |
| Phase 10 catalogue refresh | not required this wave (no coverage change) |

## Agent allocation (as executed)

| Wave | Agent | Scope | Owned packages |
|------|-------|-------|----------------|
| 1 | layout-1 | hidden/zero-size boxes, border:0 | `internal/layout` |
| 1 | pdf-1 | empty-cmap fallback, URI encoding, dash bytes | `internal/pdf` |
| 2 | layout-2 | inline measurement, sup/sub, orphans | `internal/layout` |
| 2 | svg-1 | gradient preprocess, viewBox/font fallback | `internal/svg` |
| 2 | html-1 | quote root-cause investigation (seam report) | `internal/html` (read-only) |
| 2 | pdf-2 | bullet fold, face-keyed runes, weight faces | `internal/pdf` |
| 3 | layout-3 | q-quotes, font keywords, \r, markers (cancelled after landing; verified) | `internal/layout` |
| 4 | layout-4 | image/page-1 annots | `internal/layout` |
| 4 | layout-5 | placeholder, pre break-inside, fixed chrome (cancelled after landing; verified) | `internal/layout` |
| 5 | agent 8 | `make lint` fixes | all |
| 5 | agent 9 | `make test` fixes + envelope verification | `internal/convert` |
| 5 | agent 10 | golden, catalogue, claim-scan, envelope audit | read-only |

## Out of scope

- Docs/frontend behavior claims (no claims changed; `claim-scan` clean).
- Fixing wkhtmltopdf or the reference setups beyond Phase 0.
- New third-party dependencies; none added.
- Bot-walled sites beyond honest shell classification.

## Validation record

| Date | Phase | Command | Outcome |
|------|-------|---------|---------|
| 2026-09-16 | findings | 7 read-only comparison agents | 60 findings archived under per-site `evidence/` |
| 2026-09-16 | 0 | `wkhtmltopdf --print-media-type` x6 | all exit 0, refs in `/tmp/opencode/real-sites/wk-print/` |
| 2026-09-16 | 1-8 | targeted package tests per agent | layout/pdf/svg/html suites green at each wave, red-first probes cited above |
| 2026-09-16 | 8 | `make test` (agent 9) | exit 0; fixture-61 envelope [5,8] -> [5,9], verified by fresh 9-page conversion; no needle edits |
| 2026-09-16 | 8 | `make lint` (agent 8) | exit 0; size-check clean; frontend lint clean |
| 2026-09-16 | 10 | `make golden`, `css-catalog-map --check`, `make claim-scan` (agent 10) | exit 0 (71 pass), exit 0 (240 arms), clean |
| 2026-09-16 | 10 | `bin/gowkhtmltopdf` regeneration of 7 sites + w3schools | all exit 0; before/after in each `evidence/2026-09-16-forensics-gowk-after.*` and `w3schools/evidence/` |
