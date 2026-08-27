package layout

import "strings"

const (
	boxShadowProp       = "box-shadow"
	insetKeyword        = "inset"
	boxShadowMaxTokens  = 6 // inset + x y blur spread + color
	boxShadowMaxLengths = 4 // x y blur spread
	boxShadowBlurSteps  = 4 // stacked expanding fills approximating blur
)

type parsedBoxShadow struct {
	x, y, blur float64
	color      [3]float64
}

// parseBoxShadowLayer reads offset-x offset-y [blur [spread]] [color].
// Inset layers are dropped. Spread is accepted and ignored. Blur is stored
// and painted as stacked expanding fills with decreasing alpha.
func parseBoxShadowLayer(value string, current [3]float64, fsize float64) (parsedBoxShadow, bool) {
	var tokens [boxShadowMaxTokens]string

	tokenCount := splitSpaceTokens(value, tokens[:])
	if tokenCount < pairLen || tokenCount > len(tokens) {
		return parsedBoxShadow{}, false //nolint:exhaustruct // invalid layer
	}

	lengths, lengthCount, color, ok := collectBoxShadowTokens(tokens[:tokenCount], current, fsize)
	if !ok {
		return parsedBoxShadow{}, false //nolint:exhaustruct // invalid layer
	}

	return boxShadowFromLengths(lengths, lengthCount, color)
}

func collectBoxShadowTokens(
	tokens []string, current [3]float64, fsize float64,
) ([boxShadowMaxLengths]float64, int, [3]float64, bool) {
	var (
		lengths     [boxShadowMaxLengths]float64
		lengthCount int
		color       = current
	)

	for _, tok := range tokens {
		if strings.EqualFold(tok, insetKeyword) {
			return lengths, 0, color, false
		}

		if parsed, parsedOK := parseUsedColor(tok, current); parsedOK {
			color = parsed

			continue
		}

		length, lengthOK := plainLength(tok, fsize, 0)
		if !lengthOK || lengthCount >= len(lengths) {
			return lengths, 0, color, false
		}

		lengths[lengthCount] = length
		lengthCount++
	}

	return lengths, lengthCount, color, true
}

func boxShadowFromLengths(
	lengths [boxShadowMaxLengths]float64, lengthCount int, color [3]float64,
) (parsedBoxShadow, bool) {
	if lengthCount < pairLen {
		return parsedBoxShadow{}, false //nolint:exhaustruct // invalid layer
	}

	blur := 0.0

	if lengthCount >= three {
		if lengths[two] < 0 {
			return parsedBoxShadow{}, false //nolint:exhaustruct // invalid layer
		}

		blur = lengths[two]
	}

	return parsedBoxShadow{x: lengths[0], y: lengths[1], blur: blur, color: color}, true
}

// appendBoxShadow paints one opaque fill the size of the border box, offset
// by BoxShadowX/Y. When blur > 0 it first paints stacked expanding fills with
// decreasing alpha. Layout size is unchanged. Inset and spread stay ignored.
func (e *engine) appendBoxShadow(
	dst []Op, sty ResolvedStyle, posX, posY, width, height float64,
	radiusX, radiusY [4]float64,
) []Op {
	if e == nil || !sty.BoxShadowSet || width <= 0 || height <= 0 {
		return dst
	}

	originX := posX + e.scalePt(sty.BoxShadowX)
	originY := posY + e.scalePt(sty.BoxShadowY)
	dst = appendBoxShadowBlur(
		dst, originX, originY, width, height, e.scalePt(sty.BoxShadowBlur), sty.BoxShadowColor, radiusX, radiusY,
	)

	return append(dst, shadowFillOp(originX, originY, width, height, sty.BoxShadowColor, radiusX, radiusY))
}

func shadowFillOp(
	x, y, width, height float64, color [3]float64, radiusX, radiusY [4]float64,
) Op {
	return Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpFillRect, X: x, Y: y, W: width, H: height,
		R: color[0], G: color[1], B: color[2],
		Radius: uniformRadius(radiusX), RadiusY: uniformRadius(radiusY),
		RadiusTopLeft: radiusX[0], RadiusTopRight: radiusX[1],
		RadiusBottomRight: radiusX[2], RadiusBottomLeft: radiusX[3],
		RadiusTopLeftY: radiusY[0], RadiusTopRightY: radiusY[1],
		RadiusBottomRightY: radiusY[2], RadiusBottomLeftY: radiusY[3],
	}
}

// appendBoxShadowBlur approximates CSS blur with expanding rects. Outer rings
// use lower alpha; the caller paints the opaque core afterward. PDF StyleOf
// pre-composites translucent fills against white, so the rings read as
// stepped gray. This is not a Gaussian raster of descendants.
func appendBoxShadowBlur(
	dst []Op, originX, originY, width, height, blur float64, color [3]float64,
	radiusX, radiusY [4]float64,
) []Op {
	if blur <= 0 {
		return dst
	}

	for step := boxShadowBlurSteps; step >= 1; step-- {
		expand := blur * float64(step) / float64(boxShadowBlurSteps)
		alpha := float64(boxShadowBlurSteps-step+1) / float64(boxShadowBlurSteps+1)
		op := shadowFillOp(
			originX-expand, originY-expand, width+expand*two, height+expand*two, color, radiusX, radiusY,
		)
		op.Alpha = alpha
		dst = append(dst, op)
	}

	return dst
}
