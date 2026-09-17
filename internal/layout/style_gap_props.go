package layout

// applyGapProps owns gap, row-gap, column-gap, and the legacy grid-* aliases.
func applyGapProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case gapKeyword, "grid-gap":
		return applyGapShorthand(style, value, fsize, ctx.viewportW)
	case "row-gap", "grid-row-gap":
		return applyRowGap(style, value, fsize, ctx.viewportW)
	case "column-gap", "grid-column-gap":
		return applyColumnGap(style, value, fsize, ctx.viewportW)
	default:
		return false
	}
}

func applyGapShorthand(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	if value == contentNormal {
		style.Gap = 0
		style.RowGap = 0
		style.ColumnGap = 0
		style.ColumnGapNormal = true
	} else if v, ok := lengthBox(value, fsize, viewportW, cssDisplayNone); ok && v >= 0 {
		style.Gap = v
		style.RowGap = v
		style.ColumnGap = v
		style.ColumnGapNormal = false
	}

	return true
}

func applyRowGap(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	if v, ok := lengthBox(value, fsize, viewportW, cssDisplayNone); ok && v >= 0 {
		style.RowGap = v
		style.Gap = v
	}

	return true
}

func applyColumnGap(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	if value == contentNormal {
		style.ColumnGap = 0
		style.ColumnGapNormal = true
	} else if v, ok := lengthBox(value, fsize, viewportW, cssDisplayNone); ok && v >= 0 {
		style.ColumnGap = v
		style.ColumnGapNormal = false
	}

	return true
}
