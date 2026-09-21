---
name: perf-patterns
description: Concrete, measured Go performance patterns proved in gowkhtmltopdf - before/after code, when each applies, detector greps, the warm/cold measurement protocol, and the traps already falsified in this repo. Use when optimizing hot paths, reviewing a perf change, or planning a perf pass. Companion to perf-review, which runs the parallel review-and-fix wave.
---

# Perf Patterns (proven in this repo)

Every pattern here was shipped in gowkhtmltopdf and is backed by a commit, a plan ledger row, or a review transcript. Nothing in this file is speculation dressed as advice. The catalog was mined on 2026-09-21 from HEAD `7496277` (VERSION 0.2.6), the commit range `c103141..84a5b68`, the `plans/0.2.6/` perf ledgers, and 30+ OpenCode agent sessions.

Two things this skill does not do: it does not replace `skills/perf-review/SKILLS.md` (that file runs the 5-agent review wave), and it does not license refactors. A pattern only counts when you apply it to a measured waste and prove the output did not change.

Line numbers below are accurate at HEAD `7496277`. They drift with every edit, so the symbol name is the contract and the line number is a convenience.

## How to use this skill

1. Find the waste. Run the detector greps in "Detector commands". If the profile does not show the waste, stop; guessing is how this repo shipped a 1.89x CPU regression once already.
2. Match it to a pattern. Micro-patterns (Part 1) are "instead of X, use Y". Structural patterns (Part 2) are design moves with measured payoffs.
3. Apply it with output-identity discipline. Page counts, ordered text, fonts, links, outlines, structure tags, image dimensions, and decoded pixels are contracts.
4. Measure. Warm and cold are different products, `B/op` is not RSS, and a phase closes only when the measurement lands and the gates pass.
5. Record the proof. A new pattern needs a commit, the waste class it removes, and a number (or the honest label "structural only").

## The rules that keep optimizations honest

- Warm and cold rows are never mixed and never averaged. Cold is the first conversion in a fresh process; it carries about 6.99 MB / 709 allocations of default-font work and about 4 ms of first-run time. Warm is a later row in the same process.
- `B/op` is cumulative allocation traffic during one operation, not resident memory. CLI RSS is `/usr/bin/time %M` in KiB. Do not convert one into the other.
- One sample is noise. Use the median of 3 fresh-process samples; take `B/op` and `allocs/op` from the median-time sample; never average `B/op`.
- The same code swings by more than 30 percent between adjacent processes on this WSL2 host. Paired same-binary A/B runs are the strongest evidence; single-run wall times are not.
- Equivalence is bit-level: box Y values bit-identical, fixture PDFs byte-identical after normalizing the creation date (`s/D:\d{14}Z/D:00000000000000Z/g` then `cmp`), 500-page `output-bytes` unchanged unless the ledger deliberately re-pins it.
- One agent owns one package. Phases that touch `internal/layout` paint paths run sequentially, and no two benchmarks run at once.
- A phase closes only when its measurement lands and `make test` plus `make lint` exit 0. "Golden 65/65, lint not run" is an open phase.

Gate order at closure:

1. `make test`
2. `make golden` plus a fresh uncached `go test -p 1 -parallel 2 -count=1 ./internal/convert/ -run 'TestGoldenCorpus' -v` (65/65 PASS)
3. `make claim-scan`
4. `make lint` (golangci-lint v1.64.8, plus `size-check` and frontend ESLint)
5. `make test-race` (convert, layout, pdf, imageout, load)
6. `make build`
7. `compliance/verify_pdfs.sh` (PDF/A-4 and PDF/UA-2, invoked directly)

Regression pins that gate layout and image work: style storage stays at 221,208 B per conversion, image 250/500 tile `B/op` stays at or below 21.5/27.2 MB, 500-page serial `output-bytes` is 1,419,234 (the independent-block path is deliberately re-pinned at 1,420,537), image geometry stays 1024x2056 / 1024x4040, and `unsafe.Sizeof(Op{})` is pinned at <= 256 B by `internal/layout/op_size_test.go`.

---

# Part 1: Micro-patterns - instead of X, use Y

## Quick reference

