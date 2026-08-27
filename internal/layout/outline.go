package layout

const outlineSideHint = 4 // four sides of a rectangular outline

// outlinePaints reports a CSS outline that should stroke. Empty or "none"
// OutlineStyle means no outline. Outline never changes the layout box size.
func outlinePaints(sty ResolvedStyle) bool {
	if sty.OutlineWidth <= 0 {
		return false
	}

	switch sty.OutlineStyle {
	case solidKeyword, borderStyleDashed, borderStyleDotted:
		return true
	}

	return false
}

func outlineStrokeColor(sty ResolvedStyle) (float64, float64, float64) {
	if sty.OutlineColorSet {
		return sty.OutlineColor[0], sty.OutlineColor[1], sty.OutlineColor[2]
	}

	return sty.Color[0], sty.Color[1], sty.Color[2]
}

// outlineInflate is the distance from the border-box edge to the stroke
// centerline: offset (gap from the border edge) plus half the outline width
// so the inner edge of the stroke sits at outline-offset.
func outlineInflate(width, offset float64) float64 {
	return offset + width/halfDivisor
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
	outW := boxW + two*inflate
	outH := boxH + two*inflate

	if outW <= 0 || outH <= 0 {
		return dst
	}

	dst = appendBorderLineOps(dst, outX, outY, outW, 0, width, style, red, green, blue)
	dst = appendBorderLineOps(dst, outX+outW, outY, 0, outH, width, style, red, green, blue)
	dst = appendBorderLineOps(dst, outX, outY+outH, outW, 0, width, style, red, green, blue)
	dst = appendBorderLineOps(dst, outX, outY, 0, outH, width, style, red, green, blue)

	return dst
}

func (e *engine) outlineOps(sty ResolvedStyle, posX, posY, width, height float64) []Op {
	if e == nil || !outlinePaints(sty) {
		return nil
	}

	outlineWidth := e.scalePt(sty.OutlineWidth)
	outlineOff := e.scalePt(sty.OutlineOffset)
	red, green, blue := outlineStrokeColor(sty)

	return appendOutlineOps(
		make([]Op, 0, outlineSideHint),
		posX, posY, width, height, outlineWidth, outlineOff, sty.OutlineStyle, red, green, blue,
	)
}
