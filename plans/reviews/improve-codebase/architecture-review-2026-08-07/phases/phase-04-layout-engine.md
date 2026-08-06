# Phase 4 — Layout engine

> **Parent:** [`architecture-review-2026-08-07.md`](../architecture-review-2026-08-07.md) — canonical architecture-review ledger
> **Status:** pending (findings gathered 2026-08-07 by 7 explore agents; remediation not started)
> **Depends on:** see phase map in ledger
> **Evidence archive:** raw agent findings were archived off-repo on 2026-08-07; every row below carries its Before/After snippets inline

---

## Overview

…

## Checklist

- [ ] **P4-01** — Resolve each <img> once per Layout run — one decode path, one cache, one type
- [ ] **P4-02** — Collapse content-box/used-size math duplicated ~10× across formatting contexts
- [ ] **P4-03** — Extract table column-sizing out of buildTable into a pure function
- [ ] **P4-04** — Give cells a stored start row; stop re-scanning row list 4×
- [ ] **P4-05** — One text-wrap policy for inline layout and intrinsic measurement
- [ ] **P4-06** — Thread the inline-formatting width into inline-block measurement
- [ ] **P4-07** — Stop splicing the op list per box — defer background/border chrome

---

<a id="p4-01"></a>
## P4-01 — Resolve each <img> once per Layout run — one decode path, one cache, one type

- [ ] **P4-01** — Resolve each <img> once per Layout run — one decode path, one cache, one type

- **Locations:** `internal/layout/layout.go:1160-1236` (`buildImage`), `layout.go:2452-2499` (`measureImageWidth`/`measureLargestImageWidth`), `inline.go:859-866` (inline img)
- **Evidence sources:** area-5 F1

---

### Evidence — F1

- **Severity:** high
- **Location:** `internal/layout/layout.go:1160-1236` (`buildImage`), `layout.go:2452-2499` (`measureImageWidth`/`measureLargestImageWidth`), `inline.go:859-866` (inline img)
- **Current (verbatim):**
```go
func (e *engine) buildImage(n *html.Node, st ResolvedStyle, x, y float64) *box {
	b := &box{node: n, style: st, kind: "replaced", x: x, y: y}
	src := n.Attribute("src")
	if src != "" && e.opts.Images != nil {
		if data, err := e.opts.Images(src); err == nil {
			if png, pw, ph, err := svg.Rasterize(data, 1024); err == nil {
				b.imgSrc, b.imgData, b.imgJPEG, b.imgW, b.imgH = src, png, false, pw, ph
			} else if w, h, jpeg, ok := imageDims(data); ok {
				b.imgSrc, b.imgData, b.imgJPEG, b.imgW, b.imgH = src, data, jpeg, w, h
			}
		}
	}
```
…and the measure twin (`layout.go:2452-2475`) calls `e.opts.Images(src)`,
`svg.Rasterize(data, 1024)` and `imageDims(data)` again for the same `src`.
- **Future:**
```go
// imageRef is the resolved form of one <img src>: bytes + intrinsic size,
// resolved at most once per Layout run (measure passes and build passes share it).
type imageRef struct {
    src    string
    data   []byte
    w, h   int
    isJPEG bool
}

// resolveImage fetches (once) and decodes (once) src; nil on any failure.
func (e *engine) resolveImage(src string) *imageRef {
    if src == "" || e.opts.Images == nil {
        return nil
    }
    if e.imgCache == nil {
        e.imgCache = map[string]*imageRef{}
    }
    if ref, ok := e.imgCache[src]; ok {
        return ref
    }
    data, err := e.opts.Images(src)
    if err != nil {
        return nil
    }
    ref := &imageRef{src: src, data: data}
    if png, pw, ph, err := svg.Rasterize(data, 1024); err == nil {
        ref.data, ref.w, ref.h = png, pw, ph
    } else if w, h, jpeg, ok := imageDims(data); ok {
        ref.w, ref.h, ref.isJPEG = w, h, jpeg
    }
    e.imgCache[src] = ref
    return ref
}
```
`box` and `inlineItem` replace their five/four image fields with `img *imageRef`;
`buildImage` and `measureImageWidth` both become one-liners over `resolveImage`.
No exported API change (`Layout`/`Options` untouched); cache lifetime is the
single `engine` run, so freshness semantics are identical.
- **Why:** Today the *same* image is fetched and decoded 2–4× per run: a float's
`placeFloat` → `measureLargestImageWidth` → `measureImageWidth` fetches the URL
again, then `build` → `buildImage` fetches it again; a table-cell image does the
same via `measureCellMinMax` vs. `emitCell`. `loader.FetchSub`/`imagesFn`
(`internal/convert/convert.go:327`) do **not** cache — each call is a fresh
HTTP/file fetch, and `svg.Rasterize` re-runs the (expensive) 1024px raster to
read only the width. That is a hidden external dependency inside a *measure*
pass that is supposed to be pure geometry, plus a 5-tuple `src,data,jpeg,w,h`
hand-plumbed into every call site (tuple assignment is the tell that a type is
missing). Fixing here centralizes decode policy in one place, cuts network
fetches per document, and makes the measure/build passes agree by construction.
hypothesis: the N×-fetch counts are real; validate by add a log counter in a
test that lays out a document with floats/tables and images.

