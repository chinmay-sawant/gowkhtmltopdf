package imageout

import (
	"image"
	"image/color"
)

// channelShift converts the 16-bit components color.NRGBA.RGBA returns into
// the 8-bit components color.RGBToYCbCr takes, exactly like image/jpeg does.
const channelShift = 8

// nrgbaToYCbCr420 converts an NRGBA image to the 4:2:0 YCbCr planes
// image/jpeg reads directly. The conversion reproduces image/jpeg's own NRGBA
// path bit for bit: luma uses the same premultiplied RGBA values fed through
// color.RGBToYCbCr, and each chroma sample is the rounded 2x2 source box
// average (sum+2)>>2 that the encoder's scale pass computes, with source
// coordinates clamped to the last row and column exactly like toYCbCr.
//
// The 4:2:0 sample grid stores one chroma sample per 2x2 source box, which
// matches the encoder's chroma indexing only when the rectangle origin has
// even coordinates. Odd origins cannot be represented (two source pixels
// share one chroma index), so nil is returned and callers keep the encoder's
// per-pixel conversion for exact output.
func nrgbaToYCbCr420(src *image.NRGBA) *image.YCbCr {
	bounds := src.Bounds()
	if bounds.Min.X%boxFilterFactor2 != 0 || bounds.Min.Y%boxFilterFactor2 != 0 {
		return nil
	}

	dst := image.NewYCbCr(bounds, image.YCbCrSubsampleRatio420)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			luma, _, _ := ycbcrFromNRGBA(src.NRGBAAt(x, y))
			dst.Y[dst.YOffset(x, y)] = luma
		}
	}

	xmax := bounds.Max.X - 1
	ymax := bounds.Max.Y - 1
	index := 0

	for row := range (bounds.Dy() + 1) / boxFilterFactor2 {
		baseY := bounds.Min.Y + row*boxFilterFactor2

		for col := range (bounds.Dx() + 1) / boxFilterFactor2 {
			baseX := bounds.Min.X + col*boxFilterFactor2

			var sumCb, sumCr int

			for dy := range boxFilterFactor2 {
				sampleY := min(baseY+dy, ymax)

				for dx := range boxFilterFactor2 {
					sampleX := min(baseX+dx, xmax)

					_, cb, cr := ycbcrFromNRGBA(src.NRGBAAt(sampleX, sampleY))
					sumCb += int(cb)
					sumCr += int(cr)
				}
			}

			// Same rounding as image/jpeg's scale: (sum+2)>>2 of a 2x2 box.
			//nolint:gosec // a 2x2 box sum is <= 1020, so the average fits uint8
			dst.Cb[index] = uint8((sumCb + boxFilterFactor2) >> boxFilterFactor2)
			//nolint:gosec // a 2x2 box sum is <= 1020, so the average fits uint8
			dst.Cr[index] = uint8((sumCr + boxFilterFactor2) >> boxFilterFactor2)
			index++
		}
	}

	return dst
}

// ycbcrFromNRGBA returns the Y'CbCr triple for one NRGBA pixel using the same
// premultiplied math image/jpeg's toYCbCr applies (color.NRGBA.RGBA followed
// by color.RGBToYCbCr).
func ycbcrFromNRGBA(pixel color.NRGBA) (uint8, uint8, uint8) {
	red, green, blue, _ := pixel.RGBA()

	//nolint:gosec // RGBA() components are 16-bit; >>8 yields one byte each
	luma, cb, cr := color.RGBToYCbCr(uint8(red>>channelShift), uint8(green>>channelShift), uint8(blue>>channelShift))

	return luma, cb, cr
}
