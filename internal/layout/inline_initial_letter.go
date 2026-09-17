package layout

// initial-letter drop-cap lite (CSS Inline 3).
//
// Authoring rule: put initial-letter on a real leading element (e.g.
// <span class="drop">T</span>ext...). The CSS selector parser rejects
// :first-letter, so the engine does not synthesize a first-letter box.
//
// Model: the sized letter is taken out of the inline stream, painted at the
// content origin, and registered as a left float-like exclusion so the next
// sink-1 lines shorten beside it. initial-letter-wrap:none disables the
// exclusion (letter still paints oversized). Align keywords other than
// alphabetic are stored; alphabetic is the only alignment used here.

// prepareInitialLetter extracts and sizes a leading initial-letter run.
// Returns (letter items, remaining items, ok).
func (e *engine) prepareInitialLetter(
	items []inlineItem, blockStyle *ResolvedStyle,
) (letter []inlineItem, rest []inlineItem, ok bool) {
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

	letter = make([]inlineItem, 1)
	copy(letter, items[start:end])
	e.sizeInitialLetterRun(letter, blockStyle, sty)

	rest = make([]inlineItem, 0, len(items)-1)
	rest = append(rest, items[:start]...)
	rest = append(rest, items[end:]...)

	return letter, rest, true
}

func firstInitialLetterIndex(items []inlineItem) int {
	for i := range items {
		it := items[i]
		if it.forceBreak || it.img || it.blockBox != nil || it.text == "" {
			continue
		}

		if it.style != nil && it.style.InitialLetterSize >= 1 {
			return i
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

	for i := range letter {
		letter[i].style = &cloned
		if letter[i].text == "" {
			continue
		}

		letter[i].w = e.measureTextFace(
			transformInlineText(letter[i].text, cloned.TextTransform), &cloned,
		)
		letter[i].h = targetH
	}
}

func surroundingLineHeight(blockStyle, letterStyle *ResolvedStyle) float64 {
	if blockStyle != nil {
		return lineHeightOf(blockStyle)
	}

	if letterStyle != nil {
		return lineHeightOf(letterStyle)
	}

	return 12 * defaultLineHeightRatio
}

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

// placeInitialLetter paints the sized letter at the content origin and
// records a left float-like exclusion for the sink depth so following lines
// shorten beside the letter. initial-letter-wrap contour modes are stored
// but all use the rectangular letter box (lite).
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

	sink := initialLetterSinkLines(sty)
	if parentLH <= 0 {
		parentLH = surroundingLineHeight(nil, sty) * e.scale
	}

	bandH := float64(sink) * parentLH
	if bandH <= 0 {
		bandH = letter[0].h
	}

	// Align alphabetic (and other stored keywords, lite): hang the glyph so
	// its top meets the first line top and it sinks into following lines.
	_ = sty.InitialLetterAlign
	ascent := letter[0].h * 0.8
	baseline := lineY + ascent

	leftX := contentX
	for i := range letter {
		leftX += letter[i].marginL
		e.emitLineItems(boxNode, letter[i:i+1], leftX, baseline, parentLH, lineY, 0)
		leftX += letter[i].w + letter[i].marginR
	}

	if floats != nil {
		fbox := &box{ //nolint:exhaustruct // float exclusion geometry only
			x: contentX, y: lineY, w: letterW, height: bandH,
		}
		floats.place(floatLeft, fbox, 0, 0, nil)
	}

	return letterW
}
