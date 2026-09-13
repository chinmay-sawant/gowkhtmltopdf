package layout

import (
	"strconv"
	"strings"
)

// vertical-align keyword spellings and the proportional sub/super shifts. A
// positive shift raises a run from the baseline; every consumer subtracts it.
const (
	verticalAlignSub   = "sub"
	verticalAlignSuper = "super"

	verticalAlignSubRatio   = 0.2
	verticalAlignSuperRatio = 0.4
)

// alignedInlineTop is the canvas Y of an atomic inline box (image or
// inline-block). Keywords match CSS vertical-align; a length shift raises
// (positive) or lowers (negative) a baseline-aligned box.
func (e *engine) alignedInlineTop(item *inlineItem, lineY, lineH, baseline float64) float64 {
	if item.style == nil {
		return baseline - item.h
	}

	switch item.style.VerticalAlign {
	case cssVerticalAlignTop:
		return lineY
	case cssVerticalAlignMiddle:
		return lineY + (lineH-item.h)/2
	case cssVerticalAlignBottom:
		return lineY + lineH - item.h
	default:
		return baseline - item.h - e.scalePt(e.effectiveVerticalAlignShift(item.style))
	}
}

// effectiveVerticalAlignShift maps vertical-align keywords and lengths to a
// pt shift where positive raises. Handles sub/super and % of line-height
// in addition to plain <length> stored in VerticalAlignShift.
func (e *engine) effectiveVerticalAlignShift(style *ResolvedStyle) float64 {
	if style == nil {
		return 0
	}

	switch strings.ToLower(strings.TrimSpace(style.VerticalAlign)) {
	case verticalAlignSub:
		return style.FontSize * -verticalAlignSubRatio
	case verticalAlignSuper:
		return style.FontSize * verticalAlignSuperRatio
	}
	// If VerticalAlign looks like a percent (e.g. "50%"), compute against
	// line-height per CSS spec.
	if pct := strings.TrimSpace(style.VerticalAlign); strings.HasSuffix(pct, "%") {
		if percent, ok := parsePercent(pct); ok {
			lineH := lineHeightOf(style)

			return lineH * percent / oneHundred
		}
	}

	return style.VerticalAlignShift
}

func parsePercent(value string) (float64, bool) {
	trimmed := strings.TrimSpace(value)
	if !strings.HasSuffix(trimmed, "%") {
		return 0, false
	}

	num := strings.TrimSpace(strings.TrimSuffix(trimmed, "%"))
	parsed, err := strconv.ParseFloat(num, 64)

	if err != nil {
		return 0, false
	}

	return parsed, true
}
