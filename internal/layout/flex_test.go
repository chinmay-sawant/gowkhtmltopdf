//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"math"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestFlexOrderAndShrink(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.row { display:flex; width:200pt; gap:0 }
.a { order:2; width:80pt; flex-shrink:0 }
.b { order:1; width:80pt; flex-shrink:1 }
.c { order:3; width:80pt; flex-shrink:1 }
`)
	res := layoutHTML(t, `<html><body>
<div class="row"><div class="a">A</div><div class="b">B</div><div class="c">C</div></div>
</body></html>`, cssSheet)
	pos := map[string]float64{}

	for _, op := range res.Ops {
		if op.Kind == OpText {
			pos[op.Text] = op.X
		}
	}

	if !(pos["B"] < pos["A"] && pos["A"] < pos["C"]) {
		t.Fatalf("order positions B/A/C = %.1f/%.1f/%.1f", pos["B"], pos["A"], pos["C"])
	}
	// Total intrinsic 240 > 200 → B and C shrink (to ~60pt each), A (shrink 0) stays 80pt.
	bWidth := pos["A"] - pos["B"]
	aWidth := pos["C"] - pos["A"]

	if bWidth < 55 || bWidth > 65 {
		t.Errorf("B shrunk width = %.1f, want ~60pt", bWidth)
	}

	if aWidth < 75 || aWidth > 85 {
		t.Errorf("A (non-shrunk) width = %.1f, want ~80pt", aWidth)
	}
}

func TestFloatWidthPercent(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.box { width:200pt }
.left { float:left; width:50%; background:#eee; padding:2pt }
.clear { clear:both }
`)
	res := layoutHTML(t, `<html><body>
<div class="box"><div class="left">L</div><p class="clear">after</p></div>
</body></html>`, cssSheet)

	var fillW float64

	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.R > 0.8 {
			fillW = op.W
		}
	}

	if fillW < 90 || fillW > 110 {
		t.Fatalf("float width%% fill W=%.1f, want ~100", fillW)
	}
}

func TestZIndexPaintOrder(t *testing.T) { //nolint:cyclop
	t.Parallel()

	cssSheet := sheet(t, `
.wrap { position:relative; height:40pt }
.low { position:absolute; top:0; left:0; width:40pt; height:20pt; background:#f00; z-index:1 }
.high { position:absolute; top:5pt; left:10pt; width:40pt; height:20pt; background:#00f; z-index:5 }
`)
	res := layoutHTML(t, `<html><body>
<div class="wrap"><div class="low">L</div><div class="high">H</div></div>
</body></html>`, cssSheet)
	doc := pdf.NewDocument()

	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}

	lowI, highI := -1, -1

	for idx, op := range res.Ops {
		if op.Kind == OpFillRect && op.R > 0.9 {
			lowI = idx
		}

		if op.Kind == OpFillRect && op.B > 0.9 {
			highI = idx
		}
	}

	if lowI < 0 || highI < 0 {
		t.Fatalf("fills low=%d high=%d", lowI, highI)
	}

	if !res.Ops[highI].ZIndexSet || res.Ops[highI].ZIndex != 5 {
		t.Fatalf("high z-index = %v/%d", res.Ops[highI].ZIndexSet, res.Ops[highI].ZIndex)
	}
}

func TestFlexRowLayout(t *testing.T) {
	t.Parallel()

	src := `<html><body>
<div style="display:flex;justify-content:space-between;gap:8pt;width:300pt">
  <div style="width:60pt">A</div>
  <div style="flex-grow:1">B</div>
  <div style="width:60pt">C</div>
</div>
</body></html>`

	res := layoutHTML(t, src)
	if res == nil || len(res.Ops) == 0 {
		t.Fatal("no ops")
	}
	// Three text runs should exist.
	texts := 0

	for _, op := range res.Ops {
		if op.Kind == OpText {
			texts++
		}
	}

	if texts < 3 {
		t.Fatalf("text ops = %d, want >= 3", texts)
	}
}