---

---

<a id="p4-02"></a>
## P4-02 — Collapse content-box/used-size math duplicated ~10× across formatting contexts

- [ ] **P4-02** — Collapse content-box/used-size math duplicated ~10× across formatting contexts

- **Locations:** `layout.go:448-513` (buildBlock hand-rolled width+content box), `layout.go:2023-2028` (emitCell), `layout.go:2503-2505` (layoutCell), `flex.go:24-29`, `grid.go:30-35` + `resolveUsedWidth` (grid.go:433-452) + `resolveContentHeight` (grid.go:458-475) + `resolveUsedHeight` (layout.go:555-579), `multicol.go:46-51`, `container.go:66-101` (unscaled 4th copy)
- **Evidence sources:** area-5 F2

---

### Evidence — F2

- **Severity:** medium
- **Location:** `layout.go:448-513` (buildBlock hand-rolled width+content box), `layout.go:2023-2028` (emitCell), `layout.go:2503-2505` (layoutCell), `flex.go:24-29`, `grid.go:30-35` + `resolveUsedWidth` (grid.go:433-452) + `resolveContentHeight` (grid.go:458-475) + `resolveUsedHeight` (layout.go:555-579), `multicol.go:46-51`, `container.go:66-101` (unscaled 4th copy)
- **Current (verbatim):**
```go
	contentW := b.w - e.scalePt(st.PaddingLeft) - e.scalePt(st.PaddingRight) -
		e.scalePt(st.BorderLeft.Width) - e.scalePt(st.BorderRight.Width)
	if contentW < 0 {
		contentW = 0
	}
	contentX := b.x + e.scalePt(st.BorderLeft.Width) + e.scalePt(st.PaddingLeft)
	contentStart := len(e.ops)
	cy := e.scalePt(st.PaddingTop) + e.scalePt(st.BorderTop.Width)
```
(flex.go:24-31; the same 4-term chrome expression appears in layout.go:508-512,
2023-2027, 2503-2505, grid.go:30-35, multicol.go:46-51 and, unscaled, in
container.go:97.)
- **Future:**
```go
// contentBox returns the content-box origin and width for a border box.
// Single home for the rule "content = border-box − scaled padding − scaled
// border" that every formatting context currently re-implements.
func (e *engine) contentBox(x, w float64, st ResolvedStyle) (cx, cw float64) {
    cw = w - e.scalePt(st.PaddingLeft) - e.scalePt(st.PaddingRight) -
        e.scalePt(st.BorderLeft.Width) - e.scalePt(st.BorderRight.Width)
    if cw < 0 {
        cw = 0
    }
    return x + e.scalePt(st.BorderLeft.Width) + e.scalePt(st.PaddingLeft), cw
}

// usedBorderWidth resolves width:* / width:% / auto vs. availW with
// box-sizing, min/max and margin:auto — the block, flex, grid, multicol and
// @container paths all call this instead of re-deriving it.
func usedBorderWidth(st ResolvedStyle, availW float64, e *engine) (w, x float64) { … }

// intrinsicInlineSize is the shrink-to-fit width for auto-sized floats,
// inline-blocks, and grid tracks (size-containment short-circuits included).
func (e *engine) intrinsicInlineSize(n *html.Node, st ResolvedStyle) float64 { … }
```
`buildBlock`, `buildFlex`, `buildGrid`, `buildMulticol`, `emitCell`, `layoutCell`
switch to `contentBox`; `container.go`'s `contentInlineSize` becomes a call
through `usedBorderWidth` (its unscaled copy is already a latent divergence:
it never applies `e.scalePt` with `opts.Zoom`).
- **Why:** "border box = content + padding + border" is the single most basic
layout invariant, yet it is written out ~10 times in 6 files with subtle
variants (three `resolveUsed*` helpers with different box-sizing rules, one
unscaled copy). Any padding/border-zoom change must be made in all of them or
they drift — and they already have (container.go ignores zoom). Concentrating
it gives one place to fix box-sizing bugs, and the intrinsic-size function has
seven callers today (`placeFloat`, `inlineBlockAvail`, `flex.go:274/299`,
`grid.go:945`, `measureCellMinMax`).
hypothesis: the zoom divergence (container.go vs the rest) is a real bug and
not a deliberate difference; validate by a golden render with `opts.Zoom != 1`
and an `inline-size` container (FIXME in container.go renders at unscaled
widths while the layout engine renders scaled ones).

