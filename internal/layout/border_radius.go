package layout

import (
	"math"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// splitRadiusSlash splits CSS border-radius slash syntax into the horizontal
// (rx) side and the optional vertical (ry) side.
func splitRadiusSlash(value string) (string, string, bool) {
	before, after, ok := strings.Cut(value, "/")
	horiz := strings.TrimSpace(before)

	if !ok {
		return horiz, "", false
	}

	return horiz, strings.TrimSpace(after), true
}

// splitCornerRadiusTokens reads one longhand: `rx`, `rx ry`, or `rx / ry`.
func splitCornerRadiusTokens(value string) (string, string, bool) {
	horiz, vert, hasVert := splitRadiusSlash(value)
	if hasVert {
		tokenX, _, _ := nextSpaceToken(horiz, 0)
		tokenY, _, hasY := nextSpaceToken(vert, 0)

		return tokenX, tokenY, hasY
	}

	tokenX, next, ok := nextSpaceToken(horiz, 0)
	if !ok {
		return "", "", false
	}

	tokenY, _, hasY := nextSpaceToken(horiz, next)

	return tokenX, tokenY, hasY
}

func setBorderRadius(style *ResolvedStyle, value string, fsize float64) bool {
	horiz, vert, hasVert := splitRadiusSlash(value)
	if horiz == "" {
		return true
	}

	if radiusListHasPercent(horiz) || radiusListHasPercent(vert) {
		applyUniformRadiusPercent(style, firstRadiusPercent(horiz, vert))

		return true
	}

	xs, ok := parseRadiusLengths(horiz, fsize)
	if !ok {
		return true
	}

	assignCornerRadiiX(style, expandRadius4(xs))

	if !hasVert || vert == "" {
		clearCornerRadiiY(style)

		return true
	}

	ys, yok := parseRadiusLengths(vert, fsize)
	if !yok {
		return true
	}

	assignCornerRadiiY(style, expandRadius4(ys))

	return true
}

func radiusListHasPercent(value string) bool {
	for start := 0; ; {
		token, next, ok := nextSpaceToken(value, start)
		if !ok {
			return false
		}

		if _, unit, parsed := css.ParseLength(token); parsed && unit == "%" {
			return true
		}

		start = next
	}
}

func firstRadiusPercent(horiz, vert string) float64 {
	for _, side := range []string{horiz, vert} {
		for start := 0; ; {
			token, next, ok := nextSpaceToken(side, start)
			if !ok {
				break
			}

			if percent, unit, parsed := css.ParseLength(token); parsed && unit == "%" && percent >= 0 {
				return percent
			}

			start = next
		}
	}

	return 0
}

func applyUniformRadiusPercent(style *ResolvedStyle, percent float64) {
	style.BorderRadius = 0
	style.BorderRadiusPercent = percent
	clearCornerRadiiY(style)
}

func parseRadiusLengths(value string, fsize float64) ([]float64, bool) {
	values := make([]float64, 0, borderRadiusValueCount)

	for start := 0; ; {
		token, next, ok := nextSpaceToken(value, start)
		if !ok {
			break
		}

		radius, parsed := lengthBox(token, fsize, 0, cssDisplayNone)
		if !parsed || radius < 0 {
			return nil, false
		}

		values = append(values, radius)
		start = next
	}

	if len(values) == 0 {
		return nil, false
	}

	return values, true
}

func expandRadius4(values []float64) []float64 {
	out := append([]float64(nil), values...)

	for len(out) < borderRadiusValueCount {
		switch len(out) {
		case 1:
			out = append(out, out[0], out[0], out[0])
		case borderRadiusPairCount:
			out = append(out, out[0], out[1])
		case borderRadiusTripleCount:
			out = append(out, out[1])
		}
	}

	return out[:borderRadiusValueCount]
}

func assignCornerRadiiX(style *ResolvedStyle, values []float64) {
	style.BorderRadiusTopLeft = values[0]
	style.BorderRadiusTopRight = values[1]
	style.BorderRadiusBottomRight = values[2]
	style.BorderRadiusBottomLeft = values[3]
	style.BorderRadiusPercent = -1

	if values[0] == values[1] && values[1] == values[2] && values[2] == values[3] {
		style.BorderRadius = values[0]
	} else {
		style.BorderRadius = 0
	}
}

func assignCornerRadiiY(style *ResolvedStyle, values []float64) {
	style.BorderRadiusTopLeftY = values[0]
	style.BorderRadiusTopRightY = values[1]
	style.BorderRadiusBottomRightY = values[2]
	style.BorderRadiusBottomLeftY = values[3]
}

func clearCornerRadiiY(style *ResolvedStyle) {
	style.BorderRadiusTopLeftY = 0
	style.BorderRadiusTopRightY = 0
	style.BorderRadiusBottomRightY = 0
	style.BorderRadiusBottomLeftY = 0
}

const (
	borderRadiusPercentBasis = 100.0
	borderRadiusHalf         = 2.0
)

// usedBorderRadiiXY resolves both axes of a box's corner ellipses together.
// Percent radii keep the existing uniform-circle behavior; an explicit
// vertical radius is resolved only for absolute longhand or slash values.
func usedBorderRadiiXY(sty ResolvedStyle, width, height float64) ([4]float64, [4]float64) {
	radiusX := borderRadiusValues(sty, width, height)
	clampBorderRadii(radiusX[:], width, height)
	scaleBorderRadii(radiusX[:], width, height)

	var radiusY [4]float64
	if sty.BorderRadiusPercent < 0 {
		radiusY = [4]float64{
			sty.BorderRadiusTopLeftY, sty.BorderRadiusTopRightY,
			sty.BorderRadiusBottomRightY, sty.BorderRadiusBottomLeftY,
		}
		clampBorderRadiiY(radiusY[:], height)
		scaleBorderRadiiY(radiusY[:], height)
	}

	return radiusX, radiusY
}

func usedBorderRadii(sty ResolvedStyle, width, height float64) [4]float64 {
	radiusX, _ := usedBorderRadiiXY(sty, width, height)

	return radiusX
}

func usedBorderRadius(sty ResolvedStyle, width, height float64) float64 {
	return uniformRadius(usedBorderRadii(sty, width, height))
}

func borderRadiusValues(sty ResolvedStyle, width, height float64) [4]float64 {
	var radii [4]float64

	switch {
	case sty.BorderRadiusPercent >= 0:
		radius := math.Min(width, height) * sty.BorderRadiusPercent / borderRadiusPercentBasis
		for i := range radii {
			radii[i] = radius
		}
	case sty.BorderRadiusTopLeft != 0 || sty.BorderRadiusTopRight != 0 ||
		sty.BorderRadiusBottomRight != 0 || sty.BorderRadiusBottomLeft != 0:
		radii = [4]float64{
			sty.BorderRadiusTopLeft, sty.BorderRadiusTopRight,
			sty.BorderRadiusBottomRight, sty.BorderRadiusBottomLeft,
		}
	default:
		for i := range radii {
			radii[i] = sty.BorderRadius
		}
	}

	return radii
}

//nolint:wsl // CSS radius clamping is a compact geometry loop
func clampBorderRadii(radii []float64, width, height float64) {
	short := math.Min(width, height)

	for i := range radii {
		if radii[i] < 0 {
			radii[i] = 0
		}
		if radii[i] > short/borderRadiusHalf {
			radii[i] = short / borderRadiusHalf
		}
	}
}

//nolint:wsl,mnd // CSS adjacent-radius scaling is expressed as four edge sums
func scaleBorderRadii(radii []float64, width, height float64) {
	if len(radii) < 4 {
		return
	}

	scale := 1.0
	for _, edge := range []struct {
		sum   float64
		limit float64
	}{
		{sum: radii[0] + radii[1], limit: width},
		{sum: radii[3] + radii[2], limit: width},
		{sum: radii[0] + radii[3], limit: height},
		{sum: radii[1] + radii[2], limit: height},
	} {
		if edge.sum > edge.limit && edge.sum > 0 && edge.limit/edge.sum < scale {
			scale = edge.limit / edge.sum
		}
	}
	if scale < 1 {
		for i := range radii {
			radii[i] *= scale
		}
	}
}

func clampBorderRadiiY(radii []float64, height float64) {
	limit := height / borderRadiusHalf

	for idx := range radii {
		if radii[idx] < 0 {
			radii[idx] = 0
		}

		if radii[idx] > limit {
			radii[idx] = limit
		}
	}
}

func scaleBorderRadiiY(radii []float64, height float64) {
	if len(radii) < borderRadiusValueCount {
		return
	}

	scale := 1.0

	for _, sum := range []float64{radii[0] + radii[3], radii[1] + radii[2]} {
		if sum > height && sum > 0 && height/sum < scale {
			scale = height / sum
		}
	}

	if scale < 1 {
		for idx := range radii {
			radii[idx] *= scale
		}
	}
}

func stampOpRadiiY(ops []Op, radiiY [4]float64) {
	for idx := range ops {
		stampOneOpRadiiY(&ops[idx], radiiY)
	}
}

func stampOneOpRadiiY(paintOp *Op, radiiY [4]float64) {
	if paintOp.Kind != OpFillRect && paintOp.Kind != OpStrokeRect {
		return
	}

	if paintOp.RadiusTopLeft > 0 {
		paintOp.RadiusTopLeftY = radiiY[0]
	}

	if paintOp.RadiusTopRight > 0 {
		paintOp.RadiusTopRightY = radiiY[1]
	}

	if paintOp.RadiusBottomRight > 0 {
		paintOp.RadiusBottomRightY = radiiY[2]
	}

	if paintOp.RadiusBottomLeft > 0 {
		paintOp.RadiusBottomLeftY = radiiY[3]
	}

	paintOp.RadiusY = uniformRadius([4]float64{
		paintOp.RadiusTopLeftY, paintOp.RadiusTopRightY,
		paintOp.RadiusBottomRightY, paintOp.RadiusBottomLeftY,
	})
}

func opRadiiY(paintOp *Op) [4]float64 {
	if paintOp.RadiusTopLeftY == 0 && paintOp.RadiusTopRightY == 0 &&
		paintOp.RadiusBottomRightY == 0 && paintOp.RadiusBottomLeftY == 0 {
		if paintOp.RadiusY > 0 {
			return [4]float64{paintOp.RadiusY, paintOp.RadiusY, paintOp.RadiusY, paintOp.RadiusY}
		}

		return [4]float64{}
	}

	return [4]float64{
		paintOp.RadiusTopLeftY, paintOp.RadiusTopRightY,
		paintOp.RadiusBottomRightY, paintOp.RadiusBottomLeftY,
	}
}

func opRadiiXY(paintOp *Op) ([4]float64, [4]float64) {
	radiusX := opRadii(paintOp)
	radiusY := opRadiiY(paintOp)

	for idx := range radiusX {
		if radiusX[idx] <= 0 {
			radiusY[idx] = 0

			continue
		}

		if radiusY[idx] <= 0 {
			radiusY[idx] = radiusX[idx]
		}
	}

	return radiusX, radiusY
}
