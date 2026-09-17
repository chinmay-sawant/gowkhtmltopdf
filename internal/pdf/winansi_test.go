package pdf

import (
	"strings"
	"testing"
)

// TestWinAnsiFoldBulletsKeepDisc pins the fold for every bullet code point the
// layout engine can emit: WinAnsi byte 0x95, which decodes to the disc
// (U+2022), not the middle dot (U+00B7). The middle dot shrank every disc
// marker to its 0.277em advance.
func TestWinAnsiFoldBulletsKeepDisc(t *testing.T) {
	t.Parallel()

	for _, r := range []rune{'\u2022', '\u2023', '\u25E6', '\u2043'} {
		code, ok := winAnsiFoldCode(r)
		if !ok || code != winAnsiBulletByte {
			t.Errorf("winAnsiFoldCode(%U) = (%#x, %v), want (0x%X, true)", r, code, ok, winAnsiBulletByte)
		}
	}

	if got := winAnsiDecode(winAnsiBulletByte); got != '\u2022' {
		t.Errorf("winAnsiDecode(0x%X) = %U, want U+2022", winAnsiBulletByte, got)
	}
}

// TestWinAnsiFoldCurlyQuotesKeepCodePoints pins the fold for the curly quotes
// WinAnsiEncoding carries at 0x91-0x94. They must stay on those exact codes so
// the painted glyph is the curly one and /ToUnicode extracts U+2018-U+201D.
// The old fold wrote ASCII ' and ", which painted straight glyphs and made
// every extractor report straight quotes for source text with curly ones.
func TestWinAnsiFoldCurlyQuotesKeepCodePoints(t *testing.T) {
	t.Parallel()

	for rVal, want := range map[rune]byte{
		'\u2018': 0x91, // left single quotation mark
		'\u2019': 0x92, // right single quotation mark
		'\u201C': 0x93, // left double quotation mark
		'\u201D': 0x94, // right double quotation mark
	} {
		code, ok := winAnsiFoldCode(rVal)
		if !ok || code != want {
			t.Errorf("winAnsiFoldCode(%U) = (%#x, %v), want (0x%X, true)", rVal, code, ok, want)
		}
	}
}

// TestWinAnsiDecodePunctuationBlock pins a sample of the WinAnsi 0x80-0x9F
// decode table used by the simple-font subset cmap and /ToUnicode, plus the
// Latin-1 pass-through either side of the block.
func TestWinAnsiDecodePunctuationBlock(t *testing.T) {
	t.Parallel()

	for code, want := range map[byte]rune{
		0x80: '\u20AC', // euro
		0x85: '\u2026', // ellipsis
		0x91: '\u2018', // left single quote
		0x95: '\u2022', // bullet
		0x99: '\u2122', // trademark
		0xB7: '\u00B7', // middle dot (outside the block, identity)
		0xE9: '\u00E9', // e acute (outside the block, identity)
	} {
		if got := winAnsiDecode(code); got != want {
			t.Errorf("winAnsiDecode(%#X) = %U, want %U", code, got, want)
		}
	}
}

// TestBulletPaintsDiscAndToUnicode converts a disc list marker and walks the
// fold/ToUnicode path end to end: the content stream carries WinAnsi byte
// 0x95, /ToUnicode maps that byte back to U+2022, and the subset keeps the
// bullet glyph. The old fold wrote 0xB7 (middle dot) in all three places.
func TestBulletPaintsDiscAndToUnicode(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	doc := NewDocument()
	doc.SetCompression(false)
	page := doc.AddPage(200, 200)
	content := page.Content()
	content.UseEmbeddedFont("F1", fnt)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(10, 50)
	content.TextShow("\u2022")
	content.EndText()

	out := string(writePDF(t, doc))

	assertBulletContentStream(t, out)

	// Glyph lookup runs on the decoded code point, so the subset keeps the
	// U+2022 bullet glyph instead of the U+0095 C1 control.
	sub, err := subsetFont(fnt, []rune{winAnsiBulletByte}, subsetSimple)
	if err != nil {
		t.Fatalf("subsetFont: %v", err)
	}

	if sub.glyphIDs[winAnsiBulletByte] == 0 {
		t.Error("subset dropped the bullet glyph for code 0x95")
	}

	// The subset cmap maps both the char code and its decoded code point, so
	// viewers that index the cmap by code and viewers that decode first both
	// find the glyph.
	merged, err := ParseTTF(sub.data)
	if err != nil {
		t.Fatalf("ParseTTF(subset): %v", err)
	}

	for _, r := range []rune{'\u2022', rune(winAnsiBulletByte)} {
		if merged.GlyphID(r) == 0 {
			t.Errorf("subset cmap lost the bullet glyph for %U", r)
		}
	}
}

