package layout

import (
	"strconv"
	"strings"
)

// applyFontWidthProps owns font-width and the legacy font-stretch alias.
// Both write FontWidth percent; fontWidthScale then scales measure/paint
// advances. Bundled Liberation faces have no width masters, so face lookup
// still ignores the percent.
func applyFontWidthProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext,
	_ *ResolvedStyle, _ bool,
) bool {
	switch prop {
	case "font-width", "font-stretch":
		if width, ok := parseFontWidthValue(value); ok {
			style.FontWidth = width
		}
	default:
		return false
	}

	return true
}

//nolint:cyclop,mnd // CSS keyword table intentionally mirrors the spec values.
func parseFontWidthValue(raw string) (float64, bool) {
	value := strings.ToLower(strings.TrimSpace(raw))
	if value == "" {
		return 0, false
	}

	switch value {
	case "ultra-condensed":
		return 50, true
	case "extra-condensed":
		return 62.5, true
	case "condensed":
		return 75, true
	case "semi-condensed":
		return 87.5, true
	case "normal":
		return 100, true
	case "semi-expanded":
		return 112.5, true
	case "expanded":
		return 125, true
	case "extra-expanded":
		return 150, true
	case "ultra-expanded":
		return 200, true
	}

	if strings.HasSuffix(value, "%") {
		n, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimSuffix(value, "%")), 64)
		if err != nil || n <= 0 {
			return 0, false
		}

		return n, true
	}

	return 0, false
}
