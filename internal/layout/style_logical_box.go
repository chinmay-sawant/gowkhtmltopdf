package layout

func applyLogicalMargin(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	if applyLogicalMarginPair(style, prop, value, fsize, viewportW) {
		return true
	}

	switch style.WritingMode {
	case writingModeVerticalRL:
		return applyLogicalMarginVerticalRL(style, prop, value, fsize, viewportW)
	case writingModeVerticalLR:
		return applyLogicalMarginVerticalLR(style, prop, value, fsize, viewportW)
	default:
		return applyLogicalMarginHorizontal(style, prop, value, fsize, viewportW)
	}
}

func applyLogicalMarginVerticalRL(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case propMarginInlineStart:
		if style.Direction == cssDirectionRTL {
			style.MarginBottom, style.MarginBottomAuto = marginLenAuto(value, fsize, viewportW)
		} else {
			style.MarginTop, style.MarginTopAuto = marginLenAuto(value, fsize, viewportW)
		}
	case propMarginInlineEnd:
		if style.Direction == cssDirectionRTL {
			style.MarginTop, style.MarginTopAuto = marginLenAuto(value, fsize, viewportW)
		} else {
			style.MarginBottom, style.MarginBottomAuto = marginLenAuto(value, fsize, viewportW)
		}
	case propMarginBlockStart:
		style.MarginRight, style.MarginRightAuto = marginLenAuto(value, fsize, viewportW)
	case propMarginBlockEnd:
		style.MarginLeft, style.MarginLeftAuto = marginLenAuto(value, fsize, viewportW)
	default:
		return false
	}

	return true
}

func applyLogicalMarginVerticalLR(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case propMarginInlineStart:
		if style.Direction == cssDirectionRTL {
			style.MarginBottom, style.MarginBottomAuto = marginLenAuto(value, fsize, viewportW)
		} else {
			style.MarginTop, style.MarginTopAuto = marginLenAuto(value, fsize, viewportW)
		}
	case propMarginInlineEnd:
		if style.Direction == cssDirectionRTL {
			style.MarginTop, style.MarginTopAuto = marginLenAuto(value, fsize, viewportW)
		} else {
			style.MarginBottom, style.MarginBottomAuto = marginLenAuto(value, fsize, viewportW)
		}
	case propMarginBlockStart:
		style.MarginLeft, style.MarginLeftAuto = marginLenAuto(value, fsize, viewportW)
	case propMarginBlockEnd:
		style.MarginRight, style.MarginRightAuto = marginLenAuto(value, fsize, viewportW)
	default:
		return false
	}

	return true
}

//nolint:gocritic,nestif // writing mode and direction are independent cascade axes
func applyLogicalMarginPair(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case cssPropMarginInline:
		start, end, parsed := logicalPair(value)
		if parsed {
			if isVerticalWritingMode(style.WritingMode) {
				if style.Direction == cssDirectionRTL {
					style.MarginBottom, style.MarginBottomAuto = marginLenAuto(start, fsize, viewportW)
					style.MarginTop, style.MarginTopAuto = marginLenAuto(end, fsize, viewportW)
				} else {
					style.MarginTop, style.MarginTopAuto = marginLenAuto(start, fsize, viewportW)
					style.MarginBottom, style.MarginBottomAuto = marginLenAuto(end, fsize, viewportW)
				}
			} else if style.Direction == cssDirectionRTL {
				style.MarginRight, style.MarginRightAuto = marginLenAuto(start, fsize, viewportW)
				style.MarginLeft, style.MarginLeftAuto = marginLenAuto(end, fsize, viewportW)
			} else {
				style.MarginLeft, style.MarginLeftAuto = marginLenAuto(start, fsize, viewportW)
				style.MarginRight, style.MarginRightAuto = marginLenAuto(end, fsize, viewportW)
			}
		}
	case cssPropMarginBlock:
		applyLogicalMarginBlockPair(style, value, fsize, viewportW)
	default:
		return false
	}

	return true
}

