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

// applyLogicalBorder is now a no-op wrapper: logical border expansion is handled in cascade (style_cascade.go:expandLogicalBorder).
func applyLogicalBorder(style *ResolvedStyle, prop, value string, fsize float64) bool {
	return false
}

func logicalRadiusCorner(style *ResolvedStyle, corner string) (*float64, *float64) {
	switch corner {
	case "start-start":
		return &style.BorderRadiusTopLeft, &style.BorderRadiusTopLeftY
	case "start-end":
		return &style.BorderRadiusTopRight, &style.BorderRadiusTopRightY
	case "end-start":
		return &style.BorderRadiusBottomLeft, &style.BorderRadiusBottomLeftY
	case "end-end":
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