| Instead of | Use | Repo example |
|---|---|---|
| `fmt.Sprintf`/`num()` per painted operand | append into spare buffer capacity | `appendPDFNum` / `writePDFNums`, `internal/pdf/content.go:107` |
| `pdfString(s) + " Tj\n"` | `AvailableBuffer` + one `Write` | `TextShow`, `internal/pdf/content.go` |
| `prop + "-" + side` per shorthand | static longhand table | `boxShorthandLonghands`, `internal/layout/style_cascade.go` |
| `prev.text += cur.text` merge chains | one `strings.Builder` per group, compact in place | `coalesceTextItems`, `internal/layout/inline_collect.go` |
| `Sprintf` + `strings.Join` per row | Builder with explicit separators | `buildParentTree`, `internal/pdf/structure.go` |
| `cur.text += string(r)` per rune | record a byte start, slice the source | `splitTextByFace`, `internal/layout/inline_paint.go` |
| `[]rune(s)` for first/last rune | `utf8.DecodeRuneInString` / `DecodeLastRuneInString` | `internal/layout/inline_collect.go:989` |
| `[]byte(s)` round trips | walk the source bytes | zero-alloc scanners, `internal/html`, `internal/css` |
| two cmap lookups per rune | one helper, one division | `GlyphAdvancePoints`, `internal/pdf/fonts.go:823` |
| `strings.Fields` per cell measure | byte walk, measure the space once | cell measure, `internal/layout/layout.go` |
| `strings.ToLower` over the rest of the doc | scan for `<`, fold-compare the name | `rawTextEnd`, `internal/html/html.go:682` |
| `classSet` map per selector probe | byte token scan with a Unicode fallback | `hasClassToken`, `internal/css/match.go:638` |
| per-call `strings.NewReplacer` | package-level replacer or one-pass writer | `escapeXML`, `internal/layout/layout_svg.go:221` |
| `regexp.Compile` per dynamic key | keyed compiled-pattern cache or a switch | `internal/pdf/semantic.go:841` (still present) |
| append from nil when the count is known | count first, `make([]T, 0, n)` | `Locations`, `paint.go:1121`; tokens, `html.go:469` |
| append into `map[int][]T` buckets | count each key, allocate per-key capacity | seal collection, `internal/layout/paint.go:853` |
| copy 56-byte values into every bucket | store `int32` indexes into one slice | `borderSegIndex`, `paint_pagination_seal.go:318` |
| six parallel cascade maps + per-element sort | one winner record, fixed apply order | `cascadeRaw`, `internal/layout/style.go` |
| heap `map[T]bool` for a small set | stack array or a `switch` | `restLonghandStack`, `internal/layout/style_cascade.go:1268` |
| by-value comparator on a 264 B record | compare pointers; hoist the fixed sort | `paint_order.go:44`, `paint.go:522` |
| rescan the display list per page | bucket by page / rounded Y once | `stripOrphanRowChrome`, `internal/layout/paint_pagination_seal.go:615` |
| linear span scan per membership query | prefix max + binary search | `opInPaintRange`, `paint_pagination_fixpoint.go:488` |
| per-break prefix-max rebuild | running max + difference array, one pass | `beforeAlways`, `internal/layout/paint_flow_breaks.go` |
| 1.3 KB `ResolvedStyle` by value | one heap record per node, shared by pointer | `stylePtr`, `internal/layout/layout.go` |
| `reflect.DeepEqual` interning | reusable scratch + generated equality | `style_intern_gen.go`, `internal/layout/style.go:867` |
| full display-list copy in chrome merge | backward in-place splice from spare capacity | `mergeDeferredChrome`, `layout_chrome.go:751` |
| scratch allocation per call | thread a reusable buffer with a cap check | `resetIntBuffer` / `resetPageBuckets`, `internal/layout/paint_flow_index.go:204` |
| `img.At(x,y).RGBA()` per pixel | concrete `*image.NRGBA` `Pix` paths | `downscaleBox2`, `imageout.go:823` |
| fresh parse of the same embedded asset | share bytes, parse lazily behind `sync.Once` | `ensureParsed`, `internal/pdf/fonts.go:159` |

## Detailed examples

### B1. Instead of `fmt.Sprintf` + a string-returning `num()` per painted operand, append into the buffer's spare capacity

Repo: `appendPDFNum` at `internal/pdf/content.go:107`, `writePDFNums` at `:129` (commit `c103141`).

Before:

```go
func num(v float64) string {
	if v == float64(int(v)) {
		return strconv.Itoa(int(v))
	}
	s := strconv.FormatFloat(v, 'f', 3, 64)
	s = strings.TrimRight(s, "0")
	s = strings.TrimRight(s, ".")
	return s
}

c.buf.WriteString(fmt.Sprintf("%s %s %s rg\n", num(r), num(g), num(b)))
```

After:

```go
func appendPDFNum(dst []byte, val float64) []byte {
	if val == float64(int(val)) {
		return strconv.AppendInt(dst, int64(int(val)), pdfNumBase)
	}
	dst = strconv.AppendFloat(dst, val, 'f', pdfFloatPrec, float64Bits)
	// trim trailing zeros and a trailing dot
	return dst
}

func (c *Content) writePDFNums(suffix string, count int, num1, ... float64) {
	out := c.buf.AvailableBuffer()
	out = appendPDFNum(out, num1)
	// ... one branch per argument count ...
	out = append(out, suffix...)
	_, _ = c.buf.Write(out)
}
```

Why: the old shape reparsed the format, boxed a string per operand, and let each `num()` allocate. The new one formats into capacity the buffer already owns and commits the length once. `num()` still exists for odd call sites, but now formats through a stack array: `string(appendPDFNum(buf[:0], v))` at `internal/pdf/pdf.go:1667-1671`.

Risk: `appendPDFNum` implements only `'f'` output with zero trimming, so move call sites only where the format is pinned by tests.

Proof: 500-page in-process PDF fell from 14.14 s to 6.93 s in the wave that introduced it; the 0.2.6 profile still charges content formatting 6.27 percent at 500 pages (`plans/0.2.6/perf-time/profiles/time-cpu.md:334` estimates 30 to 50 ms more from a fixed-precision writer plus buffer reuse).

### B2. Instead of `pdfString(str) + " Tj\n"`, write into `AvailableBuffer` and commit once

Repo: `TextShow` / `textShowSimple`, `internal/pdf/content.go` (commit `5025b29`).

Before:

```go
for _, rVal := range str {
	// ... fold to Latin-1 ...
	c.used[c.curFont] = append(c.used[c.curFont], rVal)
}

c.buf.WriteString(pdfString(str) + " Tj\n")
```

After:

```go
used := c.used[c.curFont]
out := c.buf.AvailableBuffer()
out = append(out, '(')

for _, rVal := range str {
	// ... fold to Latin-1 ...
	used = append(used, rVal)
	out = appendPDFLiteralByte(out, byte(rVal))
}

out = append(out, ')', ' ', 'T', 'j', '\n')
_, _ = c.buf.Write(out)
c.used[c.curFont] = used
```

Why: one allocation-free operator instead of a string, a concatenation, and a map write per rune. The same wave removed the staging slice in `dict.add` (`internal/pdf/pdf.go:111`): `append(d, append([]string{k}, v...)...)` became `d = append(d, k)` plus a loop.

Risk: never alias the destination while reading it, and always write the possibly-grown `used` slice back.

### B3. Instead of `prev.text += cur.text` merge chains, build each merged run once and compact in place

Repo: `coalesceTextItems`, `internal/layout/inline_collect.go:1096` (commit `60c3c9b`).

Before:

```go
if mergeable {
	prev.text += cur.text
	prev.w += cur.w
	// ...
}
```

After:

