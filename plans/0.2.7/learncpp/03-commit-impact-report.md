# Two-commit impact report: learncpp.com conversion (v0.2.7)

Audit date: 2026-09-16. Branch: `feature/live-website-sample`.
Commits under review:

| Commit | Files | +/− | Summary |
|--------|-------|-----|---------|
| A `73ca696` | 96 | +7605 / −751 | fix learncpp.com conversion end to end |
| B `068cd64` (HEAD) | 68 | +3225 / −1052 | align header and table with the wkhtmltopdf reference |

Base for the pair is `891cac7`.

One clarification before anything else: these commits target **learncpp.com**, the
C++ tutorial website used as the v0.2.7 real-site conversion drill. There is no
`lens.cpp` in this repo and no C++ code involved. The engine is pure Go.

All line numbers are HEAD (`068cd64`) unless a commit is named. The 10 analysis
agents reconstructed before/after from the local tree, remote git objects (the
branch is pushed), and the plan ledgers; this audit re-verified the load-bearing
claims locally and marks anything it could not verify.

## The numbers that define the impact

Baseline before commit A (`plans/0.2.7/README.md:26-29`, `learncpp_conversion.log`):

- `gowkhtmltopdf --url https://www.learncpp.com/ -o learncpp.pdf` exited 1 after
  89s and wrote a 0-byte file. The log said
  `embed png I0: image: config: png: invalid format: not a PNG file`.
- With `--no-images` it exited 0 after 82s with 37 pages, but 34 of them were
  visually blank: the text was painted under an ancestor background.

Final state after both commits (recorded in `learncpp_v027.log:84-104` and
`plans/0.2.7/README.md:37-43`):

- exit 0, 226,441 bytes, 13 pages, 0 covered-text pages, Open Sans and Liberation
  embedded as subset-tagged fonts.
- Link annotations fell from 1119 to 357, words from 3096 to 2129. The word drop
  is mostly the print-only URL suffixes that disappeared (the commit message says
  350, the phase record says 356; both point at the same suffix flood).
- Header tagline renders on one line at 125.65pt, the chapter badge digit sits
  6.0pt inside its pill, all 310 row numbers sit inside their bands.

The stack of fixes is general engine behavior, not site-specific patching. Any
page that uses color-only background shorthands, `z-index` plus transforms,
stylesheet-relative `@font-face` URLs, `calc()` widths, percentage insets, or
unclosed `<p>` tags is affected.

## Commit A: make the conversion survive

### Image pipeline safety

The fatal chain was: `#f4f6fd` is a color written in the `background` shorthand.
The cascade stored it as an image source, the loader fetched it as a URL, the
bytes were HTML, and the PDF writer tried to embed them as PNG I0. Fixes:

- `internal/layout/style_paint_props.go:368` adds `backgroundValueHasImage`, so a
  color-only `background` value never becomes an image source.
- `internal/layout/layout_flow.go:123` adds `decodeImagePayload`. A payload that
  is not a supported image (PNG/JPEG/GIF/SVG) is dropped with one warning that
  names the source, and the nil result is cached so the fetch is not retried.
- Every emit site guards on payload presence (`background_image.go:106`,
  `border_image.go:204`, `layout_images.go:422`, `layout_svg.go:44`, and the
  inline paint path). Four measure/collect sites still use the older
  `ref.data != nil` form (`layout_flow.go:933`, `inline.go:114`,
  `inline_collect.go:667`, `layout_measure.go:530`); they are safe only because
  `decodeImagePayload` never returns an empty payload. Worth tightening.
- Embed errors now name the `src` instead of the `I0` counter
  (`internal/layout/paint.go:1597`, `imageOpLabel`), and GIF frames are decoded
  and re-encoded as PNG (`layout_images.go:548`).
- The CLI stops truncating an existing output file before a run succeeds: the
  file opens on the first write (`internal/app/pdf.go:115`
  `openLazyOutput`, `:162` `lazyFileWriter`; image mode in
  `internal/app/image.go:55`).

Covered by `internal/layout/image_payload_safety_test.go` (9 tests) and
`internal/app/image_test.go:160-236`.

### Paint order and stacking contexts

