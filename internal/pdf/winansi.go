package pdf

// Simple-font WinAnsi encoding.
//
// A simple font content stream carries one byte per rune and declares
// /Encoding /WinAnsiEncoding. The embedded subset cmap is indexed by the
// Unicode code point each byte decodes to (ISO 32000-1 Annex D), so a fold has
// to keep track of both the byte written and the code point the subset maps.
// Curly quotes and bullets are the cases where the two differ: WinAnsi bytes
// 0x91-0x94 show the curly quotes, and byte 0x95 shows the disc. Folding the
// disc to the middle dot (0xB7) shrank every disc list marker to the 0.277em
// middle-dot advance, and folding the curly quotes to ASCII painted straight
// glyphs and flattened the extracted text.

// winAnsiBulletByte is the WinAnsiEncoding code for the bullet glyphs: 0x95,
// the disc (U+2022). U+2023, U+25E6 and U+2043 fold to it because the simple
// path cannot show their exact forms; a disc beats a middle dot.
const winAnsiBulletByte = 0x95

// WinAnsiEncoding codes for the curly quotes (ISO 32000-1 Annex D.2). Folding
// them to ASCII painted straight glyphs and made extractors report straight
// quotes; these codes keep both the curly glyph and the curly /ToUnicode map.
const (
	winAnsiQuoteSingleLeft  = 0x91 // U+2018 left single quotation mark
	winAnsiQuoteSingleRight = 0x92 // U+2019 right single quotation mark
	winAnsiQuoteDoubleLeft  = 0x93 // U+201C left double quotation mark
	winAnsiQuoteDoubleRight = 0x94 // U+201D right double quotation mark
)

// winAnsiFoldAboveLatin1 maps common HTML/CSS punctuation above Latin-1 to the
// WinAnsi byte a simple text run shows it with. Curly quotes and bullets keep
// their real WinAnsi codes; other marks fold to an ASCII stand-in.
//
//nolint:gochecknoglobals // immutable fold table, same pattern as winAnsiPunct
var winAnsiFoldAboveLatin1 = map[rune]byte{
	'\u2018': winAnsiQuoteSingleLeft,
	'\u2019': winAnsiQuoteSingleRight,
	'\u201C': winAnsiQuoteDoubleLeft,
	'\u201D': winAnsiQuoteDoubleRight,
	'\u2022': winAnsiBulletByte, // disc
	'\u2023': winAnsiBulletByte, // triangular bullet → disc
	'\u25E6': winAnsiBulletByte, // white bullet → disc
	'\u2043': winAnsiBulletByte, // hyphen bullet → disc
	'\u2026': '.',               // ellipsis
	'\u2009': ' ',               // thin space
	'\u200A': ' ',               // hair space
	'\u2008': ' ',               // punctuation space
	'\u2002': ' ',               // en space
	'\u2003': ' ',               // em space
	'\u2715': 'x',               // multiplication x
	'\u2716': 'x',               // heavy multiplication x
}

// winAnsiFoldCode maps r to the WinAnsi code a simple text run shows it with.
// Latin-1 passes through unchanged. Curly quotes and bullets keep their real
// WinAnsi codes (0x91-0x94, 0x95) instead of the ASCII and middle dot folds
// the PDFDocEncoding path uses; other common HTML/CSS punctuation above
// Latin-1 folds to an ASCII stand-in. ok is false when no WinAnsi code
// represents r; the simple emitter then writes '?'.
func winAnsiFoldCode(r rune) (byte, bool) {
	if r >= 0 && r <= maxLatin1Code {
		return byte(r), true
	}

	code, ok := winAnsiFoldAboveLatin1[r]

	return code, ok
}

// winAnsiPunct maps the WinAnsiEncoding 0x80-0x9F block to the Unicode code
// point each code decodes to (ISO 32000-1 Annex D.2). The block holds the
// punctuation cp1252 moved out of C1; the five unassigned codes (0x81, 0x8D,
// 0x8F, 0x90, 0x9D) are absent and decode to themselves.
//
//nolint:gochecknoglobals // immutable code table, same pattern as pdfBlendModes
var winAnsiPunct = map[byte]rune{
	0x80: '\u20AC',
	0x82: '\u201A',
	0x83: '\u0192',
	0x84: '\u201E',
	0x85: '\u2026',
	0x86: '\u2020',
	0x87: '\u2021',
	0x88: '\u02C6',
	0x89: '\u2030',
	0x8A: '\u0160',
	0x8B: '\u2039',
	0x8C: '\u0152',
	0x8E: '\u017D',
	0x91: '\u2018',
	0x92: '\u2019',
	0x93: '\u201C',
	0x94: '\u201D',
	0x95: '\u2022',
	0x96: '\u2013',
	0x97: '\u2014',
	0x98: '\u02DC',
	0x99: '\u2122',
	0x9A: '\u0161',
	0x9B: '\u203A',
	0x9C: '\u0153',
	0x9E: '\u017E',
	0x9F: '\u0178',
}

// winAnsiDecode maps a WinAnsi code to the Unicode code point it decodes to
// (ISO 32000-1 Annex D.2). Codes outside the punctuation block are Latin-1.
func winAnsiDecode(code byte) rune {
	if decoded, ok := winAnsiPunct[code]; ok {
		return decoded
	}

	return rune(code)
}
