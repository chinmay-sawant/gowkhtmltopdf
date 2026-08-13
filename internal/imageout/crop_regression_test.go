package imageout //nolint:testpackage // uses RenderOptions internals

import (
	"image"
	"image/color"
	"testing"

	"gowkhtmltopdf/internal/html"
)

// TestCropWindowsStayInkful is the visual gate for representative headers.
// It rasterizes a small window and requires mean luminance below a wash-out
// threshold so a missing masthead fails.
func TestCropWindowsStayInkful(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		html string
		maxL float64
	}{
		{
			name: "invoice-header",
			html: `<html><body><h1 style="color:#1a3d6d">Acme Widgets GmbH</h1><p>Invoice No. 2024-0001</p></body></html>`,
			maxL: 250,
		},
		{
			name: "masthead",
			html: `<html><body><h1 style="letter-spacing:2px">Northline Field Operations</h1></body></html>`,
			maxL: 250,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			root, err := html.Parse(testCase.html)
			if err != nil {
				t.Fatal(err)
			}

			img, err := Render(root, RenderOptions{Width: 240, Height: 80, Background: true}) //nolint:exhaustruct
			if err != nil {
				t.Fatal(err)
			}

			if meanLuma(img) > testCase.maxL {
				t.Fatalf("crop is washed out: luma=%.1f want <= %.1f", meanLuma(img), testCase.maxL)
			}
		})
	}
}

func meanLuma(img image.Image) float64 {
	bounds := img.Bounds()

	var sum float64

	count := 0

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			pixel := color.NRGBA{
				R: uint8(r / 256), //nolint:gosec // 16-bit channel scaled to 8-bit
				G: uint8(g / 256), //nolint:gosec
				B: uint8(b / 256), //nolint:gosec
				A: 255,
			}
			sum += 0.299*float64(pixel.R) + 0.587*float64(pixel.G) + 0.114*float64(pixel.B)
			count++
		}
	}

	if count == 0 {
		return 255
	}

	return sum / float64(count)
}
