package layout

const (
	vsTextPresentation     = '\uFE0E'
	vsEmojiPresentation    = '\uFE0F'
	fontVariantEmojiKW     = "emoji"
	fontVariantTextKW      = "text"
	fontVariantUnicode     = "unicode"
	emojiVariationCapacity = 4
	// liteEmojiFill is the solid fallback when emoji presentation is requested
	// and the face has no color glyph. Distinct from typical CSS color:#111 so
	// text vs emoji A/B demos change paint without a COLR emoji font.
	liteEmojiFillR = 1.0
	liteEmojiFillG = 0.78
	liteEmojiFillB = 0.12
)

// applyEmojiVariationSelectors inserts U+FE0E (text) or U+FE0F (emoji) after
// emoji-candidate codepoints according to font-variant-emoji. Existing
// variation selectors are replaced. Callers that paint PDF text must drop
// selectors the face cannot cmap (bundled Liberation maps them to .notdef).
func applyEmojiVariationSelectors(text, variant string) string {
	if text == "" {
		return text
	}

	out := make([]rune, 0, len(text)+emojiVariationCapacity)

	for _, candidate := range text {
		if candidate == vsTextPresentation || candidate == vsEmojiPresentation {
			continue
		}

		out = append(out, candidate)

		if !isEmojiCandidate(candidate) {
			continue
		}

		vs := vsTextPresentation
		if wantsEmojiPresentation(candidate, variant) {
			vs = vsEmojiPresentation
		}

		out = append(out, vs)
	}

	return string(out)
}

func wantsEmojiPresentation(candidate rune, variant string) bool {
	if !isEmojiCandidate(candidate) {
		return false
	}

	switch variant {
	case fontVariantEmojiKW:
		return true
	case fontVariantTextKW:
		return false
	case fontVariantUnicode, fontVariantNormal:
		return emojiDefaultIsEmoji(candidate)
	default:
		return emojiDefaultIsEmoji(candidate)
	}
}

//nolint:cyclop // Unicode emoji ranges are independent, spec-defined cases
func isEmojiCandidate(candidate rune) bool {
	switch {
	case candidate >= 0x1F000 && candidate <= 0x1FAFF:
		return true
	case candidate >= 0x2600 && candidate <= 0x27BF:
		return true
	case candidate >= 0x2300 && candidate <= 0x23FF:
		return true
	case candidate >= 0x2B50 && candidate <= 0x2B55:
		return true
	case candidate == 0x00A9 || candidate == 0x00AE || candidate == 0x2122:
		return true
	default:
		return false
	}
}

func emojiDefaultIsEmoji(r rune) bool {
	return r >= 0x1F000 && r <= 0x1FAFF
}

func isVariationSelector(r rune) bool {
	return r == vsTextPresentation || r == vsEmojiPresentation
}

// emojiPresentationAdvance returns a 1em minimum advance for emoji
// presentation of an emoji-candidate rune, else 0 (caller keeps glyph width).
func emojiPresentationAdvance(sty *ResolvedStyle, r rune, size float64) float64 {
	if sty == nil || isVariationSelector(r) {
		return 0
	}

	if !wantsEmojiPresentation(r, sty.FontVariantEmoji) {
		return 0
	}

	return size
}

// fontVariantEmojiFill applies the lite emoji presentation fill when every
// non-space, non-VS rune in text is an emoji candidate under emoji
// presentation. Mixed runs keep CSS color.
func fontVariantEmojiFill(sty *ResolvedStyle, text string) (float64, float64, float64, bool) {
	if sty == nil || text == "" {
		return 0, 0, 0, false
	}

	hasEmoji := false

	for _, r := range text {
		if r == ' ' || r == '\t' || isVariationSelector(r) {
			continue
		}

		if !wantsEmojiPresentation(r, sty.FontVariantEmoji) {
			return 0, 0, 0, false
		}

		hasEmoji = true
	}

	if !hasEmoji {
		return 0, 0, 0, false
	}

	return liteEmojiFillR, liteEmojiFillG, liteEmojiFillB, true
}
