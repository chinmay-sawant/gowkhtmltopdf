package layout

import (
	"strings"
	"unicode"
)

const (
	// autospaceGapEm is the lite inter-script gap for ideograph-alpha (1/8 em).
	autospaceGapEm = 0.125
	// softHyphenRune is Unicode SOFT HYPHEN (U+00AD).
	softHyphenRune = '\u00AD'
)

// textAutospaceGap returns the extra advance inserted between prev and cur
// when text-autospace requests ideograph/alpha spacing.
func textAutospaceGap(style *ResolvedStyle, prev, cur rune, em float64) float64 {
	if style == nil || !autospaceIdeographAlpha(style.TextAutospace) {
		return 0
	}

	if prev == 0 || cur == 0 || prev == softHyphenRune || cur == softHyphenRune {
		return 0
	}

	if ideographAlphaPair(prev, cur) || ideographAlphaPair(cur, prev) {
		return em * autospaceGapEm
	}

	return 0
}

func autospaceIdeographAlpha(val string) bool {
	if val == "" || val == "no-autospace" {
		return false
	}

	if val == "ideograph-alpha" || val == "auto" || val == contentNormal {
		return true
	}

	for _, tok := range strings.Fields(val) {
		if tok == "ideograph-alpha" || tok == "auto" {
			return true
		}
	}

	return false
}

func ideographAlphaPair(a, b rune) bool {
	return isIdeographicRune(a) && isLatinAlphaRune(b)
}

func isIdeographicRune(r rune) bool {
	switch {
	case r >= 0x3040 && r <= 0x30FF: // Hiragana + Katakana
		return true
	case r >= 0x3400 && r <= 0x9FFF: // CJK unified + ext A
		return true
	case r >= 0xF900 && r <= 0xFAFF: // CJK compatibility ideographs
		return true
	case r >= 0xFF66 && r <= 0xFF9D: // halfwidth katakana
		return true
	default:
		return unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana)
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

	if trim != "trim-start" && trim != "trim-both" && trim != "space-or-trim" {
		return 0
	}

	r := []rune(text)[0]
	if !isFullwidthOpenPunct(r) {
		return 0
	}

	return e.measureRuneFace(r, style) * 0.5
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
func resolveTextGroupAlign(block *ResolvedStyle, textAlign string) string {
	if block == nil {
		return textAlign
	}

	ga := block.TextGroupAlign
	if ga == "" || ga == cssDisplayNone {
		return textAlign
	}

	// Lite: group-align wins when the line would otherwise start-align.
	if textAlign != "" && textAlign != floatLeft && textAlign != "start" {
		return textAlign
	}

	switch ga {
	case "center":
		return fxCenter
	case "end", "right":
		return floatRight
	case "start", "left":
		return floatLeft
	default:
		return textAlign
	}
}