func applyLogicalMarginBlockPair(style *ResolvedStyle, value string, fsize, viewportW float64) {
	start, end, parsed := logicalPair(value)
	if !parsed {
		return
	}

	switch style.WritingMode {
	case writingModeVerticalRL:
		style.MarginRight, style.MarginRightAuto = marginLenAuto(start, fsize, viewportW)
		style.MarginLeft, style.MarginLeftAuto = marginLenAuto(end, fsize, viewportW)
	case writingModeVerticalLR:
		style.MarginLeft, style.MarginLeftAuto = marginLenAuto(start, fsize, viewportW)
		style.MarginRight, style.MarginRightAuto = marginLenAuto(end, fsize, viewportW)
	default:
		style.MarginTop, style.MarginTopAuto = marginLenAuto(start, fsize, viewportW)
		style.MarginBottom, style.MarginBottomAuto = marginLenAuto(end, fsize, viewportW)
	}
}

func applyLogicalPadding(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	if applyLogicalPaddingPair(style, prop, value, fsize, viewportW) {
		return true
	}

	switch style.WritingMode {
	case writingModeVerticalRL:
		return applyLogicalPaddingVerticalRL(style, prop, value, fsize, viewportW)
	case writingModeVerticalLR:
		return applyLogicalPaddingVerticalLR(style, prop, value, fsize, viewportW)
	default:
		return applyLogicalPaddingHorizontal(style, prop, value, fsize, viewportW)
	}
}

func applyLogicalPaddingVerticalRL(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case propPaddingInlineStart:
		if style.Direction == cssDirectionRTL {
			style.PaddingBottom = marginLen(value, fsize, viewportW)
		} else {
			style.PaddingTop = marginLen(value, fsize, viewportW)
		}
	case propPaddingInlineEnd:
		if style.Direction == cssDirectionRTL {
			style.PaddingTop = marginLen(value, fsize, viewportW)
		} else {
			style.PaddingBottom = marginLen(value, fsize, viewportW)
		}
	case propPaddingBlockStart:
		style.PaddingRight = marginLen(value, fsize, viewportW)
	case propPaddingBlockEnd:
		style.PaddingLeft = marginLen(value, fsize, viewportW)
	default:
		return false
	}

	return true
}

func applyLogicalPaddingVerticalLR(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case propPaddingInlineStart:
		if style.Direction == cssDirectionRTL {
			style.PaddingBottom = marginLen(value, fsize, viewportW)
		} else {
			style.PaddingTop = marginLen(value, fsize, viewportW)
		}
	case propPaddingInlineEnd:
		if style.Direction == cssDirectionRTL {
			style.PaddingTop = marginLen(value, fsize, viewportW)
		} else {
			style.PaddingBottom = marginLen(value, fsize, viewportW)
		}
	case propPaddingBlockStart:
		style.PaddingLeft = marginLen(value, fsize, viewportW)
	case propPaddingBlockEnd:
		style.PaddingRight = marginLen(value, fsize, viewportW)
	default:
		return false
	}

	return true
}

//nolint:gocritic,nestif // writing mode and direction are independent cascade axes
func applyLogicalPaddingPair(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case cssPropPaddingInline:
		start, end, parsed := logicalPair(value)
		if parsed {
			if isVerticalWritingMode(style.WritingMode) {
				if style.Direction == cssDirectionRTL {
					style.PaddingBottom = marginLen(start, fsize, viewportW)
					style.PaddingTop = marginLen(end, fsize, viewportW)
				} else {
					style.PaddingTop = marginLen(start, fsize, viewportW)
					style.PaddingBottom = marginLen(end, fsize, viewportW)
				}
			} else if style.Direction == cssDirectionRTL {
				style.PaddingRight = marginLen(start, fsize, viewportW)
				style.PaddingLeft = marginLen(end, fsize, viewportW)
			} else {
				style.PaddingLeft = marginLen(start, fsize, viewportW)
				style.PaddingRight = marginLen(end, fsize, viewportW)
			}
		}
	case cssPropPaddingBlock:
		applyLogicalPaddingBlockPair(style, value, fsize, viewportW)
	default:
		return false
	}

	return true
}

func applyLogicalPaddingBlockPair(style *ResolvedStyle, value string, fsize, viewportW float64) {
	start, end, parsed := logicalPair(value)
	if !parsed {
		return
	}

	switch style.WritingMode {
	case writingModeVerticalRL:
		style.PaddingRight = marginLen(start, fsize, viewportW)
		style.PaddingLeft = marginLen(end, fsize, viewportW)
	case writingModeVerticalLR:
		style.PaddingLeft = marginLen(start, fsize, viewportW)
		style.PaddingRight = marginLen(end, fsize, viewportW)
	default:
		style.PaddingTop = marginLen(start, fsize, viewportW)
		style.PaddingBottom = marginLen(end, fsize, viewportW)
	}
}
