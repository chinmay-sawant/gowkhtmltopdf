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
	cols2 := parseGridTracks("1fr 1fr 1fr", contentW, gap, e)
	sum2 := cols2[0] + cols2[1] + cols2[2] + 2*gap
	if sum2 < contentW-0.01 || sum2 > contentW+0.01 {
		t.Fatalf("fr list tracks+gaps = %.3f, want %.3f", sum2, contentW)
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