//nolint:lll,varnamelen,wsl // fixture-oriented flex test uses compact HTML
func TestFlexIntrinsicWidthIncludesNestedBlockContent(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.header { display:flex; justify-content:space-between; width:300pt }
.copy h2 { margin:0; font-size:20pt; line-height:1.15 }
.mark { width:30pt; height:30pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="header"><div class="copy"><div>01 / Route performance</div><h2>Where the week is moving</h2></div><div class="mark">02</div></div>
</body></html>`, cssSheet)

	var headingRuns []string
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}

		if strings.Contains(op.Text, "Where") || strings.Contains(op.Text, "week") || strings.Contains(op.Text, "moving") {
			headingRuns = append(headingRuns, op.Text)
		}
	}

	if len(headingRuns) != 1 || !strings.Contains(headingRuns[0], "Where the week is moving") {
		t.Fatalf("nested flex heading wrapped into %q; want one intrinsic line", headingRuns)
	}
}

func TestAbsoluteBottomUsesContainingBlockHeight(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin:0 }
.wrap { position:relative; width:100pt; height:100pt }
.abs { position:absolute; bottom:10pt; width:50pt; height:20pt; background:#f00 }
`)
	res := layoutHTML(t, `<html><body><div class="wrap"><div class="abs">bottom</div></div></body></html>`, cssSheet)

	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.R > 0.9 && op.W == 50 && op.H == 20 {
			if op.Y < 69 || op.Y > 71 {
				t.Fatalf("absolute bottom y=%.1f, want 70pt", op.Y)
			}

			return
		}
	}

	t.Fatal("absolute bottom background was not painted")
}

func TestAbsoluteChildUsesRelativePaddingBox(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin:0 }
.wrap { position:relative; width:100pt; height:100pt; padding:10pt 12pt; background:#eee }
.abs { position:absolute; left:12pt; bottom:10pt; width:100pt; height:10pt; background:#f00 }
`)
	res := layoutHTML(t, `<html><body><div class="wrap"><div class="abs"></div></div></body></html>`, cssSheet)

	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.R > 0.9 && op.W > 70 && op.H == 10 {
			if op.X < 11 || op.X > 13 || op.W < 99 || op.W > 101 {
				t.Fatalf("absolute child rect=(%.1f, %.1f, %.1f, %.1f), want x=12 width=100", op.X, op.Y, op.W, op.H)
			}

			return
		}
	}

	t.Fatal("absolute child background was not painted")
}

//nolint:wsl // fixture-oriented paint test keeps setup and assertions together
func TestBorderRadiusReachesRoundedPaintOps(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin:0 }
.dot { width:8pt; height:8pt; background:#f00; border-radius:50% }
.pill { width:50pt; height:12pt; border:1pt solid #00f; border-radius:6pt }
`)
	res := layoutHTML(t, `<html><body><div class="dot"></div><div class="pill"></div></body></html>`, cssSheet)

	var fillRadius, strokeRadius float64
	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.R > 0.9 {
			fillRadius = op.Radius
		}
		if op.Kind == OpStrokeRect && op.B > 0.9 {
			strokeRadius = op.Radius
		}
	}

	if fillRadius != 4 || strokeRadius != 6 {
		t.Fatalf("rounded radii fill=%.1f stroke=%.1f, want 4/6", fillRadius, strokeRadius)
	}
}

