//nolint:cyclop,funlen,mnd,goconst,wsl,nlreturn // text and decoration property helpers
package layout

import (
	"strconv"
	"strings"
)

// applyTextPropsWave3 handles CSS text and text-decoration properties.
func applyTextPropsWave3(
	style *ResolvedStyle, prop, value string, fsize float64, _ *ResolvedStyle, _ bool,
) bool {
	switch prop {
	case "text-align-last":
		setTextAlignLast(style, value)
	case "text-align-all":
		setTextAlignValue(style, value)
		setTextAlignLast(style, value)
	case "tab-size":
		setTabSize(style, value, fsize)
	case "text-wrap":
		setTextWrap(style, value)
	case "text-wrap-mode":
		setTextWrapMode(style, value)
	case "text-wrap-style":
		style.TextWrapStyle = strings.ToLower(strings.TrimSpace(value))
	case "white-space-collapse":
		setWhiteSpaceCollapse(style, value)
	case "white-space-trim":
		style.WhiteSpaceTrim = strings.ToLower(strings.TrimSpace(value))
	case "hyphens":
		setHyphens(style, value)
	case "hyphenate-character":
		style.HyphenateCharacter = strings.Trim(strings.TrimSpace(value), `"'`)
	case "text-justify":
		style.TextJustify = strings.ToLower(strings.TrimSpace(value))
	case "line-break":
		style.LineBreak = strings.ToLower(strings.TrimSpace(value))

	case "text-decoration-line":
		setTextDecorationLine(style, value)
	case "text-decoration-color":
		if c, ok := parseUsedColor(value, style.Color); ok {
			style.TextDecorationColor = c
			style.TextDecorationColorSet = true
		}
	case "text-decoration-style":
		setTextDecorationStyle(style, value)
	case "text-decoration-thickness":
		setTextDecorationThickness(style, value, fsize)
	case "text-underline-offset":
		if off, ok := plainLength(value, fsize, 0); ok {
			style.TextUnderlineOffset = off
		}
	case "text-underline-position":
		style.TextUnderlinePosition = strings.ToLower(strings.TrimSpace(value))
	case "text-shadow":
		applyTextShadow(style, value, fsize)

	default:
		return false
	}

	return true
}

func setTextAlignLast(style *ResolvedStyle, value string) {
	val := strings.ToLower(strings.TrimSpace(value))
	switch val {
	case "auto", floatLeft, floatRight, fxCenter, cssTextAlignJustify, "start", "end":
		style.TextAlignLast = val
	}
}

func setTabSize(style *ResolvedStyle, value string, fsize float64) {
	val := strings.TrimSpace(value)
	if n, err := strconv.ParseFloat(val, 64); err == nil && n >= 0 {
		style.TabSize = n
		return
	}
	if length, ok := plainLength(val, fsize, 0); ok && length >= 0 {
		style.TabSize = length
	}
}

func setTextWrap(style *ResolvedStyle, value string) {
	style.TextWrap = strings.TrimSpace(value)
	for _, tok := range strings.Fields(value) {
		tok = strings.ToLower(tok)
		if tok == "wrap" || tok == "nowrap" {
			setTextWrapMode(style, tok)
		} else {
			style.TextWrapStyle = tok
		}
	}
}

func setTextWrapMode(style *ResolvedStyle, value string) {
	val := strings.ToLower(strings.TrimSpace(value))
	style.TextWrapMode = val
	if val == "nowrap" {
		style.WhiteSpace = cssWhiteSpaceNowrap
	} else if val == "wrap" && style.WhiteSpace == cssWhiteSpaceNowrap {
		style.WhiteSpace = contentNormal
	}
}

func setWhiteSpaceCollapse(style *ResolvedStyle, value string) {
	val := strings.ToLower(strings.TrimSpace(value))
	style.WhiteSpaceCollapse = val
	switch val {
	case "collapse":
		style.WhiteSpace = contentNormal
	case "preserve":
		style.WhiteSpace = cssWhiteSpacePre
	case "preserve-breaks":
		style.WhiteSpace = cssWhiteSpacePreLine
	case "preserve-spaces", "break-spaces":
		style.WhiteSpace = cssWhiteSpacePreWrap
	}
}

func setHyphens(style *ResolvedStyle, value string) {
	val := strings.ToLower(strings.TrimSpace(value))
	switch val {
	case "none", "manual", "auto":
		style.Hyphens = val
	}
}

func setTextDecorationLine(style *ResolvedStyle, value string) {
	val := strings.ToLower(strings.TrimSpace(value))
	style.TextDecorationLine = val
	switch {
	case strings.Contains(val, "underline"):
		style.TextDecoration = cssTextDecorationUnderline
	case strings.Contains(val, "line-through"):
		style.TextDecoration = cssTextDecorationLineThrough
	case val == "none":
		style.TextDecoration = cssDisplayNone
	}
}

func setTextDecorationStyle(style *ResolvedStyle, value string) {
	val := strings.ToLower(strings.TrimSpace(value))
	switch val {
	case solidKeyword, "double", "dotted", "dashed", "wavy":
		style.TextDecorationStyle = val
	}
}

func setTextDecorationThickness(style *ResolvedStyle, value string, fsize float64) {
	val := strings.TrimSpace(value)
	if val == "auto" || val == "from-font" {
		style.TextDecorationThickness = 0
		return
	}
	if th, ok := plainLength(val, fsize, 0); ok && th > 0 {
		style.TextDecorationThickness = th
	}
}

func applyTextShadow(style *ResolvedStyle, value string, fsize float64) {
	val := strings.TrimSpace(value)
	if val == "" || strings.EqualFold(val, cssDisplayNone) {
		style.TextShadowSet = false
		style.TextShadowX, style.TextShadowY, style.TextShadowBlur = 0, 0, 0
		return
	}

	parts := strings.Fields(val)
	if len(parts) >= 2 {
		if x, ok := plainLength(parts[0], fsize, 0); ok {
			style.TextShadowX = x
		}
		if y, ok := plainLength(parts[1], fsize, 0); ok {
			style.TextShadowY = y
		}
		style.TextShadowSet = true
	}
	if len(parts) >= 3 {
		if b, ok := plainLength(parts[2], fsize, 0); ok && b >= 0 {
			style.TextShadowBlur = b
		}
	}
	if len(parts) >= 4 {
		if c, ok := parseUsedColor(parts[3], style.Color); ok {
			style.TextShadowColor = c
		}
	} else {
		style.TextShadowColor = style.Color
	}
}
