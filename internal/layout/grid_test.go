package layout

import "testing"

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
