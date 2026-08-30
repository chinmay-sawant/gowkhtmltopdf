//nolint:cyclop,funlen,mnd,goconst,wsl,nlreturn,varnamelen // logical border and radius dispatch
package layout

import (
	"strings"
)

func logicalBorderSide(style *ResolvedStyle, side string) *border {
	switch side {
	case "block-start":
		switch style.WritingMode {
		case writingModeVerticalRL:
			return &style.BorderRight
		case writingModeVerticalLR:
			return &style.BorderLeft
		default:
			return &style.BorderTop
		}
	case "block-end":
		switch style.WritingMode {
		case writingModeVerticalRL:
			return &style.BorderLeft
		case writingModeVerticalLR:
			return &style.BorderRight
		default:
			return &style.BorderBottom
		}
	case "inline-start":
		if isVerticalWritingMode(style.WritingMode) {
			return &style.BorderTop
		}
		if style.Direction == "rtl" {
			return &style.BorderRight
		}
		return &style.BorderLeft
	case "inline-end":
		if isVerticalWritingMode(style.WritingMode) {
			return &style.BorderBottom
		}
		if style.Direction == "rtl" {
			return &style.BorderLeft
		}
		return &style.BorderRight
	default:
		return nil
	}
}

func setLogicalBorderWidth(style *ResolvedStyle, side, value string, fsize float64) {
	b := logicalBorderSide(style, side)
	if b == nil {
		return
	}
	b.Width = borderWidth(value, fsize)
	b.PaintWidth = borderPaintWidth(value, fsize)
}

func setLogicalBorderStyle(style *ResolvedStyle, side, value string) {
	b := logicalBorderSide(style, side)
	if b == nil {
		return
	}
	s := strings.ToLower(strings.TrimSpace(value))
	if s != solidKeyword && s != borderStyleDashed && s != borderStyleDotted {
		s = cssDisplayNone
	}
	b.Style = s
}

func setLogicalBorderColor(style *ResolvedStyle, side, value string) {
	b := logicalBorderSide(style, side)
	if b == nil {
		return
	}
	if c, ok := parseUsedColor(value, style.Color); ok {
		b.Color = c
	}
}

func applyLogicalBorderSideShorthand(style *ResolvedStyle, side, value string, fsize float64) bool {
	b := logicalBorderSide(style, side)
	if b == nil {
		return false
	}
	setBorderSide(style, b, value, fsize)
	return true
}

func applyLogicalBorderTwoValues(value string, apply func(val1, val2 string)) {
	toks := strings.Fields(strings.TrimSpace(value))
	if len(toks) >= 2 {
		apply(toks[0], toks[1])
	} else if len(toks) == 1 {
		apply(toks[0], toks[0])
	}
}

// applyLogicalBorder handles all 24 CSS logical border properties.
func applyLogicalBorder(style *ResolvedStyle, prop, value string, fsize float64) bool {
	switch prop {
	case "border-block":
		applyLogicalBorderSideShorthand(style, "block-start", value, fsize)
		applyLogicalBorderSideShorthand(style, "block-end", value, fsize)
	case "border-block-start":
		return applyLogicalBorderSideShorthand(style, "block-start", value, fsize)
	case "border-block-end":
		return applyLogicalBorderSideShorthand(style, "block-end", value, fsize)
	case "border-block-width":
		applyLogicalBorderTwoValues(value, func(v1, v2 string) {
			setLogicalBorderWidth(style, "block-start", v1, fsize)
			setLogicalBorderWidth(style, "block-end", v2, fsize)
		})
	case "border-block-style":
		applyLogicalBorderTwoValues(value, func(v1, v2 string) {
			setLogicalBorderStyle(style, "block-start", v1)
			setLogicalBorderStyle(style, "block-end", v2)
		})
	case "border-block-color":
		applyLogicalBorderTwoValues(value, func(v1, v2 string) {
			setLogicalBorderColor(style, "block-start", v1)
			setLogicalBorderColor(style, "block-end", v2)
		})
	case "border-block-start-width":
		setLogicalBorderWidth(style, "block-start", value, fsize)
	case "border-block-start-style":
		setLogicalBorderStyle(style, "block-start", value)
	case "border-block-start-color":
		setLogicalBorderColor(style, "block-start", value)
	case "border-block-end-width":
		setLogicalBorderWidth(style, "block-end", value, fsize)
	case "border-block-end-style":
		setLogicalBorderStyle(style, "block-end", value)
	case "border-block-end-color":
		setLogicalBorderColor(style, "block-end", value)

	case "border-inline":
		applyLogicalBorderSideShorthand(style, "inline-start", value, fsize)
		applyLogicalBorderSideShorthand(style, "inline-end", value, fsize)
	case "border-inline-start":
		return applyLogicalBorderSideShorthand(style, "inline-start", value, fsize)
	case "border-inline-end":
		return applyLogicalBorderSideShorthand(style, "inline-end", value, fsize)
	case "border-inline-width":
		applyLogicalBorderTwoValues(value, func(v1, v2 string) {
			setLogicalBorderWidth(style, "inline-start", v1, fsize)
			setLogicalBorderWidth(style, "inline-end", v2, fsize)
		})
	case "border-inline-style":
		applyLogicalBorderTwoValues(value, func(v1, v2 string) {
			setLogicalBorderStyle(style, "inline-start", v1)
			setLogicalBorderStyle(style, "inline-end", v2)
		})
	case "border-inline-color":
		applyLogicalBorderTwoValues(value, func(v1, v2 string) {
			setLogicalBorderColor(style, "inline-start", v1)
			setLogicalBorderColor(style, "inline-end", v2)
		})
	case "border-inline-start-width":
		setLogicalBorderWidth(style, "inline-start", value, fsize)
	case "border-inline-start-style":
		setLogicalBorderStyle(style, "inline-start", value)
	case "border-inline-start-color":
		setLogicalBorderColor(style, "inline-start", value)
	case "border-inline-end-width":
		setLogicalBorderWidth(style, "inline-end", value, fsize)
	case "border-inline-end-style":
		setLogicalBorderStyle(style, "inline-end", value)
	case "border-inline-end-color":
		setLogicalBorderColor(style, "inline-end", value)

	default:
		return false
	}

	return true
}

