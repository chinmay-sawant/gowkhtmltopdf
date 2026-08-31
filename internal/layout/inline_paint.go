package layout

import (
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
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
	style   string
}

// flush emits the accumulated underline stroke, if any.
func (u *undRun) flush(e *engine) {
	if u.active && u.w > 0.01 {
		col := [3]float64{u.r, u.g, u.b}
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
func (e *engine) emitInlineText( //nolint:funlen // text measurement, face-run emission, and decoration share one cursor
	item *inlineItem, leftX, baseline, justifyGap float64,
	gapAfter bool, und *undRun,
) float64 {
	child := item.style.Color
	size := item.style.FontSize * e.scale
	ascent, descent := e.inlineFontMetrics(item.text, item.style)

	if ascent+descent < size*0.5 {
		// Fallback when font metrics are missing — keep hit targets usable.
		ascent = size * 0.8
		descent = size * 0.2
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

	if item.style.TextOverflow == "ellipsis" {
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

	var runSpan float64

	if run, ok := e.primaryFaceRun(item.text, item.style); ok {
		e.emitInlineTextRun(
			item,
			run,
			leftX,
			textBaseline,
			size,
			ascent,
			descent,
		)

		leftX += run.w
		runSpan = run.w
	} else {
		for _, run := range e.splitTextByFace(item.text, item.style) {
			e.emitInlineTextRun(
				item,
				run,
				leftX,
				textBaseline,
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
	e.paintDecoration(item, runStart, runSpan, size, ascent, descent, shiftedBaseline, child, und)
	e.paintEmphasis(item, runStart, runSpan, shiftedBaseline, ascent, descent, size)

	leftX += chromeRight + item.marginR
	if gapAfter && isJustifyGapAfter(*item) {
		leftX += justifyGap
	}

	return leftX
}

func (e *engine) paintInlineChrome(style *ResolvedStyle, leftX, baseline, ascent, descent, contentWidth float64) {
	top := e.inlineChromeTop(style)
	bottom := e.inlineChromeBottom(style)
	lh := lineHeightOf(style) * e.scale
	extra := (lh - ascent - descent) / 2
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
func (e *engine) effectiveVerticalAlignShift(st *ResolvedStyle) float64 {
	if st == nil {
		return 0
	}
	switch strings.ToLower(strings.TrimSpace(st.VerticalAlign)) {
	case "sub":
		return st.FontSize * 0.2
	case "super":
		return st.FontSize * -0.4
	}
	// If VerticalAlign looks like a percent (e.g. "50%"), compute against
	// line-height per CSS spec.
	if pct := strings.TrimSpace(st.VerticalAlign); strings.HasSuffix(pct, "%") {
		if v, err := parsePercent(pct); err == nil {
			lh := lineHeightOf(st)
			return lh * v / 100
		}
	}
	return st.VerticalAlignShift
}

func parsePercent(s string) (float64, error) {
	s = strings.TrimSpace(s)
	if !strings.HasSuffix(s, "%") {
		return 0, strconv.ErrSyntax
	}
	num := strings.TrimSpace(strings.TrimSuffix(s, "%"))
	return strconv.ParseFloat(num, 64)
}

func inlineBorderVisible(side border) bool {
	return side.Width > 0 && side.Style != cssDisplayNone
}

//nolint:wsl,mnd // transform width and alignment are one geometry decision
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
		textX -= textDelta / 2
	}

	if item.style.TextShadowSet {
		shadows := e.collectTextShadows(item.style)
		for _, sh := range shadows {
			opacity := 0.0
			if sh.blur > 0 {
				opacity = 1.0 / (1.0 + sh.blur*0.3)
				if opacity < 0.2 {
					opacity = 0.2
				}
				if opacity > 1 {
					opacity = 1
				}
			}
			op := Op{ //nolint:exhaustruct // intentional zero fields
				Kind: OpText, X: textX + sh.x, Y: baseline + sh.y, W: textWidth, H: item.h,
				Text: run.text, Font: run.face, Size: size,
				InkDescent:    descent,
				LetterSpacing: item.style.LetterSpacing * e.scale,
				TextTransform: item.style.TextTransform,
				Bold:          item.style.FontWeight >= 700,
				R:             sh.color[0], G: sh.color[1], B: sh.color[2],
				RotateDeg: writingModeRotate(item.style.WritingMode),
			}
			if opacity > 0 && opacity < 1 {
				op.PaintOpacity = opacity
			}
			e.add(op)
		}
	}

	e.add(Op{ //nolint:exhaustruct // intentional zero fields
		Kind: OpText, X: textX, Y: baseline, W: textWidth, H: item.h,
		Text: run.text, Font: run.face, Size: size,
		InkDescent:    descent,
		LetterSpacing: item.style.LetterSpacing * e.scale,
		TextTransform: item.style.TextTransform,
		Bold:          item.style.FontWeight >= 700,
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
		(item.style.TextDecoration == cssTextDecorationUnderline ||
			(item.href != "" && item.style.TextDecoration != cssTextDecorationLineThrough && item.style.TextDecoration != cssTextDecorationOverline && !hasOverline(item.style)))
	wsOnly := strings.TrimSpace(item.text) == ""
	wantLineThrough := hasLineThrough(item.style)
	wantOverline := hasOverline(item.style)

	if runSpan <= 0.01 {
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

func hasOverline(st *ResolvedStyle) bool {
	if st == nil {
		return false
	}
	if st.TextDecoration == cssTextDecorationOverline {
		return true
	}
	return strings.Contains(strings.ToLower(st.TextDecorationLine), "overline")
}

func hasLineThrough(st *ResolvedStyle) bool {
	if st == nil {
		return false
	}
	if st.TextDecoration == cssTextDecorationLineThrough {
		return true
	}
	return strings.Contains(strings.ToLower(st.TextDecorationLine), "line-through")
}

// paintLineThrough strokes the strike-through rule for a decorated text item.
func (e *engine) paintLineThrough(
	item *inlineItem, runStart, runSpan, baseline, ascent, uWidth float64,
	wsOnly bool, child [3]float64,
) {
	if hasLineThrough(item.style) && !wsOnly {
		y := baseline - ascent*0.3
		col := child
		switch strings.ToLower(strings.TrimSpace(item.style.TextDecorationStyle)) {
		case "dashed":
			e.emitDashedLine(runStart, y, runSpan, uWidth, col)
		case "dotted":
			e.emitDottedLine(runStart, y, runSpan, uWidth, col)
		case "wavy":
			e.emitWavyLine(runStart, y, runSpan, uWidth, col)
		case "double":
			e.emitDoubleLine(runStart, y, runSpan, uWidth, col)
		default:
			e.add(Op{ //nolint:exhaustruct // intentional zero fields
				Kind: OpLine, X: runStart, Y: y, W: runSpan, H: 0,
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
	y := baseline - ascent
	col := child
	switch strings.ToLower(strings.TrimSpace(item.style.TextDecorationStyle)) {
	case "dashed":
		e.emitDashedLine(runStart, y, runSpan, uWidth, col)
	case "dotted":
		e.emitDottedLine(runStart, y, runSpan, uWidth, col)
	case "wavy":
		e.emitWavyLine(runStart, y, runSpan, uWidth, col)
	case "double":
		e.emitDoubleLine(runStart, y, runSpan, uWidth, col)
	default:
		e.add(Op{ //nolint:exhaustruct // intentional zero fields
			Kind: OpLine, X: runStart, Y: y, W: runSpan, H: 0,
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
}

// startsActiveUnder reports that the item continues an active underline run:
// same href (or both href-less), same Y, same decoration style, and near
// enough in X — justify rivers / margins between nested chunks (up to ~2em)
// do not split it.
func startsActiveUnder(item *inlineItem, und *undRun, runStart, underY, size float64) bool {
	if !und.active || !nearUndY(und.y, underY) {
		return false
	}
	if strings.ToLower(strings.TrimSpace(und.style)) != strings.ToLower(strings.TrimSpace(item.style.TextDecorationStyle)) {
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
	uWidth := em * 0.05
	if uWidth < 0.25 {
		uWidth = 0.25
	}

	if uWidth > 0.45 {
		uWidth = 0.45
	}

	return uWidth
}

func nearUndY(a, b float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}

	return d < 0.5
}

func (e *engine) emitDashedLine(x, y, w, width float64, col [3]float64) {
	dash := width * 4
	if dash < 3 {
		dash = 3
	}
	gap := width * 2.8
	if gap < 2 {
		gap = 2
	}
	for cur := x; cur < x+w; {
		seg := dash
		if cur+seg > x+w {
			seg = x + w - cur
		}
		if seg > 0.2 {
			e.add(Op{Kind: OpLine, X: cur, Y: y, W: seg, H: 0, Width: width, R: col[0], G: col[1], B: col[2]})
		}
		cur += dash + gap
	}
}

func (e *engine) emitDottedLine(x, y, w, width float64, col [3]float64) {
	dot := width
	if dot < 1 {
		dot = 1
	}
	gap := width * 2.5
	if gap < 2.5 {
		gap = 2.5
	}
	for cur := x; cur < x+w; {
		seg := dot
		if cur+seg > x+w {
			seg = x + w - cur
		}
		if seg > 0.2 {
			e.add(Op{Kind: OpLine, X: cur, Y: y, W: seg, H: 0, Width: width, R: col[0], G: col[1], B: col[2]})
		}
		cur += dot + gap
	}
}

func (e *engine) emitWavyLine(x, y, w, width float64, col [3]float64) {
	amp := width * 1.2
	if amp < 0.8 {
		amp = 0.8
	}
	if amp > 1.8 {
		amp = 1.8
	}
	segW := width * 2.6
	if segW < 3 {
		segW = 3
	}
	steps := int(w / segW)
	if steps < 1 {
		steps = 1
	}
	step := w / float64(steps)
	for i := 0; i < steps; i++ {
		x0 := x + float64(i)*step
		x1 := x0 + step
		if x1 > x+w {
			x1 = x + w
		}
		y0 := y
		y1 := y
		if i%2 == 0 {
			y0 -= amp
			y1 += amp
		} else {
			y0 += amp
			y1 -= amp
		}
		segWidth := x1 - x0
		if segWidth < 0.2 {
			continue
		}
		// approximate wave with two diagonals
		midX := (x0 + x1) / 2
		e.add(Op{Kind: OpLine, X: x0, Y: y0, W: midX - x0, H: y - y0, Width: width, R: col[0], G: col[1], B: col[2]})
		e.add(Op{Kind: OpLine, X: midX, Y: y, W: x1 - midX, H: y1 - y, Width: width, R: col[0], G: col[1], B: col[2]})
	}
}

func (e *engine) emitDoubleLine(x, y, w, width float64, col [3]float64) {
	gap := width + 1.0
	e.add(Op{Kind: OpLine, X: x, Y: y - gap/2, W: w, H: 0, Width: width, R: col[0], G: col[1], B: col[2]})
	e.add(Op{Kind: OpLine, X: x, Y: y + gap/2, W: w, H: 0, Width: width, R: col[0], G: col[1], B: col[2]})
}

func (e *engine) paintEmphasis(item *inlineItem, runStart, runSpan, baseline, ascent, descent, size float64) {
	if item.style == nil || item.style.CustomProps == nil {
		return
	}
	styleVal := item.style.CustomProps["__emph_style"]
	colorStr := item.style.CustomProps["__emph_color"]
	posStr := item.style.CustomProps["__emph_position"]
	skipVal := item.style.CustomProps["__emph_skip"]
	if styleVal == "" && colorStr == "" && posStr == "" && skipVal == "" {
		return
	}
	if strings.EqualFold(styleVal, "none") {
		return
	}
	if styleVal == "" {
		styleVal = "filled"
	}
	r, g, b := item.style.Color[0], item.style.Color[1], item.style.Color[2]
	if colorStr != "" {
		if c, ok := parseUsedColor(colorStr, item.style.Color); ok {
			r, g, b = c[0], c[1], c[2]
		}
	}
	isUnder := strings.Contains(strings.ToLower(posStr), "under")
	dotD := size * 0.18
	if dotD < 1.1 {
		dotD = 1.1
	}
	if dotD > 2.4 {
		dotD = 2.4
	}
	var dotY float64
	if isUnder {
		dotY = baseline + descent + dotD + 1.2
	} else {
		dotY = baseline - ascent - dotD - 1.0
	}
	curX := runStart
	lowStyle := strings.ToLower(styleVal)
	isCircle := strings.Contains(lowStyle, "circle") || strings.Contains(lowStyle, "dot")
	isTriangle := strings.Contains(lowStyle, "triangle")
	isOpen := strings.Contains(lowStyle, "open")
	for _, rn := range item.text {
		if rn == ' ' {
			curX += e.measureRuneFace(rn, item.style)
			continue
		}
		if rn == '\t' {
			curX += e.measureRuneFace(rn, item.style)
			continue
		}
		adv := e.measureRuneFace(rn, item.style)
		if adv <= 0 {
			adv = size * 0.5
		}
		cx := curX + adv/2 - dotD/2
		if isTriangle {
			// triangle emphasis: keep visible as square with slight offset
			// (true triangle would need path ops; square is distinguishable and visible)
			e.add(Op{Kind: OpFillRect, X: cx, Y: dotY, W: dotD, H: dotD, R: r, G: g, B: b, Alpha: 1})
		} else if isOpen {
			radius := 0.0
			if isCircle {
				radius = dotD / 2
			}
			e.add(Op{Kind: OpStrokeRect, X: cx, Y: dotY, W: dotD, H: dotD, R: r, G: g, B: b, Alpha: 1, Width: 0.4, Radius: radius})
		} else {
			radius := 0.0
			if isCircle {
				radius = dotD / 2
			}
			e.add(Op{Kind: OpFillRect, X: cx, Y: dotY, W: dotD, H: dotD, R: r, G: g, B: b, Alpha: 1, Radius: radius})
		}
		curX += adv
		if curX > runStart+runSpan+0.5 {
			break
		}
	}
}

func (e *engine) applyEllipsis(item *inlineItem) {
	if item.style == nil {
		return
	}
	if item.text == "" {
		return
	}
	avail := 0.0
	if item.style.Width > 0 {
		avail = e.scalePt(item.style.Width)
	} else if e.inlineCBW > 0 {
		avail = e.inlineCBW
	} else if e.opts.Width > 0 {
		avail = e.opts.Width
	} else {
		return
	}
	if avail < 10 {
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
	var w float64
	end := 0
	for idx, rn := range text {
		adv := e.measureRuneFace(rn, sty)
		if w+adv > budget {
			break
		}
		w += adv
		end = idx + len(string(rn))
	}
	if end <= 0 {
		return ""
	}
	// avoid cutting mid-word leaving trailing space
	res := strings.TrimRight(text[:end], " ")
	return res
}

func (e *engine) emitInlineBullet(item *inlineItem, leftX, baseline, size float64) {
	if item.style == nil {
		return
	}
	typ := strings.ToLower(strings.TrimSpace(item.style.ListStyleType))
	if typ == "" || typ == "none" {
		return
	}
	// Only for display:list-item; text items inherit that display from parent div
	if !isDisplayListItem(item.style) {
		return
	}
	// Dedupe: avoid double bullet when emitInlineText is called per word on same line.
	for i := len(e.ops) - 1; i >= 0 && i >= len(e.ops)-4; i-- {
		if e.ops[i].Kind == OpBullet && e.ops[i].Y == baseline && e.ops[i].Text != "" {
			return
		}
	}
	text := listItemMarkerText(*item.style, nil)
	face := e.faceFor(item.style)
	if face == nil {
		face = e.font
	}
	markerW := e.measureTextFace(text, item.style)
	posX := leftX - markerW - size*0.35
	if posX < 0 {
		posX = 0
	}
	e.add(Op{Kind: OpBullet, X: posX, Y: baseline, Text: text, Font: face, Size: size, InkDescent: e.fontDescentFace(face, size), R: item.style.Color[0], G: item.style.Color[1], B: item.style.Color[2]})
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
			if isTabSizeLength(sty) {
				total += e.scalePt(sty.TabSize)
			} else {
				tabSize := sty.TabSize
				if tabSize <= 0 {
					tabSize = 8
				}
				spaceW := primary.GlyphAdvancePoints(' ', size)
				if spaceW <= 0 {
					spaceW = size * 0.25
				}
				total += spaceW * tabSize
			}
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
			spaceW = sty.FontSize * e.scale * 0.25
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
		return size * 0.8
	}

	return float64(face.Ascent()) * size / float64(face.UnitsPerEm())
}

func (e *engine) fontDescentFace(face *pdf.Font, size float64) float64 {
	if face == nil || face.UnitsPerEm() <= 0 {
		return size * 0.2
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
				if sp, ok := shadowDecode(part); ok {
					shadows = append(shadows, shadowPaint{x: sp.x, y: sp.y, blur: sp.blur, color: sp.color})
				}
			}
		}
	}
	return shadows
}
