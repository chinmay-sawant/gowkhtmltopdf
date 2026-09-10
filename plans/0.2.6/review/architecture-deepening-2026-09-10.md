# 0.2.6 review - architecture deepening (2026-09-10)

> **Parent:** `plans/0.2.6/48-canonical-0.2.6-css-coverage.md` - v0.2.6 CSS coverage ledger
> **Status:** audit-only. Every active row is `[ ]` or `[~]`; no source changed in this wave.
> **Estimated effort:** 8-12 focused engineering days for P0 + P1; P2 rows are independent small slices.
> **Date:** 2026-09-10
> **Lens:** `skills/improve-codebase/architecture-deepening/SKILL.md` (deep modules, seams, ownership, locality)
> **Snapshot:** HEAD `30bd3d6bdcdf9b61a635485ef5dde708583a70`, branch `master`, clean tree
> **Prior waves:** 0.2.0 ARC rows, 0.2.4 ARC-14..23, 0.2.6 review 2026-08-28, Go design patterns 2026-09-08. Closed rows are not re-filed without current-source proof.

---

## Overview

Six read-only explore agents each owned one disjoint slice of the tree and scored it against the
architecture-deepening rubric (deletion test, one-vs-two adapters, interface is the test surface,
locality, ownership, sentinel identity). The lead re-read every P0 and P1 row against current source
before promoting it. No agent ran `make`, `go test`, or `go build`; no file was edited.

| # | Slice | Score | Scope |
|---|-------|------:|-------|
| 1 | Layout core (style, flow, measure) | 6/10 | `style*.go`, `layout*.go`, `inline*.go` |
| 2 | Layout paint + pagination | 7/10 | `paint*.go`, `flex.go`, `grid.go`, `multicol.go`, `transform.go` |
| 3 | Convert + imageout + svg + outline | 7/10 | `internal/convert/**`, `internal/imageout/**`, `internal/svg/**`, `internal/outline/**` |
| 4 | PDF writer + pdfprofile | 6.5/10 | `internal/pdf/**`, `internal/pdfprofile/**` |
| 5 | CSS + HTML + load + line | 7/10 | `internal/css/**`, `internal/html/**`, `internal/load/**`, `internal/line/**` |
| 6 | Root API + app + cli + settings + errs | 7.5/10 | root `*.go`, `internal/app/**`, `internal/cli/**`, `internal/settings/**`, `internal/errs/**` |

Counts: about **31 raw findings** in, **22 active rows** out (2 P0, 7 P1, 12 P2, 1 P3),
**7 parked**, **10 refused** lookalikes. Cap is 25 active rows.

## Rating: 6.5 out of 10

Architecture score from this wave's evidence. 10 means no open P0/P1 and no unowned seam in the
scoped packages; each area below is scored from current source, not from taste.

| Area | Weight | Score | Why |
|------|-------:|------:|-----|
| Core pipeline depth | 0.20 | 7.5 | One job seam, small `layout.Options/Result/Op` surface, genuine `render.Pipeline` with two adapters, deep font+shaping module. Minus for three pagination owners. |
| Output integrity | 0.20 | 6.0 | `ARC-24` re-aims body links under copies; `ARC-28` truncates an existing output before failing; `ARC-41` drops a requested Info key. |
| Image adapter parity | 0.15 | 5.5 | `ARC-26`, `ARC-27`, `ARC-28`: zoom, media viewport units, and preflight dimensions are all silently wrong or missing on the image path. |
| Pagination ownership | 0.15 | 6.0 | `ARC-29` multicol height fork, `ARC-30` three bucketers, `ARC-38` three ownership predicates. |
| Parsing and trust boundary | 0.10 | 6.5 | Single loader seam and bounded HTML depth, but `ARC-25` process-global retention, `ARC-31` missing CSS depth bound, `ARC-33` inline cap bypass. |
| API and entrypoint contracts | 0.10 | 7.5 | One request shape per mode, clone at the boundary, Policy A holds. `ARC-44` and `ARC-45` show root-vs-engine predicate drift. |
| Docs and tests as proof | 0.10 | 6.5 | Golden corpus is real evidence; `ARC-42` covers three docs that contradict source, and the image viewport test named for a closed row cannot fail. |

Weighted total: 6.475, reported as **6.5/10**.

What moves it: phase 1 (both P0s plus the P1 input and pagination rows) lands about 7.5. Phase 2 seam
work lands about 8. The last points need the parked items and allocation evidence, not blind fixes.

## Frozen snapshot

| Field | Value |
|-------|-------|
| Branch | `master` |
| HEAD | `30bd3d6bdcdf9b61a635485ef5dde708583a70` |
| Tree | clean; nothing excluded |
| Date | 2026-09-10 |
| Product ceiling | controlled-report renderer. No JS. No CGO. |
| Slice count | 6 read-only agents, one per slice |

## Executive summary

