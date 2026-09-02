//nolint:wsl // alpha-channel arithmetic follows the source-over compositing formula
package imageout

import (
	"image"
	"math"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

func paintBlended(
	dst *image.NRGBA,
	paintOp *layout.Op,
	pxPerPt float64,
	atlas *glyphAtlas,
	imageCache *rasterImageCache,
) {
	source := image.NewNRGBA(dst.Bounds())
	opCopy := *paintOp
	opCopy.BlendMode = ""
	paint(source, &opCopy, pxPerPt, atlas, imageCache)
	compositeBlend(dst, source, paintOp.BlendMode)
}

func compositeBlend(dst, source *image.NRGBA, mode string) {
	bounds := dst.Bounds().Intersect(source.Bounds())
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			blendPixel(dst, source, x, y, mode)
		}
	}
}

func blendPixel(dst, source *image.NRGBA, x, y int, mode string) {
	dstOffset := dst.PixOffset(x, y)
	srcOffset := source.PixOffset(x, y)

	sourceAlpha := float64(source.Pix[srcOffset+3]) / channelMax
	if sourceAlpha <= 0 {
		return
	}
	destAlpha := float64(dst.Pix[dstOffset+3]) / channelMax
	backdrop := [3]float64{
		float64(dst.Pix[dstOffset]) / channelMax,
		float64(dst.Pix[dstOffset+1]) / channelMax,
		float64(dst.Pix[dstOffset+2]) / channelMax,
	}
	sourceColor := [3]float64{
		float64(source.Pix[srcOffset]) / channelMax,
		float64(source.Pix[srcOffset+1]) / channelMax,
		float64(source.Pix[srcOffset+2]) / channelMax,
	}
	blended := layout.BlendColor(mode, backdrop, sourceColor)
	outAlpha := sourceAlpha + destAlpha*(1-sourceAlpha)
	if outAlpha <= 0 {
		dst.Pix[dstOffset+0] = 0
		dst.Pix[dstOffset+1] = 0
		dst.Pix[dstOffset+2] = 0
		dst.Pix[dstOffset+3] = 0

		return
	}

	for index := range blended {
		outPremultiplied := (1-sourceAlpha)*destAlpha*backdrop[index] +
			(1-destAlpha)*sourceAlpha*sourceColor[index] + sourceAlpha*destAlpha*blended[index]
		dst.Pix[dstOffset+index] = uint8(math.Round(clampBlend(outPremultiplied/outAlpha) * channelMax))
	}
	dst.Pix[dstOffset+3] = uint8(math.Round(outAlpha * channelMax))
}

func clampBlend(value float64) float64 {
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}

	return value
}
