package layout

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

const (
	borderRadiusValueCount  = 4
	borderRadiusPairCount   = 2
	borderRadiusTripleCount = 3
)

func applyDisplayGroup(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	if applyDisplayFlowProps(style, prop, value) {
		return true
	}

	return applyDisplayEffectProps(style, prop, value)
}

// applyDisplayFlowProps owns the display/position/float/clear/box-sizing/
// writing-mode/overflow keyword properties.
func applyDisplayFlowProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "display":
		setDisplayKeyword(style, value)
	case "position":
		setPositionKeyword(style, value)
	case "float":
		setFloatKeyword(style, value)
	case clearKeyword:
		setClearKeyword(style, value)
	case "box-sizing":
		setBoxSizingKeyword(style, value)
	case "writing-mode":
		setWritingModeKeyword(style, value)
	case "overflow", "overflow-x", "overflow-y":
		setOverflowKeyword(style, prop, value)
	default:
		return false
	}

	return true
}

func setDisplayKeyword(style *ResolvedStyle, value string) {
	switch value {
	case displayBlock, "inline", cssDisplayNone, displayListItem, displayTable, displayTableRow, displayTableCell,
		displayRowGroup, displayHeaderGroup, displayFooterGroup,
		cssDisplayInlineBlock, displayTableCaption, "table-column", "table-column-group",
		displayFlex, displayInlineFlex, displayGrid, displayInlineGrid, displaySubgrid, displayFlowRoot:
		style.Display = value
	}
}

func setPositionKeyword(style *ResolvedStyle, value string) {
	switch value {
	case "static", "relative", "absolute", "fixed", "sticky":
		style.Position = value
	}
}

func setFloatKeyword(style *ResolvedStyle, value string) {
	switch value {
	case floatLeft, floatRight, cssDisplayNone:
		style.Float = value
	}
}

func setClearKeyword(style *ResolvedStyle, value string) {
	switch value {
	case floatLeft, floatRight, clearBoth, cssDisplayNone:
		style.Clear = value
	}
}

func setBoxSizingKeyword(style *ResolvedStyle, value string) {
	switch value {
	case "content-box", borderBox:
		style.BoxSizing = value
	}
}

// setOverflowKeyword applies overflow on either axis; a non-visible overflow
// on an axis creates a sticky scrollport (CSS Position 3).
func setOverflowKeyword(style *ResolvedStyle, prop, value string) {
	overflow, ok := parseOverflowKeyword(value)
	if !ok {
		return
	}

	switch prop {
	case "overflow":
		style.Overflow = overflow
		style.OverflowX = overflow
		style.OverflowY = overflow
	case "overflow-x":
		style.OverflowX = overflow
		if overflow != visibleKeyword {
			style.Overflow = overflow
		}
	case "overflow-y":
		style.OverflowY = overflow
		if overflow != visibleKeyword {
			style.Overflow = overflow
		}
	}
}

func setWritingModeKeyword(style *ResolvedStyle, value string) {
	switch value {
	case writingModeHorizontalTB, writingModeVerticalRL, writingModeVerticalLR:
		style.WritingMode = value
	}
}

// applyDisplayEffectProps owns z-index, opacity, filter:opacity() and visibility.
func applyDisplayEffectProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "z-index":
		setZIndexValue(style, value)
	case "opacity":
		setOpacityValue(style, value)
	case "filter":
		setFilterValue(style, value)
	case "visibility":
		setVisibilityValue(style, value)
	default:
		return false
	}

	return true
}

func setZIndexValue(style *ResolvedStyle, value string) {
	if value == overflowAuto {
		style.ZIndexSet = false
		style.ZIndex = 0
	} else if v, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		style.ZIndex = v
		style.ZIndexSet = true
	}
}

func setOpacityValue(style *ResolvedStyle, value string) {
	if v, ok := parseOpacityValue(value); ok {
		style.Opacity = v
	}
}

func setFilterValue(style *ResolvedStyle, value string) {
	// opacity() via ExtGState; blur/drop-shadow permanent non-goals.
	if v, ok := parseFilterOpacity(value); ok {
		style.Opacity *= v
	}
}

func setVisibilityValue(style *ResolvedStyle, value string) {
	switch value {
	case visibleKeyword, overflowHidden, borderCollapseValue:
		style.Visibility = value
	}
}

// applyPositionGroup handles the top/right/bottom/left offsets.
func applyPositionGroup(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	if applyLogicalInset(style, prop, value, fsize, ctx) {
		return true
	}

	switch prop {
	case cssVerticalAlignTop:
		style.Top, style.TopAuto = marginLenAuto(value, fsize, ctx.viewportH)
	case floatRight:
		style.Right, style.RightAuto = marginLenAuto(value, fsize, ctx.viewportW)
	case cssVerticalAlignBottom:
		style.Bottom, style.BottomAuto = marginLenAuto(value, fsize, ctx.viewportH)
	case floatLeft:
		style.Left, style.LeftAuto = marginLenAuto(value, fsize, ctx.viewportW)
	default:
		return false
	}

	return true
}

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
	case fxStretch, flexStartKeyword, fxFlexEnd, fxCenter, fxStart, fxEnd:
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

