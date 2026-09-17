package layout

import (
	"strconv"
	"strings"
)

const (
	altHistoricalForms = "historical-forms"
	altStylistic       = "stylistic"
	altStyleset        = "styleset"
	altCharVariant     = "character-variant"
	altSwash           = "swash"
	altOrnaments       = "ornaments"
	altAnnotation      = "annotation"
	maxStylesetIndex   = 20
	maxCharVariant     = 99
)

// canonicalFontVariantAlternates accepts normal, historical-forms, and the
// CSS Fonts function forms (stylistic/styleset/swash). Named arguments
// default to feature value 1 because @font-feature-values is not parsed.
func canonicalFontVariantAlternates(raw string) (string, bool) {
	parts, ok := parseFontVariantAlternatesList(raw)
	if !ok {
		return "", false
	}

	return strings.Join(parts, " "), true
}

func parseFontVariantAlternatesList(raw string) ([]string, bool) {
	src := strings.ToLower(strings.TrimSpace(raw))
	if src == "" {
		return nil, false
	}

	if src == fontVariantNormal {
		return []string{fontVariantNormal}, true
	}

	parts := make([]string, 0, 4)
	rest := src

	for rest != "" {
		rest = strings.TrimSpace(rest)
		if rest == "" {
			break
		}

		part, next, ok := consumeAlternatePart(rest)
		if !ok {
			return nil, false
		}

		parts = append(parts, part)
		rest = next
	}

	if len(parts) == 0 {
		return nil, false
	}

	if len(parts) > 1 {
		for _, part := range parts {
			if part == fontVariantNormal {
				return nil, false
			}
		}
	}

	return parts, true
}

func consumeAlternatePart(src string) (string, string, bool) {
	if strings.HasPrefix(src, altHistoricalForms) {
		end := len(altHistoricalForms)
		if end < len(src) && isIdentCont(src[end]) {
			return "", src, false
		}

		return altHistoricalForms, src[end:], true
	}

	name, rest, ok := consumeIdent(src)
	if !ok {
		return "", src, false
	}

	args, rest, ok := consumeParenArgs(rest)
	if !ok {
		return "", src, false
	}

	canon, ok := canonicalAlternateFunction(name, args)
	if !ok {
		return "", src, false
	}

	return canon, rest, true
}

func consumeIdent(src string) (string, string, bool) {
	if src == "" || !isIdentStart(src[0]) {
		return "", src, false
	}

	end := 1
	for end < len(src) && isIdentCont(src[end]) {
		end++
	}

	return src[:end], src[end:], true
}

func consumeParenArgs(src string) ([]string, string, bool) {
	src = strings.TrimSpace(src)
	if src == "" || src[0] != '(' {
		return nil, src, false
	}

	depth := 0

	for i := 0; i < len(src); i++ {
		switch src[i] {
		case '(':
			depth++
		case ')':
			depth--
			if depth != 0 {
				continue
			}

			inner := strings.TrimSpace(src[1:i])
			if inner == "" {
				return nil, src, false
			}

			args := splitAlternateArgs(inner)
			if len(args) == 0 {
				return nil, src, false
			}

			return args, src[i+1:], true
		}
	}

	return nil, src, false
}

func splitAlternateArgs(inner string) []string {
	raw := strings.Split(inner, ",")
	args := make([]string, 0, len(raw))

	for _, part := range raw {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil
		}

		args = append(args, part)
	}

	return args
}

func canonicalAlternateFunction(name string, args []string) (string, bool) {
	switch name {
	case altStylistic, altSwash, altOrnaments, altAnnotation:
		if len(args) != 1 || !validAlternateArg(args[0]) {
			return "", false
		}

		if n, ok := atoiBounded(args[0]); ok && n < 1 {
			return "", false
		}
	case altStyleset, altCharVariant:
		if len(args) == 0 {
			return "", false
		}

		maxN := maxStylesetIndex
		if name == altCharVariant {
			maxN = maxCharVariant
		}

		for _, arg := range args {
			if !validAlternateArg(arg) {
				return "", false
			}

			if n, ok := atoiBounded(arg); ok && (n < 1 || n > maxN) {
				return "", false
			}
		}
	default:
		return "", false
	}

	var b strings.Builder

	b.WriteString(name)
	b.WriteByte('(')
	b.WriteString(strings.Join(args, ", "))
	b.WriteByte(')')

	return b.String(), true
}

func validAlternateArg(arg string) bool {
	if _, ok := atoiBounded(arg); ok {
		return true
	}

	ident, rest, ok := consumeIdent(arg)

	return ok && rest == "" && ident != ""
}

func atoiBounded(arg string) (int, bool) {
	if arg == "" {
		return 0, false
	}

	for i := 0; i < len(arg); i++ {
		if arg[i] < '0' || arg[i] > '9' {
			return 0, false
		}
	}

	n, err := strconv.Atoi(arg)
	if err != nil || n < 0 || n > maxCharVariant {
		return 0, false
	}

	return n, true
}

func appendAlternateOTTags(value string, put func(string, uint32)) {
	if value == "" || value == fontVariantNormal {
		return
	}

	parts, ok := parseFontVariantAlternatesList(value)
	if !ok {
		return
	}

	for _, part := range parts {
		if part == altHistoricalForms {
			put("hist", 1)

			continue
		}

		name, args, ok := splitStoredFunction(part)
		if !ok {
			continue
		}

		switch name {
		case altStylistic:
			put("salt", alternateArgValue(args[0]))
		case altSwash:
			put("swsh", alternateArgValue(args[0]))
		case altOrnaments:
			put("ornm", alternateArgValue(args[0]))
		case altAnnotation:
			put("nalt", alternateArgValue(args[0]))
		case altStyleset:
			for _, arg := range args {
				if tag, ok := numberedFeatureTag("ss", arg, maxStylesetIndex); ok {
					put(tag, 1)
				}
			}
		case altCharVariant:
			for _, arg := range args {
				if tag, ok := numberedFeatureTag("cv", arg, maxCharVariant); ok {
					put(tag, 1)
				}
			}
		}
	}
}

func splitStoredFunction(part string) (string, []string, bool) {
	open := strings.IndexByte(part, '(')
	if open <= 0 || part[len(part)-1] != ')' {
		return "", nil, false
	}

	args := splitAlternateArgs(part[open+1 : len(part)-1])
	if len(args) == 0 {
		return "", nil, false
	}

	return part[:open], args, true
}

func alternateArgValue(arg string) uint32 {
	if n, ok := atoiBounded(arg); ok && n >= 1 {
		return uint32(n) //nolint:gosec // n bounded by maxCharVariant
	}

	return 1
}

func numberedFeatureTag(prefix, arg string, maxN int) (string, bool) {
	n := 1
	if v, ok := atoiBounded(arg); ok {
		n = v
	}

	if n < 1 || n > maxN {
		return "", false
	}

	if n < 10 {
		return prefix + "0" + strconv.Itoa(n), true
	}

	return prefix + strconv.Itoa(n), true
}
