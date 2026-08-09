package layout

import (
	"gowkhtmltopdf/internal/pdf"
	"strings"
)

// undRun accumulates one continuous underline stroke across adjacent
// same-href (or same-decoration) runs on a line. Multi-face items already
// share one span; this also merges bold/italic/nested chunks and skips
// whitespace-only runs so dense reference lists (wrapped archive.org URLs) do
// not paint a forest of short double-rules.
type undRun struct {
	active  bool
	x, y, w float64
	uw      float64
	r, g, b float64
	href    string
	hasHref bool
}

// flush emits the accumulated underline stroke, if any.
func (u *undRun) flush(e *engine) {
	if u.active && u.w > 0.01 {
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLine, X: u.x, Y: u.y, W: u.w, H: 0,
			Width: u.uw, R: u.r, G: u.g, B: u.b,
		})
	}

	*u = undRun{} //nolint:exhaustruct // intentional zero fields
}

// emitInlineBlock relocates a block-in-inline box onto the current line and
// returns the updated x cursor.
func (e *engine) emitInlineBlock(
	boxNode *box, item *inlineItem, leftX, baseline, justifyGap float64,
	gapAfter bool, und *undRun,
) float64 {
	und.flush(e)

	dx := leftX - item.blockBox.x
	dy := baseline - item.h - item.blockBox.y
	e.shiftBoxOps(item.blockBox, dx, dy)
	item.blockBox.x += dx
	item.blockBox.y += dy
	// Attach to parent so paint-time transforms/opacity stamp the subtree.
	if boxNode != nil {
		boxNode.children = append(boxNode.children, item.blockBox)
	}

	leftX += item.blockBox.w + item.marginR
	if gapAfter && isJustifyGapAfter(*item) {
		leftX += justifyGap
	}

	return leftX
}

// emitInlineImage places an image (or inline-block) item on the line and
// returns the updated x cursor.
func (e *engine) emitInlineImage(
	item *inlineItem, leftX, lineY, lineH, baseline, justifyGap float64,
	gapAfter bool, und *undRun,
) float64 {
	und.flush(e)

	top := baseline - item.h

	switch item.style.VerticalAlign {
	case cssVerticalAlignTop:
		top = lineY
	case cssVerticalAlignMiddle:
		top = lineY + (lineH-item.h)/two
	case cssVerticalAlignBottom:
		top = lineY + lineH - item.h
	}

	if item.imgRef != nil && item.imgRef.data != nil {
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpImage, X: leftX, Y: top, W: item.w, H: item.h,
			Image: item.imgRef.data, ImgW: item.imgRef.w, ImgH: item.imgRef.h, IsJPEG: item.imgRef.isJPEG,
		})
	}

	if item.href != "" {
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLinkURI, X: leftX, Y: top, W: item.w, H: item.h, URI: item.href,
		})
	}

	leftX += item.w + item.marginR
	if gapAfter && isJustifyGapAfter(*item) {
		leftX += justifyGap
	}

	return leftX
}

// emitInlineText paints the face runs of one text item and returns the
// updated x cursor.
func (e *engine) emitInlineText(
	item *inlineItem, leftX, baseline, justifyGap float64,
	gapAfter bool, und *undRun,
) float64 {
	child := item.style.Color
	size := item.style.FontSize * e.scale
	ascent := e.fontAscent(size)
	descent := e.fontDescent(size)

	if ascent+descent < size*0.5 {
		// Fallback when font metrics are missing — keep hit targets usable.
		ascent = size * ascentRatio
		descent = size * descentRatio
	}

	runStart := leftX

	var runSpan float64

	if run, ok := e.primaryFaceRun(item.text, *item.style); ok {
		e.emitInlineTextRun(
			item,
			run,
			leftX,
			baseline,
			size,
			ascent,
			descent,
		)

		leftX += run.w
		runSpan = run.w
	} else {
		for _, run := range e.splitTextByFace(item.text, *item.style) {
			e.emitInlineTextRun(
				item,
				run,
				leftX,
				baseline,
				size,
				ascent,
				descent,
			)

			leftX += run.w
			runSpan += run.w
		}
	}
	// Decoration: one stroke per logical link run on this line (not per
	// face-run / nested style chunk). Thin stroke ~5% em, clamped for
	// dense reference print (min 0.25pt, max 0.45pt).
	// Force-underline a[href] for PDF affordance. Bare URL strings
	// (https://…, archive fragments) never get underlines — multi-line
	// ref lists were a forest of rules; titles/prose links still underline.
	e.paintDecoration(item, runStart, runSpan, size, ascent, descent, baseline, child, und)

	leftX += item.marginR
	if gapAfter && isJustifyGapAfter(*item) {
		leftX += justifyGap
	}

	return leftX
}

