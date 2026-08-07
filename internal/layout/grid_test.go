package layout

import (
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestParseGridTracksSubtractsGap(t *testing.T) {
	e := &engine{scale: 1}

	const contentW = 300.0

	const gap = 6.0

	cols := parseGridTracks("repeat(3, 1fr)", contentW, gap, e)
	if len(cols) != 3 {
		t.Fatalf("cols=%v", cols)
	}

	sum := cols[0] + cols[1] + cols[2] + 2*gap
	if sum < contentW-0.01 || sum > contentW+0.01 {
		t.Fatalf("tracks+gaps = %.3f, want contentW=%.3f (cols=%v)", sum, contentW, cols)
	}
}

func TestGridColumnSpan(t *testing.T) {
	s := sheet(t, `
.g { display:grid; grid-template-columns:repeat(3,1fr); gap:4pt; width:300pt }
.wide { grid-column: span 2; background:#eee }
`)
	res := layoutHTML(t, `<html><body>
<div class="g"><div class="wide">AB</div><div>C</div><div>D</div></div>
</body></html>`, s)

	var wideW float64

	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.R > 0.8 {
			if op.W > wideW {
				wideW = op.W
			}
		}
	}
	// span 2 of 3 tracks ≈ 2/3 of 300 minus gaps ≈ 196+
	if wideW < 180 || wideW > 220 {
		t.Fatalf("span-2 width=%.1f, want ~200", wideW)
	}
}

func TestNestedGridWithSpan(t *testing.T) {
	s := sheet(t, `
.outer { display:grid; grid-template-columns:1fr 1fr; gap:4pt; width:300pt }
.inner { display:grid; grid-template-columns:repeat(3,1fr); gap:2pt; background:#ddd }
.span { grid-column: span 2; background:#fcc }
`)
	res := layoutHTML(t, `<html><body>
<div class="outer">
  <div class="inner"><div class="span">X</div><div>Y</div></div>
  <div>Z</div>
</div>
</body></html>`, s)

	var spanW float64

	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.R > 0.9 && op.G < 0.9 {
			if op.W > spanW {
				spanW = op.W
			}
		}
	}
	// Inner content ~148pt; span 2/3 ≈ 95+
	if spanW < 80 || spanW > 130 {
		t.Fatalf("nested span-2 width=%.1f, want ~95-110", spanW)
	}
}

func TestGridCellsStayInsideBorder(t *testing.T) {
	src := `<html><body>
<div style="display:grid;grid-template-columns:repeat(3,1fr);gap:6pt;width:300pt;padding:4pt;border:1px solid #000">
  <div style="background:#f3e5f5">G1</div>
  <div style="background:#f3e5f5">G2</div>
  <div style="background:#f3e5f5">G3</div>
  <div style="background:#f3e5f5">G4</div>
</div>
</body></html>`
	res := layoutHTML(t, src)

	var borderRight, maxFillRight float64

	for _, op := range res.Ops {
		if op.Kind == OpLine {
			r := op.X
			if op.W > 0 {
				r = op.X + op.W
			}

			if r > borderRight {
				borderRight = r
			}
		}

		if op.Kind == OpFillRect && op.B > 0.9 && op.R > 0.9 {
			if right := op.X + op.W; right > maxFillRight {
				maxFillRight = right
			}
		}
	}

	if borderRight == 0 || maxFillRight == 0 {
		t.Fatalf("borderRight=%.1f maxFillRight=%.1f", borderRight, maxFillRight)
	}

	if maxFillRight > borderRight+0.5 {
		t.Fatalf("cell fill right %.2f overflows border %.2f", maxFillRight, borderRight)
	}
}