func TestFlexRowReverse(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.row { display:flex; flex-direction:row-reverse; width:200pt; gap:0 }
.a { width:40pt }
.b { width:40pt }
.c { width:40pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="row"><div class="a">A</div><div class="b">B</div><div class="c">C</div></div>
</body></html>`, cssSheet)
	pos := map[string]float64{}

	for _, op := range res.Ops {
		if op.Kind == OpText {
			pos[op.Text] = op.X
		}
	}

	if !(pos["C"] < pos["B"] && pos["B"] < pos["A"]) {
		t.Fatalf("row-reverse positions C/B/A = %.1f/%.1f/%.1f", pos["C"], pos["B"], pos["A"])
	}
}

func TestFlexSpaceEvenly(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.row { display:flex; justify-content:space-evenly; width:300pt; gap:0 }
.item { width:40pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="row"><div class="item">A</div><div class="item">B</div><div class="item">C</div></div>
</body></html>`, cssSheet)
	pos := map[string]float64{}

	for _, op := range res.Ops {
		if op.Kind == OpText {
			pos[op.Text] = op.X
		}
	}

	if len(pos) < 3 {
		t.Fatalf("missing texts: %v", pos)
	}
	// space-evenly: equal gaps at edges and between; A should not be at x≈0.
	dist1 := pos["B"] - pos["A"]
	dist2 := pos["C"] - pos["B"]

	if dist1 < 50 || dist2 < 50 {
		t.Fatalf("space-evenly gaps too small: A=%.1f B=%.1f C=%.1f", pos["A"], pos["B"], pos["C"])
	}

	if diff := dist1 - dist2; diff > 5 || diff < -5 {
		t.Fatalf("space-evenly gaps unequal: AB=%.1f BC=%.1f", dist1, dist2)
	}

	if pos["A"] < 20 {
		t.Fatalf("space-evenly leading gap missing: A.x=%.1f", pos["A"])
	}
}

func TestFlexColumnGapVsRowGap(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.row {
  display:flex; flex-wrap:wrap; width:100pt;
  column-gap:20pt; row-gap:40pt;
}
.item { width:40pt; height:10pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="row">
  <div class="item">A</div><div class="item">B</div>
  <div class="item">C</div><div class="item">D</div>
</div>
</body></html>`, cssSheet)
	posX, posY := map[string]float64{}, map[string]float64{}

	for _, op := range res.Ops {
		if op.Kind == OpText {
			posX[op.Text] = op.X
			posY[op.Text] = op.Y
		}
	}

	for _, k := range []string{"A", "B", "C", "D"} {
		if _, ok := posX[k]; !ok {
			t.Fatalf("missing text %s", k)
		}
	}
	// A,B on first line with ~20pt column-gap; C,D on second with ~40pt row-gap.
	if col := posX["B"] - posX["A"]; col < 50 || col > 70 {
		t.Fatalf("column-gap AB dx=%.1f, want ~60 (40+20)", col)
	}

	if row := posY["C"] - posY["A"]; row < 40 || row > 70 {
		t.Fatalf("row-gap AC dy=%.1f, want ~50 (10+40)", row)
	}

	if posY["B"]-posY["A"] > 5 {
		t.Fatalf("B should share A's line: Ay=%.1f By=%.1f", posY["A"], posY["B"])
	}
}

func isWrapItemFill(op Op) bool {
	return op.Kind == OpFillRect && op.W > 60 && op.W < 80 && op.H > 10 && op.H < 40
}

func TestFlexWrapRowGapSurvivesPaint(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.wrap {
  display: flex;
  flex-direction: row-reverse;
  flex-wrap: wrap;
  justify-content: space-evenly;
  column-gap: 12pt;
  row-gap: 4pt;
  width: 240pt;
  border: 1pt solid #1565c0;
  padding: 4pt;
}
.wrap > div {
  width: 70pt;
  padding: 4pt;
  border: 1pt solid #90caf9;
  background: #fff;
  box-sizing: border-box;
}
`)
	res := layoutHTML(t, `<html><body>
<div class="wrap"><div>A</div><div>B</div><div>C</div><div>D</div></div>
</body></html>`, cssSheet)

	if err := Paint(pdf.NewDocument(), res, PaintOptions{ //nolint:exhaustruct
		PageWidth: 400, PageHeight: 400,
	}); err != nil {
		t.Fatal(err)
	}

	var fills []Op

	for _, op := range res.Ops {
		if isWrapItemFill(op) {
			fills = append(fills, op)
		}
	}

	if len(fills) < 4 {
		t.Fatalf("wrap item fills = %d, want 4", len(fills))
	}

	gap := fills[3].Y - (fills[0].Y + fills[0].H)
	if gap < 3.5 || gap > 5 {
		t.Fatalf("row-gap after paint = %.2fpt, want 4pt: first=%+v second=%+v", gap, fills[0], fills[3])
	}
}

