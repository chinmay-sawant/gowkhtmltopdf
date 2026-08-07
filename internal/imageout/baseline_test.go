package imageout

import (
	"image"
	"image/color"
	"math"
	"testing"

	"gowkhtmltopdf/internal/pdf"
)

// TestGlyphBaselineStable checks that non-descender letters share a common
// baseline within 1px (the pre-fix Floor/Round mix made letters bob).
func TestGlyphBaselineStable(t *testing.T) {
	face, err := pdf.DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	s := "Hamburgevons"
	sizePt := 24.0

	img := image.NewNRGBA(image.Rect(0, 0, 900, 90))
	for i := range img.Pix {
		img.Pix[i] = 255
	}

	const baseY = 55.0

	ttfDrawString(img, 10, baseY, s, sizePt, face, color.NRGBA{0, 0, 0, 255}, ptToPx, nil)

	x := 10.0

	var bottoms []int

	for _, r := range s {
		adv := face.AdvanceInPoints(r, sizePt) * ptToPx
		x0 := int(math.Floor(x))
		x1 := int(math.Ceil(x + adv))
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

		if r != 'g' && r != 'y' && r != 'p' && r != 'q' && r != 'j' && bot >= 0 {
			bottoms = append(bottoms, bot)
			t.Logf("%q bottom=%d (baseline %.0f, off=%d)", r, bot, baseY, bot-int(baseY))
		}

		x += adv
	}

	if len(bottoms) < 4 {
		t.Fatalf("need non-descender samples, got %d", len(bottoms))
	}

	minB, maxB := bottoms[0], bottoms[0]
	for _, b := range bottoms[1:] {
		if b < minB {
			minB = b
		}

		if b > maxB {
			maxB = b
		}
	}

	if maxB-minB > 1 {
		t.Errorf("baseline jitter: non-descender bottoms span %d px (want <=1)", maxB-minB)
	}
}