// applyMulticolGroup handles column-* props.
func applyMulticolGroup(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	if applyColumnCountWidthProps(style, prop, value, fsize, ctx.viewportW) {
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

func applyColumnFillSpanProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "column-span":
		switch value {
		case cssDisplayNone, "all":
			style.ColumnSpan = value
		}
	case "column-fill":
		switch value {
		case "balance", overflowAuto:
			style.ColumnFill = value
		}
	default:
		return false
	}

	return true
}

// applyGridGroup handles grid template/placement props.
func applyGridGroup(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	if applyGridTemplateProps(style, prop, value) {
		return true
	}

	return applyGridPlacementProps(style, prop, value)
}

func applyGridTemplateProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "grid-template-columns":
		style.GridTemplateColumns = value
	case "grid-template-rows":
		style.GridTemplateRows = value
	case "grid-template-areas":
		style.GridTemplateAreas = value
	case "grid-area":
		parseGridArea(style, value)
	case "grid-auto-flow":
		style.GridAutoFlow = parseGridAutoFlowValue(value)
	case "grid-template":
		parseGridTemplateShorthand(style, value)
	case "grid":
		parseGridShorthand(style, value)
	default:
		return false
	}

	return true
}

func applyGridPlacementProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "grid-column", "grid-column-end":
		parseGridColumn(style, value)
	case "grid-column-start":
		setGridStartIndex(style, "grid-column-start", value)
	case "grid-row", "grid-row-end":
		parseGridRow(style, value)
	case "grid-row-start":
		setGridStartIndex(style, "grid-row-start", value)
	default:
		return false
	}

	return true
}

// setGridStartIndex parses a positive grid line index for one axis.
func setGridStartIndex(style *ResolvedStyle, prop, value string) {
	line, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || line <= 0 {
		return
	}

	if prop == "grid-row-start" {
		style.GridRowStart = line
	} else {
		style.GridColumnStart = line
	}
}

// applyBoxGroup handles the sizing and box props (width/height/margins/padding).
func applyBoxGroup(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	if applyBoxSizingProps(style, prop, value, fsize, ctx) {
		return true
	}

	if applyLogicalBoxProps(style, prop, value, fsize, ctx) {
		return true
	}

	return applyBoxSpacingProps(style, prop, value, fsize, ctx.viewportW)
}

func applyBoxSizingProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case "width", "height":
		return applyBoxMainSizeProps(style, prop, value, fsize, ctx)
	case "min-width", "min-height":
		return applyBoxMinExtentProps(style, prop, value, fsize, ctx)
	case "max-width", "max-height":
		return applyBoxMaxExtentProps(style, prop, value, fsize, ctx)
	default:
		return false
	}
}

func applyBoxSpacingProps(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case "margin":
		return applyMarginShorthandProps(style, value, fsize, viewportW)
	case "margin-top", "margin-bottom":
		return applyMarginVerticalProps(style, prop, value, fsize, viewportW)
	case "margin-right", "margin-left":
		return applyMarginHorizontalProps(style, prop, value, fsize, viewportW)
	case paddingProperty:
		return applyPaddingShorthandProps(style, value, fsize, viewportW)
	case "padding-top", "padding-right", "padding-bottom", "padding-left":
		return applyPaddingSideProps(style, prop, value, fsize, viewportW)
	default:
		return false
	}
}

func applyBoxMainSizeProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case "width":
		return setWidthValue(style, value, fsize, ctx)
	case "height":
		return setHeightValue(style, value, fsize, ctx)
	default:
		return false
	}
}

func setWidthValue(style *ResolvedStyle, value string, fsize float64, ctx *styleContext) bool {
	if value == overflowAuto {
		style.Width = -1
		style.WidthPercent = -1
	} else if v, ok := vminVmaxPt(value, ctx.viewportW, ctx.viewportH); ok {
		style.Width = v
		style.WidthPercent = -1
	} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
		// Resolve % against the layout containing block (availW), not
		// the viewport — nested width:100% must fill the parent cell.
		style.WidthPercent = v
		style.Width = -1
	} else if v, ok := lengthBox(value, fsize, ctx.viewportW, overflowAuto); ok {
		style.Width = v
		style.WidthPercent = -1
	}

	return true
}

func setHeightValue(style *ResolvedStyle, value string, fsize float64, ctx *styleContext) bool {
	if value == overflowAuto {
		style.Height = -1
		style.HeightPercent = -1
	} else if v, ok := vminVmaxPt(value, ctx.viewportW, ctx.viewportH); ok {
		style.Height = v
		style.HeightPercent = -1
	} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
		// Defer % height to layout; indefinite containing block → auto
		// (cyclic percentage honesty for flex/grid children).
		style.HeightPercent = v
		style.Height = -1
	} else if v, ok := lengthBox(value, fsize, ctx.viewportH, overflowAuto); ok {
		style.Height = v
		style.HeightPercent = -1
	}

	return true
}

func applyBoxMinExtentProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case "min-width":
		return setMinWidthValue(style, value, fsize, ctx.viewportW)
	case "min-height":
		return setMinHeightValue(style, value, fsize, ctx.viewportH)
	default:
		return false
	}
}

func setMinWidthValue(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	style.MinWidthSet = true

	if value == overflowAuto || value == cssDisplayNone {
		style.MinWidth = 0
		style.MinWidthPercent = -1
	} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
		style.MinWidthPercent = v
		style.MinWidth = 0
	} else if v, ok := lengthBox(value, fsize, viewportW, cssDisplayNone); ok {
		style.MinWidth = v
		style.MinWidthPercent = -1
	}

	return true
}

func setMinHeightValue(style *ResolvedStyle, value string, fsize, viewportH float64) bool {
	if value == overflowAuto || value == cssDisplayNone {
		style.MinHeight = 0
		style.MinHeightPercent = -1
	} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
		style.MinHeightPercent = v
		style.MinHeight = 0
	} else if v, ok := lengthBox(value, fsize, viewportH, cssDisplayNone); ok {
		style.MinHeight = v
		style.MinHeightPercent = -1
	}

	return true
}

func applyBoxMaxExtentProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case "max-width":
		return setMaxWidthValue(style, value, fsize, ctx.viewportW)
	case "max-height":
		return setMaxHeightValue(style, value, fsize, ctx.viewportH)
	default:
		return false
	}
}

func setMaxWidthValue(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	if value == cssDisplayNone {
		style.MaxWidth = -1
		style.MaxWidthPercent = -1
	} else if v, unit, ok := css.ParseLength(value); ok && unit == "%" {
		style.MaxWidthPercent = v
		style.MaxWidth = -1
	} else if v, ok := lengthBox(value, fsize, viewportW, cssDisplayNone); ok {
		style.MaxWidth = v
		style.MaxWidthPercent = -1
	}

	return true
}

func setMaxHeightValue(style *ResolvedStyle, value string, fsize, viewportH float64) bool {
	if v, ok := lengthBox(value, fsize, viewportH, cssDisplayNone); ok {
		style.MaxHeight = v
	}

	return true
}

func applyMarginShorthandProps(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	setFourMargin(style, value, fsize, viewportW)

	return true
}

func applyMarginVerticalProps(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case "margin-top":
		style.MarginTop, style.MarginTopAuto = marginLenAuto(value, fsize, viewportW)
	case "margin-bottom":
		style.MarginBottom, style.MarginBottomAuto = marginLenAuto(value, fsize, viewportW)
	default:
		return false
	}

	return true
}

func applyMarginHorizontalProps(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case "margin-right":
		style.MarginRight, style.MarginRightAuto = marginLenAuto(value, fsize, viewportW)
	case "margin-left":
		style.MarginLeft, style.MarginLeftAuto = marginLenAuto(value, fsize, viewportW)
	default:
		return false
	}

	return true
}

func applyPaddingShorthandProps(style *ResolvedStyle, value string, fsize, viewportW float64) bool {
	setFour(style, value,
		&style.PaddingTop, &style.PaddingRight, &style.PaddingBottom, &style.PaddingLeft,
		fsize, viewportW)

	return true
}

func applyPaddingSideProps(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case "padding-top":
		style.PaddingTop = marginLen(value, fsize, viewportW)
	case "padding-right":
		style.PaddingRight = marginLen(value, fsize, viewportW)
	case "padding-bottom":
		style.PaddingBottom = marginLen(value, fsize, viewportW)
	case "padding-left":
		style.PaddingLeft = marginLen(value, fsize, viewportW)
	default:
		return false
	}

	return true
}

func mapsLogicalToPhysical(style *ResolvedStyle) bool {
	return style.WritingMode == "" || style.WritingMode == writingModeHorizontalTB
}

func applyLogicalBoxProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case cssPropMarginInline, "margin-inline-start", "margin-inline-end",
		cssPropMarginBlock, "margin-block-start", "margin-block-end":
		return applyLogicalMargin(style, prop, value, fsize, ctx.viewportW)
	case cssPropPaddingInline, "padding-inline-start", "padding-inline-end",
		cssPropPaddingBlock, "padding-block-start", "padding-block-end":
		return applyLogicalPadding(style, prop, value, fsize, ctx.viewportW)
	case containerInlineSize, "block-size",
		"min-inline-size", "max-inline-size",
		"min-block-size", "max-block-size":
		return applyLogicalSize(style, prop, value, fsize, ctx)
	default:
		return false
	}
}

