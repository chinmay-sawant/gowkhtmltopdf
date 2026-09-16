package pdf

// Simple-font WinAnsi encoding.
//
// A simple font content stream carries one byte per rune and declares
// /Encoding /WinAnsiEncoding. The embedded subset cmap is indexed by the
// Unicode code point each byte decodes to (ISO 32000-1 Annex D), so a fold has
// to keep track of both the byte written and the code point the subset maps.
// Bullets are the case where the two differ: WinAnsi byte 0x95 shows the disc,
// and folding it to the middle dot (0xB7) shrank every disc list marker to the
// 0.277em middle-dot advance.

// winAnsiBulletByte is the WinAnsiEncoding code for the bullet glyphs: 0x95,
// the disc (U+2022). U+2023, U+25E6 and U+2043 fold to it because the simple
// path cannot show their exact forms; a disc beats a middle dot.
const winAnsiBulletByte = 0x95

// winAnsiFoldCode maps r to the WinAnsi code a simple text run shows it with.
// Latin-1 passes through unchanged. Common HTML/CSS punctuation above Latin-1
// folds to an ASCII stand-in; bullets keep their real WinAnsi code (0x95)
// instead of the middle dot the PDFDocEncoding fold uses. ok is false when no
// WinAnsi code represents r; the simple emitter then writes '?'.
func winAnsiFoldCode(r rune) (byte, bool) {
	if r >= 0 && r <= maxLatin1Code {
		return byte(r), true
	}

	switch r {
	case '\u2018', '\u2019': // curly single quotes
		return '\'', true
	case '\u201C', '\u201D': // curly double quotes
		return '"', true
	case '\u2022', '\u2023', '\u25E6', '\u2043': // bullets → disc byte
		return winAnsiBulletByte, true
	case '\u2026': // ellipsis
		return '.', true
	case '\u2009', '\u200A', '\u2008', '\u2002', '\u2003': // thin/space runs
		return ' ', true
	case '\u2715', '\u2716': // cross marks → ASCII x
		return 'x', true
	}

	return 0, false
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
