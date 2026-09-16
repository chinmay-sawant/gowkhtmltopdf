package layout

import "testing"

// tutorialspoint.com resets every element with border:0 solid, but the engine
// painted 1219 black 1pt strokes because parseBorder defaulted a parsed width
// of 0 back to 1 (real-sites evidence 2026-09-16, tutorialspoint-cpp row 1).
// border:0 solid, border:0px solid and border-width:0 must paint nothing.
func TestBorderZeroEmitsNoStrokes(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
		body { margin: 0 }
		.b0solid { border: 0 solid #000 }
		.b0px { border: 0px solid #000 }
		.b0width { border: 1pt solid #000; border-width: 0 }
		.b1 { border: 1pt solid #000 }
	`)
	res := layoutHTML(t, `<html><body>
		<div class="b0solid">a</div>
		<div class="b0px">b</div>
		<div class="b0width">c</div>
		<div class="b1">d</div>
	</body></html>`, cssSheet)

	for _, class := range []string{"b0solid", "b0px", "b0width"} {
		b := findBoxByClass(t, res, class)
		if b.style == nil {
			t.Fatalf("no resolved style for .%s", class)
		}

		if w := borderPaint(b.style.BorderTop); w != 0 {
			t.Fatalf(".%s BorderTop paint width = %.2f, want 0", class, w)
		}
	}

	// Only the positive control keeps its four 1pt edges.
	lines := opsOfKind(res, OpLine)
	if len(lines) != 4 {
		t.Fatalf("active border strokes = %d, want 4 (only .b1's 1pt frame)", len(lines))
	}
}