func applyLogicalMargin(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	if !mapsLogicalToPhysical(style) {
		return true
	}

	if applyLogicalMarginPair(style, prop, value, fsize, viewportW) {
		return true
	}

	switch prop {
	case "margin-inline-start":
		style.MarginLeft, style.MarginLeftAuto = marginLenAuto(value, fsize, viewportW)
	case "margin-inline-end":
		style.MarginRight, style.MarginRightAuto = marginLenAuto(value, fsize, viewportW)
	case "margin-block-start":
		style.MarginTop, style.MarginTopAuto = marginLenAuto(value, fsize, viewportW)
	case "margin-block-end":
		style.MarginBottom, style.MarginBottomAuto = marginLenAuto(value, fsize, viewportW)
	default:
		return false
	}

	return true
}

func applyLogicalMarginPair(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case cssPropMarginInline:
		start, end, parsed := logicalPair(value)
		if parsed {
			style.MarginLeft, style.MarginLeftAuto = marginLenAuto(start, fsize, viewportW)
			style.MarginRight, style.MarginRightAuto = marginLenAuto(end, fsize, viewportW)
		}
	case cssPropMarginBlock:
		start, end, parsed := logicalPair(value)
		if parsed {
			style.MarginTop, style.MarginTopAuto = marginLenAuto(start, fsize, viewportW)
			style.MarginBottom, style.MarginBottomAuto = marginLenAuto(end, fsize, viewportW)
		}
	default:
		return false
	}

	return true
}

func applyLogicalPadding(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	if !mapsLogicalToPhysical(style) {
		return true
	}

	if applyLogicalPaddingPair(style, prop, value, fsize, viewportW) {
		return true
	}

	switch prop {
	case "padding-inline-start":
		style.PaddingLeft = marginLen(value, fsize, viewportW)
	case "padding-inline-end":
		style.PaddingRight = marginLen(value, fsize, viewportW)
	case "padding-block-start":
		style.PaddingTop = marginLen(value, fsize, viewportW)
	case "padding-block-end":
		style.PaddingBottom = marginLen(value, fsize, viewportW)
	default:
		return false
	}

	return true
}

func applyLogicalPaddingPair(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case cssPropPaddingInline:
		start, end, parsed := logicalPair(value)
		if parsed {
			style.PaddingLeft = marginLen(start, fsize, viewportW)
			style.PaddingRight = marginLen(end, fsize, viewportW)
		}
	case cssPropPaddingBlock:
		start, end, parsed := logicalPair(value)
		if parsed {
			style.PaddingTop = marginLen(start, fsize, viewportW)
			style.PaddingBottom = marginLen(end, fsize, viewportW)
		}
	default:
		return false
	}

	return true
}

func applyLogicalSize(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	if !mapsLogicalToPhysical(style) {
		return true
	}

	switch prop {
	case containerInlineSize:
		return setWidthValue(style, value, fsize, ctx)
	case "block-size":
		return setHeightValue(style, value, fsize, ctx)
	case "min-inline-size":
		return setMinWidthValue(style, value, fsize, ctx.viewportW)
	case "max-inline-size":
		return setMaxWidthValue(style, value, fsize, ctx.viewportW)
	case "min-block-size":
		return setMinHeightValue(style, value, fsize, ctx.viewportH)
	case "max-block-size":
		return setMaxHeightValue(style, value, fsize, ctx.viewportH)
	default:
		return false
	}
}

func applyLogicalInset(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case insetKeyword, cssPropInsetBlock, cssPropInsetInline,
		"inset-block-start", "inset-block-end",
		"inset-inline-start", "inset-inline-end":
		if mapsLogicalToPhysical(style) {
			assignLogicalInset(style, prop, value, fsize, ctx.viewportW, ctx.viewportH)
		}

		return true
	default:
		return false
	}
}

func assignLogicalInset(style *ResolvedStyle, prop, value string, fsize, viewportW, viewportH float64) {
	switch prop {
	case insetKeyword:
		applyInsetShorthand(style, value, fsize, viewportW, viewportH)
	case cssPropInsetBlock:
		start, end, parsed := logicalPair(value)
		if parsed {
			style.Top, style.TopAuto = marginLenAuto(start, fsize, viewportH)
			style.Bottom, style.BottomAuto = marginLenAuto(end, fsize, viewportH)
		}
	case cssPropInsetInline:
		start, end, parsed := logicalPair(value)
		if parsed {
			style.Left, style.LeftAuto = marginLenAuto(start, fsize, viewportW)
			style.Right, style.RightAuto = marginLenAuto(end, fsize, viewportW)
		}
	case "inset-block-start":
		style.Top, style.TopAuto = marginLenAuto(value, fsize, viewportH)
	case "inset-block-end":
		style.Bottom, style.BottomAuto = marginLenAuto(value, fsize, viewportH)
	case "inset-inline-start":
		style.Left, style.LeftAuto = marginLenAuto(value, fsize, viewportW)
	case "inset-inline-end":
		style.Right, style.RightAuto = marginLenAuto(value, fsize, viewportW)
	}
}

