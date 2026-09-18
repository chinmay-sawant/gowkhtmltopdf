package layout

import (
	"strings"
	"unicode"
)

const (
	// autospaceGapEm is the lite inter-script gap for ideograph-alpha (1/8 em).
	autospaceGapEm = 0.125
	// softHyphenRune is Unicode SOFT HYPHEN (U+00AD).
	softHyphenRune              = '\u00AD'
	textAutospaceNone           = "no-autospace"
	textAutospaceIdeographAlpha = "ideograph-alpha"
	textSpacingTrimRatio        = 0.5
)

// textAutospaceGap returns the extra advance inserted between prev and cur
// when text-autospace requests ideograph/alpha spacing.
func textAutospaceGap(style *ResolvedStyle, prev, cur rune, emSize float64) float64 {
	if style == nil || !autospaceIdeographAlpha(style.TextAutospace) {
		return 0
	}

	if prev == 0 || cur == 0 || prev == softHyphenRune || cur == softHyphenRune {
		return 0
	}

	if IsIdeographAlphaBoundary(prev, cur) {
		return emSize * autospaceGapEm
	}

	return 0
}

// IsIdeographAlphaBoundary reports either direction of an ideograph/Latin
// boundary. The image painter uses the same classifier as layout measurement.
func IsIdeographAlphaBoundary(a, b rune) bool {
	return ideographAlphaPair(a, b) || ideographAlphaPair(b, a)
}

// TextAutospaceGap returns the point-sized gap requested by a text op.
// Paint backends use it to reproduce the layout advance between the same
// ideograph and alphabetic boundaries.
func (op Op) TextAutospaceGap() float64 {
	if !autospaceIdeographAlpha(op.TextAutospace()) || op.Size <= 0 {
		return 0
	}

	return op.Size * autospaceGapEm
}

func autospaceIdeographAlpha(val string) bool {
	if val == "" || val == textAutospaceNone {
		return false
	}

	if val == textAutospaceIdeographAlpha || val == "auto" || val == contentNormal {
		return true
	}

	for _, tok := range strings.Fields(val) {
		if tok == textAutospaceIdeographAlpha || tok == "auto" {
			return true
		}
	}

	return false
}

func ideographAlphaPair(a, b rune) bool {
	return isIdeographicRune(a) && isLatinAlphaRune(b)
}

func isIdeographicRune(candidate rune) bool {
	switch {
	case candidate >= 0x3040 && candidate <= 0x30FF: // Hiragana + Katakana
		return true
	case candidate >= 0x3400 && candidate <= 0x9FFF: // CJK unified + ext A
		return true
	case candidate >= 0xF900 && candidate <= 0xFAFF: // CJK compatibility ideographs
		return true
	case candidate >= 0xFF66 && candidate <= 0xFF9D: // halfwidth katakana
		return true
	default:
		return unicode.In(candidate, unicode.Han, unicode.Hiragana, unicode.Katakana)
	}
}

func isLatinAlphaRune(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z')
}

// textSpacingTrimLeading returns how much to hang/trim at the start of a line
// for text-spacing-trim (lite: leading fullwidth punctuation only).
func (e *engine) textSpacingTrimLeading(style *ResolvedStyle, text string) float64 {
	if style == nil || text == "" {
		return 0
	}

	trim := style.TextSpacingTrim
	if trim == "" || trim == "space-all" || trim == contentNormal {
		return 0
	}

	if trim != textBoxTrimStartKeyword && trim != textBoxTrimBothKeyword && trim != "space-or-trim" {
		return 0
	}

	r := []rune(text)[0]
	if !isFullwidthOpenPunct(r) {
		return 0
	}

	return e.measureRuneFace(r, style) * textSpacingTrimRatio
}

func isFullwidthOpenPunct(r rune) bool {
	switch r {
	case '「', '『', '（', '［', '｛', '｢', '〈', '《', '【', '〔':
		return true
	default:
		return false
	}
}

// resolveTextGroupAlign maps text-group-align onto the shared line-origin
// align keywords when text-align left the line at start.
//
//nolint:cyclop // group alignment maps several CSS aliases to line origins
func resolveTextGroupAlign(block *ResolvedStyle, textAlign string) string {
	if block == nil {
		return textAlign
	}

	groupAlign := block.TextGroupAlign
	if groupAlign == "" || groupAlign == cssDisplayNone {
		return textAlign
	}

	// Lite: group-align wins when the line would otherwise start-align.
	if textAlign != "" && textAlign != floatLeft && textAlign != fxStart {
		return textAlign
	}

	switch groupAlign {
	case "center":
		return fxCenter
	case fxEnd, "right":
		return floatRight
	case fxStart, "left":
		return floatLeft
	default:
		return textAlign
	}
}