The engine's core shape is sound: one request per mode (`convert.Request`, `imageout.Request`), one
3-stage lifecycle with two real adapters, and layout/pdf export small surfaces (`Options`, `Result`,
`Op`; the font and shaping half of `internal/pdf`). The friction is in five repeated themes:

1. **Positional identities break when pages move.** `ARC-24` stores link destinations as page
   indices; copies and reorder run after link assembly. `ARC-39` reconstructs heading-to-`StructElem`
   identity by index at two call sites with different arithmetic.
2. **Process-wide state and missing bounds.** `ARC-25` retains every parsed document in a
   package-global cache that is never evicted. `ARC-31` has no CSS parse depth cap. `ARC-33` lets
   the `inline:` branch bypass the loader body cap.
3. **Image mode drops advertised inputs.** `ARC-26` zoom, `ARC-27` media viewport units, and
   `ARC-28` preflight dimensions each fail silently on the image path while working on PDF.
4. **Pagination membership is re-derived per pass.** `ARC-29` multicol uses `Options.Height`,
   `ARC-30` three bucketers disagree at boundaries, `ARC-38` three predicates decide op ownership.
5. **Style resolution forks.** `ARC-34` font shorthand, `ARC-35` pseudo-element custom properties,
   and `ARC-43` style overrides each have two paths where one would do.

No P0 security issue was proven in this wave. Both P0s are correctness and retention defects.

## Phase 1: Integrity

Fix first. One row is one behavior defect with a current-source location.

### 1.1 Wrong output and unbounded retention