func logicalRadiusCorner(style *ResolvedStyle, corner string) (*float64, *float64) {
	switch corner {
	case "start-start":
		if isVerticalWritingMode(style.WritingMode) {
			if style.WritingMode == writingModeVerticalRL {
				return &style.BorderRadiusTopRight, &style.BorderRadiusTopRightY
			}
			return &style.BorderRadiusTopLeft, &style.BorderRadiusTopLeftY
		}
		if style.Direction == "rtl" {
			return &style.BorderRadiusTopRight, &style.BorderRadiusTopRightY
		}
		return &style.BorderRadiusTopLeft, &style.BorderRadiusTopLeftY

	case "start-end":
		if isVerticalWritingMode(style.WritingMode) {
			if style.WritingMode == writingModeVerticalRL {
				return &style.BorderRadiusBottomRight, &style.BorderRadiusBottomRightY
			}
			return &style.BorderRadiusBottomLeft, &style.BorderRadiusBottomLeftY
		}
		if style.Direction == "rtl" {
			return &style.BorderRadiusTopLeft, &style.BorderRadiusTopLeftY
		}
		return &style.BorderRadiusTopRight, &style.BorderRadiusTopRightY

	case "end-start":
		if isVerticalWritingMode(style.WritingMode) {
			if style.WritingMode == writingModeVerticalRL {
				return &style.BorderRadiusTopLeft, &style.BorderRadiusTopLeftY
			}
			return &style.BorderRadiusTopRight, &style.BorderRadiusTopRightY
		}
		if style.Direction == "rtl" {
			return &style.BorderRadiusBottomRight, &style.BorderRadiusBottomRightY
		}
		return &style.BorderRadiusBottomLeft, &style.BorderRadiusBottomLeftY

	case "end-end":
		if isVerticalWritingMode(style.WritingMode) {
			if style.WritingMode == writingModeVerticalRL {
				return &style.BorderRadiusBottomLeft, &style.BorderRadiusBottomLeftY
			}
			return &style.BorderRadiusBottomRight, &style.BorderRadiusBottomRightY
		}
		if style.Direction == "rtl" {
			return &style.BorderRadiusBottomLeft, &style.BorderRadiusBottomLeftY
		}
		return &style.BorderRadiusBottomRight, &style.BorderRadiusBottomRightY

	default:
		return nil, nil
	}
}

// applyLogicalRadiusLonghand handles CSS logical corner radii and logical side radii.
func applyLogicalRadiusLonghand(style *ResolvedStyle, prop, value string, fsize float64) bool {
	switch prop {
	case "border-start-start-radius":
		dx, dy := logicalRadiusCorner(style, "start-start")
		if dx != nil {
			setCornerRadius(style, dx, dy, value, fsize)
		}
	case "border-start-end-radius":
		dx, dy := logicalRadiusCorner(style, "start-end")
		if dx != nil {
			setCornerRadius(style, dx, dy, value, fsize)
		}
	case "border-end-start-radius":
		dx, dy := logicalRadiusCorner(style, "end-start")
		if dx != nil {
			setCornerRadius(style, dx, dy, value, fsize)
		}
	case "border-end-end-radius":
		dx, dy := logicalRadiusCorner(style, "end-end")
		if dx != nil {
			setCornerRadius(style, dx, dy, value, fsize)
		}
	case "border-block-start-radius":
		d1x, d1y := logicalRadiusCorner(style, "start-start")
		d2x, d2y := logicalRadiusCorner(style, "start-end")
		if d1x != nil && d2x != nil {
			setCornerRadius(style, d1x, d1y, value, fsize)
			setCornerRadius(style, d2x, d2y, value, fsize)
		}
	case "border-block-end-radius":
		d1x, d1y := logicalRadiusCorner(style, "end-start")
		d2x, d2y := logicalRadiusCorner(style, "end-end")
		if d1x != nil && d2x != nil {
			setCornerRadius(style, d1x, d1y, value, fsize)
			setCornerRadius(style, d2x, d2y, value, fsize)
		}
	case "border-inline-start-radius":
		d1x, d1y := logicalRadiusCorner(style, "start-start")
		d2x, d2y := logicalRadiusCorner(style, "end-start")
		if d1x != nil && d2x != nil {
			setCornerRadius(style, d1x, d1y, value, fsize)
			setCornerRadius(style, d2x, d2y, value, fsize)
		}
	case "border-inline-end-radius":
		d1x, d1y := logicalRadiusCorner(style, "start-end")
		d2x, d2y := logicalRadiusCorner(style, "end-end")
		if d1x != nil && d2x != nil {
			setCornerRadius(style, d1x, d1y, value, fsize)
			setCornerRadius(style, d2x, d2y, value, fsize)
		}
	default:
		return false
	}

	return true
}
