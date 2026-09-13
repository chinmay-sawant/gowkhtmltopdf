package imageout

import (
	"bytes"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// TestRasterGlyphAlphaReusesScratch proves that one scratch can serve glyphs
// with different edge counts: a second, larger glyph must not leave stale
// edges or active rows behind, so the coverage bytes for the first glyph are
// reproduced exactly on the third call.
func TestRasterGlyphAlphaReusesScratch(t *testing.T) {
	t.Parallel()

	scratch := &glyphScratch{}
	scratch.edges = append(scratch.edges, glyphEdge{yMin: -20, yMax: 20, xAtMin: 0, dxdy: 1})
	scratch.activeRows[0] = append(scratch.activeRows[0], glyphEdge{yMin: -20, yMax: 20, xAtMin: 0, dxdy: 1})

	// A 6x8 rectangle from y -12 to 2, sampled with a 4x4 grid.
	square := []glyphEdge{
		{yMin: -12, yMax: 2, xAtMin: 2, dxdy: 0},
		{yMin: -12, yMax: 2, xAtMin: 8, dxdy: 0},
	}

	first := rasterGlyphAlpha(scratch, square, 12, 12, 4, 16, 0, 0, 1)

	// A much larger edge list grows the scratch's slices and dirty active rows.
	big := make([]glyphEdge, 0, 64)
	for i := range 32 {
		big = append(big, glyphEdge{
			yMin: float64(i), yMax: float64(i) + 2,
			xAtMin: float64(i), dxdy: 0.5,
		})
	}

	_ = rasterGlyphAlpha(scratch, big, 40, 40, 4, 16, 0, 0, 1)

	again := rasterGlyphAlpha(scratch, square, 12, 12, 4, 16, 0, 0, 1)

	if first.Bounds() != again.Bounds() {
		t.Fatalf("bounds changed after scratch reuse: %v vs %v", first.Bounds(), again.Bounds())
	}

	if !bytes.Equal(first.Pix, again.Pix) {
		t.Fatal("coverage changed after scratch reuse")
	}
}

// TestGlyphScratchReuseKeepsCoverage runs the real glyph path with alternating
// glyph complexity so the pooled scratch is exercised end to end.
func TestGlyphScratchReuseKeepsCoverage(t *testing.T) {
	t.Parallel()

	face, err := pdf.DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	scale := float64(14) / float64(face.UnitsPerEm())

	first := rasterGlyph(face, 'A', scale)
	_ = rasterGlyph(face, 'i', scale)
	again := rasterGlyph(face, 'A', scale)

	if first.img == nil || again.img == nil {
		t.Fatal("rasterGlyph returned no alpha")
	}

	if first.img.Bounds() != again.img.Bounds() {
		t.Fatalf("bounds changed after scratch reuse: %v vs %v", first.img.Bounds(), again.img.Bounds())
	}

	if !bytes.Equal(first.img.Pix, again.img.Pix) {
		t.Fatal("glyph coverage changed after scratch reuse")
	}
}
