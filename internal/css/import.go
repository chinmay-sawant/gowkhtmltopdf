package css

import (
	"strings"
)

// parseImportRule consumes one @import from the start of src. A well-formed
// import is appended to str.Imports; a malformed one is skipped. Parse never
// fails solely because an import is invalid, and this step does not fetch.
func parseImportRule(src string, str *Stylesheet) (string, error) {
	if len(src) < len("@import") {
		return skipAtRule(src)
	}

	url, rest, ok := parseImportURL(src[len("@import"):])
	if !ok || url == "" {
		return skipAtRule(src)
	}

	rest = strings.TrimLeft(rest, " \t\r\n")
	if importHasBlock(rest) {
		return skipAtRule(src)
	}

	media, rest := splitImportPrelude(rest)
	str.Imports = append(str.Imports, ImportRule{URL: url, Media: media})

	return rest, nil
}

func importHasBlock(rest string) bool {
	semi := strings.IndexByte(rest, ';')
	brace := strings.IndexByte(rest, '{')

	return brace >= 0 && (semi < 0 || brace < semi)
}

func splitImportPrelude(rest string) (string, string) {
	semi := strings.IndexByte(rest, ';')
	if semi < 0 {
		return strings.TrimSpace(rest), ""
	}

	return strings.TrimSpace(rest[:semi]), rest[semi+1:]
}

// parseImportURL reads url("..."), url('...'), url(...), "..." or '...' after
// @import. ok is false when the prelude is not one of those forms.
func parseImportURL(src string) (string, string, bool) {
	src = strings.TrimLeft(src, " \t\r\n")
	if src == "" {
		return "", src, false
	}

	if src[0] == '"' || src[0] == '\'' {
		return parseQuotedImportURL(src)
	}

	if len(src) >= 4 && strings.EqualFold(src[:4], "url(") {
		return parseImportURLFunction(src)
	}

	return "", src, false
}

// parseImportURLFunction reads url(...) after @import. Quoted spans are
// skipped so a ')' inside the string is not treated as the closer.
func parseImportURLFunction(src string) (string, string, bool) {
	idx := len("url(")
	for idx < len(src) {
		switch src[idx] {
		case '"', '\'':
			idx = skipQuoted(src, idx, src[idx])

			continue
		case ')':
			inner := stripAttrQuotes(strings.TrimSpace(src[len("url("):idx]))
			inner = strings.TrimSpace(inner)

			if inner == "" {
				return "", src, false
			}

			return inner, src[idx+1:], true
		}

		idx++
	}

	return "", src, false
}

func parseQuotedImportURL(src string) (string, string, bool) {
	q := src[0]
	end := skipQuoted(src, 0, q)

	if end < minQuotedLen || src[end-1] != q {
		return "", src, false
	}

	url := src[1 : end-1]
	if url == "" {
		return "", src, false
	}

	return url, src[end:], true
}
