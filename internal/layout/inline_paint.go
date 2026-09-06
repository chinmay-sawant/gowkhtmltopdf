package layout

import (
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

const (
	textOverflowEllipsis = "ellipsis"
	verticalAlignSub     = "sub"
	verticalAlignSuper   = "super"
	emphasisStyleFilled  = "filled"
	inlineStyleNone      = "none"

	fallbackAscentRatio  = 0.8
	fallbackDescentRatio = 0.2

	verticalAlignSubRatio = 0.2

	emptyRunEpsilon = 0.01

	lineThroughOffsetRatio = 0.3

	underlineWidthRatio = 0.05
	underlineMinWidth   = 0.25
	underlineMaxWidth   = 0.45
	underlineYTolerance = 0.5

	dashLenRatio = 4
	dashGapRatio = 2.8
	dotGapRatio  = 2.5
	minDotGap    = 2.5
	wavyAmpRatio = 1.2
	minWavyAmp   = 0.8
	maxWavyAmp   = 1.8
	wavySegRatio = 2.6

	minVisibleSeg = 0.2

	emphasisDotRatio = 0.18
	minEmphasisDot   = 1.1
	maxEmphasisDot   = 2.4
	emphasisUnderGap = 1.2

	emphasisOpenStrokeWidth = 0.4

	minEllipsisAvail = 10

	tabSpaceRatio = 0.25

	shadowBlurRatio    = 0.3
	shadowOpacityFloor = 0.2

	fallbackAdvanceRatio = 0.5
)

// undRun accumulates one continuous underline stroke across adjacent
// same-href (or same-decoration) runs on a line. Multi-face items already
// share one span; this also merges bold/italic/nested chunks and skips
// whitespace-only runs so dense reference lists (wrapped archive.org URLs) do
// not paint a forest of short double-rules.
type undRun struct {
	active    bool
	x, y, w   float64
	uw        float64
	r, g, b   float64
	href      string
	hasHref   bool
	style     string
	blendMode string
}

// flush emits the accumulated underline stroke, if any.
//
//nolint:varnamelen,wsl,goconst // underline variants share the existing engine helpers
func (u *undRun) flush(e *engine) {
	if u.active && u.w > 0.01 {
		col := [3]float64{u.r, u.g, u.b}
		prevBlend := e.blendMode
		e.blendMode = u.blendMode
		switch strings.ToLower(strings.TrimSpace(u.style)) {
		case "dashed":
			e.emitDashedLine(u.x, u.y, u.w, u.uw, col)
		case "dotted":
			e.emitDottedLine(u.x, u.y, u.w, u.uw, col)
		case "wavy":
			e.emitWavyLine(u.x, u.y, u.w, u.uw, col)
		case "double":
			e.emitDoubleLine(u.x, u.y, u.w, u.uw, col)
		default:
			e.add(Op{ //nolint:exhaustruct // intentional zero fields
				Kind: OpLine, X: u.x, Y: u.y, W: u.w, H: 0,
				Width: u.uw, R: u.r, G: u.g, B: u.b,
			})
		}
		e.blendMode = prevBlend
	}

	*u = undRun{} //nolint:exhaustruct // intentional zero fields
}

// emitInlineBlock relocates a block-in-inline box onto the current line and
// returns the updated x cursor.
func (e *engine) emitInlineBlock(
	boxNode *box, item *inlineItem, leftX, lineY, lineH, baseline, justifyGap float64,
	gapAfter bool, und *undRun,
) float64 {
	und.flush(e)

	dx := leftX - item.blockBox.x
	dy := e.alignedInlineTop(item, lineY, lineH, baseline) - item.blockBox.y
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

func (e *engine) applyInlineImageBorders(item *inlineItem, leftX, top float64) (float64, float64, float64, float64) {
	imgX, imgY, imgW, imgH := leftX, top, item.w, item.h
	if item.style == nil || !inlineHasBorder(*item.style) {
		return imgX, imgY, imgW, imgH
	}

	if item.thumbImg {
		// Collapsed figure owns L/R/T; only the image/caption separator paints.
		e.emitThumbImageBottomSeparator(*item.style, leftX, top, item.w, item.h)

		return imgX, imgY, imgW, imgH
	}

	insetL := e.inlineChromeLeft(item.style)
	insetT := e.inlineChromeTop(item.style)
	insetR := e.inlineChromeRight(item.style)
	insetB := e.inlineChromeBottom(item.style)
	imgX += insetL
	imgY += insetT
	imgW -= insetL + insetR
	imgH -= insetT + insetB

	for _, op := range e.borderOps(*item.style, leftX, top, item.w, item.h) {
		e.add(op)
	}

	return imgX, imgY, imgW, imgH
}

// emitInlineImage places an image (or inline-block) item on the line and
// returns the updated x cursor. Replaced content never receives the href
// force-underline used for text links (that was the thumb hairline source).
func (e *engine) emitInlineImage(
	item *inlineItem, leftX, lineY, lineH, baseline, justifyGap float64,
	gapAfter bool, und *undRun,
) float64 {
	und.flush(e)

	top := e.alignedInlineTop(item, lineY, lineH, baseline)
	imgX, imgY, imgW, imgH := e.applyInlineImageBorders(item, leftX, top)

	if item.imgRef != nil && item.imgRef.data != nil && imgW > 0 && imgH > 0 {
		imgData := item.imgRef.data
		isJPEG := item.imgRef.isJPEG

		if item.style != nil && item.style.Filter != "" {
			filters := parseFilterList(item.style.Filter, item.style.Color, item.style.FontSize)
			imgData = applyImageFilterToImage(imgData, filters)
			isJPEG = false
		}

		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpImage, X: imgX, Y: imgY, W: imgW, H: imgH,
			Image: imgData, ImgW: item.imgRef.w, ImgH: item.imgRef.h, IsJPEG: isJPEG,
			Alt: item.alt,
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
	ascent, descent := e.inlineFontMetrics(item.text, item.style)

	if ascent+descent < size*0.5 {
		// Fallback when font metrics are missing — keep hit targets usable.
		ascent = size * fallbackAscentRatio
		descent = size * fallbackDescentRatio
	}

	chromeLeft, chromeRight := 0.0, 0.0
	if item.chrome {
		chromeLeft = e.inlineChromeLeft(item.style)
		chromeRight = e.inlineChromeRight(item.style)
	}

	contentWidth := item.w - chromeLeft - chromeRight
	if contentWidth < 0 {
		contentWidth = 0
	}

	if item.chrome {
		e.paintInlineChrome(item.style, leftX, baseline, ascent, descent, contentWidth)
	}

	leftX += chromeLeft

	if item.style.TextOverflow == textOverflowEllipsis {
		e.applyEllipsis(item)
	}

	runStart := leftX

	// Apply vertical-align <length> for text spans: positive raises.
	shiftedBaseline := baseline
	if item.style != nil {
		shiftedBaseline -= e.scalePt(e.effectiveVerticalAlignShift(item.style))
	}

	textBaseline := shiftedBaseline

	if isVerticalWritingMode(item.style.WritingMode) {
		// A rotated run uses the baseline as its vertical start. The normal
		// horizontal baseline already includes ascent, which would otherwise
		// push vertical labels down by one ascent before rotation.
		textBaseline -= ascent
	}

	leftX, runSpan := e.emitInlineFaceRuns(item, leftX, textBaseline, size, ascent, descent)

	// Decoration: one stroke per logical link run on this line (not per
	// face-run / nested style chunk). Thin stroke ~5% em, clamped for
	// dense reference print (min 0.25pt, max 0.45pt).
	// Force-underline a[href] for PDF affordance. Bare URL strings
	// (https://…, archive fragments) never get underlines — multi-line
	// ref lists were a forest of rules; titles/prose links still underline.
	e.paintDecoration(item, runStart, runSpan, size, ascent, descent, shiftedBaseline, child, und)
	e.paintEmphasis(item, runStart, runSpan, shiftedBaseline, ascent, descent, size)

	leftX += chromeRight + item.marginR
	if gapAfter && isJustifyGapAfter(*item) {
		leftX += justifyGap
	}

	return leftX
}

// emitInlineFaceRuns paints the primary face run (or the per-face fallback
// runs) of one text item and returns the updated x cursor and total span.
func (e *engine) emitInlineFaceRuns(
	item *inlineItem, leftX, textBaseline, size, ascent, descent float64,
) (float64, float64) {
	var runSpan float64

	if run, ok := e.primaryFaceRun(item.text, item.style); ok {
		e.emitInlineTextRun(item, run, leftX, textBaseline, size, ascent, descent)
		leftX += run.w
		runSpan = run.w

		return leftX, runSpan
	}

	for _, run := range e.splitTextByFace(item.text, item.style) {
		e.emitInlineTextRun(item, run, leftX, textBaseline, size, ascent, descent)
		leftX += run.w
		runSpan += run.w
	}

	return leftX, runSpan
}

func (e *engine) paintInlineChrome(style *ResolvedStyle, leftX, baseline, ascent, descent, contentWidth float64) {
	top := e.inlineChromeTop(style)
	bottom := e.inlineChromeBottom(style)
	lh := lineHeightOf(style) * e.scale
	extra := (lh - ascent - descent) / two
	boxY := baseline - ascent - extra - top
	boxH := ascent + extra + top + descent + extra + bottom
	boxW := contentWidth + e.inlineChromeLeft(style) + e.inlineChromeRight(style)

	if boxW <= 0 || boxH <= 0 {
		return
	}

	if style.BGColor[3] > 0 && e.opts.Background {
		radii, radiiY := usedBorderRadiiXY(*style, boxW, boxH)
		fill := Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpFillRect, X: leftX, Y: boxY, W: boxW, H: boxH,
			R: style.BGColor[0], G: style.BGColor[1], B: style.BGColor[2], Alpha: style.BGColor[3],
			Radius: uniformRadius(radii), RadiusTopLeft: radii[0], RadiusTopRight: radii[1],
			RadiusBottomRight: radii[2], RadiusBottomLeft: radii[3],
		}
		stampOneOpRadiiY(&fill, radiiY)
		e.add(fill)
	}

	if !inlineHasBorder(*style) {
		return
	}

	radii, radiiY := usedBorderRadiiXY(*style, boxW, boxH)
	radius := uniformRadius(radii)

	if hasRoundedRadii(radii) && uniformRoundedBorder(*style) {
		b := style.BorderTop
		stroke := Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpStrokeRect, X: leftX, Y: boxY, W: boxW, H: boxH,
			R: b.Color[0], G: b.Color[1], B: b.Color[2], Width: e.scalePt(borderPaint(b)), Radius: radius,
			RadiusTopLeft: radii[0], RadiusTopRight: radii[1], RadiusBottomRight: radii[2], RadiusBottomLeft: radii[3],
		}
		stampOneOpRadiiY(&stroke, radiiY)
		e.add(stroke)

		return
	}

	for _, op := range e.borderOps(*style, leftX, boxY, boxW, boxH) {
		e.add(op)
	}
}

func inlineHasBorder(st ResolvedStyle) bool {
	return inlineBorderVisible(st.BorderTop) || inlineBorderVisible(st.BorderRight) ||
		inlineBorderVisible(st.BorderBottom) || inlineBorderVisible(st.BorderLeft)
}

func writingModeRotate(mode string) float64 {
	if isVerticalWritingMode(mode) {
		return -90
	}

	return 0
}

// alignedInlineTop is the canvas Y of an atomic inline box (image or
// inline-block). Keywords match CSS vertical-align; a length shift raises
// (positive) or lowers (negative) a baseline-aligned box.
func (e *engine) alignedInlineTop(item *inlineItem, lineY, lineH, baseline float64) float64 {
	if item.style == nil {
		return baseline - item.h
	}

	switch item.style.VerticalAlign {
	case cssVerticalAlignTop:
		return lineY
	case cssVerticalAlignMiddle:
		return lineY + (lineH-item.h)/2
	case cssVerticalAlignBottom:
		return lineY + lineH - item.h
	default:
		return baseline - item.h - e.scalePt(e.effectiveVerticalAlignShift(item.style))
	}
}

// effectiveVerticalAlignShift maps vertical-align keywords and lengths to a
// pt shift where positive raises. Handles sub/super and % of line-height
// in addition to plain <length> stored in VerticalAlignShift.
func (e *engine) effectiveVerticalAlignShift(style *ResolvedStyle) float64 {
	if style == nil {
		return 0
	}

	switch strings.ToLower(strings.TrimSpace(style.VerticalAlign)) {
	case verticalAlignSub:
		return style.FontSize * verticalAlignSubRatio
	case verticalAlignSuper:
		return style.FontSize * -0.4
	}
	// If VerticalAlign looks like a percent (e.g. "50%"), compute against
	// line-height per CSS spec.
	if pct := strings.TrimSpace(style.VerticalAlign); strings.HasSuffix(pct, "%") {
		if percent, ok := parsePercent(pct); ok {
			lineH := lineHeightOf(style)

			return lineH * percent / oneHundred
		}
	}

	return style.VerticalAlignShift
}

func parsePercent(value string) (float64, bool) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasSuffix(trimmed, "%") {
		return 0, false
	}

	num := strings.TrimSpace(strings.TrimSuffix(trimmed, "%"))
	parsed, err := strconv.ParseFloat(num, 64)

	if err != nil {
		return 0, false
	}

	return parsed, true
}