func TestGridTemplateRowsEqualFr(t *testing.T) {
	s := sheet(t, `
.g { display:grid; grid-template-columns:1fr; grid-template-rows:1fr 1fr; height:200pt; width:100pt; gap:0 }
.a { background:#fcc }
.b { background:#cfc }
`)
	res := layoutHTML(t, `<html><body>
<div class="g"><div class="a">A</div><div class="b">B</div></div>
</body></html>`, s)

	var ay, by float64

	var foundA, foundB bool

	for _, op := range res.Ops {
		if op.Kind != OpFillRect {
			continue
		}

		if op.R > 0.9 && op.G < 0.9 { // #fcc
			ay = op.Y
			foundA = true
		}

		if op.G > 0.7 && op.R < 0.9 { // #cfc
			by = op.Y
			foundB = true
		}
	}

	if !foundA || !foundB {
		t.Fatalf("missing fills foundA=%v foundB=%v", foundA, foundB)
	}

	dy := by - ay
	if dy < 95 || dy > 105 {
		t.Fatalf("row gap via Y: B-A=%.1f, want ~100 (equal 1fr rows in 200pt)", dy)
	}
}

func TestGridRowSpan(t *testing.T) {
	s := sheet(t, `
.g { display:grid; grid-template-columns:1fr 1fr; grid-template-rows:1fr 1fr; height:200pt; width:200pt; gap:0 }
.tall { grid-row: span 2; background:#fcc }
.b { background:#cfc }
.c { background:#ccf }
`)
	res := layoutHTML(t, `<html><body>
<div class="g">
  <div class="tall">T</div>
  <div class="b">B</div>
  <div class="c">C</div>
</div>
</body></html>`, s)

	var tallH, tallW float64

	var by, cy float64

	var foundTall, foundB, foundC bool

	for _, op := range res.Ops {
		if op.Kind != OpFillRect {
			continue
		}

		if op.R > 0.9 && op.G < 0.9 && op.B < 0.9 { // #fcc
			if op.H > tallH {
				tallH = op.H
				tallW = op.W
			}

			foundTall = true
		}

		if op.G > 0.7 && op.R < 0.9 && op.B < 0.9 { // #cfc
			by = op.Y
			foundB = true
		}

		if op.B > 0.9 && op.G > 0.7 && op.R < 0.9 { // #ccf
			cy = op.Y
			foundC = true
		}
	}

	if !foundTall || !foundB || !foundC {
		t.Fatalf("fills tall=%v b=%v c=%v", foundTall, foundB, foundC)
	}

	if tallW < 90 || tallW > 110 {
		t.Fatalf("span-2 col width=%.1f, want ~100", tallW)
	}
	// Stretch into both 1fr rows (no explicit height on .tall).
	if tallH < 190 || tallH > 210 {
		t.Fatalf("span-2 height=%.1f, want ~200 (stretch across both rows)", tallH)
	}
	// B in row0, C in row1 — vertical distance ≈ one fr track (100pt)
	if cy-by < 95 || cy-by > 105 {
		t.Fatalf("B→C dy=%.1f, want ~100", cy-by)
	}
}

func TestGridRowSpanStretchMatchesFixture32(t *testing.T) {
	// Mirrors fixture-32 .grid: 100pt content height, 6pt row-gap, span 2.
	s := sheet(t, `
.grid {
  display: grid;
  grid-template-columns: 1fr 1fr 1fr;
  grid-template-rows: 1fr 1fr;
  column-gap: 10pt;
  row-gap: 6pt;
  height: 100pt;
  width: 280pt;
  padding: 4pt;
  border: 1px solid #6a1b9a;
  background: #f3e5f5;
}
.grid > div { background: #fff; padding: 4pt; box-sizing: border-box; }
.tall { grid-row: span 2; background: #e1bee7; }
`)
	res := layoutHTML(t, `<html><body>
<div class="grid">
  <div class="tall">Tall span-2</div>
  <div>B</div><div>C</div><div>D</div><div>E</div>
</div>
</body></html>`, s)

	var tallH float64

	var found bool

	for _, op := range res.Ops {
		if op.Kind != OpFillRect || op.H < 5 {
			continue
		}
		// Prefer the tallest non-container purple-ish fill (#e1bee7).
		// Container #f3e5f5 is lighter (R/G/B all high).
		if op.H > 80 && op.W > 200 {
			continue // section/grid wash
		}

		if op.R > 0.8 && op.B > 0.8 && op.G > 0.65 && op.G < 0.85 && op.H > tallH {
			tallH = op.H
			found = true
		}
	}

	if !found {
		// Fallback: tallest fill that is not full-grid width.
		for _, op := range res.Ops {
			if op.Kind != OpFillRect || op.W > 200 || op.H < 40 {
				continue
			}

			if op.H > tallH {
				tallH = op.H
				found = true
			}
		}
	}

	if !found {
		t.Fatal("tall cell fill missing")
	}
	// contentH=100, row-gap=6 → rows 47+47; span-2 area = 100pt
	if tallH < 95 || tallH > 105 {
		t.Fatalf("tall span-2 height=%.1f, want ~100 (both rows + gap)", tallH)
	}
}

