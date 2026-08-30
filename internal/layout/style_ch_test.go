//nolint:testpackage // ch unit layout probe
package layout

import (
	"math"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

func TestChUsesZeroGlyphAdvance(t *testing.T) {
	t.Parallel()

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatal(err)
	}

	fontSize := float64(defaultFontSizePt)
	zeroAdv := faces.Regular.GlyphAdvancePoints(digitZero, fontSize)
	want := 10 * zeroAdv
	halfEm := 10 * halfRatio * fontSize

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
.ch { width: 10ch; }
`)
	res := layoutHTML(t, `<html><body><div class="ch">x</div></body></html>`, cssSheet)
	box := findBoxByClass(t, res, "ch")

	if math.Abs(box.w-want) > 0.5 {
		t.Fatalf("width:10ch box.w=%.3f, want ~%.3f (10 * zero advance), not 0.5em (%.3f)",
			box.w, want, halfEm)
	}

	if math.Abs(box.w-halfEm) < 0.5 {
		t.Fatalf("width:10ch resolved as 0.5em (%.3f); want glyph advance %.3f", halfEm, want)
	}

	if box.style != nil && math.Abs(box.style.Width-want) > 0.5 {
		t.Fatalf("style.Width=%.3f, want ~%.3f", box.style.Width, want)
	}
}