func inlineBorderVisible(side border) bool {
	return side.Width > 0 && side.Style != cssDisplayNone
}

//nolint:wsl // transform width and alignment are one geometry decision
func (e *engine) emitInlineTextRun(
	item *inlineItem,
	run faceRun,
	leftX, baseline, size, ascent, descent float64,
) {
	child := item.style.Color
	text := transformInlineText(run.text, item.style.TextTransform)
	textWidth := run.w
	if text != run.text {
		textWidth = e.measureTextFace(text, item.style)
	}

	textX := leftX
	textDelta := textWidth - run.w
	switch item.style.TextAlign {
	case floatRight:
		textX -= textDelta
	case fxCenter:
		textX -= textDelta / two
	}

	if item.style.TextShadowSet {
		e.emitTextShadowRuns(item, run, textX, baseline, textWidth, size, descent)
	}

	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpText, X: textX, Y: baseline, W: textWidth, H: item.h,
		Text: run.text, Font: run.face, Size: size,
		InkDescent:    descent,
		LetterSpacing: item.style.LetterSpacing * e.scale,
		TextTransform: item.style.TextTransform,
		Bold:          item.style.FontWeight >= fontWeightBoldValue,
		R:             child[0], G: child[1], B: child[2],
		RotateDeg: writingModeRotate(item.style.WritingMode),
	})

	if item.href != "" {
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLinkURI, X: textX, Y: baseline - ascent, W: textWidth,
			H: ascent + descent, URI: item.href,
		})
	}
}

