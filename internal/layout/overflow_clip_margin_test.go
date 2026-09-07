//nolint:testpackage,wsl,varnamelen,paralleltest,exhaustruct // overflow-clip-margin probes
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
			check: func(t *testing.T, parent *box, fill Op) {
				t.Helper()
				if fill.X > parent.x-6 {
					t.Fatalf("left fill X=%.2f parent.x=%.2f want left protrusion", fill.X, parent.x)
				}
			},
		},
		{
			name: "right", prop: "overflow-clip-margin-right",
			child: "right:-12pt;top:4pt;width:20pt;height:16pt",
			check: func(t *testing.T, parent *box, fill Op) {
				t.Helper()
				right := parent.x + parent.w
				if fill.X+fill.W < right+6 {
					t.Fatalf("right fill ends at %.2f parent.right=%.2f want right protrusion", fill.X+fill.W, right)
				}
			},
		},
		{
			name: "top", prop: "overflow-clip-margin-top",
			child: "left:6pt;top:-12pt;width:36pt;height:18pt",
			check: func(t *testing.T, parent *box, fill Op) {
				t.Helper()
				if fill.Y > parent.y-6 {
					t.Fatalf("top fill Y=%.2f parent.y=%.2f want top protrusion", fill.Y, parent.y)
				}
			},
		},
		{
			name: "bottom", prop: "overflow-clip-margin-bottom",
			child: "left:6pt;bottom:-12pt;width:36pt;height:18pt",
			check: func(t *testing.T, parent *box, fill Op) {
				t.Helper()
				bot := parent.y + parent.height
				if fill.Y+fill.H < bot+6 {
					t.Fatalf("bottom fill ends at %.2f parent.bottom=%.2f want bottom protrusion", fill.Y+fill.H, bot)
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			cssSheet := sheet(t, `
body { margin: 0 }
.parent { overflow: clip; `+tc.prop+`: 8pt; width: 48pt; height: 24pt; position: relative; border: 2pt solid #000; margin: 20pt }
.child { position: absolute; `+tc.child+`; background: #cc0000 }
`)
			res := layoutHTML(t, `<html><body><div class="parent"><div class="child">x</div></div></body></html>`, cssSheet)
			parent := findBoxByClass(t, res, "parent")

			var fills []Op
			for _, op := range res.Ops {
				if op.Kind == OpFillRect && op.R > 0.7 && op.G < 0.2 {
					fills = append(fills, op)
				}
			}
			if len(fills) == 0 {
				t.Fatal("missing red child fill")
			}
			tc.check(t, parent, fills[0])
		})
	}
}
