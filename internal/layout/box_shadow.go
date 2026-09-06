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
	if value == "" || strings.EqualFold(value, cssDisplayNone) || strings.EqualFold(value, cssKeywordInitial) {
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
	if value == "" || strings.EqualFold(value, cssDisplayNone) || strings.EqualFold(value, cssKeywordInitial) {
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
	if strings.EqualFold(value, cssKeywordInitial) || strings.EqualFold(value, cssDisplayNone) {
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
	if value == "" || strings.EqualFold(value, cssDisplayNone) || strings.EqualFold(value, cssKeywordInitial) {
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
	case "outset", cssKeywordInitial, "":
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

// ApplyBoxShadowPosition parses box-shadow-position (inset/outset) and stores
// style.BoxShadowInset. When a box-shadow shorthand Raw string is already set,
// paint keeps that layer list (including its own inset keyword) so we match
// Chrome, which does not implement this Borders 4 longhand yet. Structured
// fields are used only when Raw is empty (longhand-only / blur/offset setters).
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

// appendBoxShadow paints box-shadow layers in reverse order so layer 0 is on
// top. insetOnly selects inset vs outset layers: callers paint outset behind
// the background and inset after it so the fill does not hide inner rims.
func (e *engine) appendBoxShadow(
	dst []Op, sty ResolvedStyle, posX, posY, width, height float64,
	radiusX, radiusY [4]float64, insetOnly bool,
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
		if shadow.inset != insetOnly {
			continue
		}
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

	// Soft wash, not opaque slabs. Chrome inset on small chips (fixture-61
	// #12) keeps the cream fill and label readable with a light top/left rim.
	depth := insetShadowDepth(spread, blur, width, height, offX, offY)

	baseAlpha := shadow.alpha
	if baseAlpha <= 0 {
		baseAlpha = 1.0
	}

	// Positive offset → shadow falls down/right → rim reads on top/left.
	paintTop := offY >= -1e-6
	paintLeft := offX >= -1e-6
	paintBot := offY <= 1e-6
	paintRight := offX <= 1e-6

	steps := insetShadowStepCount(blur)

	for step := 1; step <= steps; step++ {
		t := float64(step) / float64(steps)
		d := depth * t
		alpha, ok := insetShadowStepAlpha(baseAlpha, blur, t)
		if !ok {
			continue
		}
		topH, botH, leftW, rightW := insetShadowRimSizes(offX, offY, d, depth)
		rims := insetShadowRims{
			paintTop: paintTop, paintBot: paintBot, paintLeft: paintLeft, paintRight: paintRight,
			topH: topH, botH: botH, leftW: leftW, rightW: rightW, alpha: alpha,
		}
		dst = appendInsetShadowTopBottom(dst, shadow, posX, posY, width, height, radiusX, radiusY, rims)
		dst = appendInsetShadowLeftRight(dst, shadow, posX, posY, width, height, radiusX, radiusY, rims)
	}

	return dst
}

// insetShadowDepth caps the inner wash so small chips keep their fill readable.
func insetShadowDepth(spread, blur, width, height, offX, offY float64) float64 {
	depth := spread + blur*0.5 + math.Max(math.Abs(offX), math.Abs(offY))*0.5
	if depth < 1 {
		depth = 1
	}
	maxDepth := math.Min(width, height) * 0.22
	if maxDepth < 1.5 {
		maxDepth = 1.5
	}
	if depth > maxDepth {
		depth = maxDepth
	}
	return depth
}

// insetShadowStepCount spreads blurred shadows over more bands than sharp ones.
func insetShadowStepCount(blur float64) int {
	steps := boxShadowBlurSteps * 2
	if blur <= 0 {
		steps = 3
	}
	return steps
}

// insetShadowStepAlpha fades each band inward, keeping the peak alpha low so
// the fill stays visible. ok is false when the band is too faint to paint.
func insetShadowStepAlpha(baseAlpha, blur, t float64) (float64, bool) {
	alpha := baseAlpha * (1.0 - t) * 0.28
	if blur <= 0 {
		alpha = baseAlpha * (1.0 - t) * 0.4
	}
	if alpha < 0.025 {
		return 0, false
	}
	return alpha, true
}

// insetShadowRimSizes biases rim thickness toward the light side: a positive
// offset throws the shadow down/right, so the top/left rim reads stronger.
func insetShadowRimSizes(offX, offY, d, depth float64) (float64, float64, float64, float64) {
	topH, botH, leftW, rightW := d, d*0.3, d, d*0.3
	if offY > 0 {
		topH = math.Min(d+offY*0.4, depth)
	} else if offY < 0 {
		botH = math.Min(d-offY*0.4, depth)
		topH = d * 0.3
	}
	if offX > 0 {
		leftW = math.Min(d+offX*0.4, depth)
	} else if offX < 0 {
		rightW = math.Min(d-offX*0.4, depth)
		leftW = d * 0.3
	}
	return topH, botH, leftW, rightW
}

// insetShadowRims is one band of the inset wash: which edges paint and how thick.
type insetShadowRims struct {
	paintTop, paintBot, paintLeft, paintRight bool
	topH, botH, leftW, rightW                 float64
	alpha                                     float64
}

func appendInsetShadowTopBottom(
	dst []Op, shadow parsedBoxShadow, posX, posY, width, height float64,
	radiusX, radiusY [4]float64, rims insetShadowRims,
) []Op {
	if rims.paintTop && rims.topH > 0 && rims.topH < height-0.5 {
		op := shadowFillOp(posX, posY, width, rims.topH, shadow.color, radiusX, radiusY)
		op.Alpha = rims.alpha
		dst = append(dst, op)
	}
	if rims.paintBot && rims.botH > 0 && height > rims.botH+0.5 {
		op := shadowFillOp(posX, posY+height-rims.botH, width, rims.botH, shadow.color, radiusX, radiusY)
		op.Alpha = rims.alpha
		dst = append(dst, op)
	}
	return dst
}

func appendInsetShadowLeftRight(
	dst []Op, shadow parsedBoxShadow, posX, posY, width, height float64,
	radiusX, radiusY [4]float64, rims insetShadowRims,
) []Op {
	if rims.paintLeft && rims.leftW > 0 && rims.leftW < width-0.5 {
		op := shadowFillOp(posX, posY, rims.leftW, height, shadow.color, radiusX, radiusY)
		op.Alpha = rims.alpha
		dst = append(dst, op)
	}
	if rims.paintRight && rims.rightW > 0 && width > rims.rightW+0.5 {
		op := shadowFillOp(posX+width-rims.rightW, posY, rims.rightW, height, shadow.color, radiusX, radiusY)
		op.Alpha = rims.alpha
		dst = append(dst, op)
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
