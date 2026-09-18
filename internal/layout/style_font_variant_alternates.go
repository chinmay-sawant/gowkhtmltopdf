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

	parts := make([]string, 0)
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

	name, rest, valid := consumeIdent(src)
	if !valid {
		return "", src, false
	}

	args, rest, valid := consumeParenArgs(rest)
	if !valid {
		return "", src, false
	}

	canon, valid := canonicalAlternateFunction(name, args)
	if !valid {
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

	for index := range len(src) {
		switch src[index] {
		case '(':
			depth++
		case ')':
			depth--
			if depth != 0 {
				continue
			}

			inner := strings.TrimSpace(src[1:index])
			if inner == "" {
				return nil, src, false
			}

			args := splitAlternateArgs(inner)
			if len(args) == 0 {
				return nil, src, false
			}

			return args, src[index+1:], true
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

//nolint:cyclop // CSS alternate-function grammar is intentionally explicit.
func canonicalAlternateFunction(name string, args []string) (string, bool) {
	switch name {
	case altStylistic, altSwash, altOrnaments, altAnnotation:
		if !validSingleAlternateArg(args) {
			return "", false
		}

		if number, ok := atoiBounded(args[0]); ok && number < 1 {
			return "", false
		}
	case altStyleset, altCharVariant:
		if !validMultipleAlternateArgs(args) {
			return "", false
		}

		maxNumber := maxStylesetIndex
		if name == altCharVariant {
			maxNumber = maxCharVariant
		}

		for _, arg := range args {
			if number, ok := atoiBounded(arg); ok && (number < 1 || number > maxNumber) {
				return "", false
			}
		}
	default:
		return "", false
	}

	var builder strings.Builder

	builder.WriteString(name)
	builder.WriteByte('(')
	builder.WriteString(strings.Join(args, ", "))
	builder.WriteByte(')')

	return builder.String(), true
}

func validSingleAlternateArg(args []string) bool {
	return len(args) == 1 && validAlternateArg(args[0])
}

func validMultipleAlternateArgs(args []string) bool {
	if len(args) == 0 {
		return false
	}

	for _, arg := range args {
		if !validAlternateArg(arg) {
			return false
		}
	}

	return true
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

	for i := range len(arg) {
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

//nolint:cyclop // OpenType tag mapping stays aligned with CSS function names.
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
	number := 1
	if v, ok := atoiBounded(arg); ok {
		number = v
	}

	if number < 1 || number > maxN {
		return "", false
	}

	numberText := strconv.Itoa(number)
	if len(numberText) == 1 {
		numberText = "0" + numberText
	}

	return prefix + numberText, true
}
