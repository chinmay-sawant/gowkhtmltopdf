package layout

import (
	"strconv"
	"strings"
)

// Multicol Level 2 extras: column-height / column-wrap.
const (
	columnWrapAuto   = "auto"
	columnWrapWrap   = "wrap"
	columnWrapNowrap = "nowrap"
)

// applyMulticolGroup handles column-* props (count/width/columns, rule,
// fill/span, height, wrap).
func applyMulticolGroup(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	if applyColumnCountWidthProps(style, prop, value, fsize, ctx.viewportW) {
		return true
	}

	if applyColumnHeightWrapProps(style, prop, value, fsize, ctx.viewportW) {
		return true
	}

	if applyColumnRuleProps(style, prop, value, fsize) {
		return true
	}

	return applyColumnFillSpanProps(style, prop, value)
}

func applyColumnRuleProps(style *ResolvedStyle, prop, value string, fsize float64) bool {
	switch prop {
	case "column-rule":
		applyColumnRuleShorthand(style, value, fsize)
	case "column-rule-width":
		if width, parsed := parseOutlineWidth(value, fsize); parsed {
			style.ColumnRuleWidth = width
		}
	case "column-rule-style":
		if ruleStyle, parsed := parseOutlineStyle(value); parsed {
			style.ColumnRuleStyle = ruleStyle
		}
	case "column-rule-color":
		if color, parsed := parseUsedColor(value, style.Color); parsed {
			style.ColumnRuleColor = color
			style.ColumnRuleColorSet = true
		}
	default:
		return false
	}

	return true
}

func applyColumnRuleShorthand(style *ResolvedStyle, value string, fsize float64) {
	width, ruleStyle, color, ok := parseRuleShorthand(value, fsize, style.Color)
	if !ok {
		return
	}

	style.ColumnRuleWidth = width
	style.ColumnRuleStyle = ruleStyle
	style.ColumnRuleColor = color
	style.ColumnRuleColorSet = true
}

func applyColumnCountWidthProps(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case "column-count":
		return setColumnCountValue(style, value)
	case "column-width":
		return setColumnWidthValue(style, value, fsize, viewportW)
	case "columns":
		parseColumnsShorthand(style, value, fsize, viewportW)
	default:
		return false
	}

	return true
}

func setColumnCountValue(style *ResolvedStyle, value string) bool {
	if value == overflowAuto {
		style.ColumnCount = 0
	} else if n, err := strconv.Atoi(strings.TrimSpace(value)); err == nil && n >= 1 {
		style.ColumnCount = n
	}

	return true
}

func setColumnWidthValue(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	if value == overflowAuto {
		style.ColumnWidth = -1
	} else if v, ok := lengthBox(value, fsize, viewportW, overflowAuto); ok && v >= 0 {
		style.ColumnWidth = v
	}

	return true
}

func applyColumnHeightWrapProps(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case "column-height":
		return setColumnHeightValue(style, value, fsize, viewportW)
	case "column-wrap":
		return setColumnWrapValue(style, value)
	default:
		return false
	}
}

func setColumnHeightValue(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	val := strings.TrimSpace(value)
	if val == overflowAuto {
		style.ColumnHeight = -1

		return true
	}

	if v, ok := lengthBox(val, fsize, viewportW, overflowAuto); ok && v >= 0 {
		style.ColumnHeight = v
	}

	return true
}

func setColumnWrapValue(style *ResolvedStyle, value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case columnWrapAuto, columnWrapWrap, columnWrapNowrap:
		style.ColumnWrap = strings.ToLower(strings.TrimSpace(value))
	}

	return true
}

func applyColumnFillSpanProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "column-span":
		switch value {
		case cssDisplayNone, "all":
			style.ColumnSpan = value
		}
	case "column-fill":
		switch value {
		case balanceKeyword, overflowAuto:
			style.ColumnFill = value
		}
	default:
		return false
	}

	return true
}

// columnWrapCreatesRows reports whether overflow past column-height (or a
// definite container height) should open a new multicol row in the block
// direction. Multicol-2: auto behaves as wrap when column-height is set,
// otherwise nowrap. nowrap overflow columns (inline direction) are not
// implemented; nowrap still caps height but does not open further rows.
func columnWrapCreatesRows(style ResolvedStyle) bool {
	switch style.ColumnWrap {
	case columnWrapWrap:
		return true
	case columnWrapNowrap:
		return false
	default: // auto
		return style.ColumnHeight >= 0
	}
}