func TestFlexAlignSelf(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.row { display:flex; align-items:flex-start; width:200pt; height:60pt; gap:0 }
.a { width:40pt; height:10pt }
.b { width:40pt; height:10pt; align-self:flex-end }
`)
	res := layoutHTML(t, `<html><body>
<div class="row"><div class="a">A</div><div class="b">B</div></div>
</body></html>`, cssSheet)
	posY := map[string]float64{}

	for _, op := range res.Ops {
		if op.Kind == OpText {
			posY[op.Text] = op.Y
		}
	}

	if posY["B"] <= posY["A"]+10 {
		t.Fatalf("align-self flex-end: A.y=%.1f B.y=%.1f, want B lower", posY["A"], posY["B"])
	}
}

// TestFlexAlignItemsStretchRow matches fixture-33 definite row: container
// height 36pt, items flex-basis 50% with auto height → stretch to line cross size.
func TestFlexAlignItemsStretchRow(t *testing.T) { //nolint:cyclop
	t.Parallel()

	cssSheet := sheet(t, `
.row { display:flex; width:240pt; height:36pt; gap:0; border:1px solid #1565c0; background:#e3f2fd }
.half { flex:0 0 50%; box-sizing:border-box; padding:6pt; background:#90caf9 }
`)
	res := layoutHTML(t, `<html><body>
<div class="row"><div class="half">Left 50%</div><div class="half">Right 50%</div></div>
</body></html>`, cssSheet)

	var itemH []float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpFillRect {
			continue
		}
		// Item blue (#90caf9 ≈ 0.565, 0.792, 0.976), not container wash.
		if paintOp.R > 0.5 && paintOp.R < 0.7 && paintOp.B > 0.9 && paintOp.W > 80 {
			itemH = append(itemH, paintOp.H)
		}
	}

	if len(itemH) < 2 {
		t.Fatalf("expected ≥2 item fills, got %d (ops=%d)", len(itemH), len(res.Ops))
	}

	for i, h := range itemH {
		if h < 34 || h > 38 {
			t.Errorf("item[%d] fill h=%.2f, want ~36pt (stretched to row height)", i, h)
		}
	}
}

// TestAlignContentStretch grows wrapped flex lines into leftover cross space
// (and stretch items with auto height). Height:auto still packs at start.
func TestAlignContentStretch(t *testing.T) { //nolint:cyclop
	t.Parallel()

	cssSheet := sheet(t, `
.row {
  display: flex;
  flex-wrap: wrap;
  align-content: stretch;
  width: 100pt;
  height: 100pt;
  gap: 0;
}
.row > div { width: 80pt; background: #90caf9; font-size: 10pt }
.auto {
  display: flex;
  flex-wrap: wrap;
  align-content: stretch;
  width: 100pt;
  gap: 0;
}
.auto > div { width: 80pt; background: #ffcc80; font-size: 10pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="row"><div>A</div><div>B</div></div>
<div class="auto"><div>C</div><div>D</div></div>
</body></html>`, cssSheet)

	var stretchH, autoH []float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpFillRect {
			continue
		}

		if paintOp.R > 0.5 && paintOp.R < 0.7 && paintOp.B > 0.9 && paintOp.W > 60 {
			stretchH = append(stretchH, paintOp.H)
		}

		if paintOp.R > 0.9 && paintOp.G > 0.7 && paintOp.G < 0.9 && paintOp.W > 60 {
			autoH = append(autoH, paintOp.H)
		}
	}

	if len(stretchH) < 2 {
		t.Fatalf("stretch item fills = %d, want 2", len(stretchH))
	}

	for i, h := range stretchH {
		if h < 40 {
			t.Errorf("stretch item[%d] h=%.2f, want ~50pt (free cross space split across 2 lines)", i, h)
		}
	}

	if math.Abs(stretchH[0]-stretchH[1]) > 4 {
		t.Errorf("stretch line heights %.2f vs %.2f, want roughly equal", stretchH[0], stretchH[1])
	}

	if len(autoH) < 2 {
		t.Fatalf("height:auto item fills = %d, want 2", len(autoH))
	}

	for i, h := range autoH {
		if h > 30 {
			t.Errorf("height:auto item[%d] h=%.2f, want content-sized pack-at-start", i, h)
		}
	}
}

func TestFlexShorthandParsing(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()

	cases := []struct {
		name     string
		css      string
		grow     float64
		sh       float64
		basis    float64
		basisPct float64
	}{
		{cssDisplayNone, "flex:none", 0, 0, -1, -1},
		{overflowAuto, "flex:auto", 1, 1, -1, -1},
		{"one", "flex:1", 1, 1, -1, 0},
		{"three", "flex:0 0 80pt", 0, 0, 80, -1},
		{"grow-shrink-auto", "flex:1 1 auto", 1, 1, -1, -1},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			cssSheet := sheet(t, ".x { display:flex } .i { "+testCase.css+" }")
			doc := `<html><body><div class="x"><div class="i">Z</div></div></body></html>`

			res := layoutHTML(t, doc, cssSheet)
			if res == nil {
				t.Fatal("nil result")
			}
			// Re-resolve styles the same way layout does.
			root := mustParse(t, doc)
			styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "screen", 612, 792)

			var item *html.Node

			var find func(*html.Node)
			find = func(node *html.Node) {
				if node.Type == html.ElementNode && node.Attribute("class") == "i" {
					item = node

					return
				}

				for _, c := range node.Children {
					find(c)
				}
			}
			find(root)

			if item == nil {
				t.Fatal("item not found")
			}

			sty := styles[item]
			if sty.FlexGrow != testCase.grow || sty.FlexShrink != testCase.sh {
				t.Fatalf("grow/shrink = %v/%v, want %v/%v", sty.FlexGrow, sty.FlexShrink, testCase.grow, testCase.sh)
			}

			if sty.FlexBasis != testCase.basis || sty.FlexBasisPercent != testCase.basisPct {
				t.Fatalf("basis/pct = %v/%v, want %v/%v", sty.FlexBasis, sty.FlexBasisPercent, testCase.basis, testCase.basisPct)
			}
		})
	}
}

func TestFlexBasisPercentDefinite(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.row { display:flex; width:200pt; gap:0; height:30pt }
.a { flex: 0 0 50%; background:#fcc }
.b { flex: 0 0 50%; background:#ccf }
`)
	res := layoutHTML(t, `<html><body>
<div class="row"><div class="a">A</div><div class="b">B</div></div>
</body></html>`, cssSheet)

	var fills []float64

	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.W > 10 && op.H > 5 {
			fills = append(fills, op.W)
		}
	}

	if len(fills) < 2 {
		t.Fatalf("expected item fills, got %d", len(fills))
	}

	for i, w := range fills[:2] {
		if w < 90 || w > 110 {
			t.Fatalf("item %d width=%.1f, want ~100 (50%% of 200pt)", i, w)
		}
	}
}

func TestFlexBasisPercentCyclicColumn(t *testing.T) { //nolint:cyclop
	t.Parallel()
	// height:auto column → main size indefinite; % flex-basis must act as auto
	// (content-based), not resolve as 0.
	cssSheet := sheet(t, `
.col { display:flex; flex-direction:column; width:120pt; gap:0 }
.pct { flex: 0 0 50%; background:#cfc; padding:4pt }
.auto { flex: 0 0 auto; background:#ffc; padding:4pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="col">
  <div class="pct">PERCENT BASIS</div>
  <div class="auto">AUTO BASIS</div>
</div>
</body></html>`, cssSheet)

	var pctH, autoH float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpFillRect || paintOp.H < 2 {
			continue
		}

		switch {
		case paintOp.G > 0.7 && paintOp.R < 0.9: // #cfc
			pctH = paintOp.H
		case paintOp.R > 0.9 && paintOp.G > 0.9 && paintOp.B < 0.9: // #ffc
			autoH = paintOp.H
		}
	}

	if pctH < 8 {
		t.Fatalf("cyclic %% basis height=%.1f, want content-sized (>0), not silent 0", pctH)
	}

	if autoH < 8 {
		t.Fatalf("auto basis height=%.1f, want content-sized", autoH)
	}
	// Same padding/font → cyclic-% (as auto) should be near the auto sibling.
	if diff := pctH - autoH; diff > 4 || diff < -4 {
		t.Fatalf("cyclic %% height=%.1f vs auto=%.1f, want near-equal content sizes", pctH, autoH)
	}
}

func TestFlexBasisPercentDefiniteColumn(t *testing.T) { //nolint:cyclop
	t.Parallel()

	cssSheet := sheet(t, `
.col { display:flex; flex-direction:column; width:100pt; height:100pt; gap:0 }
.a { flex: 0 0 40%; background:#f99 }
.b { flex: 0 0 60%; background:#99f }
`)
	res := layoutHTML(t, `<html><body>
<div class="col"><div class="a">A</div><div class="b">B</div></div>
</body></html>`, cssSheet)

	var h40, h60 float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpFillRect || paintOp.H < 5 {
			continue
		}

		if paintOp.R > 0.9 && paintOp.G < 0.7 {
			h40 = paintOp.H
		}

		if paintOp.B > 0.9 && paintOp.R < 0.7 {
			h60 = paintOp.H
		}
	}

	if h40 < 35 || h40 > 45 {
		t.Fatalf("40%% basis height=%.1f, want ~40", h40)
	}

	if h60 < 55 || h60 > 65 {
		t.Fatalf("60%% basis height=%.1f, want ~60", h60)
	}
}

