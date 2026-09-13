//nolint:cyclop,varnamelen // text support property parsers
package layout

import (
	"strconv"
	"strings"
)

// applyTextSupportProps is the re-added advanced text support group. It owns
// text-combine-upright, text-decoration-inset, text-decoration-skip,
// text-decoration-skip-box, text-decoration-skip-self,
// text-decoration-skip-spaces, text-orientation, and unicode-bidi.
//
// The group consumes every declaration for those names; invalid values leave
// the current (inherited or initial) field untouched, matching a dropped
// declaration. Consumers live in inline_collect.go and inline_paint.go:
//   - UnicodeBidi scopes and reverses inline runs at collection time.
//   - TextOrientation / TextCombineUpright pick the text run rotation.
//   - TextDecorationInset / the four TextDecorationSkip* longhands adjust
//     decoration geometry in paintDecoration.
func applyTextSupportProps(
	style *ResolvedStyle, prop, value string, fsize float64, _ *styleContext,
	_ *ResolvedStyle, _ bool,
) bool {
	val := strings.ToLower(strings.TrimSpace(value))
	if val == "" {
		return false
	}

	switch prop {
	case "unicode-bidi":
		if isUnicodeBidiValue(val) {
			style.UnicodeBidi = val
		}
	case "text-orientation":
		if val == textOrientationMixed || val == textOrientationUpright || val == textOrientationSideways {
			style.TextOrientation = val
		}
	case "text-combine-upright":
		if combined, ok := parseTextCombineUpright(val); ok {
			style.TextCombineUpright = combined
		}
	case "text-decoration-inset":
		if off, ok := parseTextDecorationInset(val, fsize); ok {
			style.TextDecorationInset = off
		}
	case "text-decoration-skip":
		applyTextDecorationSkipShorthand(style, val)
	case "text-decoration-skip-box":
		if val == cssDisplayNone || val == columnSpanAll {
			style.TextDecorationSkipBox = val
		}
	case "text-decoration-skip-self":
		if self, ok := parseTextDecorationSkipSelf(val); ok {
			style.TextDecorationSkipSelf = self
		}
	case "text-decoration-skip-spaces":
		if spaces, ok := parseTextDecorationSkipSpaces(val); ok {
			style.TextDecorationSkipSpaces = spaces
		}
	default:
		return false
	}

	return true
}

const (
	textOrientationMixed    = "mixed"
	textOrientationUpright  = "upright"
	textOrientationSideways = "sideways"

	// unicodeBidiEmbed and unicodeBidiPlaintext are shared by the value
	// validator and the inline collector's scope check.
	unicodeBidiEmbed           = "embed"
	unicodeBidiIsolate         = "isolate"
	unicodeBidiOverride        = "bidi-override"
	unicodeBidiIsolateOverride = "isolate-override"
	unicodeBidiPlaintext       = "plaintext"

	// textDecorationSkipNoSkip and textDecorationSkipStartEnd are the
	// canonical skip values shared across the style and paint passes.
	textDecorationSkipNoSkip   = "no-skip"
	textDecorationSkipStartEnd = "start end"
)

// isUnicodeBidiValue reports the six legal unicode-bidi keywords.
func isUnicodeBidiValue(val string) bool {
	switch val {
	case "normal", unicodeBidiEmbed, unicodeBidiIsolate,
		unicodeBidiOverride, unicodeBidiIsolateOverride, unicodeBidiPlaintext:
		return true
	default:
		return false
	}
}

// parseTextCombineUpright normalizes none | all | digits <integer >= 1>.
func parseTextCombineUpright(val string) (string, bool) {
	if val == cssDisplayNone || val == columnSpanAll {
		return val, true
	}

	parts := strings.Fields(val)
	if len(parts) != two || parts[0] != "digits" {
		return "", false
	}

	n, err := strconv.Atoi(parts[1])
	if err != nil || n < 1 {
		return "", false
	}

	return "digits " + strconv.Itoa(n), true
}

// parseTextDecorationInset accepts auto or 1-2 lengths and returns the first
// endpoint in points, matching the single TextDecorationInset field. One
// stored value applies to both endpoints. Percentages are rejected because
// the style pass has no final containing size and the field has no basis to
// resolve them against.
func parseTextDecorationInset(val string, fsize float64) (float64, bool) {
	if val == overflowAuto {
		return 0, true
	}

	toks := strings.Fields(val)
	if len(toks) < 1 || len(toks) > two {
		return 0, false
	}

	first := 0.0

	for idx, tok := range toks {
		if strings.Contains(tok, "%") {
			return 0, false
		}

		pt, ok := plainLength(tok, fsize, 0)
		if !ok {
			return 0, false
		}

		if idx == 0 {
			first = pt
		}
	}

	return first, true
}

