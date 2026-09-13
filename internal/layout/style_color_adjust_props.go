//nolint:cyclop // color-adjust, forced-colors, and dynamic-range property dispatch
package layout

import "strings"

// Print color adjustment keywords. The apply arm stores only these normalized
// spellings; paint consumers compare against the same constants.
const (
	colorAdjustEconomy = "economy"
	colorAdjustExact   = "exact"

	forcedColorAdjustAuto = "auto"
	forcedColorAdjustNone = "none"
	// forcedColorAdjustPreserveParentColor is spec syntax (css-color-adjust-1).
	// Print has no forced-colors mode, so the paint path treats it like auto.
	forcedColorAdjustPreserveParentColor = "preserve-parent-color"

	colorSchemeNormal = "normal"
	colorSchemeLight  = "light"
	colorSchemeDark   = "dark"

	dynamicRangeLimitNoLimit         = "no-limit"
	dynamicRangeLimitStandard        = "standard"
	dynamicRangeLimitHigh            = "high"
	dynamicRangeLimitConstrainedHigh = "constrained-high"
)

// applyColorAdjustProps owns color-adjust / print-color-adjust,
// forced-color-adjust, color-scheme, and dynamic-range-limit. Values are
// normalized to the constants above; anything else is dropped. The CSS-wide
// keywords are resolved here because inheritProps suppresses its parent copy
// for any declared property (style_cascade.go), so an explicit
// "inherit"/"initial"/"unset"/"revert" must land on this arm.
func applyColorAdjustProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext, parent *ResolvedStyle, hasParent bool,
) bool {
	switch prop {
	case "color-adjust", "print-color-adjust":
		parentValue := colorAdjustEconomy
		if parent != nil {
			parentValue = parent.ColorAdjust
		}

		if v, ok := inheritedToken(value, parentValue, colorAdjustEconomy, hasParent, normalizeColorAdjust); ok {
			style.ColorAdjust = v
		}
	case "forced-color-adjust":
		parentValue := forcedColorAdjustAuto
		if parent != nil {
			parentValue = parent.ForcedColorAdjust
		}

		if v, ok := inheritedToken(value, parentValue, forcedColorAdjustAuto, hasParent, normalizeForcedColorAdjust); ok {
			style.ForcedColorAdjust = v
		}
	case "color-scheme":
		parentValue := colorSchemeNormal
		if parent != nil {
			parentValue = parent.ColorScheme
		}

		if v, ok := inheritedToken(value, parentValue, colorSchemeNormal, hasParent, normalizeColorScheme); ok {
			style.ColorScheme = v
		}
	case "dynamic-range-limit":
		parentValue := dynamicRangeLimitNoLimit
		if parent != nil {
			parentValue = parent.DynamicRangeLimit
		}

		if v, ok := inheritedToken(value, parentValue, dynamicRangeLimitNoLimit, hasParent, normalizeDynamicRangeLimit); ok {
			style.DynamicRangeLimit = v
		}
	default:
		return false
	}

	return true
}

// inheritedToken resolves one declared token for an inherited property.
// CSS-wide keywords map to the parent's value (inherit/unset) or the property
// initial (initial/revert); ordinary tokens must pass normalize, and an
// invalid token leaves the field untouched.
func inheritedToken(
	value, parentValue, initial string, hasParent bool,
	normalize func(string) (string, bool),
) (string, bool) {
	if v, ok := cssWideValue(value, parentValue, initial, hasParent); ok {
		return v, true
	}

	return normalize(value)
}

// cssWideValue resolves the CSS-wide keywords shared by every property in
// this group: parent's value for inherit/unset (initial when there is no
// parent), property initial for initial/revert. Ordinary values return
// ("", false) so the caller can validate them.
func cssWideValue(value, parentValue, initial string, hasParent bool) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case inheritKeyword, cssKeywordUnset:
		if hasParent {
			return parentValue, true
		}

		return initial, true
	case cssKeywordInitial, cssKeywordRevert:
		return initial, true
	default:
		return "", false
	}
}

func normalizeColorAdjust(value string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case colorAdjustEconomy:
		return colorAdjustEconomy, true
	case colorAdjustExact:
		return colorAdjustExact, true
	default:
		return "", false
	}
}

func normalizeForcedColorAdjust(value string) (string, bool) {
	switch low := strings.ToLower(strings.TrimSpace(value)); low {
	case forcedColorAdjustAuto, forcedColorAdjustNone, forcedColorAdjustPreserveParentColor:
		return low, true
	default:
		return "", false
	}
}

// normalizeColorScheme accepts the keyword combinations this engine can use:
// normal, light, dark, light dark, dark light, only light, only dark. Custom
// idents are legal CSS but carry no print meaning here, so they are dropped.
func normalizeColorScheme(value string) (string, bool) {
	normalized := strings.Join(strings.Fields(strings.ToLower(value)), " ")

	switch normalized {
	case colorSchemeNormal, colorSchemeLight, colorSchemeDark,
		"light dark", "dark light", "only light", "only dark":
		return normalized, true
	default:
		return "", false
	}
}

// normalizeDynamicRangeLimit accepts the spec keywords plus the "constrained"
// spelling the catalog records, canonicalizing it to constrained-high.
func normalizeDynamicRangeLimit(value string) (string, bool) {
	switch low := strings.ToLower(strings.TrimSpace(value)); low {
	case dynamicRangeLimitNoLimit, dynamicRangeLimitStandard, dynamicRangeLimitHigh:
		return low, true
	case dynamicRangeLimitConstrainedHigh, "constrained":
		return dynamicRangeLimitConstrainedHigh, true
	default:
		return "", false
	}
}
