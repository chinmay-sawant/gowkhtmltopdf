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
		<div class="spread">x</div>
		<div class="none">x</div>
		<div class="inset">x</div>
		<div class="named">x</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.xy { box-shadow: 2pt 2pt #000 }
		.blur0 { box-shadow: 2pt 2pt 0 #000 }
		.blur { box-shadow: 2pt 2pt 4pt #000 }
		.spread { box-shadow: 2pt 2pt 4pt 3pt #000 }
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

	spread := styleByClass(t, styles, "spread")
	assertBoxShadow(t, spread, 2, 2, 4, black, true)
	if !near(spread.BoxShadowSpread, 3) {
		t.Fatalf("box-shadow spread = %.3f, want 3", spread.BoxShadowSpread)
	}

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
	t.Run("spread-fill", testBoxShadowSpreadFill)
	t.Run("layout-size-unchanged", testBoxShadowLayoutSizeUnchanged)
	t.Run("rounded-fill", testBoxShadowRoundedFill)
}

func testBoxShadowSpreadFill(t *testing.T) {
	t.Parallel()

	eng := &engine{scale: 1, opts: Options{Background: true}} //nolint:exhaustruct // chrome probe
	sty := ResolvedStyle{                                     //nolint:exhaustruct // shadow fields under test
		BGColor:         [4]float64{1, 1, 1, 1},
		BoxShadowX:      2,
		BoxShadowY:      2,
		BoxShadowSpread: 5,
		BoxShadowColor:  [3]float64{0, 0, 0},
		BoxShadowSet:    true,
	}
	boxNode := &box{style: &sty, w: 100, height: 50} //nolint:exhaustruct // geometry probe
	eng.prependChrome(0, boxNode, sty, 10, 20, 100, 50)

	if len(eng.deferredChrome) != 1 {
		t.Fatalf("deferred chrome entries = %d, want 1", len(eng.deferredChrome))
	}

	// 10 + 2 - 5 = 7, 20 + 2 - 5 = 17, 100 + 10 = 110, 50 + 10 = 60
	if !hasOffsetShadowFill(eng.deferredChrome[0].ops, 7, 17, 110, 60) {
		t.Fatalf("missing spread shadow fill at 7,17 110x60 in %+v", eng.deferredChrome[0].ops)
	}
}

func TestBoxShadowBlurPaints(t *testing.T) {
	t.Parallel()

	t.Run("stacked-fills", testBoxShadowBlurStackedFills)
	t.Run("layout-size-unchanged", testBoxShadowBlurLayoutSizeUnchanged)
}

func testBoxShadowBlurStackedFills(t *testing.T) {
	t.Parallel()

	eng := &engine{scale: 1, opts: Options{Background: true}} //nolint:exhaustruct // chrome probe
	sharp := ResolvedStyle{                                   //nolint:exhaustruct // shadow fields under test
		BoxShadowX:     2,
		BoxShadowY:     2,
		BoxShadowColor: [3]float64{0, 0, 0},
		BoxShadowSet:   true,
	}
	soft := sharp
	soft.BoxShadowBlur = 4
	boxNode := &box{style: &sharp, w: 100, height: 50} //nolint:exhaustruct // geometry probe

	eng.prependChrome(0, boxNode, sharp, 10, 20, 100, 50)
	sharpFills := countBlackFills(eng.deferredChrome[0].ops)

	eng.deferredChrome = nil
	boxNode.style = &soft
	eng.prependChrome(0, boxNode, soft, 10, 20, 100, 50)
	softOps := eng.deferredChrome[0].ops
	softFills := countBlackFills(softOps)

	if sharpFills != 1 {
		t.Fatalf("blur=0 black fills = %d, want 1", sharpFills)
	}

	if softFills <= sharpFills {
		t.Fatalf("blur>0 black fills = %d, want more than blur=0 (%d)", softFills, sharpFills)
	}

	if !hasOffsetShadowFill(softOps, 12, 22, 100, 50) {
		t.Fatalf("missing core shadow fill at 12,22 100x50 in %+v", softOps)
	}

	if !hasExpandedShadowFill(softOps, 12, 22, 100, 50) {
		t.Fatalf("missing expanded blur fill larger than 100x50 in %+v", softOps)
	}
}

func testBoxShadowBlurLayoutSizeUnchanged(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0 }
.box { width: 100pt; height: 40pt; background: #fff; box-shadow: 2pt 2pt 4pt #000 }
`)
	res := layoutHTML(t, `<html><body><div class="box">x</div></body></html>`, cssSheet)
	boxNode := findBoxByClass(t, res, "box")
	if !near(boxNode.w, 100) {
		t.Fatalf("box width = %.3f, want 100 (blur must not grow layout)", boxNode.w)
	}

	if !near(boxNode.height, 40) {
		t.Fatalf("box height = %.3f, want 40 (blur must not grow layout)", boxNode.height)
	}

	if !hasOffsetShadowFill(res.Ops, boxNode.x+2, boxNode.y+2, boxNode.w, boxNode.height) {
		t.Fatalf("missing core shadow fill at offset of box %+v in ops", boxNode)
	}

	if countBlackFills(res.Ops) <= 1 {
		t.Fatalf("blur>0 should paint more than the core fill, got %d black fills", countBlackFills(res.Ops))
	}
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

func testBoxShadowRoundedFill(t *testing.T) {
	t.Parallel()

	eng := &engine{scale: 1, opts: Options{Background: true}} //nolint:exhaustruct // chrome probe
	sty := ResolvedStyle{                                     //nolint:exhaustruct // rounded shadow fields under test
		BoxShadowX: 2, BoxShadowY: 2, BoxShadowColor: [3]float64{0, 0, 0}, BoxShadowSet: true,
		BorderRadius: 8, BorderRadiusPercent: -1,
	}
	boxNode := &box{style: &sty, w: 100, height: 50} //nolint:exhaustruct // geometry probe
	eng.prependChrome(0, boxNode, sty, 10, 20, 100, 50)

	for _, op := range eng.deferredChrome[0].ops {
		if op.Kind != OpFillRect || !near(op.X, 12) || !near(op.Y, 22) {
			continue
		}

		radiusX, radiusY := opRadiiXY(&op)
		if !near(radiusX[0], 8) || !near(radiusY[0], 8) {
			t.Fatalf("rounded shadow radii = X %.1f Y %.1f, want 8/8", radiusX[0], radiusY[0])
		}

		return
	}

	t.Fatal("rounded core shadow fill missing")
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

func hasExpandedShadowFill(ops []Op, x, y, w, h float64) bool {
	for _, op := range ops {
		if op.Kind != OpFillRect || op.R != 0 || op.G != 0 || op.B != 0 {
			continue
		}

		if op.W > w+0.01 && op.H > h+0.01 && op.X < x-0.01 && op.Y < y-0.01 {
			return true
		}
	}

	return false
}

func countBlackFills(ops []Op) int {
	count := 0

	for _, op := range ops {
		if op.Kind == OpFillRect && op.R == 0 && op.G == 0 && op.B == 0 {
			count++
		}
	}

	return count
}
