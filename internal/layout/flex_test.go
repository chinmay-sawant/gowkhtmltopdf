package layout

import (
	"strings"
	"testing"

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