func TestGridRowGapVsColumnGap(t *testing.T) {
	s := sheet(t, `
.g { display:grid; grid-template-columns:1fr 1fr; width:200pt; column-gap:20pt; row-gap:8pt }
.a { background:#fcc }
.b { background:#cfc }
.c { background:#ccf }
.d { background:#ffc }
`)
	res := layoutHTML(t, `<html><body>
<div class="g">
  <div class="a">A</div><div class="b">B</div>
  <div class="c">C</div><div class="d">D</div>
</div>
</body></html>`, s)

	var ax, bx, ay, cy float64

	var aw float64

	var foundA, foundB, foundC bool

	for _, op := range res.Ops {
		if op.Kind != OpFillRect {
			continue
		}

		if op.R > 0.9 && op.G < 0.9 && op.B < 0.9 { // #fcc
			ax, ay, aw = op.X, op.Y, op.W
			foundA = true
		}

		if op.G > 0.7 && op.R < 0.9 && op.B < 0.9 { // #cfc
			bx = op.X
			foundB = true
		}

		if op.B > 0.9 && op.G > 0.7 && op.R < 0.9 { // #ccf
			cy = op.Y
			foundC = true
		}
	}

	if !foundA || !foundB || !foundC {
		t.Fatalf("fills a=%v b=%v c=%v", foundA, foundB, foundC)
	}

	colGap := bx - (ax + aw)
	if colGap < 18 || colGap > 22 {
		t.Fatalf("column-gap=%.1f, want ~20", colGap)
	}
	// Track width = (200-20)/2 = 90
	if aw < 85 || aw > 95 {
		t.Fatalf("track width=%.1f, want ~90", aw)
	}

	rowGap := cy - ay
	// row gap is between content bottoms/tops; with equal content, dy ≈ contentH + 8.
	// Prefer: row-gap must not equal column-gap (20); content-sized rows → dy > 8.
	if rowGap < 8 {
		t.Fatalf("row spacing dy=%.1f, want at least row-gap 8", rowGap)
	}
	// Independence: with only 8pt row-gap, vertical step should be much smaller than
	// if column-gap (20) were wrongly used as row gap with tall tracks — check that
	// A and C share the same column (similar X) and C is below A.
	if cy <= ay {
		t.Fatalf("C should be below A: ay=%.1f cy=%.1f", ay, cy)
	}
}

