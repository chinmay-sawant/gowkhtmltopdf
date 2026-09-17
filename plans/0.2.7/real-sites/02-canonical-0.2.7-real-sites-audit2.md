# Real-sites audit wave 2 - phase-wise checklist (0.2.7)

> **Parent:** [`01-canonical-0.2.7-real-sites.md`](01-canonical-0.2.7-real-sites.md) (wave 1), [`README.md`](README.md), [`../README.md`](../README.md)
> **Status:** audit complete for all 10 artifacts. Fix waves 1 through 7
> plus the 2026-09-17 pre-gate / deferred close-out landed in the working
> tree (P1 LookupRune wiring, P2 root overflow skip, P3 flex pseudo, D1
> rows, 0-word warn). Phase 7 gates green 2026-09-17 (`make lint` /
> `make test` / `make golden` / `make claim-scan` all exit 0). Phase 8
> regen+verify green 2026-09-17: `REGEN2_FORCE=1` rebuild of all 10 PDFs,
> `verify.py` 23/23 PASS (`/tmp/opencode/regen2/verify-out.txt`). No
> `go.mod` replace for w3schools-6 (upstream pin only). Committed on
> `feature/live-website-sample`.
> **Estimated effort:** one fix wave per root-cause cluster plus gates and a full regeneration.
> **Depends on:** `scripts/real_site_drill.sh`, `scripts/pdf_page_forensics.py`, `bin/gowkhtmltopdf`.
> **Evidence:** per-site `plans/0.2.7/real-sites/<site>/evidence/2026-09-16-audit2/`
> plus post-fix forensics `2026-09-17-postfix-forensics.*` and regenerated PDFs.

## Close-out (live, 2026-09-17)

Code clusters below are closed with named tests. Phase 7 and Phase 8 are green.

### Closed this session (P1 / P2 / P3 / D1 / product warn)

- [x] P1 LookupRune wiring: `lookupFaceForRune` in
  `internal/layout/layout_font_faces.go` calls `Registry.LookupRune`; pinned by
  `TestRuneFaceHonorsUnicodeRangePartition`
  (`internal/layout/font_rune_partition_test.go`).
- [x] P2 w3schools-5 root overflow: root clip skipped so the footer survives;
  `TestRootOverflowDoesNotClipFooter`
  (`internal/layout/w3schools_root_overflow_test.go`).
- [x] P3 flex pseudo: `flexChildren` synthetic items plus
  `paintPositionedPseudo` in `internal/layout/flex.go`;
  `TestFlexPseudoContentPaints` (`internal/layout/audit2_flex_pseudo_test.go`).
- [x] ANA-17: block-in-inline forceBreak / margins / zero-height floor;
  `TestInfoboxMarriageLineStaysCompact` gap <= 12
  (`internal/layout/audit2_infobox_inline_test.go`).
- [x] ANA-18: square `OpFillRect` 0.3125em; `TestSquareListMarkerGeometry`
  (`internal/layout/audit2_square_marker_test.go`).
- [x] LCO-13: collapse-through escape margin;
  `TestLastChildMarginEscapesBorderlessParent`
  (`internal/layout/margin_height_test.go`); golden exit 0.
- [x] gobyexample-8: body paper wash on continuation pages;
  `TestBodyPaperWashCoversContinuationPage`
  (`internal/layout/audit2_body_wash_test.go`).
- [x] learn-cpp-org-7 ICO: hand-written PNG-in-ICO + 32bpp BMP in
  `internal/layout/layout_ico.go`; `TestICOToPNGExtractsEmbeddedPNG`,
  `TestICOToPNGExtractsBMP32`, `TestICOToPNGRejectsGarbage`
  (`layout_ico_test.go`).
- [x] programiz-cpp-8: `svg.ResolveUseReferences` plus layout wiring;
  tests in `internal/svg/use_resolve_test.go`.
- [x] 0-word warn: convert warns when zero text ops and no images;
  `TestVisibilityHiddenBodyWarnsNoExtractableText`,
  `TestNormalBodyDoesNotWarnNoExtractableText`
  (`internal/convert/layout_warn_test.go`).
- [x] w3schools-6 CLOSED AS UPSTREAM PIN (no `go.mod` replace; user rejected
  replace). `TestDecodeWOFF2W3SchoolsRobotoMono` pins the bbox failure;
  LiberationMono fallback remains graceful. Do not claim decode fixed.

### G1. Phase 7 gates (frozen tree, 2026-09-17)

