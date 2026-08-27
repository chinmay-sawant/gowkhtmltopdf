package layout

import "strings"

const (
	boxShadowProp       = "box-shadow"
	insetKeyword        = "inset"
	boxShadowMaxTokens  = 6 // inset + x y blur spread + color
	boxShadowMaxLengths = 4 // x y blur spread
)

type parsedBoxShadow struct {
	x, y, blur float64
	color      [3]float64
}

// parseBoxShadowLayer reads offset-x offset-y [blur [spread]] [color].
// Inset layers are dropped. Spread is accepted and ignored. Blur is stored
// for callers; paint uses a single un-inset opaque offset fill.
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
// by BoxShadowX/Y. Layout size is unchanged. Blur is not rasterized.
func (e *engine) appendBoxShadow(
	dst []Op, sty ResolvedStyle, posX, posY, width, height float64,
) []Op {
	if e == nil || !sty.BoxShadowSet || width <= 0 || height <= 0 {
		return dst
	}

	return append(dst, Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpFillRect,
		X:    posX + e.scalePt(sty.BoxShadowX),
		Y:    posY + e.scalePt(sty.BoxShadowY),
		W:    width,
		H:    height,
		R:    sty.BoxShadowColor[0],
		G:    sty.BoxShadowColor[1],
		B:    sty.BoxShadowColor[2],
	})
}
