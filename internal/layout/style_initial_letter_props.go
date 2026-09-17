package layout

import (
	"strconv"
	"strings"
)

// CSS Inline 3 initial-letter family. :first-letter is rejected by the CSS
// selector parser, so authors must put initial-letter on a real leading
// element (typically a <span> wrapping the first letter or word).
const (
	initialLetterNormal     = "normal"
	initialLetterDrop       = "drop"
	initialLetterRaise      = "raise"
	initialLetterAlignAlpha = "alphabetic"
	initialLetterWrapNone   = "none"
	initialLetterWrapFirst  = "first"
	initialLetterWrapAll    = "all"
	initialLetterWrapGrid   = "grid"
)

// applyInitialLetterProps owns initial-letter, initial-letter-align, and
// initial-letter-wrap. The layout consumer lives in inline_initial_letter.go.
func applyInitialLetterProps(
	style *ResolvedStyle, prop, value string, _ float64, _ *styleContext,
	_ *ResolvedStyle, _ bool,
) bool {
	switch prop {
	case "initial-letter":
		return applyInitialLetterValue(style, value)
	case "initial-letter-align":
		return applyInitialLetterAlignValue(style, value)
	case "initial-letter-wrap":
		return applyInitialLetterWrapValue(style, value)
	default:
		return false
	}
}

func applyInitialLetterValue(style *ResolvedStyle, value string) bool {
	val := strings.ToLower(strings.TrimSpace(value))
	if val == "" {
		return false
	}

	if val == initialLetterNormal {
		style.InitialLetterSize = 0
		style.InitialLetterSink = 0

		return true
	}

	tokens := strings.Fields(val)
	if len(tokens) == 0 || len(tokens) > 3 {
		return false
	}

	size, err := strconv.ParseFloat(tokens[0], 64)
	if err != nil || size < 1 {
		return false
	}

	sink := int(size + 0.5) // drop default: sink == size
	raise := false
	sawSink := false

	for _, tok := range tokens[1:] {
		switch tok {
		case initialLetterDrop:
			raise = false
		case initialLetterRaise:
			raise = true
		default:
			n, nErr := strconv.Atoi(tok)
			if nErr != nil || n < 1 || sawSink {
				return false
			}

			sink = n
			sawSink = true
		}
	}

	if raise && !sawSink {
		sink = 1
	}

	style.InitialLetterSize = size
	style.InitialLetterSink = sink

	return true
}

func applyInitialLetterAlignValue(style *ResolvedStyle, value string) bool {
	val := strings.ToLower(strings.TrimSpace(value))
	// border-box optional prefix is accepted then ignored (lite).
	val = strings.TrimPrefix(val, "border-box")
	val = strings.TrimSpace(val)

	if val == "" {
		val = initialLetterAlignAlpha
	}

	switch val {
	case initialLetterAlignAlpha, "ideographic", "hanging", "leading":
		style.InitialLetterAlign = val

		return true
	default:
		return false
	}
}

func applyInitialLetterWrapValue(style *ResolvedStyle, value string) bool {
	val := strings.ToLower(strings.TrimSpace(value))
	switch val {
	case initialLetterWrapNone, initialLetterWrapFirst, initialLetterWrapAll, initialLetterWrapGrid:
		style.InitialLetterWrap = val

		return true
	default:
		// <length-percentage> stored raw for cascade fidelity; consumer
		// treats unknown as none.
		if v, ok := lengthBox(val, style.FontSize, 0, ""); ok && v >= 0 {
			style.InitialLetterWrap = val

			return true
		}

		return false
	}
}
