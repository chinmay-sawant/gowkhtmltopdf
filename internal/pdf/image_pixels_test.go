package pdf

import (
	"bytes"
	"image"
	"testing"
)

const (
	testPixelStride = 37
	testPixelOffset = 11
)

// renderImagePixelsReference is the pre-fast-path generic implementation,
// kept as the bit-exact oracle for the concrete accessor paths.
func renderImagePixelsReference(img image.Image, bounds image.Rectangle, grayscale bool) ([]byte, bool) {
	width, height := bounds.Dx(), bounds.Dy()
	rgba := make([]byte, width*height*rgbChannels)
	hasAlpha := false

	for yy := range height {
		for xx := range width {
			red, green, blue, alpha := img.At(bounds.Min.X+xx, bounds.Min.Y+yy).RGBA()
			off := (yy*width + xx) * rgbChannels

			if grayscale {
				v := byte(lumaR*float64(red>>bitsPerByte) + lumaG*float64(green>>bitsPerByte) + lumaB*float64(blue>>bitsPerByte))
				rgba[off], rgba[off+1], rgba[off+2] = v, v, v
			} else {
				rgba[off] = byte(red >> bitsPerByte)
				rgba[off+1] = byte(green >> bitsPerByte)
				rgba[off+2] = byte(blue >> bitsPerByte)
			}

			if alpha < maxUint16Val {
				hasAlpha = true
			}
		}
	}

	return rgba, hasAlpha
}

// renderAlphaMaskReference is the generic oracle for renderAlphaMask.
func renderAlphaMaskReference(img image.Image, bounds image.Rectangle) []byte {
	width, height := bounds.Dx(), bounds.Dy()
	mask := make([]byte, width*height)

	for yy := range height {
		for xx := range width {
			_, _, _, alpha := img.At(bounds.Min.X+xx, bounds.Min.Y+yy).RGBA()
			mask[yy*width+xx] = byte(alpha >> bitsPerByte)
		}
	}

	return mask
}

func filledRGBA(bounds image.Rectangle) *image.RGBA {
	img := image.NewRGBA(bounds)

	for i := range img.Pix {
		img.Pix[i] = byte(i*testPixelStride + testPixelOffset)
	}

	return img
}

func opaqueRGBA(bounds image.Rectangle) *image.RGBA {
	img := image.NewRGBA(bounds)

	for i := range img.Pix {
		img.Pix[i] = maxUint8
	}

	return img
}

func filledNRGBA(bounds image.Rectangle) *image.NRGBA {
	img := image.NewNRGBA(bounds)

	for i := range img.Pix {
		img.Pix[i] = byte(i*testPixelStride + testPixelOffset)
	}

	return img
}

func filledGray(bounds image.Rectangle) *image.Gray {
	img := image.NewGray(bounds)

	for i := range img.Pix {
		img.Pix[i] = byte(i*testPixelStride + testPixelOffset)
	}

	return img
}

func TestRenderImagePixelsConcreteAccessorsBitExact(t *testing.T) {
	t.Parallel()

	// Non-zero bounds.Min proves the fast paths honor the sampling origin.
	bounds := image.Rect(3, 5, 3+7, 5+5)

	cases := []struct {
		name string
		img  image.Image
	}{
		{name: "RGBA", img: filledRGBA(bounds)},
		{name: "RGBA-opaque", img: opaqueRGBA(bounds)},
		{name: "NRGBA", img: filledNRGBA(bounds)},
		{name: "Gray", img: filledGray(bounds)},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			for _, grayscale := range []bool{false, true} {
				got, gotAlpha := renderImagePixels(testCase.img, bounds, grayscale)
				want, wantAlpha := renderImagePixelsReference(testCase.img, bounds, grayscale)

				if gotAlpha != wantAlpha {
					t.Fatalf("grayscale=%v: hasAlpha = %v, want %v", grayscale, gotAlpha, wantAlpha)
				}

				if !bytes.Equal(got, want) {
					t.Fatalf("grayscale=%v: pixels differ from the generic path", grayscale)
				}
			}

			gotMask := renderAlphaMask(testCase.img, bounds)
			wantMask := renderAlphaMaskReference(testCase.img, bounds)

			if !bytes.Equal(gotMask, wantMask) {
				t.Fatalf("alpha mask differs from the generic path")
			}
		})
	}
}

func TestRenderImagePixelsSubImageBounds(t *testing.T) {
	t.Parallel()

	// A sub-image keeps a non-zero Rect over a larger backing Pix; the fast
	// path must read through RGBAAt/NRGBAAt offsets, not Pix[0].
	base := filledRGBA(image.Rect(0, 0, 12, 9))
	sub, ok := base.SubImage(image.Rect(2, 1, 9, 6)).(*image.RGBA)

	if !ok {
		t.Fatal("RGBA sub-image lost its concrete type")
	}

	bounds := sub.Bounds()

	got, gotAlpha := renderImagePixels(sub, bounds, false)
	want, wantAlpha := renderImagePixelsReference(sub, bounds, false)

	if gotAlpha != wantAlpha {
		t.Fatalf("hasAlpha = %v, want %v", gotAlpha, wantAlpha)
	}

	if !bytes.Equal(got, want) {
		t.Fatal("sub-image pixels differ from the generic path")
	}

	gotMask, wantMask := renderAlphaMask(sub, bounds), renderAlphaMaskReference(sub, bounds)
	if !bytes.Equal(gotMask, wantMask) {
		t.Fatal("sub-image alpha mask differs from the generic path")
	}
}