---

<a id="p4-03"></a>
## P4-03 — Extract table column-sizing out of buildTable into a pure function

- [ ] **P4-03** — Extract table column-sizing out of buildTable into a pure function

- **Locations:** `internal/layout/layout.go:1288-1803` (`buildTable`, ~515 lines), column-sizing slab at `layout.go:1472-1633`
- **Evidence sources:** area-5 F3

---

### Evidence — F3

- **Severity:** medium
- **Location:** `internal/layout/layout.go:1288-1803` (`buildTable`, ~515 lines), column-sizing slab at `layout.go:1472-1633`
- **Current (verbatim):**
```go
	// table width
	// border-collapse: collapse suppresses the separate-border gap so colspan
	// header rows and body cells share edges instead of looking double-lined.
	spacing := e.scalePt(st.BorderSpacing)
	if st.BorderCollapse == "collapse" {
		spacing = 0
	}
	sumMax := 0.0
	sumMin := 0.0
	for i := range colW {
		sumMax += colW[i]
		sumMin += colMin[i]
	}
	chrome := spacing*float64(nCols+1) +
		e.scalePt(st.BorderLeft.Width) + e.scalePt(st.BorderRight.Width) +
		e.scalePt(st.PaddingLeft) + e.scalePt(st.PaddingRight)
	sumMax += chrome
	sumMin += chrome
	tableW := availW
	definiteTable := false
	if st.WidthPercent >= 0 {
		// width:% of the containing block (parent cell / block), not viewport
		tableW = availW * st.WidthPercent / 100
		definiteTable = true
	} else if st.Width >= 0 {
		tableW = e.scalePt(st.Width)
		…
```
(…the whole algorithm continues through the pct/abs hints, min-content floor
and definite/auto scaling to `tb.w = tableW` at line 1634.)
- **Future:**
```go
// tableColumnEnv is everything the column-sizing pass needs; no DOM, no ops.
type tableColumnEnv struct {
    colMin  []float64 // min-content per column (content only)
    colW    []float64 // max-content per column (content only)
    colPct  []float64 // width:% hints, -1 = auto
    colAbs  []float64 // width:pt hints, -1 = auto
    chrome  float64   // spacing + border + padding
    availW  float64
}

// sizeTableColumns resolves used column widths and the table border-box width
// (CSS2.1-lite auto/fixed: sum, % and abs hints, min floors, definite scaling).
func sizeTableColumns(env tableColumnEnv) (colW []float64, tableW float64) { … }
```
`buildTable` keeps structure collection (rows/cells/rowspan placement),
measures cells, then calls `sizeTableColumns`; emission stays where it is.
No exported behavior change; `sizeTableColumns` becomes directly unit-testable
without a DOM, fonts, or op emission.
- **Why:** `buildTable` interleaves five phases (flatten, rowspan grid, measure,
column sizing, row-growth/emit) and the sizing slab alone is ~160 lines of
nested `if/elseif` on definite/auto tables. The algorithm has a textbook small
interface (content widths in → used widths out) and a deep implementation — that
is exactly the shape worth carving out. It also de-scopes the giant function so
remaining table work (rowspan growth, collapsed-grid emission) becomes
reviewable, and column-math regressions no longer require golden-layout tests.
hypothesis: `sizeTableColumns` is a stable, self-contained algorithm; validate
by moving it verbatim and checking the fixture suite (table_ref_stack,
table_rowspan, table_col_pct) stays green with zero op changes.

