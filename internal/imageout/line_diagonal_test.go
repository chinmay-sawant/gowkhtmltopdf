package imageout

import (
	"image"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

// TestPaintLineDiagonalStrokesCenterline pins the checkbox-tick fix: a
// diagonal OpLine must paint its true centerline, not the vertical bar the
// axis-aligned fast path used to leave for both tick strokes.
func TestPaintLineDiagonalStrokesCenterline(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(0, 0, 24, 24))
	op := layout.Op{
		Kind: layout.OpLine,
		X:    4, Y: 4, W: 12, H: 12,
		Width: 2, R: 1, G: 0, B: 0,
	}
	paintLine(img, &op, layout.StyleOf(&op), 1)

	if pixel := img.NRGBAAt(10, 10); pixel.R == 0 {
		t.Fatalf("centerline midpoint not painted: %+v", pixel)
	}

	// Off the centerline in the old vertical-bar footprint; must stay clear.
	if pixel := img.NRGBAAt(4, 14); pixel.R != 0 {
		t.Fatalf("pixel off the centerline painted: %+v", pixel)
	}
}
