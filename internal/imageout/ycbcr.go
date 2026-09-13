package imageout

import (
	"image"
	"image/color"
)

// nrgbaToYCbCr converts an NRGBA canvas to the YCbCr image that image/jpeg
// encodes through its plane path instead of boxing one color per pixel.
//
// The chroma planes stay at full resolution so image/jpeg's own 4:2:0
// downsampling (its scale step) still averages the same per-pixel values as its
// NRGBA path, which makes jpeg.Encode byte-identical for every size, including
// partial MCUs at odd and even edges. A 4:2:0 preconversion cannot represent
// those edge blocks exactly: the encoder clamps reads to the last chroma
// sample, which for an even-width partial MCU is the average of the last two
// pixels rather than the last pixel alone.
//
// image/jpeg's NRGBA path fills each 8x8 block from m.At(x, y).RGBA() and then
// averages adjacent 2x2 chroma groups per block. This conversion performs the
// same per-pixel math once, so the blocks the encoder builds from the planes
// hold the same values and the same averages follow. Luma and chroma come from
// color.RGBToYCbCr of the premultiplied 8-bit channels, exactly the bytes the
// NRGBA path feeds it.
//
// opaque reports the caller guarantee that every alpha is 255, which skips
// premultiplication; with alpha 255 the premultiplied channels equal the stored
// channels, so the bytes are unchanged. Passing false is always safe.
func nrgbaToYCbCr(src *image.NRGBA, opaque bool) *image.YCbCr {
	bounds := src.Bounds()
	dst := image.NewYCbCr(bounds, image.YCbCrSubsampleRatio444)

	width, height := bounds.Dx(), bounds.Dy()
	if width == 0 || height == 0 {
		return dst
	}

	srcPix := src.Pix
	srcStride := src.Stride
	yStride := dst.YStride
	cStride := dst.CStride

	for row := range height {
		rowPix := srcPix[row*srcStride:]
		yOffset := row * yStride
		cOffset := row * cStride

		for col := range width {
			pixel := rowPix[col*4:]
			red, green, blue := pixel[0], pixel[1], pixel[2]

			if !opaque && pixel[3] != opaqueAlpha {
				red = premultiplied8(red, pixel[3])
				green = premultiplied8(green, pixel[3])
				blue = premultiplied8(blue, pixel[3])
			}

			luma, cb, cr := color.RGBToYCbCr(red, green, blue)
			dst.Y[yOffset+col] = luma
			dst.Cb[cOffset+col] = cb
			dst.Cr[cOffset+col] = cr
		}
	}

	return dst
}

// premultiplied8 returns the high byte of the 16-bit premultiplied channel
// color.NRGBA.RGBA produces, which is the byte image/jpeg's toYCbCr feeds to
// color.RGBToYCbCr.
func premultiplied8(channel, alpha uint8) uint8 {
	value := uint32(channel)
	value |= value << channelShift
	value *= uint32(alpha)
	value /= opaqueAlpha

	return uint8(value >> channelShift) //nolint:gosec // 16-bit premultiplied channel scaled to 8 bits
}