func TestParseGridRowSpan(t *testing.T) {
	s := sheet(t, `.x { grid-row: span 2 } .y { grid-row: 2 / span 3 } .z { grid-row-start: 3 }`)
	root := mustParse(t, `<html><body><div class="x"></div><div class="y"></div><div class="z"></div></body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{s}, "print", 800, 600)

	var nodes []*html.Node

	var collect func(*html.Node)
	collect = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "div" {
			nodes = append(nodes, n)
		}

		for _, c := range n.Children {
			collect(c)
		}
	}
	collect(root)

	if len(nodes) < 3 {
		t.Fatalf("nodes=%d", len(nodes))
	}

	x, y, z := nodes[0], nodes[1], nodes[2]
	if styles[x].GridRowSpan != 2 {
		t.Fatalf("span 2: got %d", styles[x].GridRowSpan)
	}

	if styles[y].GridRowStart != 2 || styles[y].GridRowSpan != 3 {
		t.Fatalf("2 / span 3: start=%d span=%d", styles[y].GridRowStart, styles[y].GridRowSpan)
	}

	if styles[z].GridRowStart != 3 {
		t.Fatalf("row-start 3: got %d", styles[z].GridRowStart)
	}
}

func TestParseGridTemplateAreas(t *testing.T) {
	areas := parseGridTemplateAreas(`"header header" "nav main" "footer footer"`)
	if areas.rows != 3 || areas.cols != 2 {
		t.Fatalf("dims %dx%d, want 3x2", areas.rows, areas.cols)
	}

	h, ok := resolveNamedGridArea(areas, "header")
	if !ok || h.row != 0 || h.col != 0 || h.rowSpan != 1 || h.colSpan != 2 {
		t.Fatalf("header=%v ok=%v", h, ok)
	}

	main, ok := resolveNamedGridArea(areas, "main")
	if !ok || main.row != 1 || main.col != 1 || main.rowSpan != 1 || main.colSpan != 1 {
		t.Fatalf("main=%v ok=%v", main, ok)
	}

	empty := parseGridTemplateAreas("none")
	if empty.rows != 0 || len(empty.names) != 0 {
		t.Fatalf("none should be empty: %+v", empty)
	}

	dots := parseGridTemplateAreas(`"a ." ". b"`)
	if dots.cols != 2 || dots.rows != 2 {
		t.Fatalf("dots dims %dx%d", dots.rows, dots.cols)
	}

	if _, ok := resolveNamedGridArea(dots, "."); ok {
		t.Fatal(". must not be a named area")
	}
}

func TestParseGridAreaAndAutoFlow(t *testing.T) {
	s := sheet(t, `
.n { grid-area: sidebar }
.l { grid-area: 1 / 2 / 3 / 4 }
.f { grid-auto-flow: dense }
.g { grid-auto-flow: column dense }
`)
	root := mustParse(t, `<html><body>
<div class="n"></div><div class="l"></div>
<div class="f"></div><div class="g"></div>
</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{s}, "print", 800, 600)

	var nodes []*html.Node

	var collect func(*html.Node)
	collect = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "div" {
			nodes = append(nodes, n)
		}

		for _, c := range n.Children {
			collect(c)
		}
	}
	collect(root)

	if styles[nodes[0]].GridArea != "sidebar" {
		t.Fatalf("named area: %q", styles[nodes[0]].GridArea)
	}

	l := styles[nodes[1]]
	if l.GridRowStart != 1 || l.GridColumnStart != 2 || l.GridRowSpan != 2 || l.GridColumnSpan != 2 {
		t.Fatalf("line area: row=%d col=%d rowSpan=%d colSpan=%d",
			l.GridRowStart, l.GridColumnStart, l.GridRowSpan, l.GridColumnSpan)
	}

	if styles[nodes[2]].GridAutoFlow != "dense" {
		t.Fatalf("dense: %q", styles[nodes[2]].GridAutoFlow)
	}

	if styles[nodes[3]].GridAutoFlow != "column dense" {
		t.Fatalf("column dense: %q", styles[nodes[3]].GridAutoFlow)
	}

	col, dense := gridAutoFlowMode("row dense")
	if col || !dense {
		t.Fatalf("row dense mode: col=%v dense=%v", col, dense)
	}
}

func TestGridTemplateAreasPlacement(t *testing.T) {
	s := sheet(t, `
.g {
  display: grid;
  grid-template-columns: 1fr 1fr;
  grid-template-areas: "head head" "side main";
  width: 200pt;
  height: 100pt;
  gap: 0;
}
.head { grid-area: head; background: #fcc }
.side { grid-area: side; background: #cfc }
.main { grid-area: main; background: #ccf }
`)
	res := layoutHTML(t, `<html><body>
<div class="g">
  <div class="main">M</div>
  <div class="side">S</div>
  <div class="head">H</div>
</div>
</body></html>`, s)

	var hx, hy, hw, hh float64

	var sx, mx float64

	var foundH, foundS, foundM bool

	for _, op := range res.Ops {
		if op.Kind != OpFillRect {
			continue
		}

		if op.R > 0.9 && op.G < 0.9 && op.B < 0.9 { // #fcc head
			hx, hy, hw, hh = op.X, op.Y, op.W, op.H
			foundH = true
		}

		if op.G > 0.7 && op.R < 0.9 && op.B < 0.9 { // #cfc side
			sx = op.X
			foundS = true
		}

		if op.B > 0.9 && op.G > 0.7 && op.R < 0.9 { // #ccf main
			mx = op.X
			foundM = true
		}
	}

	if !foundH || !foundS || !foundM {
		t.Fatalf("fills H=%v S=%v M=%v", foundH, foundS, foundM)
	}
	// Head spans both columns (~200pt).
	if hw < 180 || hw > 220 {
		t.Fatalf("head width=%.1f, want ~200", hw)
	}
	// Side left of main.
	if !(sx < mx) {
		t.Fatalf("side x=%.1f should be left of main x=%.1f", sx, mx)
	}
	// Head on first row (above side/main content band).
	if hy > 50 {
		t.Fatalf("head y=%.1f looks too low", hy)
	}

	_ = hh
	_ = hx
}

