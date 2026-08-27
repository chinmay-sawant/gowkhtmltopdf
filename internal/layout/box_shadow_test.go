//nolint:testpackage,wsl,varnamelen,paralleltest,unparam // box-shadow chrome probes
package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

func TestBoxShadowParse(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="xy">x</div>
		<div class="blur0">x</div>
		<div class="blur">x</div>
		<div class="none">x</div>
		<div class="inset">x</div>
		<div class="named">x</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.xy { box-shadow: 2pt 2pt #000 }
		.blur0 { box-shadow: 2pt 2pt 0 #000 }
		.blur { box-shadow: 2pt 2pt 4pt #000 }
		.none { box-shadow: none }
		.inset { box-shadow: inset 2pt 2pt #000 }
		.named { box-shadow: 2pt 2pt black }
	`)}, "print", testViewport, 800)

	black := [3]float64{0, 0, 0}

	xy := styleByClass(t, styles, "xy")
	assertBoxShadow(t, xy, 2, 2, 0, black, true)

	blur0 := styleByClass(t, styles, "blur0")
	assertBoxShadow(t, blur0, 2, 2, 0, black, true)

	blur := styleByClass(t, styles, "blur")
	assertBoxShadow(t, blur, 2, 2, 4, black, true)

	none := styleByClass(t, styles, "none")
	if none.BoxShadowSet {
		t.Fatalf("box-shadow:none set=%v, want false", none.BoxShadowSet)
	}

	inset := styleByClass(t, styles, "inset")
	if inset.BoxShadowSet {
		t.Fatalf("inset box-shadow set=%v, want false (ignored)", inset.BoxShadowSet)
	}

	named := styleByClass(t, styles, "named")
	assertBoxShadow(t, named, 2, 2, 0, black, true)
}

func TestBoxShadowPaints(t *testing.T) {
	t.Parallel()

	t.Run("offset-fill", testBoxShadowOffsetFill)
	t.Run("layout-size-unchanged", testBoxShadowLayoutSizeUnchanged)
}

func testBoxShadowOffsetFill(t *testing.T) {
	t.Parallel()

	eng := &engine{scale: 1, opts: Options{Background: true}} //nolint:exhaustruct // chrome probe
	sty := ResolvedStyle{                                     //nolint:exhaustruct // shadow fields under test
		BGColor:        [4]float64{1, 1, 1, 1},
		BoxShadowX:     2,
		BoxShadowY:     2,
		BoxShadowColor: [3]float64{0, 0, 0},
		BoxShadowSet:   true,
	}
	boxNode := &box{style: &sty, w: 100, height: 50} //nolint:exhaustruct // geometry probe
	eng.prependChrome(0, boxNode, sty, 10, 20, 100, 50)

	if len(eng.deferredChrome) != 1 {
		t.Fatalf("deferred chrome entries = %d, want 1", len(eng.deferredChrome))
	}

	if !hasOffsetShadowFill(eng.deferredChrome[0].ops, 12, 22, 100, 50) {
		t.Fatalf("missing shadow fill at 12,22 100x50 in %+v", eng.deferredChrome[0].ops)
	}
}

func testBoxShadowLayoutSizeUnchanged(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.box { width: 100pt; height: 40pt; background: #fff; box-shadow: 2pt 2pt #000 }
`)
	res := layoutHTML(t, `<html><body><div class="box">x</div></body></html>`, cssSheet)
	boxNode := findBoxByClass(t, res, "box")
	if !near(boxNode.w, 100) {
		t.Fatalf("box width = %.3f, want 100 (box-shadow must not grow layout)", boxNode.w)
	}

	if !hasOffsetShadowFill(res.Ops, boxNode.x+2, boxNode.y+2, boxNode.w, boxNode.height) {
		t.Fatalf("missing shadow fill at offset of box %+v in ops", boxNode)
	}
}

func assertBoxShadow(t *testing.T, sty *ResolvedStyle, offsetX, offsetY, blur float64, color [3]float64, set bool) {
	t.Helper()

	if sty.BoxShadowSet != set {
		t.Fatalf("BoxShadowSet = %v, want %v", sty.BoxShadowSet, set)
	}

	if !near(sty.BoxShadowX, offsetX) || !near(sty.BoxShadowY, offsetY) {
		t.Fatalf("box-shadow offset = %.3f,%.3f, want %.3f,%.3f",
			sty.BoxShadowX, sty.BoxShadowY, offsetX, offsetY)
	}

	if !near(sty.BoxShadowBlur, blur) {
		t.Fatalf("box-shadow blur = %.3f, want %.3f", sty.BoxShadowBlur, blur)
	}

	if sty.BoxShadowColor != color {
		t.Fatalf("box-shadow color = %v, want %v", sty.BoxShadowColor, color)
	}
}

func hasOffsetShadowFill(ops []Op, x, y, w, h float64) bool {
	for _, op := range ops {
		if op.Kind != OpFillRect {
			continue
		}

		black := op.R == 0 && op.G == 0 && op.B == 0
		if black && near(op.X, x) && near(op.Y, y) && near(op.W, w) && near(op.H, h) {
			return true
		}
	}

	return false
}