- [ ] **ARC-24 · P0 · defect - Body and TOC link destinations are positional and re-aim after copies**
      `internal/pdf/pdf.go:471`, `:424`, `:1191-1207`; `internal/convert/links.go:419-421`;
      `internal/convert/pdf_pipeline.go:50-58`, `:179`.
      `AddLinkDest(rect, page int, ...)` stores a page index (`pdf.go:471`). `DuplicatePage` copies
      the annotation slice unchanged (`pdf.go:424`). `writeAnnotDest` resolves `doc.pages[arg.destPage]`
      at finalize (`pdf.go:1196`), after `ReorderPages` has already permuted `d.pages` (`pdf.go:371`).
      `assembleLinks` runs at `pdf_pipeline.go:50`; the page plan is not built until
      `assembleDocument` at `:179`, and copies/materialization run at `:58`, so body and TOC passes
      cannot remap. Only the header/footer path remaps (`internal/convert/hf.go:619-622`).
      Change: store destination identity, not position. `AddLinkDest` takes a page handle or the plan
      is built before links and threaded through both passes exactly as HF does.
      Proof: new `TestBodyLinkDestRemapCopiesNonCollate` in `internal/convert` (Copies=2,
      Collate=false, one body internal link; assert the copy's `/Dest` points at the right page),
      plus writer-level `TestLinkDestSurvivesReorder` in `internal/pdf`.
      Depends-on: none. Not: a mutex on `Document`, a global page index, or a paint-sink interface.

- [ ] **ARC-25 · P0 · defect - `internal/css` retains every parsed document in a process-global cache**
      `internal/css/match.go:32-63`.
      `sibCache = make(map[*html.Node]*parentSibCache)` is package level and guarded by `sibMu`.
      `getParentCache` inserts and never deletes; no `delete`, `clear`, or reset exists. Keys are
      per-parse `*html.Node` pointers, and the values hold four more node-keyed maps, so every
      document ever matched stays reachable. Real callers: layout selector matching and outline
      matching run on every conversion, including long-lived library and server embeds. The 0.2.6
      CSS-07 row chose a per-parent index to fix O(n) scans; cache lifetime was never decided.
      Change: make the sibling index owned by the conversion (a matcher/index value created per job
      and passed to matching), or bound the global with a small eviction policy like the PDF regex
      cache. Proof: `TestSiblingCacheReleasesDocuments` (match one tree, assert the cache is empty
      after the job), or a two-conversion `runtime.ReadMemStats` check near baseline.
      Depends-on: none. Not: rewriting selector matching as id/class dispatch or a matcher framework.

### 1.2 Image mode drops advertised inputs

- [ ] **ARC-26 · P1 · defect - Image mode silently ignores `--zoom`**
      `internal/cli/flags.go:325`; `document.go:533`; `internal/imageout/imageout.go:251-269`;
      `internal/layout/layout.go:113`.
      The flag is registered `ModeBoth` (`flags.go:325`), `ImageDocument.Zoom` is public
      (`document.go:158`), the C and Python bindings map it, and the matrix marks it "Both |
      Supported". `internal/imageout` contains zero `Zoom` reads; `layoutOptions` never sets
      `layout.Options.Zoom`. PDF consumes the same object field at `internal/convert/convert.go:611`.
      Change: add `Zoom` to `imageout.RenderOptions`, fill it from
      `req.Objects[0].Load.ZoomFactor`, and pass it to `layoutOptions`. Validate it with the layout
      finite-positive rule. Proof: `TestRenderZoom` asserts a text op is twice as wide at zoom 2.
      Depends-on: none. Not: an `ImageGlobal.Zoom` field or copying smart-shrink into imageout.

- [ ] **ARC-27 · P1 · defect - Image prepare viewport is in pixels where the media matcher expects points**
      `internal/imageout/imageout.go:1834-1860`; `internal/css/media.go:11-12`, `:153`;
      `internal/convert/convert.go:533-540`.
      Image mode passes `imageSet.Width` (CSS pixels, default 1024) straight into
      `prepare.BuildOptions`, while PDF passes `geom.contentW/contentH` in points. `MediaMatches`
      documents and evaluates `widthPt/heightPt`; layout is also in points
      (`imageout.go:258-260`). Linked and imported sheets are gated by that call, so
      `<link media="(min-width: 1300px)">` loads against a 1024pt viewport even though the real
      1024px viewport fails, and `(max-width: 1300px)` is dropped. The test named for the closed
      image-viewport row (`imageout_test.go:568-593`) only asserts `len(prep.Sheets) != 0` on an
      inline `@media` rule that prepare never gates, so it cannot fail. This is current-source
      incompleteness of that closed row, not a re-file.
      Change: one resolved viewport helper in imageout returning both representations (px for the
      raster context, pt for `prepare.BuildOptions`), with height defaulting to width.
      Proof: `TestPrepareImageDocumentLinkMediaUsesLayoutViewport` (`Width: 1024`,
      `(max-width: 1100px)` collected, `(min-width: 1100px)` skipped). Depends-on: none.
      Not: a second media matcher; the point contract is right, the caller is wrong.

- [ ] **ARC-28 · P1 · defect - App image preflight misses dimensions, so a bad request truncates output first**
      `internal/app/image.go:38-61`; `internal/imageout/request.go:49-66`;
      `internal/imageout/imageout.go:129-136`; `internal/cli/cli.go:83-95`.
      `Request.Validate` checks sink, object count, and renderable sources only.
      `RenderOptions.Validate` owns the negative dimension and crop predicate, but it runs inside
      `RenderContext` after `RunRequest` has loaded the input. `cmd.OpenOutput` uses `os.Create`,
      which truncates an existing file. So `gowkhtmltoimage --width -1 -o report.png report.html`
      destroys report.png, loads the source, then errors. `01-entrypoints-cli.md:304-306` promises a
      bad request never destroys a previous output file.
      Change: put the non-negative dimension and crop predicate in `imageout.Request.Validate`
      behind one helper that `RenderOptions.Validate` also calls, so the app preflight covers it
      before `OpenOutput`. Optionally range the settings setters. Proof:
      `TestRunImageRejectsNegativeWidthBeforeOpeningOutput` with an existing output file and a
      content assertion. Depends-on: none. Not: a third copy of the check in `app` or `cli`.

- [ ] **ARC-32 · P1 · defect - `web.images` is registered on three layers but gated on one each**
      `internal/settings/reflect.go:811-840`; `internal/convert/convert.go:559-560`, `:584`;
      `internal/imageout/imageout.go:1882-1884`.
      `registerWebKeys` registers `web.images` on the global, object, and `ImageGlobal` tables.
      PDF reads only `Global.Web.Images` (also for header/footer), image mode reads only
      `ImageGlobal.Web.Images`, and no production code reads `obj.Web.Images`. No CLI flag registers
      the key (`rg images internal/cli/flags.go` is empty), while `10-imageout-svg.md` describes a
      `--no-images` flag for image mode. `Set("web.images", "false")` on an object silently renders
      images in both modes.
      Change: one resolver shaped like `settings.ResolveMedia` folding global, mode, and object
      layers, consumed by the three fetch closures, and register the flag or correct the docs in the
      same change. Proof: a layer-folding table test in `internal/settings` plus engine tests for
      object-level PDF gating and global-level image gating. Depends-on: none.
      Not: a per-mode boolean copy in each pipeline; the key already has a table.

### 1.3 Pagination and parse ownership

- [ ] **ARC-29 · P1 · defect - Multicol paginates against `Options.Height` while Paint paginates against page content height**
      `internal/layout/multicol.go:337`, `:552-566`; `internal/layout/paint.go:152`, `:170`;
      `internal/layout/layout.go:97-99`.
      `flowMulticolSegment` reads `pageH := e.opts.Height`, documented as "viewport height in points
      (for % heights)". Every other page break is decided in `Paint` from
      `contentH := opts.PageHeight - opts.MarginTop - opts.MarginBottom`. Production happens to keep
      both equal, but the package's own tests do not (`layout_test.go:53` uses `Height: 800`,
      `paintOpts()` uses `PageHeight: 842`), and `TestMulticolLinesDoNotStraddlePages`
      (`multicol_test.go:212-231`) proves the straddle guarantee without ever calling `Paint`.
      A column line snapped to y=800 with a strip taller than 42pt crosses Paint's 842 boundary;
      `snapCrossingTextOps` then moves text while column rules and chrome stay.
      Change: give page geometry one owner. Add an explicit page content height to `Options` (or
      record `contentH` on `Result`), have `Paint` reject a mismatch, or have multicol emit a
      page-top hint that `paginateOps` consumes. Proof: `TestMulticolPageHeightMustMatchPaint`.
      Depends-on: none. Not: moving column balancing into Paint.