// emitTextShadowRuns paints one soft text-shadow copy per collected shadow.
func (e *engine) emitTextShadowRuns(
	item *inlineItem, run faceRun, textX, baseline, textWidth, size, descent float64,
) {
	shadows := e.collectTextShadows(item.style)

	for _, shadow := range shadows {
		opacity := shadowOpacity(shadow.blur)

		shadowOp := Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpText, X: textX + shadow.x, Y: baseline + shadow.y, W: textWidth, H: item.h,
			Text: run.text, Font: run.face, Size: size,
			InkDescent:    descent,
			LetterSpacing: item.style.LetterSpacing * e.scale,
			TextTransform: item.style.TextTransform,
			Bold:          item.style.FontWeight >= fontWeightBoldValue,
			R:             shadow.color[0], G: shadow.color[1], B: shadow.color[2],
			RotateDeg: writingModeRotate(item.style.WritingMode),
		}
		if opacity > 0 && opacity < 1 {
			shadowOp.PaintOpacity = opacity
		}

		e.add(shadowOp)
	}
}

// shadowOpacity fades a text shadow with its blur radius: sharp shadows stay
// opaque and wide ones fade toward the floor so dense print stays legible.
func shadowOpacity(blur float64) float64 {
	if blur <= 0 {
		return 0
	}

	opacity := 1.0 / (1.0 + blur*shadowBlurRatio)
	if opacity < shadowOpacityFloor {
		opacity = shadowOpacityFloor
	}

	if opacity > 1 {
		opacity = 1
	}

	return opacity
}