```go
var builder strings.Builder
writeIdx := 0
merged := false

for i := range line {
	cur := line[i]
	mergeable := writeIdx > 0 && /* same predicate as before */ ...

	if mergeable {
		if !merged {
			merged = true
			builder.WriteString(line[writeIdx-1].text)
		}
		builder.WriteString(cur.text)
		prev := &line[writeIdx-1]
		prev.w += cur.w
		continue
	}
	// ... compact into writeIdx, return line[:writeIdx] ...
}
```

Why: each `+=` copied the whole accumulated prefix, so a k-item merge did O(k^2) byte copies and k allocations. One Builder per group makes it one allocation plus one copy, and the write-index pass reuses the backing array the caller already owns.

Risk: compaction must not read elements the write index has overwritten; return the prefix so stale tails are invisible; merge order must stay identical for byte-identical output.

Proof: commit message records "was O(k^2) concat"; wave result 500-page `B/op` 392.2 -> 335.8 MB, median 0.936 -> 0.878 s.

### B4. Instead of `cur.text += string(r)` per rune, record a byte start and slice the source

Repo: `splitTextByFace`, `internal/layout/inline_paint.go` (commit `c103141`).

Before:

```go
for _, r := range s {
	face := e.faceForRune(st, r)
	if cur.face == nil {
		cur = faceRun{face: face}
	} else if face != cur.face {
		runs = append(runs, cur)
		cur = faceRun{face: face}
	}
	cur.text += string(r)
	cur.w += face.AdvanceInPoints(r, size)
}
```

After:

```go
start := 0
var current *pdf.Font
for i, r := range s {
	face := e.faceForRune(st, r)
	if current != nil && face != current {
		runs = append(runs, faceRun{text: s[start:i], face: current, w: width})
		start = i
		width = 0
	}
	// ...
	current = face
	width += face.AdvanceInPoints(r, size)
}
if current != nil {
	runs = append(runs, faceRun{text: s[start:], face: current, w: width})
}
```

Why: a 40-rune run produced 40 strings and up to 40 backing arrays. `range` gives byte offsets, which is exactly what slicing needs. Cold profile: `layout.splitTextByFace` was 2.84M objects, 25.89 percent of allocated objects; the document fell from 14,387,130 to 8,875,233 allocs/op in the compared snapshots.

Risk: boundaries must stay in byte offsets; a run pins its source string (fine when the source outlives the run).

### B5. Instead of two cmap lookups behind two wrappers, use one helper with the same fallback

Repo: `GlyphAdvancePoints`, `internal/pdf/fonts.go:823` (commit `60c3c9b`).

Before: `AdvanceInPoints` called `Advance` called `GlyphID`, so every measured rune paid the cmap lookup twice plus a call boundary.

After:

```go
func (f *Font) GlyphAdvancePoints(r rune, size float64) float64 {
	g := f.GlyphID(r)

	adv := float64(f.advance[0])
	if int(g) < len(f.advance) {
		adv = float64(f.advance[g])
	}

	return adv / float64(f.unitsPerEm) * size
}
```

Why: text measurement is the hottest path in the engine (50K table cells x 4 to 5 passes), so a duplicated map probe per rune multiplies. The out-of-range fallback is preserved exactly, which is what keeps measurement and paint in agreement.

Risk: keep `Advance` / `AdvanceInPoints` for non-hot callers, and never rebuild the chain in a new glyph path.

### B6. Instead of `strings.Fields` per cell measure, walk bytes and measure the separator once

Repo: cell measure, `internal/layout/layout_measure.go:245` (commit `f689d64`; the function moved out of `layout.go` when that file was split).

Before: `fields := strings.Fields(text)` allocated a `[]string` plus a string header per word on every measure call; the separator was measured again per word.

After:

```go
spaceW := eng.measureTextFace(" ", cstate)
// ...
for i < len(text) {
	for i < len(text) && isHTMLSpace(text[i]) { i++ }
	if i >= len(text) { break }
	j := i
	for j < len(text) && !isHTMLSpace(text[j]) { j++ }
	word := text[i:j]
	if !first { m.lineW += spaceW }
	first = false
	m.lineW += eng.measureTextFace(word, cstate)
	m.noteWord(eng.minContentWidth(word, cstate))
	i = j
}
```

Why: no `[]string`, no word copies, one space measurement. Table cells are measured repeatedly during column sizing, so this compounds.

Risk: whitespace semantics must match byte for byte (`isHTMLSpace` covers space, tab, newline, CR, FF; leading-space and non-ASCII rules preserved). The landing commit kept a parity test.

### B7. Instead of lowercasing the rest of the document, scan for `<` and fold-compare the name

Repo: `rawTextEnd`, `internal/html/html.go:682` (commit `60c3c9b`). Before: `strings.ToLower(src[from:])` per `<script>`/`<style>`, quadratic on many-block documents. After:

```go
for offset < srcLen {
	lt := strings.IndexByte(src[offset:], '<')
	if lt < 0 { return 0, 0, false }
	candidate := offset + lt
	if candidate+1 < srcLen && src[candidate+1] == '/' && rawNameFolds(src[candidate+2:], name) {
		// ...
	}
}
```

The same trick lands as `containsVarFunc` (`internal/layout/style.go:456`), which replaced `strings.Contains(strings.ToLower(val), "var(")` with a four-byte fold scan, and as `hasClassToken` (`internal/css/match.go:638`), which replaced a per-probe `map[string]bool` class set with a byte token scan and a Unicode fallback on the first byte >= 0x80.

Why: parsers and matchers run per element or per selector, so an O(n) lowercase pass inside them is an O(n^2) document scan.

Risk: the ASCII fast path must fall back to a Unicode-aware version, and trimming/whitespace rules must match the stdlib helper exactly. Write the parity test.

### B8. Instead of appending from nil, count first and allocate exact capacity

Repo: locations at `internal/layout/paint.go:1121`, tokens at `internal/html/html.go:469`, table cells at `internal/layout/layout_tables.go:427`, border segments at `internal/layout/paint_pagination_seal.go:385`.

