package layout

import "strings"

// emitVerticalUprightRun paints a rotation-0 text run inside a vertical
// writing column and reports whether it handled the run. Two shapes exist:
//
//   - text-orientation: upright stacks each rune down the column, each glyph
//     upright and centered in the column.
//   - text-combine-upright paints the whole run as one upright cell centered
//     in the column (no scaled cell).
//
// Rotated runs keep the shared face-run path.
func (e *engine) emitVerticalUprightRun(
	item *inlineItem, leftX, baseline, size, ascent, descent float64,
) (float64, bool) {
	style := item.style
	if style == nil || !isVerticalWritingMode(style.WritingMode) {
		return 0, false
	}

	if inlineRunRotation(style, item.text) != 0 {
		return 0, false
	}

	if strings.EqualFold(strings.TrimSpace(style.TextOrientation), textOrientationUpright) {
		return e.emitUprightColumn(item, leftX, baseline, size, ascent, descent), true
	}

	// text-combine-upright: one upright cell centered in the column.
	runWidth := e.measureTextFace(transformInlineText(item.text, style.TextTransform), style)
	e.emitInlineFaceRuns(item, leftX+(item.w-runWidth)/two, baseline, size, ascent, descent)

	return item.w, true
}

// emitUprightColumn draws one rune per vertical step, centered in the column.
// The rune's measured width doubles as its vertical advance: the engine reads
// no vertical font metrics, so this matches the spec's synthesized advance and
// keeps Latin and CJK glyphs evenly stacked.
func (e *engine) emitUprightColumn(
	item *inlineItem, leftX, baseline, size, ascent, descent float64,
) float64 {
	step := baseline

	for _, r := range item.text {
		glyph := string(r)
		glyphWidth := e.measureTextFace(transformInlineText(glyph, item.style.TextTransform), item.style)
		e.emitGlyphRuns(item, glyph, leftX+(item.w-glyphWidth)/two, step, size, ascent, descent)
		step += glyphWidth
	}

	return item.w
}

// emitTextRuns chooses the vertical upright/combined emitter or the shared
// face-run path and returns the updated x cursor and the painted span.
func (e *engine) emitTextRuns(
	item *inlineItem, leftX, textBaseline, uprightBaseline, size, ascent, descent float64,
) (float64, float64) {
	if span, ok := e.emitVerticalUprightRun(item, leftX, uprightBaseline, size, ascent, descent); ok {
		return leftX + item.w, span
	}

	return e.emitInlineFaceRuns(item, leftX, textBaseline, size, ascent, descent)
}

// emitGlyphRuns paints one glyph through the item's primary face or its
// per-face fallback runs.
func (e *engine) emitGlyphRuns(
	item *inlineItem, glyph string, glyphX, baseline, size, ascent, descent float64,
) {
	if run, ok := e.primaryFaceRun(glyph, item.style); ok {
		e.emitInlineTextRun(item, run, glyphX, baseline, size, ascent, descent)

		return
	}

	for _, run := range e.splitTextByFace(glyph, item.style) {
		e.emitInlineTextRun(item, run, glyphX, baseline, size, ascent, descent)
	}
}
