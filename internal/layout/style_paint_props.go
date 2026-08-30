package layout

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

const (
	currentColorKeyword = "currentcolor"
	propContent         = "content"
)

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

func applyOutlineProps(style *ResolvedStyle, prop, value string, fsize float64) bool {
	if applyBoxShadowProp(style, prop, value, fsize) {
		return true
	}

	if applySVGPresentationProps(style, prop, value, fsize) {
		return true
	}

	if prop == "outline" {
		applyOutlineShorthand(style, value, fsize)

		return true
	}

	return applyOutlineLonghands(style, prop, value, fsize)
}

//nolint:cyclop // SVG presentation longhands
func applySVGPresentationProps(style *ResolvedStyle, prop, value string, fsize float64) bool {
	switch prop {
	case "fill":
		if color, parsed := parseUsedColor(value, style.Color); parsed {
			style.Fill = color
			style.FillSet = true
		}
	case "fill-opacity":
		if opacity, parsed := parseOpacityValue(value); parsed {
			style.FillOpacity = opacity
		}
	case "stroke":
		if color, parsed := parseUsedColor(value, style.Color); parsed {
			style.Stroke = color
			style.StrokeSet = true
		}
	case "stroke-width":
		if width, parsed := plainLength(value, fsize, 0); parsed {
			style.StrokeWidth = width
			style.StrokeWidthSet = true
		}
	case "stroke-opacity":
		if opacity, parsed := parseOpacityValue(value); parsed {
			style.StrokeOpacity = opacity
		}
	case "stroke-dasharray", "stroke-dashoffset", "stroke-linecap", "stroke-linejoin",
		"stroke-miterlimit", "fill-rule", "clip-rule", "color-interpolation",
		"color-interpolation-filters", "shape-rendering", "text-anchor",
		"dominant-baseline", "alignment-baseline", "clip-path", "clip",
		"overflow-clip-margin", "scroll-margin", "scroll-margin-top", "scroll-margin-right",
		"scroll-margin-bottom", "scroll-margin-left", "ruby-align", "ruby-position",
		"ruby-merge", "ruby-overhang":
		return applyLeftoversProps(style, prop, value, fsize)
	default:
		return false
	}

	return true
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
	applyBoxShadowValue(style, value, fsize) //nolint:wsl

	return true
}

func applyBoxShadowValue(style *ResolvedStyle, value string, fsize float64) {
	value = strings.TrimSpace(value)
	if value == "" {
		return
	}

	if strings.EqualFold(value, cssDisplayNone) {
		clearBoxShadow(style)

		return
	}

	shadows := parseBoxShadowList(value, style.Color, fsize)
	if len(shadows) == 0 {
		return
	}

	first := shadows[0]
	style.BoxShadowX = first.x
	style.BoxShadowY = first.y
	style.BoxShadowBlur = first.blur
	style.BoxShadowSpread = first.spread
	style.BoxShadowColor = first.color
	style.BoxShadowInset = first.inset
	style.BoxShadowRaw = value
	style.BoxShadowSet = true
}

func clearBoxShadow(style *ResolvedStyle) {
	style.BoxShadowX = 0
	style.BoxShadowY = 0
	style.BoxShadowBlur = 0
	style.BoxShadowSpread = 0
	style.BoxShadowColor = [3]float64{}
	style.BoxShadowSet = false
	style.BoxShadowInset = false
	style.BoxShadowRaw = ""
}

func parseRuleShorthand(value string, fsize float64, current [3]float64) (float64, string, [3]float64, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, "", [3]float64{}, false
	}

	// Unspecified shorthand components reset to CSS initial values.
	width := borderWidth(mediumKeyword, fsize)
	ruleStyle := cssDisplayNone
	color := current

	if strings.EqualFold(value, cssDisplayNone) {
		return width, ruleStyle, color, true
	}

	for start := 0; ; {
		token, next, ok := nextSpaceToken(value, start)
		if !ok {
			return width, ruleStyle, color, true
		}

		if parsedStyle, parsed := parseOutlineStyle(token); parsed {
			ruleStyle = parsedStyle
		} else if parsedWidth, parsed := parseOutlineWidth(token, fsize); parsed {
			width = parsedWidth
		} else if parsedColor, parsed := parseUsedColor(token, current); parsed {
			color = parsedColor
		}

		start = next
	}
}

func applyOutlineShorthand(style *ResolvedStyle, value string, fsize float64) {
	width, outlineStyle, color, ok := parseRuleShorthand(value, fsize, style.Color)
	if !ok {
		return
	}

	style.OutlineWidth = width
	style.OutlineStyle = outlineStyle
	style.OutlineColor = color
	style.OutlineColorSet = true
}