Before: `res.Locations = append(res.Locations, ...)` from zero capacity (14.63 MB in one call per the pagination scan), `var toks []token` doubling and copying ~220K+ tokens.

After:

```go
res.Locations = make([]ElementLocation, 0, count)
toks := make([]token, 0, strings.Count(src, "<")+1)
```

And for per-key map buckets (commit `aa8d446`):

```go
startCounts := make(map[int]int)
for i := range verts {
	k0, k1 := roundY(verts[i].y0), roundY(verts[i].y1)
	startCounts[k0]++
	endCounts[k1]++
}
vertStarts := make(map[int][]vseg, len(startCounts))
for key, count := range startCounts {
	vertStarts[key] = make([]vseg, 0, count)
}
```

Why: growth leaves every earlier array as garbage. An exact count pass (or a cheap upper bound) replaces doubling plus copies. For buckets, the win is when keys are low-cardinality and each accumulates many values.

Risk: a counting pass costs more than growth on tiny slices. Measure before moving a preallocation constant: this repo raised `opsPerNodeHint` from 3 to 5, measured 174K real ops, and reverted because `nodes*3/2` already covered it while the higher hint over-reserved about 39 MB.

### B9. Instead of copying 56-byte values into per-key buckets, store `int32` indexes

Repo: `borderSegIndex`, `internal/layout/paint_pagination_seal.go:318` (commit `5483ec6`).

Before: `map[int][]vseg` where each bucket appended a copy of the 56-byte value.

After:

```go
// borderSegIndex maps a rounded Y bucket to positions in the segment slice.
type borderSegIndex map[int][]int32

vertStarts[k0] = append(vertStarts[k0], segIdx(i))
```

Why: the base slice is counted once (about 63,000 vertical segments per 500 pages), and buckets hold 4-byte positions instead of 56-byte copies. Seal family dropped 71.1 -> 14.95 MB at 500 pages and CPU share 5.02 -> 2.00 percent.

Risk: the base slice must stay immutable and unappended while the index is live.

### B10. Instead of six parallel cascade maps plus a per-element sort, keep one winner record

Repo: `cascadeRaw`, `internal/layout/style.go` (commits `4e5c498`, `f689d64`).

Before: `normal`, `important`, `nSpec`, `nOrder`, `iSpec`, `iOrder` (six maps, up to four writes per declaration), plus building a `[]string` and calling `sort.Strings` per element.

After:

```go
wins := make(map[string]cascadeWin, cascadeWinHint)
// one lookup, at most one assignment per declaration
```

and a static apply order:

```go
var restShorthandProps = [...]string{ //nolint:gochecknoglobals // static apply order
	"margin", "padding", "border", borderWidthKeyword, borderStyleKeyword,
	borderColorKeyword, gapKeyword, flexKeyword, containerKeyword,
}
```

Why: removes map churn, the per-element slice, and the sort. The companion fixes: `declaredInheritableMask` folds about 65 lookups per element into one pass over the raw keys, and `boxShorthandLonghands(prop)` replaces `prop + "-" + side` concatenation (6.4 MB per 500-page conversion in the profile).

Risk: winner semantics are load-bearing ("any `!important` beats any normal", inline specificity). The insertion order must reproduce the old `sort.Strings` byte order where longhands overlap (`overflow` and `overflow-x`). Keep a regression test.

### B11. Instead of rescanning the display list per page, bucket once by page and by rounded Y

Repo: `stripOrphanRowChrome`, `internal/layout/paint_pagination_seal.go:615` (commit `c103141`; later moved out of `paint.go`); the shared index lives at `internal/layout/paint_flow_index.go:490` (`buildPageIndex`, commit `5483ec6`).

Before:

```go
for p := 0; p <= maxPage; p++ {
	for i := range res.Ops {
		op := &res.Ops[i]
		if op.Fixed || op.Y < pageTop-1e-9 || op.Y >= pageBot-1e-9 { continue }
		// ...
	}
}
```

After:

```go
pageOps := make([][]int, maxPage+1)
for i := range res.Ops {
	if res.Ops[i].Fixed { continue }
	page := int(res.Ops[i].Y / contentH)
	pageOps[page] = append(pageOps[page], i)
}
for p := 0; p <= maxPage; p++ {
	for _, i := range pageOps[p] {
		op := &res.Ops[i]
		if op.Y < pageTop-1e-9 || op.Y >= pageBot-1e-9 { continue } // exact band kept
	}
}
```

Why: the old shape was O(pages x ops); `stripOrphanRowChrome` alone was 5.41 s flat, 29.73 percent of the cold 500-page profile, and is "absent from the top 30" after. The shared index replaced two names for one algorithm and keeps its arrays between rebuilds (reset in place, counted capacity per bucket).

Risk: bucket boundaries must match the scanner's band exactly or boundary ops are dropped. Mutating `Y` invalidates the buckets; `invalidateFlowIndex` plus `ensureFlowIndex` is the rebuild path.

### B12. Instead of a linear span scan per query, prefix-max plus binary search

Repo: `opInPaintRange`, `internal/layout/paint_pagination_fixpoint.go:488` (commit `ca761bb`).

Before: a loop over every `paintRange` per membership query.

After:

```go
slices.SortFunc(ranges, func(left, right paintRange) int {
	return cmp.Compare(left.first, right.first)
})
for idx := 1; idx < len(ranges); idx++ {
	if ranges[idx].last < ranges[idx-1].last {
		ranges[idx].last = ranges[idx-1].last
	}
}

func opInPaintRange(index int, ranges []paintRange) bool {
	// binary search rightmost span with first <= index, then test its prefix-max last
}
```

Why: sorting once and folding `last` into a running prefix maximum makes membership a monotone predicate. 87M span checks became 597k binary steps.

Risk: spans must not change between queries; on tiny sets the scan is cheaper.

### B13. Instead of a per-call scratch, thread a reusable buffer with a capacity check

