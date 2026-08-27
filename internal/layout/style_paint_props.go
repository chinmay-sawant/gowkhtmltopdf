package layout

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

const currentColorKeyword = "currentcolor"

func isCurrentColor(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), currentColorKeyword)
}

// parseUsedColor maps a CSS color token onto 0..1 RGB. currentColor uses the
// element's used color (already inherited or applied).
func parseUsedColor(value string, current [3]float64) ([3]float64, bool) {
	if isCurrentColor(value) {
		return current, true
	}

	r, g, b, _, ok := css.ParseColor(value)
	if !ok {
		return [3]float64{}, false
	}

	return [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}, true
}

// applyOutlineProps owns outline and its longhands. Outline is painted outside
// the border edge and must not change used Width/Height. box-shadow is applied
// here so the existing applyBorderGroup dispatch can reach it.
func applyOutlineProps(style *ResolvedStyle, prop, value string, fsize float64) bool {
	if applyBoxShadowProp(style, prop, value, fsize) {
		return true
	}

	if prop == "outline" {
		applyOutlineShorthand(style, value, fsize)

		return true
	}

	return applyOutlineLonghands(style, prop, value, fsize)
}

func applyOutlineLonghands(style *ResolvedStyle, prop, value string, fsize float64) bool {
	switch prop {
	case "outline-width":
		if width, parsed := parseOutlineWidth(value, fsize); parsed {
			style.OutlineWidth = width
		}
	case "outline-style":
		if outlineStyle, parsed := parseOutlineStyle(value); parsed {
			style.OutlineStyle = outlineStyle
		}
	case "outline-color":
		if color, parsed := parseUsedColor(value, style.Color); parsed {
			style.OutlineColor = color
			style.OutlineColorSet = true
		}
	case "outline-offset":
		if offset, parsed := plainLength(value, fsize, 0); parsed {
			style.OutlineOffset = offset
		}
	default:
		return false
	}

	return true
}

func applyBoxShadowProp(style *ResolvedStyle, prop, value string, fsize float64) bool {
	if prop != boxShadowProp {
		return false
	}

	applyBoxShadowValue(style, value, fsize)

	return true
}

func applyBoxShadowValue(style *ResolvedStyle, value string, fsize float64) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}

	layer := strings.TrimSpace(firstCommaLayer(value))
	if strings.EqualFold(layer, cssDisplayNone) {
		clearBoxShadow(style)

		return
	}

	shadow, ok := parseBoxShadowLayer(layer, style.Color, fsize)
	if !ok {
		return
	}

	style.BoxShadowX = shadow.x
	style.BoxShadowY = shadow.y
	style.BoxShadowBlur = shadow.blur
	style.BoxShadowColor = shadow.color
	style.BoxShadowSet = true
}

func clearBoxShadow(style *ResolvedStyle) {
	style.BoxShadowX = 0
	style.BoxShadowY = 0
	style.BoxShadowBlur = 0
	style.BoxShadowColor = [3]float64{}
	style.BoxShadowSet = false
}

func applyOutlineShorthand(style *ResolvedStyle, value string, fsize float64) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}

	// Unspecified shorthand components reset to CSS initial values.
	style.OutlineWidth = borderWidth(mediumKeyword, fsize)
	style.OutlineStyle = cssDisplayNone
	style.OutlineColor = style.Color
	style.OutlineColorSet = true

	if strings.EqualFold(value, cssDisplayNone) {
		return
	}

	for start := 0; ; {
		token, next, ok := nextSpaceToken(value, start)
		if !ok {
			return
		}

		applyOutlineToken(style, token, fsize)

		start = next
	}
}

func applyOutlineToken(style *ResolvedStyle, token string, fsize float64) {
	if outlineStyle, ok := parseOutlineStyle(token); ok {
		style.OutlineStyle = outlineStyle

		return
	}

	if width, ok := parseOutlineWidth(token, fsize); ok {
		style.OutlineWidth = width

		return
	}

	if color, ok := parseUsedColor(token, style.Color); ok {
		style.OutlineColor = color
		style.OutlineColorSet = true
	}
}

