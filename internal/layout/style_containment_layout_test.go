//nolint:all // containment layout behavior tests
package layout

import (
	"strings"
	"testing"
)

// content-visibility: hidden must skip descendant layout and paint. The box
// keeps its own chrome and has no content height when no intrinsic size is set.
func TestContainmentContentVisibilityHidden(t *testing.T) {
	t.Parallel()

	s := sheet(t, `.hidden { content-visibility: hidden }`+
		` .hidden p { margin: 0; font-size: 40pt; line-height: 1 }`)
	res := layoutHTML(t, `<html><body><div class="hidden"><p>HIDDEN-TEXT</p></div></body></html>`, s)

	b := findBox(t, res, "div")
	if !near(b.height, 0) {
		t.Fatalf("content-visibility: hidden div height = %v, want 0", b.height)
	}

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "HIDDEN-TEXT") {
			t.Fatalf("content-visibility: hidden descendant painted: %+v", op)
		}
	}
}

// content-visibility: hidden still honors contain-intrinsic-size for its own
// content height.
func TestContainmentContentVisibilityHiddenIntrinsicSize(t *testing.T) {
	t.Parallel()

	s := sheet(t, `.hidden { content-visibility: hidden; contain-intrinsic-size: 30pt 18pt }`)
	res := layoutHTML(t, `<html><body><div class="hidden"><p>HIDDEN-TEXT</p></div></body></html>`, s)

	b := findBox(t, res, "div")
	if !near(b.height, 18) {
		t.Fatalf("hidden div height = %v, want contain-intrinsic height 18", b.height)
	}
}

// A size-contained block uses contain-intrinsic-size (height axis) instead of
// its real content height, while descendants still paint and may overflow.
func TestContainmentIntrinsicSizeOverridesContentHeight(t *testing.T) {
	t.Parallel()

	s := sheet(t, `.sized { contain: size; contain-intrinsic-size: 10pt 20pt }`+
		` .sized p { margin: 0; font-size: 40pt; line-height: 1.2 }`)
	res := layoutHTML(t, `<html><body><div class="sized"><p>OVERFLOW</p></div></body></html>`, s)

	b := findBox(t, res, "div")
	if !near(b.height, 20) {
		t.Fatalf("size-contained div height = %v, want 20 (contain-intrinsic-size height)", b.height)
	}

	if len(opsOfKind(res, OpText)) == 0 {
		t.Fatal("contain: size must keep descendant paint, got no text ops")
	}
}

// contain-intrinsic-block-size maps to height in horizontal-tb.
func TestContainmentIntrinsicBlockSizeMapsToHeight(t *testing.T) {
	t.Parallel()

	s := sheet(t, `.sized { contain: size; contain-intrinsic-block-size: 24pt }`+
		` .sized p { margin: 0; font-size: 30pt }`)
	res := layoutHTML(t, `<html><body><div class="sized"><p>X</p></div></body></html>`, s)

	b := findBox(t, res, "div")
	if !near(b.height, 24) {
		t.Fatalf("block-size contained div height = %v, want 24", b.height)
	}
}

// contain-intrinsic-inline-size maps to width in horizontal-tb: a size-contained
// float takes the intrinsic inline size, not the max-content of its text.
func TestContainmentIntrinsicInlineSizeSizesFloat(t *testing.T) {
	t.Parallel()

	s := sheet(t, `.f { float: left; contain: size; contain-intrinsic-inline-size: 60pt }`)
	res := layoutHTML(t, `<html><body><div class="f">hello</div></body></html>`, s)

	b := findBox(t, res, "div")
	if !near(b.w, 60) {
		t.Fatalf("size-contained float width = %v, want intrinsic inline size 60", b.w)
	}
}

// size-contained inline-blocks use contain-intrinsic-width,
// contain-intrinsic-inline-size, and the contain-intrinsic-size x-axis as their
// used content width. The property is an explicit intrinsic inner size, so the
// used border-box width adds the horizontal chrome for every box-sizing value.
// Without an intrinsic width the box stays at its chrome-only size.
func TestContainmentIntrinsicWidthSizesInlineBlock(t *testing.T) {
	t.Parallel()

	s := sheet(t, `.ib { display: inline-block; contain: size; padding: 5pt; border: 2pt solid #000 }`+
		` .width-intr { contain-intrinsic-width: 88px }`+
		` .bordbox-intr { contain-intrinsic-width: 88px; box-sizing: border-box }`+
		` .inline-intr { contain-intrinsic-inline-size: 70pt }`+
		` .short-intr { contain-intrinsic-size: 80pt 30pt }`+
		` .fixture-bordbox { contain-intrinsic-inline-size: 96px; box-sizing: border-box;`+
		` padding: 2px; border: 1px solid #246 }`)
	res := layoutHTML(t, `<html><body>`+
		`<div class="ib width-intr">x</div>`+
		`<div class="ib bordbox-intr">x</div>`+
		`<div class="ib inline-intr">x</div>`+
		`<div class="ib short-intr">x</div>`+
		`<div class="ib chrome-only">x</div>`+
		`<div class="ib fixture-bordbox">x</div>`+
		`</body></html>`, s)

	// contain-intrinsic-* is a content (inner) size, so the used border-box
	// width adds padding+border for content-box and border-box alike.
	const chrome = 5 + 5 + 2 + 2

	cases := []struct {
		class string
		want  float64
	}{
		{"width-intr", pxToPt(88) + chrome},
		{"bordbox-intr", pxToPt(88) + chrome},
		{"inline-intr", 70 + chrome},
		{"short-intr", 80 + chrome},
		{"chrome-only", chrome},
		// fixture-61 row 36 shape: 96px inner + 2px padding per side + 1px
		// border per side. The engine clamps borders to a 1pt hairline, so
		// the total is 77pt (102.67 CSS px), not the ideal 102.
		{"fixture-bordbox", pxToPt(96) + 2*1.5 + 2*1},
	}

	for _, testCase := range cases {
		b := findBoxByClass(t, res, testCase.class)
		if !near(b.w, testCase.want) {
			t.Fatalf("%s inline-block width = %v, want %v", testCase.class, b.w, testCase.want)
		}
	}
}

