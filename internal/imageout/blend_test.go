//nolint:testpackage,exhaustruct // white-box raster compositing test
package imageout

import (
	"image"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

func TestPaintBlendedFillUsesMultiply(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(0, 0, 1, 1))
	atlas := newGlyphAtlas()
	cache := newRasterImageCache()
	paint(img, &layout.Op{
		Kind: layout.OpFillRect,
		W:    1,
		H:    1,
		R:    0.5,
		G:    0.5,
		B:    0.5,
	}, 1, atlas, cache)
	paint(img, &layout.Op{
		Kind:      layout.OpFillRect,
		W:         1,
		H:         1,
		R:         1,
		BlendMode: "multiply",
	}, 1, atlas, cache)

	got := img.NRGBAAt(0, 0)
	if got.R != 127 || got.G != 0 || got.B != 0 || got.A != 255 {
		t.Fatalf("pixel = %#v, want red 127 over opaque black channels", got)
	}
}
