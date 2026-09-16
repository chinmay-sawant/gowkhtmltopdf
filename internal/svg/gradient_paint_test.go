package svg

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// Gradient paint servers are the reason prepareCanvasInput exists: the
// Programiz sp_logo.svg defines its linearGradient elements in a <defs> at the
// end of the document, and canvas only resolves defs already parsed. Before
// the hoist the P body rasterized solid black (0,0,0); these tests pin the
// gradient colors instead.

func decodeRaster(t *testing.T, pngBytes []byte) image.Image {
	t.Helper()

	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("png.Decode: %v", err)
	}

	return img
}

const pixelTolerance = 24

// pixel stores image samples as 16-bit-per-channel values, the form
// image.Image.At returns; keeping them wide avoids narrowing conversions.
type pixel struct {
	red   uint32
	green uint32
	blue  uint32
	alpha uint32
}

func pixelAt(img image.Image, fx, fy float64) pixel {
	bounds := img.Bounds()
	x := bounds.Min.X + int(fx*float64(bounds.Dx()))
	y := bounds.Min.Y + int(fy*float64(bounds.Dy()))
	red, green, blue, alpha := img.At(x, y).RGBA()

	return pixel{red: red, green: green, blue: blue, alpha: alpha}
}

func assertPixelNear(t *testing.T, label string, got pixel, want color.RGBA) {
	t.Helper()

	// Compare in 16-bit space: each 8-bit channel c scales to c*0x101.
	scale := func(channel uint8) uint32 { return uint32(channel) * 0x101 }

	diff := func(a, b uint32) uint32 {
		if a > b {
			return a - b
		}

		return b - a
	}

	tolerance := uint32(pixelTolerance) * 0x101

	if diff(got.red, scale(want.R)) > tolerance || diff(got.green, scale(want.G)) > tolerance ||
		diff(got.blue, scale(want.B)) > tolerance || diff(got.alpha, scale(want.A)) > tolerance {
		t.Fatalf("%s = %v, want ~%v (+-8-bit %d)", label, got, want, pixelTolerance)
	}
}

func TestRasterizeResolvesForwardReferencedLinearGradient(t *testing.T) {
	t.Parallel()

	// The rectangle references the gradient before the <defs> that declares
	// it, the order Programiz ships. Both gradientUnits variants are covered:
	// objectBoundingBox percentages and the file's userSpaceOnUse coordinates.
	cases := map[string]string{
		"objectBoundingBox": `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 20 20">` +
			`<rect width="20" height="20" fill="url(#a)"/>` +
			`<defs><linearGradient id="a" x1="0%" y1="0%" x2="100%" y2="0%">` +
			`<stop offset="0" stop-color="#ff0000"/><stop offset="1" stop-color="#0000ff"/>` +
			`</linearGradient></defs></svg>`,
		"userSpaceOnUse": `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 20 20">` +
			`<rect width="20" height="20" fill="url(#a)"/>` +
			`<defs><linearGradient id="a" gradientUnits="userSpaceOnUse" x1="0" y1="0" x2="20" y2="0">` +
			`<stop stop-color="#ff0000"/><stop offset="1" stop-color="#0000ff"/>` +
			`</linearGradient></defs></svg>`,
	}

	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			pngBytes, width, height, err := Rasterize([]byte(src), 256)
			if err != nil {
				t.Fatal(err)
			}

			if width != 20 || height != 20 {
				t.Fatalf("logical size = %dx%d, want 20x20", width, height)
			}

			img := decodeRaster(t, pngBytes)

			// Red at the start of the axis, blue at the end. Black (0,0,0)
			// means the def was not resolved before prepareCanvasInput.
			assertPixelNear(t, "left sample", pixelAt(img, 0.2, 0.5), color.RGBA{R: 204, B: 51, A: 255})
			assertPixelNear(t, "right sample", pixelAt(img, 0.8, 0.5), color.RGBA{R: 51, B: 204, A: 255})
		})
	}
}

func TestRasterizeResolvesLooseGradientOutsideDefs(t *testing.T) {
	t.Parallel()

	// A gradient element outside <defs> is legal SVG that canvas ignores
	// entirely; hoisting wraps it in a synthetic <defs>.
	src := `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 20 20">` +
		`<rect width="20" height="20" fill="url(#a)"/>` +
		`<linearGradient id="a" x1="0%" y1="0%" x2="100%" y2="0%">` +
		`<stop offset="0" stop-color="#ff0000"/><stop offset="1" stop-color="#0000ff"/>` +
		`</linearGradient></svg>`

	pngBytes, _, _, err := Rasterize([]byte(src), 256)
	if err != nil {
		t.Fatal(err)
	}

	img := decodeRaster(t, pngBytes)

	assertPixelNear(t, "left sample", pixelAt(img, 0.2, 0.5), color.RGBA{R: 204, B: 51, A: 255})
	assertPixelNear(t, "right sample", pixelAt(img, 0.8, 0.5), color.RGBA{R: 51, B: 204, A: 255})
}