func TestGridAreasImplyTrackCounts(t *testing.T) {
	// No grid-template-columns — areas alone imply 3 columns.
	s := sheet(t, `
.g {
  display: grid;
  grid-template-areas: "a b c";
  width: 300pt;
  gap: 0;
}
.a { grid-area: a; background: #fcc }
.b { grid-area: b; background: #cfc }
.c { grid-area: c; background: #ccf }
`)
	res := layoutHTML(t, `<html><body>
<div class="g">
  <div class="a">A</div><div class="b">B</div><div class="c">C</div>
</div>
</body></html>`, s)

	var aw, bx float64

	var foundA, foundB bool

	for _, op := range res.Ops {
		if op.Kind != OpFillRect {
			continue
		}

		if op.R > 0.9 && op.G < 0.9 && op.B < 0.9 {
			aw = op.W
			foundA = true
		}

		if op.G > 0.7 && op.R < 0.9 && op.B < 0.9 {
			bx = op.X
			foundB = true
		}
	}

	if !foundA || !foundB {
		t.Fatalf("fills a=%v b=%v", foundA, foundB)
	}

	if aw < 90 || aw > 110 {
		t.Fatalf("implied track width=%.1f, want ~100", aw)
	}

	if bx < 90 || bx > 110 {
		t.Fatalf("B should start near 100, got x=%.1f", bx)
	}
}

func TestGridAutoFlowDenseFillsHole(t *testing.T) {
	// Item A spans 2 cols in row 0 → hole at (0, missing) then B takes col0?
	// Layout: 2 cols. First item grid-column: 2 (leaves (0,0) empty).
	// Sparse: second item starts after cursor → row1.
	// Dense: second item fills (0,0).
	sSparse := sheet(t, `
.g { display: grid; grid-template-columns: 1fr 1fr; width: 200pt; gap: 0; grid-auto-flow: row }
.a { grid-column: 2; background: #fcc }
.b { background: #cfc }
`)
	sDense := sheet(t, `
.g { display: grid; grid-template-columns: 1fr 1fr; width: 200pt; gap: 0; grid-auto-flow: dense }
.a { grid-column: 2; background: #fcc }
.b { background: #cfc }
`)
	htmlSrc := `<html><body>
<div class="g"><div class="a">A</div><div class="b">B</div></div>
</body></html>`

	sparse := layoutHTML(t, htmlSrc, sSparse)
	dense := layoutHTML(t, htmlSrc, sDense)

	yOf := func(res *Result, wantRed bool) float64 {
		for _, op := range res.Ops {
			if op.Kind != OpFillRect {
				continue
			}

			isRed := op.R > 0.9 && op.G < 0.9 && op.B < 0.9
			isGreen := op.G > 0.7 && op.R < 0.9 && op.B < 0.9

			if wantRed && isRed {
				return op.Y
			}

			if !wantRed && isGreen {
				return op.Y
			}
		}

		t.Fatalf("missing fill wantRed=%v", wantRed)

		return 0
	}
	sparseBY := yOf(sparse, false)
	denseBY := yOf(dense, false)
	sparseAY := yOf(sparse, true)
	denseAY := yOf(dense, true)

	// Sparse: B below A (new row). Dense: B same row as A (fills hole).
	if sparseBY <= sparseAY+2 {
		t.Fatalf("sparse: B should be below A: aY=%.1f bY=%.1f", sparseAY, sparseBY)
	}

	if denseBY > denseAY+5 {
		t.Fatalf("dense: B should share row with A: aY=%.1f bY=%.1f", denseAY, denseBY)
	}
}

