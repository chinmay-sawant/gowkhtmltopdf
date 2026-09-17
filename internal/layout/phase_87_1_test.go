//nolint:cyclop,funlen,paralleltest // phase 87.1 property proofs
package layout

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

func TestGridGapAliasesMatchGap(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.gap { display:grid; grid-template-columns:1fr 1fr; width:200pt; gap:10pt }
.grid-gap { display:grid; grid-template-columns:1fr 1fr; width:200pt; grid-gap:10pt }
.row { display:grid; grid-template-columns:1fr; width:100pt; grid-row-gap:9pt }
.col { display:grid; grid-template-columns:1fr 1fr; width:200pt; grid-column-gap:15pt }
`)
	root := mustParse(t, `<html><body>
<div class="gap"></div><div class="grid-gap"></div>
<div class="row"></div><div class="col"></div>
</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", testViewport, 800)

	gap := styleByClass(t, styles, "gap")
	alias := styleByClass(t, styles, "grid-gap")
	row := styleByClass(t, styles, "row")
	col := styleByClass(t, styles, "col")

	if !near(gap.RowGap, alias.RowGap) || !near(gap.ColumnGap, alias.ColumnGap) {
		t.Fatalf("grid-gap alias mismatch: gap=(%.1f,%.1f) alias=(%.1f,%.1f)",
			gap.RowGap, gap.ColumnGap, alias.RowGap, alias.ColumnGap)
	}

	if !near(gap.RowGap, 10) {
		t.Fatalf("gap RowGap=%.3f, want 10", gap.RowGap)
	}

	if !near(row.RowGap, 9) {
		t.Fatalf("grid-row-gap = %.3f, want 9", row.RowGap)
	}

	if !near(col.ColumnGap, 15) {
		t.Fatalf("grid-column-gap = %.3f, want 15", col.ColumnGap)
	}
}

func TestGridAutoColumns(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.g { display:grid; grid-template-columns:60pt; grid-auto-columns:40pt; width:120pt; gap:0 }
.b { grid-column: 2; background:#0a0; height:10pt }
.a { background:#a00; height:10pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="g"><div class="a">A</div><div class="b">B</div></div>
</body></html>`, cssSheet)

	var aW, bW float64

	for _, op := range res.Ops {
		if op.Kind != OpFillRect {
			continue
		}

		if op.R > 0.5 && op.G < 0.2 {
			aW = op.W
		}

		if op.G > 0.5 && op.R < 0.2 {
			bW = op.W
		}
	}

	if aW < 55 || aW > 65 {
		t.Fatalf("explicit col width=%.1f, want ~60", aW)
	}

	if bW < 35 || bW > 45 {
		t.Fatalf("grid-auto-columns width=%.1f, want ~40", bW)
	}
}

func TestGridAutoRows(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.g { display:grid; grid-template-columns:50pt; grid-auto-rows:30pt; width:50pt; gap:0 }
.cell { background:#00a; }
`)
	res := layoutHTML(t, `<html><body>
<div class="g"><div class="cell">A</div><div class="cell">B</div></div>
</body></html>`, cssSheet)

	var heights []float64

	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.B > 0.5 && op.R < 0.2 {
			heights = append(heights, op.H)
		}
	}

	if len(heights) < 2 {
		t.Fatalf("want 2 cell fills, got %d", len(heights))
	}

	for i, h := range heights[:2] {
		if h < 28 || h > 32 {
			t.Fatalf("row %d height=%.1f, want ~30 from grid-auto-rows", i, h)
		}
	}
}