func applyInsetShorthand(style *ResolvedStyle, value string, fsize, viewportW, viewportH float64) {
	var val [4]string

	count := splitSpaceTokens(value, val[:])
	if count == 0 {
		return
	}

	top, right, bottom, left := val[0], val[0], val[0], val[0]

	switch {
	case count > three:
		top, right, bottom, left = val[0], val[1], val[2], val[3]
	case count == three:
		top, right, bottom, left = val[0], val[1], val[2], val[1]
	case count == two:
		top, right, bottom, left = val[0], val[1], val[0], val[1]
	}

	style.Top, style.TopAuto = marginLenAuto(top, fsize, viewportH)
	style.Right, style.RightAuto = marginLenAuto(right, fsize, viewportW)
	style.Bottom, style.BottomAuto = marginLenAuto(bottom, fsize, viewportH)
	style.Left, style.LeftAuto = marginLenAuto(left, fsize, viewportW)
}

func logicalPair(value string) (string, string, bool) {
	var tok [2]string

	n := splitSpaceTokens(value, tok[:])
	if n == 0 {
		return "", "", false
	}

	if n == 1 {
		return tok[0], tok[0], true
	}

	return tok[0], tok[1], true
}

// applyBorderGroup handles the border shorthand and per-side props.
func applyBorderGroup(
	style *ResolvedStyle, prop, value string, fsize float64, _ *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	if applyOutlineProps(style, prop, value, fsize) {
		return true
	}

	if applyRadiusLonghand(style, prop, value, fsize) {
		return true
	}

	switch prop {
	case "border":
		return applyBorderAllSides(style, value, fsize)
	case "border-top", "border-right", "border-bottom", "border-left":
		return applyBorderOneSide(style, prop, value, fsize)
	case borderWidthKeyword, "border-top-width", "border-right-width", "border-bottom-width", "border-left-width":
		return applyBorderWidthProps(style, prop, value, fsize)
	case borderStyleKeyword, borderColorKeyword,
		"border-top-color", "border-right-color", "border-bottom-color", "border-left-color":
		return applyBorderStyleColorProps(style, prop, value)
	case "border-radius":
		return setBorderRadius(style, value, fsize)
	default:
		return false
	}
}

func applyBorderAllSides(style *ResolvedStyle, value string, fsize float64) bool {
	if strings.EqualFold(strings.TrimSpace(value), cssDisplayNone) || strings.TrimSpace(value) == "0" {
		zero := border{Width: 0, PaintWidth: 0, Style: "", Color: [3]float64{0, 0, 0}}
		style.BorderTop, style.BorderRight, style.BorderBottom, style.BorderLeft = zero, zero, zero, zero

		return true
	}

	if b, ok := parseBorder(value, fsize, style.Color); ok {
		style.BorderTop, style.BorderRight, style.BorderBottom, style.BorderLeft = b, b, b, b
	}

	return true
}

func applyBorderOneSide(style *ResolvedStyle, prop, value string, fsize float64) bool {
	switch prop {
	case "border-top":
		setBorderSide(style, &style.BorderTop, value, fsize)
	case "border-right":
		setBorderSide(style, &style.BorderRight, value, fsize)
	case "border-bottom":
		setBorderSide(style, &style.BorderBottom, value, fsize)
	case "border-left":
		setBorderSide(style, &style.BorderLeft, value, fsize)
	default:
		return false
	}

	return true
}

func setBorderSide(style *ResolvedStyle, side *border, value string, fsize float64) {
	if strings.EqualFold(strings.TrimSpace(value), cssDisplayNone) || strings.TrimSpace(value) == "0" {
		*side = border{Width: 0, PaintWidth: 0, Style: "", Color: [3]float64{0, 0, 0}}

		return
	}

	if b, ok := parseBorder(value, fsize, style.Color); ok {
		*side = b
	}
}

func applyBorderWidthProps(style *ResolvedStyle, prop, value string, fsize float64) bool {
	switch prop {
	case borderWidthKeyword:
		w := borderWidth(value, fsize)
		pw := borderPaintWidth(value, fsize)
		style.BorderTop.Width, style.BorderRight.Width, style.BorderBottom.Width, style.BorderLeft.Width = w, w, w, w
		style.BorderTop.PaintWidth = pw
		style.BorderRight.PaintWidth = pw
		style.BorderBottom.PaintWidth = pw
		style.BorderLeft.PaintWidth = pw
	case "border-top-width":
		style.BorderTop.Width = borderWidth(value, fsize)
		style.BorderTop.PaintWidth = borderPaintWidth(value, fsize)
	case "border-right-width":
		style.BorderRight.Width = borderWidth(value, fsize)
		style.BorderRight.PaintWidth = borderPaintWidth(value, fsize)
	case "border-bottom-width":
		style.BorderBottom.Width = borderWidth(value, fsize)
		style.BorderBottom.PaintWidth = borderPaintWidth(value, fsize)
	case "border-left-width":
		style.BorderLeft.Width = borderWidth(value, fsize)
		style.BorderLeft.PaintWidth = borderPaintWidth(value, fsize)
	default:
		return false
	}

	return true
}

