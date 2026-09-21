//nolint:cyclop // CSS Text 4/5 spacing property parsers
package layout

import "strings"

// applyTextAutospaceProps owns text-autospace, text-spacing, text-spacing-trim,
// text-group-align, and text-fit. Named to avoid clashing with the existing
// letter/word-spacing helper applyTextSpacingProps in style_properties.go.
func applyTextAutospaceProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext,
	_ *ResolvedStyle, _ bool,
) bool {
	val := strings.ToLower(strings.TrimSpace(value))
	if val == "" {
		return false
	}

	switch prop {
	case "text-autospace":
		if isTextAutospaceValue(val) {
			style.TextAutospace = val
		}
	case "text-spacing":
		applyTextSpacingShorthand(style, val)
	case "text-spacing-trim":
		if isTextSpacingTrimValue(val) {
			style.TextSpacingTrim = val
		}
	case "text-group-align":
		if isTextGroupAlignValue(val) {
			style.TextGroupAlign = val
		}
	case "text-fit":
		// Stored for cascade honesty; no scale-search consumer yet (L).
		if isTextFitValue(val) {
			style.TextFit = val
		}
	default:
		return false
	}

	return true
}

func applyTextSpacingShorthand(style *ResolvedStyle, val string) {
	style.TextSpacing = val
	// Mirror trim keywords onto text-spacing-trim when present.
	for _, tok := range strings.Fields(val) {
		if isTextSpacingTrimValue(tok) {
			style.TextSpacingTrim = tok
		}
	}
}

func isTextAutospaceValue(val string) bool {
	switch val {
	case "no-autospace", contentNormal, "auto",
		textAutospaceIdeographAlpha, "ideograph-numeric", "punctuation",
		"insert", "replace":
		return true
	default:
		// Allow space-separated combinations used by the draft.
		for _, tok := range strings.Fields(val) {
			switch tok {
			case "ideograph-alpha", "ideograph-numeric", "punctuation",
				"insert", "replace", "auto", "no-autospace":
				continue
			default:
				return false
			}
		}

		return val != ""
	}
}

func isTextSpacingTrimValue(val string) bool {
	switch val {
	case "space-all", "normal", "space-first", "trim-start", "trim-both",
		"space-or-trim", "trim-all", "trim-auto":
		return true
	default:
		return false
	}
}

func isTextGroupAlignValue(val string) bool {
	switch val {
	case cssDisplayNone, "start", "end", "left", "right", "center":
		return true
	default:
		return false
	}
}

func isTextFitValue(val string) bool {
	switch val {
	case cssDisplayNone, "auto", "scale":
		return true
	default:
		return false
	}
}