func TestFlexContentMinSizeDefiniteRow(t *testing.T) {
	t.Parallel()
	// Content-based min-width:auto must stop shrink from crushing long
	// unbreakable text below its intrinsic width inside a definite row.
	cssSheet := sheet(t, `
.row { display:flex; width:120pt; gap:0 }
.a { flex: 1 1 80pt; background:#fcc; white-space:nowrap }
.b { flex: 1 1 80pt; background:#ccf }
`)
	res := layoutHTML(t, `<html><body>
<div class="row">
  <div class="a">LONGWORDWITHOUTSPACES</div>
  <div class="b">B</div>
</div>
</body></html>`, cssSheet)

	var availW float64

	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.R > 0.9 && op.G < 0.9 && op.W > 5 {
			if op.W > availW {
				availW = op.W
			}
		}
	}

	if availW < 50 {
		t.Fatalf("content min floor crushed A to W=%.1f (want substantial intrinsic)", availW)
	}
	// A should keep more than equal half when content min exceeds shrink share.
	if availW <= 60 {
		t.Fatalf("A width=%.1f; content min should prefer A over equal 60pt split", availW)
	}
}

//nolint:lll,wsl // fixture-oriented flex test uses compact HTML and setup blocks
func TestFlexAutoMinUsesMinContentForWrappingText(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin:0; font-size:10pt; line-height:1.45 }
* { box-sizing:border-box }
.row { display:flex; width:504.6pt; gap:9pt; align-items:stretch }
.two-col > .panel { flex:1; padding:16px; border:1px solid #d8ddd7; background:#fffefa }
h3 { color:#173f45; font-size:10pt; letter-spacing:.05em; text-transform:uppercase }
`)
	res := layoutHTML(t, `<html><body>
<div class="row two-col">
  <section class="panel"><h3>Readout</h3><ul><li>Early loading is protecting the first two departures.</li><li>Canal-side delays cluster around the handoff.</li></ul></section>
  <section class="panel"><h3>Measure twice</h3><p>Use the return scan as the source of truth for completed stops. Driver notes remain useful context, but should not replace the scan.</p></section>
</div>
</body></html>`, cssSheet)

	maxRight := 0.0
	for _, op := range res.Ops {
		if op.Kind != OpFillRect || op.W < 5 || op.H < 5 {
			continue
		}

		if right := op.X + op.W; right > maxRight {
			maxRight = right
		}
	}

	// The row is 504.6pt wide and must not expand to the second panel's
	// max-content width.
	if maxRight > 520 {
		t.Fatalf("flex row overflowed to %.1fpt; ordinary text should use min-content auto minimum", maxRight)
	}
}

//nolint:lll,wsl,varnamelen // fixture-oriented masthead test uses compact HTML
func TestMastheadKeepsBrandAndMetadataOnTheirIntendedLines(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin:0; font-family:Arial, Helvetica, sans-serif; font-size:10pt; line-height:1.45 }
* { box-sizing:border-box }
.masthead, .brand-line { display:flex; align-items:center }
.masthead { width:504.6pt; justify-content:space-between; border-bottom:1px solid #d8ddd7; padding-bottom:10px }
.brand-line { gap:6.75pt }
.brand-mark { width:21pt; height:21pt; border:2px solid #173f45; border-radius:50%; font-size:9pt; font-weight:700; line-height:18pt; text-align:center }
.brand-name { font-size:11pt; font-weight:700; letter-spacing:.08em; text-transform:uppercase }
.eyebrow { font-size:7.5pt; font-weight:700; letter-spacing:.13em; text-transform:uppercase; text-align:right }
`)
	res := layoutHTML(t, `<html><body><header class="masthead">
<div class="brand-line"><span class="brand-mark">N</span><span class="brand-name">Northline Cooperative</span></div>
<div class="eyebrow">Field operations brief<br>06–12 August 2026</div>
</header></body></html>`, cssSheet)
	var brandY float64
	var brandRuns int
	var metadataLine bool

	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		switch {
		case strings.Contains(op.Text, "Northline"), strings.Contains(op.Text, "Cooperative"):
			if brandRuns == 0 {
				brandY = op.Y
			} else if op.Y != brandY {
				t.Fatalf("brand wrapped across baselines %.1f and %.1f: %q", brandY, op.Y, op.Text)
			}

			brandRuns++
		case strings.Contains(op.Text, "Field operations brief"):
			metadataLine = true
		}
	}

	if brandRuns == 0 {
		t.Fatal("brand name text was not painted")
	}

	if !metadataLine {
		t.Fatal("right-side metadata wrapped before its explicit line break")
	}
}