Repo: `prefixMaxOfOps(ops, buf)` (commit `b494aac`; since folded into `breakScanState` in `internal/layout/paint_flow_breaks.go`), same shape at `resetIntBuffer` / `resetPageBuckets`, `internal/layout/paint_flow_index.go:204-217` (commit `5483ec6`).

Before:

```go
func prefixMaxOfOps(ops []Op) []float64 {
	prefixMaxY := make([]float64, len(ops)+1)
	// ...
}
```

After:

```go
func prefixMaxOfOps(ops []Op, buf []float64) []float64 {
	need := len(ops) + 1
	if cap(buf) < need {
		buf = make([]float64, need)
	} else {
		buf = buf[:need]
	}
	if need > 0 { buf[0] = 0 } // must reset a dirty buffer
	// ...
	return buf
}
```

Why: the function ran once per forced break (about 500 times) and `prefixMaxOfOps` was 16.9 percent flat CPU plus 677 MB alloc_space. Callers pass the previous result back.

Risk: the returned slice is valid until the next call; document it. Reset logically-constant slots (`buf[0] = 0`). Never return a pooled buffer at zero capacity: the `d0c75fe` `flateBytes` rewrite copied the compressed stream and returned an empty `bytes.Buffer`, forcing every future pooled instance to reallocate (review finding MEM-01); the current code retains capacity (`internal/pdf/pdf.go:1673`).

### B14. Instead of interface-boxed pixel access, use concrete `*image.NRGBA` byte paths

Repo: `downscaleBox2` at `internal/imageout/imageout.go:823`, `fillNRGBAOpaque` at `:1529`, `drawNRGBAOpaque` at `:1564` (commit `c103141`); concrete plane paths for JPEG at `internal/imageout/ycbcr.go` (commit `84a5b68`).

Before:

```go
c := src.NRGBAAt(x, y)
dst.SetNRGBA(x, y, color.NRGBA{R: uint8(r / n), ...})
```

After:

```go
srcTop := src.PixOffset(sb.Min.X, sb.Min.Y+y*2)
dstOffset := dst.PixOffset(0, y)
r := uint32(src.Pix[srcTop+left]) + uint32(src.Pix[srcTop+right]) + ...
dst.Pix[dstOffset] = uint8(r / 4)
```

Why: `NRGBAAt` recomputes `PixOffset` and round-trips a `color.NRGBA`; direct `Pix` arithmetic removes both. Opaque branches skip alpha compositing entirely (copy whole rows, or pack four pixels per 64-bit store).

Risk: guard every fast path by concrete type and keep the generic fallback. `Pix` access ignores a non-zero `Rect.Min` unless you go through `PixOffset`; premultiplied vs non-premultiplied alpha differs per type. Add pixel-equivalence tests (`TestScaleNearestNRGBAMatchesGeneric`, `TestNRGBAToYCbCrDoesNotBoxPixels`).

### B15. Instead of `reflect.DeepEqual` interning, generate equality and reuse one scratch (the cautionary tale)

The first `styleStore` implementation boxed two 1.33 KB structs into `any` on every hash hit and let `&tmp` escape. Result: 537,489,496 B/op count-3, up about 201 MB from the 335.8 MB baseline, with `internResolvedStyle` 32.59 percent flat. The fix: one reusable heap scratch plus an allocation-free equality method, landing at 259,243,992 B/op / 315,843 allocs/op.

The shipped design (`internal/layout/style.go:867`, `style_intern_gen.go`): a 64-bit fingerprint picks a bucket, then a generated field-complete equality decides; chunks of 64 keep pointers stable; the store never crosses `Layout` calls. Lesson: intern only when a cheap fingerprint exists and equality is exact over every field. If you add a field to `ResolvedStyle`, regenerate the hash and equality (`go generate` runs `scripts/gen-style-intern`), or equal styles silently stop sharing.

---

# Part 2: Structural patterns

Each entry is a design move with its measured payoff. "Warm 500p" means the warm 500-page generic internal PDF row.

