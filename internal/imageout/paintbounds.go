package imageout

import (
	"image"
	"math"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

const (
	// paintOpMarginPt is the point-space margin every op edge keeps for stroke
	// width and antialiasing.
	paintOpMarginPt = 2
	// paintOpMarginSpanPt is the total margin the two opposite edges add to one
	// dimension, derived from paintOpMarginPt.
	paintOpMarginSpanPt = 2 * paintOpMarginPt
	// textDownwardGrowth is the fraction of the font size text paints below its
	// box, the asymmetric growth paintTransformedOp assumes.
	textDownwardGrowth = 0.5
)

// paintOpBounds returns the conservative canvas-pixel rectangle an op can
// paint into before clipping. Text grows upward by one font size and downward
// by half (the asymmetric box paintTransformedOp assumes), every edge keeps a
// 2pt margin for stroke width and antialiasing, and a set transform
// contributes its transformed corner extent. Callers intersect the result
// with the target canvas; an empty intersection means the op cannot paint.
func paintOpBounds(paintOp *layout.Op, pxPerPt float64) image.Rectangle {
	paintOp.BindEmptyExtra()

	minX, minY, maxX, maxY := opRectBounds(paintOp)

	if paintOp.XformSet && !paintOp.Xform.IsIdentity() {
		corners := [4][2]float64{
			{minX, minY},
			{maxX, minY},
			{maxX, maxY},
			{minX, maxY},
		}
		minX, minY = math.MaxFloat64, math.MaxFloat64
		maxX, maxY = -math.MaxFloat64, -math.MaxFloat64

		for _, c := range corners {
			tx, ty := paintOp.Xform.Apply(c[0], c[1])
			minX = math.Min(minX, tx)
			minY = math.Min(minY, ty)
			maxX = math.Max(maxX, tx)
			maxY = math.Max(maxY, ty)
		}
	}

	return ptRectScale(
		minX-paintOpMarginPt,
		minY-paintOpMarginPt,
		(maxX-minX)+paintOpMarginSpanPt,
		(maxY-minY)+paintOpMarginSpanPt,
		pxPerPt,
	)
}

// opRectBounds returns the untransformed point-space op rectangle before the
// paint margin: text expands upward by one font size and downward by half and
// every degenerate dimension clamps to one point so the rectangle never
// collapses.
func opRectBounds(paintOp *layout.Op) (float64, float64, float64, float64) {
	minX, minY := paintOp.X, paintOp.Y
	maxX, maxY := paintOp.X+paintOp.W, paintOp.Y+paintOp.H

	if paintOp.Kind == layout.OpText || paintOp.Kind == layout.OpBullet {
		minY = paintOp.Y - paintOp.Size
		maxY = paintOp.Y + paintOp.Size*textDownwardGrowth

		if paintOp.W <= 0 {
			maxX = paintOp.X + paintOp.Size*float64(len(paintOp.Text))
		}
	}

	if maxX <= minX {
		maxX = minX + 1
	}

	if maxY <= minY {
		maxY = minY + 1
	}

	return minX, minY, maxX, maxY
}
