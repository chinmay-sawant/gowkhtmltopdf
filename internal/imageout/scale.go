package imageout

import (
	"image"
	"image/color"
)

// opaque16 is the fully opaque 16-bit channel value color.NRGBA.RGBA returns.
const opaque16 = 0xffff

// channelShift converts the 16-bit channel values color.Color.RGBA returns
// into the 8-bit channels the NRGBA model stores, matching the model's
// truncating shift.
const channelShift = 8

// scaleNearest resizes src to w×h with nearest-neighbour sampling. Go 1.26
// removed image/draw's BiLinear/NearestNeighbor scalers, so a tiny scaler
// lives here; natural-size images take the draw.Draw fast path in paint.
func scaleNearest(src image.Image, w, h int) *image.NRGBA {
	full := image.Rect(0, 0, w, h)

	return scaleNearestWindow(src, full, full)
}

// scaleNearestWindow returns only window of the nearest-neighbour scale that
// scaleNearest(src, full.Dx(), full.Dy()) would produce. A clipped draw whose
// full scaled canvas exceeds the scaled-image cache uses it so the strip
// raster never rebuilds the same oversized canvas per strip; the sampling grid
// and clamping are the same, so window pixels byte-match the full-scale crop.
func scaleNearestWindow(src image.Image, full, window image.Rectangle) *image.NRGBA {
	if nrgba, ok := src.(*image.NRGBA); ok {
		return scaleNearestNRGBAWindow(nrgba, full, window)
	}

	return scaleNearestGenericWindow(src, full, window)
}

func scaleNearestNRGBAWindow(src *image.NRGBA, full, window image.Rectangle) *image.NRGBA {
	dst, err := newRasterImage(window.Dx(), window.Dy())
	if err != nil {
		return nil
	}

	srcBounds := src.Bounds()

	if srcBounds.Dx() == 0 || srcBounds.Dy() == 0 {
		return dst
	}

	scaleX := float64(srcBounds.Dx()) / float64(full.Dx())
	scaleY := float64(srcBounds.Dy()) / float64(full.Dy())
	offsetX := window.Min.X - full.Min.X
	offsetY := window.Min.Y - full.Min.Y

	for row := range window.Dy() {
		srcY := srcBounds.Min.Y + int((float64(offsetY+row)+pixelCenter)*scaleY)
		if srcY > srcBounds.Max.Y-1 {
			srcY = srcBounds.Max.Y - 1
		}

		for col := range window.Dx() {
			srcX := srcBounds.Min.X + int((float64(offsetX+col)+pixelCenter)*scaleX)
			if srcX > srcBounds.Max.X-1 {
				srcX = srcBounds.Max.X - 1
			}

			srcOffset := src.PixOffset(srcX, srcY)
			dstOffset := dst.PixOffset(col, row)
			copy(dst.Pix[dstOffset:dstOffset+4], src.Pix[srcOffset:srcOffset+4])
		}
	}

	return dst
}

// scaleNearestGeneric scales sources without a Pix copy fast path. Concrete
// YCbCr, NRGBA, and RGBA sources convert through their own accessors with the
// exact color.NRGBAModel math; every other image type keeps
// color.NRGBAModel.Convert as the fallback. All paths produce byte-identical
// output, and the concrete paths avoid boxing one color per sampled pixel.
func scaleNearestGeneric(src image.Image, w, h int) *image.NRGBA {
	full := image.Rect(0, 0, w, h)

	return scaleNearestGenericWindow(src, full, full)
}