func (e *engine) emitInlineTextRun(
	item *inlineItem,
	run faceRun,
	leftX, baseline, size, ascent, descent float64,
) {
	child := item.style.Color
	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpText, X: leftX, Y: baseline, W: run.w, H: item.h,
		Text: run.text, Font: run.face, Size: size, Bold: item.style.FontWeight >= fontWeightBold,
		R: child[0], G: child[1], B: child[2],
	})

	if item.href != "" {
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLinkURI, X: leftX, Y: baseline - ascent, W: run.w,
			H: ascent + descent, URI: item.href,
		})
	}
}

// paintDecoration draws the underline / line-through strokes for one text
// item, extending the active underline run when the styling continues.
func (e *engine) paintDecoration(
	item *inlineItem, runStart, runSpan, size, ascent, descent, baseline float64,
	child [3]float64, und *undRun,
) {
	bareURL := isBareURLText(item.text)
	wantUnderline := !bareURL && (item.style.TextDecoration == cssTextDecorationUnderline ||
		(item.href != "" && item.style.TextDecoration != cssTextDecorationLineThrough))
	wsOnly := strings.TrimSpace(item.text) == ""

	if runSpan <= layoutSlack {
		if !wantUnderline {
			und.flush(e)
		}

		return
	}

	if !wantUnderline && item.style.TextDecoration != cssTextDecorationLineThrough {
		und.flush(e)

		return
	}

	uWidth := underlineStrokeWidth(size)

	if wantUnderline {
		// Sit clearly below glyph descenders (~1–2mm visual gap).
		underY := baseline + descent + size*underlineOffsetRatio
		e.paintUnderline(item, runStart, runSpan, underY, uWidth, size, wsOnly, child, und)
	} else {
		und.flush(e)
	}

	e.paintLineThrough(item, runStart, runSpan, baseline, ascent, uWidth, wsOnly, child)
}

// paintLineThrough strokes the strike-through rule for a decorated text item.
func (e *engine) paintLineThrough(
	item *inlineItem, runStart, runSpan, baseline, ascent, uWidth float64,
	wsOnly bool, child [3]float64,
) {
	if item.style.TextDecoration == cssTextDecorationLineThrough && !wsOnly {
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLine, X: runStart, Y: baseline - ascent*0.3, W: runSpan, H: 0,
			Width: uWidth, R: child[0], G: child[1], B: child[2],
		})
	}
}

// paintUnderline extends or starts the active underline run for one text item.
func (e *engine) paintUnderline(
	item *inlineItem, runStart, runSpan, underY, uWidth, size float64,
	wsOnly bool, child [3]float64, und *undRun,
) {
	if wsOnly {
		// Do not start an underline on whitespace-only, but extend
		// an active same-href run across inter-word spaces.
		if und.active && item.href != "" && und.hasHref && item.href == und.href {
			extendUnder(und, runStart, runSpan, uWidth)
		}

		return
	}

	if startsActiveUnder(item, und, runStart, underY, size) {
		extendUnder(und, runStart, runSpan, uWidth)

		return
	}
	// Prefer first chunk's color (title) when styles mix.
	und.flush(e)
	und.active = true
	und.x = runStart
	und.y = underY
	und.w = runSpan
	und.uw = uWidth
	und.r, und.g, und.b = child[0], child[1], child[2]
	und.href = item.href
	und.hasHref = item.href != ""
}

