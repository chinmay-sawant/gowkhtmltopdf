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
			// display:-webkit-box maps to flex and forces nowrap; line-clamp
			// needs normal wrapping so the clamp can keep N lines.
			if style.WhiteSpace == cssWhiteSpaceNowrap {
				style.WhiteSpace = "normal"
			}
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
	case "empty-cells":
		if val == "show" || val == "hide" {
			style.EmptyCells = val
			return true
		}

	// Wave C: Fragmentation
	case "box-decoration-break", "-webkit-box-decoration-break":
		if val == "slice" || val == "clone" {
			style.BoxDecorationBreak = val
			return true
		}

	// Wave D: Blend Modes and Text Decoration
	case "mix-blend-mode":
		if mode, ok := normalizeBlendMode(val); ok {
			style.MixBlendMode = mode
			return true
		}
	case "background-blend-mode":
		modes := splitCommaLayers(value)
		if len(modes) == 0 {
			return false
		}
		normalized := make([]string, 0, len(modes))
		for index := range modes {
			mode, ok := normalizeBlendMode(modes[index])
			if !ok {
				return false
			}
			normalized = append(normalized, mode)
		}
		style.BackgroundBlendMode = strings.Join(normalized, ", ")
		return true
	case "isolation":
		if val == "auto" || val == "isolate" {
			style.Isolation = val
			return true
		}
	case "text-decoration-skip-ink":
		if val == "auto" || val == "none" || val == "all" {
			style.TextDecorationSkipInk = val
			return true
		}
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
	case "overflow-clip-margin", "overflow-clip-margin-inline",
		"overflow-clip-margin-inline-start", "overflow-clip-margin-inline-end",
		"overflow-clip-margin-block", "overflow-clip-margin-block-start",
		"overflow-clip-margin-block-end":
		vals := parseOverflowClipMarginLengths(val, fsize)
		if len(vals) == 0 {
			return true
		}
		// Shorthand and logical variants map to physical sides for horizontal-tb.
		switch prop {
		case "overflow-clip-margin":
			switch len(vals) {
			case 1:
				style.OverflowClipMarginTop = vals[0]
				style.OverflowClipMarginRight = vals[0]
				style.OverflowClipMarginBottom = vals[0]
				style.OverflowClipMarginLeft = vals[0]
			case 2:
				style.OverflowClipMarginTop = vals[0]
				style.OverflowClipMarginBottom = vals[0]
				style.OverflowClipMarginRight = vals[1]
				style.OverflowClipMarginLeft = vals[1]
			case 3:
				style.OverflowClipMarginTop = vals[0]
				style.OverflowClipMarginRight = vals[1]
				style.OverflowClipMarginLeft = vals[1]
				style.OverflowClipMarginBottom = vals[2]
			default:
				style.OverflowClipMarginTop = vals[0]
				style.OverflowClipMarginRight = vals[1]
				style.OverflowClipMarginBottom = vals[2]
				style.OverflowClipMarginLeft = vals[3]
			}
		case "overflow-clip-margin-inline":
			style.OverflowClipMarginLeft = vals[0]
			style.OverflowClipMarginRight = vals[0]
		case "overflow-clip-margin-inline-start":
			style.OverflowClipMarginLeft = vals[0]
		case "overflow-clip-margin-inline-end":
			style.OverflowClipMarginRight = vals[0]
		case "overflow-clip-margin-block":
			style.OverflowClipMarginTop = vals[0]
			style.OverflowClipMarginBottom = vals[0]
		case "overflow-clip-margin-block-start":
			style.OverflowClipMarginTop = vals[0]
		case "overflow-clip-margin-block-end":
			style.OverflowClipMarginBottom = vals[0]
		}
		return true
	}

	return false
}

func parseAdvancedLength(val string, fsize float64) float64 {
	if v, ok := parseAdvancedLengthOK(val, fsize); ok {
		return v
	}
	return 0
}

func parseAdvancedLengthOK(val string, fsize float64) (float64, bool) {
	if v, unit, ok := css.ParseLength(val); ok {
		switch unit {
		case "px":
			return v * 0.75, true
		case "pt":
			return v, true
		case "em", "rem":
			return v * fsize, true
		case "in":
			return v * 72.0, true
		case "mm":
			return v * 72.0 / 25.4, true
		case "cm":
			return v * 72.0 / 2.54, true
		default:
			return v, true
		}
	}
	return 0, false
}

func parseOverflowClipMarginLengths(val string, fsize float64) []float64 {
	toks := strings.Fields(val)
	out := make([]float64, 0, 4)
	for _, tok := range toks {
		low := strings.ToLower(tok)
		if low == "content-box" || low == "padding-box" || low == "border-box" ||
			low == "fill-box" || low == "stroke-box" || low == "view-box" {
			continue
		}
		if v, ok := parseAdvancedLengthOK(tok, fsize); ok {
			out = append(out, v)
			if len(out) >= 4 {
				break
			}
		}
	}
	return out
}

//nolint:unused
func parseAdvancedColor(val string) ([3]float64, bool) {
	r, g, b, _, ok := css.ParseColor(val)
	if !ok {
		return [3]float64{}, false
	}
	return [3]float64{float64(r) / 255.0, float64(g) / 255.0, float64(b) / 255.0}, true
}