//nolint:cyclop // border shorthand/property dispatch
func applyBorderStyleColorProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case borderStyleKeyword:
		s := value
		if s != solidKeyword && s != borderStyleDashed && s != borderStyleDotted {
			s = cssDisplayNone
		}

		style.BorderTop.Style, style.BorderRight.Style, style.BorderBottom.Style, style.BorderLeft.Style = s, s, s, s
	case borderColorKeyword:
		if c, ok := parseUsedColor(value, style.Color); ok {
			style.BorderTop.Color, style.BorderRight.Color, style.BorderBottom.Color, style.BorderLeft.Color = c, c, c, c
		}
	case "border-top-color":
		setBorderColor(&style.BorderTop, value, style.Color)
	case "border-right-color":
		setBorderColor(&style.BorderRight, value, style.Color)
	case "border-bottom-color":
		setBorderColor(&style.BorderBottom, value, style.Color)
	case "border-left-color":
		setBorderColor(&style.BorderLeft, value, style.Color)
	default:
		return false
	}

	return true
}

func setBorderColor(side *border, value string, current [3]float64) {
	if c, ok := parseUsedColor(value, current); ok {
		side.Color = c
	}
}

// applyColorGroup handles foreground and background colors.
func applyColorGroup(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext, parent *ResolvedStyle, hasParent bool,
) bool {
	if applyColorForegroundProps(style, prop, value, parent, hasParent) {
		return true
	}

	return applyColorBackgroundProps(style, prop, value)
}

//nolint:cyclop // foreground props are a flat mapping over color candidates
func applyColorForegroundProps(style *ResolvedStyle, prop, value string, parent *ResolvedStyle, hasParent bool) bool {
	switch prop {
	case "color":
		if value == inheritKeyword || isCurrentColor(value) {
			if hasParent && parent != nil {
				style.Color = parent.Color
			}
		} else if c, ok := parseUsedColor(value, style.Color); ok {
			style.Color = c
		}
	case "accent-color":
		if value == inheritKeyword {
			if hasParent && parent != nil && parent.AccentColorSet {
				style.AccentColor = parent.AccentColor
				style.AccentColorSet = true
			}

			return true
		}

		if r, g, b, _, ok := css.ParseColor(value); ok {
			style.AccentColor = [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}
			style.AccentColorSet = true
		}
	default:
		return false
	}

	return true
}

func applyColorBackgroundProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "background-color":
		if r, g, b, a, ok := css.ParseColor(value); ok {
			style.BGColor = [4]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255, a}
		}
	case "background":
		applyBackgroundShorthand(style, value)
	case "background-image":
		applyBackgroundImageValue(style, value)
	default:
		return false
	}

	return true
}

// firstBackgroundColor returns the first parseable color field in a
// background shorthand without materializing strings.Fields. The full-value
// fast path covers the common single-color shorthand; the scanner preserves
// the previous field-by-field fallback for image/repeat tokens.
func firstBackgroundColor(value string) (int, int, int, float64, bool) {
	if r, g, b, a, ok := css.ParseColor(value); ok {
		return r, g, b, a, true
	}

	for start := 0; start < len(value); {
		for start < len(value) {
			runeValue, size := utf8.DecodeRuneInString(value[start:])
			if !unicode.IsSpace(runeValue) {
				break
			}

			start += size
		}

		end := start
		for end < len(value) {
			runeValue, size := utf8.DecodeRuneInString(value[end:])
			if unicode.IsSpace(runeValue) {
				break
			}

			end += size
		}

		if start < end {
			if r, g, b, a, ok := css.ParseColor(value[start:end]); ok {
				return r, g, b, a, true
			}
		}

		start = end
	}

	return 0, 0, 0, 0, false
}

// applyTextGroup handles typography and list props.
func applyTextGroup(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext,
	parent *ResolvedStyle, hasParent bool,
) bool {
	if applyTextLayoutProps(style, prop, value) {
		return true
	}

	if applyTextWrapProps(style, prop, value) {
		return true
	}

	if applyTextDecorationProps(style, prop, value, parent, hasParent) {
		return true
	}

	if applyListProps(style, prop, value) {
		return true
	}

	if applyGeneratedContentProps(style, prop, value) {
		return true
	}

	return applyTextSpacingProps(style, prop, value, fsize, ctx)
}

func applyTextLayoutProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "line-height":
		style.LineHeightUnitless = 0
		if ratio, ok := css.ParseNumber(value); ok {
			style.LineHeightUnitless = ratio
		}

		style.LineHeight = lineHeight(value, style.FontSize)
	case "text-align":
		setTextAlignValue(style, value)
	case "text-transform":
		setTextTransformValue(style, value)
	case "vertical-align":
		setVerticalAlignValue(style, value)
	case "white-space":
		setWhiteSpaceValue(style, value)
	default:
		return false
	}

	return true
}