- [x] `make test` exit 0.
- [x] `make golden` exit 0.
- [x] `make lint` exit 0.
- [x] `make claim-scan` exit 0.

### R1. Phase 8 regeneration and verification

- [x] `make build` exit 0 (binaries in `bin/`).
- [x] `REGEN2_FORCE=1 bash /tmp/opencode/regen2/run.sh` regenerated all 10
  PDFs (`summary.txt`: failed=0). Harness recreated under
  `/tmp/opencode/regen2/` after the prior tmp tree was wiped.
- [x] `python3 /tmp/opencode/regen2/verify.py` -> checks=23 pass=23 fail=0
  (`/tmp/opencode/regen2/verify-out.txt`).
- [x] Per-site page / word / link counts (PyMuPDF `get_links`):

  | slug | pages | words | links |
  |------|------:|------:|------:|
  | learncpp | 14 | 2092 | 358 |
  | learncpp-noimages | 14 | 2092 | 358 |
  | ana-de-armas | 22 | 7496 | 1008 |
  | gobyexample | 3 | 231 | 95 |
  | learn-cpp-org | 2 | 181 | 23 |
  | cplusplus-tutorial | 2 | 127 | 30 |
  | programiz-cpp | 9 | 1212 | 33 |
  | tutorialspoint-cpp | 8 | 1629 | 79 |
  | geeksforgeeks-cpp | 6 | 0 | 0 |
  | w3schools | 7 | 630 | 108 |

  Live learncpp paints Open Sans (liberation_pages=0). Post-fix forensics
  written as `evidence/2026-09-17-postfix-forensics.*`.
- [x] Working tree committed on `feature/live-website-sample`.

## Overview

Wave 1 fixed 30+ defects and regenerated seven site PDFs plus learncpp and
w3schools. This second wave audited the regenerated artifacts from scratch:
ten read-only agents, one per PDF file, each comparing its artifact against a
wkhtmltopdf print reference where one exists (Chrome fallback for
ana-de-armas; fresh wk references for learncpp and w3schools), re-checking the
wave 1 findings, and classifying every new difference as engine defect,
reference artifact, expected no-JS, probe needed, or media artifact.

The audit found the wave 1 engine fixes hold: every previously fixed row that
the agents re-checked still passes (gobyexample font-inherit and wrap,
ana-de-armas superscripts and overflow, cplusplus-tutorial borders and
visibility, programiz accordion and tofu, tutorialspoint border count, and
more). It also found open defects, including one regression introduced by
`99f0788`. Those clusters, plus the 2026-09-17 deferred close-out, are closed
in the working tree; only Phase 8 regen/verify remains.

## Executive summary

All 10 audits are complete: 5 high, 22 medium, 14 low/info across nine sites,
plus one site-side classification (geeksforgeeks, no engine defect). The
former product question (silent 0-word conversion) now warns through convert.

High severity:

- `ana-de-armas-13` regression: wrapped link text paints on interleaved
  baselines with a chained x pen after `99f0788`.
- `learncpp-1`: the fetched Open Sans webfont is selected only for spaces; all
  visible text paints in the Liberation fallback.
- `learn-cpp-org-3`: the `Output` h5 in a flex percentage column collapses and
  wraps letter by letter.
- `w3schools-1`: the footer float chain paints over the content and the footer
  words never reach the text layer.
- `w3schools-2`: the fixed language strip ignores `white-space:nowrap` and
  wraps into five rows that paint over content on every page.

## Phase 1: inline and pagination correctness

### 1.1 Regression: interleaved link-line baselines (`ana-de-armas-13`, high)

- [x] Reproduce with a minimal fixture: a link whose text wraps inside a
  paragraph, with more text after it; assert every painted fragment sits on its
  line baseline and the x pen resets at each line start.
- [x] Root-caused: `rowChromeBandCandidate` in
  `internal/layout/paint_pagination_fixpoint.go:500-528` matched the previous
  line's link runs because link chrome adds 1pt to the op box bottom, so
  `snapOpForward` dragged them with the snapped line. Fixed by only carrying
  `OpText`/`OpBullet` whose box reaches past the snapped baseline (not merely
  touching it).
- [x] Red-first tests: `TestSnapKeepsPreviousLineLinkRuns`,
  `TestSnapKeepsLineBaselinesTogether` in
  `internal/layout/pagination_snap_link_test.go` (red before the fix with
  "Bond girl" dragged 24pt; green after).
- [x] `go test ./internal/layout -count=1` green 2026-09-16; the fix agent's
  real-page reconstruction dropped 17 x-chained/y-jump span pairs to 0.
