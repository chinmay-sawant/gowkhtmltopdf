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
	switch prop {
	case "font-synthesis":
		if ok := applyFontSynthesisShorthand(style, value); !ok {
			return true
		}
	case "font-synthesis-weight":
		if allow, ok := parseFontSynthesisLonghand(value); ok {
			style.FontSynthesisWeight = allow
		}
	case "font-synthesis-style":
		if allow, ok := parseFontSynthesisLonghand(value); ok {
			style.FontSynthesisStyle = allow
		}
	case "font-synthesis-small-caps":
		if allow, ok := parseFontSynthesisLonghand(value); ok {
			style.FontSynthesisSmallCaps = allow
		}
	case "font-synthesis-position":
		if allow, ok := parseFontSynthesisLonghand(value); ok {
			style.FontSynthesisPosition = allow
		}
	default:
		return false
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
		case "small-caps":
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