func TestFlexPercentChildDefiniteRow(t *testing.T) {
	t.Parallel()
	// % width child inside a definite flex item re-resolves against the item's
	// used main size (not the viewport).
	cssSheet := sheet(t, `
.row { display:flex; width:200pt; gap:0 }
.item { flex: 0 0 100pt; background:#eee }
.inner { width:50%; background:#f99; height:12pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="row"><div class="item"><div class="inner">X</div></div></div>
</body></html>`, cssSheet)

	var innerW float64

	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.R > 0.9 && op.G < 0.7 {
			innerW = op.W
		}
	}

	if innerW < 45 || innerW > 55 {
		t.Fatalf("inner width%% = %.1f, want ~50 (50%% of 100pt flex item)", innerW)
	}
}

func TestFlexMinWidthPercentDefinite(t *testing.T) { //nolint:cyclop
	t.Parallel()
	// min-width:% against definite flex container: A cannot shrink below 60%.
	cssSheet := sheet(t, `
.row { display:flex; width:200pt; gap:0 }
.a { flex: 0 1 150pt; min-width: 60%; background:#cfc }
.b { flex: 0 1 150pt; background:#ffc }
`)
	res := layoutHTML(t, `<html><body>
<div class="row"><div class="a">A</div><div class="b">B</div></div>
</body></html>`, cssSheet)

	var availW, boxW float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpFillRect || paintOp.W < 5 {
			continue
		}

		if paintOp.G > 0.7 && paintOp.R < 0.9 {
			availW = paintOp.W
		}

		if paintOp.R > 0.9 && paintOp.G > 0.9 && paintOp.B < 0.9 {
			boxW = paintOp.W
		}
	}

	if availW < 115 || availW > 130 {
		t.Fatalf("min-width:60%% floor A=%.1f, want ~120", availW)
	}

	if boxW < 65 || boxW > 90 {
		t.Fatalf("B after rebalance=%.1f, want ~80", boxW)
	}
}