---

<a id="p4-04"></a>
## P4-04 — Give cells a stored start row; stop re-scanning row list 4×

- [ ] **P4-04** — Give cells a stored start row; stop re-scanning row list 4×

- **Locations:** `internal/layout/layout.go:1681-1699` and `1724-1735` (buildTable scans), `layout.go:1913-1966` (`rowspanCovers`, `colspanCovers`, `cellStartRow`)
- **Evidence sources:** area-5 F4

---

### Evidence — F4

- **Severity:** medium
- **Location:** `internal/layout/layout.go:1681-1699` and `1724-1735` (buildTable scans), `layout.go:1913-1966` (`rowspanCovers`, `colspanCovers`, `cellStartRow`)
- **Current (verbatim):**
```go
func cellStartRow(tb *box, cell *box) int {
	for ri, cells := range tb.rows {
		for _, c := range cells {
			if c == cell {
				return ri
			}
		}
	}
	return -1
}
```
…and in `buildTable` the same scan is inlined a second time:
```go
	for _, cell := range tb.children {
		if cell.rowSpan <= 1 {
			continue
		}
		start := -1
		for ri, cells := range cellData {
			for _, c := range cells {
				if c == cell {
					start = ri
					break
				}
			}
			...
```
- **Future:**
```go
// Store the owned row on the cell at placement time instead of searching for it.
// (box already carries col, span, rowSpan — one more int closes the identity.)
type box struct {
    …
    col, span    int
    row, rowSpan int // row = owning table row index, set once at placement
}
// buildTable sets cell.row when it places cells:
//   for _, p := range placed {
//       cell := e.buildCell(p.node, p.col, p.cSpan)
//       cell.row, cell.rowSpan = p.row, p.rSpan
```
then `cellStartRow(tb, cell)` is deleted and `rowspanCovers`, `colspanCovers`
and the two buildTable loops use `cell.row`. `tb.rows` stays as the
per-row op/emit structure — nothing else changes.
- **Why:** `cell` already remembers `col`, `span`, `rowSpan`. The row identity
is recomputed by scanning every cell of every row each time it is needed:
in `rowspanCovers`/`colspanCovers` that happens per (row, column) pair of every
collapsed-grid row (O(rows·cols·cells), visible the moment a page has several large tables), and twice more in `buildTable`. Four copy-pastes of the same
"which row owns this cell" query across the file, with different loop
structures each time — a locality violation: cell identity is knowledge owned
by the placement pass, not by the scanners.
hypothesis: worst-case is O(rows·cols·cells) on collapsed grids; validate by
grepping the three scan sites (already listed) and profiling a 500-row table
fixture before/after.)