- [~] **ARC-30 · P1 · risk - Three op-to-page bucketers and fourteen raw `int(Y/contentH)` copies disagree at boundaries**
      `internal/layout/paint.go:305` (adds `layoutEpsilon`); `internal/layout/paint_flow_index.go:276`
      (no epsilon, read by every shift pass); `internal/layout/paint_pagination_seal.go:554`, `:559-561`.
      Raw `int(Y/contentH)` remains at `paint_flow_breaks.go:150-151,548,558,726,786,877,889`,
      `paint_flow_tables.go:62-64,112,262`, `sticky.go:121`, and `paint.go:853`. The epsilon comment
      at `paint.go:286-294` documents a real two-band fill bug fixed in one bucketer only, and
      `layout_test.go:1296` documents a hang from the unguarded form.
      Validation step (this is why the row is `[~]`): a table test at `y = k*contentH - 1e-9`,
      `y = k*contentH`, and `y = 21*785.197 - 1e-6` must first prove disagreement between
      `pageBuckets`, `buildFlowOpIndex`, and `pageIndexedOps` before the fix is sized.
      Change: one `flowPageOfY(y, contentH, edgeBias)` plus one `bucketOpsByPage` owner on `Result`,
      folding `maxNonFixedOpPage`/`pageOpCounts`/`fillPageOpBuckets`. Proof after validation: the
      boundary table test plus one golden with a split fill starting on a boundary, to pin the bias.
      Depends-on: none. Not: changing `checkedFlowPageOfY`'s `maxFlowPageIndex` guard.

- [~] **ARC-31 · P1 · risk - CSS recursive parse/match has no depth bound and rescans nested functional pseudos quadratically**
      `internal/css/has.go:68-88`; `internal/css/selector_parser.go:397-410`, `:141-146`, `:317-327`;
      `internal/css/container.go:288-394`; `internal/css/css.go:613-705`.
      `parseSelectorListStrict` calls `parseSelectorCtx`, which reaches `appendNotPseudo` /
      `appendIsWherePseudo` and calls back, while each level also scans to the matching paren. That
      is O(n) per level and O(n^2) total. `internal/html` owns `maxElementDepth = 1024`
      (`html.go:237-254`) and the CSS package already caps var() chains at 16 and `@import` depth
      at 8, so the missing cap is an ownership gap. The CSS fuzz guard skips inputs over 64 KiB
      (`fuzz_test.go:24-27`).
      Validation step (why `[~]`): the `:not(` x 200k probe must first reproduce unacceptable time
      or stack growth, then the fix is sized.
      Change: one `maxParseDepth` shared by compound pseudos, container conditions, and nested
      at-rules, rejecting before `splitSelectorChain`. Proof after validation:
      `TestParseNestedFunctionalPseudoDepth` with a bounded timeout. Depends-on: none.
      Not: a tokenizer rewrite or a byte-size check inside `css`; size belongs to load/prepare.

- [ ] **ARC-33 · P2 · defect - The `inline:` loader branch bypasses the body size cap**
      `internal/load/load.go:966-987` versus `:920-923`.
      The `InlineHTML` branch calls `checkBodyLimit`, and `data:` uses `decodeDataURLLimited`, but
      `strings.HasPrefix(target, "inline:")` builds a `Resource` with the caller's bytes and no cap.
      Both branches produce `KindInline`; only one is bounded. `GuessURL` advertises the prefix
      (`load.go:314-322`) and callers use it.
      Change: one inline-resource constructor that always applies `checkBodyLimit`, used by both
      branches. Proof: `TestInlinePrefixHonorsBodyLimit` with `MaxBodySize = 4`.
      Depends-on: none. Not: removing the wkhtmltopdf-compat `inline:` prefix.

## Phase 2: Seams and shared forks

One owner per rule. Behavior already works on the happy path.

