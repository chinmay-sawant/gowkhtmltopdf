//nolint:varnamelen,wsl,mnd,nlreturn // multi-layer and inset box-shadow parser and renderer
package layout

import (
	"math"
	"strings"
)

const (
	boxShadowProp       = "box-shadow"
	insetKeyword        = "inset"
	boxShadowMaxTokens  = 6 // inset + x y blur spread + color
	boxShadowMaxLengths = 4 // x y blur spread
	boxShadowBlurSteps  = 4 // stacked expanding fills approximating blur
)

type parsedBoxShadow struct {
	x, y, blur, spread float64
	color              [3]float64
	inset              bool
}

// parseBoxShadowList parses all comma-separated box-shadow layers.
func parseBoxShadowList(value string, current [3]float64, fsize float64) []parsedBoxShadow {
	layers := splitCommaLayers(value)
	var list []parsedBoxShadow
	for _, layer := range layers {
		layer = strings.TrimSpace(layer)
		if layer == "" || strings.EqualFold(layer, cssDisplayNone) {
			continue
		}
		if s, ok := parseBoxShadowLayer(layer, current, fsize); ok {
			list = append(list, s)
		}
	}
	return list
}

// parseBoxShadowLayer reads [inset] offset-x offset-y [blur [spread]] [color].
func parseBoxShadowLayer(value string, current [3]float64, fsize float64) (parsedBoxShadow, bool) {
	var tokens [boxShadowMaxTokens]string

	tokenCount := splitSpaceTokens(value, tokens[:])
	if tokenCount < pairLen || tokenCount > len(tokens) {
		return parsedBoxShadow{}, false //nolint:exhaustruct // invalid layer
	}

	lengths, lengthCount, color, inset, ok := collectBoxShadowTokens(tokens[:tokenCount], current, fsize)
	if !ok {
		return parsedBoxShadow{}, false //nolint:exhaustruct // invalid layer
	}

	return boxShadowFromLengths(lengths, lengthCount, color, inset)
}

func collectBoxShadowTokens(
	tokens []string, current [3]float64, fsize float64,
) ([boxShadowMaxLengths]float64, int, [3]float64, bool, bool) {
	var (
		lengths     [boxShadowMaxLengths]float64
		lengthCount int
		color       = current
		inset       = false
	)

	for _, tok := range tokens {
		if strings.EqualFold(tok, insetKeyword) {
			inset = true

			continue
		}

		if parsed, parsedOK := parseUsedColor(tok, current); parsedOK {
			color = parsed

			continue
		}

		length, lengthOK := plainLength(tok, fsize, 0)
		if !lengthOK || lengthCount >= len(lengths) {
			return lengths, 0, color, false, false
		}

		lengths[lengthCount] = length
		lengthCount++
	}

	return lengths, lengthCount, color, inset, true
}

func boxShadowFromLengths(
	lengths [boxShadowMaxLengths]float64, lengthCount int, color [3]float64, inset bool,
) (parsedBoxShadow, bool) {
	if lengthCount < pairLen {
		return parsedBoxShadow{}, false //nolint:exhaustruct // invalid layer
	}

	blur := 0.0
	spread := 0.0

	if lengthCount >= three {
		if lengths[two] < 0 {
			return parsedBoxShadow{}, false //nolint:exhaustruct // invalid layer
		}

		blur = lengths[two]
	}

	if lengthCount >= boxShadowMaxLengths {
		spread = lengths[three]
	}

	return parsedBoxShadow{x: lengths[0], y: lengths[1], blur: blur, spread: spread, color: color, inset: inset}, true
}

// appendBoxShadow paints all box-shadow layers in reverse order so layer 0 is on top.
func (e *engine) appendBoxShadow(
	dst []Op, sty ResolvedStyle, posX, posY, width, height float64,
	radiusX, radiusY [4]float64,
) []Op {
	if e == nil || !sty.BoxShadowSet || width <= 0 || height <= 0 {
		return dst
	}

	shadows := parseBoxShadowList(sty.BoxShadowRaw, sty.Color, sty.FontSize)
	if len(shadows) == 0 {
		shadows = []parsedBoxShadow{{
			x: sty.BoxShadowX, y: sty.BoxShadowY, blur: sty.BoxShadowBlur,
			spread: sty.BoxShadowSpread, color: sty.BoxShadowColor, inset: sty.BoxShadowInset,
		}}
	}

	for i := len(shadows) - 1; i >= 0; i-- {
		shadow := shadows[i]
		if shadow.inset {
			dst = e.appendInsetBoxShadow(dst, shadow, posX, posY, width, height, radiusX, radiusY)
		} else {
			dst = e.appendOuterBoxShadow(dst, shadow, posX, posY, width, height, radiusX, radiusY)
		}
	}

	return dst
}

func (e *engine) appendOuterBoxShadow(
	dst []Op, shadow parsedBoxShadow, posX, posY, width, height float64,
	radiusX, radiusY [4]float64,
) []Op {
	spread := e.scalePt(shadow.spread)
	originX := posX + e.scalePt(shadow.x) - spread
	originY := posY + e.scalePt(shadow.y) - spread
	shadowW := width + spread*two
	shadowH := height + spread*two

	if shadowW <= 0 || shadowH <= 0 {
		return dst
	}

	dst = appendBoxShadowBlur(
		dst, originX, originY, shadowW, shadowH, e.scalePt(shadow.blur), shadow.color, radiusX, radiusY,
	)

	return append(dst, shadowFillOp(originX, originY, shadowW, shadowH, shadow.color, radiusX, radiusY))
}

func (e *engine) appendInsetBoxShadow(
	dst []Op, shadow parsedBoxShadow, posX, posY, width, height float64,
	radiusX, radiusY [4]float64,
) []Op {
	spread := e.scalePt(shadow.spread)
	blur := e.scalePt(shadow.blur)
	offX := e.scalePt(shadow.x)
	offY := e.scalePt(shadow.y)

	insetDepth := math.Max(spread+blur, 1.0)
	if insetDepth > width/2 {
		insetDepth = width / 2
	}
	if insetDepth > height/2 {
		insetDepth = height / 2
	}

	// Paint inner blur / shadow band inside padding box
	steps := boxShadowBlurSteps
	if blur <= 0 {
		steps = 1
	}

	for step := 1; step <= steps; step++ {
		d := insetDepth * float64(step) / float64(steps)
		alpha := 1.0 - float64(step-1)/float64(steps)
		if blur > 0 {
			alpha *= 0.5
		}

		op := shadowFillOp(
			posX+offX, posY+offY, width, d, shadow.color, radiusX, radiusY,
		)
		op.Alpha = alpha
		dst = append(dst, op)

		if height > d {
			opBot := shadowFillOp(
				posX+offX, posY+height-d+offY, width, d, shadow.color, radiusX, radiusY,
			)
			opBot.Alpha = alpha
			dst = append(dst, opBot)
		}
	}

	return dst
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
