package layout

// initial-letter drop-cap lite (CSS Inline 3).
//
// Authoring rule: put initial-letter on a real leading element (e.g.
// <span class="drop">T</span>ext...). The CSS selector parser rejects
// :first-letter, so the engine does not synthesize a first-letter box.
//
// Model: the sized letter is taken out of the inline stream and painted at
// the content origin. initial-letter-wrap chooses the rectangular exclusion:
// none = no side wrap, first = first line only, all/grid = every overlapping
// sink line. initial-letter-align picks the vertical metric (alphabetic,
// hanging, leading, ideographic). Both require initial-letter: N on the
// same element; :first-letter is not synthesized.

// initialLetterCapRatio is the synthesized Latin cap-height fraction of the
// sized letter box. Alphabetic hanging uses this as the ascent.
const initialLetterCapRatio = 0.8

// prepareInitialLetter extracts and sizes a leading initial-letter run.
// Returns (letter items, remaining items, ok).
func (e *engine) prepareInitialLetter(
	items []inlineItem, blockStyle *ResolvedStyle,
) ([]inlineItem, []inlineItem, bool) {
	start := firstInitialLetterIndex(items)
	if start < 0 {
		return nil, items, false
	}

	sty := items[start].style
	if sty == nil || sty.InitialLetterSize < 1 {
		return nil, items, false
	}

	// Lite: one leading inline item (typically a single-letter <span>).
	// Do not consume siblings that share the same computed style, so a
	// block-level initial-letter declaration only drops the first word.
	end := start + 1

	letter := make([]inlineItem, 1)
	copy(letter, items[start:end])
	e.sizeInitialLetterRun(letter, blockStyle, sty)

	rest := make([]inlineItem, 0, len(items)-1)
	rest = append(rest, items[:start]...)
	rest = append(rest, items[end:]...)

	return letter, rest, true
}

func firstInitialLetterIndex(items []inlineItem) int {
	for itemIndex := range items {
		it := items[itemIndex]
		if it.forceBreak || it.img || it.blockBox != nil || it.text == "" {
			continue
		}

		if it.style != nil && it.style.InitialLetterSize >= 1 {
			return itemIndex
		}
	}

	return -1
}

func (e *engine) sizeInitialLetterRun(letter []inlineItem, blockStyle, letterStyle *ResolvedStyle) {
	if len(letter) == 0 || letterStyle == nil {
		return
	}

	parentLH := surroundingLineHeight(blockStyle, letterStyle) * e.scale
	size := letterStyle.InitialLetterSize

	if size < 1 {
		size = 1
	}

	// Cap-height ≈ 0.7em; size N lines → font-size ≈ N * lineH / 0.7.
	const capRatio = 0.7

	targetH := size * parentLH
	newFont := targetH / capRatio / e.scale

	if newFont < letterStyle.FontSize {
		newFont = letterStyle.FontSize * size
	}

	cloned := *letterStyle
	cloned.FontSize = newFont
	// Keep line-height tight so the letter box matches the sink band.
	cloned.LineHeight = targetH / e.scale
	cloned.LineHeightUnitless = 0

	for itemIndex := range letter {
		letter[itemIndex].style = &cloned
		if letter[itemIndex].text == "" {
			continue
		}

		letter[itemIndex].w = e.measureTextFace(
			transformInlineText(letter[itemIndex].text, cloned.TextTransform), &cloned,
		)
		letter[itemIndex].h = targetH
	}
}

//nolint:mnd // CSS initial-letter fallback uses the engine's default line size
func surroundingLineHeight(blockStyle, letterStyle *ResolvedStyle) float64 {
	if blockStyle != nil {
		return lineHeightOf(blockStyle)
	}

	if letterStyle != nil {
		return lineHeightOf(letterStyle)
	}

	return 12 * defaultLineHeightRatio
}