A stacking context is a box whose children paint together as one group; inside
it, children sort by z-index. learncpp has `main#main { z-index: 1;
background: #fff }` with `article.hentry { transform: ... }`. Transforms create
a stacking context too, so the old flat sort painted descendant text before the
ancestor white background, and 34 of 37 pages came out blank under that
background.

- New `internal/layout/paint_order.go` compares full context chains: sibling
  contexts order by `(z, positioned, seq)` at the first frame where the chains
  diverge, and ancestor chrome always paints before descendant content
  (`paintStackingBefore:95`, `paintAncestorContextFirst:134`).
- `internal/layout/layout_stacking.go` (new) collects the stacking helpers that
  moved out of `layout.go` plus the new frame machinery
  (`stampPaintState:186`).
- Identity transforms still create contexts, matching CSS. A test asserts the
  falsification case fails without the fix
  (`paint_order_test.go:141` `TestTransformedDescendantTextPaintsAfterAncestorChrome`).

This is the single largest correctness win in the pair.

### Fonts, stylesheet base URLs, and WOFF2

- Every stylesheet now carries the URL it was loaded from
  (`internal/css/css.go:73` `Base`) and resolves `url()` and `@font-face src`
  against that base (`internal/css/url_resolve.go:24` `ResolveURLs`, called from
  `internal/convert/prepare/styles.go:202`). Before, the base was computed for
  `@import` and then thrown away, so document-relative font URLs 404ed.
- The font extension gate accepts `.woff2`, `.woff`, `.ttf`, `.otf` and `data:`
  URIs; only `.eot` stays policy-skipped (`styles.go:478-487`). WOFF2 bytes now
  decode through `tdewolff/font` (`internal/pdf/woff.go:40-62`).
- The 42KB base64 `data:` URI is no longer dumped into logs. Warnings print the
  scheme, media type and a 64-char metadata cap (`styles.go:502-562`), enforced
  by `TestFontFaceDataURILogHygiene` (`internal/convert/fontface_test.go:342`).
- Embedded fonts get deterministic subset tags: sha256 of the subset bytes
  mapped to six A-Z letters (`internal/pdf/fonttype0.go:22`), so the same input
  produces the same `ABCDEF+OpenSans` name.
- Dependency change: `github.com/tdewolff/font` moved from indirect to direct
  (`go.mod:10`), the third allowed direct module, and nine indirect modules
  bumped (`brotli`, `tdewolff/minify`, `tdewolff/parse`, `x/image`, `x/text`,
  and others). `TestDirectModuleAllowlist` (`internal/pdf/shape_test.go:186`)
  enforces the allowlist.

Caveat: the commit message says "data-URI WOFF1/TTF/OTF register". CFF-flavored
OTF (`OTTO`) is still rejected (`internal/pdf/fonts.go:131-133`), so the claim
holds only for TrueType-flavored OTF.

### Layout: flex width, calc, generated content, overflow warning

- Flex rows use the remaining width instead of collapsing to a tiny column. The
  learncpp TOC title went from 141.25pt to 310.50pt (`flex_lesson_row_test.go:81`).
- `calc()` percentages resolve against the containing block, not the viewport.
  `calc(100% - 200px)` in a 400px parent measured 518px before and 200px after
  (`style_values.go:894-1025`, `style_calc_test.go`).
- Generated `::before`/`::after` content is folded into min/max content
  measurement for cells (`layout_measure.go:100` `measureCellMinMaxMode`), which
  stopped the print URL suffix from inflating layout widths in commit A.
- A run-level text overflow check now warns when text crosses the page width
  (`layout_census.go:49` `warnTextOverflow`). It only warns; it does not clip.

### HTML parser: unquoted attribute values ending in `/`

`<a href=https://example.com/x/>` used to trigger self-closing detection because
the tag ended with `/`, so the anchor was closed early and its text orphaned.
`internal/html/html.go:763-813` now checks that the trailing slash follows
whitespace or a quote before treating the tag as self-closing. Fixed in commit A,
pinned by `html_test.go:750-793` and the layout test
`flex_lesson_row_test.go:151`.

### PDF writer and CLI behavior

