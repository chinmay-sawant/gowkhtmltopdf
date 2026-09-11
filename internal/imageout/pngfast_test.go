package imageout

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"
)

// pngfast_test.go pins the fast large-canvas PNG encoder: the gate stays on
// the directRasterPixels boundary, small canvases remain byte-identical to
// image/png, and every fast-path output decodes back to the exact source
// pixels (PNG filters and compression levels are lossless). A change that
// breaks any of those contracts fails here instead of silently changing
// published image bytes or pixels.

// TestFastPNGCanvasGate checks the large-canvas boundary. 1024*2048 is
// exactly directRasterPixels, so the threshold row and its neighbor are both
// pinned.
func TestFastPNGCanvasGate(t *testing.T) {
	t.Parallel()

	if !fastPNGCanvas(image.Rect(0, 0, 1024, 2048)) {
		t.Error("canvas exactly at directRasterPixels = false, want true")
	}

	if fastPNGCanvas(image.Rect(0, 0, 1024, 2047)) {
		t.Error("canvas one row below directRasterPixels = true, want false")
	}
}

// TestSmallPNGStaysByteIdenticalToStdlib proves the gate keeps small outputs
// unchanged: below the boundary encodePNG must return exactly image/png bytes.
func TestSmallPNGStaysByteIdenticalToStdlib(t *testing.T) {
	t.Parallel()

	src := probePatternNRGBA(image.Rect(0, 0, 64, 48), true)

	var fast, stdlib bytes.Buffer

	if err := encodePNG(&fast, src, true); err != nil {
		t.Fatalf("encodePNG: %v", err)
	}

	if err := png.Encode(&stdlib, src); err != nil {
		t.Fatalf("png.Encode: %v", err)
	}

	if !bytes.Equal(fast.Bytes(), stdlib.Bytes()) {
		t.Fatal("small canvas did not use image/png byte-for-byte")
	}
}

// TestFastPNGRoundTripsSourcePixels encodes odd and small canvases directly
// through encodeFastPNG, with and without alpha, and requires the decoded
// pixels to equal the source. Odd widths exercise the 4-pixel packing tail,
// and the sub-image case covers a non-zero origin with a padded stride.
func TestFastPNGRoundTripsSourcePixels(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name      string
		width     int
		height    int
		opaque    bool
		subImageX int
	}{
		{name: "1x1-opaque", width: 1, height: 1, opaque: true},
		{name: "3x2-opaque", width: 3, height: 2, opaque: true},
		{name: "5x3-alpha", width: 5, height: 3, opaque: false},
		{name: "97x53-opaque", width: 97, height: 53, opaque: true},
		{name: "64x64-alpha", width: 64, height: 64, opaque: false},
		{name: "31x17-opaque-subimage", width: 31, height: 17, opaque: true, subImageX: 5},
		{name: "37x23-opaque-no-hint", width: 37, height: 23, opaque: true, subImageX: -1},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			src := probeCaseImage(t, testCase.width, testCase.height, testCase.opaque, testCase.subImageX)
			// subImageX -1 marks the opaque canvas encoded without the hint,
			// so the scan path is covered too.
			hint := testCase.opaque && testCase.subImageX != -1

			assertFastPNGRoundTrip(t, src, hint)
		})
	}
}

// probeCaseImage builds the source canvas for one round-trip case: a plain
// pattern, or a sub-image window when subImageX is positive.
func probeCaseImage(t *testing.T, width, height int, opaque bool, subImageX int) *image.NRGBA {
	t.Helper()

	if subImageX <= 0 {
		return probePatternNRGBA(image.Rect(0, 0, width, height), opaque)
	}

	wide := probePatternNRGBA(image.Rect(0, 0, width+2*subImageX, height), opaque)
	window := wide.SubImage(image.Rect(subImageX, 0, subImageX+width, height))

	nrgba, ok := window.(*image.NRGBA)
	if !ok {
		t.Fatalf("sub-image type = %T, want *image.NRGBA", window)
	}

	return nrgba
}

// assertFastPNGRoundTrip encodes src through the fast path with the given
// opaque hint and requires the decoded pixels to equal the source.
func assertFastPNGRoundTrip(t *testing.T, src *image.NRGBA, opaque bool) {
	t.Helper()

	var encoded bytes.Buffer

	if err := encodeFastPNG(&encoded, src, opaque); err != nil {
		t.Fatalf("encodeFastPNG: %v", err)
	}

	decoded, err := png.Decode(bytes.NewReader(encoded.Bytes()))
	if err != nil {
		t.Fatalf("decode fast PNG: %v", err)
	}

	assertPNGPixelsEqual(t, src, decoded)
}

// probePatternNRGBA builds a deterministic NRGBA canvas whose channels vary
// per pixel, so a mispacked row or dropped alpha fails the pixel compare.
func probePatternNRGBA(bounds image.Rectangle, opaque bool) *image.NRGBA {
	img := image.NewNRGBA(bounds)

	for pixelY := bounds.Min.Y; pixelY < bounds.Max.Y; pixelY++ {
		for pixelX := bounds.Min.X; pixelX < bounds.Max.X; pixelX++ {
			alpha := uint8((pixelX*7 + pixelY*13) % 251) //nolint:gosec // pattern modulo 251 fits uint8
			if opaque {
				alpha = opaqueAlpha
			}

			img.SetNRGBA(pixelX, pixelY, color.NRGBA{
				//nolint:gosec // small canvas coordinates, channels wrap on purpose
				R: uint8(pixelX*3 + pixelY),
				//nolint:gosec // small canvas coordinates, channels wrap on purpose
				G: uint8(pixelX + 5*pixelY),
				//nolint:gosec // small canvas coordinates, channels wrap on purpose
				B: uint8(pixelX*11 + pixelY*2),
				A: alpha,
			})
		}
	}

	return img
}

// assertPNGPixelsEqual normalizes decoded to NRGBA and compares every channel
// with the source, including alpha, over the source bounds. PNG loses the
// source origin, so only the dimensions and the pixel values are compared.
func assertPNGPixelsEqual(t *testing.T, src *image.NRGBA, decoded image.Image) {
	t.Helper()

	srcBounds := src.Bounds()
	decodedBounds := decoded.Bounds()

	if decodedBounds.Dx() != srcBounds.Dx() || decodedBounds.Dy() != srcBounds.Dy() {
		t.Fatalf("decoded dimensions = %dx%d, want %dx%d",
			decodedBounds.Dx(), decodedBounds.Dy(), srcBounds.Dx(), srcBounds.Dy())
	}

	dst := image.NewNRGBA(srcBounds)
	draw.Draw(dst, dst.Bounds(), decoded, decodedBounds.Min, draw.Src)

	if !bytes.Equal(dst.Pix, src.Pix) {
		for pixelY := srcBounds.Min.Y; pixelY < srcBounds.Max.Y; pixelY++ {
			for pixelX := srcBounds.Min.X; pixelX < srcBounds.Max.X; pixelX++ {
				if dst.NRGBAAt(pixelX, pixelY) != src.NRGBAAt(pixelX, pixelY) {
					t.Fatalf("pixel (%d,%d) = %v, want %v",
						pixelX, pixelY, dst.NRGBAAt(pixelX, pixelY), src.NRGBAAt(pixelX, pixelY))
				}
			}
		}
	}
}