// TestCurlyQuotesPaintRealCodesAndToUnicode converts the four curly quotes and
// walks the fold/ToUnicode path end to end: the content stream carries WinAnsi
// bytes 0x91-0x94, /ToUnicode maps each byte back to its curly code point, and
// the subset keeps the curly glyph. The old fold wrote ASCII ' and " in all
// three places.
func TestCurlyQuotesPaintRealCodesAndToUnicode(t *testing.T) {
	t.Parallel()

	fnt, err := DefaultFont()
	if err != nil {
		t.Fatal(err)
	}

	doc := NewDocument()
	doc.SetCompression(false)
	page := doc.AddPage(200, 200)
	content := page.Content()
	content.UseEmbeddedFont("F1", fnt)
	content.BeginText()
	content.SetFont("F1", 12)
	content.TextAt(10, 50)
	content.TextShow("\u2018\u2019\u201C\u201D")
	content.EndText()

	out := string(writePDF(t, doc))

	assertCurlyQuoteContentStream(t, out)

	// Glyph lookup runs on the decoded code point, so the subset keeps the
	// curly quote glyphs instead of the ASCII ones.
	quotes := []rune{
		winAnsiQuoteSingleLeft,
		winAnsiQuoteSingleRight,
		winAnsiQuoteDoubleLeft,
		winAnsiQuoteDoubleRight,
	}

	sub, err := subsetFont(fnt, quotes, subsetSimple)
	if err != nil {
		t.Fatalf("subsetFont: %v", err)
	}

	for _, rVal := range quotes {
		if sub.glyphIDs[rVal] == 0 {
			t.Errorf("subset dropped the curly quote glyph for code 0x%X", rVal)
		}
	}

	// The subset cmap maps both the char code and its decoded code point, so
	// viewers that index the cmap by code and viewers that decode first both
	// find the glyph.
	merged, err := ParseTTF(sub.data)
	if err != nil {
		t.Fatalf("ParseTTF(subset): %v", err)
	}

	for _, rVal := range []rune{'\u2018', '\u2019', '\u201C', '\u201D'} {
		if merged.GlyphID(rVal) == 0 {
			t.Errorf("subset cmap lost the curly quote glyph for %U", rVal)
		}
	}
}

// assertCurlyQuoteContentStream pins the content-stream and /ToUnicode choices
// for the curly quotes: bytes 0x91-0x94 escape as octal literals, /ToUnicode
// maps each byte back to its curly code point, and the simple WinAnsi path
// stays in use.
func assertCurlyQuoteContentStream(t *testing.T, out string) {
	t.Helper()

	// The four quotes are one string, so one literal carries all four bytes.
	if !strings.Contains(out, `(\221\222\223\224) Tj`) {
		t.Error(`content stream must carry the WinAnsi curly bytes (\221\222\223\224) Tj`)
	}

	for _, want := range []string{"<91> <2018>", "<92> <2019>", "<93> <201C>", "<94> <201D>"} {
		if !strings.Contains(out, want) {
			t.Errorf("/ToUnicode must map %s", want)
		}
	}

	if strings.Contains(out, "/Identity-H") {
		t.Error("curly quotes must stay on the simple WinAnsi path, not Type0")
	}
}

// assertBulletContentStream pins the content-stream and /ToUnicode choices for
// the disc bullet: byte 0x95 escapes as the octal literal \225, byte 0xB7 must
// not appear, and the simple WinAnsi path stays in use.
func assertBulletContentStream(t *testing.T, out string) {
	t.Helper()

	// Byte 0x95 escapes as the octal literal \225.
	if !strings.Contains(out, "(\\225) Tj") {
		t.Error("content stream must carry the WinAnsi disc byte (\\225) Tj")
	}

	if strings.Contains(out, "(\\267) Tj") {
		t.Error("content stream still paints the middle dot byte 0xB7")
	}

	if !strings.Contains(out, "<95> <2022>") {
		t.Error("/ToUnicode must map byte 0x95 back to U+2022")
	}

	if strings.Contains(out, "/Identity-H") {
		t.Error("a plain bullet must stay on the simple WinAnsi path, not Type0")
	}
}