- [ ] **ARC-34 · P2 · defect - `font` shorthand bypasses cascade precedence and the apply tables already disagree**
      `internal/layout/style_cascade.go:551-588` (expansions), `:915-930` (`font` always applied
      last), `:1002-1012` and `:1042-1052` (two hand-maintained lists);
      `internal/layout/style_values.go:47-77` (`parseFontShorthand` overwrites longhands).
      `applyCascadeDeclaration` expands `list-style`, logical box, and box shorthands before the
      cascade so source order decides. `font` was never added, so
      `.total { font: 12pt serif } .total { font-size: 20pt }` renders 12pt. The two apply lists
      list the same ~32 properties and already disagree (`display` is in the first list only, so it
      is applied twice).
      Change: decide precedence once. Either expand `font` into parsed longhands like
      `expandListStyleDeclaration`, or give the cascade an ordered declaration record; derive the
      second apply pass from one ordered table. Proof: `TestFontLonghandAfterShorthandWins`
      following the shape of `TestCascadeShorthandRespectsSourceOrder`. Depends-on: none.
      Not: a visitor or registry framework for CSS properties.

- [ ] **ARC-35 · P2 · defect - Pseudo-element styles skip custom property resolution**
      `internal/layout/pseudo_content.go:96-116`; `internal/layout/style.go:742-743`;
      `internal/layout/style_cascade.go:543-548`.
      The element path does `sty.CustomProps = mergeCustomProps(...)` and
      `raw = resolveRawVars(raw, sty.CustomProps)` before applying. `pseudoStyle` calls
      `cascadePseudoRaw`, `inheritProps`, `applyFontProps`, and `applyRestProps` without either
      step, so `p::after { color: var(--accent) }` reaches the color parser as the literal
      `var(--accent)` and is dropped. Real callers: `internal/layout/inline.go:117`,
      `layout.go:1590`, list and quote generation.
      Change: extract the raw-to-used apply sequence from `resolveElementStyle` into one function
      shared by element and pseudo paths. Proof: `TestPseudoElementResolvesCustomProperties`.
      Depends-on: none. Not: a plugin for generated content.

- [ ] **ARC-36 · P2 · defect - Link and `@import` media gate against pre-`@page` geometry while rule media uses post-`@page` geometry**
      `internal/convert/convert.go:527-557`; `internal/convert/prepare/styles.go:119`, `:200-205`,
      `:362-371`; `internal/layout/style_cascade.go:359`; `internal/convert/hf.go:356`.
      Prepare gates linked and imported sheets with the viewport captured before
      `applyCSSPageMargins` changes it (`convert.go:557`), while layout re-evaluates stored rule
      media with the final viewport. With `@page { margin: 0 }` on A4, content width moves from
      about 538pt to 595pt, so `<link media="(min-width: 560px)">` is dropped before fetch while
      layout would match it. Header/footer sheets already use post-`@page` geometry, so one
      predicate sees two snapshots inside one run.
      Change: one owner for the media viewport: resolve the unnamed `@page` box from inline sheets
      before gating links and imports, or gate on media type only and let the cascade decide
      feature queries. Proof: `TestLinkMediaGateUsesFinalPageBox`. Depends-on: none.
      Not: deleting link gating (it saves fetches) or moving `@page` handling into `css`.

- [ ] **ARC-37 · P2 · defect - Auto-height bottom chrome has five homes and they already disagree**
      `internal/layout/layout.go:1487-1501`; `internal/layout/flex.go:95-97`;
      `internal/layout/multicol.go:227-231`; `internal/layout/grid.go:684-710`;
      `internal/layout/layout_tables.go:97`; `internal/layout/layout_measure.go:740`.
      `buildBlock` adds bottom border only for border-image; flex and multicol add only
      `PaddingBottom`; grid and table cells add `BorderBottom.Width`. An auto-height bordered block
      or flex container is shorter than a bordered grid or table with identical content by the
      bottom border width, and the border paints half outside the reported box.
      Change: one used-height resolver that takes the content bottom and returns the border-box
      height for auto, definite, min, and max, called by every formatting context.
      Proof: `TestAutoHeightIncludesBottomBorder` comparing block and grid and checking the next
      sibling's top. Depends-on: none. Not: a new formatting-context abstraction or `layout/flex`.

- [ ] **ARC-38 · P2 · defect - "Op owned by this box" is derived three ways; chrome repair does not know outline shapes**
      `internal/layout/paint_pagination_chrome.go:345-353`;
      `internal/layout/overflow_clip.go:226-272`;
      `internal/layout/paint_pagination_seal.go:1237-1255`;
      `internal/layout/outline.go:96`; `internal/layout/layout_chrome.go:406`, `:703-724`.
      The rect-proximity predicate is copied in three passes with different tolerances. Outline ops
      are inflated and spliced into the owner's op range, and only the overflow-clip copy knows
      outline shapes (`paint_style_view.go:7-15` has no outline field). A bordered, background box
      with an outline therefore gets chrome stretched past its border box by the chrome-repair and
      seal passes.
      Change: one `opOwnedBy(op, boxNode, phase)` helper covering rect, line, masked side, outline,
      and shadow shapes, called by all three passes. Proof: `TestOutlineDoesNotStretchOwnedChrome`.
      Depends-on: none. Not: a `chromeKind` bitfield on every op or a paint-sink interface.