// applyRadiusLonghand sets one corner radius. Percentages store the percent
// number on both the corner field and BorderRadiusPercent (paint uses the
// uniform percent path). Absolute lengths clear BorderRadiusPercent so paint
// uses the per-corner fields.
func applyRadiusLonghand(style *ResolvedStyle, prop, value string, fsize float64) bool {
	switch prop {
	case "border-top-left-radius":
		setCornerRadius(style, &style.BorderRadiusTopLeft, value, fsize)
	case "border-top-right-radius":
		setCornerRadius(style, &style.BorderRadiusTopRight, value, fsize)
	case "border-bottom-right-radius":
		setCornerRadius(style, &style.BorderRadiusBottomRight, value, fsize)
	case "border-bottom-left-radius":
		setCornerRadius(style, &style.BorderRadiusBottomLeft, value, fsize)
	default:
		return false
	}

	return true
}

func setCornerRadius(style *ResolvedStyle, dest *float64, value string, fsize float64) {
	token := firstRadiusToken(value)
	if token == "" {
		return
	}

	if v, unit, ok := css.ParseLength(token); ok && unit == "%" {
		if v >= 0 {
			*dest = v
			style.BorderRadius = 0
			style.BorderRadiusPercent = v
		}

		return
	}

	radius, ok := lengthBox(token, fsize, 0, cssDisplayNone)
	if !ok || radius < 0 {
		return
	}

	*dest = radius
	style.BorderRadius = 0
	style.BorderRadiusPercent = -1
}

func firstRadiusToken(value string) string {
	value = strings.TrimSpace(value)
	if before, _, ok := strings.Cut(value, "/"); ok {
		value = strings.TrimSpace(before)
	}

	token, _, ok := nextSpaceToken(value, 0)
	if !ok {
		return ""
	}

	return token
}

func applyBackgroundShorthand(style *ResolvedStyle, value string) {
	applyBackgroundImageValue(style, value)

	if r, g, b, a, ok := firstBackgroundColor(value); ok {
		style.BGColor = [4]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255, a}
	}
}

func applyBackgroundImageValue(style *ResolvedStyle, value string) {
	if url, ok := firstCSSUrl(value); ok {
		style.BackgroundImage = url

		return
	}

	if strings.EqualFold(strings.TrimSpace(value), cssDisplayNone) {
		style.BackgroundImage = ""
	}
}

func firstCSSUrl(value string) (string, bool) {
	urls := css.FontFaceURLs(value)
	if len(urls) == 0 {
		return "", false
	}

	return urls[0], true
}

func applyGeneratedContentProps(style *ResolvedStyle, prop, value string) bool {
	switch prop {
	case "quotes":
		if openQuote, closeQuote, parsed := parseQuotesPair(value); parsed {
			style.QuotesOpen = openQuote
			style.QuotesClose = closeQuote
		}
	case "counter-reset":
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			style.CounterReset = trimmed
		}
	case "counter-increment":
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			style.CounterIncrement = trimmed
		}
	case "list-style-image":
		applyListStyleImageValue(style, value)
	default:
		return false
	}

	return true
}

// applyListStyleImageValue stores the first url(...) from list-style-image or
// the list-style shorthand. none (the whole value, or a lone none token with
// no url) clears the image so the type marker is used instead.
func applyListStyleImageValue(style *ResolvedStyle, value string) {
	if url, ok := firstCSSUrl(value); ok {
		style.ListStyleImage = url

		return
	}

	if listStyleImageNone(value) {
		style.ListStyleImage = ""
	}
}

func listStyleImageNone(value string) bool {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return false
	}

	if strings.EqualFold(trimmed, cssDisplayNone) {
		return true
	}

	for start := 0; ; {
		token, next, ok := nextSpaceToken(trimmed, start)
		if !ok {
			return false
		}

		if strings.EqualFold(token, cssDisplayNone) {
			return true
		}

		start = next
	}
}
