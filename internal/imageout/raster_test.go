package imageout

import (
	"image/color"
	"testing"

	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/pdf"
)

func TestTTFRasterAntiAliased(t *testing.T) {
	face, err := pdf.DefaultFont()
	if err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(`<html><body><h1>Acme Widgets</h1><p>Invoice line item spacing.</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	img, err := Render(root, RenderOptions{Width: 400, Font: face, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	// Count unique colors: 5x7 bitmap had ~few solid colors; AA TTF has many.
	seen := map[color.Color]struct{}{}
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			seen[img.At(x, y)] = struct{}{}
			if len(seen) > 40 {
				return
			}
		}
	}
	// 4x4 coverage AA yields many greys vs the old solid 5x7 bitmap (~few colors).
	if len(seen) < 10 {
		t.Errorf("unique colors = %d, want >= 10 (AA TTF text)", len(seen))
	}
}

func TestTTFAdvanceMatchesLayoutWidth(t *testing.T) {
	face, err := pdf.DefaultFont()
	if err != nil {
		t.Fatal(err)
	}
	s := "Acme"
	var layoutW float64
	for _, r := range s {
		layoutW += face.AdvanceInPoints(r, 12)
	}
	// image advances use same AdvanceInPoints * ptToPx
	imgW := layoutW * ptToPx
	if imgW < 10 || imgW > 200 {
		t.Fatalf("unexpected width scale %v", imgW)
	}
	// render single line and ensure non-white span is roughly that width
	root, err := html.Parse(`<html><body style="margin:0;padding:0"><p style="margin:0;font-size:12pt">` + s + `</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	img, err := Render(root, RenderOptions{Width: 200, Font: face, SmartWidth: false, Background: true})
	if err != nil {
		t.Fatal(err)
	}
	b := img.Bounds()
	minX, maxX := b.Max.X, b.Min.X
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, _ := img.At(x, y).RGBA()
			if r < 0xf000 || g < 0xf000 || bl < 0xf000 {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
			}
		}
	}
	if maxX <= minX {
		t.Fatal("no ink found")
	}
	span := float64(maxX - minX)
	// allow generous tolerance for glyph bearings
	if span < imgW*0.4 || span > imgW*2.5 {
		t.Errorf("ink span %v vs advance-based %v", span, imgW)
	}
}