- [x] Anchor check: ana-de-armas p2, p5, p10-p13, p16, p17, p19, p20 crops
  under `evidence/2026-09-16-audit2/` show the defect; artifact re-verified in
  Phase 8.

### 1.2 Citation fragments keep the previous x cursor (`ana-de-armas-14`, medium)

- [x] Same emission path as 1.1; the `rowChromeBandCandidate` fix covers the
  citation fragments (previous line's x1 retained, baseline one line off).
- [x] Proof: `TestSnapKeepsLineBaselinesTogether` plus the fix agent's local
  rebuild; p11-p20 citation geometry matches the pre-fix-correct renders.
  Artifact re-verified in Phase 8.

### 1.3 Pagination relocation does not carry line decoration (`gobyexample-7`, `tutorialspoint-cpp-9`, medium)

- [x] Already fixed by the wave 1/3 `rowChromeBandCandidate` refinement in
  `internal/layout/paint_pagination_fixpoint.go` (a text/bullet run joins the
  snapped line only when its box reaches past the snapped baseline).
- [x] Regression-pinned red-first in
  `internal/layout/pagination_relocation_invariants_test.go`:
  `TestSnapRelocatesUnderlineAndHitboxWithText` (red on pristine HEAD:
  underline -14.109pt from baseline) and `TestSnapKeepsListMarkerWithItsLine`
  (red: marker with no text on its baseline).
- [x] Live checks 2026-09-16: gobyexample `Timeouts` pitch back to 15.0 with
  underline and annot rect covering the glyphs; 0 orphan markers in the
  tutorialspoint artifact. Full re-check in Phase 8.

### 1.4 Fragment backgrounds close a line short or paint empty bands (`tutorialspoint-cpp-10`, `cplusplus-tutorial-10`, `programiz-cpp-15`, low/medium)

- [x] tutorialspoint-cpp-10 and cplusplus-tutorial-10: already fixed by the
  wave 3 card-face stretch plus the band refinement; pinned red-first with
  `TestSplitListFragmentFillCoversLastLine` (red on pristine HEAD: fill
  [785.20, 798.00] vs text ink bottom 814.02).
- [x] programiz-cpp-15: fixed with `stripEmptyLeadingWashes` in
  `internal/layout/paint_pagination_seal.go:663` (a fill larger than
  `rowChromeMaxHeight` that starts below the page's last ink is zeroed when
  its owning box's first content is on a later page); tests
  `TestEmptySectionFragmentPaintsNoBackground` with two sub-cases.
- [x] Live checks: tutorialspoint fragment bottoms 143.88 and 406.17 cover
  their last lines; the programiz page-5 band is gone, page 6 unchanged.
- Note: `programiz-cpp-16` (card top band with h4 content on the page) is a
  different shape, tracked in 5.1.

## Phase 2: text encoding and fonts

### 2.1 Curly quotes fold to ASCII (`learncpp-2`, `learncpp-noimages-1`, medium)

- [x] `internal/pdf/winansi.go:23-44` (`winAnsiFoldCode`) now keeps U+2018 ->
  0x91, U+2019 -> 0x92, U+201C -> 0x93, U+201D -> 0x94; the table, ToUnicode
  path, and glyph lookup already decoded those codes, so no other writer
  change was needed. PDF 1.4 metadata stays deliberately ASCII-folded
  (`pdfDocPunctFold`); 1.7/2.0 metadata already uses UTF-16BE/UTF-8.
- [x] Red-first tests: `TestWinAnsiFoldCurlyQuotesKeepCodePoints`,
  `TestCurlyQuotesPaintRealCodesAndToUnicode` (`internal/pdf/winansi_test.go`)
  and `TestConvertedCurlyQuotesSurviveContentAndToUnicode`
  (`internal/pdf/convert_curly_quotes_test.go`); red before the fix, green
  after.
- [x] `go test ./internal/pdf -count=1` green 2026-09-16. All 10 artifacts
  re-verified in Phase 8.

### 2.2 Webfont selection ignores weight, style, unicode-range (`learncpp-1`, high)

- [x] Implemented: `internal/css/fontface_descriptors.go` parses weight,
  style, and unicode-range; `internal/convert/prepare/styles.go` carries them
  through `MergeFontFaces`; `internal/pdf/face_spec.go` plus
  `Registry.LookupRune` select per code point, so a family split into
  latin-ext and latin partitions paints each rune with its declared face.
- [x] Red-first tests: `TestParseFontFaceDescriptors`,
  `TestParseUnicodeRanges`, `TestUnicodeRangeCovers` (css),
  `TestMergeFontFacesCarriesUnicodeRanges` (prepare),
  `TestRegistryLookupSkipsFaceWithoutPrimaryGlyphs`,
  `TestRegistryLookupRuneHonorsDeclaredRanges`,
  `TestRegistryLookupUsesDeclaredWeightAndStyle` (pdf).
- [x] `go test ./internal/css ./internal/convert/prepare ./internal/pdf
  -count=1` green 2026-09-16. Live learncpp face check in Phase 8.
- [x] Wiring gap closed 2026-09-17: `lookupFaceForRune` in
  `internal/layout/layout_font_faces.go` now calls `Registry.LookupRune` so
  each code point picks its declared unicode-range partition face (with
  Liberation fallback when no partition covers the rune). Red-first
  `TestRuneFaceHonorsUnicodeRangePartition`
  (`internal/layout/font_rune_partition_test.go`). Live learncpp Open Sans
  split re-check stays in Phase 8.

### 2.3 SVG text fallback drops bold/italic (`cplusplus-tutorial-9`, low)

- [x] Implemented in `internal/svg/text_outline.go` (new): styled `<text>`
  (weight not 400 or italic/oblique, plain content, no dx/dy/rotate) is
  shaped through canvas and re-emitted as an outlined `<path>`, because the
  canvas SVG parser reads only font-family/font-size and always asks for the
  regular face. `prepareCanvasInput` wires `outlineStyledText` after
  `withFontFallbacks`. Plain text stays byte-identical.
- [x] Red-first tests in `internal/svg/text_outline_test.go` (6 tests,
  including the exact live wordmark markup); red showed the styled raster
  identical to the regular control, green shows bold italic ink > 1.2x.
- [x] Caveat: styled glyph shapes follow installed faces on the host
  (DejaVu Bold Oblique here) while browsers use the site's webfont, and the
  rewrite skips tspan, dx/dy/rotate, and CSS-block-only styles. Host-font
  dependence in the SVG path predates this fix and is an accepted
  rasterization limit; artifact re-check in Phase 8.

### 2.4 FontAwesome pseudo icons never paint (`learn-cpp-org-5`, medium)

- [x] Fixed: generated content on a block with no renderable in-flow
  children is now laid out (`emptyFlowRendersPseudo`,
  `internal/layout/layout_flow.go:301-307, 748`). Red-first tests:
  `TestEmptyInlineBlockPseudoContentPaints`,
  `TestEmptyInlinePseudoContentPaints`
  (`internal/layout/audit2_pseudo_empty_test.go`). Live probe: U+F04B (play
  x2), U+F2F1 (sync) and U+F102 (angle) paint in FontAwesome5FreeIdentity on
  pages 1-2, matching the wk face and glyph set.
- Note: generated content on `display:flex` containers is a separate row
  (5.1, programiz findings id -11); closed 2026-09-17 with
  `TestFlexPseudoContentPaints`.

## Phase 3: inline formatting details

### 3.1 Empty and whitespace-only inline spans (`learn-cpp-org-11`, `learn-cpp-org-12`, `ana-de-armas-16`, medium/low)

- [x] learn-cpp-org-11 and learn-cpp-org-12: fixed by skipping a packed line
  whose only content is collapsible whitespace
  (`internal/layout/inline.go:238, 388`); zero-content items that carry
  margin/padding/border/preserved whitespace keep their line box. Red-first
  tests: `TestWhitespaceOnlyInlineCreatesNoLineBox` and
  `TestWhitespaceBeforePercentImageDoesNotAddLine`
  (`internal/layout/audit2_inline_empty_test.go`). Live: learn-cpp-org H1
  baseline 58.1pt (was 71.17, wk 55.6, Chrome 57.7); hero gap 13.57pt (was
  27.3, Chrome 14.2).
- [x] ana-de-armas-16: an empty padded span now emits a zero-content item
  carrying its chrome (`inline_collect.go:664`), the placeholder is no longer
  mistaken for a collapsible space (`:998`), and the bracket space heuristic
  was removed. Red-first tests
  `TestEmptyInlineSpanContributesPadding`,
  `TestSpaceBeforeNowrapBracketSpanKept`. Live: quote kern 2.40pt, 0 adjacent
  `""` runs (was 14), IPA reads `pronunciation: [`. 
- [x] Proof: micro-fixtures in the wave 5 report
  (`ana-de-armas/evidence/2026-09-16-audit2/fix-report-layout-wave5.md`);
  `go test ./internal/layout ./internal/convert -count=1` green 2026-09-16.

### 3.2 Figure captions, infobox lines, square marker (`ana-de-armas-15`, `-17`, `-18`, medium/low)

- [x] -15: `overflow-wrap: anywhere` now breaks whole words before a mid-word
  split (new `breakAnywhere` policy, `layout_measure.go:653-720`); `word-break:
  break-all` keeps the old behavior. Red-first test
  `TestFigureCaptionBreaksWholeWordsWithAnywhere`
  (`audit2_caption_wrap_test.go`). Live: captions read `number`,
  `Sebastián`, `International`, `Comic-Con` whole.
- [x] -17: closed 2026-09-17. Block-in-inline forceBreak / margins /
  zero-height floor keep the spouse and marriage lines compact; red-first
  `TestInfoboxMarriageLineStaysCompact` asserts gap <= 12pt
  (`internal/layout/audit2_infobox_inline_test.go`; Chrome target ~8.2pt).
- [x] -18: closed 2026-09-17. `list-style-type: square` paints a vector
  `OpFillRect` at 0.3125em (Chrome geometry) instead of the U+25AA glyph;
  `TestSquareListMarkerGeometry`
  (`internal/layout/audit2_square_marker_test.go`).

### 3.3 Breadcrumb inline-block whitespace wrap (`cplusplus-tutorial-3`, medium)

- [x] Root cause: intrinsic max-content measurement skipped whitespace-only
  text nodes (`cellMeasure.measureTextBoundary`), so the shrink-to-fit `ul`
  was one space narrower than its painted content; the second `<li>` wrapped.
  Fixed with `pendingSpace` in `cellMeasure`
  (`layout_measure.go:179-187`). A second interaction was found and fixed:
  inline-blocks aligned with their bottom edge on the baseline, growing mixed
  lines by the box descent (`atomicInlineAlign`,
  `inline_vertical_align.go:33`).
- [x] Red-first test: `TestInlineBlockBreadcrumbWhitespaceStaysOnOneLine`
  (`internal/layout/audit2_breadcrumb_measure_test.go`); red showed two rows
  at y=25.16 and 66.00, green shows one row and one shared baseline.
- [x] Live: `#I_bar` is 32.0pt tall with `Tutorials : C++ Language` on one
  line (was 94.70pt, two rows). Re-open note: the wave 1 closure had used a
  hand-compacted probe that could not reproduce the real markup.

### 3.4 Fixed strip ignores nowrap and paints over content (`w3schools-2`, high)

- [x] Implemented: `internal/layout/inline.go` keeps an over-wide nowrap
  cluster on one line when it cannot fit even a fresh line (overflow is left
  to the clipping ancestor), and `internal/layout/inline_collect.go` prefers
  the nowrap cluster rule over the replaced-content break at atomic-inline
  boundaries.
- [x] Red-first test: `TestNowrapInlineBlockStripStaysOnOneRow`
  (`internal/layout/w3schools_audit2_test.go`); `go test ./internal/layout
  -count=1` green 2026-09-16. Live strip row count verified in Phase 8.

### 3.5 Words cut mid-word at the right box edge (`w3schools-3`, medium)

- [x] Implemented: `internal/layout/layout_flow.go` now clears any normal
  flow block below preceding floats when its exclusion slot is narrower than
  its min-content width (`flowSlotWidthNeeded`, computed for every block child
  beside active floats), not only BFC roots. The line packer can no longer
  feed a starved width into the emergency mid-word split.
- [x] Red-first test: `TestNormalFlowFooterClearsNarrowFloatSlot`
  (`internal/layout/float_flow_slot_test.go`); red showed
  "Top Tuto rials FOOTE R SENTINEL TEXT", green shows whole words.
- [x] Probe check 2026-09-16: `F-float-nowrapoverflow.html` now reads
  "Top Tutorials" and "FOOTER SENTINEL TEXT" whole on page 3.
- [x] `go test ./internal/layout -count=1` green. Live w3schools pages 3-5
  re-checked in Phase 8 (the audit also reported truncated words inside the
  normal content column, not covered by the probe).

## Phase 4: replaced content and sizing

### 4.1 Image sizing and continuation (`tutorialspoint-cpp-11`, `programiz-cpp-12`, `programiz-cpp-13`, medium)

- [x] tutorialspoint-cpp-11: fixed with a containing-block width field
  (`imgCBW`) installed by `pushReplacedCBW` for one img/svg build
  (`internal/layout/layout_images.go:54`, `layout_flow.go:735`);
  `banner-probe.html` now draws 507.18x137.82 inside the card (was
  538.68x146.38 past the page frame). Test:
  `TestBlockImagePercentWidthUsesContainingBlock`.
- [x] programiz-cpp-12: the percent base and the column-flex measurement now
  use the container width (`flex.go:1314, 1629`); the screenshot keeps its
  966:570 ratio and no longer overflows. Tests:
  `TestFlexColumnImagePercentWidthKeepsRatio`,
  `TestFlexColumnImageWidthAttributeKeepsRatio`.
- [x] programiz-cpp-13: already fixed by the earlier avoid-inside wave; pinned
  with `TestImageAvoidInsideMovesWholeToNextPage` and
  `TestFlexItemImageAvoidInsideMovesWholeToNextPage`.
- [x] `go test ./internal/layout -count=1 -race` green; report:
  `programiz-cpp/evidence/2026-09-16-audit2/fix-report-layout-wave6.md`.

### 4.2 Inline SVG ignores CSS-resolved size (`cplusplus-tutorial-8`, medium)

- [x] Fixed: `usedInlineSVGSize` resolves percent/calc against the
  containing block, and a viewBox-only svg defaults to 100% of the containing
  block at the viewBox ratio (SVG 2), matching Chrome. Wordmark 90x27 ->
  48x14.4 with the exact live markup; explicit attributes still win. Tests:
  `TestInlineSVGViewBoxUsesContainingBlockWidth`,
  `TestInlineSVGAttributesWin`. Report: same wave 6 file.

### 4.3 Flex percentage columns and split cards (`learn-cpp-org-3`, `learncpp-3`, high/medium)

- [x] learn-cpp-org-3: fixed by an earlier wave
  (`forceFlexItemMainSize` now converts percent min/max-width against the
  flex container content width, `flex.go:1156-1170`; `.col-3` no longer
  collapses to 6.25 percent) and re-verified on the audit input: one span
  `Output` at x0=36.5 x1=69.1. Pinned red-first with
  `TestFlexPercentColumnKeepsWordWhole` and
  `TestFlexPercentColumnBases` (`audit2_flex_percent_col_test.go`).
- [x] learncpp-3: implemented in `paint_pagination_fixpoint.go`
  (`moveSnappedRowBoxes`/`shiftBoxTreeY` move the box subtree at the snapped
  row so the stale-top re-shift no longer compounds) and
  `paint_pagination_chrome.go` (`hasOwnCardFace` admits background/box-shadow
  cards, `stretchPaginatedChrome(growShadowLayers)` grows shadow faces). Red
  before: continuation row 49.77pt below the fill; green after with all 356
  of 356 row bands inside a card face in the mirror check.
- [x] Red-first test: `TestSplitCardRowsStayInsideCardFill`
  (`internal/layout/card_split_fill_test.go`).
- [x] Proof for learn-cpp-org-3: `TestFlexPercentColumnKeepsWordWhole` (and
  `TestFlexPercentColumnBases`) in `audit2_flex_percent_col_test.go`
  assert the letter-heavy heading stays whole.

### 4.4 Image ratios broken (`w3schools-4`, medium)

- [x] Fixed via three mechanisms: inline-SVG percent sizing stays in its icon
  box (`TestInlineSVGPercentSizeStaysInIconBox`), flex-column cert preview
  keeps its ratio (`TestFlexColumnCertPreviewKeepsRatio` red 187.5x50.34 ->
  green 187.5x134.25), and auto-sized replaced flex items no longer have
  their main axis force-set (`flexReplacedAutoHeight`, `flex.go:1129`). Pins:
  `TestFlexRowPercentImageStaysSquare`, `TestFlexColumnAutoImageStaysSquare`,
  `TestFlexItemPlainImageKeepsRatio`, `TestInlineAutoImageKeepsIntrinsicSize`.
- [x] Caveat: the exact live right-rail badge trigger was not reproducible
  from the fetched markup; the governing paths are pinned by the tests
  above. Live badge re-check stays in Phase 8.
- [x] Housekeeping: wave 6 extracted font-face resolution to
  `internal/layout/layout_font_faces.go`, trimmed `layout.go` 2350 -> 2149,
  and updated `scripts/file-size-allowlist.txt`; size-check clean.

## Phase 5: flow and flex edge cases

### 5.1 programiz header and code cards (`programiz-cpp-11`, `-14`, `-16`, medium/low)

- [x] findings id -1 / search field (the audit text called it -11): fixed. The
  real cause was a stray `img{width:100%}` plus the viewport fallback in
  `imageContainingWidth`; measure-only percentage images are now indefinite
  without a containing block (`layout_images.go:42-57`), direct flex items get
  the container width in `measureFlexCrossMax`, and deferred absolute replaced
  children get the absolute containing block. Live probe: placeholder x0=109.35
  with the full text (was x0=415.35 clipped). Tests:
  `TestFlexItemImagePercentWidthIsIndefiniteInMeasure`,
  `TestFlexItemImageExplicitWidthInMeasure`.
- [x] findings id -11 (flex pseudo content): closed 2026-09-17.
  `flexChildren` builds synthetic items for `::before`/`::after`, and
  `paintPositionedPseudo` paints them (`internal/layout/flex.go`);
  `TestFlexPseudoContentPaints` (`internal/layout/audit2_flex_pseudo_test.go`).
- [x] -14: fixed with the flex shrink floor at used-width height
  (`flex.go:1386-1397`); the button no longer overlaps the paragraph. Test:
  `TestFlexColumnParagraphNotSqueezedUnderButton` (red: 1.30pt overlap).
- [x] -16: already fixed by the avoid-inside/relocation wave; pinned with
  `TestFlexRowPreBordersStayWithCodeAfterBreak`.

### 5.2 Dropped margins (`learn-cpp-org-13`, low)

- [x] Button `mt-2`: fixed by carrying inline-block vertical margins into the
  line metrics (`inline.go:49-50`, `inline_collect.go:641-646`,
  `inline_vertical_align.go:26-35`). Test:
  `TestInlineBlockTopMarginMovesContent` (red: gap 7.85pt with the margin
  invisible).
- [x] `ul` escaping bottom margin: closed 2026-09-17. Collapse-through lets
  the last-child margin escape a borderless parent;
  `TestLastChildMarginEscapesBorderlessParent`
  (`internal/layout/margin_height_test.go`); `make golden` exit 0.

### 5.3 Footer float chain paints over content (`w3schools-1`, high)

- [x] Implemented: `opExtra.FloatOp` plus `restampFloatOps` mark float-owned
  ops and `paintLayer` orders them between in-flow chrome and in-flow content
  (CSS 2.1 Appendix E); `bfcSlotWidthNeeded` in `internal/layout/layout_flow.go`
  clears a BFC root below preceding floats when the exclusion slot is too
  narrow (`bfcSlotEpsilon = 0.5`).
- [x] Probe verification 2026-09-16 with the saved `B-float.html`: before, only
  "PLUS" of the footer words reached the text layer, fragmented in a ~40pt
  column at x=457 on page 1; after, the footer clears to page 3 at x=87-102
  with whole lines "PLUS SPACES FOR TEACHERS", "Top Tutorials", "FOOTER
  SENTINEL TEXT".
- [x] Red-first tests: `TestFooterClearsPageTallFloats`
  (`internal/layout/w3schools_audit2_test.go`); `go test ./internal/layout
  -count=1` green. Live w3schools footer re-checked in Phase 8.

### 5.4 Root overflow drops the footer (`w3schools-5`, medium)

- [x] Closed 2026-09-17: root `overflow` no longer installs a per-page clip
  that drops the float-cleared footer. Red-first
  `TestRootOverflowDoesNotClipFooter`
  (`internal/layout/w3schools_root_overflow_test.go`). Live
  `w3schools/footer_words` check stays in Phase 8.

## Phase 6: low and info leftovers

- [x] `gobyexample-8`: closed 2026-09-17. Body paper wash continues onto
  later pages; `TestBodyPaperWashCoversContinuationPage`
  (`internal/layout/audit2_body_wash_test.go`).
- [x] `learn-cpp-org-7`: closed 2026-09-17 (no longer classification-only).
  Hand-written PNG-in-ICO and 32bpp BMP decode in
  `internal/layout/layout_ico.go` (wired through `decodeImagePayload`);
  `TestICOToPNGExtractsEmbeddedPNG`, `TestICOToPNGExtractsBMP32`,
  `TestICOToPNGRejectsGarbage` (`layout_ico_test.go`) plus the payload-safety
  png-in-ico case. Earlier report still useful for history:
  `learn-cpp-org/evidence/2026-09-16-audit2/fix-report-ico.md`.
- [x] `learncpp-5`: verified as correct behavior, not a cache bug. The two
  LiberationSans-Bold subsets serve genuinely different rune unions because
  page 10 assigns bold the name `/F1` (first paint use) while the other 12
  pages assign `/F2`; the cache key is name-independent, and the same-name
  case does dedupe (OpenSans xref 399 is one subset across `/F1` and `/F2`).
  Pinned with `TestFontSubsetSplitByResourceName` and
  `TestFontSubsetSharedAcrossResourceNames`
  (`internal/pdf/font_union_test.go`). Report:
  `learncpp/evidence/2026-09-16-audit2/fix-report-pdf-dup-subset.md`.
- [x] `programiz-cpp-8`: closed 2026-09-17. `svg.ResolveUseReferences` inlines
  external and same-document `<use href|xlink:href="...#id">` before canvas
  parse; layout passes `resolveImageData` at the Rasterize call sites. Tests
  in `internal/svg/use_resolve_test.go` (`TestResolveUseReferencesExternal`
  and siblings).
- [x] `w3schools-6`: CLOSED AS UPSTREAM PIN 2026-09-17 (user rejected a
  `go.mod` replace). The font is valid and spec-conforming;
  `github.com/tdewolff/parse/v2` `BitmapReader.Read` uses a bound that refuses
  the last bit of a full 32-bit-aligned bitmap (28 bytes, 224 glyphs,
  composite glyph 223), and `tdewolff/font` treats the missing bbox as fatal.
  Do not claim decode fixed. Fixture
  `testdata/fonts/woff2/roboto-mono-v13-latin-500.woff2` plus
  `TestDecodeWOFF2W3SchoolsRobotoMono` pin the bbox failure; LiberationMono
  fallback remains graceful. Report:
  `evidence/2026-09-16-audit2/fix-report-woff2.md`.
- [x] `learncpp-4`, `gobyexample-9`: reference/annotation nuances, no engine
  action. learncpp's annot shapes differ while URI targets match Chrome;
  gobyexample's empty-path href keeps the literal href where Chrome
  serializes a trailing slash. Closed with reason.
- [x] `geeksforgeeks-cpp`: no engine defect (JS/bot wall). 0-word warn shipped
  2026-09-17: convert warns when there are zero text ops and no images
  (`document contains no extractable text (page may require JavaScript)`).
  Tests: `TestVisibilityHiddenBodyWarnsNoExtractableText`,
  `TestNormalBodyDoesNotWarnNoExtractableText`
  (`internal/convert/layout_warn_test.go`).
- [x] `cplusplus-tutorial-11`: fixed in `scripts/pdf_page_forensics.py`
  (+84/-18): suspicious pages are now detected by per-span ink
  (`spans_missing_ink`, `SPAN_INK_FLOOR = 0.01`) instead of whole-page
  coverage; coverage stays in the report for information. CLI, JSON keys,
  and markdown shape unchanged. Validated: cplusplus-tutorial 1 -> 0
  suspicious, a synthetic covered-text PDF still flags 1, and the corpus
  sweep changed only that one artifact. Report:
  `cplusplus-tutorial/evidence/2026-09-16-audit2/fix-report-forensics.md`.

## Phase 7: verification gates

- [x] `make test` exit 0, recorded 2026-09-17.
- [x] `make golden` exit 0 (layout and paint changes), recorded 2026-09-17.
- [x] `make lint` exit 0 (single run after the tree settled), recorded
  2026-09-17.
- [x] `make claim-scan` exit 0, recorded 2026-09-17.

## Phase 8: regeneration and re-verification

- [x] `make build` with the fixed tree.
- [x] Regenerated all 10 files via `/tmp/opencode/regen2/run.sh`
  (`real_site_drill.sh` for eight sites; learncpp `--no-images` and
  w3schools Chrome UA as direct converts). `summary.txt` failed=0.
- [x] Re-ran `scripts/pdf_page_forensics.py` into
  `evidence/2026-09-17-postfix-forensics.*` on all 10 (plus learncpp
  noimages).
- [x] Re-ran per-site audit checks via `verify.py`: 23/23 PASS. Live
  learncpp Open Sans census green; ana xchain=0; w3schools footer words
  present; GFG still 0 words / 6 pages.
- [x] Updated `plans/0.2.7/README.md`, `01-canonical-0.2.7-real-sites.md`,
  and `knowledge-base/` with the Phase 8 numbers.

## Dependencies

- Phases 1 to 5 change `internal/layout` or `internal/pdf`; run them
  sequentially per package, never two writers on one package.
- Phase 7 runs once, after Phases 1 to 6 land.
- Phase 8 depends on a green Phase 7.