---

<a id="p4-05"></a>
## P4-05 — One text-wrap policy for inline layout and intrinsic measurement

- [ ] **P4-05** — One text-wrap policy for inline layout and intrinsic measurement

- **Locations:** `internal/layout/layout.go:2189-2379` (`measureCellMinMax`, `unbreakableMinWidth`, `maxSoftSegmentWidth`) vs. `internal/layout/inline.go:201-451` (`breakOverflowItem`, `splitTextToWidth`, `isSoftWrapRune`); plus `flex.go:274,299`, `grid.go:945`, `inline.go:958`, `layout.go:1041` all call the "cell" measurer.
- **Evidence sources:** area-5 F5

---

### Evidence — F5

- **Severity:** medium
- **Location:** `internal/layout/layout.go:2189-2379` (`measureCellMinMax`, `unbreakableMinWidth`, `maxSoftSegmentWidth`) vs. `internal/layout/inline.go:201-451` (`breakOverflowItem`, `splitTextToWidth`, `isSoftWrapRune`); plus `flex.go:274,299`, `grid.go:945`, `inline.go:958`, `layout.go:1041` all call the "cell" measurer.
- **Current (verbatim):** the cell measurer re-implements whitespace collapsing and token breaking:
```go
			if !nowrap {
				// Collapse runs of whitespace to a single space for measure,
				// matching normal white-space:normal inline layout.
				fields := strings.Fields(text)
				if len(fields) == 0 {
					return
				}
				lineOnlyNowrap = false
				lineHasInk = true
				// Leading space if original had leading WS and line already started.
				if lineW > 0 && len(text) > 0 && isHTMLSpace(text[0]) {
					lineW += e.measureTextFace(" ", cs)
				}
				for i, word := range fields {
					if i > 0 {
						lineW += e.measureTextFace(" ", cs)
					}
					w := e.measureTextFace(word, cs)
					uw := e.unbreakableMinWidth(word, cs)
					if uw > longestWord {
						longestWord = uw
					}
					lineW += w
				}
```
while inline.go has its own copy of the same word-break/overflow-wrap policy:
```go
	breakAll := it.style.WordBreak == "break-all" || it.style.OverflowWrap == "anywhere"
	breakWord := it.style.OverflowWrap == "break-word"
	tokenExceedsLine := adv > fullLineW+0.01
	// Mid-line: a normal / break-word token that fits a full next line must
	// wrap whole — not mid-break into a tight remainW (captions: "International").
	if !breakAll && !tokenExceedsLine {
		return nil
	}
```
Plus a third partial copy: `unbreakableMinWidth` / `maxSoftSegmentWidth`
(layout.go:2309-2359) decide min-content by re-deriving break-all /
overflow-wrap:break-word with another `isSoftWrapRune` walk.
- **Future:**
```go
// wordBreakPolicy is the single table for "how may a token split?" —
// white-space, word-break and overflow-wrap combine into one enum.
type wordBreakPolicy int

const (
    breakNormal  wordBreakPolicy = iota
    breakAll                     // word-break:break-all / overflow-wrap:anywhere
    breakWord                    // overflow-wrap:break-word (soft only)
    breakNever                   // white-space:nowrap|pre
)

func wordBreakOf(st ResolvedStyle) wordBreakPolicy { … }

// breakToken splits s into chunks that each fit their max width under the
// policy — the SAME routine inline line-packing and table-cell min-content
// measurement call.
func (e *engine) breakToken(s string, st ResolvedStyle, firstMax, restMax float64) []string { … }

// minContentWidth = widest unbreakable chunk of s under the policy
// (replaces unbreakableMinWidth + maxSoftSegmentWidth).
func (e *engine) minContentWidth(s string, st ResolvedStyle) float64 { … }
```
Then `measureCellMinMax` computes words+min via `minContentWidth` and drops its
own `lineOnlyNowrap`/folding; breakOverflow keeps calling `breakToken` for
actual splitting. Rename `measureCellContent` → `intrinsicContentWidth` and move
it out of the tables section — it is already the intrinsic measure for floats,
flex, grid tracks and inline-blocks. All unexported; no API break. Callers to
touch: `flex.go:274,299`, `grid.go:945`, `inline.go:958`, `layout.go:1041`.
- **Why:** table-cell intrinsic sizing and inline line-breaking encode the same
  wrapping policy twice and they drift (there are already policy-shaped
  heuristics unique to one side: the `em*10` nowrap cap at layout.go:2207, the
  `em*8` nowrap-cluster glue at inline.go:125). Every soft-break char, break-all
  tweak, or whitespace rule change must be made twice — and divergence is
  invisible except as wrong table widths vs. wrong paragraph wrapping on the
  same fixture. A single tokenizer (shared min-content *and* chunker, plus a
  shared soft-break table through `isSoftWrapRune`) guarantees the two layout
  paths agree by construction, and moves the "how long is this text when it
  wraps" knowledge out of the two callers to make it defensible in one file.