// paintDecoration draws the underline / line-through / overline strokes for one text
// item, extending the active underline run when the styling continues.
//
//nolint:cyclop // decoration painting
func (e *engine) paintDecoration(
	item *inlineItem, runStart, runSpan, size, ascent, descent, baseline float64,
	child [3]float64, und *undRun,
) {
	// A visible border-bottom is already the link affordance (wiki
	// `.mw-body a:not(.image){border-bottom:1px solid #aaa}`). Painting the
	// forced href underline on top makes every link look double.
	hasBottomBorder := inlineBorderVisible(item.style.BorderBottom)
	wantUnderline := !hasBottomBorder &&
		(item.style.TextDecoration == cssTextDecorationUnderline || forceLinkUnderline(item))
	wsOnly := strings.TrimSpace(item.text) == ""
	wantLineThrough := hasLineThrough(item.style)
	wantOverline := hasOverline(item.style)

	if runSpan <= emptyRunEpsilon {
		if !wantUnderline {
			und.flush(e)
		}

		return
	}

	if !wantUnderline && !wantLineThrough && !wantOverline {
		und.flush(e)

		return
	}

	decColor := child
	if item.style.TextDecorationColorSet {
		decColor = item.style.TextDecorationColor
	}

	uWidth := underlineStrokeWidth(size)
	if item.style.TextDecorationThickness > 0 {
		uWidth = item.style.TextDecorationThickness
	}

	if wantUnderline {
		// Sit clearly below glyph descenders (~1–2mm visual gap).
		underY := baseline + descent + size*0.22 + item.style.TextUnderlineOffset
		e.paintUnderline(item, runStart, runSpan, underY, uWidth, size, wsOnly, decColor, und)
	} else {
		und.flush(e)
	}

	e.paintLineThrough(item, runStart, runSpan, baseline, ascent, uWidth, wsOnly, decColor)

	if wantOverline && !wsOnly {
		e.paintOverline(item, runStart, runSpan, baseline, ascent, uWidth, decColor)
	}
}

// forceLinkUnderline reports whether a bare href forces an underline for PDF
// link affordance. Struck-through and overlined links keep only their own
// decoration.
func forceLinkUnderline(item *inlineItem) bool {
	if item.href == "" {
		return false
	}

	if item.style.TextDecoration == cssTextDecorationLineThrough {
		return false
	}

	if item.style.TextDecoration == cssTextDecorationOverline || hasOverline(item.style) {
		return false
	}

	return true
}

func hasOverline(style *ResolvedStyle) bool {
	if style == nil {
		return false
	}

	if style.TextDecoration == cssTextDecorationOverline {
		return true
	}

	return strings.Contains(strings.ToLower(style.TextDecorationLine), "overline")
}

func hasLineThrough(style *ResolvedStyle) bool {
	if style == nil {
		return false
	}

	if style.TextDecoration == cssTextDecorationLineThrough {
		return true
	}

	return strings.Contains(strings.ToLower(style.TextDecorationLine), "line-through")
}

// paintLineThrough strokes the strike-through rule for a decorated text item.
func (e *engine) paintLineThrough(
	item *inlineItem, runStart, runSpan, baseline, ascent, uWidth float64,
	wsOnly bool, child [3]float64,
) {
	if hasLineThrough(item.style) && !wsOnly {
		strikeY := baseline - ascent*lineThroughOffsetRatio

		col := child

		switch strings.ToLower(strings.TrimSpace(item.style.TextDecorationStyle)) {
		case "dashed":
			e.emitDashedLine(runStart, strikeY, runSpan, uWidth, col)
		case "dotted":
			e.emitDottedLine(runStart, strikeY, runSpan, uWidth, col)
		case "wavy":
			e.emitWavyLine(runStart, strikeY, runSpan, uWidth, col)
		case "double":
			e.emitDoubleLine(runStart, strikeY, runSpan, uWidth, col)
		default:
			e.add(Op{ //nolint:exhaustruct // intentional zero fields
				Kind: OpLine, X: runStart, Y: strikeY, W: runSpan, H: 0,
				Width: uWidth, R: col[0], G: col[1], B: col[2],
			})
		}
	}
}

