//nolint:cyclop,mnd,varnamelen // font variation property parsers
package layout

import (
	"strconv"
	"strings"
)

// CSS font variation and shaping support group: font-language-override,
// font-optical-sizing, font-palette, and font-variation-settings.
//
// Supported subset:
//   - font-language-override: normal | <string>. The stored value is "normal"
//     or the unquoted OpenType language system tag ("TRK", "dflt"). The tag
//     subset is 1-4 ASCII letters or digits; empty strings, escapes, and
//     BCP 47 forms longer than four characters are dropped.
//   - font-optical-sizing: auto | none, lowercased.
//   - font-palette: normal | light | dark | <dashed-ident>. Keywords are
//     lowercased, identifiers keep their case. palette-mix() is not
//     supported and is dropped.
//   - font-variation-settings: normal | [ <string tag> <number> ]#. Tags must
//     be quoted and exactly four ASCII letters or digits. The stored value is
//     canonical (`"wght" 700, "wdth" 87.5`); a repeated axis keeps the last
//     value, matching CSS variation precedence.
//
// The face-resolution consumer lives in layout.go (resolveFontVariants). The
// language override reaches the shaper as OpText.TextLanguage and
// pdf.ShapeTextFontWithFeaturesLanguage.
const (
	fontVariantNormal     = "normal"
	fontOpticalAuto       = "auto"
	fontOpticalNone       = "none"
	fontPaletteLight      = "light"
	fontPaletteDark       = "dark"
	fontAxisTagLength     = 4
	fontLangTagMaxLen     = 4
	fontDashedIdentPrefix = "--"
)

// applyFontVariantProps owns font-language-override, font-optical-sizing,
// font-palette, and font-variation-settings. Invalid values leave the current
// (inherited or initial) field untouched, matching a dropped declaration.
func applyFontVariantProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext,
	_ *ResolvedStyle, _ bool,
) bool {
	switch prop {
	case "font-language-override":
		if tag, ok := normalizeFontLanguageOverride(value); ok {
			style.FontLanguageOverride = tag
		}
	case "font-optical-sizing":
		switch strings.ToLower(strings.TrimSpace(value)) {
		case fontOpticalAuto:
			style.FontOpticalSizing = fontOpticalAuto
		case fontOpticalNone:
			style.FontOpticalSizing = fontOpticalNone
		}
	case "font-palette":
		if palette, ok := normalizeFontPalette(value); ok {
			style.FontPalette = palette
		}
	case "font-variation-settings":
		if settings, ok := parseFontVariationSettings(value); ok {
			style.FontVariationSettings = settings
		}
	default:
		return false
	}

	return true
}

// normalizeFontLanguageOverride returns "normal" or the unquoted language
// tag. The accepted tag subset is 1-4 ASCII letters or digits, the OpenType
// language system tag shape used by the shaping language override.
func normalizeFontLanguageOverride(raw string) (string, bool) {
	value := strings.TrimSpace(raw)

	if strings.EqualFold(value, fontVariantNormal) {
		return fontVariantNormal, true
	}

	tag, ok := unquoteCSSString(value)
	if !ok || len(tag) == 0 || len(tag) > fontLangTagMaxLen {
		return "", false
	}

	for i := range len(tag) {
		if !isASCIIAlnum(tag[i]) {
			return "", false
		}
	}

	return tag, true
}

// normalizeFontPalette returns normal | light | dark | <dashed-ident>.
// <dashed-ident> is case-sensitive, so only the keywords are lowercased.
func normalizeFontPalette(raw string) (string, bool) {
	value := strings.TrimSpace(raw)

	switch strings.ToLower(value) {
	case fontVariantNormal:
		return fontVariantNormal, true
	case fontPaletteLight:
		return fontPaletteLight, true
	case fontPaletteDark:
		return fontPaletteDark, true
	}

	if len(value) <= len(fontDashedIdentPrefix) || !strings.HasPrefix(value, fontDashedIdentPrefix) {
		return "", false
	}

	rest := value[len(fontDashedIdentPrefix):]
	if rest[0] >= '0' && rest[0] <= '9' {
		return "", false
	}

	for i := range len(rest) {
		if c := rest[i]; c != '-' && c != '_' && !isASCIIAlnum(c) {
			return "", false
		}
	}

	return value, true
}

// parseFontVariationSettings validates and canonicalizes the axis list.
// A repeated axis keeps its position and last value.
func parseFontVariationSettings(raw string) (string, bool) {
	value := strings.TrimSpace(raw)

	if strings.EqualFold(value, fontVariantNormal) {
		return fontVariantNormal, true
	}

	if value == "" {
		return "", false
	}

	parts := splitFontVariationList(value)

	order := make([]string, 0, len(parts))
	tags := make(map[string]string, len(parts))

	for _, part := range parts {
		tag, number, ok := parseFontVariationPair(part)
		if !ok {
			return "", false
		}

		if _, seen := tags[tag]; !seen {
			order = append(order, tag)
		}

		tags[tag] = number
	}

	var out strings.Builder

	for i, tag := range order {
		if i > 0 {
			out.WriteString(", ")
		}

		out.WriteString(`"`)
		out.WriteString(tag)
		out.WriteString(`" `)
		out.WriteString(tags[tag])
	}

	return out.String(), true
}