1. **Intern repeated records at the store boundary.** `styleStore` (64-entry chunks, fingerprint + field-complete equality) turned 66K near-identical 1.3 KB styles into a handful of canonical records. Warm 500p `B/op` 557.5 -> 321.1 MB (-42 percent); style storage pins flat at 221,208 B from 2 to 500 pages. Fix aliasing writers first (`mergeCustomProps` clones before interning), and never cross `Layout` calls with the store.
2. **Memoize repeated resolution behind a provable key.** `resolveElementStyleMemo` (`internal/layout/style_memo.go`) keys parent, node name, inline style, rem base, link-underline policy, and the matched-rule sequence. Style phase 444 -> 30 ms; 99.9697 percent hit rate (65,985 hits per 500-page conversion). Removing it was a measured 1.89x CPU regression (500p 591 -> 1118 ms), restored in `8bd9eda`. Every input the resolver reads must be in the key.
3. **Share immutable parsed artifacts across conversions.** `parsedFontCache` keyed by path + size + mtime (`internal/pdf/font_file_cache.go`), stat-sized reads, and no caching when the read length does not match the stat. PDF corpus 2.262 GB -> 1.089 GB `B/op` (-52 percent), allocs 21.72M -> 1.19M (-94.5 percent). Do not cache mutable per-run containers; bound by bytes and entries.
4. **Parse embedded faces lazily with one owned copy.** `newLazyFont` + `ensureParsed` (`internal/pdf/fonts.go:159`, commit `5483ec6`): 14 eager parses become lazy handles; the double `bytes.Clone` is gone. Cold 2-page `B/op` 9.58 -> 2.67 MB, `LoadDefaultFaces` 5-8 ms -> 0.02 ms, CLI empty-document floor 18.5 -> 9.6 ms. Pin lazy-vs-eager equivalence (`TestLazyFaceMatchesEager`) and call the ensure step from every accessor.
5. **Paint at final resolution above a pixel-area threshold.** `directRasterPixels = maxPooledRasterBytes / (4 * rasterSS * rasterSS)` = 2,097,152 px (`internal/imageout/imageout.go:104`): above the threshold the final canvas is the paint target, no 2x supersample and no downscale copy. 250/500 tile `B/op` 56.0/94.97 -> 21.5/27.2 MB. Small canvases keep the 2x path; raising the cache cap was rejected because it pins RSS.
6. **Give large scratch buffers a byte-budgeted retention cache, not `sync.Pool`.** `pixBufferCache` (`internal/imageout/pixbuffer.go`) returns the smallest fitting buffer, caps per buffer at 32 MiB and total at 64 MiB, and bypasses oversized buffers. `sync.Pool` was emptied by harness GCs and could not express a budget; PNG `B/op` 3.379 -> 2.827 GB after the explicit cache.
7. **Reuse a workspace across sequential layouts.** Page-at-a-time with `Workspace.Release` (`internal/layout/layout.go:335`): layout, paint, release, next candidate; fail closed on blocks that are not provably independent. Warm 500p 733.48 -> 624.49 ms and 163.02 -> 121.02 MB. The output pin moved on purpose (1,419,234 -> 1,420,537); golden fixtures still 65/65 because they do not take the detector.
8. **Batch a logical group of primitives into one op.** `OpGridRun` (`internal/layout/grid_run.go`): a collapsed row's border grid becomes one run expanded at paint time. Display list 174,000 -> 66,500 entries, about 240 ms of warm 500p time and about 55 MB `B/op`. Every op reader must learn the new kind; batching broke `shiftBoxOps` once (fixed and regression-tested).
9. **Count first, allocate exact capacity, bucket indexes not values.** Seal family 71.1 -> 14.95 MB and 5.02 -> 2.00 percent CPU (see B8/B9).
10. **Batch ordered range updates behind a monotone precondition, with an exact fallback.** `beforeAlwaysBatch` (`internal/layout/paint_flow_breaks.go:592`): when boxes and targets both ascend by Y, one pass with a difference array replaces targets x elements. 27,196,997 box visits -> 0, placements bit-identical, `paginateOps` CPU 12.75 -> 8.04 percent. Unsorted input, negative deltas, and long tie runs fall back to the exact scan.
11. **Retain scratch index storage, reset in place, share one builder.** `buildPageIndex` + `flowIndexStorage` replaced `bucketOpsByPage` and `buildFlowOpIndex` (two names for one algorithm). Warm 500p `B/op` 321.10 -> 234.92 MB; `buildPageIndex` 18.67 MB vs `bucketOpsByPage` 28.28 MB. Release at the end (`releaseFlowIndex`) and never hand retained buckets to a live consumer then reset.
12. **Retain parallel compressor workers; window the work.** `finalizePagesWindowed` (`internal/pdf/flate_parallel.go:133`): up to 8 parked workers, each with one zlib writer, compressing windows of 8 pages; output order is input order. Finalize + compression 100.5 -> 22.1 ms, byte-identical output. A per-call pool was rejected because `sync.Pool` dropped about nine 663 KB writers between conversions (+6.0 MB `B/op`).
13. **Stream large raster output and reuse one row window.** Filter-None PNG writer at deflate level 2 plus strip painting through a 1 MiB reused buffer (`internal/imageout/tile_raster.go`, `pngfast.go`). 250/500 tile time 50.72/98.47 -> 24.15/43.43 ms; strip `B/op` 14.30/26.41 -> 6.38/10.59 MB. The documented trade: encoded PNG grows about 50 percent. All five alternative encoder settings were measured and rejected.
14. **Window only what is visible when the full artifact cannot be admitted by the cache.** `scaleNearestWindow` (`internal/imageout/scale.go`): guard on cache-admission math, not a fixed size. fixture-49 1395 ms / 1334 MB -> 151 ms / 43 MB; fixture-53 1486 / 1329 -> 151 / 38; PNG corpus 8023 ms / 3.65 GB -> 4917 ms / 0.94 GB. Parity tests prove window pixels match a crop of the full result.
15. **Hand planes to the encoder instead of per-pixel color boxing.** JPEG from full-resolution 4:4:4 YCbCr planes (`internal/imageout/ycbcr.go`, commit `84a5b68`): allocs 101,272,253 -> 526,832 (-99.48 percent), time 6935 -> 5248 ms. A 4:2:0 variant failed a 40x20 probe by one byte, so the parity suite now covers 13 shapes.
16. **Key run-scoped caches by an allocation-free hash.** `hashFontFamily` (`internal/layout/style.go:73`) fingerprints the font-family list without building a joined key; `faceFor` / `faceForRune` (`internal/layout/layout.go:693`, `:763`) give text a one-lookup primary face with a fallback; the per-render raster image cache keys on an FNV-1a hash of the encoded bytes confirmed with `bytes.Equal`. Caches live one run and are not concurrency-safe.
17. **Release what provably has no reader left.** Delete discarded intermediates (`assignFlowPages` removed, `paginateOps` returns only `error`), release pagination indexes after the last flow reader with an on-demand rebuild, and release each raw content buffer after its stream is materialized with a sticky finalize error so a retry cannot serialize an empty page (`internal/pdf/content.go:105`, `pdf.go:1126`).
18. **Guard expensive passes with a conservative census or presence flag.** `buildPaginationCensus` skips the `afterBreaks` walk when no box declares it; `contentNeedsEnv` (`internal/layout/pseudo_content.go:20`) needs three substring checks to skip building a counter/quote environment; `QuotesRaw` turns an ancestor re-cascade into field reads; `stickySectionChromeTargets` moves from `maxPage + 2` walks per pass to one. Guards must fail toward doing the work: a false "does not need" is a correctness bug. The sibling table guard was rejected because "the census costs its savings".

---

# Part 3: Detector commands

Run from the repo root. All read-only. `--glob '!*_test.go'` keeps test formatting out of the signal.