// paintOverline strokes the overline rule for a decorated text item at the top.
func (e *engine) paintOverline(
	item *inlineItem, runStart, runSpan, baseline, ascent, uWidth float64,
	child [3]float64,
) {
	overlineY := baseline - ascent
	col := child

	switch strings.ToLower(strings.TrimSpace(item.style.TextDecorationStyle)) {
	case "dashed":
		e.emitDashedLine(runStart, overlineY, runSpan, uWidth, col)
	case "dotted":
		e.emitDottedLine(runStart, overlineY, runSpan, uWidth, col)
	case "wavy":
		e.emitWavyLine(runStart, overlineY, runSpan, uWidth, col)
	case "double":
		e.emitDoubleLine(runStart, overlineY, runSpan, uWidth, col)
	default:
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLine, X: runStart, Y: overlineY, W: runSpan, H: 0,
			Width: uWidth, R: col[0], G: col[1], B: col[2],
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
	und.style = item.style.TextDecorationStyle
	und.blendMode = e.blendMode
}

// startsActiveUnder reports that the item continues an active underline run:
// same href (or both href-less), same Y, same decoration style, and near
// enough in X — justify rivers / margins between nested chunks (up to ~2em)
// do not split it.
func startsActiveUnder(item *inlineItem, und *undRun, runStart, underY, size float64) bool {
	if !und.active || !nearUndY(und.y, underY) {
		return false
	}

	undStyle := strings.ToLower(strings.TrimSpace(und.style))
	itemStyle := strings.ToLower(strings.TrimSpace(item.style.TextDecorationStyle))

	if undStyle != itemStyle {
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
	uWidth := em * underlineWidthRatio
	if uWidth < underlineMinWidth {
		uWidth = underlineMinWidth
	}

	if uWidth > underlineMaxWidth {
		uWidth = underlineMaxWidth
	}

	return uWidth
}

func nearUndY(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}

	return d < underlineYTolerance
}

func (e *engine) emitDashedLine(startX, lineY, lineW, width float64, col [3]float64) {
	dash := width * dashLenRatio
	if dash < three {
		dash = three
	}

	gap := width * dashGapRatio
	if gap < two {
		gap = two
	}

	for cur := startX; cur < startX+lineW; {
		seg := dash
		if cur+seg > startX+lineW {
			seg = startX + lineW - cur
		}

		if seg > minVisibleSeg {
			e.add(Op{ //nolint:exhaustruct // intentional zero fields
				Kind: OpLine, X: cur, Y: lineY, W: seg, H: 0,
				Width: width, R: col[0], G: col[1], B: col[2],
			})
		}

		cur += dash + gap
	}
}

func (e *engine) emitDottedLine(startX, lineY, lineW, width float64, col [3]float64) {
	dot := width
	if dot < 1 {
		dot = 1
	}

	gap := width * dotGapRatio
	if gap < minDotGap {
		gap = minDotGap
	}

	for cur := startX; cur < startX+lineW; {
		seg := dot
		if cur+seg > startX+lineW {
			seg = startX + lineW - cur
		}

		if seg > minVisibleSeg {
			e.add(Op{ //nolint:exhaustruct // intentional zero fields
				Kind: OpLine, X: cur, Y: lineY, W: seg, H: 0,
				Width: width, R: col[0], G: col[1], B: col[2],
			})
		}

		cur += dot + gap
	}
}

func (e *engine) emitWavyLine(startX, lineY, lineW, width float64, col [3]float64) {
	amp := width * wavyAmpRatio
	if amp < minWavyAmp {
		amp = minWavyAmp
	}

	if amp > maxWavyAmp {
		amp = maxWavyAmp
	}

	segW := width * wavySegRatio
	if segW < three {
		segW = three
	}

	steps := int(lineW / segW)
	if steps < 1 {
		steps = 1
	}

	step := lineW / float64(steps)

	for segIdx := range steps {
		segStart := startX + float64(segIdx)*step
		segEnd := segStart + step

		if segEnd > startX+lineW {
			segEnd = startX + lineW
		}

		segStartY := lineY
		segEndY := lineY

		if segIdx%2 == 0 {
			segStartY -= amp
			segEndY += amp
		} else {
			segStartY += amp
			segEndY -= amp
		}

		segWidth := segEnd - segStart
		if segWidth < minVisibleSeg {
			continue
		}

		// approximate wave with two diagonals
		segMid := (segStart + segEnd) / two

		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLine, X: segStart, Y: segStartY, W: segMid - segStart, H: lineY - segStartY,
			Width: width, R: col[0], G: col[1], B: col[2],
		})

		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLine, X: segMid, Y: lineY, W: segEnd - segMid, H: segEndY - lineY,
			Width: width, R: col[0], G: col[1], B: col[2],
		})
	}
}

func (e *engine) emitDoubleLine(startX, lineY, lineW, width float64, col [3]float64) {
	gap := width + 1.0

	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpLine, X: startX, Y: lineY - gap/2, W: lineW, H: 0,
		Width: width, R: col[0], G: col[1], B: col[2],
	})

	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpLine, X: startX, Y: lineY + gap/2, W: lineW, H: 0,
		Width: width, R: col[0], G: col[1], B: col[2],
	})
}

// emphasisMark is one resolved text-emphasis dot style: geometry plus color.
type emphasisMark struct {
	dotD, dotY                   float64
	isCircle, isTriangle, isOpen bool
	red, green, blue             float64
}

func (e *engine) paintEmphasis(item *inlineItem, runStart, runSpan, baseline, ascent, descent, size float64) {
	mark, ok := resolveEmphasisMark(item, baseline, ascent, descent, size)
	if !ok {
		return
	}

	curX := runStart

	for _, runic := range item.text {
		if runic == ' ' || runic == '\t' {
			curX += e.measureRuneFace(runic, item.style)

			continue
		}

		adv := e.measureRuneFace(runic, item.style)
		if adv <= 0 {
			adv = size * fallbackAdvanceRatio
		}

		cx := curX + adv/two - mark.dotD/two
		e.emitEmphasisDot(cx, mark.dotY, mark.dotD, mark.isCircle, mark.isTriangle, mark.isOpen,
			mark.red, mark.green, mark.blue)

		curX += adv

		if curX > runStart+runSpan+0.5 {
			break
		}
	}
}

