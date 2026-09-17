package layout

import (
	"testing"
)

// ANA-18: list-style-type:square must paint a vector square (OpFillRect) of
// 0.3125em, hanging ~0.66em outside the content edge, in the list color.
// Disc/circle stay as glyph OpBullet markers.

func TestSquareListMarkerGeometry(t *testing.T) { //nolint:cyclop // marker geometry gates are explicit
	t.Parallel()

	const fontPt = 12.0

	cssSheet := sheet(t, `
body { margin: 0; font-family: sans-serif; font-size: 12pt; color: #204080 }
ul { margin: 0; padding-left: 24pt; list-style-type: square }
`)

	res := layoutHTML(t, `<html><body><ul><li>See also</li></ul></body></html>`, cssSheet)

	li := findBox(t, res, "li")
	contentLeft := li.x

	if li.style != nil {
		contentLeft += li.style.PaddingLeft + li.style.BorderLeft.Width
	}

	wantSide := fontPt * 0.3125

	var marker *Op

	for i := range res.Ops {
		paintOp := &res.Ops[i]
		if paintOp.Kind != OpFillRect {
			continue
		}

		if near(paintOp.W, wantSide) && near(paintOp.H, wantSide) {
			marker = paintOp

			break
		}
	}

	if marker == nil {
		t.Fatalf("no %.3fpt square OpFillRect marker among %d ops", wantSide, len(res.Ops))
	}

	gutter := contentLeft - marker.X
	wantGutter := fontPt * 0.66

	if gutter < wantGutter-1 || gutter > wantGutter+1 {
		t.Fatalf("square marker gutter = %.3fpt (contentLeft=%.3f markerX=%.3f), want ~%.3fpt (0.66em)",
			gutter, contentLeft, marker.X, wantGutter)
	}

	if marker.R < 0.1 || marker.B < 0.4 {
		t.Fatalf("square marker color = (%.2f,%.2f,%.2f), want list color #204080",
			marker.R, marker.G, marker.B)
	}

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpBullet && paintOp.Text == "\u25AA" {
			t.Fatalf("square still emitted U+25AA OpBullet; want OpFillRect only")
		}
	}
}

func TestDiscCircleListMarkersStayGlyphs(t *testing.T) {
	t.Parallel()

	for _, typ := range []string{listStyleDisc, "circle"} {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()

			cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt }
ul { margin: 0; padding-left: 24pt; list-style-type: `+typ+` }
`)
			res := layoutHTML(t, `<html><body><ul><li>item</li></ul></body></html>`, cssSheet)
			bullets := opsOfKind(res, OpBullet)

			if len(bullets) != 1 {
				t.Fatalf("%s bullets = %d, want 1 OpBullet", typ, len(bullets))
			}
		})
	}
}
