package layout

func applyLogicalMarginHorizontal(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	rtl := style.Direction == cssDirectionRTL

	switch prop {
	case propMarginInlineStart:
		if rtl {
			style.MarginRight, style.MarginRightAuto = marginLenAuto(value, fsize, viewportW)
		} else {
			style.MarginLeft, style.MarginLeftAuto = marginLenAuto(value, fsize, viewportW)
		}
	case propMarginInlineEnd:
		if rtl {
			style.MarginLeft, style.MarginLeftAuto = marginLenAuto(value, fsize, viewportW)
		} else {
			style.MarginRight, style.MarginRightAuto = marginLenAuto(value, fsize, viewportW)
		}
	case propMarginBlockStart:
		style.MarginTop, style.MarginTopAuto = marginLenAuto(value, fsize, viewportW)
	case propMarginBlockEnd:
		style.MarginBottom, style.MarginBottomAuto = marginLenAuto(value, fsize, viewportW)
	default:
		return false
	}

	return true
}

func applyLogicalPaddingHorizontal(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	rtl := style.Direction == cssDirectionRTL

	switch prop {
	case propPaddingInlineStart:
		if rtl {
			style.PaddingRight = marginLen(value, fsize, viewportW)
		} else {
			style.PaddingLeft = marginLen(value, fsize, viewportW)
		}
	case propPaddingInlineEnd:
		if rtl {
			style.PaddingLeft = marginLen(value, fsize, viewportW)
		} else {
			style.PaddingRight = marginLen(value, fsize, viewportW)
		}
	case propPaddingBlockStart:
		style.PaddingTop = marginLen(value, fsize, viewportW)
	case propPaddingBlockEnd:
		style.PaddingBottom = marginLen(value, fsize, viewportW)
	default:
		return false
	}

	return true
}