- [ ] **ARC-40 · P2 · defect - Op-radius resolution has a second, unscaled home in imageout**
      `internal/imageout/imageout.go:1234-1272`; `internal/layout/paint.go:1114-1120`;
      `internal/layout/border_radius.go:340-372`.
      Layout owns `opRadii`, `opRadiiY`, and `opRadiiXY`; imageout reimplements shorthand and XY
      fallback in `scaledRadii`/`scaledRadiiXY` and multiplies by `pxPerPt`. The closed 0.2.6 radius
      row collapsed the style-level resolver inside layout; this op-level raster copy was untouched
      and has no parity test. The next radii change must be edited in two packages.
      Change: export one `layout.OpRadiiXY(op)` (resolved, unscaled) and let imageout scale it, or
      stamp used radii onto `Op` so both painters read the same fields. Proof: parity test for
      shorthand, corner longhand, and Y-only ops. Depends-on: none.
      Not: moving raster paint into layout or a paint-sink interface.

- [ ] **ARC-43 · P2 · friction - Two ways to substitute a node's style, and six readers bypass the documented one**
      `internal/layout/layout.go:1124-1138` (`stylePtr`, `styleOverrides`), `:514-518`;
      `internal/layout/flex.go:136-141`, `:140`, `:1139`, `:1528`;
      `internal/layout/multicol.go:163`; `internal/layout/layout_measure.go:413-419`;
      `internal/layout/layout_chrome.go:290`; `internal/layout/layout_svg.go:158`.
      `buildWithStyle` pushes an override that `stylePtr` honors, but flex and multicol write
      synthetic nodes directly into `e.styles`, and multiple readers index `e.styles[node]`, so
      overrides are invisible to them. `e.styles` is documented as immutable after the cascade.
      Change: route every reader through `stylePtr` and keep synthetic nodes on an engine-owned
      side map or session-scoped overrides. Proof: `rg '\.styles\['` in production files returns
      only store writers, plus a test where an override changes a child's measured style.
      Depends-on: none. Not: interning `ResolvedStyle` or a second settings hierarchy.

## Phase 3: Contracts and locality

- [ ] **ARC-39 · P2 · defect - Heading-to-`StructElem` identity is rebuilt by callers with two index schemes**
      `internal/pdf/structure.go:221`; `internal/convert/links.go:204-208`;
      `internal/convert/outline.go:226-244`.
      `HeadingStructElems` returns a flat document-order slice. `links.go` zips it with headings and
      guards only `i < len(headingElems)`; `outline.go` checks equal lengths and otherwise falls back
      to page matching. The counts genuinely diverge: outline collection drops objects with
      outline flags off (`outline.go:134`), while layout tags every painted h1-h6
      (`internal/layout/tagging.go:442-444`); a cover defaults `IncludeInOutline=false`
      (`internal/settings/object_roles.go:23-24`). With a cover plus TOC, `links.go` binds the TOC
      forward link `/SD` to the wrong heading.
      Change: record the association where the structure tree is built. Tag creation already walks
      boxes with the heading node in hand, so stamp `*StructElem` on the heading (or op) once.
      Proof: `TestTOCLinkStructDestIdentity` under UA-2 with a cover h1 and TOC forward links.
      Depends-on: none. Not: a structure-tree visitor framework.

- [ ] **ARC-41 · P2 · defect - `SetInfo("Producer", ...)` is documented but finalize overwrites it**
      `internal/pdf/pdf.go:267`, `:1014-1020`; `internal/convert/pdf_pipeline.go:198`.
      `SetInfo` accepts any key; `infoDict` writes a fixed list plus `/Producer` from
      `policy.ProducerVersion()`. The only production `SetInfo` call besides Title sets Producer,
      so that call can never reach the file. `TestInfoDict` (`pdf_test.go:432`) locks the policy
      value, so the dead path stays green.
      Change: one owner per Info key. Honor a caller-set Producer as an override with the policy
      string as fallback, or make `SetInfo` reject keys outside a writer-owned allowlist and delete
      the dead call. Proof: `TestSetInfoProducer` asserts `/Producer (x)` after `SetInfo`.
      Depends-on: none. Not: a second metadata subsystem or typed Info hierarchy.

- [ ] **ARC-44 · P2 · defect - Finite setters admit NaN and Inf, so CLI values fail after the output is opened**
      `internal/settings/reflect.go:312-327`; `internal/settings/unitreal.go:51-56`;
      `document_validate.go:222-228`; `internal/layout/layout.go:127-133`;
      `internal/app/pdf.go:88-99`.
      `setFloatMin` uses `strconv.ParseFloat` and then `value < minimum`. NaN compares false and
      +Inf passes, so `--zoom nan` or `--margin-top nan` is stored. The root API rejects the same
      values up front via `finitePositive`/`finiteNonNegative`, and layout rejects them later,
      after `cmd.OpenOutput()` has truncated the destination.
      Change: reject non-finite values in the shared setters with one `finite` helper so `Set` fails
      at parse, matching the root predicates. Proof: `TestFiniteSetters` asserting
      `Set("load.zoomfactor","NaN")` and `Set("margin.top","Inf")` error.
      Depends-on: none. Not: patching every flag applier.

