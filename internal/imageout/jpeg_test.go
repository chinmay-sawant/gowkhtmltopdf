package imageout

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

// makeJPEGProbeImage builds a deterministic NRGBA image with varied colors,
// and transparent pixels when requested (alpha reaches 0 so premultiplication
// matters).
func makeJPEGProbeImage(rect image.Rectangle, transparent bool) *image.NRGBA {
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

// stdlibJPEGBytes encodes img through image/jpeg with the same options encode
// passes, producing the reference bytes for the parity assertions below.
func stdlibJPEGBytes(t *testing.T, img image.Image, quality int) []byte {
	t.Helper()

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		t.Fatalf("jpeg.Encode: %v", err)
	}

	return buf.Bytes()
}

// TestEncodeJPGMatchesStdlibBytes is the byte-stability proof for the JPEG
// encoder: encode hands the image straight to image/jpeg, so its bytes must
// equal a direct jpeg.Encode of the same image at the same quality, including
// odd dimensions, transparency, and shifted even origins.
func TestEncodeJPGMatchesStdlibBytes(t *testing.T) {
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
			img := makeJPEGProbeImage(rect, testCase.transparent)
			want := stdlibJPEGBytes(t, img, 90)

			got, err := encode(img, formatJPG, 90, false)
			if err != nil {
				t.Fatalf("encode: %v", err)
			}

			if !bytes.Equal(got, want) {
				t.Fatalf("encode JPEG differs from jpeg.Encode at %v: %d vs %d bytes", rect, len(got), len(want))
			}
		})
	}
}

// TestEncodeJPGMatchesStdlibAtQuality checks the quality wiring: the quality
// reaches jpeg.Options unchanged for a transparent probe image.
func TestEncodeJPGMatchesStdlibAtQuality(t *testing.T) {
	t.Parallel()

	for _, size := range []image.Point{{X: 64, Y: 48}, {X: 37, Y: 23}} {
		img := makeJPEGProbeImage(image.Rect(0, 0, size.X, size.Y), true)
		want := stdlibJPEGBytes(t, img, 80)

		got, err := encode(img, formatJPG, 80, false)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}

		if !bytes.Equal(got, want) {
			t.Fatalf("encode JPEG differs from jpeg.Encode at %v: %d vs %d bytes", size, len(got), len(want))
		}
	}
}
