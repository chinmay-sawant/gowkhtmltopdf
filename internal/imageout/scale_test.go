package imageout

import (
	"bytes"
	"image"
	"image/color"
	"testing"
)

// scaleNearestGenericReference is the pre-IMPROV-10 loop: color.NRGBAModel
// boxes every sampled pixel. The parity test uses it as the byte-exact oracle
// for the concrete fast paths.
func scaleNearestGenericReference(src image.Image, width, height int) *image.NRGBA {
	dst, err := newRasterImage(width, height)
	if err != nil {
		return nil
	}

	srcBounds := src.Bounds()

	if srcBounds.Dx() == 0 || srcBounds.Dy() == 0 {
		return dst
	}

	scaleX := float64(srcBounds.Dx()) / float64(width)

	scaleY := float64(srcBounds.Dy()) / float64(height)
	for row := range height {
		srcY := srcBounds.Min.Y + int((float64(row)+pixelCenter)*scaleY)
		if srcY > srcBounds.Max.Y-1 {
			srcY = srcBounds.Max.Y - 1
		}

		for col := range width {
			srcX := srcBounds.Min.X + int((float64(col)+pixelCenter)*scaleX)
			if srcX > srcBounds.Max.X-1 {
				srcX = srcBounds.Max.X - 1
			}

			nc, ok := color.NRGBAModel.Convert(src.At(srcX, srcY)).(color.NRGBA)
			if !ok {
				continue
			}

			dst.SetNRGBA(col, row, nc)
		}
	}

	return dst
}

// fillScaleProbeYCbCr writes deterministic luma and chroma planes, keeping
// the source on a non-zero origin.
func fillScaleProbeYCbCr(rect image.Rectangle, ratio image.YCbCrSubsampleRatio) *image.YCbCr {
	img := image.NewYCbCr(rect, ratio)

	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			img.Y[img.YOffset(x, y)] = uint8((x*11 + y*5) % 256)  //nolint:gosec // modulo 256 fits uint8
			img.Cb[img.COffset(x, y)] = uint8((x*3 + y*13) % 256) //nolint:gosec // modulo 256 fits uint8
			img.Cr[img.COffset(x, y)] = uint8((x*17 + y*7) % 256) //nolint:gosec // modulo 256 fits uint8
		}
	}

	return img
}

// newScaleProbeSources builds one deterministic image per supported source
// type, all sharing the same non-zero origin so the offset math is exercised.
func newScaleProbeSources() map[string]image.Image {
	rect := image.Rect(2, 3, 2+41, 3+27)
	sources := map[string]image.Image{
		"ycbcr-420": fillScaleProbeYCbCr(rect, image.YCbCrSubsampleRatio420),
		"ycbcr-422": fillScaleProbeYCbCr(rect, image.YCbCrSubsampleRatio422),
		"ycbcr-444": fillScaleProbeYCbCr(rect, image.YCbCrSubsampleRatio444),
	}

	nrgba := image.NewNRGBA(rect)
	rgba := image.NewRGBA(rect)
	gray := image.NewGray(rect)
	paletted := image.NewPaletted(rect, color.Palette{
		color.NRGBA{R: 255, A: 255},
		color.NRGBA{G: 255, A: 255},
		color.NRGBA{B: 255, A: 255},
		color.NRGBA{R: 255, G: 255, B: 255, A: 128},
		color.NRGBA{R: 10, G: 200, B: 40, A: 200},
		color.NRGBA{},
	})

	for row := rect.Min.Y; row < rect.Max.Y; row++ {
		for col := rect.Min.X; col < rect.Max.X; col++ {
			alpha := uint8((col*29 + row*17) % 256) //nolint:gosec // modulo 256 fits uint8
			pixel := color.NRGBA{
				R: uint8((col*7 + row*3) % 256),   //nolint:gosec // modulo 256 fits uint8
				G: uint8((col*5 + row*11) % 256),  //nolint:gosec // modulo 256 fits uint8
				B: uint8((col*13 + row*17) % 256), //nolint:gosec // modulo 256 fits uint8
				A: alpha,
			}
			nrgba.SetNRGBA(col, row, pixel)
			rgba.Set(col, row, pixel)
			gray.SetGray(col, row, color.Gray{Y: uint8((col*19 + row*23) % 256)})    //nolint:gosec // modulo 256 fits uint8
			paletted.SetColorIndex(col, row, uint8((col+row)%len(paletted.Palette))) //nolint:gosec // palette index fits uint8
		}
	}

	sources["nrgba"] = nrgba
	sources["rgba"] = rgba
	sources["gray"] = gray
	sources["paletted"] = paletted

	return sources
}

// TestScaleNearestGenericMatchesConvert is the byte-parity proof for the
// concrete conversion fast paths: every source type and scale step must equal
// the color.NRGBAModel.Convert reference exactly.
func TestScaleNearestGenericMatchesConvert(t *testing.T) {
	t.Parallel()

	sizes := []image.Point{
		{X: 41, Y: 27}, // 1:1
		{X: 17, Y: 9},  // downscale
		{X: 83, Y: 54}, // upscale
		{X: 1, Y: 1},
		{X: 40, Y: 26}, // slight shrink from odd dimensions
	}

	for name, src := range newScaleProbeSources() {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			for _, size := range sizes {
				want := scaleNearestGenericReference(src, size.X, size.Y)
				got := scaleNearestGeneric(src, size.X, size.Y)

				if want == nil || got == nil {
					t.Fatalf("scale to %v returned nil (want=%v got=%v)", size, want, got)
				}

				if got.Bounds() != want.Bounds() || !bytes.Equal(got.Pix, want.Pix) {
					t.Fatalf("scale to %v differs from the Convert reference", size)
				}
			}
		})
	}
}

// TestScaleNearestGenericDoesNotBoxPixels guards the point of the fast path:
// per-pixel color boxing would show up as allocations proportional to the
// destination, far above the destination canvas itself.
//
//nolint:paralleltest // testing.AllocsPerRun panics during parallel tests.
func TestScaleNearestGenericDoesNotBoxPixels(t *testing.T) {
	src := image.NewYCbCr(image.Rect(0, 0, 32, 32), image.YCbCrSubsampleRatio420)

	allocs := testing.AllocsPerRun(10, func() {
		_ = scaleNearestGeneric(src, 16, 16)
	})

	if allocs > 4 {
		t.Fatalf("scaleNearestGeneric allocated %.0f times per run, want at most 4", allocs)
	}

	t.Logf("scaleNearestGeneric allocations per 16x16 scale: %.0f", allocs)
}
