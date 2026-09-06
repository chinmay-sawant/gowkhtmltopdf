package layout

const (
	outlineSideHint        = 4 // four sides of a rectangular outline
	outlineBothSidesFactor = 2 // outline inflates both sides of the box
	mediumOutlineWidthPt   = 3 // fallback when medium width resolves to 0
)

// effectiveOutline returns the effective outline width/style after applying
// CSS initial values for isolated longhands: width defaults to medium (3px)
// when style is solid but width is 0, and style defaults to solid when
// width>0 or color is set but style is empty/none. This makes
// outline-color/style/width demos visible (fixture-62 rows 22,24,25).
func effectiveOutline(sty *ResolvedStyle) (float64, string) {
	if sty == nil {
		return 0, ""
	}

	outlineStyle := defaultedOutlineStyle(sty)

	return defaultedOutlineWidth(sty, outlineStyle), outlineStyle
}

// defaultedOutlineStyle applies the CSS initial-value rule for isolated
// style longhands: an empty/none style becomes solid when width>0 or a
// color is set.
func defaultedOutlineStyle(sty *ResolvedStyle) string {
	style := sty.OutlineStyle
	if (style == "" || style == textTransformNone) && (sty.OutlineWidth > 0 || sty.OutlineColorSet) {
		return solidKeyword
	}

	return style
}

// defaultedOutlineWidth applies the CSS initial-value rule for isolated
// width longhands: a zero width with a visible style becomes medium.
func defaultedOutlineWidth(sty *ResolvedStyle, style string) float64 {
	if sty.OutlineWidth > 0 || !outlineNeedsMediumWidth(style) {
		return sty.OutlineWidth
	}

	if medium := borderWidth("medium", sty.FontSize); medium > 0 {
		return medium
	}

	return mediumOutlineWidthPt
}

// outlineNeedsMediumWidth reports visible outline styles that force a
// medium width when no explicit width is set.
func outlineNeedsMediumWidth(style string) bool {
	switch style {
	case solidKeyword, borderStyleDashed, borderStyleDotted:
		return true
	}

	return false
}

// outlinePaints reports a CSS outline that should stroke. Empty or "none"
// OutlineStyle means no outline. Outline never changes the layout box size.
func outlinePaints(sty *ResolvedStyle) bool {
	if sty == nil {
		return false
	}

	w, s := effectiveOutline(sty)
	if w <= 0 {
		return false
	}

	switch s {
	case solidKeyword, borderStyleDashed, borderStyleDotted:
		return true
	}

	return false
}

func outlineStrokeColor(sty *ResolvedStyle) (float64, float64, float64) {
	if sty == nil {
		return 0, 0, 0
	}

	if sty.OutlineColorSet {
		return sty.OutlineColor[0], sty.OutlineColor[1], sty.OutlineColor[2]
	}

	return sty.Color[0], sty.Color[1], sty.Color[2]
}

// outlineInflate is the distance from the border-box edge to the stroke
// centerline: offset (gap from the border edge) plus half the outline width
// so the inner edge of the stroke sits at outline-offset.
func outlineInflate(width, offset float64) float64 {
	return offset + width/2
}

// appendOutlineOps strokes an outline outside the border box by inflating the
// rect and reusing appendBorderLineOps (solid/dashed/dotted).
func appendOutlineOps(
	dst []Op, posX, posY, boxW, boxH, width, offset float64, style string, red, green, blue float64,
) []Op {
	if width <= 0 {
		return dst
	}

	switch style {
	case solidKeyword, borderStyleDashed, borderStyleDotted:
	default:
		return dst
	}

	inflate := outlineInflate(width, offset)
	outX := posX - inflate
	outY := posY - inflate
	outW := boxW + outlineBothSidesFactor*inflate
	outH := boxH + outlineBothSidesFactor*inflate

	if outW <= 0 || outH <= 0 {
		return dst
	}

	dst = appendBorderLineOps(dst, outX, outY, outW, 0, width, style, red, green, blue)
	dst = appendBorderLineOps(dst, outX+outW, outY, 0, outH, width, style, red, green, blue)
	dst = appendBorderLineOps(dst, outX, outY+outH, outW, 0, width, style, red, green, blue)
	dst = appendBorderLineOps(dst, outX, outY, 0, outH, width, style, red, green, blue)

	return dst
}

func hasRoundedCorners(sty *ResolvedStyle) bool {
	return sty.BorderRadiusTopLeft > 0 || sty.BorderRadiusTopRight > 0 ||
		sty.BorderRadiusBottomRight > 0 || sty.BorderRadiusBottomLeft > 0
}

func inflateRadius(rad, inflate float64) float64 {
	if rad > 0 {
		return rad + inflate
	}

	return 0
}

func (e *engine) roundedOutlineOp(
	sty *ResolvedStyle, posX, posY, width, height, outlineWidth, outlineOff, red, green, blue float64,
) (Op, bool) {
	inflate := outlineInflate(outlineWidth, outlineOff)
	outX := posX - inflate
	outY := posY - inflate
	outW := width + outlineBothSidesFactor*inflate
	outH := height + outlineBothSidesFactor*inflate

	if outW <= 0 || outH <= 0 {
		return Op{}, false //nolint:exhaustruct
	}

	radTL := inflateRadius(e.scalePt(sty.BorderRadiusTopLeft), inflate)
	radTR := inflateRadius(e.scalePt(sty.BorderRadiusTopRight), inflate)
	radBR := inflateRadius(e.scalePt(sty.BorderRadiusBottomRight), inflate)
	radBL := inflateRadius(e.scalePt(sty.BorderRadiusBottomLeft), inflate)

	return Op{ //nolint:exhaustruct // outline rounded stroke op
		Kind:              OpStrokeRect,
		X:                 outX,
		Y:                 outY,
		W:                 outW,
		H:                 outH,
		Width:             outlineWidth,
		R:                 red,
		G:                 green,
		B:                 blue,
		RadiusTopLeft:     radTL,
		RadiusTopRight:    radTR,
		RadiusBottomRight: radBR,
		RadiusBottomLeft:  radBL,
	}, true
}

func (e *engine) outlineOps(sty *ResolvedStyle, posX, posY, width, height float64) []Op {
	if e == nil || !outlinePaints(sty) {
		return nil
	}

	effW, effStyle := effectiveOutline(sty)
	outlineWidth := e.scalePt(effW)
	outlineOff := e.scalePt(sty.OutlineOffset)
	red, green, blue := outlineStrokeColor(sty)

	if effStyle == solidKeyword && hasRoundedCorners(sty) {
		if op, ok := e.roundedOutlineOp(sty, posX, posY, width, height, outlineWidth, outlineOff, red, green, blue); ok {
			return []Op{op}
		}
	}

	return appendOutlineOps(
		make([]Op, 0, outlineSideHint),
		posX, posY, width, height, outlineWidth, outlineOff, effStyle, red, green, blue,
	)
}
