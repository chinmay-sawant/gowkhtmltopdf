package layout

import (
	"strings"
	"testing"
)

func TestFontVariantEmojiTextVsEmojiDiffers(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
.text { font-size: 18pt; color: #111111; font-variant-emoji: text; }
.emoji { font-size: 18pt; color: #111111; font-variant-emoji: emoji; }
`)
	resText := layoutHTML(t, `<html><body><span class="text">☺</span></body></html>`, cssSheet)
	resEmoji := layoutHTML(t, `<html><body><span class="emoji">☺</span></body></html>`, cssSheet)

	textOp := textOpRGB(t, resText, "☺")
	emojiOp := textOpRGB(t, resEmoji, "☺")

	if colorsNear(textOp, emojiOp, 0.05) {
		t.Fatalf("emoji fill RGB=(%.3f,%.3f,%.3f) must differ from text=(%.3f,%.3f,%.3f)",
			emojiOp.R, emojiOp.G, emojiOp.B, textOp.R, textOp.G, textOp.B)
	}

	if textOp.R > 0.2 || textOp.G > 0.2 || textOp.B > 0.2 {
		t.Fatalf("text presentation should keep CSS #111, got RGB=(%.3f,%.3f,%.3f)",
			textOp.R, textOp.G, textOp.B)
	}

	if emojiOp.R < 0.8 || emojiOp.G < 0.5 || emojiOp.B > 0.3 {
		t.Fatalf("emoji presentation fill RGB=(%.3f,%.3f,%.3f), want gold-ish",
			emojiOp.R, emojiOp.G, emojiOp.B)
	}

	if emojiOp.W+0.01 < textOp.W {
		t.Fatalf("emoji width=%.2f should be >= text width=%.2f", emojiOp.W, textOp.W)
	}
}

func TestFontVariantEmojiSelectors(t *testing.T) {
	t.Parallel()

	text := applyEmojiVariationSelectors("☺", "text")
	emoji := applyEmojiVariationSelectors("☺", "emoji")

	if !strings.Contains(text, string(vsTextPresentation)) {
		t.Fatalf("text presentation %q should include U+FE0E", text)
	}

	if !strings.Contains(emoji, string(vsEmojiPresentation)) {
		t.Fatalf("emoji presentation %q should include U+FE0F", emoji)
	}

	if text == emoji {
		t.Fatal("text vs emoji selectors must differ")
	}
}