// hasEmphasisProps reports whether any text-emphasis custom prop is set.
func hasEmphasisProps(style *ResolvedStyle) bool {
	if style == nil || style.CustomProps == nil {
		return false
	}

	return style.CustomProps["__emph_style"] != "" ||
		style.CustomProps["__emph_color"] != "" ||
		style.CustomProps["__emph_position"] != "" ||
		style.CustomProps["__emph_skip"] != ""
}

// resolveEmphasisMark reads the __emph_* custom props into paint-ready mark
// geometry. It reports false when emphasis is absent or disabled.
func resolveEmphasisMark(item *inlineItem, baseline, ascent, descent, size float64) (emphasisMark, bool) {
	var mark emphasisMark

	if !hasEmphasisProps(item.style) {
		return mark, false
	}

	styleVal := item.style.CustomProps["__emph_style"]
	colorStr := item.style.CustomProps["__emph_color"]
	posStr := item.style.CustomProps["__emph_position"]

	if strings.EqualFold(styleVal, inlineStyleNone) {
		return mark, false
	}

	if styleVal == "" {
		styleVal = emphasisStyleFilled
	}

	mark.red, mark.green, mark.blue = item.style.Color[0], item.style.Color[1], item.style.Color[2]

	if colorStr != "" {
		if parsed, ok := parseUsedColor(colorStr, item.style.Color); ok {
			mark.red, mark.green, mark.blue = parsed[0], parsed[1], parsed[2]
		}
	}

	lowStyle := strings.ToLower(styleVal)
	mark.isCircle = strings.Contains(lowStyle, "circle") || strings.Contains(lowStyle, "dot")
	mark.isTriangle = strings.Contains(lowStyle, "triangle")
	mark.isOpen = strings.Contains(lowStyle, "open")

	isUnder := strings.Contains(strings.ToLower(posStr), "under")
	mark.dotD, mark.dotY = emphasisDotGeometry(size, ascent, descent, baseline, isUnder)

	return mark, true
}

// emphasisDotGeometry sizes one text-emphasis dot and returns its diameter
// and canvas Y above or below the text.
func emphasisDotGeometry(size, ascent, descent, baseline float64, isUnder bool) (float64, float64) {
	dotD := size * emphasisDotRatio
	if dotD < minEmphasisDot {
		dotD = minEmphasisDot
	}

	if dotD > maxEmphasisDot {
		dotD = maxEmphasisDot
	}

	var dotY float64

	if isUnder {
		dotY = baseline + descent + dotD + emphasisUnderGap
	} else {
		dotY = baseline - ascent - dotD - 1.0
	}

	return dotD, dotY
}

// emitEmphasisDot paints one text-emphasis mark: a triangle reads as a square
// (true triangles need path ops), open marks stroke, solid marks fill.
func (e *engine) emitEmphasisDot(
	centerX, dotY, dotD float64, isCircle, isTriangle, isOpen bool, red, green, blue float64,
) {
	radius := 0.0
	if isCircle {
		radius = dotD / two
	}

	switch {
	case isTriangle:
		// triangle emphasis: keep visible as square with slight offset
		// (true triangle would need path ops; square is distinguishable and visible)
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpFillRect, X: centerX, Y: dotY, W: dotD, H: dotD,
			R: red, G: green, B: blue, Alpha: 1,
		})
	case isOpen:
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpStrokeRect, X: centerX, Y: dotY, W: dotD, H: dotD,
			R: red, G: green, B: blue, Alpha: 1, Width: emphasisOpenStrokeWidth, Radius: radius,
		})
	default:
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpFillRect, X: centerX, Y: dotY, W: dotD, H: dotD,
			R: red, G: green, B: blue, Alpha: 1, Radius: radius,
		})
	}
}

// ellipsisAvail resolves the width budget for text-overflow ellipsis from the
// style width, the inline containing block, or the page width.
func (e *engine) ellipsisAvail(item *inlineItem) (float64, bool) {
	switch {
	case item.style.Width > 0:
		return e.scalePt(item.style.Width), true
	case e.inlineCBW > 0:
		return e.inlineCBW, true
	case e.opts.Width > 0:
		return e.opts.Width, true
	default:
		return 0, false
	}
}

func (e *engine) applyEllipsis(item *inlineItem) {
	if item.style == nil {
		return
	}

	if item.text == "" {
		return
	}

	avail, ok := e.ellipsisAvail(item)
	if !ok {
		return
	}

	if avail < minEllipsisAvail {
		return
	}

	totalW := e.measureTextFace(item.text, item.style)
	if totalW <= avail+0.5 {
		return
	}

	ellipsis := "..."
	ellipsisW := e.measureTextFace(ellipsis, item.style)
	budget := avail - ellipsisW - 1

	if budget <= 0 {
		item.text = ellipsis
		item.w = ellipsisW

		return
	}

	trimmed := e.truncateForEllipsis(item.text, budget, item.style)
	if trimmed == "" {
		item.text = ellipsis
		item.w = ellipsisW

		return
	}

	item.text = trimmed + ellipsis
	item.w = e.measureTextFace(item.text, item.style)
}

