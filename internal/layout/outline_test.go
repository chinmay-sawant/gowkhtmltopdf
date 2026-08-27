//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"testing"
)

func TestOutlineStroke(t *testing.T) {
	t.Parallel()

	t.Run("outside-border-edge", testOutlineOutsideBorder)
	t.Run("dashed-dotted", testOutlineDashedDotted)
	t.Run("offset-gap", testOutlineOffsetGap)
	t.Run("layout-size-unchanged", testOutlineLayoutSizeUnchanged)
	t.Run("prepend-chrome", testOutlinePrependChrome)
}

func testOutlineOutsideBorder(t *testing.T) {
	t.Parallel()

	const boxW, boxH, width = 100.0, 40.0, 10.0

	ops := appendOutlineOps(nil, 0, 0, boxW, boxH, width, 0, solidKeyword, 1, 0, 0)
	if len(ops) != 4 {
		t.Fatalf("solid outline ops = %d, want 4", len(ops))
	}

	inflate := outlineInflate(width, 0)
	assertOutlineOnInflatedRect(t, ops, -inflate, -inflate, boxW+two*inflate, boxH+two*inflate, width)

	for _, op := range ops {
		if op.Kind != OpLine {
			t.Fatalf("outline op kind = %v, want OpLine", op.Kind)
		}

		if op.Width != width {
			t.Fatalf("outline width = %v, want %v", op.Width, width)
		}

		inside := op.X > 0 && op.X+op.W < boxW && op.Y > 0 && op.Y+op.H < boxH
		if inside {
			t.Fatalf("outline segment %+v sits inside the border box", op)
		}
	}
}

func testOutlineDashedDotted(t *testing.T) {
	t.Parallel()

	dashed := appendOutlineOps(nil, 0, 0, 100, 40, 2, 0, borderStyleDashed, 0, 0, 1)
	if len(dashed) <= 4 {
		t.Fatalf("dashed outline ops = %d, want dashed segments (>4)", len(dashed))
	}

	dotted := appendOutlineOps(nil, 0, 0, 100, 40, 2, 0, borderStyleDotted, 0, 0, 1)
	if len(dotted) <= 4 {
		t.Fatalf("dotted outline ops = %d, want dotted segments (>4)", len(dotted))
	}

	none := appendOutlineOps(nil, 0, 0, 100, 40, 10, 0, cssDisplayNone, 1, 0, 0)
	if len(none) != 0 {
		t.Fatalf("none outline ops = %d, want 0", len(none))
	}
}

func testOutlineOffsetGap(t *testing.T) {
	t.Parallel()

	const width, offset = 8.0, 5.0

	ops := appendOutlineOps(nil, 10, 20, 100, 40, width, offset, solidKeyword, 0, 1, 0)
	inflate := outlineInflate(width, offset)
	zeroOff := outlineInflate(width, 0)
	if inflate <= zeroOff {
		t.Fatalf("offset inflate %v should exceed zero-offset %v", inflate, zeroOff)
	}

	assertOutlineOnInflatedRect(t, ops, 10-inflate, 20-inflate, 100+two*inflate, 40+two*inflate, width)
}

func testOutlineLayoutSizeUnchanged(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.box { width: 100pt; outline: 10pt solid #f00; outline-offset: 4pt }
`)
	res := layoutHTML(t, `<html><body><div class="box">x</div></body></html>`, cssSheet)
	boxNode := findBoxByClass(t, res, "box")
	if !near(boxNode.w, 100) {
		t.Fatalf("box width = %.3f, want 100 (outline must not grow layout)", boxNode.w)
	}
}

func testOutlinePrependChrome(t *testing.T) {
	t.Parallel()

	eng := &engine{scale: 1, opts: Options{Background: true}} //nolint:exhaustruct // chrome probe
	sty := ResolvedStyle{                                     //nolint:exhaustruct // outline fields under test
		OutlineWidth:    10,
		OutlineStyle:    solidKeyword,
		OutlineColor:    [3]float64{1, 0, 0},
		OutlineColorSet: true,
		OutlineOffset:   4,
	}
	boxNode := &box{style: &sty, w: 100, height: 50} //nolint:exhaustruct // geometry probe
	eng.prependChrome(0, boxNode, sty, 0, 0, 100, 50)

	if len(eng.deferredChrome) != 1 {
		t.Fatalf("deferred chrome entries = %d, want 1", len(eng.deferredChrome))
	}

	ops := eng.deferredChrome[0].ops
	inflate := outlineInflate(10, 4)
	assertOutlineOnInflatedRect(t, ops, -inflate, -inflate, 100+two*inflate, 50+two*inflate, 10)
}

func assertOutlineOnInflatedRect(t *testing.T, ops []Op, x, y, w, h, width float64) {
	t.Helper()

	var top, right, bottom, left bool

	for _, op := range ops {
		if op.Kind != OpLine || !near(op.Width, width) {
			continue
		}

		switch {
		case op.H == 0 && near(op.Y, y) && near(op.X, x):
			top = true
		case op.H == 0 && near(op.Y, y+h) && near(op.X, x):
			bottom = true
		case op.W == 0 && near(op.X, x) && near(op.Y, y):
			left = true
		case op.W == 0 && near(op.X, x+w) && near(op.Y, y):
			right = true
		}
	}

	if !top || !right || !bottom || !left {
		t.Fatalf("missing outline sides top=%v right=%v bottom=%v left=%v (ops=%d)",
			top, right, bottom, left, len(ops))
	}
}
