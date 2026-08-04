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
  <div style="position:absolute;top:5pt;left:15pt;width:50pt">abs</div>
</div>
</body></html>`
	res := layoutHTML(t, src)
	var foundAbs, foundRel bool
	for _, op := range res.Ops {
		if op.Kind != OpText {
			continue
		}
		t.Logf("text=%q x=%.1f y=%.1f", op.Text, op.X, op.Y)
		if strings.Contains(op.Text, "rel") && op.X >= 20 {
			foundRel = true
		}
		if strings.Contains(op.Text, "abs") {
			foundAbs = true
			if op.X < 10 {
				t.Errorf("abs x=%.1f, want offset", op.X)
			}
		}
	}
	if !foundRel {
		t.Error("relative offset text not found")
	}
	if !foundAbs {
		t.Error("absolute text not found")
	}
}
