package layout

import (
	"strings"
	"testing"
)

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
