//nolint:testpackage,wsl,varnamelen,paralleltest,unparam // box-shadow chrome probes
package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// Chrome ignores box-shadow-position when a box-shadow shorthand is present.
// Paint must keep the shorthand layer (outer here), not force inset.
func TestBoxShadowPositionLonghandDoesNotOverrideShorthandRaw(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div class="pos">x</div>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		.pos { background: #ffe; box-shadow: 2pt 2pt 8pt #333; box-shadow-position: inset }
	`)}, "print", testViewport, 800)
	sty := styleByClass(t, styles, "pos")
	if !sty.BoxShadowSet {
		t.Fatal("BoxShadowSet false")
	}
	if sty.BoxShadowRaw == "" {
		t.Fatal("BoxShadowRaw empty; shorthand raw must remain for Chrome-matched paint")
	}
	if !strings.Contains(sty.BoxShadowRaw, "2pt") {
		t.Fatalf("BoxShadowRaw = %q, want shorthand lengths", sty.BoxShadowRaw)
	}

	eng := &engine{scale: 1, opts: Options{Background: true}} //nolint:exhaustruct // chrome probe
	boxNode := &box{style: sty, w: 80, height: 36}            //nolint:exhaustruct // geometry probe
	eng.prependChrome(0, boxNode, *sty, 10, 20, 80, 36)
	ops := eng.deferredChrome[0].ops
	// Outer shadow: dark fill before cream background, larger/offset from box.
	bgIdx := -1
	outerBefore := false
	for i, op := range ops {
		if op.Kind == OpFillRect && op.R > 0.9 && op.B > 0.9 {
			bgIdx = i
			break
		}
	}
	if bgIdx < 0 {
		t.Fatalf("missing background in %+v", ops)
	}
	for i := 0; i < bgIdx; i++ {
		op := ops[i]
		if op.Kind == OpFillRect && op.R < 0.5 {
			outerBefore = true
			break
		}
	}
	if !outerBefore {
		t.Fatalf("expected outer shadow fills before background; bgIdx=%d", bgIdx)
	}
}

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
	assertBoxShadow(t, inset, 2, 2, 0, black, true)
	if !inset.BoxShadowInset {
		t.Fatal("inset box-shadow not marked BoxShadowInset")
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
	t.Run("inset-after-background", testBoxShadowInsetAfterBackground)
	t.Run("inset-top-left-rim", testBoxShadowInsetTopLeftRim)
}

// Inset shadows must splice after the background fill so the cream box does
// not hide the inner rim (fixture-61 #12 / Chrome print).
func testBoxShadowInsetAfterBackground(t *testing.T) {
	t.Parallel()

	eng := &engine{scale: 1, opts: Options{Background: true}} //nolint:exhaustruct // chrome probe
	sty := ResolvedStyle{                                     //nolint:exhaustruct // shadow fields under test
		BGColor:        [4]float64{1, 1, 0.933, 1},
		BoxShadowX:     2,
		BoxShadowY:     2,
		BoxShadowBlur:  8,
		BoxShadowColor: [3]float64{0.2, 0.2, 0.2},
		BoxShadowInset: true,
		BoxShadowSet:   true,
		BoxShadowRaw:   "inset 2pt 2pt 8pt #333",
	}
	boxNode := &box{style: &sty, w: 80, height: 30} //nolint:exhaustruct // geometry probe
	eng.prependChrome(0, boxNode, sty, 10, 20, 80, 30)

	if len(eng.deferredChrome) != 1 {
		t.Fatalf("deferred chrome entries = %d, want 1", len(eng.deferredChrome))
	}
	ops := eng.deferredChrome[0].ops
	bgIdx, shadowAfter := -1, false
	for i, op := range ops {
		if op.Kind == OpFillRect && op.R > 0.9 && op.G > 0.9 && op.B > 0.9 {
			bgIdx = i
		}
		if bgIdx >= 0 && i > bgIdx && op.Kind == OpFillRect && op.R < 0.5 && op.G < 0.5 {
			shadowAfter = true
			break
		}
	}
	if bgIdx < 0 {
		t.Fatalf("missing cream background fill in %+v", ops)
	}
	if !shadowAfter {
		t.Fatalf("inset shadow fills must follow background (bgIdx=%d) in %+v", bgIdx, ops)
	}
}

// inset 2pt 2pt must darken the top and left inner edges, not only top/bottom.
func testBoxShadowInsetTopLeftRim(t *testing.T) {
	t.Parallel()

	eng := &engine{scale: 1, opts: Options{Background: true}} //nolint:exhaustruct // chrome probe
	sty := ResolvedStyle{                                     //nolint:exhaustruct // shadow fields under test
		BGColor:        [4]float64{1, 1, 0.933, 1},
		BoxShadowX:     2,
		BoxShadowY:     2,
		BoxShadowBlur:  8,
		BoxShadowColor: [3]float64{0.2, 0.2, 0.2},
		BoxShadowInset: true,
		BoxShadowSet:   true,
		BoxShadowRaw:   "inset 2pt 2pt 8pt #333",
	}
	boxNode := &box{style: &sty, w: 80, height: 30} //nolint:exhaustruct // geometry probe
	eng.prependChrome(0, boxNode, sty, 10, 20, 80, 30)
	ops := eng.deferredChrome[0].ops

	hasTop, hasLeft := false, false
	for _, op := range ops {
		if op.Kind != OpFillRect || op.R > 0.5 {
			continue
		}
		// Top rim: near box top, spans most of width, short height.
		if near(op.Y, 20) && op.H > 1 && op.H < 20 && op.W > 40 {
			hasTop = true
		}
		// Left rim: near box left, tall strip.
		if near(op.X, 10) && op.W > 1 && op.W < 25 && op.H > 15 {
			hasLeft = true
		}
	}
	if !hasTop || !hasLeft {
		t.Fatalf("inset rim top=%v left=%v, want both; ops=%+v", hasTop, hasLeft, ops)
	}
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
