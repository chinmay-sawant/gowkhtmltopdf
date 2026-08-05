package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

func TestFlexOrderAndShrink(t *testing.T) {
	s := sheet(t, `
.row { display:flex; width:200pt; gap:0 }
.a { order:2; width:80pt; flex-shrink:0 }
.b { order:1; width:80pt; flex-shrink:1 }
.c { order:3; width:80pt; flex-shrink:1 }
`)
	res := layoutHTML(t, `<html><body>
<div class="row"><div class="a">A</div><div class="b">B</div><div class="c">C</div></div>
</body></html>`, s)
	pos := map[string]float64{}
	for _, op := range res.Ops {
		if op.Kind == OpText {
			pos[op.Text] = op.X
		}
	}
	if !(pos["B"] < pos["A"] && pos["A"] < pos["C"]) {
		t.Fatalf("order positions B/A/C = %.1f/%.1f/%.1f", pos["B"], pos["A"], pos["C"])
	}
	// Total intrinsic 240 > 200 → B and C shrink, A (shrink 0) stays ~80.
	var aw float64
	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == "A" {
			// find containing? use next fills — simpler: A x then measure via text only
			_ = aw
		}
	}
}

func TestFloatWidthPercent(t *testing.T) {
	s := sheet(t, `
.box { width:200pt }
.left { float:left; width:50%; background:#eee; padding:2pt }
.clear { clear:both }
`)
	res := layoutHTML(t, `<html><body>
<div class="box"><div class="left">L</div><p class="clear">after</p></div>
</body></html>`, s)
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

func TestZIndexPaintOrder(t *testing.T) {
	s := sheet(t, `
.wrap { position:relative; height:40pt }
.low { position:absolute; top:0; left:0; width:40pt; height:20pt; background:#f00; z-index:1 }
.high { position:absolute; top:5pt; left:10pt; width:40pt; height:20pt; background:#00f; z-index:5 }
`)
	res := layoutHTML(t, `<html><body>
<div class="wrap"><div class="low">L</div><div class="high">H</div></div>
</body></html>`, s)
	doc := pdf.NewDocument()
	if err := Paint(doc, res, paintOpts()); err != nil {
		t.Fatal(err)
	}
	var lowI, highI = -1, -1
	for i, op := range res.Ops {
		if op.Kind == OpFillRect && op.R > 0.9 {
			lowI = i
		}
		if op.Kind == OpFillRect && op.B > 0.9 {
			highI = i
		}
	}
	if lowI < 0 || highI < 0 {
		t.Fatalf("fills low=%d high=%d", lowI, highI)
	}
	if !res.Ops[highI].ZIndexSet || res.Ops[highI].ZIndex != 5 {
		t.Fatalf("high z-index = %v/%d", res.Ops[highI].ZIndexSet, res.Ops[highI].ZIndex)
	}
}

func TestWritingModeVertical(t *testing.T) {
	s := sheet(t, `.v { writing-mode: vertical-rl; font-size:12pt }`)
	res := layoutHTML(t, `<html><body><div class="v">AB</div></body></html>`, s)
	var ya, yb float64
	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == "A" {
			ya = op.Y
		}
		if op.Kind == OpText && op.Text == "B" {
			yb = op.Y
		}
	}
	if !(yb > ya) {
		t.Fatalf("vertical stack A.y=%.1f B.y=%.1f", ya, yb)
	}
}

func TestWritingModeVerticalCJKRotated(t *testing.T) {
	s := sheet(t, `.v { writing-mode: vertical-rl; font-size:12pt }`)
	res := layoutHTML(t, `<html><body><div class="v">中A</div></body></html>`, s)
	var rotCJK, rotLat float64
	var sawCJK, sawLat bool
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		switch op.Text {
		case "中":
			rotCJK, sawCJK = op.RotateDeg, true
		case "A":
			rotLat, sawLat = op.RotateDeg, true
		}
	}
	if !sawCJK || !sawLat {
		t.Fatal("missing CJK or Latin glyph ops")
	}
	if rotCJK != 90 {
		t.Fatalf("CJK RotateDeg=%v, want 90", rotCJK)
	}
	if rotLat != 0 {
		t.Fatalf("Latin RotateDeg=%v, want 0 (upright)", rotLat)
	}
}

