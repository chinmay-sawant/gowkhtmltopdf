# Phase 5 — Output, fonts, raster, shared paint

> **Parent:** [`architecture-review-2026-08-07.md`](../architecture-review-2026-08-07.md) — canonical architecture-review ledger
> **Status:** pending (findings gathered 2026-08-07 by 7 explore agents; remediation not started)
> **Depends on:** see phase map in ledger
> **Evidence archive:** raw agent findings were archived off-repo on 2026-08-07; every row below carries its Before/After snippets inline

---

## Overview

…

## Checklist

- [ ] **P5-01** — One paint-semantics table — convert body, HF and image adapters must not drift
- [ ] **P5-02** — Extract the one page-assembly pipeline; stop maintaining PDF/image forks
- [ ] **P5-03** — `SetGrayscale`: implement the promised paint-time seam or delete the setter
- [ ] **P5-04** — Give PDF object references a real type; centralize dict assembly
- [ ] **P5-05** — Move derived font data (font-face, reverse cmap) onto *Font; bound the atlas
- [ ] **P5-06** — Collapse the two font-embed pipelines (simple + type0) into one with a mode switch
- [ ] **P5-07** — Stop swallowing the font-embed error (unembedded runes render invisible)

---

<a id="p5-01"></a>
## P5-01 — One paint-semantics table — convert body, HF and image adapters must not drift

- [ ] **P5-01** — One paint-semantics table — convert body, HF and image adapters must not drift