func TestFlexNestedSmoke(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.outer { display:flex; width:220pt; gap:4pt }
.inner { display:flex; flex:1; gap:2pt; background:#eee }
.cell { flex:1; background:#ddd; min-width:20pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="outer">
  <div class="inner"><div class="cell">1</div><div class="cell">2</div></div>
  <div class="inner"><div class="cell">3</div><div class="cell">4</div></div>
</div>
</body></html>`, cssSheet)
	texts := 0

	for _, op := range res.Ops {
		if op.Kind == OpText {
			texts++
		}
	}

	if texts < 4 {
		t.Fatalf("nested flex smoke: got %d text ops, want ≥4", texts)
	}
}

func TestPositionRelativeAbsolute(t *testing.T) { //nolint:gocognit,cyclop,funlen
	t.Parallel()

	src := `<html><body>
<div style="position:relative;top:10pt;left:20pt">rel</div>
<div style="position:relative;height:40pt">
  <div style="position:absolute;top:5pt;left:15pt;width:50pt;background:#fff3e0">abs</div>
  in-flow
</div>
</body></html>`
	res := layoutHTML(t, src)

	var foundAbs, foundRel bool

	absTextIdx, flowTextIdx, absFillIdx := -1, -1, -1

	for idx, paintOp := range res.Ops {
		if paintOp.Kind == OpFillRect && paintOp.R > 0.9 && paintOp.G > 0.9 && paintOp.B > 0.8 {
			absFillIdx = idx
		}

		if paintOp.Kind != OpText {
			continue
		}

		t.Logf("text=%q x=%.1f y=%.1f", paintOp.Text, paintOp.X, paintOp.Y)

		if strings.Contains(paintOp.Text, "rel") && paintOp.X >= 20 {
			foundRel = true
		}

		if strings.Contains(paintOp.Text, "abs") {
			foundAbs = true
			absTextIdx = idx

			if paintOp.X < 10 {
				t.Errorf("abs x=%.1f, want offset", paintOp.X)
			}
		}

		if strings.Contains(paintOp.Text, "in-flow") {
			flowTextIdx = idx
		}
	}

	if !foundRel {
		t.Error("relative offset text not found")
	}

	if !foundAbs {
		t.Error("absolute text not found")
	}

	if flowTextIdx >= 0 && absFillIdx >= 0 && absFillIdx < flowTextIdx {
		t.Errorf("absolute fill op %d before in-flow text %d (overlay must paint above)", absFillIdx, flowTextIdx)
	}

	if flowTextIdx >= 0 && absTextIdx >= 0 && absTextIdx < flowTextIdx {
		t.Errorf("absolute text op %d before in-flow text %d", absTextIdx, flowTextIdx)
	}

	if flowTextIdx >= 0 && absFillIdx >= 0 {
		ordered := make([]int, len(res.Ops))
		for i := range ordered {
			ordered[i] = i
		}

		sortPaintIndices(res.Ops, ordered)

		flowOrder, absFillOrder := -1, -1

		for order, idx := range ordered {
			switch idx {
			case flowTextIdx:
				flowOrder = order
			case absFillIdx:
				absFillOrder = order
			}
		}

		if absFillOrder < flowOrder {
			t.Errorf("absolute overlay fill paints before in-flow text: fill order=%d flow order=%d", absFillOrder, flowOrder)
		}
	}
}