// startsActiveUnder reports that the item continues an active underline run:
// same href (or both href-less), same Y, and near enough in X — justify
// rivers / margins between nested chunks (up to ~2em) do not split it.
func startsActiveUnder(item *inlineItem, und *undRun, runStart, underY, size float64) bool {
	if !und.active || !nearUndY(und.y, underY) {
		return false
	}

	if (item.href != "" && und.hasHref && item.href == und.href) ||
		(item.href == "" && !und.hasHref) {
		return runStart <= und.x+und.w+size*2+0.5
	}

	return false
}

// extendUnder widens the active underline run over this text span and keeps
// the thinner of the two stroke widths.
func extendUnder(und *undRun, runStart, runSpan, uWidth float64) {
	end := runStart + runSpan
	if end > und.x+und.w {
		und.w = end - und.x
	}

	if uWidth < und.uw {
		und.uw = uWidth
	}
}

// underlineStrokeWidth returns a light print-friendly underline thickness.
// ~5% em, clamped so small ref text stays visible without dense double-rules.
func underlineStrokeWidth(em float64) float64 {
	uWidth := em * underlineWidthEm
	if uWidth < baselineInsetRatio {
		uWidth = 0.25
	}

	if uWidth > underlineWidthMax {
		uWidth = 0.45
	}

	return uWidth
}

// isBareURLText reports that s is essentially a URL (optional leading
// punctuation from "(https://…)" wrappers). Used to skip force-underlines on
// raw link text in reference lists.
func isBareURLText(s string) bool {
	tmp := strings.TrimSpace(s)
	tmp = strings.TrimLeft(tmp, "([\"' \t")

	if tmp == "" {
		return false
	}

	low := strings.ToLower(tmp)
	if strings.HasPrefix(low, "https://") || strings.HasPrefix(low, "http://") ||
		strings.HasPrefix(low, "www.") {
		return true
	}
	// Continuation fragment of a wrapped URL (no scheme on later lines).
	return hasURLFragmentMarker(tmp)
}

// hasURLFragmentMarker applies the tail heuristics for wrapped URLs whose
// scheme ended up on an earlier line.
func hasURLFragmentMarker(text string) bool {
	if !strings.Contains(text, "/") || strings.ContainsAny(text, " \t") {
		return false
	}

	for _, marker := range []string{".com", ".org", "archive.", ".html", ".php"} {
		if strings.Contains(text, marker) {
			return true
		}
	}

	return strings.Count(text, "/") >= two
}

func nearUndY(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}

	return d < halfRatio
}

// measureTextFace measures s using per-rune CSS font-family fallback
// (same face selection as paint).
//
// Primary-face fast path: resolve the style face once and only consult the
// fallback cache when a glyph is missing — Latin report text almost never
// leaves the primary face.
func (e *engine) measureTextFace(cssSheet string, sty ResolvedStyle) float64 {
	if cssSheet == "" {
		return 0
	}

	size := sty.FontSize * e.scale
	lstyle := sty.LetterSpacing * e.scale
	primary := e.faceFor(sty)

	if primary == nil {
		primary = e.font
	}

	var total float64

	runeCount := 0

	for _, runic := range cssSheet {
		face := primary
		if !isRuneWhitespace(runic) && primary.GlyphID(runic) == 0 {
			face = e.faceForRuneFallback(sty, runic, primary)
			if face == nil {
				face = e.font
			}
		}

		total += face.GlyphAdvancePoints(runic, size)
		runeCount++
	}

	if lstyle != 0 && runeCount > 0 {
		total += lstyle * float64(runeCount)
	}

	return total
}

