package layout

import "strings"

// splitParenArgs splits s by sep at top-level paren depth, respecting single and
// double quotes. It trims each part and skips empty trailing parts, returning
// nil for empty input. This consolidates depth-only parsers (transform,
// grid_parse, gradient) and the quoted-aware background-image parser.
func splitParenArgs(s string, sep byte) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}

	var parts []string

	depth := 0
	inQuote := byte(0)
	start := 0

	for idx := 0; idx < len(s); idx++ {
		char := s[idx]
		if inQuote != 0 {
			if char == inQuote {
				inQuote = 0
			}

			continue
		}

		switch char {
		case '"', '\'':
			inQuote = char
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		default:
			if char == sep && depth == 0 {
				parts = append(parts, strings.TrimSpace(s[start:idx]))
				start = idx + 1
			}
		}
	}

	if start < len(s) {
		trimmed := strings.TrimSpace(s[start:])
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}

	return parts
}
