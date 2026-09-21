package layout

import "strings"

// applyFontSynthesisProps owns font-synthesis and its longhands. Only
// font-synthesis-weight (and shorthand weight/none) has a paint consumer today
// (Op.NoFakeBold / FakeBoldFor). Style / small-caps / position are stored for
// cascade honesty and stay Partial until a consumer exists.
func applyFontSynthesisProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext,
	_ *ResolvedStyle, _ bool,
) bool {
	if prop == "font-synthesis" {
		applyFontSynthesisShorthand(style, value)

		return true
	}

	var target *bool

	switch prop {
	case "font-synthesis-weight":
		target = &style.FontSynthesisWeight
	case "font-synthesis-style":
		target = &style.FontSynthesisStyle
	case "font-synthesis-small-caps":
		target = &style.FontSynthesisSmallCaps
	case "font-synthesis-position":
		target = &style.FontSynthesisPosition
	default:
		return false
	}

	if allow, ok := parseFontSynthesisLonghand(value); ok {
		*target = allow
	}

	return true
}

func parseFontSynthesisLonghand(raw string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "auto":
		return true, true
	case "none":
		return false, true
	default:
		return false, false
	}
}

func applyFontSynthesisShorthand(style *ResolvedStyle, raw string) bool {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return false
	}

	if value == "none" {
		style.FontSynthesisWeight = false
		style.FontSynthesisStyle = false
		style.FontSynthesisSmallCaps = false
		style.FontSynthesisPosition = false

		return true
	}

	tokens := strings.Fields(value)
	if len(tokens) == 0 {
		return false
	}

	weight, styleAllow, smallCaps, position := false, false, false, false

	for _, tok := range tokens {
		switch tok {
		case "weight":
			weight = true
		case "style":
			styleAllow = true
		case fontVariantCapsSmall:
			smallCaps = true
		case "position":
			position = true
		default:
			return false
		}
	}

	style.FontSynthesisWeight = weight
	style.FontSynthesisStyle = styleAllow
	style.FontSynthesisSmallCaps = smallCaps
	style.FontSynthesisPosition = position

	return true
}

// textOpDisablesFakeBold reports whether CSS font-synthesis forbids fake bold.
func textOpDisablesFakeBold(sty *ResolvedStyle) bool {
	return sty != nil && !sty.FontSynthesisWeight
}
