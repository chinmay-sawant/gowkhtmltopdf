package css

import "strings"

// maxUnicodeCodePoint is the highest code point a unicode-range may name.
const maxUnicodeCodePoint = 0x10FFFF

// hexShiftBits is the bit width of one hex digit; hexWildMask is the value of
// one trailing '?' wildcard nibble.
const (
	hexShiftBits = 4
	hexWildMask  = 0xF
)

// cssWeightNormal and cssWeightBold are the font-weight keyword values.
const (
	cssWeightNormal = 400
	cssWeightBold   = 700
)

// UnicodeRange is one inclusive code point span from a unicode-range
// descriptor, for example U+0100-02AF.
type UnicodeRange struct {
	Lo rune
	Hi rune
}

// Covers reports whether r falls inside the span.
func (u UnicodeRange) Covers(r rune) bool {
	return r >= u.Lo && r <= u.Hi
}

// ParseUnicodeRanges parses an @font-face unicode-range descriptor. Tokens are
// comma-separated U+XXXX forms: one code point (U+0304), a span
// (U+0100-02AF), or a trailing-? wildcard span (U+4??). Unrecognized,
// out-of-range, and reversed tokens are skipped. A descriptor with no valid
// tokens returns nil, which callers read as "no restriction", matching the CSS
// descriptor default of U+0-10FFFF.
func ParseUnicodeRanges(spec string) []UnicodeRange {
	if strings.TrimSpace(spec) == "" {
		return nil
	}

	var out []UnicodeRange

	for _, token := range strings.Split(spec, ",") {
		if span, ok := parseUnicodeRangeToken(token); ok {
			out = append(out, span)
		}
	}

	return out
}

// parseUnicodeRangeToken parses one "u+..." token.
func parseUnicodeRangeToken(token string) (UnicodeRange, bool) {
	token = strings.TrimSpace(token)
	if len(token) < 3 || (token[0] != 'u' && token[0] != 'U') || token[1] != '+' {
		return UnicodeRange{}, false //nolint:exhaustruct // failed parse
	}

	lowerPart, upperPart, hasSpan := strings.Cut(token[2:], "-")

	lower, upper, ok := hexSpan(lowerPart)
	if !ok {
		return UnicodeRange{}, false //nolint:exhaustruct // failed parse
	}

	if hasSpan {
		_, upperEnd, okEnd := hexSpan(upperPart)
		if !okEnd {
			return UnicodeRange{}, false //nolint:exhaustruct // failed parse
		}

		upper = upperEnd
	}

	if lower > upper {
		return UnicodeRange{}, false //nolint:exhaustruct // failed parse
	}

	return UnicodeRange{Lo: lower, Hi: upper}, true
}

// hexSpan parses 1-6 hex digits with optional trailing '?' wildcards and
// returns the inclusive span they name. Odd-length spans are legal per the CSS
// grammar (U+4?? is U+0400-04FF), so digits are read left to right and each
// wildcard widens the high end.
func hexSpan(part string) (rune, rune, bool) {
	if part == "" || len(part) > 6 {
		return 0, 0, false
	}

	var lower, upper rune

	wildcard := false

	for i := range len(part) {
		nibble := part[i]

		switch {
		case nibble == '?':
			wildcard = true
			lower <<= hexShiftBits
			upper = upper<<hexShiftBits | hexWildMask
		case !wildcard && isHexChar(nibble):
			digit := rune(hexVal(nibble))
			lower = lower<<hexShiftBits | digit
			upper = upper<<hexShiftBits | digit
		default:
			return 0, 0, false
		}
	}

	if upper > maxUnicodeCodePoint {
		return 0, 0, false
	}

	return lower, upper, true
}

// fontWeightDescriptor parses an @font-face font-weight descriptor into a CSS
// weight. bolder and lighter are relative to a parent and have no meaning on a
// face, so they report ok=false; values outside 1..1000 do too.
func fontWeightDescriptor(val string) (int, bool) {
	switch strings.ToLower(strings.TrimSpace(val)) {
	case "normal":
		return cssWeightNormal, true
	case "bold":
		return cssWeightBold, true
	}

	n, ok := ParseNumber(val)
	if !ok || n < 1 || n > 1000 {
		return 0, false
	}

	return int(n), true
}

// italicDescriptor parses an @font-face font-style descriptor. ok=false for an
// absent or invalid descriptor; "normal" reports (false, true) so it can
// override a file that declares an italic macStyle. `oblique 10deg` counts as
// oblique: the writer cannot synthesize an angle, so the italic flag carries
// the selection.
func italicDescriptor(val string) (bool, bool) {
	low := strings.ToLower(strings.TrimSpace(val))

	switch {
	case low == "normal":
		return false, true
	case low == "italic":
		return true, true
	case low == "oblique" || strings.HasPrefix(low, "oblique "):
		return true, true
	default:
		return false, false
	}
}
