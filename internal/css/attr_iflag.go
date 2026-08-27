package css

import "strings"

// splitAttrIFlag strips a trailing Selectors 4 ASCII i / I or s / S flag from
// the attribute-selector value. The s flag is the default exact comparison,
// so both flags return the same ignoreCase=false value.
func splitAttrIFlag(raw string) (string, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return raw, false
	}

	if raw[0] == '"' || raw[0] == '\'' {
		return splitQuotedAttrIFlag(raw)
	}

	return splitIdentAttrIFlag(raw)
}

func splitQuotedAttrIFlag(raw string) (string, bool) {
	quote := raw[0]
	closeAt := strings.LastIndexByte(raw, quote)

	if closeAt <= 0 {
		return raw, false
	}

	rest := strings.TrimSpace(raw[closeAt+1:])
	if isAttrIgnoreCaseFlag(rest) {
		return raw[:closeAt+1], true
	}

	if !isAttrCaseSensitiveFlag(rest) {
		return raw, false
	}

	return raw[:closeAt+1], false
}

func splitIdentAttrIFlag(raw string) (string, bool) {
	end := len(raw) - 1
	for end >= 0 && !isClassSpace(raw[end]) {
		end--
	}

	if end < 0 {
		return raw, false
	}

	flag := raw[end+1:]
	val := strings.TrimSpace(raw[:end])

	if val == "" {
		return raw, false
	}

	if isAttrIgnoreCaseFlag(flag) {
		return val, true
	}

	if isAttrCaseSensitiveFlag(flag) {
		return val, false
	}

	return raw, false
}

func isAttrIgnoreCaseFlag(s string) bool {
	return len(s) == 1 && (s[0] == 'i' || s[0] == 'I')
}

func isAttrCaseSensitiveFlag(s string) bool {
	return len(s) == 1 && (s[0] == 's' || s[0] == 'S')
}
