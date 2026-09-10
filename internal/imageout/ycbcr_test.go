package imageout

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// makeYCbCrProbeImage builds a deterministic NRGBA image with varied colors,
// and transparent pixels when requested (alpha reaches 0 so premultiplication
// matters).
func makeYCbCrProbeImage(rect image.Rectangle, transparent bool) *image.NRGBA {
	img := image.NewNRGBA(rect)

	for row := rect.Min.Y; row < rect.Max.Y; row++ {
		for col := rect.Min.X; col < rect.Max.X; col++ {
			alpha := uint8(opaqueAlpha)
			if transparent {
				alpha = uint8((col*29 + row*17) % 256) //nolint:gosec // modulo 256 fits uint8
			}

			img.SetNRGBA(col, row, color.NRGBA{
				R: uint8((col*7 + row*3) % 256),   //nolint:gosec // modulo 256 fits uint8
				G: uint8((col*5 + row*11) % 256),  //nolint:gosec // modulo 256 fits uint8
				B: uint8((col*13 + row*17) % 256), //nolint:gosec // modulo 256 fits uint8
				A: alpha,
			})
		}
	}

	return img
}

// stdlibJPEGBytes encodes img through image/jpeg, which uses its own
// per-pixel toYCbCr conversion for NRGBA input.
func stdlibJPEGBytes(t *testing.T, img image.Image, quality int) []byte {
	t.Helper()

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		t.Fatalf("jpeg.Encode: %v", err)
	}

	return buf.Bytes()
}

// TestNRGBAToYCbCr420KeepsJPEGBytes is the byte-equality proof for the fast
// path: the precomputed 4:2:0 planes must encode to exactly the bytes the
// stdlib NRGBA path produces, including odd dimensions, transparency, and
// shifted even origins.
func TestNRGBAToYCbCr420KeepsJPEGBytes(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name        string
		origin      image.Point
		width       int
		height      int
		transparent bool
	}{
		{name: "even", width: 64, height: 48},
		{name: "odd", width: 37, height: 23},
		{name: "transparent", width: 41, height: 19, transparent: true},
		{name: "shifted-origin", origin: image.Point{X: 2, Y: 4}, width: 33, height: 17, transparent: true},
		{name: "single-pixel", width: 1, height: 1, transparent: true},
		{name: "thin", width: 5, height: 1, transparent: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			rect := image.Rect(
				testCase.origin.X, testCase.origin.Y,
				testCase.origin.X+testCase.width, testCase.origin.Y+testCase.height,
			)
			img := makeYCbCrProbeImage(rect, testCase.transparent)
			want := stdlibJPEGBytes(t, img, 90)

			ycbcr := nrgbaToYCbCr420(img)
			if ycbcr == nil {
				t.Fatal("nrgbaToYCbCr420 returned nil for an even origin")
			}

			got := stdlibJPEGBytes(t, ycbcr, 90)
			if !bytes.Equal(got, want) {
				t.Fatalf("YCbCr JPEG differs from NRGBA JPEG: %d vs %d bytes", len(got), len(want))
			}
		})
	}
}

// TestEncodeJPGUsesFastPathForNRGBA checks the encode wiring: a JPEG from
// encode must match the stdlib NRGBA path byte for byte.
func TestEncodeJPGUsesFastPathForNRGBA(t *testing.T) {
	t.Parallel()

	for _, size := range []image.Point{{X: 64, Y: 48}, {X: 37, Y: 23}} {
		img := makeYCbCrProbeImage(image.Rect(0, 0, size.X, size.Y), true)
		want := stdlibJPEGBytes(t, img, 80)

		got, err := encode(img, formatJPG, 80)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}

		if !bytes.Equal(got, want) {
			t.Fatalf("encode JPEG differs from stdlib NRGBA JPEG at %v: %d vs %d bytes", size, len(got), len(want))
		}
	}
}

// TestNRGBAToYCbCr420RejectsOddOrigin covers the fallback: an odd origin
// cannot be represented on the 4:2:0 grid, so encode must keep the stdlib
// NRGBA conversion and still produce identical bytes.
func TestNRGBAToYCbCr420RejectsOddOrigin(t *testing.T) {
	t.Parallel()

	img := image.NewNRGBA(image.Rect(1, 2, 33, 18))
	if ycbcr := nrgbaToYCbCr420(img); ycbcr != nil {
		t.Fatal("nrgbaToYCbCr420 accepted an odd origin")
	}

	want := stdlibJPEGBytes(t, img, 75)

	got, err := encode(img, formatJPG, 75)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}

	if !bytes.Equal(got, want) {
		t.Fatalf("odd-origin JPEG differs from stdlib NRGBA JPEG: %d vs %d bytes", len(got), len(want))
	}
}
