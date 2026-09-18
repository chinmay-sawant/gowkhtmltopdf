package layout

import (
	"strings"
	"unicode"
)

const (
	hyphenationManual      = "manual"
	defaultHyphenateWord   = 5
	defaultHyphenateBefore = 2
	defaultHyphenateAfter  = 2
	hyphenationPercentBase = 100
)

// hyphenationAllowed reports whether soft-hyphen breaks may fire for style.
// hyphens:none suppresses SHY breaks; manual and auto both honor authored SHY
// (true dictionary auto is out of this batch).
func hyphenationAllowed(style *ResolvedStyle) bool {
	if style == nil {
		return true // CSS initial hyphens:manual
	}

	switch style.Hyphens {
	case cssDisplayNone:
		return false
	case hyphenationManual, "auto", "":
		return true
	default:
		return true
	}
}

func hyphenateCharacterOf(style *ResolvedStyle) string {
	if style == nil || style.HyphenateCharacter == "" || style.HyphenateCharacter == "auto" {
		return "-"
	}

	return style.HyphenateCharacter
}

func hyphenateLimitWord(style *ResolvedStyle) int {
	if style == nil || style.HyphenateLimitMinWord <= 0 {
		return defaultHyphenateWord
	}

	return style.HyphenateLimitMinWord
}

func hyphenateLimitBefore(style *ResolvedStyle) int {
	if style == nil || style.HyphenateLimitMinBefore <= 0 {
		return defaultHyphenateBefore
	}

	return style.HyphenateLimitMinBefore
}

func hyphenateLimitAfter(style *ResolvedStyle) int {
	if style == nil || style.HyphenateLimitMinAfter <= 0 {
		return defaultHyphenateAfter
	}

	return style.HyphenateLimitMinAfter
}

// hyphenateZoneWidth is the trailing slack that must remain for a SHY break
// to be preferred (hyphenate-limit-zone). Zero means any slack is fine.
func (e *engine) hyphenateZoneWidth(style *ResolvedStyle, lineW float64) float64 {
	if style == nil {
		return 0
	}

	if style.HyphenateLimitZonePct >= 0 {
		return lineW * style.HyphenateLimitZonePct / hyphenationPercentBase
	}

	if style.HyphenateLimitZonePt > 0 {
		return e.scalePt(style.HyphenateLimitZonePt)
	}

	return 0
}

// shyBreakIndex picks a soft-hyphen split that fits remainW, honors
// hyphenate-limit-chars, and (when zone > 0) keeps leftover slack inside the
// zone. Returns the rune index of the SHY (exclusive end of left piece) or -1.
//
//nolint:cyclop,nestif // soft-hyphen selection combines width, zone, and limit rules
func (e *engine) shyBreakIndex(
	text string, style *ResolvedStyle, remainW, lineW float64,
) int {
	if !hyphenationAllowed(style) || text == "" || style == nil {
		return -1
	}

	runes := []rune(text)
	hyphen := hyphenateCharacterOf(style)
	hyphenW := 0.0

	for _, r := range hyphen {
		hyphenW += e.measureRuneFace(r, style)
	}

	minWord := hyphenateLimitWord(style)
	minBefore := hyphenateLimitBefore(style)
	minAfter := hyphenateLimitAfter(style)
	zone := e.hyphenateZoneWidth(style, lineW)

	letterCount := countLetters(runes)
	if letterCount < minWord {
		return -1
	}

	best := -1
	width := 0.0

	for runeIndex, candidate := range runes {
		if candidate == softHyphenRune {
			before := countLetters(runes[:runeIndex])
			after := countLetters(runes[runeIndex+1:])

			if before >= minBefore && after >= minAfter {
				leftW := width + hyphenW
				if leftW <= remainW+inlineFitEpsilon {
					slack := remainW - leftW
					if zone <= 0 || slack <= zone+inlineFitEpsilon {
						best = runeIndex
					}
				}
			}
			// SHY itself has zero advance.
			continue
		}

		width += e.measureRuneFace(candidate, style)
		em := style.FontSize * e.scale

		if runeIndex > 0 {
			width += textAutospaceGap(style, runes[runeIndex-1], candidate, em)
		}
	}

	return best
}

func countLetters(runes []rune) int {
	letters := 0

	for _, candidate := range runes {
		if candidate != softHyphenRune && unicode.IsLetter(candidate) {
			letters++
		}
	}

	return letters
}

// lineEndsWithHyphen reports that the packed line finished with a hyphenation
// character (used for hyphenate-limit-lines consecutive tracking).
func lineEndsWithHyphen(line []inlineItem) bool {
	for idx := len(line) - 1; idx >= 0; idx-- {
		text := strings.TrimRight(line[idx].text, " ")
		if text == "" {
			continue
		}

		hyphen := hyphenateCharacterOf(line[idx].style)
		if hyphen != "" && strings.HasSuffix(text, hyphen) {
			return true
		}

		return false
	}

	return false
}

// maybeSplitHyphen splits item at a fitting soft hyphen. Returns nil when no
// SHY break applies.
func (e *engine) maybeSplitHyphen(item inlineItem, remainW, lineW float64) []inlineItem {
	if item.img || item.blockBox != nil || item.forceBreak || item.text == "" || item.noSplit {
		return nil
	}

	idx := e.shyBreakIndex(item.text, item.style, remainW, lineW)
	if idx < 0 {
		return nil
	}

	runes := []rune(item.text)
	hyphen := hyphenateCharacterOf(item.style)
	left := string(runes[:idx]) + hyphen
	right := string(runes[idx+1:])

	if strings.TrimSpace(right) == "" {
		return nil
	}

	return e.chunkParts(item, []string{left, right})
}

// stripSoftHyphens removes U+00AD for paint/measure when hyphens:none (SHY
// must not render and must not create breaks).
func stripSoftHyphens(text string) string {
	if !strings.ContainsRune(text, softHyphenRune) {
		return text
	}

	return strings.Map(func(r rune) rune {
		if r == softHyphenRune {
			return -1
		}

		return r
	}, text)
}

// hangingPunctuationFirstWidth returns the advance of a leading opening
// punctuation mark when hanging-punctuation:first is set.
func (e *engine) hangingPunctuationFirstWidth(style *ResolvedStyle, text string) float64 {
	if style == nil || text == "" {
		return 0
	}

	if !hangingPunctuationHas(style.HangingPunctuation, "first") {
		return 0
	}

	r := []rune(text)[0]
	if !isHangingOpenPunct(r) {
		return 0
	}

	return e.measureRuneFace(r, style)
}

func hangingPunctuationHas(val, flag string) bool {
	if val == "" || val == cssDisplayNone {
		return false
	}

	for _, tok := range strings.Fields(val) {
		if tok == flag {
			return true
		}
	}

	return false
}

func isHangingOpenPunct(r rune) bool {
	switch r {
	case '"', '\'', '`', '(', '[', '{', '«', '『', '「', '（', '［', '｛', '｢',
		'‘', '“', '‹', '〈', '《', '【', '〔':
		return true
	default:
		return false
	}
}

// hyphenateLimitLastBlocks reports that hyphenation should be suppressed on
// this line because hyphenate-limit-last says so (lite: always blocks the
// last line of the inline formatting context).
func hyphenateLimitLastBlocks(style *ResolvedStyle, lastLine bool) bool {
	if style == nil || !lastLine {
		return false
	}

	switch style.HyphenateLimitLast {
	case pageBreakAlways, floatRefColumn, floatRefPage, "spread":
		return true
	default:
		return false
	}
}
