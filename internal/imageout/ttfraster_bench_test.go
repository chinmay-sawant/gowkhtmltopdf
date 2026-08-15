package imageout //nolint:testpackage // benchmark exercises the unexported raster hot path.

import (
	"image"
	"image/color"
	"strconv"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

//nolint:wsl // benchmark timing boundaries intentionally surround the loop.
func BenchmarkColdGlyphRaster(b *testing.B) {
	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		b.Fatal(err)
	}

	for _, size := range []int{12, 24, 72} {
		for _, sample := range []string{"Latin", "CJK"} {
			text := "Benchmark"
			face := faces.Regular
			if sample == "CJK" {
				text = "日本語"
				face = faces.UnicodeFallback
			}

			b.Run(sample+"/"+strconv.Itoa(size)+"px", func(b *testing.B) {
				b.ReportAllocs()
				for range b.N {
					atlas := newGlyphAtlas()
					img := image.NewNRGBA(image.Rect(0, 0, size*16, size*2))
					ttfDrawString(
						img, 0, float64(size), text, float64(size),
						0, 0, face, color.NRGBA{R: 0, G: 0, B: 0, A: 255}, 1, atlas,
					)
				}
			})
		}
	}
}
