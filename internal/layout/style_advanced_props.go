//nolint:varnamelen,cyclop,gocyclo,funlen,mnd,goconst,wsl,nlreturn,gocognit,maintidx // advanced print CSS properties
package layout

import (
	"strconv"
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

// applyAdvancedProps handles GCPM bookmarks/paged media, line clamping, text truncation,
// fragmentation, blend modes, and font variation properties.
func applyAdvancedProps(style *ResolvedStyle, prop, value string, fsize float64) bool {
	val := strings.ToLower(strings.TrimSpace(value))
	if val == "" {
		return false
	}

	switch prop {
	// Wave B: Text Truncation, Clamping, Margins
	case "text-overflow":
		if val == "clip" || val == "ellipsis" {
			style.TextOverflow = val
			return true
		}
	case "line-clamp", "-webkit-line-clamp":
		if val == "none" {
			style.LineClamp = 0
			return true
		}
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			style.LineClamp = n
			return true
		}
	case "max-lines":
		if val == "none" {
			style.MaxLines = 0
			return true
		}
		if n, err := strconv.Atoi(val); err == nil && n > 0 {
			style.MaxLines = n
			return true
		}
	case "margin-trim":
		if val == "none" || val == "block" || val == "inline" ||
			val == "block-start" || val == "block-end" || val == "inline-start" || val == "inline-end" {
			style.MarginTrim = val
			return true
		}

	// Wave C: Fragmentation, Image/Color Metadata, Containment
	case "box-decoration-break", "-webkit-box-decoration-break":
		if val == "slice" || val == "clone" {
			style.BoxDecorationBreak = val
			return true
		}
	case "image-orientation":
		if val == "from-image" || val == "none" {
			style.ImageOrientation = val
			return true
		}
	case "image-resolution":
		if dppx := parseResolution(val); dppx > 0 {
			style.ImageResolution = dppx
			return true
		}
	case "object-view-box":
		style.ObjectViewBox = value
		return true
	case "print-color-adjust", "-webkit-print-color-adjust", "color-adjust":
		if val == "economy" || val == "exact" {
			style.PrintColorAdjust = val
			return true
		}
	case "forced-color-adjust":
		if val == "auto" || val == "none" {
			style.ForcedColorAdjust = val
			return true
		}
	case "color-scheme":
		style.ColorScheme = val
		return true
	case "dynamic-range-limit":
		if val == "standard" || val == "constrained-high" || val == "no-limit" {
			style.DynamicRangeLimit = val
			return true
		}
	case "contain-intrinsic-size":
		style.ContainIntrinsicSize = val
		return true
	case "contain-intrinsic-width":
		style.ContainIntrinsicWidth = parseAdvancedLength(val, fsize)
		return true
	case "contain-intrinsic-height":
		style.ContainIntrinsicHeight = parseAdvancedLength(val, fsize)
		return true
	case "contain-intrinsic-inline-size":
		style.ContainIntrinsicInlineSize = parseAdvancedLength(val, fsize)
		return true
	case "contain-intrinsic-block-size":
		style.ContainIntrinsicBlockSize = parseAdvancedLength(val, fsize)
		return true
	case "contain":
		style.Contain = val
		return true
	case "content-visibility":
		if val == "visible" || val == "auto" || val == "hidden" {
			style.ContentVisibility = val
			return true
		}

	// Wave D: Fonts, Blend Modes, Advanced Text Decoration
	case "font-variation-settings":
		style.FontVariationSettings = value
		return true
	case "font-optical-sizing":
		if val == "auto" || val == "none" {
			style.FontOpticalSizing = val
			return true
		}
	case "font-language-override":
		style.FontLanguageOverride = strings.Trim(value, "'\"")
		return true
	case "font-palette":
		style.FontPalette = val
		return true
	case "text-combine-upright", "-webkit-text-combine":
		style.TextCombineUpright = val
		return true
	case "text-orientation":
		if val == "mixed" || val == "upright" || val == "sideways" {
			style.TextOrientation = val
			return true
		}
	case "unicode-bidi":
		style.UnicodeBidi = val
		return true
	case "text-decoration-skip":
		style.TextDecorationSkip = val
		return true
	case "text-decoration-skip-ink":
		if val == "auto" || val == "none" || val == "all" {
			style.TextDecorationSkipInk = val
			return true
		}
	case "text-decoration-skip-box":
		style.TextDecorationSkipBox = val
		return true
	case "text-decoration-skip-self":
		style.TextDecorationSkipSelf = val
		return true
	case "text-decoration-skip-spaces":
		style.TextDecorationSkipSpaces = val
		return true
	case "text-decoration-inset":
		style.TextDecorationInset = val
		return true
	case "overflow-clip-margin-top":
		style.OverflowClipMarginTop = parseAdvancedLength(val, fsize)
		return true
	case "overflow-clip-margin-right":
		style.OverflowClipMarginRight = parseAdvancedLength(val, fsize)
		return true
	case "overflow-clip-margin-bottom":
		style.OverflowClipMarginBottom = parseAdvancedLength(val, fsize)
		return true
	case "overflow-clip-margin-left":
		style.OverflowClipMarginLeft = parseAdvancedLength(val, fsize)
		return true
	case "overflow-clip-margin-inline", "overflow-clip-margin-inline-start", "overflow-clip-margin-inline-end",
		"overflow-clip-margin-block", "overflow-clip-margin-block-start", "overflow-clip-margin-block-end":
		return false
	}

	return false
}

func parseAdvancedLength(val string, fsize float64) float64 {
	if v, unit, ok := css.ParseLength(val); ok {
		switch unit {
		case "px":
			return v * 0.75
		case "pt":
			return v
		case "em", "rem":
			return v * fsize
		case "in":
			return v * 72.0
		case "mm":
			return v * 72.0 / 25.4
		case "cm":
			return v * 72.0 / 2.54
		default:
			return v
		}
	}
	return 0
}

//nolint:unused
func parseAdvancedColor(val string) ([3]float64, bool) {
	r, g, b, _, ok := css.ParseColor(val)
	if !ok {
		return [3]float64{}, false
	}
	return [3]float64{float64(r) / 255.0, float64(g) / 255.0, float64(b) / 255.0}, true
}

func parseResolution(val string) float64 {
	val = strings.ToLower(strings.TrimSpace(val))
	if strings.HasSuffix(val, "dpi") {
		num := strings.TrimSuffix(val, "dpi")
		if n, err := strconv.ParseFloat(num, 64); err == nil && n > 0 {
			return n
		}
	}
	if strings.HasSuffix(val, "dppx") || strings.HasSuffix(val, "x") {
		num := strings.TrimSuffix(strings.TrimSuffix(val, "dppx"), "x")
		if n, err := strconv.ParseFloat(num, 64); err == nil && n > 0 {
			return n * 96.0
		}
	}
	return 0
}
