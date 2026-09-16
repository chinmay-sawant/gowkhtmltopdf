package layout

import (
	"strings"
	"testing"
)

// Programiz accordion: .accordion-body{height:0;overflow:hidden} rendered
// expanded and added 7 pages (real-sites evidence 2026-09-16, programiz-cpp
// row 2). The box must collapse to used height 0 and emit no ink.
func TestHeightZeroOverflowHiddenCollapses(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
		body { margin: 0; font: 12px sans-serif }
		.accordion { height: 0; overflow: hidden }
		.accordion .body { height: 40pt; background: #ff0000 }
	`)
	res := layoutHTML(t, `<html><body>
		<div class="accordion"><div class="body">ACCORDION-BODY-TEXT</div></div>
		<div class="after">AFTER-ACCORDION</div>
	</body></html>`, cssSheet)

	accordion := findBoxByClass(t, res, "accordion")
	if !near(accordion.height, 0) {
		t.Fatalf("height:0 overflow:hidden box height = %.2f, want 0", accordion.height)
	}

	if got := joinedText(res); strings.Contains(got, "ACCORDION-BODY-TEXT") {
		t.Fatalf("collapsed accordion painted child text: %q", got)
	}

	for i := accordion.opStart; i <= accordion.opEnd && i < len(res.Ops); i++ {
		if res.Ops[i].Kind != opKindNoop {
			t.Fatalf("collapsed accordion op %d still paints: %+v", i, res.Ops[i])
		}
	}

	afterY := textY(t, res, "AFTER-ACCORDION")
	if afterY > 20 {
		t.Fatalf("AFTER-ACCORDION y = %.2f, want near the top after a collapsed accordion", afterY)
	}
}