- **Locations:** `internal/convert/hf.go:359-460` (`paintLayoutOps`); the sibling dispatch lives in `internal/layout/paint.go:120-182` (`Paint`'s `paintOp` + `sortPaintIndices`) with draw helpers at 1550-1640, `internal/layout/paint.go:1602-1620` (PDF fake bold) vs `internal/imageout/imageout.go:322-327` (raster fake bold); same split for fill alpha (`layout/paint.go drawFill` pre-composites against white vs `imageout paint` raw `Op.Alpha`) and stroke width (`strokeWidthScale` vs PDF `SetLineWidth`)
- **Evidence sources:** area-6 F2; area-7 F7

---

### Evidence A — F2

- **Severity:** high
- **Location:** `internal/convert/hf.go:359-460` (`paintLayoutOps`); the sibling dispatch lives in `internal/layout/paint.go:120-182` (`Paint`'s `paintOp` + `sortPaintIndices`) with draw helpers at 1550-1640
- **Current (verbatim):**
```go
func paintLayoutOps(page *pdf.Page, c *pdf.Content, ops []layout.Op, originX, yTop float64, links hfLinkContext) {
	fontNames := map[*pdf.Font]string{}
	nextFont := 0
	resName := func(f *pdf.Font) string {
		if f == nil {
			return "F0"
		}
		if n, ok := fontNames[f]; ok {
			return n
		}
		n := "HF" + strconv.Itoa(nextFont)
		nextFont++
		fontNames[f] = n
		c.UseEmbeddedFont(n, f)
		return n
	}
	nextImg := 0
	for i := range ops {
		op := &ops[i]
		x := originX + op.X
		switch op.Kind {
		case layout.OpFillRect:
			y := yTop - (op.Y + op.H)
			if op.Alpha < 1 {
				c.SetOpacity(op.Alpha)
			}
			c.SetFillColor(op.R, op.G, op.B)
			c.Rect(x, y, op.W, op.H)
			c.Fill()
		case layout.OpStrokeRect:
			y := yTop - (op.Y + op.H)
			c.SetStrokeColor(op.R, op.G, op.B)
			c.SetLineWidth(1)
			c.Rect(x, y, op.W, op.H)
			c.Stroke()
		case layout.OpLine:
```
- **Future:** the op→PDF dispatch belongs to the package that owns `Op`. `layout` exports a band painter built on the same per-op code body `Paint` uses; `paintLayoutOps` keeps only the band clip, origin and link-annotation logic:
```go
// layout/paint.go — exported band painter, the shared dispatch.
type BandOptions struct {
	OriginX, OriginY float64 // canvas origin on the page (y-down canvas)
	Clip             [4]float64 // page rect [x, y, w, h]; zero = no clip
}

// PaintBand paints ops onto an existing page's content stream at a canvas
// origin, clipped to the band. Same dispatch as Paint (colors, opacity,
// transforms, embedded fonts, fake-bold policy); pagination, z-sorting and
// fixed stamps are skipped. Link ops are left to the caller (annotations
// need document-level context).
func PaintBand(p *pdf.Page, c *pdf.Content, ops []Op, opts BandOptions) {
	// hoisted from paintOp: fill/stroke/line/text/image, font+image naming
}
```
```go
// convert/hf.go — drawHTMLHF keeps only what is HF-specific.
func paintLayoutOps(page *pdf.Page, c *pdf.Content, ops []layout.Op, originX, yTop float64, links hfLinkContext) {
	c.Save()
	c.Rect(0, bandTop, page.Width(), bandH)
	c.Clip()
	layout.PaintBand(page, c, ops, layout.BandOptions{OriginX: originX, OriginY: yTop})
	c.Restore()
	for i := range ops { /* OpLinkURI annotation branch only */ }
}
```
No exported convert behaviour changes (`paintLayoutOps` is unexported); `layout.Paint` callers unchanged.
- **Why:** body and HTML-header paint each carry the full op-semantics table, and they have already drifted. (a) Body fake-bold skips CJK (`layout/paint.go:1602-1612`: "Stroking CJK/Type0 outlines creates horizontal streak artifacts") while the HF copy strokes every rune (`hf.go:410`). (b) Body honors `Op.Xform`/`PaintOpacity`/z-order (`layout/paint.go:121-135`) while HF paints raw op order with only `Alpha`. (c) font/image naming conventions and line-width defaults differ. Anyone changing op paint semantics edits two packages and re-verifies goldens twice; the CJK guard is evidence a one-sided fix already shipped. This is the "one adapter is not yet a real seam" case: the op dispatch is the seam, but convert does not cross it. hypothesis: bold CJK text in `--header-html` renders with stroke artifacts while the same text in the body does not — validate with a golden fixture containing bold CJK inside a header.

---

### Evidence B — F7

- **Severity:** medium
- **Location:** `internal/layout/paint.go:1602-1620` (PDF fake bold) vs `internal/imageout/imageout.go:322-327` (raster fake bold); same split for fill alpha (`layout/paint.go drawFill` pre-composites against white vs `imageout paint` raw `Op.Alpha`) and stroke width (`strokeWidthScale` vs PDF `SetLineWidth`)
- **Current (verbatim):**

```go
	// Fake bold only for Latin when CSS wants bold but the face is not bold.
	// Stroking CJK/Type0 outlines creates horizontal streak artifacts.
	fakeBold := op.Bold && (op.Font == nil || !op.Font.Bold())
	if fakeBold {
		for _, r := range op.Text {
			if r > 0xFF {
				fakeBold = false
				break
			}
		}
	}
	if fakeBold {
		c.SetLineWidth(op.Size * 0.06)
		c.TextRenderMode(2) // fill + stroke
	}
```

```go
		ttfDrawString(img, bx, by, op.Text, op.Size, face, c, pxPerPt)
		// ponytail: fake-bold double-draw when CSS weight wants bold but the
		// face is regular; upgrade when synthetic bold outlines land in pdf.
		if op.Bold && !face.Bold() {
			ttfDrawString(img, bx+float64(rasterSS), by, op.Text, op.Size, face, c, pxPerPt)
		}
```

- **Future:**

```go
// In the layout package — the seam both adapters already consume — one
// per-op appearance resolution that both painters call:
type PaintStyle struct {
	Fill        [3]float64 // final RGB, alpha already composited against white
	StrokeWidth float64
	FakeBold    bool // Latin-only gate lives here (CJK stroking streaks)
}

func StyleOf(op *Op) PaintStyle {
	// single home for: translucent-fill pre-composition, stroke min-width,
	// the fake-bold gate, image scaling policy.
}

// layout/paint.go and imageout/paint then only map PaintStyle → a vector
// operator or a pixel fill; policy lives once.
```

- **Why:** `layout.Op` is already the seam — both adapters consume the identical display list — but every paint-semantics decision is re-implemented per adapter and the copies have already diverged: PDF fake-bold has a Latin-only gate ("stroking CJK/Type0 outlines creates horizontal streak artifacts"), the raster version double-draws CJK too at a fixed `rasterSS`-pixel offset; PDF `drawFill` pre-composites translucent fills against white paper while raster feeds raw `Op.Alpha` into `draw.Over`, so the same `rgba(…)` fill renders differently in PDF vs PNG. Who pays: any fidelity bug is found twice (golden PDF and image suites) and fixed twice; the divergence is why fixture output must be validated per format. `hypothesis: rendering the same fixture (e.g. fixture-14 rgba band, bold CJK) as PDF and PNG currently differs in alpha/bold appearance` — validate by rendering one fixture to both formats and comparing the PDF operators against the raster pixels. Putting `PaintStyle` next to `layout.Op` puts the invariants next to their data (locality item 3).

---

Not raised as separate findings (adjacent, small): `Font.FamilyNames()` mutates `f.PostScriptName` as a getter side-effect (`fonts.go:388-460`) — make it explicit (`LoadNames(f) (names []string, psName string)`); `nfcNormalize` (`shape.go`) is a misnomer — it only drops leading combining marks, full NFC is blocked by the no-`x/text` policy, so rename to `stripOrphanMark`; `jpegSize`/`jpegColorSpace` (`images.go`) are the same marker-walk loop twice — merge into one `jpegScan(data) (w, h, components, err)`; `pdf.go` `Write` records xref offsets for objects whose dict was never set (offset of the *next* object — latent corruption if a caller forgets `setDict`). Each folds naturally into F3/F5 rather than deserving its own finding.

> **Fan-in note:** Merged from area-6 F2 + area-7 F7: the `layout.Op` seam exists, but every adapter re-implements paint semantics; body-vs-HF and PDF-vs-image copies have already diverged (fake-bold CJK gate, alpha, stroke).

---

<a id="p5-02"></a>
## P5-02 — Extract the one page-assembly pipeline; stop maintaining PDF/image forks

- [ ] **P5-02** — Extract the one page-assembly pipeline; stop maintaining PDF/image forks

- **Locations:** `internal/imageout/imageout.go:420-477` vs `internal/convert/convert.go:50-54,301-336`; same-named forked helpers at the bottom of both files (`collectSheets`/`styleText`/`linkStylesheet`/`mediaFor` — `styleText` is byte-for-byte identical)
- **Evidence sources:** area-7 F1

---

### Evidence — F1

- **Severity:** high
- **Location:** `internal/imageout/imageout.go:420-477` vs `internal/convert/convert.go:50-54,301-336`; same-named forked helpers at the bottom of both files (`collectSheets`/`styleText`/`linkStylesheet`/`mediaFor` — `styleText` is byte-for-byte identical)
- **Current (verbatim):** the imageout prologue — `internal/imageout/imageout.go:420-477`

```go
	loader := load.NewLoader(cmd.Image.Load)
	loader.Log = log
	loader.Allow = cmd.Global.Allow
	loader.EnableLocalFileAccess = cmd.Global.EnableLocalFileAccess

	font, err := pdf.DefaultFont()
	if err != nil {
		return fmt.Errorf("default font: %w", err)
	}
	var registry *pdf.Registry
	if dirs := cmd.Global.FontPaths; len(dirs) > 0 || cmd.Global.UseSystemFonts {
		scan := append([]string{}, dirs...)
		if cmd.Global.UseSystemFonts {
			scan = append(scan, pdf.DefaultSystemFontDirs()...)
		}
		registry = pdf.ScanFontDirs(scan)
		if log != io.Discard && len(scan) > 0 {
			fmt.Fprintf(log, "info: scanned %d font path(s)\n", len(scan))
		}
	}

	res, err := loader.Load(ctx, obj.Page, obj.Load)
	if err != nil {
		return fmt.Errorf("load %q: %w", obj.Page, err)
	}
	if res.Skip {
		return fmt.Errorf("load %q: load-error policy is skip; nothing to render", obj.Page)
	}

	root, err := html.Parse(string(res.Body))
	if err != nil {
		return fmt.Errorf("parse html: %w", err)
	}

	sheets := collectSheets(ctx, loader, root, res.Base, obj.Load, log)
	enabled := convert.SimplifyDOMEnabled(cmd.Image.Web, obj.Web) || cmd.Global.Web.SimplifyDOM
	profile := convert.SimplifyDOMProfile(cmd.Image.Web, obj.Web)
	if profile == "" {
		profile = convert.SimplifyDOMProfile(cmd.Global.Web, settings.Web{})
	}
	sheets = convert.AppendSimplifySheet(sheets, enabled, profile)
	registry = convert.MergeFontFaces(ctx, loader, registry, sheets, res.Base, obj.Load, 1, log)

	cache := map[string][]byte{}
	imagesFn := func(src string) ([]byte, error) {
		if !cmd.Image.Web.Images {
			return nil, fmt.Errorf("images disabled")
		}
		if b, ok := cache[src]; ok {
			return b, nil
		}
		r, err := loader.FetchSub(ctx, res.Base, src, obj.Load)
		if err != nil {
			return nil, err
		}
		cache[src] = r.Body
		return r.Body, nil
	}
```

…and the PDF-mode equivalent, `internal/convert/convert.go:301-326` (plus `font, err := pdf.DefaultFont(); registry := loadFontRegistry(cmd, log)` at 50-54):

```go
	res, err := loader.Load(ctx, obj.Page, obj.Load)
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): load: %w", idx+1, obj.Page, err)
	}
	if res.Skip {
		fmt.Fprintf(log, "warning: object %d (%s): load error policy is skip, omitting\n", idx+1, obj.Page)
		return nil, nil
	}

	root, err := html.Parse(string(res.Body))
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): parse html: %w", idx+1, obj.Page, err)
	}

	pageW, pageH, err := pageGeometry(cmd.Global)
	if err != nil {
		return nil, fmt.Errorf("object %d (%s): %w", idx+1, obj.Page, err)
	}
	contentW := pageW - cmd.Global.Margin.Left*mmToPt - cmd.Global.Margin.Right*mmToPt
	contentH := pageH - cmd.Global.Margin.Top*mmToPt - cmd.Global.Margin.Bottom*mmToPt
	media := mediaFor(cmd.Global, obj)
	sheets := collectSheets(ctx, loader, root, res.Base, obj.Load, idx+1, log, contentW, contentH, media)
	sheets = AppendSimplifySheet(sheets, SimplifyDOMEnabled(cmd.Global.Web, obj.Web), SimplifyDOMProfile(cmd.Global.Web, obj.Web))
	registry = MergeFontFaces(ctx, loader, registry, sheets, res.Base, obj.Load, idx+1, log)

	imagesFn := func(src string) ([]byte, error) {
		if !cmd.Global.Web.Images {
			return nil, fmt.Errorf("images disabled")
		}
		r, err := loader.FetchSub(ctx, res.Base, src, obj.Load)
		if err != nil {
			return nil, err
		}
		return r.Body, nil
	}
```

- **Future:** one internal package (e.g. `internal/pipe`, stdlib-only) that owns loader config, `DefaultFont` + font-path scanning, `Load`+`html.Parse`, sheet collection, simplify-sheet append, `MergeFontFaces`, and the images fetcher closure; `convert` and `imageout` become thin callers:

```go
// PageContext is everything a paint path needs after the prologue: the
// parsed DOM, merged sheets/registry, fetcher and resolved media.
type PageContext struct {
	Root     *html.Node
	Base     string
	Loader   *load.Loader
	Font     *pdf.Font
	Registry *pdf.Registry
	Sheets   []*css.Stylesheet
	Media    string
	Images   func(src string) ([]byte, error)
}

// Assemble runs the load → parse → sheets → fonts → fetcher prologue once.
func Assemble(ctx context.Context, loader *load.Loader, log io.Writer,
	font *pdf.Font, registry *pdf.Registry, page string, lp settings.LoadPage,
	media string, vw, vh float64) (*PageContext, error)

// styleText/linkStylesheet/mediaFor move here; media + viewport become
// parameters, removing convert's softRuleWarn fork and imageout's
// hard-coded 768×576 screen viewport. convert and imageout keep only
// their output-specific tails (paint pages vs rasterize).
```

- **Why:** the classic locality violation — the same knowledge (loader ACL bits, font scanning, sheet gathering, simplify, `@font-face` merge, image fetcher policy) lives in two callers and has already forked on real behaviour: convert's `collectSheets` emits the `large stylesheet volume` warning and evaluates link `media=` against the true page geometry; image's fork silently dropped both and hard-codes a 768×576 screen viewport; `mediaFor` forks print-default vs screen-default; convert's imagesFn has no cache while imageout's does. Every future loader/sheet policy change must be made twice and the drift is invisible to golden tests. `hypothesis: a loading-policy change has already shipped in only one path` — validate by diffing the two files' history (`git log -p -- internal/convert/convert.go internal/imageout/imageout.go`).

> **Fan-in note:** Depends on P2-01 and P2-14 — the forked prologue includes stylesheet harvesting.

---

<a id="p5-03"></a>
## P5-03 — `SetGrayscale`: implement the promised paint-time seam or delete the setter

- [ ] **P5-03** — `SetGrayscale`: implement the promised paint-time seam or delete the setter

- **Locations:** `internal/pdf/pdf.go:34,57-61`; sole writer `internal/convert/convert.go:159-160`
- **Evidence sources:** area-7 F2

---

### Evidence — F2

- **Severity:** medium
- **Location:** `internal/pdf/pdf.go:34,57-61`; sole writer `internal/convert/convert.go:159-160`
- **Current (verbatim):** `internal/pdf/pdf.go:57-61`

```go
// SetGrayscale paints grayscale content (converted at paint time by layout).
func (d *Document) SetGrayscale(on bool) { d.grayscale = on }

// Grayscale reports whether grayscale mode is on.
func (d *Document) Grayscale() bool { return d.grayscale }
```

and the only writer, `internal/convert/convert.go:159-160`: `doc.SetGrayscale(cmd.Global.Grayscale)`. Nothing reads the flag except the getter — `grep -rn grayscale internal/layout internal/imageout internal/pdf` shows no conversion anywhere (the only other hits are `images.go` JPEG colorspace handling and this struct field).

- **Future:** implement where the comment already promises, at paint time — the colors live in `Content`, not `Document`:

```go
// SetFillColor sets the fill color (RGB, 0..1); grayscale is applied at this
// paint-time seam, which is what Document.SetGrayscale promises today.
func (c *Content) SetFillColor(r, g, b float64) {
	if c.doc != nil && c.doc.grayscale {
		v := 0.299*r + 0.587*g + 0.114*b // Rec.601 luma
		r, g, b = v, v, v
	}
	c.buf.WriteString(fmt.Sprintf("%s %s %s rg\n", num(r), num(g), num(b)))
}

// same fold in SetStrokeColor; JPEG/PNG XObjects desaturate at embed time in
// images.go. No API change — the flag starts having the effect its doc
// comment already claims. (Image mode can reuse the same conversion when
// --grayscale is added there; today the flag is ModePDF-only.)
```

- **Why:** hunt item *hypothetical seam*: the option is threaded through the whole settings surface (`settings` reflect table, `cli --grayscale`, `api.go`) into `Document`, where it is stored and never consumed — so `--grayscale`/`colormode=grayscale` produces a full-color PDF. The seam is real but misplaced: color flows through `Content` at paint time, so the conversion belongs there (or as a `ColorConverter` hook `Document` exposes to `Content`). Who pays today: every user who asks for grayscale and every maintainer who reads the false doc comment. If grayscale is genuinely out of scope, delete `SetGrayscale`/`Grayscale` and the colormode mapping instead — the current half-wire is the worst option. `hypothesis: --grayscale and --no-grayscale emit byte-identical PDFs` — validate with a golden render of both flag values.

---

<a id="p5-04"></a>
## P5-04 — Give PDF object references a real type; centralize dict assembly

- [ ] **P5-04** — Give PDF object references a real type; centralize dict assembly

- **Locations:** `internal/pdf/pdf.go:69-96` plus hand-stringed dict builders in `fonttype0.go:53-81,109-190`, `fontpdf.go`, `images.go:50-90`, `pdf.go:300+`
- **Evidence sources:** area-7 F3

---

### Evidence — F3

- **Severity:** medium
- **Location:** `internal/pdf/pdf.go:69-96` plus hand-stringed dict builders in `fonttype0.go:53-81,109-190`, `fontpdf.go`, `images.go:50-90`, `pdf.go:300+`
- **Current (verbatim):** `internal/pdf/pdf.go:69-96`

```go
// newObject allocates an indirect object and returns its reference string.
func (d *Document) newObject() string {
	d.nextID++
	id := d.nextID
	d.objects = append(d.objects, &object{id: id})
	return strconv.Itoa(id) + " 0 R"
}

// setDict replaces the object's dict body.
func (d *Document) setDict(ref string, dict string) {
	id := refID(ref)
	d.objects[id-1].dict = dict
}

// setStream attaches a raw stream (compressed later at write time).
func (d *Document) setStream(ref string, raw []byte) {
	id := refID(ref)
	d.objects[id-1].stream = raw
}

func refID(ref string) int {
	end := strings.IndexByte(ref, ' ')
	n, err := strconv.Atoi(ref[:end])
	if err != nil {
		panic("bad ref " + ref)
	}
	return n
}
```

- **Future:**

```go
// objRef is a typed indirect-object handle; the "N 0 R" spelling is a
// formatting concern, not a data type, and refs cannot be malformed.
type objRef int

func (r objRef) String() string { return strconv.Itoa(int(r)) + " 0 R" }

func (d *Document) newObject() objRef // append &object{id: int(r)}
func (d *Document) setDict(r objRef, s string)
func (d *Document) setStream(r objRef, raw []byte)

// dict is a tiny ordered builder so PDF syntax (escaping, /Name tokens,
// pdfString folding) lives in one place instead of ~20 fmt.Sprintf sites.
type dict []string
func (d dict) add(k string, v ...string) dict { return append(d, append([]string{k}, v...)...) }
func (d dict) String() string { return "<< " + strings.Join(d, " ") + " >>" }
```

Exported surface keeps strings where callers already consume them (`PageRef`/`Outline.PageRef`/`AddLinkURI`/`AddLinkDest` return and take strings, so `internal/convert` — emitOutline, links, hf — does not move); `finalizeOutlines` validates `PageRef` by parsing it as an `objRef` and returns an error instead of emitting a bogus `/Dest`. Internal refs (fontCache values, annotation refs) become `objRef`.

- **Why:** the *seam leaks implementation types* item: the atomic unit of a PDF (an indirect object) is a scalar in `Document.objects` but is threaded as the string `"N 0 R"` through `newObject`→`setDict`/`setStream`, with a parse-back (`refID`) that **panics** on any misuse and no validation when a caller hand-builds a ref (e.g. `Outline.PageRef`). PDF dict grammar is additionally re-stringed by hand at every call site — the knowledge that should sit inside the writer is spread across `pdf.go`/`fonttype0.go`/`images.go`. A `dict` builder also gives the fold+escape invariant (`pdfString`/`winAnsiFold`, runes-vs-codes) one home instead of being reproduced per caller. `hypothesis: an out-of-range PageRef emits a corrupt /Dest with no error` — validate by feeding `PageRef(999999)` into `Outline.PageRef` and inspecting the serialized outline.

---

<a id="p5-05"></a>
## P5-05 — Move derived font data (font-face, reverse cmap) onto *Font; bound the atlas

- [ ] **P5-05** — Move derived font data (font-face, reverse cmap) onto *Font; bound the atlas

- **Locations:** `internal/pdf/shape_gotext.go:14-29,299-314`; `internal/imageout/ttfraster.go:42-57`
- **Evidence sources:** area-7 F4

---

### Evidence — F4

- **Severity:** medium
- **Location:** `internal/pdf/shape_gotext.go:14-29,299-314`; `internal/imageout/ttfraster.go:42-57`
- **Current (verbatim):** `internal/pdf/shape_gotext.go:14-15` plus the per-call rebuild at `shape_gotext.go:299-313`:

```go
// gotextFaceCache maps *Font → *gtfont.Face (or false-sentinel on failure).
var gotextFaceCache sync.Map
```

```go
func (f *Font) reverseCmap() map[uint16]rune {
	out := make(map[uint16]rune, len(f.cmap))
	for cp, gid := range f.cmap {
		if gid == 0 {
			continue
		}
		r := rune(cp)
		if prev, ok := out[gid]; ok {
			if cmapRuneScore(r) <= cmapRuneScore(prev) {
				continue
			}
		}
		out[gid] = r
	}
	return out
```

`tryShapeOpenType` calls `rev := f.reverseCmap()` on every shaped run (RTL/combining/CJK text, once per text op) even though `f.cmap` is immutable after parse — a full map allocation over `len(f.cmap)` entries per call (CJK faces ~30k entries). The other unbounded cache is the image-mode glyph atlas, `internal/imageout/ttfraster.go:54-57`: `var glyphCache = map[glyphKey]*glyphCacheEntry{}` guarded by a global mutex, keyed by `*pdf.Font` pointer — every re-parse of the same TTF (each conversion parses its own Liberation bytes) is a new key, so a long-lived library process grows without bound and tests share state with no reset hook.

- **Future:**

```go
// Derived, immutable caches live on the Font next to the data they derive
// from (locality) and disappear with it (bounds). gotextFaceCache sync.Map
// and the package-level glyphCache are deleted.
type Font struct {
	// …
	gotOnce sync.Once
	gotFace *gtfont.Face // parsed go-text face (nil on failure)
	revOnce sync.Once
	rev     map[uint16]rune
}

func (f *Font) reverseCmap() map[uint16]rune {
	f.revOnce.Do(func() { /* build f.rev from f.cmap once */ })
	return f.rev
}

// The raster glyph atlas becomes per-Render state owned by imageout.rasterize
// (created per Render, size-capped e.g. 4096 entries with eviction), so
// conversions cannot leak into each other and tests start clean.
```

- **Why:** testability trap + locality. The go-text face cache already established the pattern (heavy derived data keyed by `*Font`) but was parked at package level and only half applied: the reverse map — the *same kind* of derivation — is recomputed per call while its sibling is cached. Package-level `sync.Map`/global-mutex maps are hidden global state that makes every Render depend on shared process memory. Who pays: library embedders get an unbounded glyph atlas; every shaped CJK text run allocates and drops a full reverse map. `hypothesis: a long-lived process rendering many docs grows RSS without bound, and CJK pages spend measurable time in reverseCmap` — validate with a loop of `imageout.Render` calls + `runtime.ReadMemStats` (Alloc before/after) and a `go test -bench` on `ShapeTextFontWithFeatures` with a CJK face.

---

<a id="p5-06"></a>
## P5-06 — Collapse the two font-embed pipelines (simple + type0) into one with a mode switch

- [ ] **P5-06** — Collapse the two font-embed pipelines (simple + type0) into one with a mode switch

- **Locations:** `internal/pdf/fonttype0.go:35-85` and `:89-171`
- **Evidence sources:** area-7 F5

---

### Evidence — F5

- **Severity:** medium
- **Location:** `internal/pdf/fonttype0.go:35-85` and `:89-171`
- **Current (verbatim):** the two functions open identically; lines 58-73 of the first are duplicated in the second with a one-string difference. `internal/pdf/fonttype0.go:35-73`:

```go
func (d *Document) ensureFontSimple(f *Font, used []rune) (string, error) {
	baseName := f.PostScriptName
	if baseName == "" {
		baseName = "LiberationSans"
	}
	key := "simple|" + baseName + "|" + runesKey(used)
	if ref, ok := d.fontCache[key]; ok {
		return ref, nil
	}
	sub, err := subsetFont(f, used, subsetSimple)
	if err != nil {
		return "", err
	}
	ef := &embeddedFont{subset: sub}
	ef.fontRef = d.newObject()
	ef.descRef = d.newObject()
	ef.ref = d.newObject()

	raw := flateBytes(sub.data)
	d.setDict(ef.ref, fmt.Sprintf("<< /Length %d /Filter /FlateDecode /Length1 %d >>",
		len(raw), len(sub.data)))
	d.setStream(ef.ref, raw)

	xMin, yMin, xMax, yMax := f.PDFBBox()
	flags := 32
	italicAngle := 0
	if f.Italic() {
		italicAngle = -12
		flags |= 64
	}
	if f.Bold() {
		flags |= 4
	}
	pdfName := pdfNameToken(baseName)
	d.setDict(ef.descRef, fmt.Sprintf(
		"<< /Type /FontDescriptor /FontName /%s /Flags %d /FontBBox [%d %d %d %d] /ItalicAngle %d /Ascent %d /Descent %d /CapHeight %d /StemV 80 /FontFile2 %s >>",
		pdfName, flags, xMin, yMin, xMax, yMax, italicAngle,
		f.PDFAscent(), f.PDFDescent(), f.PDFCapHeight(), ef.ref))
```

`ensureFontType0` repeats all of the above (same cache-key shape with `"type0|"`, same descriptor with `Identity` appended to the name, same FontFile2 stream) and then adds CIDFontType2/ToUnicode. The 1000-em width math is also written twice with different shapes: `subsetWidths` (`fontpdf.go:28-49`) for simple `/Widths` and an inline loop (`fonttype0.go:145-153`) for the Type0 `/W` array.

- **Future:**

```go
// One embed routine; the only variance is how runes map to char codes.
func (d *Document) ensureFont(f *Font, used []rune) (string, error) {
	type0 := needsType0(used)
	key := fmt.Sprintf("v%d|%s|%s", type0, f.PostScriptName, runesKey(used))
	if ref, ok := d.fontCache[key]; ok {
		return ref, nil
	}
	scope := subsetSimple
	if type0 {
		scope = subsetUnicode
	}
	sub, err := subsetFont(f, used, scope)
	if err != nil {
		return "", err
	}
	fileRef, descRef, err := d.embedFontFile(f, sub) // FontFile2 + FontDescriptor,
	if err != nil {                                  // shared by both modes
		return "", err
	}
	if type0 {
		return d.emitType0(f, sub, fileRef, descRef)
	}
	return d.emitSimple(f, sub, fileRef, descRef)
}

// widthsInEm(sub, upm) []float64 is the single home of the font-units→1000-em
// conversion, feeding both /Widths and the Type0 /W array.
```

- **Why:** hunt item 5 — repetitive struct/plumbing that one small type would collapse. The ~25-line FontDescriptor/FontFile2 constructor is copied with a one-string mutation, and the units→1000-em conversion is written twice with different shapes, so a width-scale or descriptor change must be found and fixed in two places. `hypothesis: a FontDescriptor fix (e.g. /StemH or flags) has already been applied once and missed once` — validate by `git log -p -- internal/pdf/fonttype0.go` and counting distinct edits touching the two copies.

---

<a id="p5-07"></a>
## P5-07 — Stop swallowing the font-embed error (unembedded runes render invisible)

- [ ] **P5-07** — Stop swallowing the font-embed error (unembedded runes render invisible)

- **Locations:** `internal/pdf/content.go:327-332`; discard sites `_ = c.AddJPEGImage(...)` / `_ = c.AddPNGImage(...)` at `internal/layout/paint.go:1636-1639`
- **Evidence sources:** area-7 F6

---

### Evidence — F6

- **Severity:** medium
- **Location:** `internal/pdf/content.go:327-332`; discard sites `_ = c.AddJPEGImage(...)` / `_ = c.AddPNGImage(...)` at `internal/layout/paint.go:1636-1639`
- **Current (verbatim):** `internal/pdf/content.go:327-332`

```go
		ref, err := c.doc.ensureFont(f, c.used[name])
		if err != nil {
			continue // skip broken font, layout should have caught it
		}
		c.fontUses[name] = ref
		out[name] = ref
```

- **Future:**

```go
// fonts() returns the embed error instead of dropping the resource; a page
// whose text has no embedded subset is a broken page, not a fast one.
func (c *Content) fonts() (map[string]string, error) {
	out := map[string]string{}
	for name := range c.fontUses {
		f, ok := c.fontFiles[name]
		if !ok {
			continue
		}
		ref, err := c.doc.ensureFont(f, c.used[name])
		if err != nil {
			return nil, fmt.Errorf("embed font %s: %w", name, err)
		}
		c.fontUses[name] = ref
		out[name] = ref
	}
	return out, nil
}

// finalizePage/finalize propagate that error into Document.Write's return
// value; drawImage returns the AddJPEGImage/AddPNGImage error instead of
// discarding it with `_ =`.
```

- **Why:** hunt item 4 (swallowed errors). The invariant "every resource name in the content stream must exist in /Resources" is enforced today by dropping the resource and continuing: a subsetting failure (e.g. a corrupted table in a user-supplied WOFF that the loader accepted) produces a PDF whose text operator names a font absent from /Resources — viewers render nothing (invisible text) with no diagnostic, and the library API (`api.go` Convert) has no channel to hear about it. `hypothesis: forcing ensureFont to error yields a page with text ops but no font resource and a clean exit code` — validate with a unit test that injects a failing subsetter and asserts `fonts()` returns an error once the plumbing lands.

---