// parseFontVariationPair parses one `"tag" number` pair and returns the tag
// plus the canonical number text.
func parseFontVariationPair(part string) (string, string, bool) {
	if len(part) < 2 || (part[0] != '"' && part[0] != '\'') {
		return "", "", false
	}

	quote := part[0]

	end := strings.IndexByte(part[1:], quote)
	if end < 0 {
		return "", "", false
	}

	tag := part[1 : 1+end]
	if len(tag) != fontAxisTagLength {
		return "", "", false
	}

	for i := range len(tag) {
		if !isASCIIAlnum(tag[i]) {
			return "", "", false
		}
	}

	rest := strings.TrimSpace(part[2+end:])
	if rest == "" {
		return "", "", false
	}

	number, ok := parseFontAxisNumber(rest)
	if !ok {
		return "", "", false
	}

	return tag, number, true
}

// parseFontAxisNumber validates one CSS <number> (sign, decimal point, and
// exponent allowed, matching the CSS Syntax number token) and returns its
// plain decimal form. Hex floats, NaN, infinity, and Go digit separators are
// rejected.
func parseFontAxisNumber(token string) (string, bool) {
	idx, ok := scanFontAxisMantissa(token)
	if !ok {
		return "", false
	}

	if idx < len(token) && (token[idx] == 'e' || token[idx] == 'E') {
		idx, ok = scanFontAxisExponent(token, idx)
		if !ok {
			return "", false
		}
	}

	if idx != len(token) {
		return "", false
	}

	value, err := strconv.ParseFloat(token, 64)
	if err != nil {
		return "", false
	}

	return strconv.FormatFloat(value, 'f', -1, 64), true
}

// scanFontAxisMantissa consumes the sign, integer digits, and optional
// fraction of a CSS number and returns the index after the mantissa.
func scanFontAxisMantissa(token string) (int, bool) {
	idx := 0

	if idx < len(token) && (token[idx] == '+' || token[idx] == '-') {
		idx++
	}

	digits := 0

	for idx < len(token) && isASCIIDigit(token[idx]) {
		idx++
		digits++
	}

	if idx < len(token) && token[idx] == '.' {
		idx++

		fractionDigits := 0

		for idx < len(token) && isASCIIDigit(token[idx]) {
			idx++
			fractionDigits++
		}

		if fractionDigits == 0 {
			return 0, false
		}

		digits += fractionDigits
	}

	if digits == 0 {
		return 0, false
	}

	return idx, true
}

// scanFontAxisExponent consumes the exponent marker at idx followed by an
// optional sign and at least one digit.
func scanFontAxisExponent(token string, idx int) (int, bool) {
	idx++

	if idx < len(token) && (token[idx] == '+' || token[idx] == '-') {
		idx++
	}

	digits := 0

	for idx < len(token) && isASCIIDigit(token[idx]) {
		idx++
		digits++
	}

	if digits == 0 {
		return 0, false
	}

	return idx, true
}

// splitFontVariationList splits a comma-separated axis list without splitting
// commas inside quoted tags. Empty items are preserved so the parser can
// reject a trailing or doubled comma.
func splitFontVariationList(value string) []string {
	parts := make([]string, 0, strings.Count(value, ",")+1)
	start := 0
	inQuote := byte(0)

	for i := range len(value) {
		c := value[i]

		switch {
		case inQuote != 0:
			if c == inQuote {
				inQuote = 0
			}
		case c == '"' || c == '\'':
			inQuote = c
		case c == ',':
			parts = append(parts, strings.TrimSpace(value[start:i]))
			start = i + 1
		}
	}

	return append(parts, strings.TrimSpace(value[start:]))
}

// unquoteCSSString strips one matching pair of single or double quotes. The
// inner text must not contain the quote or a backslash escape.
func unquoteCSSString(value string) (string, bool) {
	if len(value) < 2 {
		return "", false
	}

	quote := value[0]
	if quote != '"' && quote != '\'' {
		return "", false
	}

	if value[len(value)-1] != quote {
		return "", false
	}

	inner := value[1 : len(value)-1]
	if strings.IndexByte(inner, quote) >= 0 || strings.IndexByte(inner, '\\') >= 0 {
		return "", false
	}

	return inner, true
}

func isASCIIAlnum(c byte) bool {
	return isASCIIDigit(c) || (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isASCIIDigit(c byte) bool {
	return c >= '0' && c <= '9'
}

// fontShapingLanguage returns the language tag shaping must use for sty.
// font-language-override wins over the document language for the element. The
// document language is not available in the layout engine today (convert
// reads <html lang> only for the PDF/UA /Lang tag), so an unset override
// yields "".
//
// internal/layout attaches the result to each OpText and the shaper consumes
// it in internal/pdf/shape_gotext.go (shapingInput).
func fontShapingLanguage(sty *ResolvedStyle) string {
	if sty == nil || sty.FontLanguageOverride == fontVariantNormal {
		return ""
	}

	return sty.FontLanguageOverride
}
