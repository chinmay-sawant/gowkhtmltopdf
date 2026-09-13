package imageout

import (
	"bytes"
	"image"
	"image/jpeg"
	"testing"
)

// TestNRGBAToYCbCrMatchesStdlibBytes is the byte-parity proof for the
// conversion: jpeg.Encode of the converted planes must equal jpeg.Encode of the
// NRGBA source for every size and origin, including partial-MCU edges at both
// odd and even dimensions, transparency, and both opaque-hint values.
func TestNRGBAToYCbCrMatchesStdlibBytes(t *testing.T) {
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
		{name: "mcu-edge", width: 17, height: 17},
		{name: "two-mcus", width: 40, height: 20},
		{name: "even-partial", width: 24, height: 40},
		{name: "even-partial-height", width: 30, height: 6},
		{name: "tiny-even", width: 2, height: 2, transparent: true},
		{name: "eight-square", width: 8, height: 8},
		{name: "tall-thin", width: 3, height: 34, transparent: true},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			rect := image.Rect(
				testCase.origin.X, testCase.origin.Y,
				testCase.origin.X+testCase.width, testCase.origin.Y+testCase.height,
			)
			img := makeJPEGProbeImage(rect, testCase.transparent)

			var want bytes.Buffer
			if err := jpeg.Encode(&want, img, &jpeg.Options{Quality: 90}); err != nil {
				t.Fatalf("stdlib jpeg.Encode: %v", err)
			}

			hints := []bool{false}
			if !testCase.transparent {
				hints = append(hints, true)
			}

			for _, opaque := range hints {
				converted := nrgbaToYCbCr(img, opaque)

				var got bytes.Buffer
				if err := jpeg.Encode(&got, converted, &jpeg.Options{Quality: 90}); err != nil {
					t.Fatalf("jpeg.Encode(YCbCr): %v", err)
				}

				if !bytes.Equal(got.Bytes(), want.Bytes()) {
					t.Fatalf("opaque=%v: %d bytes differ from stdlib %d bytes",
						opaque, got.Len(), want.Len())
				}
			}
		})
	}
}

// TestNRGBAToYCbCrDoesNotBoxPixels guards the point of the conversion:
// per-pixel color boxing would allocate proportional to the destination.
//
//nolint:paralleltest // testing.AllocsPerRun panics during parallel tests.
func TestNRGBAToYCbCrDoesNotBoxPixels(t *testing.T) {
	img := makeJPEGProbeImage(image.Rect(0, 0, 64, 48), false)

	allocs := testing.AllocsPerRun(10, func() {
		_ = nrgbaToYCbCr(img, true)
	})

	if allocs > 4 {
		t.Fatalf("nrgbaToYCbCr allocated %.0f times per run, want at most 4", allocs)
	}

	t.Logf("nrgbaToYCbCr allocations per 64x48 conversion: %.0f", allocs)
}
