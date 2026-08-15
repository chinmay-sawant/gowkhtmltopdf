//nolint:testpackage // white-box test drives ttfDrawString and ptToPx directly
package imageout

import (
	"image"
	"image/color"
	"math"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// glyphInkBottom scans the columns in [x0, x1) for the lowest dark pixel row
// (-1 when the glyph paints no dark pixel in that band).
func glyphInkBottom(img *image.NRGBA, x0, x1 int) int {
	bot := -1

	for yy := range 90 {
		for xx := x0; xx < x1 && xx < 900; xx++ {
			c := img.NRGBAAt(xx, yy)
			if c.R < 40 && c.G < 40 && c.B < 40 {
				if yy > bot {
					bot = yy
				}
			}
		}
	}

	return bot
}

// bottomExtremes returns the min and max of bottoms.
func bottomExtremes(bottoms []int) (int, int) {
	minB, maxB := bottoms[0], bottoms[0]

	for _, bottom := range bottoms[1:] {
		if bottom < minB {
			minB = bottom
		}

		if bottom > maxB {
			maxB = bottom
		}
	}

	return minB, maxB
}

// TestGlyphBaselineStable checks that non-descender letters share a common
// baseline within 1px (the pre-fix Floor/Round mix made letters bob).
func TestGlyphBaselineStable(t *testing.T) {
	t.Parallel()

	face, err := pdf.DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	sample := "Hamburgevons"
	sizePt := 24.0

	img := image.NewNRGBA(image.Rect(0, 0, 900, 90))
	for i := range img.Pix {
		img.Pix[i] = 255
	}

	const baseY = 55.0

	ttfDrawString(img, 10, baseY, sample, sizePt, 0, 0, face, color.NRGBA{0, 0, 0, 255}, ptToPx, nil)

	penX := 10.0

	var bottoms []int

	for _, runeVal := range sample {
		adv := face.AdvanceInPoints(runeVal, sizePt) * ptToPx
		x0 := int(math.Floor(penX))
		x1 := int(math.Ceil(penX + adv))
		bot := glyphInkBottom(img, x0, x1)

		if bot >= 0 && !strings.ContainsRune("gypqj", runeVal) {
			bottoms = append(bottoms, bot)
			t.Logf("%q bottom=%d (baseline %.0f, off=%d)", runeVal, bot, baseY, bot-int(baseY))
		}

		penX += adv
	}

	if len(bottoms) < 4 {
		t.Fatalf("need non-descender samples, got %d", len(bottoms))
	}

	minB, maxB := bottomExtremes(bottoms)
	if maxB-minB > 1 {
		t.Errorf("baseline jitter: non-descender bottoms span %d px (want <=1)", maxB-minB)
	}
}