func (e *engine) truncateForEllipsis(text string, budget float64, sty *ResolvedStyle) string {
	var runWidth float64

	end := 0

	for idx, runic := range text {
		adv := e.measureRuneFace(runic, sty)
		if runWidth+adv > budget {
			break
		}

		runWidth += adv
		end = idx + len(string(runic))
	}

	if end <= 0 {
		return ""
	}

	// avoid cutting mid-word leaving trailing space
	res := strings.TrimRight(text[:end], " ")

	return res
}

// emitInlineBullet paints the list-item marker for a bullet item.
//
//nolint:unused // list-item bullet painter kept with its marker and display helpers in pseudo_content.go
func (e *engine) emitInlineBullet(item *inlineItem, leftX, baseline, size float64) {
	if item.style == nil {
		return
	}

	typ := strings.ToLower(strings.TrimSpace(item.style.ListStyleType))
	if typ == "" || typ == inlineStyleNone {
		return
	}

	// Only for display:list-item; text items inherit that display from parent div
	if !isDisplayListItem(item.style) {
		return
	}

	// Dedupe: avoid double bullet when emitInlineText is called per word on same line.
	if e.hasInlineBullet(baseline) {
		return
	}

	text := listItemMarkerText(*item.style, nil)
	face := e.faceFor(item.style)

	if face == nil {
		face = e.font
	}

	const markerGapRatio = 0.35

	markerW := e.measureTextFace(text, item.style)
	posX := leftX - markerW - size*markerGapRatio

	if posX < 0 {
		posX = 0
	}

	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpBullet, X: posX, Y: baseline, Text: text, Font: face, Size: size,
		InkDescent: e.fontDescentFace(face, size),
		R:          item.style.Color[0], G: item.style.Color[1], B: item.style.Color[2],
	})
}

// hasInlineBullet reports whether a bullet marker is already painted at the
// baseline: emitInlineText runs per word on a line, so markers dedupe.
//
//nolint:unused // dedupe helper for the retained list-item bullet painter
func (e *engine) hasInlineBullet(baseline float64) bool {
	for idx := len(e.ops) - 1; idx >= 0 && idx >= len(e.ops)-4; idx-- {
		if e.ops[idx].Kind == OpBullet && e.ops[idx].Y == baseline && e.ops[idx].Text != "" {
			return true
		}
	}

	return false
}

// measureTextFace measures s using per-rune CSS font-family fallback
// (same face selection as paint).
//
// Primary-face fast path: resolve the style face once and only consult the
// fallback cache when a glyph is missing — Latin report text almost never
// leaves the primary face.
func (e *engine) measureTextFace(cssSheet string, sty *ResolvedStyle) float64 {
	if cssSheet == "" || sty == nil {
		return 0
	}

	size := sty.FontSize * e.scale
	lstyle := sty.LetterSpacing * e.scale
	wstyle := sty.WordSpacing * e.scale
	primary := e.faceFor(sty)

	if primary == nil {
		primary = e.font
	}

	total, runeCount, spaceCount := e.accumulateTextFaceWidth(cssSheet, sty, primary, size)

	if lstyle != 0 && runeCount > 0 {
		total += lstyle * float64(runeCount)
	}

	if wstyle != 0 && spaceCount > 0 {
		total += wstyle * float64(spaceCount)
	}

	return total
}

func (e *engine) accumulateTextFaceWidth(
	cssSheet string, sty *ResolvedStyle, primary *pdf.Font, size float64,
) (float64, int, int) {
	var total float64

	runeCount := 0
	spaceCount := 0

	for _, runic := range cssSheet {
		if runic == '\t' {
			total += e.tabStopAdvance(sty, primary, size)
			runeCount++

			continue
		}

		face := primary
		if !isRuneWhitespace(runic) && primary.GlyphID(runic) == 0 {
			face = e.faceForRuneFallback(sty, runic, primary)
			if face == nil {
				face = e.font
			}
		}

		total += face.GlyphAdvancePoints(runic, size)
		runeCount++

		if runic == ' ' {
			spaceCount++
		}
	}

	return total, runeCount, spaceCount
}

// tabStopAdvance returns the advance width of a tab stop: an absolute length
// when tab-size is a length, else TabSize (default 8) space widths.
func (e *engine) tabStopAdvance(sty *ResolvedStyle, primary *pdf.Font, size float64) float64 {
	if isTabSizeLength(sty) {
		return e.scalePt(sty.TabSize)
	}

	tabSize := sty.TabSize
	if tabSize <= 0 {
		tabSize = 8
	}

	spaceW := primary.GlyphAdvancePoints(' ', size)
	if spaceW <= 0 {
		spaceW = size * tabSpaceRatio
	}

	return spaceW * tabSize
}

// measureRuneFace measures a single rune with the same face selection as
// measureTextFace, without allocating string(r).
func (e *engine) measureRuneFace(curRune rune, sty *ResolvedStyle) float64 {
	if sty == nil {
		return 0
	}

	if curRune == '\t' {
		if isTabSizeLength(sty) {
			return e.scalePt(sty.TabSize)
		}

		tabSize := sty.TabSize
		if tabSize <= 0 {
			tabSize = 8
		}

		// \t expands to TabSize * space width per CSS tab-size.
		spaceW := e.measureRuneFace(' ', sty)
		if spaceW <= 0 {
			spaceW = sty.FontSize * e.scale * tabSpaceRatio
		}

		return spaceW * tabSize
	}

	size := sty.FontSize * e.scale
	face := e.faceForRune(sty, curRune)

	if face == nil {
		face = e.font
	}

	advance := face.GlyphAdvancePoints(curRune, size)
	if sty.LetterSpacing != 0 {
		advance += sty.LetterSpacing * e.scale
	}

	if curRune == ' ' && sty.WordSpacing != 0 {
		advance += sty.WordSpacing * e.scale
	}

	return advance
}