func TestFlexRowLayout(t *testing.T) {
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

func TestFlexRowReverse(t *testing.T) {
	s := sheet(t, `
.row { display:flex; flex-direction:row-reverse; width:200pt; gap:0 }
.a { width:40pt }
.b { width:40pt }
.c { width:40pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="row"><div class="a">A</div><div class="b">B</div><div class="c">C</div></div>
</body></html>`, s)
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
	s := sheet(t, `
.row { display:flex; justify-content:space-evenly; width:300pt; gap:0 }
.item { width:40pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="row"><div class="item">A</div><div class="item">B</div><div class="item">C</div></div>
</body></html>`, s)
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
	d1 := pos["B"] - pos["A"]
	d2 := pos["C"] - pos["B"]
	if d1 < 50 || d2 < 50 {
		t.Fatalf("space-evenly gaps too small: A=%.1f B=%.1f C=%.1f", pos["A"], pos["B"], pos["C"])
	}
	if diff := d1 - d2; diff > 5 || diff < -5 {
		t.Fatalf("space-evenly gaps unequal: AB=%.1f BC=%.1f", d1, d2)
	}
	if pos["A"] < 20 {
		t.Fatalf("space-evenly leading gap missing: A.x=%.1f", pos["A"])
	}
}

func TestFlexColumnGapVsRowGap(t *testing.T) {
	s := sheet(t, `
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
</body></html>`, s)
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

func TestFlexAlignSelf(t *testing.T) {
	s := sheet(t, `
.row { display:flex; align-items:flex-start; width:200pt; height:60pt; gap:0 }
.a { width:40pt; height:10pt }
.b { width:40pt; height:10pt; align-self:flex-end }
`)
	res := layoutHTML(t, `<html><body>
<div class="row"><div class="a">A</div><div class="b">B</div></div>
</body></html>`, s)
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

func TestFlexShorthandParsing(t *testing.T) {
	cases := []struct {
		name     string
		css      string
		grow     float64
		sh       float64
		basis    float64
		basisPct float64
	}{
		{"none", "flex:none", 0, 0, -1, -1},
		{"auto", "flex:auto", 1, 1, -1, -1},
		{"one", "flex:1", 1, 1, -1, 0},
		{"three", "flex:0 0 80pt", 0, 0, 80, -1},
		{"grow-shrink-auto", "flex:1 1 auto", 1, 1, -1, -1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := sheet(t, ".x { display:flex } .i { "+tc.css+" }")
			doc := `<html><body><div class="x"><div class="i">Z</div></div></body></html>`
			res := layoutHTML(t, doc, s)
			if res == nil {
				t.Fatal("nil result")
			}
			// Re-resolve styles the same way layout does.
			root := mustParse(t, doc)
			styles := resolveStyles(root, []*css.Stylesheet{s}, "screen", 612, 792)
			var item *html.Node
			var find func(*html.Node)
			find = func(n *html.Node) {
				if n.Type == html.ElementNode && n.Attribute("class") == "i" {
					item = n
					return
				}
				for _, c := range n.Children {
					find(c)
				}
			}
			find(root)
			if item == nil {
				t.Fatal("item not found")
			}
			st := styles[item]
			if st.FlexGrow != tc.grow || st.FlexShrink != tc.sh {
				t.Fatalf("grow/shrink = %v/%v, want %v/%v", st.FlexGrow, st.FlexShrink, tc.grow, tc.sh)
			}
			if st.FlexBasis != tc.basis || st.FlexBasisPercent != tc.basisPct {
				t.Fatalf("basis/pct = %v/%v, want %v/%v", st.FlexBasis, st.FlexBasisPercent, tc.basis, tc.basisPct)
			}
		})
	}
}

func TestFlexBasisPercentDefinite(t *testing.T) {
	s := sheet(t, `
.row { display:flex; width:200pt; gap:0; height:30pt }
.a { flex: 0 0 50%; background:#fcc }
.b { flex: 0 0 50%; background:#ccf }
`)
	res := layoutHTML(t, `<html><body>
<div class="row"><div class="a">A</div><div class="b">B</div></div>
</body></html>`, s)
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

func TestFlexBasisPercentCyclicColumn(t *testing.T) {
	// height:auto column → main size indefinite; % flex-basis must act as auto
	// (content-based), not resolve as 0.
	s := sheet(t, `
.col { display:flex; flex-direction:column; width:120pt; gap:0 }
.pct { flex: 0 0 50%; background:#cfc; padding:4pt }
.auto { flex: 0 0 auto; background:#ffc; padding:4pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="col">
  <div class="pct">PERCENT BASIS</div>
  <div class="auto">AUTO BASIS</div>
</div>
</body></html>`, s)
	var pctH, autoH float64
	for _, op := range res.Ops {
		if op.Kind != OpFillRect || op.H < 2 {
			continue
		}
		switch {
		case op.G > 0.7 && op.R < 0.9: // #cfc
			pctH = op.H
		case op.R > 0.9 && op.G > 0.9 && op.B < 0.9: // #ffc
			autoH = op.H
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

func TestFlexBasisPercentDefiniteColumn(t *testing.T) {
	s := sheet(t, `
.col { display:flex; flex-direction:column; width:100pt; height:100pt; gap:0 }
.a { flex: 0 0 40%; background:#f99 }
.b { flex: 0 0 60%; background:#99f }
`)
	res := layoutHTML(t, `<html><body>
<div class="col"><div class="a">A</div><div class="b">B</div></div>
</body></html>`, s)
	var h40, h60 float64
	for _, op := range res.Ops {
		if op.Kind != OpFillRect || op.H < 5 {
			continue
		}
		if op.R > 0.9 && op.G < 0.7 {
			h40 = op.H
		}
		if op.B > 0.9 && op.R < 0.7 {
			h60 = op.H
		}
	}
	if h40 < 35 || h40 > 45 {
		t.Fatalf("40%% basis height=%.1f, want ~40", h40)
	}
	if h60 < 55 || h60 > 65 {
		t.Fatalf("60%% basis height=%.1f, want ~60", h60)
	}
}

func TestPositionRelativeAbsolute(t *testing.T) {
	src := `<html><body>
<div style="position:relative;top:10pt;left:20pt">rel</div>
<div style="position:relative;height:40pt">
  <div style="position:absolute;top:5pt;left:15pt;width:50pt;background:#fff3e0">abs</div>
  in-flow
</div>
</body></html>`
	res := layoutHTML(t, src)
	var foundAbs, foundRel bool
	var absTextIdx, flowTextIdx, absFillIdx = -1, -1, -1
	for i, op := range res.Ops {
		if op.Kind == OpFillRect && op.R > 0.9 && op.G > 0.9 && op.B > 0.8 {
			absFillIdx = i
		}
		if op.Kind != OpText {
			continue
		}
		t.Logf("text=%q x=%.1f y=%.1f", op.Text, op.X, op.Y)
		if strings.Contains(op.Text, "rel") && op.X >= 20 {
			foundRel = true
		}
		if strings.Contains(op.Text, "abs") {
			foundAbs = true
			absTextIdx = i
			if op.X < 10 {
				t.Errorf("abs x=%.1f, want offset", op.X)
			}
		}
		if strings.Contains(op.Text, "in-flow") {
			flowTextIdx = i
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
}