```bash
# 1. fmt.Sprintf sites, ranked by file.
rg -n 'fmt\.Sprintf\(' --glob '!*_test.go' internal/

# 2. Sprintf inside a loop (loop line as context; confirm by opening the file).
rg -n -B12 'fmt\.Sprintf\(' --glob '!*_test.go' internal/ | rg 'for .*(:=|range)'

# 3. Fprintf into in-memory buffers (per-object writers).
rg -n 'fmt\.Fprintf\(&' --glob '!*_test.go' internal/

# 4. ReplaceAll chains over the same string.
rg -n -U 'strings\.ReplaceAll\([^\n]*\n(?:[^\n]*\n){0,3}[^\n]*strings\.ReplaceAll\(' --glob '!*_test.go' internal/

# 5. NewReplacer built inside a function body (should be package level).
rg -n 'strings\.NewReplacer\(' --glob '!*_test.go' internal/

# 6. Compiled patterns outside a package var block.
rg -n -B4 'regexp\.MustCompile\(' --glob '!*_test.go' internal/
rg -n 'regexp\.Compile\(' --glob '!*_test.go' internal/

# 7. []byte(string) and string([]byte) conversions in hot packages.
rg -n '\[\]byte\(|string\(' --glob '!*_test.go' \
  internal/pdf internal/layout internal/imageout internal/css internal/html internal/convert

# 8. sync primitives on raster/paint paths (a Lock around per-glyph work is the smell).
rg -n 'sync\.(Pool|Mutex|RWMutex)' --glob '!*_test.go' internal/
rg -n '\.Lock\(\)|\.RLock\(\)' --glob '!*_test.go' internal/pdf internal/imageout internal/svg internal/layout
```

Landed from this list on 2026-09-21 (micro A/B of the old shape vs the shipped helper, output parity checked): xref entry formatting (`appendPadZero`), bfchar chunk lines (`appendBfcharSection`), trailer `/ID` hashing, link annotation dicts, ExtGState dicts, `/W` width arrays, the SVG XML escaper and `cssColorHex`, and per-lookup regexp compilation (`dynamicRE`). Micro deltas: bfchar 7,957 -> 2 allocs per 4,096 lines; `/W` rows 9,986 -> 3 per 2,048 rows; xref entries 4,116 -> 21 per 4,096. Gates at that pass: `make test`, golden 65/65, `make lint`, `make test-race`, PDF/A-4 + PDF/UA-2 compliance, `make claim-scan`, `make build`, all green.

Still open: one `ReplaceAll` pass per header/footer key at `internal/convert/hf.go:51-57` (the template scan is not hot, and a replacer rewrite changes overlapping-key semantics); `string(r)` per rune in vertical text at `internal/layout/inline_vertical_writing.go:46-49` (the measure and emit APIs take strings, so a static ASCII table would add a global for a cold path).

Honesty note (from `plans/0.2.6/perf-time/profiles/time-cpu.md:346-352`): the largest project flat function (`buildPageIndex`) is 2.83 percent at 500 pages, and deleting it entirely saves about 1.03x. Sell append-based formatting fixes on `B/op` and `allocs/op` first, wall time second.

False positives that cost people time in this repo: plain `fmt.Sprintf("%d", n)` is already a `perfsprint` lint error (the 46 surviving sites mix verbs and formats); `TrimSuffix`/`TrimPrefix` do not allocate when the affix is absent; `[]byte("literal")` in `bytes.HasPrefix` does not allocate; `internal/settings`, `internal/cli/help.go`, and `internal/imageout/rasterbudget.go` are startup or fatal-error paths; `internal/pdf/semantic.go` is validation-side, not the render path.

---

# Part 4: Traps - things already tried here

Rejected or reverted, with the reason. Do not revive one without new measured evidence that its blocker is gone.

- **Live page-index epoch** (keep buckets populated through invalidation): changed fixture-56 output. The shipped model keeps invalidation semantics.
- **`estimateOpCapacity` tightening**: the design ceiling was 13.47 MB but the corpus maximum is 15.16 ops/node; one growth at 500 pages requests about 120 MB. Rejected.
- **`opsPerNodeHint` 3 -> 5**: over-reserved about 97 MB against 58 MB that sufficed (real op count 174,000, hint-3 capacity 202.5K). Reverted.
- **Parallel layout (`ParallelLayout`)**: identity-safe but 5.4 percent paired gain against a 15 percent floor. Left unwired.
- **Naive per-call flate pool**: +6.0 MB `B/op` because `sync.Pool` dropped writers between conversions.
- **Deflate level 1**: saves about 32 ms but changes every stream and grows output 17.1 percent. **No compression**: 12 MB of raw streams must be copied; slower (1,822 ms vs 1,411-1,477 ms warm).
- **PNG alternatives**: `BestSpeed` still pays filter selection; `NoCompression` produces 67x larger files; level 3 costs about 6 ms more; parallel deflate cannot keep one zlib stream.
- **4:2:0 JPEG preconversion**: failed a 40x20 probe by one byte at partial-MCU edges. Full-resolution 4:4:4 planes shipped instead.
- **Chrome clone of repeated sections**: 500p time fell to about 217-312 ms, but `output-bytes` dropped to 1,335,615. The 1,419,234 pin is the kill switch; clone is off.
- **Per-section style store**: 500 x 219,648 B is about 109.8 MB. Resolve once for the whole document.
- **`reflect.DeepEqual` interning**: boxed two 1.33 KB structs per hit; +201 MB regression (see B15).
- **Table guard census**: "the census costs its savings". Deferred.
- **Sibling selector-index cache**: removed; its `:nth` microbenchmark regression (0.07 -> 0.73 us per match) was accepted because the product hot path does not use nth selectors.
- **`mergeDeferredChrome` forward copy**: allocated a second ~49 MB display list. The backward in-place splice replaced it.
- **`d0c75fe` flateBytes rewrite**: reintroduced a full compressed-stream copy and returned a 0-capacity buffer to the pool (MEM-01). Current code retains capacity.
- **Strip raster without windowed scaling**: rebuilt a 65.6 MiB poster canvas per 1 MiB strip on fixtures 49/53 (corpus `B/op` 206.7 -> 1334.1 MB). Fixed by `scaleNearestWindow`.
- **Smart-shrink relayout for sub-point overflow**: on = 1,000 `buildTable` calls / 2.51 s; off = 500 / 0.998 s. A 0.1pt tolerance removed the re-run (`TestSmartShrinkNoRelayoutWhenWithinTenthPoint`).
- **Unfinished business, stated as such**: wave-2 PDF 50 percent target missed (624.49 ms / 121 MB vs 366.74 ms / 81.51 MB); recovery missed the 0.2.4 PDF row (321.1 MB / 1,246 ms vs 237.76 MB / 1,010 ms); image 50 percent time missed (about 42 ms vs 22 ms); windowed flate CLI RSS gate not done.