- Link annotations merge when they share a target and their boxes are adjacent or
  overlapping (`internal/pdf/annots.go`, `internal/pdf/pdf.go:466` `addLink`).
  Merging is disabled for PDF/UA documents because tagged structures assume one
  annotation per structure element. Annotation count on learncpp went 1119 to
  371 in commit A (partly fewer pages, partly merging); the final B-era run
  records 357.
- `/Lang` emits whenever a language is set, defaulting to `en-US` for PDF/UA
  (`pdf.go:1065`), and `/Title` falls back to the document `<title>` for every
  document, not just PDF/UA (`internal/convert/pdf_pipeline.go:209`).
- `--no-print-media-type` was a no-op and now selects screen media. Settings
  store a tri-state (`internal/settings/settings.go:190` `MediaOverride`), so
  absent, screen and print are distinguishable, and an explicit screen override
  beats `--media-type print` (`settings.go:231` `ResolveMedia`). Pinned by
  `internal/cli/cli_test.go:261` and `internal/settings/settings_test.go:862`.

### Tooling and evidence

- New `scripts/real_site_drill.sh` (conversion + log capture + forensics) and
  `scripts/pdf_page_forensics.py` (PyMuPDF page metrics, low-ink page detection,
  optional comparison). Both are manual tools; neither is wired into `make` or
  CI, and the forensics script needs python3 + PyMuPDF plus network for live runs.
- Baseline and after artifacts are committed at the repo root (`learncpp.pdf`,
  `learncpp_noimages.pdf`, three logs) and under
  `plans/0.2.7/learncpp/evidence/`. Plan 01 line 19 still says these are
  "untracked by design"; `git ls-files` shows five tracked artifacts. Worth
  either fixing the line or adding an ignore rule before the next drill.
- The perf3 output pin moved +86 bytes (1,420,537 to 1,420,623) because subset
  tags, `/Title` and `/Lang` changed the bytes (`document_perf3_pin_test.go:11`).
- `scripts/file-size-allowlist.txt` moved `layout.go` 2497 (parent) to 2342 (A),
  then 2349 (B). Note the commit message says 2501, which was the working-tree
  count, not the committed parent (2497).

## Commit B: make it look like the wkhtmltopdf reference

Commit B came from a picture-level diagnosis against wkhtmltopdf 0.12.6.1 and
Chrome at a 718px viewport (`plans/0.2.7/learncpp/evidence/2026-09-16-visual-diagnosis.md`).
Artifact checks could not see any of these defects.

### Header float sizing with a descendant image

A float containing text plus a small logo was shrink-to-fit at the logo's width,
so the tagline wrapped to a narrow column. Floats now measure the largest
descendant image only inside floated subtrees and add it to the text width
(`layout_flow.go:1192-1253`, `layout_measure.go:823` `measureLargestFloatImageWidth`).
In-flow images keep the old behavior (`learncpp_header_test.go:241`).

### Text decoration does not propagate into floats

CSS 2.1 16.3.1 propagates `text-decoration` only to in-flow descendants.
Floating, absolutely positioned, and atomic inline boxes are excluded. The
cascade now vetoes the copy before it happens (`style_cascade.go:398`
`decorationBlocked`, gated at `:369`). Floats went from 12 stroked runs to 0.

### Uppercase advance matches the painted glyphs

`text-transform: uppercase` changed the glyphs but the cursor advanced by the
untransformed width, so `LEARN` and `C++` overlapped by 6.09pt. Paint now
returns the real advance and measurement uses the transformed run
(`inline_paint.go:274-298`, `layout_measure.go:283-293`). Gap is 3.43pt.

### Relative percentage insets resolve in a post-measure pass

`position: relative; top: 100%` needs the containing block's height, which does
not exist until after measure. New `internal/layout/relative_percent.go`
registers pending percentage insets during build and resolves them once the
root box exists (`resolveRelativePercents:33`), shifting the box and its ops.
Auto-height containing blocks keep the static position. Before, the test
measured `shift=0.00, want 100`; after, the title centers.

### Vendor-prefixed aliases stop double-applying

When a rule had both `-webkit-transform` and `transform`, both applied and the
transform accumulated (rendered at -100%). The cascade now skips the prefixed
alias when the canonical property is present
(`style_cascade.go:1306-1328`, `style_cascade_test.go:424`). Caveat: it always
prefers the canonical property, so a deliberately prefixed-last override is
ignored.

