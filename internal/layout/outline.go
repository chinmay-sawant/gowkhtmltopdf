package layout

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
	x := posX - inflate
	y := posY - inflate
	w := boxW + two*inflate
	h := boxH + two*inflate

	if w <= 0 || h <= 0 {
		return dst
	}

	dst = appendBorderLineOps(dst, x, y, w, 0, width, style, red, green, blue)
	dst = appendBorderLineOps(dst, x+w, y, 0, h, width, style, red, green, blue)
	dst = appendBorderLineOps(dst, x, y+h, w, 0, width, style, red, green, blue)
	dst = appendBorderLineOps(dst, x, y, 0, h, width, style, red, green, blue)

	return dst
}

func (e *engine) outlineOps(sty ResolvedStyle, posX, posY, width, height float64) []Op {
	if e == nil || !outlinePaints(sty) {
		return nil
	}

	ow := e.scalePt(sty.OutlineWidth)
	off := e.scalePt(sty.OutlineOffset)
	red, green, blue := outlineStrokeColor(sty)

	return appendOutlineOps(nil, posX, posY, width, height, ow, off, sty.OutlineStyle, red, green, blue)
}