// measureRuneFace measures a single rune with the same face selection as
// measureTextFace, without allocating string(r).
func (e *engine) measureRuneFace(curRune rune, sty ResolvedStyle) float64 {
	size := sty.FontSize * e.scale
	face := e.faceForRune(sty, curRune)

	if face == nil {
		face = e.font
	}

	w := face.GlyphAdvancePoints(curRune, size)
	if sty.LetterSpacing != 0 {
		w += sty.LetterSpacing * e.scale
	}

	return w
}

type faceRun struct {
	text string
	face *pdf.Font
	w    float64
}

// splitTextByFace splits s into contiguous runs that share the same face
// under CSS font-family fallback.
//
//nolint:cyclop,funlen // hot path: per-rune face-fallback run splitting
func (e *engine) splitTextByFace(cssSheet string, sty ResolvedStyle) []faceRun {
	if cssSheet == "" {
		return nil
	}

	primaryRun, allPrimary := e.primaryFaceRun(cssSheet, sty)
	// Fast path: every non-whitespace glyph is on the primary face (typical
	// for Latin). One faceFor + per-rune GlyphID, single output run.
	if allPrimary {
		return []faceRun{primaryRun}
	}

	size := sty.FontSize * e.scale

	primary := e.faceFor(sty)
	if primary == nil {
		primary = e.font
	}

	runs := make([]faceRun, 0, 1)
	start := 0

	var current *pdf.Font

	var width float64

	for idx, runic := range cssSheet {
		face := primary
		if isRuneWhitespace(runic) {
			face = primary
		} else if primary.GlyphID(runic) == 0 {
			face = e.faceForRuneFallback(sty, runic, primary)
			if face == nil {
				face = e.font
			}
		}

		if current != nil && face != current {
			runs = append(runs, faceRun{
				text: cssSheet[start:idx],
				face: current,
				w:    width,
			})
			start = idx
			width = 0
		}

		if current == nil {
			start = idx
		}

		current = face
		width += face.AdvanceInPoints(runic, size)
	}

	if current != nil {
		runs = append(runs, faceRun{
			text: cssSheet[start:],
			face: current,
			w:    width,
		})
	}

	return runs
}

func (e *engine) primaryFaceRun(cssSheet string, sty ResolvedStyle) (faceRun, bool) {
	if cssSheet == "" {
		return faceRun{}, false //nolint:exhaustruct // intentional zero fields
	}

	primary := e.faceFor(sty)
	if primary == nil {
		primary = e.font
	}

	if !faceRunAllPrimary(cssSheet, primary) {
		return faceRun{}, false //nolint:exhaustruct // intentional zero fields
	}

	size := sty.FontSize * e.scale

	var width float64

	for _, runic := range cssSheet {
		width += primary.AdvanceInPoints(runic, size)
	}

	return faceRun{text: cssSheet, face: primary, w: width}, true
}

// faceRunAllPrimary reports that every non-whitespace rune in s is covered by
// primary (so splitTextByFace can emit a single run).
func faceRunAllPrimary(s string, primary *pdf.Font) bool {
	if primary == nil {
		return false
	}

	for _, r := range s {
		if isRuneWhitespace(r) {
			continue
		}

		if primary.GlyphID(r) == 0 {
			return false
		}
	}

	return true
}

func (e *engine) fontAscent(size float64) float64 {
	return float64(e.font.Ascent()) * size / float64(e.font.UnitsPerEm())
}

func (e *engine) fontDescent(size float64) float64 {
	return float64(-e.font.Descent()) * size / float64(e.font.UnitsPerEm())
}

// isExternalHref reports whether a link target should become a URI
// annotation (http/https/mailto). Local same-document fragments are handled
// separately as internal GoTo links.
func isExternalHref(href string) bool {
	low := strings.ToLower(href)

	return strings.HasPrefix(low, "http://") || strings.HasPrefix(low, "https://") || strings.HasPrefix(low, "mailto:")
}

// isInternalHref reports a same-document fragment link (#id).
func isInternalHref(href string) bool {
	return strings.HasPrefix(href, "#") && len(href) > 1
}