- [ ] **ARC-45 · P2 · defect - Root Document margins reject the `-1` auto margin the engine and CLI honor**
      `document_validate.go:200-208`; `internal/convert/hf.go:686-710`;
      `internal/settings/reflect.go:442-472`; `document_test.go:223-224`; `samples.md:62`.
      `validateMargins` requires finite non-negative values, but `hf.effectiveMargins` treats
      negative top and bottom as auto (measure the band and reserve it), and the CLI margin setter
      accepts negatives. A library user cannot express the auto header/footer margin that the same
      job gets through the CLI.
      Change: share one predicate. Either accept finite negative top and bottom as the engine's auto
      sentinel in `validateMargins`, or reject negatives in settings and delete auto everywhere;
      the engine and samples point at the first option. Proof: `TestDocumentAutoMargin` asserts
      `Validate() == nil` and keeps `-1` in `toPDFRequest`. Depends-on: none.
      Not: an `AutoHeaderMargin` bool; the sentinel already exists.

## Phase 4: Docs honesty

- [ ] **ARC-42 · P3 · friction - Architecture docs contradict current source after the 0.2.6 splits**
      `documentation/architecture/09-pdf-writer.md:361-366` claims `internal/pdf` imports only
      stdlib, shaping, assets, and `pdfprofile`, but `internal/pdf/registry.go:11-12` imports
      `internal/settings` and `internal/line` (the `RegistryFromGlobal` exception that ARC-18 made
      deliberate). `documentation/architecture/06-css.md:355`, `:443` says `@import` and
      `:is()`/`:where()` are unsupported while `internal/css/css.go:273-274`,
      `selector_parser.go:345-410`, and `internal/convert/prepare/styles.go:145-261` implement them.
      `documentation/architecture/04-load.md:50`, `:231-235` describes a 1322-line file with no
      `errs` import; source is 1803 lines and imports `errs` (`load.go:26`).
      `documentation/architecture/01-entrypoints-cli.md:386-410` lists about twenty test names that
      do not exist; `documentation/compatibility-matrix.md:611` says `--allow` is not registered in
      image mode while `internal/cli/flags.go:247` registers it `ModeBoth`.
      Change: refresh the contradicted claims (not a full regeneration) and align the DAG note in
      `skills/improve-codebase/references/gowkhtmltopdf.md:34`. Proof: re-grep each claim against
      source and keep `make claim-scan` clean. Depends-on: none.
      Not: a generated import-graph linter.

## Phase 5: Closure gates

This wave is documentation-only, so no lint or test row is checked here. The rows below activate
when any named ID is implemented.

- [ ] **GATE-01** Before closing any non-doc row, record `make lint` and `make test` exit 0 on the
      final implementation. Leave the row unchecked if either fails.

- [ ] **GATE-02** Layout, paint, and pagination rows (ARC-29, ARC-30, ARC-37, ARC-38, ARC-43) also
      pass `go test ./internal/layout/ -count=1` and
      `go test ./internal/convert -run 'TestGoldenCorpus' -count=1`; run `make golden` when a
      fixture or paint op changes.

- [ ] **GATE-03** Rows that add a golden fixture (none planned here) need a `fixturePageBounds` row
      in `internal/convert/golden_test.go` or the walker fatals.

## Dependencies

```text
ARC-24 (link identity)      -- independent; touches pdf + convert link passes
ARC-25 (sibCache)           -- independent; touches css + its two callers
ARC-26/27/28 (image inputs) -- independent of each other; all touch imageout/app/cli
ARC-29 (multicol height)    -- independent; must land before ARC-30 touches bucketers
ARC-30 (bucketers)          -- validation table first, then one owner
ARC-31 (css depth)          -- validation probe first, then one cap
ARC-32 (web.images)         -- independent; settings + convert + imageout + cli/docs
ARC-33 (inline cap)         -- independent
ARC-34/35/43 (style paths)  -- independent; 35 and 43 share the style apply seam
ARC-36 (media viewport)     -- independent; convert + prepare
ARC-37 (height resolver)    -- independent; layout only
ARC-38 (op ownership)       -- independent; layout paint passes
ARC-39 (heading identity)   -- independent; layout tagging + pdf + convert
ARC-40 (radius parity)      -- independent; layout + imageout
ARC-41/42/44/45 (contracts) -- independent
GATE-01..03                 -- after any implementation, before any [x]
```