### `box-sizing: inherit` resolves

`*{box-sizing: inherit}` was silently dropped, so the badge sized on the content
box and hung about 44pt too wide out of its card. The parent value now copies
(`style_properties.go:70-79`, `learncpp_chapter_badge_test.go`).

### Overflow clip clamps text runes

`overflow: hidden` clipped whole text ops, which dropped or kept entire lines
that only partly crossed the edge. `clampTextOpToClip`
(`internal/layout/overflow_clip.go:475`) walks per-rune advances and rewrites
the run so only the runes inside the clip paint. The clip walk itself predates
v0.2.7; only the rune clamp is new. Transformed, rotated, fontless, or vertically
partial runs still fall back to whole-op clipping.

### Pagination keeps a row number with its band

When a row with its badge number was snapped to the next page, the number could
stay behind in the page gap. `rowChromeBandCandidate`
(`paint_pagination_fixpoint.go:479`) now accepts `OpText`, `OpBullet`,
`OpImage`, and `OpLinkURI` on the same band, so the number travels with the row.
Two caveats: the band tolerance is a heuristic (`H <= 40`), and the rule shifts
any qualifying text on the band, not only badge digits.

### HTML5 `<p>` auto-close

learncpp's home page opens six `<p>` tags and never closes them. Anchors inside
following `<div>`s stayed `<p>` descendants, so the print rule
`.cryout p a::after` matched them and appended a URL to all 350 lesson rows.
The parser now closes an open `<p>` at the HTML5 block-level start tags
(`internal/html/html.go:443-456` `closesOpenParagraph`). Result: 0 URL suffixes
and 19 to 14 pages. This also explains the mysterious literal `()`: it was
`attr(href)` on anchors whose href had been stripped by the self-closing bug,
not a missing glyph. The missing-icon path in the writer is unchanged
(`internal/pdf/content.go:765` folds unresolved runes to `?`).
The implementation checks only the top of the open-element stack, so a wrapped
case like `<p><b><div>` keeps the old nesting. The tag list also omits a few
HTML5 triggers (`center`, `dialog`, `dir`, `search`, `summary`). Both are known
narrowings, not regressions.

### Docs, catalogue, and the built site

- `AGENTS.md` now lists three direct dependencies, matching `go.mod:8-10` and
  the Makefile allowlist comment (`Makefile:7`).
- WOFF2 wording was corrected in `documentation/` and in six frontend content
  pages, and `docs/` was rebuilt.
- The CSS catalogue (`plans/0.2.6/catalog/mapping.json`,
  `implemented-code-evidence.json`) was re-pointed at the new consumer evidence:
  354 implemented / 0 partial / 464 unsupported, no status moves. The evidence
  rows refreshed are 20 + 332 + 11 + 27; the 11 are the new
  `relative_percent.go:33` citations.

## Impact by audience

| Audience | What changes for them |
|----------|------------------------|
| CLI users converting arbitrary pages | No more 0-byte crashes on color-only backgrounds or non-image payloads; `--no-print-media-type` works; failed runs no longer truncate an existing output; image failures name the source URL; `<p>` auto-close and unquoted-href parsing match HTML5 more closely |
| PDF readers | Correct paint order (no more text hidden under ancestor backgrounds), fewer duplicate link annotations, `/Lang` and `/Title` set, deterministic subset-tagged fonts, one-line header and intact chapter badges on learncpp-like pages |
| Library users (`Document`) | Same engine gains; `internal/*` changes are not importable, and the root API is unchanged |
| Repo / build | Third direct dependency `tdewolff/font` approved and enforced; `make test`, `make lint`, `make golden` recorded green; perf pin, size allowlist, and docs/catalogue updated |

## Verification status

Verified locally during this audit, from git objects and files:

- parent `layout.go` 2497 lines, HEAD 2349; allowlist entry matches
  (`scripts/file-size-allowlist.txt:9`).
- `go.mod:8-10` has exactly three direct modules.
- Root artifacts are tracked by git; `VERSION` is still `0.2.6`.
- The parser (`closesOpenParagraph`), cascade (`decorationBlocked`, alias skip),
  and perf pin (+86) hunks shown above.
