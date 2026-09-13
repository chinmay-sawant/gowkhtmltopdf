package imageout

import (
	"image"
	"image/draw"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

// paintImage draws a decoded paintOp image, scaled via the per-run cache.
//
//nolint:cyclop // raster image painting pipeline
func paintImage(img *image.NRGBA, paintOp *layout.Op, pxPerPt float64, imageCache *rasterImageCache) {
	decoded, err := imageCache.decode(paintOp)
	if err != nil || paintOp.W <= 0 || paintOp.H <= 0 {
		return // layout already validated the bytes; skip on failure
	}

	src := decoded.image
	fullRect := ptRectScale(paintOp.X, paintOp.Y, paintOp.W, paintOp.H, pxPerPt)
	rect := fullRect.Intersect(img.Bounds())

	if rect.Empty() {
		return
	}

	// Scale to the unclipped dest size, then draw the visible window. A strip
	// (or canvas edge) that cuts the dest must show a slice of the full-size
	// image, not a squashed copy fitted to the clip.
	srcPoint := image.Pt(rect.Min.X-fullRect.Min.X, rect.Min.Y-fullRect.Min.Y)
	sb := src.Bounds()

	if fullRect.Dx() == sb.Dx() && fullRect.Dy() == sb.Dy() {
		srcPoint = srcPoint.Add(sb.Min)
		if nrgba, ok := src.(*image.NRGBA); ok && nrgba.Opaque() {
			drawNRGBAOpaque(img, rect, nrgba, srcPoint)
		} else {
			draw.Draw(img, rect, src, srcPoint, draw.Over)
		}

		return
	}

	// A clipped draw whose full scaled canvas can never fit the scaled-image
	// cache re-scales it for every strip; scale only the visible window
	// instead, with the same sampling grid (scaleNearestWindow).
	if rect != fullRect && scaledCanvasExceedsCache(fullRect) {
		window := scaleNearestWindow(src, fullRect, rect)
		if window == nil {
			return
		}

		if window.Opaque() {
			drawNRGBAOpaque(img, rect, window, window.Bounds().Min)
		} else {
			draw.Draw(img, rect, window, window.Bounds().Min, draw.Over)
		}

		return
	}

	// Go 1.26 removed image/draw's scalers; nearest
	// neighbour keeps it stdlib-only.
	scaled := imageCache.scaledImage(decoded, fullRect.Dx(), fullRect.Dy())
	if scaled == nil {
		return
	}

	if scaled.Opaque() {
		drawNRGBAOpaque(img, rect, scaled, srcPoint)
	} else {
		draw.Draw(img, rect, scaled, srcPoint, draw.Over)
	}
}

// scaledCanvasExceedsCache reports whether a full-size scaled canvas is larger
// than rasterImageCache can ever store, so a clipped draw gains nothing from
// building it.
func scaledCanvasExceedsCache(full image.Rectangle) bool {
	return int64(full.Dx())*int64(full.Dy())*nrgbaBytes > maxScaledCacheBytes
}