---

# Part 5: Measurement protocol in brief

Harness: `scripts/bench-performance-recovery.sh` with closed modes `internal-pdf`, `internal-pdf-warm`, `public-pdf`, `public-image`, `cli-rss`, `external`. Warm matrix sizes: 2, 5, 10, 20, 50, 100, 200, 250, 500 in one process; row 1 is cold. Each capture header records date, mode, sizes, Go version, CPU, memory, OS, cache state, fixture hash, source hashes, and for CLI the binary path and hash.

- `internal-pdf`: `go test ./internal/convert -run '^$' -bench 'BenchmarkPDFPages/generic/(...)$' -benchmem -benchtime=$BENCHTIME -count=$COUNT`. Generic filter only, so internal-only rows can never leak into a product capture.
- `internal-pdf-warm`: the full matrix in one process (first row cold, rest warm).
- `public-pdf` / `public-image`: `go test .` over `BenchmarkLibraryPDF` / `BenchmarkLibraryImage`.
- `cli-rss`: build, render the exact report fixture, one warmup plus 3 timed `/usr/bin/time -f '%e %M'` runs, page-count check that fails the run on a mismatch.
- `external`: `scripts/bench-external.sh` (WeasyPrint, Puppeteer), 3 runs after one warmup.

Three fresh-process samples per row; report the median and take `B/op`/`allocs` from the median-time sample. `output-bytes` is reported after the timed loop (Go's `ResetTimer` deletes user metrics reported before it, which is why the old `pages` column was empty for months).

pprof recipe (profiles are diagnostics, not benchmarks):

```sh
profile_root=/tmp/gowkhtmltopdf-pprof
GOCACHE=/tmp/gowkhtmltopdf-go-cache go test -c -o "$profile_root/document.test" .
GOCACHE=/tmp/gowkhtmltopdf-go-cache "$profile_root/document.test" \
  -test.run '^$' -test.bench '^BenchmarkLibraryPDF/500Pages$' \
  -test.benchmem -test.benchtime=5s -test.count=1 \
  -test.cpuprofile="$profile_root/cpu.pprof"
GOCACHE=/tmp/gowkhtmltopdf-go-cache "$profile_root/document.test" \
  -test.run '^$' -test.bench '^BenchmarkLibraryPDF/500Pages$' \
  -test.benchmem -test.benchtime=1x -test.count=1 \
  -test.memprofilerate=1 -test.memprofile="$profile_root/heap.pprof"
go tool pprof -top -nodecount=30 "$profile_root/document.test" "$profile_root/cpu.pprof"
go tool pprof -top -alloc_space -nodecount=30 "$profile_root/document.test" "$profile_root/heap.pprof"
```

`-test.run '^$'` is mandatory or the CLI comparison tests poison the profile. `-memprofilerate=1` is exact but inflates time (~4.9x on the PDF run) and allocates extra; use 65536 for lighter runs. `inuse_space` (retained) and `alloc_space` (traffic) answer different questions. Counters come from temporary probe files in a copy of the tree, never the committed tree.

Throughput: `ops/sec = 1e9 / ns_per_op`; `make bench-throughput` runs `BenchmarkLibraryPDFThroughput` over a 10 s window and reports `pdfs/sec`. The `-24` suffix in benchmark names is `GOMAXPROCS`, not parallelism; the body runs serially in one goroutine.

Current anchors (medians, quoted from `testdata/golden/benchmarks/benchmark-results.txt`):

| Snapshot | Date | Warm 500p in-process |
|---|---|---|
| M (perf-time closure) | 2026-09-11 | 733.48 ms / 163,021,712 B / 754,893 allocs |
| N (released 0.2.6) | 2026-09-13 | 539.33 ms / 111,433,640 B / 687,406 allocs |

CLI 2026-09-13 capture: 2 pages 13 ms / 19,584 KiB vs wkhtmltopdf 258 ms; 500 pages 573 ms / 80,448 KiB vs 1.718 s. Image 500 tiles 33.51 ms / 9.92 MB (the raw Snapshot N row says 10,397,400 B/op; both are committed, quote one and name it). These are labeled snapshots, not a live SLA.

## Running an optimization pass

Use `skills/perf-review/SKILLS.md` for the wave mechanics (5 read-only reviewers by area, synthesized fix briefs, file-scoped fix agents, one orchestrator-run gate). The areas: A css+html, B inline text, C layout core, D paint/display list, E pdf+convert/load.

Finding schema every reviewer uses: ID and `file:line`; severity HIGH/MED/LOW; hot-path evidence (called per document, per page, per element, per rune, and the loop nesting); the exact waste (re-alloc, re-scan, fmt cost, map churn, slice growth, avoidable copy, O(n^2) scan, hoisting invariant work, zero-work fast path); a named fix sketch; effort and output-change risk. Fix agents never compile; the orchestrator is the only one who runs the gate.

Rules that keep a wave honest: reviewers read only, fix agents write only, one agent per file, one final gate, no two benchmarks at once, and every claim labelled cold or warm. When a lever fails its equivalence probe, it is reverted and recorded, not shipped behind a softer claim.
