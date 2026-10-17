package blinkengine

import (
	"errors"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

// TestLayoutMapsInvoiceOps runs the blinkless pipeline on a small invoice and
// checks the adapter output: canvas bounds from the options, text operations
// with an attached face, and the invoice text on the list.
func TestLayoutMapsInvoiceOps(t *testing.T) {
	t.Parallel()

	source := []byte(`<!DOCTYPE html><html><head><style>
h1 { color: #1a3d6d; font-size: 18pt }
table { width: 100%; border-collapse: collapse }
</style></head><body>
<h1>Invoice</h1>
<table><tbody><tr><td>Total</td><td>234.40 EUR</td></tr></tbody></table>
<div class="footer"><p>IBAN DE00 0000 0000 0000 0000 00</p></div>
</body></html>`)

	res, err := Layout(t.Context(), source, Options{
		WidthPt:  538.6,
		HeightPt: 785.2,
		Media:    "print",
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	if res.Width != 538.6 {
		t.Errorf("Width = %g, want 538.6", res.Width)
	}

	if res.Height <= 0 {
		t.Fatalf("Height = %g, want positive", res.Height)
	}

	if len(res.Ops) == 0 {
		t.Fatal("no operations mapped")
	}

	text, textOps := mappedText(t, res)
	if textOps == 0 {
		t.Fatal("no text operations mapped")
	}

	for _, needle := range []string{"Invoice", "234.40"} {
		if !strings.Contains(text, needle) {
			t.Errorf("mapped text %q missing %q", text, needle)
		}
	}
}

// TestLayoutMapsCornerShapePath proves the corner-shape bridge: a box with a
// non-round corner paints through blinkless as one fill path and one stroke
// path carrying the shaped outline in PathValue (the layout kinds this
// module's painters read), instead of being dropped as an unknown kind. The
// outline has more than a rectangle's four points and at least one point
// strictly inside the box, which only a shaped corner produces.
func TestLayoutMapsCornerShapePath(t *testing.T) {
	t.Parallel()

	source := []byte(`<!DOCTYPE html><html><head><style>
body { margin: 0 }
.shape-box { width: 132px; height: 64px; background: #e8eef8; border: 2px solid #1f4b99 }
</style></head><body>
<div class="shape-box" style="border-radius: 24px; corner-top-left-shape: bevel;"></div>
</body></html>`)

	res, err := Layout(t.Context(), source, Options{
		WidthPt:  538.6,
		HeightPt: 785.2,
		Media:    "print",
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	fills, strokes := mappedPathOps(res)

	if len(fills) != 1 || len(strokes) != 1 {
		t.Fatalf("path ops = %d fills, %d strokes, want one of each", len(fills), len(strokes))
	}

	path, _ := fills[0].PathValue()
	if !path.Closed {
		t.Error("fill path is not closed")
	}

	if len(path.Points) <= 4 {
		t.Fatalf("fill path points = %d, want more than a rectangle's four", len(path.Points))
	}

	if !pathHasInteriorPoint(path, &fills[0]) {
		t.Error("fill path has no point strictly inside the box; the shaped corner is missing")
	}
}

// mappedPathOps splits the mapped ops that carry path geometry into fills and
// strokes.
func mappedPathOps(res *layout.Result) ([]layout.Op, []layout.Op) {
	fills := make([]layout.Op, 0, 1)
	strokes := make([]layout.Op, 0, 1)

	for idx := range res.Ops {
		paintOp := &res.Ops[idx]

		if paintOp.Kind != layout.OpFillRect && paintOp.Kind != layout.OpStrokeRect {
			continue
		}

		path, ok := paintOp.PathValue()
		if !ok || len(path.Points) == 0 {
			continue
		}

		if paintOp.Kind == layout.OpStrokeRect {
			strokes = append(strokes, *paintOp)

			continue
		}

		fills = append(fills, *paintOp)
	}

	return fills, strokes
}

// pathHasInteriorPoint reports whether any path point sits strictly inside the
// op's box bounds (with a small tolerance). Points are local to the box, and a
// rectangle outline keeps every point on its border; a bevel, scoop, or notch
// dips inside.
func pathHasInteriorPoint(path layout.PathGeometry, paintOp *layout.Op) bool {
	const edgeTolerance = 0.01

	for _, point := range path.Points {
		insideX := point.X > edgeTolerance && point.X < paintOp.W-edgeTolerance
		insideY := point.Y > edgeTolerance && point.Y < paintOp.H-edgeTolerance

		if insideX || insideY {
			return true
		}
	}

	return false
}

// mappedText concatenates the mapped text runs and counts them, checking each
// carries a face.
func mappedText(t *testing.T, res *layout.Result) (string, int) {
	t.Helper()

	var text strings.Builder

	textOps := 0

	for idx := range res.Ops {
		textOp := &res.Ops[idx]
		if textOp.Kind != layout.OpText {
			continue
		}

		textOps++

		if textOp.Font == nil {
			t.Errorf("text op %d has no face", idx)
		}

		text.WriteString(textOp.Text)
	}

	return text.String(), textOps
}

func TestLayoutRejectsBadInput(t *testing.T) {
	t.Parallel()

	if _, err := Layout(nil, []byte("<p>x</p>"), Options{WidthPt: 500, HeightPt: 700}); !errors.Is(err, errNilContext) {
		t.Fatalf("nil context err = %v, want %v", err, errNilContext)
	}

	for _, opts := range []Options{
		{WidthPt: 0, HeightPt: 700},
		{WidthPt: 500, HeightPt: -1},
	} {
		if _, err := Layout(t.Context(), []byte("<p>x</p>"), opts); !errors.Is(err, errBadViewport) {
			t.Fatalf("viewport %+v err = %v, want %v", opts, err, errBadViewport)
		}
	}
}