// applyTextDecorationSkipShorthand expands the shorthand onto its longhands:
// none | auto plus the CSS Text Decoration 3 legacy keywords objects, spaces,
// ink, edges, and box-decoration. Each legacy value sets the longhand it names
// and resets the rest to their initial values. none turns every skip off
// (skip-self has no none keyword, its off value is no-skip) and auto resets
// every longhand to its initial value. TextDecorationSkipInk is written too so
// the shorthand resets that longhand even though its apply arm lives in
// applyAdvancedProps.
func applyTextDecorationSkipShorthand(style *ResolvedStyle, val string) bool {
	switch val {
	case cssDisplayNone:
		style.TextDecorationSkip = cssDisplayNone
		style.TextDecorationSkipSelf = textDecorationSkipNoSkip
		style.TextDecorationSkipBox = cssDisplayNone
		style.TextDecorationSkipSpaces = cssDisplayNone
		style.TextDecorationSkipInk = cssDisplayNone
	case overflowAuto:
		setTextDecorationSkipInitial(style)
		style.TextDecorationSkip = overflowAuto
	case "objects", "box-decoration":
		setTextDecorationSkipInitial(style)
		style.TextDecorationSkip = val
		style.TextDecorationSkipBox = columnSpanAll
	case "spaces":
		setTextDecorationSkipInitial(style)
		style.TextDecorationSkip = val
		style.TextDecorationSkipSpaces = columnSpanAll
	case "edges":
		setTextDecorationSkipInitial(style)
		style.TextDecorationSkip = val
		style.TextDecorationSkipSpaces = textDecorationSkipStartEnd
	case "ink":
		setTextDecorationSkipInitial(style)
		style.TextDecorationSkip = val
		style.TextDecorationSkipInk = overflowAuto
	default:
		return false
	}

	return true
}

// setTextDecorationSkipInitial resets the four skip longhands to their CSS
// Text Decoration initial values: skip-self auto, skip-box none, skip-spaces
// start end, skip-ink auto. The raw TextDecorationSkip field is the caller's
// to set.
func setTextDecorationSkipInitial(style *ResolvedStyle) {
	style.TextDecorationSkipSelf = overflowAuto
	style.TextDecorationSkipBox = cssDisplayNone
	style.TextDecorationSkipSpaces = textDecorationSkipStartEnd
	style.TextDecorationSkipInk = overflowAuto
}

// parseTextDecorationSkipSelf accepts auto | skip-all | no-skip or any
// combination of the three per-line keywords and returns the canonical order.
func parseTextDecorationSkipSelf(val string) (string, bool) {
	toks := strings.Fields(val)
	if len(toks) == 0 {
		return "", false
	}

	if len(toks) == 1 {
		switch toks[0] {
		case overflowAuto, "skip-all", textDecorationSkipNoSkip:
			return toks[0], true
		}
	}

	var underline, overline, lineThrough bool

	for _, tok := range toks {
		switch tok {
		case "skip-underline":
			if underline {
				return "", false
			}

			underline = true
		case "skip-overline":
			if overline {
				return "", false
			}

			overline = true
		case "skip-line-through":
			if lineThrough {
				return "", false
			}

			lineThrough = true
		default:
			return "", false
		}
	}

	parts := make([]string, 0, three)

	if underline {
		parts = append(parts, "skip-underline")
	}

	if overline {
		parts = append(parts, "skip-overline")
	}

	if lineThrough {
		parts = append(parts, "skip-line-through")
	}

	if len(parts) == 0 {
		return "", false
	}

	return strings.Join(parts, " "), true
}

// parseTextDecorationSkipSpaces accepts none | all | [ start || end ] and
// returns the canonical order.
func parseTextDecorationSkipSpaces(val string) (string, bool) {
	toks := strings.Fields(val)
	if len(toks) == 0 {
		return "", false
	}

	if len(toks) == 1 && (toks[0] == cssDisplayNone || toks[0] == columnSpanAll) {
		return toks[0], true
	}

	var start, end bool

	for _, tok := range toks {
		switch tok {
		case fxStart:
			if start {
				return "", false
			}

			start = true
		case fxEnd:
			if end {
				return "", false
			}

			end = true
		default:
			return "", false
		}
	}

	switch {
	case start && end:
		return textDecorationSkipStartEnd, true
	case start:
		return fxStart, true
	case end:
		return fxEnd, true
	default:
		return "", false
	}
}