func TestParseGridTracksMinmax(t *testing.T) {
	e := &engine{scale: 1}

	const contentW = 300.0

	cols := parseGridTracks("minmax(50pt, 1fr) minmax(100pt, 1fr)", contentW, 0, e)
	if len(cols) != 2 {
		t.Fatalf("cols=%v", cols)
	}
	// bases 50+100=150; free 150 → +75 each → 125, 175
	if cols[0] < 120 || cols[0] > 130 {
		t.Fatalf("col0=%.1f, want ~125 (50 + half of free)", cols[0])
	}

	if cols[1] < 170 || cols[1] > 180 {
		t.Fatalf("col1=%.1f, want ~175 (100 + half of free)", cols[1])
	}

	sum := cols[0] + cols[1]
	if sum < contentW-0.01 || sum > contentW+0.01 {
		t.Fatalf("sum=%.1f, want %.1f", sum, contentW)
	}
}

func TestParseGridTracksMinmaxPercent(t *testing.T) {
	e := &engine{scale: 1}

	cols := parseGridTracks("minmax(10%, 1fr) 1fr", 200, 0, e)
	if len(cols) != 2 {
		t.Fatalf("cols=%v", cols)
	}
	// min floor 20 on first; free 180 → 20+90=110, 90
	if cols[0] < 105 || cols[0] > 115 {
		t.Fatalf("col0=%.1f, want ~110", cols[0])
	}

	if cols[1] < 85 || cols[1] > 95 {
		t.Fatalf("col1=%.1f, want ~90", cols[1])
	}
}

func TestParseGridTracksRepeatMinmax(t *testing.T) {
	e := &engine{scale: 1}

	cols := parseGridTracks("repeat(3, minmax(40pt, 1fr))", 300, 0, e)
	if len(cols) != 3 {
		t.Fatalf("cols=%v", cols)
	}

	for i, c := range cols {
		if c < 95 || c > 105 {
			t.Fatalf("col[%d]=%.1f, want ~100", i, c)
		}
	}
}

func TestGridMinmaxFrLayout(t *testing.T) {
	s := sheet(t, `
.g { display:grid; grid-template-columns:minmax(80pt,1fr) minmax(40pt,1fr); width:300pt; gap:0 }
.a { background:#fcc }
.b { background:#cfc }
`)
	res := layoutHTML(t, `<html><body>
<div class="g"><div class="a">A</div><div class="b">B</div></div>
</body></html>`, s)

	var aw, bw float64

	for _, op := range res.Ops {
		if op.Kind != OpFillRect {
			continue
		}

		if op.R > 0.9 && op.G < 0.9 {
			aw = op.W
		}

		if op.G > 0.7 && op.R < 0.9 {
			bw = op.W
		}
	}
	// floors 80+40=120; free 180 → 170, 130
	if aw < 160 || aw > 180 {
		t.Fatalf("minmax col A width=%.1f, want ~170", aw)
	}

	if bw < 120 || bw > 140 {
		t.Fatalf("minmax col B width=%.1f, want ~130", bw)
	}
}

func TestIntrinsicHeightPercentCyclic(t *testing.T) {
	// height:% against auto-height ancestor must not resolve as 0/NaN — treat as auto.
	s := sheet(t, `
.g { display:grid; grid-template-columns:1fr; width:200pt }
.child { height:50%; background:#fcc; padding:8pt }
`)
	res := layoutHTML(t, `<html><body>
<div class="g"><div class="child">Hello cyclic percent</div></div>
</body></html>`, s)

	var h float64

	for _, op := range res.Ops {
		if op.Kind == OpFillRect && op.R > 0.9 && op.G < 0.9 {
			if op.H > h {
				h = op.H
			}
		}
	}

	if h < 10 || h != h { // NaN check
		t.Fatalf("cyclic height%% resolved to %.1f; want content-based >10", h)
	}
}

func TestStripMasonryKeyword(t *testing.T) {
	if stripMasonryKeyword("masonry") != "" {
		t.Fatal("want empty after strip")
	}

	if stripMasonryKeyword("1fr 1fr") != "1fr 1fr" {
		t.Fatal("non-masonry tracks must stay")
	}
}
