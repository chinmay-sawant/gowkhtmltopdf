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