// contain: paint clips descendant paint to the box, reusing the overflow:clip
// pass (overflow_clip.go). An oversized descendant background must be cut down
// to the containing box width.
func TestContainmentPaintClipsDescendants(t *testing.T) {
	t.Parallel()

	s := sheet(t, `.clip { contain: paint; width: 40pt; height: 30pt }`+
		` .clip p { margin: 0; width: 300pt; height: 10pt; background: #00ff00 }`)
	res := layoutHTML(t, `<html><body><div class="clip"><p>WIDE</p></div></body></html>`, s)

	if len(opsOfKind(res, OpFillRect)) == 0 {
		t.Fatal("no fill ops emitted")
	}

	found := false

	for _, op := range res.Ops {
		if op.Kind != OpFillRect || op.Alpha <= 0 {
			continue
		}

		if op.R > 0.1 || op.G < 0.9 || op.B > 0.1 {
			continue
		}

		found = true

		if op.W > 41 {
			t.Fatalf("contain: paint descendant fill not clipped: W = %v, op = %+v", op.W, op)
		}
	}

	if !found {
		t.Fatal("contain: paint descendant green fill not found")
	}
}

// contain: layout makes the box the containing block for absolute descendants
// (padding box), so a child with left:0 anchors at the parent border box, not
// inside the parent padding.
func TestContainmentLayoutIsAbsoluteContainingBlock(t *testing.T) {
	t.Parallel()

	s := sheet(t, `.cb { contain: layout; padding: 10pt }`+
		` .cb .abs { position: absolute; left: 0; top: 0 }`)
	res := layoutHTML(t, `<html><body><div class="cb"><span class="abs">A</span></div></body></html>`, s)

	texts := opsOfKind(res, OpText)
	if len(texts) != 1 {
		t.Fatalf("text ops = %+v, want 1", texts)
	}

	// Body UA margin 8px = 6pt. Without containment the absolute box anchors
	// to the parent content box (6 + 10); layout containment uses the padding
	// box (6).
	if !near(texts[0].X, 6) {
		t.Fatalf("absolute descendant X = %v, want 6 (contain:layout padding-box origin)", texts[0].X)
	}
}

// contain: layout is an independent formatting context: descendant floats are
// trapped and enclosed by its used height.
func TestContainmentLayoutEnclosesFloats(t *testing.T) {
	t.Parallel()

	s := sheet(t, `.fc { contain: layout } .fc .fl { float: left; width: 20pt; height: 30pt }`)
	res := layoutHTML(t, `<html><body><div class="fc"><div class="fl"></div></div></body></html>`, s)

	b := findBox(t, res, "div")
	if b.height < 29.99 {
		t.Fatalf("layout containment did not enclose the float: height = %v, want >= 30", b.height)
	}
}

// Auto table layout must not let size-contained descendants widen the column:
// contain: size / strict and content-visibility: hidden size the box from
// contain-intrinsic-* as if its contents were absent (compatibility matrix
// §2.2). Regression: the cellMeasure walk descended into the contained
// content and used the long word's width for the column.
func TestContainmentIntrinsicWidthSizesAutoTableCell(t *testing.T) {
	t.Parallel()

	const long = "supercalifragilisticexpialidocious-supercalifragilisticexpialidocious"

	s := sheet(t, `
body { margin: 0; font-size: 10pt; }
table { table-layout: auto; border-collapse: collapse; border-spacing: 0; }
td { padding: 0; }
.hidden { content-visibility: hidden; }
.sized { contain: size; contain-intrinsic-inline-size: 30pt; }
`)

	res := layoutHTML(t, `<html><body><table><tr>`+
		`<td><div class="hidden">`+long+`</div></td>`+
		`<td><div class="sized">`+long+`</div></td>`+
		`<td>B</td>`+
		`</tr></table></body></html>`, s)

	tbl := findNamedBox(res.root, "table")
	if tbl == nil || len(tbl.rows) == 0 || len(tbl.rows[0]) < 3 {
		t.Fatal("missing table or first-row cells")
	}

	hiddenW := tbl.rows[0][0].w
	sizedW := tbl.rows[0][1].w

	if hiddenW > 5 {
		t.Fatalf("content-visibility:hidden cell width = %v, want chrome-only (hidden text must not widen the auto column)", hiddenW)
	}

	if !near(sizedW, 30) {
		t.Fatalf("contain:size cell width = %v, want contain-intrinsic-inline-size 30", sizedW)
	}
}