- `frontend/public/data/issues.json` still contains 17 stale "WOFF2 skipped"
  rows plus one "two direct dependencies" row (see leftovers).
- `AGENTS.md:258-260` still quotes old file sizes; `AGENTS.md:280` still ends the
  plans range at 0.2.4.
- Knowledge-base pages still say WOFF2 is rejected
  (`knowledge-base/wiki/syntheses/fonts-and-typography.md:119,126`,
  `knowledge-base/wiki/concepts/trust-boundary.md:26`).

Recorded in ledgers and evidence, not re-run by this audit:

- `make test`, `make lint`, `make golden` exit 0 for both waves
  (`plans/0.2.7/README.md:34,41`).
- Final forensics: 13 pages, 2129 words, 357 link annots, 0 low-ink pages,
  subset fonts (`learncpp_v027.log:84-104`).
- `make claim-scan`, frontend lint, and a production `docs/` build ran in the
  docs wave (`knowledge-base/wiki/log.md`, V7 entry).

Caveats this audit could not close:

- No CI check runs exist on either commit (the "gates green" claims rest on the
  ledger records).
- The "1119 to 371" annotation drop mixes merging with the page-count drop, so it
  is not a measure of merging alone.
- Subset tags come from sha256 mapped to six letters; collisions are possible in
  theory and no test proves name uniqueness across a document.
- The "logs stay under 1KB" claim is enforced for one data-URI test only;
  production caps the URI metadata, not every warning line.

## Leftovers and follow-ups found by this audit

1. `frontend/public/data/issues.json` is stale and ships to the site as
   `docs/data/issues.json`: 17 rows still say WOFF2 is skipped and one says two
   direct dependencies. `claim-scan` only scans `frontend/src/data/content`
   (`Makefile:124-133`), so it does not catch data files. Fix the data file and
   rebuild `docs/`, or widen the scan.
2. `AGENTS.md` stale paragraphs: file-size bullet (2,497 / 2,171 / 2,058 versus
   the allowlist's 2349 / under-limit / 2010) and the plans partition range
   ending at 0.2.4.
3. Knowledge-base wiki pages still describe WOFF2 as rejected. Local-only, but
   golden rule 6 says code and KB move together.
4. Ledger drift: plan 01 line 4 says the docs wave is pending while lines
   181/309 say it is done; plan 01 line 19 calls root artifacts untracked; the
   commit message says `layout.go` 2501 to 2342 while the committed parent is
   2497 to 2342.
5. Deferred by design: the fixed-header chain offset (phase 9.4c), catalogue
   map-scan extension (240 of 354 arms), vertical partial-overlap clipping, and
   prefixed-last vendor alias overrides.
6. Release mechanics: `VERSION` is still `0.2.6` and `CHANGELOG.md` has no 0.2.7
   entry. The v0.2.7 release flow must start at `RELEASE.md`.
7. Committed binary artifacts (two PDFs, three logs, seven PNGs, about 1.5MB in
   commit A alone) grow repo history on every re-run and have no `.gitignore`
   rule. Intentional for evidence, but the drill script dirties tracked files.
8. No golden fixture covers image payload safety, stacking order, or the `<p>`
   auto-close. Coverage for those is unit tests plus the live drill only.

## Key files for a follow-up read

| Subsystem | Files |
|-----------|-------|
| Image safety | `internal/layout/style_paint_props.go`, `layout_flow.go`, `background_image.go`, `image_payload_safety_test.go` |
| Paint order | `internal/layout/paint_order.go`, `layout_stacking.go`, `paint_order_test.go` |
| Fonts/URLs | `internal/css/url_resolve.go`, `internal/convert/prepare/styles.go`, `internal/pdf/woff.go`, `fonttype0.go` |
| Layout metrics | `internal/layout/style_values.go`, `style_flex_props.go`, `layout_census.go`, `relative_percent.go`, `overflow_clip.go` |
| Parser | `internal/html/html.go` |
| Writer/CLI | `internal/pdf/annots.go`, `pdf.go`, `internal/settings/settings.go`, `internal/app/pdf.go` |
| Evidence | `scripts/real_site_drill.sh`, `scripts/pdf_page_forensics.py`, `plans/0.2.7/learncpp/evidence/` |