//nolint:mnd // CSS initial-letter sink rounding uses the authored half-unit
func initialLetterSinkLines(sty *ResolvedStyle) int {
	if sty == nil {
		return 0
	}

	if sty.InitialLetterSink > 0 {
		return sty.InitialLetterSink
	}

	if sty.InitialLetterSize >= 1 {
		return int(sty.InitialLetterSize + 0.5)
	}

	return 0
}

// initialLetterAlignOffset is the extra block-axis shift (positive = lower)
// so alphabetic / hanging / leading / ideographic land on distinct metrics.
// Requires initial-letter: N (the caller already sized the letter).
//
//nolint:mnd // CSS initial-letter alignment ratios are specification metrics
func initialLetterAlignOffset(align string, letterH, parentLH float64) float64 {
	if parentLH <= 0 {
		parentLH = 12 * defaultLineHeightRatio
	}

	if letterH <= 0 {
		letterH = parentLH
	}

	switch align {
	case initialLetterAlignHanging:
		// Hanging baseline sits near the top of the em (~0.2em from over).
		// Matching first-line hanging to the large letter hanging raises it.
		return 0.2 * (parentLH - letterH)
	case initialLetterAlignLeading:
		// Leading edges include half-leading, so the letter sits lower than
		// a cap-height hang.
		return 0.25 * parentLH
	case initialLetterAlignIdeo:
		// Ideographic face is the em box, not cap-height; center in the
		// N-line band relative to an alphabetic cap hang.
		return 0.12 * parentLH
	default:
		return 0
	}
}

// initialLetterExclusionHeight is the rectangular wrap band. none is 0 (no
// side wrap); first is one line; all/grid and stored <length> cover the sink.
func initialLetterExclusionHeight(sty *ResolvedStyle, parentLH float64) float64 {
	if sty == nil || parentLH <= 0 {
		return 0
	}

	wrap := sty.InitialLetterWrap
	if wrap == "" {
		wrap = initialLetterWrapNone
	}

	switch wrap {
	case initialLetterWrapNone:
		return 0
	case initialLetterWrapFirst:
		return parentLH
	default:
		// all, grid, and stored <length-percentage>: rectangular sink band.
		sink := initialLetterSinkLines(sty)
		if sink < 1 {
			sink = 1
		}

		return float64(sink) * parentLH
	}
}

// placeInitialLetter paints the sized letter at the content origin using the
// chosen align metric, then records a rectangular left exclusion whose height
// follows initial-letter-wrap (none/first/all).
func (e *engine) placeInitialLetter(
	boxNode *box, letter []inlineItem, contentX, lineY, parentLH float64, floats *floatState,
) float64 {
	if len(letter) == 0 {
		return 0
	}

	sty := letter[0].style
	letterW := 0.0

	for i := range letter {
		letterW += letter[i].marginL + letter[i].w + letter[i].marginR
	}

	if parentLH <= 0 {
		parentLH = surroundingLineHeight(nil, sty) * e.scale
	}

	align := initialLetterAlignAlpha
	if sty != nil && sty.InitialLetterAlign != "" {
		align = sty.InitialLetterAlign
	}

	letterTop := lineY + initialLetterAlignOffset(align, letter[0].h, parentLH)
	ascent := letter[0].h * initialLetterCapRatio
	baseline := letterTop + ascent

	leftX := contentX
	for i := range letter {
		leftX += letter[i].marginL
		e.emitLineItems(boxNode, letter[i:i+1], leftX, baseline, parentLH, letterTop, 0)
		leftX += letter[i].w + letter[i].marginR
	}

	bandH := initialLetterExclusionHeight(sty, parentLH)
	if floats != nil && bandH > 0 && letterW > 0 {
		fbox := &box{ //nolint:exhaustruct // float exclusion geometry only
			x: contentX, y: lineY, w: letterW, height: bandH,
		}
		floats.place(floatLeft, fbox, 0, 0, nil)
	}

	return letterW
}
