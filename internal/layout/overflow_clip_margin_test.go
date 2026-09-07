//nolint:testpackage,wsl // overflow-clip-margin probes
package layout

import "testing"

// TestOverflowClipMarginDirectionalPaint proves overflow-clip-margin expands the
// clip edge so an abspos child fill can paint past the border on that side.
// Fixture-62 rows 27-37 rely on this for their Effect demos.
func TestOverflowClipMarginDirectionalPaint(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name, prop, child string
		check             func(t *testing.T, parent *box, fill Op)
	}{
		{
			name: "left", prop: "overflow-clip-margin-left",
			child: "left:-12pt;top:4pt;width:20pt;height:16pt",
			check: checkLeftClipProtrusion,
		},
		{
			name: "right", prop: "overflow-clip-margin-right",
			child: "right:-12pt;top:4pt;width:20pt;height:16pt",
			check: checkRightClipProtrusion,
		},
		{
			name: "top", prop: "overflow-clip-margin-top",
			child: "left:6pt;top:-12pt;width:36pt;height:18pt",
			check: checkTopClipProtrusion,
		},
		{
			name: "bottom", prop: "overflow-clip-margin-bottom",
			child: "left:6pt;bottom:-12pt;width:36pt;height:18pt",
			check: checkBottomClipProtrusion,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			runClipMarginCase(t, tc.prop, tc.child, tc.check)
		})
	}
}

func checkLeftClipProtrusion(t *testing.T, parent *box, fill Op) {
	t.Helper()
	if fill.X > parent.x-6 {
		t.Fatalf("left fill X=%.2f parent.x=%.2f want left protrusion", fill.X, parent.x)
	}
}

func checkRightClipProtrusion(t *testing.T, parent *box, fill Op) {
	t.Helper()
	right := parent.x + parent.w
	if fill.X+fill.W < right+6 {
		t.Fatalf("right fill ends at %.2f parent.right=%.2f want right protrusion", fill.X+fill.W, right)
	}
}

func checkTopClipProtrusion(t *testing.T, parent *box, fill Op) {
	t.Helper()
	if fill.Y > parent.y-6 {
		t.Fatalf("top fill Y=%.2f parent.y=%.2f want top protrusion", fill.Y, parent.y)
	}
}

func checkBottomClipProtrusion(t *testing.T, parent *box, fill Op) {
	t.Helper()
	bot := parent.y + parent.height
	if fill.Y+fill.H < bot+6 {
		t.Fatalf("bottom fill ends at %.2f parent.bottom=%.2f want bottom protrusion", fill.Y+fill.H, bot)
	}
}

func redChildFills(ops []Op) []Op {
	var fills []Op
	for _, op := range ops {
		if op.Kind == OpFillRect && op.R > 0.7 && op.G < 0.2 {
			fills = append(fills, op)
		}
	}

	return fills
}

func runClipMarginCase(t *testing.T, prop, child string, check func(t *testing.T, parent *box, fill Op)) {
	t.Helper()

	cssSheet := sheet(t, `
body { margin: 0 }
.parent { overflow: clip; `+prop+`: 8pt; width: 48pt; height: 24pt;
`+`position: relative; border: 2pt solid #000; margin: 20pt }
.child { position: absolute; `+child+`; background: #cc0000 }
`)
	res := layoutHTML(t, `<html><body><div class="parent"><div class="child">x</div></div></body></html>`, cssSheet)
	parent := findBoxByClass(t, res, "parent")

	fills := redChildFills(res.Ops)
	if len(fills) == 0 {
		t.Fatal("missing red child fill")
	}
	check(t, parent, fills[0])
}