// applyRadiusLonghand sets one corner radius. Percentages store the percent
// number on both the corner field and BorderRadiusPercent (paint uses the
// uniform percent path). Absolute lengths clear BorderRadiusPercent so paint
// uses the per-corner fields.
//
//nolint:cyclop // radius longhands
func applyRadiusLonghand(style *ResolvedStyle, prop, value string, fsize float64) bool {
	switch prop {
	case "border-top-left-radius":
		setCornerRadius(style, &style.BorderRadiusTopLeft, &style.BorderRadiusTopLeftY, value, fsize)
	case "border-top-right-radius":
		setCornerRadius(style, &style.BorderRadiusTopRight, &style.BorderRadiusTopRightY, value, fsize)
	case "border-bottom-right-radius":
		setCornerRadius(style, &style.BorderRadiusBottomRight, &style.BorderRadiusBottomRightY, value, fsize)
	case "border-bottom-left-radius":
		setCornerRadius(style, &style.BorderRadiusBottomLeft, &style.BorderRadiusBottomLeftY, value, fsize)
	case "border-top-radius":
		setCornerRadius(style, &style.BorderRadiusTopLeft, &style.BorderRadiusTopLeftY, value, fsize)
		setCornerRadius(style, &style.BorderRadiusTopRight, &style.BorderRadiusTopRightY, value, fsize)
	case "border-bottom-radius":
		setCornerRadius(style, &style.BorderRadiusBottomLeft, &style.BorderRadiusBottomLeftY, value, fsize)
		setCornerRadius(style, &style.BorderRadiusBottomRight, &style.BorderRadiusBottomRightY, value, fsize)
	case "border-left-radius":
		setCornerRadius(style, &style.BorderRadiusTopLeft, &style.BorderRadiusTopLeftY, value, fsize)
		setCornerRadius(style, &style.BorderRadiusBottomLeft, &style.BorderRadiusBottomLeftY, value, fsize)
	case "border-right-radius":
		setCornerRadius(style, &style.BorderRadiusTopRight, &style.BorderRadiusTopRightY, value, fsize)
		setCornerRadius(style, &style.BorderRadiusBottomRight, &style.BorderRadiusBottomRightY, value, fsize)
	case "border-start-start-radius", "border-start-end-radius",
		"border-end-start-radius", "border-end-end-radius",
		"border-block-start-radius", "border-block-end-radius",
		"border-inline-start-radius", "border-inline-end-radius":
		return applyLogicalRadiusLonghand(style, prop, value, fsize)
	default:
		return false
	}

	return true
}

func setCornerRadius(style *ResolvedStyle, destX, destY *float64, value string, fsize float64) {
	rxTok, ryTok, hasY := splitCornerRadiusTokens(value)
	if rxTok == "" {
		return
	}

	if applyCornerRadiusPercent(style, destX, destY, rxTok) {
		return
	}

	if !applyCornerRadiusX(style, destX, destY, rxTok, fsize) {
		return
	}

	if hasY {
		applyCornerRadiusY(destY, ryTok, fsize)
	}
}

func applyCornerRadiusX(style *ResolvedStyle, destX, destY *float64, token string, fsize float64) bool {
	radius, ok := lengthBox(token, fsize, 0, cssDisplayNone)
	if !ok || radius < 0 {
		return false
	}

	*destX = radius
	*destY = 0
	style.BorderRadius = 0
	style.BorderRadiusPercent = -1

	return true
}

func applyCornerRadiusY(destY *float64, token string, fsize float64) {
	if token == "" {
		return
	}

	if _, unit, parsed := css.ParseLength(token); parsed && unit == "%" {
		return
	}

	radiusY, ok := lengthBox(token, fsize, 0, cssDisplayNone)
	if !ok || radiusY < 0 {
		return
	}

	*destY = radiusY
}

func applyCornerRadiusPercent(style *ResolvedStyle, destX, destY *float64, token string) bool {
	percent, unit, ok := css.ParseLength(token)
	if !ok || unit != "%" || percent < 0 {
		return false
	}

	*destX = percent
	*destY = 0
	style.BorderRadius = 0
	style.BorderRadiusPercent = percent

	return true
}

func applyBackgroundShorthand(style *ResolvedStyle, value string) {
	applyBackgroundImageValue(style, value)

	if r, g, b, a, ok := firstBackgroundColor(value); ok {
		style.BGColor = [4]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255, a}
	}
}

func applyBackgroundImageValue(style *ResolvedStyle, value string) {
	trimmed := strings.TrimSpace(value)
	if strings.EqualFold(trimmed, cssDisplayNone) {
		style.BackgroundImage = ""

		return
	}

	if url, ok := firstCSSUrl(trimmed); ok && !strings.Contains(trimmed, ",") && !isGradientFunc(trimmed) {
		style.BackgroundImage = url

		return
	}

	if trimmed != "" {
		style.BackgroundImage = trimmed
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
		style.QuotesRaw = value
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
	case propContent:
		style.Content = strings.TrimSpace(value)
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
