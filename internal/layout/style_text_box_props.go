package layout

import "strings"

// applyTextBoxProps owns text-box / text-box-trim / text-box-edge (CSS Inline 3).
func applyTextBoxProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext,
	_ *ResolvedStyle, _ bool,
) bool {
	val := strings.ToLower(strings.TrimSpace(value))
	if val == "" {
		return false
	}

	switch prop {
	case "text-box":
		applyTextBoxShorthand(style, val)
	case "text-box-trim":
		if trim, ok := parseTextBoxTrim(val); ok {
			style.TextBoxTrim = trim
		}
	case "text-box-edge":
		if over, under, ok := parseTextBoxEdge(val); ok {
			style.TextBoxEdgeOver = over
			style.TextBoxEdgeUnder = under
		}
	default:
		return false
	}

	return true
}

func applyTextBoxShorthand(style *ResolvedStyle, val string) {
	if val == contentNormal {
		style.TextBoxTrim = cssDisplayNone
		style.TextBoxEdgeOver = "auto"
		style.TextBoxEdgeUnder = "auto"

		return
	}

	parts := strings.Fields(val)
	if len(parts) == 0 {
		return
	}

	rest := parts

	if trim, ok := parseTextBoxTrim(parts[0]); ok {
		style.TextBoxTrim = trim
		rest = parts[1:]
	}

	if len(rest) == 0 {
		return
	}

	if over, under, ok := parseTextBoxEdge(strings.Join(rest, " ")); ok {
		style.TextBoxEdgeOver = over
		style.TextBoxEdgeUnder = under
	}
}

func parseTextBoxTrim(val string) (string, bool) {
	switch val {
	case cssDisplayNone, textBoxTrimStartKeyword, "trim-end", textBoxTrimBothKeyword:
		return val, true
	default:
		return "", false
	}
}

const textBoxEdgePartCount = 2

func parseTextBoxEdge(val string) (string, string, bool) {
	parts := strings.Fields(val)
	switch len(parts) {
	case 1:
		switch parts[0] {
		case "auto":
			return "auto", "auto", true
		case "text":
			return "text", "text", true
		default:
			return "", "", false
		}
	case textBoxEdgePartCount:
		if !isTextBoxEdgeOver(parts[0]) || !isTextBoxEdgeUnder(parts[1]) {
			return "", "", false
		}

		return parts[0], parts[1], true
	default:
		return "", "", false
	}
}

func isTextBoxEdgeOver(val string) bool {
	switch val {
	case "auto", "text", "cap", "ex", "ideographic", "ideographic-ink":
		return true
	default:
		return false
	}
}

func isTextBoxEdgeUnder(val string) bool {
	switch val {
	case "auto", "text", "alphabetic", "ideographic", "ideographic-ink":
		return true
	default:
		return false
	}
}