hypothesis: the two copies already diverge on edge cases (nowrap caps at
layout.go:2207 vs inline.go:125, and the soft-break rune set); validate by
writing a table cell and a paragraph with the same token under
`overflow-wrap:break-word` and comparing min-content vs rendered line breaks.

---

<a id="p4-06"></a>
## P4-06 — Thread the inline-formatting width into inline-block measurement

- [ ] **P4-06** — Thread the inline-formatting width into inline-block measurement

- **Locations:** `internal/layout/inline.go:930-966` (`inlineBlockAvail`), `inline.go:969` (`availWForInline`)
- **Evidence sources:** area-5 F6

---

### Evidence — F6

- **Severity:** low
- **Location:** `internal/layout/inline.go:930-966` (`inlineBlockAvail`), `inline.go:969` (`availWForInline`)
- **Current (verbatim):**
```go
func (e *engine) inlineBlockAvail(n *html.Node, st ResolvedStyle) float64 {
	if st.WidthPercent >= 0 {
		// Percent of viewport is a best-effort stand-in; real containing
		// block width is not threaded into collectInline.
		if e.opts.Width > 0 {
			return e.opts.Width * st.WidthPercent / 100
		}
	}
	...
// availWForInline is a generous width for block-in-inline measurement.
func availWForInline() float64 { return 1 << 30 }
```
- **Future:**
```go
// layoutInlineFloats already knows the formatting-context width (contentW);
// thread it through collection so inline-blocks resolve % against the real CB.
func (e *engine) layoutInlineFloats(b *box, nodes []*html.Node, contentW, contentX, y float64, floats *floatState) float64 {
	...
	e.collectInline(nodes, &items, contentW)
	...
}

func (e *engine) inlineBlockAvail(n *html.Node, st ResolvedStyle, cbW float64) float64 {
	if st.WidthPercent >= 0 {
		return cbW * st.WidthPercent / 100
	}
	...
}
```
`collectInlineNode` passes `cbW` down to `inlineBlockAvail` (or a scoped
`e.inlineCBW` mirroring the `e.imgMaxW` pattern). The `1<<30` block-in-inline
measurement stays a documented approximation unless a second layout pass at the
packed line width is added later.
- **Why:** the comment at inline.go:932 admits the mechanism: the width of an
element whose containing-block width is needed is known at `layoutInlineFloats` (and per-line
`floats.exclusion`) but not handed to the collector, so the collector reaches
into `e.opts.Width` — a global stand-in that is wrong inside a float, a table
cell, or a multicol column, and wrong for `screen` (viewport ≠ canvas). This is
a seam that leaks the containing-block fact through an unrelated global, the
kind of "invariant known at one call site, re-derived at another" the locality
rule targets. It is low severity because it only affects width-percentage inline-blocks in nested contexts; it is a
cheap fix.
hypothesis: a 50% inline-block inside a 30%-width float is the wrong width
today; validate by a fixture that measures the block's rendered width.