func TestRasterizeResolvesForwardReferencedRadialGradient(t *testing.T) {
	t.Parallel()

	// Radial gradients share the defs parse/hoist path with linear ones, so
	// this pins the shared seam rather than a second implementation.
	src := `<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 20 20">` +
		`<rect width="20" height="20" fill="url(#a)"/>` +
		`<defs><radialGradient id="a" cx="50%" cy="50%" r="50%">` +
		`<stop offset="0" stop-color="#ff0000"/><stop offset="1" stop-color="#0000ff"/>` +
		`</radialGradient></defs></svg>`

	pngBytes, _, _, err := Rasterize([]byte(src), 256)
	if err != nil {
		t.Fatal(err)
	}

	img := decodeRaster(t, pngBytes)

	center := pixelAt(img, 0.5, 0.5)
	outer := pixelAt(img, 0.5, 0.95)

	if center.red <= center.blue {
		t.Fatalf("center = %v, want red-dominant gradient start", center)
	}

	if outer.blue <= outer.red {
		t.Fatalf("outer = %v, want blue-dominant gradient end", outer)
	}
}

func TestRasterizeResolvesLowercasedInlineGradient(t *testing.T) {
	t.Parallel()

	// The HTML pipeline lowercases element and attribute names, so inline SVG
	// reaches the rasterizer as <lineargradient> with viewbox.
	src := `<svg viewbox="0 0 20 20"><defs><lineargradient id="a" x1="0%" y1="0%" x2="100%" y2="0%">` +
		`<stop offset="0" stop-color="#ff0000"></stop><stop offset="1" stop-color="#0000ff"></stop>` +
		`</lineargradient></defs><rect width="20" height="20" fill="url(#a)"></rect></svg>`

	pngBytes, _, _, err := Rasterize([]byte(src), 256)
	if err != nil {
		t.Fatal(err)
	}

	img := decodeRaster(t, pngBytes)

	assertPixelNear(t, "left sample", pixelAt(img, 0.2, 0.5), color.RGBA{R: 204, B: 51, A: 255})
	assertPixelNear(t, "right sample", pixelAt(img, 0.8, 0.5), color.RGBA{R: 51, B: 204, A: 255})
}

func TestRasterizeNormalizesLowercasedRootViewBox(t *testing.T) {
	t.Parallel()

	// cplusplus.com's burger <svg> carries only viewBox="0 0 24 24" (the CSS
	// supplies 36px). Lowercased, canvas misses the viewBox and fits the
	// raster to the path's tight 18x12 bounds, stretching a 96x64 bitmap into
	// a 27x27pt box.
	src := `<svg focusable="false" viewbox="0 0 24 24">` +
		`<path d="M3 18h18v-2H3v2zm0-5h18v-2H3v2zm0-7v2h18V6H3z"></path></svg>`

	pngBytes, width, height, err := Rasterize([]byte(src), 1024)
	if err != nil {
		t.Fatal(err)
	}

	if width != 24 || height != 24 {
		t.Fatalf("logical size = %dx%d, want 24x24", width, height)
	}

	img := decodeRaster(t, pngBytes)
	bounds := img.Bounds()

	if bounds.Dx() != bounds.Dy() {
		t.Fatalf("raster = %dx%d pixels, want square viewBox aspect; tight-bounds fallback gives 96x64 at 4x",
			bounds.Dx(), bounds.Dy())
	}
}

func TestRasterizeFallsBackForUnresolvableFontFamily(t *testing.T) {
	t.Parallel()

	if !fontFamilyResolves("sans-serif") {
		t.Skip("no system fonts available for canvas font loading")
	}

	// cplusplus.com's wordmark asks for font-family="Roboto,arial"; with
	// neither installed, canvas panics and the whole wordmark image is lost.
	// A CSS-correct fallback must paint it with an installed family.
	src := `<svg xmlns="http://www.w3.org/2000/svg" width="60" height="20" viewBox="0 0 60 20">` +
		`<text x="1" y="15" font-family="Roboto,arial" font-size="14">wordmark</text></svg>`

	pngBytes, _, _, err := Rasterize([]byte(src), 256)
	if err != nil {
		t.Fatalf("Rasterize with unresolvable font family: %v", err)
	}

	img := decodeRaster(t, pngBytes)

	dark := 0

	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a > 0 && r>>8 < 128 && g>>8 < 128 && b>>8 < 128 {
				dark++
			}
		}
	}

	if dark == 0 {
		t.Fatal("fallback family painted no ink")
	}
}
