//nolint:varnamelen,wsl,mnd,nlreturn // multi-layer and inset box-shadow parser and renderer
package layout

import (
	"math"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
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
	alpha              float64
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
	if tokenCount < 2 || tokenCount > len(tokens) {
		return parsedBoxShadow{}, false //nolint:exhaustruct // invalid layer
	}

	lengths, lengthCount, color, alpha, inset, ok := collectBoxShadowTokens(tokens[:tokenCount], current, fsize)
	if !ok {
		return parsedBoxShadow{}, false //nolint:exhaustruct // invalid layer
	}

	return boxShadowFromLengths(lengths, lengthCount, color, alpha, inset)
}

func collectBoxShadowTokens(
	tokens []string, current [3]float64, fsize float64,
) ([boxShadowMaxLengths]float64, int, [3]float64, float64, bool, bool) {
	var (
		lengths     [boxShadowMaxLengths]float64
		lengthCount int
		color       = current
		alpha       = 1.0
		inset       = false
	)

	for _, tok := range tokens {
		if strings.EqualFold(tok, insetKeyword) {
			inset = true

			continue
		}

		if strings.EqualFold(tok, "currentcolor") {
			color = current
			alpha = 1.0

			continue
		}

		if r, g, b, a, parsedOK := css.ParseColor(tok); parsedOK {
			color = [3]float64{float64(r) / 255.0, float64(g) / 255.0, float64(b) / 255.0} //nolint:mnd // scale 0-255 to 0-1
			alpha = a

			continue
		}

		length, lengthOK := plainLength(tok, fsize, 0)
		if !lengthOK || lengthCount >= len(lengths) {
			return lengths, 0, color, alpha, false, false
		}

		lengths[lengthCount] = length
		lengthCount++
	}

	return lengths, lengthCount, color, alpha, inset, true
}

func boxShadowFromLengths(
	lengths [boxShadowMaxLengths]float64, lengthCount int, color [3]float64, alpha float64, inset bool,
) (parsedBoxShadow, bool) {
	if lengthCount < 2 {
		return parsedBoxShadow{}, false //nolint:exhaustruct // invalid layer
	}

	blur := 0.0
	spread := 0.0

	if lengthCount >= 3 {
		if lengths[2] < 0 {
			return parsedBoxShadow{}, false //nolint:exhaustruct // invalid layer
		}

		blur = lengths[2]
	}

	if lengthCount >= boxShadowMaxLengths {
		spread = lengths[3]
	}

	return parsedBoxShadow{
		x:      lengths[0],
		y:      lengths[1],
		blur:   blur,
		spread: spread,
		color:  color,
		alpha:  alpha,
		inset:  inset,
	}, true
}

// ParseBoxShadowBlur parses a CSS blur radius (third length) for box-shadow-blur.
// It returns the length in points and ok==true when the value is a valid
// non-negative length (em/rem/px/pt etc resolved against fsize). A missing or
// invalid value returns ok==false so the caller keeps the prior style.
func ParseBoxShadowBlur(value string, fsize float64) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, cssDisplayNone) || strings.EqualFold(value, "initial") {
		return 0, false
	}
	v, ok := plainLength(value, fsize, 0)
	if !ok || v < 0 {
		return 0, false
	}
	return v, true
}

// ApplyBoxShadowBlur parses value and, when valid, writes it to style.
// It sets BoxShadowSet so the shadow layer is painted even when the shorthand
// raw is empty (longhand-only authoring). Returns true when handled.
func ApplyBoxShadowBlur(style *ResolvedStyle, value string, fsize float64) bool {
	if style == nil {
		return false
	}
	v, ok := ParseBoxShadowBlur(value, fsize)
	if !ok {
		if strings.EqualFold(strings.TrimSpace(value), cssDisplayNone) {
			clearBoxShadow(style)
			return true
		}
		return false
	}
	style.BoxShadowBlur = v
	style.BoxShadowSet = true
	return true
}

// ParseBoxShadowSpread parses a CSS spread radius (fourth length) for
// box-shadow-spread. Negative values are allowed per CSS; they shrink the
// shadow. Returns ok==false for missing/invalid.
func ParseBoxShadowSpread(value string, fsize float64) (float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, cssDisplayNone) || strings.EqualFold(value, "initial") {
		return 0, false
	}
	v, ok := plainLength(value, fsize, 0)
	if !ok {
		return 0, false
	}
	return v, true
}

// ApplyBoxShadowSpread parses value and writes style.BoxShadowSpread.
func ApplyBoxShadowSpread(style *ResolvedStyle, value string, fsize float64) bool {
	if style == nil {
		return false
	}
	v, ok := ParseBoxShadowSpread(value, fsize)
	if !ok {
		if strings.EqualFold(strings.TrimSpace(value), cssDisplayNone) {
			clearBoxShadow(style)
			return true
		}
		return false
	}
	style.BoxShadowSpread = v
	style.BoxShadowSet = true
	return true
}