---

<a id="p4-07"></a>
## P4-07 — Stop splicing the op list per box — defer background/border chrome

- [ ] **P4-07** — Stop splicing the op list per box — defer background/border chrome

- **Locations:** `internal/layout/layout.go:676-700` (`prependChrome`; called from layout.go:548, flex.go:67, grid.go:415, multicol.go:78,142)
- **Evidence sources:** area-5 F7

---

### Evidence — F7

- **Severity:** low
- **Location:** `internal/layout/layout.go:676-700` (`prependChrome`; called from layout.go:548, flex.go:67, grid.go:415, multicol.go:78,142)
- **Current (verbatim):**
```go
	// insert chrome before content ops
	tail := append([]Op(nil), e.ops[insertAt:]...)
	e.ops = e.ops[:insertAt]
	e.ops = append(e.ops, chrome...)
	e.ops = append(e.ops, tail...)
}
```
- **Future:**
```go
// chromeEntry records one box's background/border ops for insertion before
// its content op at the recorded index. Calls are recorded during the build;
// finalizeChrome merges them into the op list in ONE linear pass.
type chromeEntry struct {
    at  int
    ops []Op
}
func (e *engine) prependChrome(insertAt int, st ResolvedStyle, x, y, w, h float64) {
    ...
    e.deferredChrome = append(e.deferredChrome, chromeEntry{at: insertAt, ops: chrome})
}

// finalizeChrome builds the final ops slice (chrome first, then content) and
// reindexes box op-ranges by the count of chrome inserted before each range.
func finalizeChrome(ops []Op, boxes []*box, chrome []chromeEntry) []Op {
    out := make([]Op, 0, len(ops)+len(chrome))
    ci := 0
    for i, op := range ops { // chrome.at is non-decreasing by construction
        for ci < len(chrome) && chrome[ci].at <= i {
            out = append(out, chrome[ci].ops...)
            ci++
        }
        out = append(out, op)
    }
    for ci < len(chrome) { // trailing chrome (box with no content ops)
        out = append(out, chrome[ci].ops...)
        ci++
    }
    adjustOpRanges(boxes, chrome) // opStart/opEnd += count(chrome.at <= range start)
    return out
}
```
`Layout` calls `finalizeChrome(e.ops, …)` before building `Result`. Box op
ranges (`opStart/opEnd`), `Op.StickyID` and Paint's range logic all keep the
same semantics; the build-time index bookkeeping becomes append-only (a net
simplification: `shiftBoxOps`/`markOpsFixed` no longer move other boxes' ops).
- **Why:** every box with a background or borders currently copies the whole
  remaining op tail (`append([]Op(nil), e.ops[insertAt:]...)`) to insert 2-5
  ops at the front. Ops accumulate N ops over a run, so a document with B
  chromed boxes performs O(B·N) element copies; on D pump-style reports (thousands
  of colorful rows) this is the dominant op-list cost, quadratic by the time
  paint's later passes re-walk. More importantly it forces every later stage
  (sticky id mapping, fixed marking, pagination) onto an index space that is
  being mutated while the build is still running — the comment at
  layout.go:117 (`StickyID ... after parent prependChrome shifts op indices`)
  shows the coupling. Deferring the merge makes the op list append-only during
  the build and moves the interleaving into one linear pass.
  hypothesis: order-of-magnitude gain on chromed-heavy fixtures; validate by
  timing Layout before/after on a 1000-row table fixture with cell backgrounds.

---

---