func TestOverflowBlockInlineMapToAxes(t *testing.T) {
	t.Parallel()

	t.Run("horizontal", func(t *testing.T) {
		t.Parallel()

		cssSheet := sheet(t, `
.block { overflow-block: hidden; width: 80pt; height: 50pt; max-height: 50pt }
.inline { overflow-inline: hidden; width: 80pt; height: 50pt }
.child { height: 200pt; width: 80pt; background: #cc0000 }
`)
		root := mustParse(t, `<html><body>
<div class="block"></div><div class="inline"></div>
</body></html>`)
		styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", testViewport, 800)
		block := styleByClass(t, styles, "block")
		inline := styleByClass(t, styles, "inline")

		if block.OverflowY != overflowHidden || block.OverflowX != visibleKeyword {
			t.Fatalf("overflow-block horizontal: x=%q y=%q", block.OverflowX, block.OverflowY)
		}

		if inline.OverflowX != overflowHidden || inline.OverflowY != visibleKeyword {
			t.Fatalf("overflow-inline horizontal: x=%q y=%q", inline.OverflowX, inline.OverflowY)
		}

		res := layoutHTML(t, `<html><body>
<div class="block"><div class="child">x</div></div>
</body></html>`, cssSheet)
		parent := findBoxByClass(t, res, "block")
		pad := paddingBoxOfTest(parent)

		for _, fill := range res.Ops {
			if fill.Kind != OpFillRect || fill.R < 0.7 || fill.G > 0.2 {
				continue
			}

			if fillOutsideClip(fill, pad) {
				t.Fatalf("overflow-block child fill %+v outside pad %+v", fill, pad)
			}
		}
	})

	t.Run("verticalWritingMode", func(t *testing.T) {
		t.Parallel()

		cssSheet := sheet(t, `
.v { writing-mode: vertical-rl; overflow-block: scroll; overflow-inline: hidden }
`)
		root := mustParse(t, `<html><body><div class="v"></div></body></html>`)
		styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", testViewport, 800)
		sty := styleByClass(t, styles, "v")

		// vertical: block→X, inline→Y
		if sty.OverflowX != overflowScroll || sty.OverflowY != overflowHidden {
			t.Fatalf("vertical map: x=%q y=%q, want scroll/hidden", sty.OverflowX, sty.OverflowY)
		}
	})
}

func TestObjectFitCover(t *testing.T) {
	t.Parallel()

	data := phase871PNG(t, 40, 10)
	res := imgAdjustLayout(t, "width:40pt;height:40pt;object-fit:cover", data)
	img := imgAdjustSingleImage(t, res)

	// cover of 40x10 into 40x40 → height fills, width ~160
	if img.H < 38 || img.H > 42 {
		t.Fatalf("cover H=%.1f, want ~40", img.H)
	}

	if img.W < 140 || img.W > 180 {
		t.Fatalf("cover W=%.1f, want ~160 (wider than box)", img.W)
	}
}

func TestObjectPositionRightBottom(t *testing.T) {
	t.Parallel()

	data := phase871PNG(t, 10, 10)
	res := imgAdjustLayout(t,
		"width:40pt;height:40pt;object-fit:none;object-position:right bottom", data)
	img := imgAdjustSingleImage(t, res)

	// 10px → 7.5pt intrinsic; body margin is 6pt so the 40pt content box
	// ends at 46pt. right/bottom object-position pins the image there.
	if img.W < 6 || img.W > 9 || img.H < 6 || img.H > 9 {
		t.Fatalf("none size = %.1fx%.1f, want ~7.5", img.W, img.H)
	}

	right := img.X + img.W
	bottom := img.Y + img.H
	if right < 44 || right > 48 {
		t.Fatalf("right edge = %.1f, want ~46 (6pt margin + 40pt box)", right)
	}

	if bottom < 44 || bottom > 48 {
		t.Fatalf("bottom edge = %.1f, want ~46", bottom)
	}
}

func TestCounterSetBeforeIncrement(t *testing.T) {
	t.Parallel()

	cmap := newCounterMap()
	cmap.applyReset("section 0")
	cmap.applySet("section 10")
	cmap.applyIncrement("section")

	if cmap.value("section") != 11 {
		t.Fatalf("reset→set→increment want 11 got %d", cmap.value("section"))
	}

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
.box { counter-set: item 5; counter-increment: item; }
.box::before { content: counter(item) " "; }
`)
	res := layoutHTML(t, `<html><body>
<div class="box">A</div>
</body></html>`, cssSheet)

	got := joinedPaintText(res)
	if got != "6 A" {
		t.Fatalf("counter-set before increment layout: %q, want %q", got, "6 A")
	}
}

func TestAspectRatioOneToOne(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.box { width: 100pt; aspect-ratio: 1 / 1; background: #f00 }
`)
	res := layoutHTML(t, `<html><body><div class="box"></div></body></html>`, cssSheet)
	box := findBoxByClass(t, res, "box")

	if box.w < 99 || box.w > 101 {
		t.Fatalf("width=%.1f, want 100", box.w)
	}

	if box.height < 99 || box.height > 101 {
		t.Fatalf("aspect-ratio height=%.1f, want 100", box.height)
	}
}

func phase871PNG(t *testing.T, w, h int) []byte {
	t.Helper()

	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.Set(x, y, color.RGBA{R: 0xFF, A: 0xFF})
		}
	}

	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatalf("encode PNG: %v", err)
	}

	return out.Bytes()
}