// ParseBoxShadowColor parses a CSS color for box-shadow-color against the
// element's current color. It returns the 0..1 RGB triple, alpha, and ok.
func ParseBoxShadowColor(value string, current [3]float64) ([3]float64, float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return [3]float64{}, 0, false
	}
	if strings.EqualFold(value, "currentcolor") {
		return current, 1.0, true
	}
	if strings.EqualFold(value, "initial") || strings.EqualFold(value, cssDisplayNone) {
		return [3]float64{}, 0, false
	}
	r, g, b, a, ok := css.ParseColor(value)
	if !ok {
		return [3]float64{}, 0, false
	}
	return [3]float64{float64(r) / 255.0, float64(g) / 255.0, float64(b) / 255.0}, a, true
}

// ApplyBoxShadowColor parses value and writes style.BoxShadowColor.
func ApplyBoxShadowColor(style *ResolvedStyle, value string) bool {
	if style == nil {
		return false
	}
	c, _, ok := ParseBoxShadowColor(value, style.Color)
	if !ok {
		return false
	}
	style.BoxShadowColor = c
	style.BoxShadowSet = true
	return true
}

// ParseBoxShadowOffset parses box-shadow-offset which may be one or two
// lengths (x y). A single length sets both axes (common authoring shorthand
// for symmetric offset). Returns x, y in points.
func ParseBoxShadowOffset(value string, fsize float64) (float64, float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" || strings.EqualFold(value, cssDisplayNone) || strings.EqualFold(value, "initial") {
		return 0, 0, false
	}
	var toks [2]string
	n := splitSpaceTokens(value, toks[:])
	if n == 0 || n > 2 {
		return 0, 0, false
	}
	x, ok := plainLength(toks[0], fsize, 0)
	if !ok {
		return 0, 0, false
	}
	if n == 1 {
		// Single token: treat as uniform offset for both axes so that
		// "box-shadow-offset: 2pt" matches shorthand 2px 2px expectation.
		return x, x, true
	}
	y, ok := plainLength(toks[1], fsize, 0)
	if !ok {
		return 0, 0, false
	}
	return x, y, true
}

// ApplyBoxShadowOffset parses value and writes style.BoxShadowX/Y.
func ApplyBoxShadowOffset(style *ResolvedStyle, value string, fsize float64) bool {
	if style == nil {
		return false
	}
	x, y, ok := ParseBoxShadowOffset(value, fsize)
	if !ok {
		return false
	}
	style.BoxShadowX = x
	style.BoxShadowY = y
	style.BoxShadowSet = true
	return true
}

// ParseBoxShadowInset parses box-shadow-position / box-shadow-inset. The CSS
// value "inset" yields true; "outset", "initial", or empty yields false.
// "none" is handled by the caller as a full clear.
func ParseBoxShadowInset(value string) (bool, bool) {
	v := strings.ToLower(strings.TrimSpace(value))
	switch v {
	case insetKeyword:
		return true, true
	case "outset", "initial", "":
		return false, true
	case cssDisplayNone:
		return false, true
	default:
		return false, false
	}
}

// ParseBoxShadowPosition is an alias for ParseBoxShadowInset for the
// box-shadow-position longhand (inset vs outset). See ParseBoxShadowInset.
func ParseBoxShadowPosition(value string) (bool, bool) { return ParseBoxShadowInset(value) }

// ApplyBoxShadowPosition parses box-shadow-position (inset/outset) and writes
// style.BoxShadowInset.
func ApplyBoxShadowPosition(style *ResolvedStyle, value string) bool {
	if style == nil {
		return false
	}
	if strings.EqualFold(strings.TrimSpace(value), cssDisplayNone) {
		clearBoxShadow(style)
		return true
	}
	inset, ok := ParseBoxShadowInset(value)
	if !ok {
		return false
	}
	style.BoxShadowInset = inset
	style.BoxShadowSet = true
	return true
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
			spread: sty.BoxShadowSpread, color: sty.BoxShadowColor, alpha: 1.0, inset: sty.BoxShadowInset,
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
	blur := e.scalePt(shadow.blur)
	offX := e.scalePt(shadow.x)
	offY := e.scalePt(shadow.y)
	originX := posX + offX - spread
	originY := posY + offY - spread
	shadowW := width + spread*2
	shadowH := height + spread*2

	if shadowW <= 0 || shadowH <= 0 {
		return dst
	}

	baseAlpha := shadow.alpha
	if baseAlpha <= 0 {
		baseAlpha = 1.0
	}

	if blur > 0 {
		for step := boxShadowBlurSteps; step >= 1; step-- {
			expand := blur * float64(step) / float64(boxShadowBlurSteps)
			layerAlpha := baseAlpha * (float64(boxShadowBlurSteps-step+1) / float64(boxShadowBlurSteps*3))
			op := shadowFillOp(
				originX-expand, originY-expand, shadowW+expand*2, shadowH+expand*2, shadow.color, radiusX, radiusY,
			)
			op.Alpha = layerAlpha
			dst = append(dst, op)
		}
	}

	coreOp := shadowFillOp(originX, originY, shadowW, shadowH, shadow.color, radiusX, radiusY)
	coreOp.Alpha = baseAlpha
	return append(dst, coreOp)
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

	baseAlpha := shadow.alpha
	if baseAlpha <= 0 {
		baseAlpha = 1.0
	}

	// Paint inner blur / shadow band inside padding box
	steps := boxShadowBlurSteps
	if blur <= 0 {
		steps = 1
	}

	for step := 1; step <= steps; step++ {
		d := insetDepth * float64(step) / float64(steps)
		alpha := baseAlpha * (1.0 - float64(step-1)/float64(steps))
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