// scaleNearestGenericWindow is scaleNearestGeneric for a window of the full
// scale; scaleNearestWindow routes every non-NRGBA source here.
func scaleNearestGenericWindow(src image.Image, full, window image.Rectangle) *image.NRGBA {
	dst, err := newRasterImage(window.Dx(), window.Dy())
	if err != nil {
		return nil
	}

	srcBounds := src.Bounds()

	if srcBounds.Dx() == 0 || srcBounds.Dy() == 0 {
		return dst
	}

	switch typed := src.(type) {
	case *image.YCbCr:
		scaleNearestSampled(dst, srcBounds, full, window, func(x, y int) color.NRGBA {
			return nrgbaFromYCbCr(typed.YCbCrAt(x, y))
		})

	case *image.NRGBA:
		scaleNearestSampled(dst, srcBounds, full, window, func(x, y int) color.NRGBA {
			return typed.NRGBAAt(x, y)
		})

	case *image.RGBA:
		scaleNearestSampled(dst, srcBounds, full, window, func(x, y int) color.NRGBA {
			return nrgbaFromRGBA(typed.RGBAAt(x, y))
		})

	default:
		scaleNearestSampled(dst, srcBounds, full, window, func(x, y int) color.NRGBA {
			converted, ok := color.NRGBAModel.Convert(src.At(x, y)).(color.NRGBA)
			if !ok {
				var zero color.NRGBA

				return zero
			}

			return converted
		})
	}

	return dst
}

// scaleNearestSampled writes the nearest-neighbour sample of srcBounds for
// every pixel of window in the full-scale destination grid, resolving the
// source pixel through sample.
func scaleNearestSampled(
	dst *image.NRGBA, srcBounds, full, window image.Rectangle, sample func(x, y int) color.NRGBA,
) {
	scaleX := float64(srcBounds.Dx()) / float64(full.Dx())
	scaleY := float64(srcBounds.Dy()) / float64(full.Dy())
	offsetX := window.Min.X - full.Min.X
	offsetY := window.Min.Y - full.Min.Y

	for row := range window.Dy() {
		srcY := srcBounds.Min.Y + int((float64(offsetY+row)+pixelCenter)*scaleY)
		if srcY > srcBounds.Max.Y-1 {
			srcY = srcBounds.Max.Y - 1
		}

		for col := range window.Dx() {
			srcX := srcBounds.Min.X + int((float64(offsetX+col)+pixelCenter)*scaleX)
			if srcX > srcBounds.Max.X-1 {
				srcX = srcBounds.Max.X - 1
			}

			dst.SetNRGBA(col, row, sample(srcX, srcY))
		}
	}
}

// nrgbaFromYCbCr converts one YCbCr pixel with the color.NRGBAModel formula.
// YCbCr colors are opaque, so the model takes the a == 0xffff branch.
func nrgbaFromYCbCr(c color.YCbCr) color.NRGBA {
	red, green, blue, _ := c.RGBA()

	return color.NRGBA{
		R: uint8(red >> channelShift),   //nolint:gosec // 16-bit channel scaled to 8 bits
		G: uint8(green >> channelShift), //nolint:gosec // 16-bit channel scaled to 8 bits
		B: uint8(blue >> channelShift),  //nolint:gosec // 16-bit channel scaled to 8 bits
		A: opaqueAlpha,
	}
}

// nrgbaFromRGBA converts one premultiplied RGBA pixel with the exact
// color.NRGBAModel formula, without boxing the color into an interface.
func nrgbaFromRGBA(c color.RGBA) color.NRGBA {
	red, green, blue, alpha := c.RGBA()
	if alpha == opaque16 {
		return color.NRGBA{
			R: uint8(red >> channelShift),   //nolint:gosec // 16-bit channel scaled to 8 bits
			G: uint8(green >> channelShift), //nolint:gosec // 16-bit channel scaled to 8 bits
			B: uint8(blue >> channelShift),  //nolint:gosec // 16-bit channel scaled to 8 bits
			A: opaqueAlpha,
		}
	}

	if alpha == 0 {
		var zero color.NRGBA

		return zero
	}

	red = (red * opaque16) / alpha
	green = (green * opaque16) / alpha
	blue = (blue * opaque16) / alpha

	return color.NRGBA{
		R: uint8(red >> channelShift),   //nolint:gosec // 16-bit channel scaled to 8 bits
		G: uint8(green >> channelShift), //nolint:gosec // 16-bit channel scaled to 8 bits
		B: uint8(blue >> channelShift),  //nolint:gosec // 16-bit channel scaled to 8 bits
		A: uint8(alpha >> channelShift), //nolint:gosec // 16-bit channel scaled to 8 bits
	}
}