type faceRun struct {
	text string
	face *pdf.Font
	w    float64
}

// splitTextByFace splits s into contiguous runs that share the same face
// under CSS font-family fallback.
//
//nolint:cyclop,funlen,wsl // hot path: per-rune face-fallback run splitting
func (e *engine) splitTextByFace(cssSheet string, sty *ResolvedStyle) []faceRun {
	if cssSheet == "" || sty == nil {
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
	runeCount := 0
	spaceCount := 0

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
				w: width + sty.LetterSpacing*e.scale*float64(runeCount) +
					sty.WordSpacing*e.scale*float64(spaceCount),
			})
			start = idx
			width = 0
			runeCount = 0
			spaceCount = 0
		}

		if current == nil {
			start = idx
		}

		current = face
		width += face.AdvanceInPoints(runic, size)
		runeCount++

		if runic == ' ' {
			spaceCount++
		}
	}

	if current != nil {
		runs = append(runs, faceRun{
			text: cssSheet[start:],
			face: current,
			w: width + sty.LetterSpacing*e.scale*float64(runeCount) +
				sty.WordSpacing*e.scale*float64(spaceCount),
		})
	}

	return runs
}

//nolint:wsl // hot path keeps the primary-face fast path compact
func (e *engine) primaryFaceRun(cssSheet string, sty *ResolvedStyle) (faceRun, bool) {
	if cssSheet == "" || sty == nil {
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
	paintText := transformInlineText(cssSheet, sty.TextTransform)

	var width float64
	runeCount := 0
	spaceCount := 0

	for _, runic := range paintText {
		width += primary.AdvanceInPoints(runic, size)
		runeCount++

		if runic == ' ' {
			spaceCount++
		}
	}

	return faceRun{
		text: cssSheet,
		face: primary,
		w: width + sty.LetterSpacing*e.scale*float64(runeCount) +
			sty.WordSpacing*e.scale*float64(spaceCount),
	}, true
}

// faceRunAllPrimary reports that every non-whitespace rune in s is covered by
// primary (so splitTextByFace can emit a single run).
func faceRunAllPrimary(cssSheet string, primary *pdf.Font) bool {
	if primary == nil {
		return false
	}

	for _, runic := range cssSheet {
		if !isRuneWhitespace(runic) && primary.GlyphID(runic) == 0 {
			return false
		}
	}

	return true
}

func (e *engine) fontAscent(size float64) float64 {
	return e.fontAscentFace(e.font, size)
}

func (e *engine) fontAscentFace(face *pdf.Font, size float64) float64 {
	if face == nil || face.UnitsPerEm() <= 0 {
		return size * fallbackAscentRatio
	}

	return float64(face.Ascent()) * size / float64(face.UnitsPerEm())
}

func (e *engine) fontDescentFace(face *pdf.Font, size float64) float64 {
	if face == nil || face.UnitsPerEm() <= 0 {
		return size * fallbackDescentRatio
	}

	return float64(-face.Descent()) * size / float64(face.UnitsPerEm())
}

// inlineFontMetrics returns the largest ascent/descent pair used by an inline
// item. A line can contain a mix of sans, serif, mono, and fallback faces;
// using the engine default face for all of them shifts baselines and gives
// inline chrome the wrong height.
func (e *engine) inlineFontMetrics(text string, style *ResolvedStyle) (float64, float64) {
	if style == nil {
		return 0, 0
	}

	size := style.FontSize * e.scale
	runs := e.splitTextByFace(text, style)

	if len(runs) == 0 {
		runs = []faceRun{{face: e.faceFor(style)}} //nolint:exhaustruct // text and width are filled by shaping
	}

	maxAscent, maxDescent := 0.0, 0.0

	for _, run := range runs {
		ascent := e.fontAscentFace(run.face, size)
		descent := e.fontDescentFace(run.face, size)

		if ascent > maxAscent {
			maxAscent = ascent
		}

		if descent > maxDescent {
			maxDescent = descent
		}
	}

	return maxAscent, maxDescent
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

func isTabSizeLength(sty *ResolvedStyle) bool {
	if sty == nil || sty.CustomProps == nil {
		return false
	}

	return sty.CustomProps["__tab_size_is_length"] == "1"
}

type shadowPaint struct {
	x, y, blur float64
	color      [3]float64
}

func (e *engine) collectTextShadows(sty *ResolvedStyle) []shadowPaint {
	if sty == nil || !sty.TextShadowSet {
		return nil
	}

	base := shadowPaint{x: sty.TextShadowX, y: sty.TextShadowY, blur: sty.TextShadowBlur, color: sty.TextShadowColor}
	shadows := []shadowPaint{base}

	if sty.CustomProps != nil {
		if extra, ok := sty.CustomProps["__text_shadow_extra"]; ok && extra != "" {
			for _, part := range strings.Split(extra, "|") {
				if spec, ok := shadowDecode(part); ok {
					shadows = append(shadows, shadowPaint(spec))
				}
			}
		}
	}

	return shadows
}