Suggested first slice: ARC-24, ARC-25, ARC-28, then the image input cluster (ARC-26, ARC-27,
ARC-32). Close both P0s before any P2.

## Refused

Lookalikes that are decided tradeoffs or out of scope. Do not re-file without current-source
regression.

| Refused | Why |
|---------|-----|
| `imageout` -> `convert/prepare` + `convert/render` | Known live leak tracked since ARC-06 direction. Not a new invention. |
| Paint-sink interface for page painter vs band paint | Decided. The two adapters share ops and paint policy, not a paint loop. Extract a helper if needed. |
| `layout/flex` or `layout/table` packages | Package-per-file DAG refused. Keep same-package helpers. |
| Second settings hierarchy | Product decision; `styleGroups` and the dotted tables are the owners. |
| Plugin/visitor framework for CSS properties | Refused. Use the apply tables. |
| Mutex on layout or `pdf.Document` | Single-goroutine engine, documented. |
| Pixel-diff goldens as architecture proof | Goldens are structural. |
| Dual public stories (`Document` + `convert.Request`) as a bug | Tax, not a defect; the request is the one internal job type. |
| `x/net/html` swap, CGO HarfBuzz, `gofumpt`, live network in tests | Decided in the calibration reference. |
| Document-global Y shifts, site-specific MediaWiki cascade hacks | Refused. |

## Parked (weaker than the 22 rows)

Re-open only when a named ID touches the same file and the parked item blocks it.

- Paint mutation contract is asserted both ways, so convert clones `Result` and builds a scratch
  PDF just to count pages (`internal/convert/toc.go:143-164`, `:266`;
  `internal/layout/paint.go:168-206`; `internal/layout/layout.go:183-212`).
- Two inert-op sentinels for one condition: `OpUnknown` (`layout.go:334-337`) and `opKindNoop`
  (`layout_measure.go:1101-1108`); `DeactivateOp` is exported and uses the private one.
- `paintOrderSubset` (`paint_order.go:19-27`) has no production caller, and
  `rasterPaintOrder` (`imageout.go:543-552`) is a one-line delegate into `layout.PaintOrder`.
- `imageout.Request.Now` is set by `ImageDocument.Now` and never read (`request.go:23-29`,
  `document.go:540-541`); `library-api.md:265` promises it follows Document policy.
- `objectRenderer` and `hfLoader` are single-implementation interfaces whose declared test double
  never appears (`convert.go:300-320`, `:344-391`, `pdf_pipeline.go:23`).
- `@import` base rebinding and header/footer loads disassemble `load.ResourceContext` through
  `Loader()`/`PageLoad()` (`prepare/styles.go:247-261`, `hf.go:313-320`,
  `load.go:224-236`, `seams_test.go:147-151`).
- Profile-to-base-version table is encoded twice (`convert.go:189-204`,
  `pdf/policy.go:190-221`) with two error sentinels for one invalid profile.

## Evidence method

- Six `explore` agents ran read-only on 2026-09-10 against frozen HEAD
  `30bd3d6bdcdf9b61a635485ef5dde708583a70`, one slice each, and returned finding-schema blocks plus
  a slice score. No agent ran build, test, or make; no file was edited.
- The lead re-read every P0 and P1 claim in current source:
  `ARC-24` (`pdf.go:424,471,1191-1207`, `links.go:419-421`, `hf.go:619-622`,
  `pdf_pipeline.go:50-58,179`), `ARC-25` (`css/match.go:32-63`), `ARC-26` (zero `Zoom` reads in
  imageout), `ARC-27` (`imageout.go:1834-1860`, `css/media.go:11-12`, `convert.go:533-540`),
  `ARC-28` (`request.go:49-66`, `image.go:38-61`, `cli.go:83-95`), `ARC-29` (`multicol.go:337`,
  `layout.go:97-99`), `ARC-30` (`paint.go:305`, `paint_flow_index.go:276`,
  `paint_pagination_seal.go:554`), `ARC-31` (no depth constant in `internal/css`; HTML owns its
  own), `ARC-32` (`settings/reflect.go:811-840`, three read sites).
- P2 rows were spot-checked where the fix shape depended on it: `font` apply order
  (`style_cascade.go:915-930`), pseudo custom properties (`pseudo_content.go:96-116`), style
  overrides versus direct map reads (`layout.go:1124-1138`, `flex.go:140`, `multicol.go:163`).
- This is a documentation-only wave. Per `skills/phase-wise-checklist/SKILLS.md`, lint and test
  were not run and GATE rows stay unchecked.

## What this wave did not do

- No implementation, no commits, no history changes.
- Only the architecture-deepening lens ran. The pack's extension-seams and go-practices lenses,
  plus perf-review, ponytail, and critical-go-review, were not run in this wave.
- No second status document beside this ledger and its README entry. The HTML report beside this
  ledger renders the same rows; the ledger is canonical.