func setTextTransformValue(style *ResolvedStyle, value string) {
	switch value {
	case textTransformNone, textTransformUppercase, textTransformLowercase, textTransformCapitalize:
		style.TextTransform = value
	}
}

func setTextAlignValue(style *ResolvedStyle, value string) {
	switch value {
	case floatLeft, fxCenter, cssTextAlignJustify:
		style.TextAlign = value
	case floatRight, fxEnd:
		style.TextAlign = floatRight
	case fxStart:
		style.TextAlign = floatLeft
	}
}

func setVerticalAlignValue(style *ResolvedStyle, value string) {
	switch value {
	case "baseline", cssVerticalAlignTop, "middle", cssVerticalAlignBottom:
		style.VerticalAlign = value
		style.VerticalAlignShift = 0
	default:
		if shift, ok := plainLength(value, style.FontSize, 0); ok {
			style.VerticalAlign = "baseline"
			style.VerticalAlignShift = shift
		}
	}
}

func setWhiteSpaceValue(style *ResolvedStyle, value string) {
	switch value {
	case contentNormal, cssWhiteSpaceNowrap, cssWhiteSpacePre,
		cssWhiteSpacePreWrap, cssWhiteSpacePreLine:
		style.WhiteSpace = value
	}
}

func applyTextWrapProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "overflow-wrap", "word-wrap":
		setOverflowWrapValue(style, value)
	case "word-break":
		setWordBreakValue(style, value)
	default:
		return false
	}

	return true
}

func setOverflowWrapValue(style *ResolvedStyle, value string) {
	// word-wrap is the legacy alias of overflow-wrap.
	switch value {
	case contentNormal, "break-word", overflowWrapAnywhere:
		style.OverflowWrap = value
	case "break-spaces":
		// Treat like anywhere for line breaking (extra space preservation omitted).
		style.OverflowWrap = overflowWrapAnywhere
	}
}

func setWordBreakValue(style *ResolvedStyle, value string) {
	switch value {
	case contentNormal, "break-all", "keep-all":
		style.WordBreak = value
	case "break-word":
		// Legacy alias ≈ overflow-wrap:anywhere + word-break:normal.
		style.OverflowWrap = overflowWrapAnywhere
	}
}

func applyTextDecorationProps(style *ResolvedStyle, prop, value string, parent *ResolvedStyle, hasParent bool) bool {
	switch prop {
	case "text-decoration":
		switch value {
		case cssTextDecorationUnderline:
			style.TextDecoration = cssTextDecorationUnderline
		case "line-through":
			style.TextDecoration = "line-through"
		case cssDisplayNone:
			style.TextDecoration = cssDisplayNone
		case inheritKeyword:
			if hasParent && parent != nil {
				style.TextDecoration = parent.TextDecoration
			}
		}
	default:
		return false
	}

	return true
}

func applyListProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "list-style-type":
		if t := parseListStyleType(value); t != "" {
			style.ListStyleType = t
		}
	case "list-style-position":
		if pos := parseListStylePosition(value); pos != "" {
			style.ListStylePosition = pos
		}
	case "list-style":
		applyListStyleImageValue(style, value)

		for _, tok := range strings.Fields(value) {
			if t := parseListStyleType(tok); t != "" {
				style.ListStyleType = t
			}

			if pos := parseListStylePosition(tok); pos != "" {
				style.ListStylePosition = pos
			}
		}
	default:
		return false
	}

	return true
}

func applyTextSpacingProps(style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext) bool {
	switch prop {
	case "letter-spacing":
		style.LetterSpacing = marginLen(value, fsize, ctx.viewportW)
	case "word-spacing":
		if value == contentNormal {
			style.WordSpacing = 0
		} else {
			style.WordSpacing = marginLen(value, fsize, ctx.viewportW)
		}
	case "text-indent":
		style.TextIndent = marginLen(value, fsize, ctx.viewportW)
	default:
		return false
	}

	return true
}

// applyTableBreakGroup handles table borders/spacing and page-break props.
func applyTableBreakGroup(
	style *ResolvedStyle, prop, value string, fsize float64, ctx *styleContext, parent *ResolvedStyle, _ bool,
) bool {
	switch prop {
	case "border-collapse", "border-spacing", "table-layout", "caption-side":
		return applyTableProps(style, prop, value, fsize, ctx.viewportW)
	case "page-break-before", "break-before", "page-break-after", "break-after",
		"page-break-inside", "break-inside":
		return applyPageBreakProps(style, prop, value)
	case pageKeyword:
		return applyPageNameProp(style, value, parent)
	case "orphans", "widows":
		return applyOrphansWidowsProps(style, prop, value)
	case "container-type", "container-name", containerKeyword:
		return applyContainerProps(style, prop, value)
	default:
		return false
	}
}

