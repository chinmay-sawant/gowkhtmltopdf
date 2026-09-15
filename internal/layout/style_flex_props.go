package layout

// Flex and gap property parsing, extracted from style_properties.go to keep
// that file under the size gate. The group owns the flex container/item
// longhands plus the gap family and the place-*/flow shorthands; the align
// setters are also used by style_values.go's alignment shorthand parsers.
import (
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// applyFlexGroup handles flex layout props and the gap family.
func applyFlexGroup(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	switch prop {
	case gapKeyword, "row-gap", "column-gap":
		return applyGapProps(style, prop, value, fsize, ctx)
	case "flex-direction", "flex-wrap", "justify-content", "align-items",
		"align-content", "align-self", "justify-items", "justify-self",
		"flex-flow", "place-content", "place-items", "place-self":
		return applyFlexAlignmentProps(style, prop, value)
	case flexKeyword, "flex-grow", "flex-shrink", "flex-basis", "order":
		return applyFlexBasisProps(style, prop, value, fsize, ctx)
	default:
		return false
	}
}

// applyGapProps owns the gap family, dispatching each property to its parser.
func applyGapProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case gapKeyword:
		return applyGapShorthand(style, value, fsize, ctx.viewportW)
	case "row-gap":
		return applyRowGap(style, value, fsize, ctx.viewportW)
	case "column-gap":
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

// applyFlexAlignmentProps owns the flex/grid alignment keywords.
func applyFlexAlignmentProps(style *ResolvedStyle, prop, value string) bool {
	if applyPlaceAndFlowShorthands(style, prop, value) {
		return true
	}

	return applyFlexKeywordProps(style, prop, value)
}

func applyFlexKeywordProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "flex-direction":
		setFlexDirectionValue(style, value)
	case "flex-wrap":
		setFlexWrapValue(style, value)
	case "justify-content":
		setJustifyContentValue(style, value)
	case "align-items":
		setAlignItemsValue(style, value)
	case "align-content":
		setAlignContentValue(style, value)
	case "align-self":
		setAlignSelfValue(style, value)
	case "justify-items":
		setJustifyItemsValue(style, value)
	case "justify-self":
		setJustifySelfValue(style, value)
	default:
		return false
	}

	return true
}

func applyPlaceAndFlowShorthands(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "flex-flow":
		parseFlexFlow(style, value)
	case "place-content":
		parsePlaceContent(style, value)
	case "place-items":
		parsePlaceItems(style, value)
	case "place-self":
		parsePlaceSelf(style, value)
	default:
		return false
	}

	return true
}

func setFlexDirectionValue(style *ResolvedStyle, value string) {
	switch value {
	case fxRow, fxCol, fxRowRev, fxColRev:
		style.FlexDirection = value
	}
}

func setFlexWrapValue(style *ResolvedStyle, value string) {
	if value == cssWhiteSpaceNowrap || value == fxWrap || value == fxWrapRev {
		style.FlexWrap = value
	}
}

func setJustifyContentValue(style *ResolvedStyle, value string) {
	switch value {
	case flexStartKeyword, fxFlexEnd, fxCenter, fxBetween, fxAround, fxEvenly, fxStart, fxEnd:
		style.JustifyContent = value
	}
}

func setAlignItemsValue(style *ResolvedStyle, value string) {
	switch value {
	case fxStretch, flexStartKeyword, fxFlexEnd, fxCenter, fxStart, fxEnd, "baseline":
		style.AlignItems = value
	}
}

func setAlignContentValue(style *ResolvedStyle, value string) {
	switch value {
	case flexStartKeyword, fxFlexEnd, fxCenter, fxBetween, fxAround,
		fxEvenly, fxStretch, fxStart, fxEnd:
		style.AlignContent = value
	}
}

func setAlignSelfValue(style *ResolvedStyle, value string) {
	switch value {
	case overflowAuto, fxStretch, flexStartKeyword, fxFlexEnd, fxCenter, fxStart, fxEnd:
		style.AlignSelf = value
	}
}

func setJustifyItemsValue(style *ResolvedStyle, value string) {
	switch value {
	case fxStretch, fxStart, fxEnd, fxCenter, flexStartKeyword, fxFlexEnd:
		style.JustifyItems = value
	}
}

func setJustifySelfValue(style *ResolvedStyle, value string) {
	switch value {
	case overflowAuto, fxStretch, fxStart, fxEnd, fxCenter, flexStartKeyword, fxFlexEnd:
		style.JustifySelf = value
	}
}

// applyFlexBasisProps owns the flex shorthand, grow/shrink/basis and order.
func applyFlexBasisProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case flexKeyword:
		parseFlexShorthand(style, value, fsize, ctx.viewportW)
	case "flex-grow":
		setFlexGrowValue(style, value)
	case "flex-shrink":
		setFlexShrinkValue(style, value)
	case "flex-basis":
		setFlexBasisValue(style, value, fsize, ctx.viewportW)
	case "order":
		setFlexOrderValue(style, value)
	default:
		return false
	}

	return true
}

func setFlexGrowValue(style *ResolvedStyle, value string) {
	if v, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && v >= 0 {
		style.FlexGrow = v
	}
}

func setFlexShrinkValue(style *ResolvedStyle, value string) {
	if v, err := strconv.ParseFloat(strings.TrimSpace(value), 64); err == nil && v >= 0 {
		style.FlexShrink = v
	}
}

func setFlexBasisValue(style *ResolvedStyle, value string, fsize, viewportW float64) {
	if value == overflowAuto {
		style.FlexBasis = -1
		style.FlexBasisPercent = -1
	} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
		style.FlexBasisPercent = v
		style.FlexBasis = -1
	} else if v, ok := lengthBox(value, fsize, viewportW, overflowAuto); ok {
		style.FlexBasis = v
		style.FlexBasisPercent = -1
	}
}

func setFlexOrderValue(style *ResolvedStyle, value string) {
	if v, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		style.FlexOrder = v
	}
}