func applyTableProps(style *ResolvedStyle, prop, value string, fsize, viewportW float64) bool {
	switch prop {
	case "border-collapse":
		if value == "collapse" || value == "separate" {
			style.BorderCollapse = value
		}
	case "border-spacing":
		style.BorderSpacing = marginLen(value, fsize, viewportW)
	case "table-layout":
		if value == positionFixed || value == overflowAuto {
			style.TableLayout = value
		}
	case "caption-side":
		applyCaptionSideValue(style, value)
	default:
		return false
	}

	return true
}

func applyCaptionSideValue(style *ResolvedStyle, value string) {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case cssVerticalAlignTop, cssVerticalAlignBottom, floatLeft, floatRight:
		style.CaptionSide = value
	}
}

func applyPageBreakProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "page-break-before", "break-before":
		return applyBreakBeforeProps(style, value)
	case "page-break-after", "break-after":
		return applyBreakAfterProps(style, value)
	case "page-break-inside", "break-inside":
		return applyBreakInsideProps(style, value)
	default:
		return false
	}
}

func applyBreakBeforeProps(style *ResolvedStyle, value string) bool {
	// column → page always is a multicol approximation.
	// avoid-column is column-only (CSS Break) — do NOT map to page avoid
	// (wiki .mw-references-columns li{break-inside:avoid-column} was
	// leaving huge gaps between reference list items).
	switch value {
	case pageBreakAlways, fxCol, pageKeyword, floatLeft, floatRight:
		style.PageBreakBefore = pageBreakAlways
	case avoidKeyword, avoidPageValue:
		style.PageBreakBefore = avoidKeyword
	default:
		return false
	}

	return true
}

func applyBreakAfterProps(style *ResolvedStyle, value string) bool {
	switch value {
	case pageBreakAlways, fxCol, pageKeyword, floatLeft, floatRight:
		style.PageBreakAfter = pageBreakAlways
	case avoidKeyword, avoidPageValue:
		style.PageBreakAfter = avoidKeyword
	default:
		return false
	}

	return true
}

func applyBreakInsideProps(style *ResolvedStyle, value string) bool {
	// avoid-column: ignored for page pagination.
	switch value {
	case pageBreakAlways, pageKeyword:
		style.PageBreakInside = pageBreakAlways
	case avoidKeyword, avoidPageValue:
		style.PageBreakInside = avoidKeyword
	default:
		return false
	}

	return true
}

func applyPageNameProp(style *ResolvedStyle, value string, parent *ResolvedStyle) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}

	low := strings.ToLower(value)
	switch low {
	case "auto", inheritKeyword:
		if parent != nil {
			style.PageName = parent.PageName
		}

		return true
	case "initial", "unset":
		style.PageName = ""

		return true
	}

	if !css.IsIdentToken(value) {
		return false
	}

	style.PageName = low

	return true
}

func applyOrphansWidowsProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "orphans":
		if n, ok := parseOrphansWidowsInt(value); ok {
			style.Orphans = n
		}
	case "widows":
		if n, ok := parseOrphansWidowsInt(value); ok {
			style.Widows = n
		}
	default:
		return false
	}

	return true
}

func applyContainerProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "container-type":
		switch strings.ToLower(value) {
		case contentNormal, "size", "inline-size":
			style.ContainerType = strings.ToLower(value)
		}
	case "container-name":
		style.ContainerName = css.ParseContainerNameValue(value)
	case containerKeyword:
		name, ctype := css.ParseContainerShorthand(value)
		style.ContainerName = name

		if ctype != "" {
			style.ContainerType = ctype
		}
	default:
		return false
	}

	return true
}

// applyTransformGroup handles transform and transform-origin.
func applyTransformGroup(
	style *ResolvedStyle, prop, value string, fsize float64, _ *styleContext, _ *ResolvedStyle, _ bool,
) bool {
	switch prop {
	case "transform":
		// Animations/transitions ignored: cascaded static value only.
		if m, has, ok := parseTransformList(value, fsize); ok {
			style.Transform = m
			style.HasTransform = has
		}
	case "transform-origin":
		if spec, ok := parseTransformOrigin(value, fsize); ok {
			style.TransformOrigin = spec
		}
	default:
		return false
	}

	return true
}

// applyIgnoredGroup parses but ignores the animation/transition family.
func applyIgnoredGroup(st *ResolvedStyle, prop, value string) {
	_ = st
	_ = prop
	_ = value
}

// parseOrphansWidowsInt accepts a CSS <integer ≥ 1>. Invalid values are
// ignored by the caller (declaration dropped).
func parseOrphansWidowsInt(value string) (int, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	n, err := strconv.Atoi(value)
	if err != nil || n < 1 {
		return 0, false
	}

	return n, true
}

// parseColumnsShorthand parses CSS columns: [ <column-width> || <column-count> ].
// Both longhands reset to auto before applying tokens (CSS Cascading).
