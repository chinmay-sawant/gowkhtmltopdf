package pdf

// UnicodeRange is one inclusive code point span from a CSS unicode-range
// descriptor, for example U+0100-02AF.
type UnicodeRange struct {
	Lo rune
	Hi rune
}

// Covers reports whether r falls inside the span.
func (u UnicodeRange) Covers(r rune) bool {
	return r >= u.Lo && r <= u.Hi
}

// FaceSpec carries the CSS @font-face descriptors that select a registered
// face. The zero value keeps the pre-descriptor behavior: weight and italic
// come from the font file (OS/2 usWeightClass and the macStyle italic bit),
// and the face covers every code point.
type FaceSpec struct {
	// Weight is the font-weight descriptor (1..1000); 0 when unspecified.
	Weight int
	// Italic is the font-style descriptor; StyleSet records whether the
	// descriptor was present. Without it the file's own italic bit decides.
	Italic   bool
	StyleSet bool
	// Ranges is the unicode-range descriptor; empty covers every code point,
	// the CSS descriptor default.
	Ranges []UnicodeRange
}

// covers reports whether the spec declares coverage of codePoint. No declared
// ranges means "everything".
func (s FaceSpec) covers(codePoint rune) bool {
	if len(s.Ranges) == 0 {
		return true
	}

	for _, span := range s.Ranges {
		if span.Covers(codePoint) {
			return true
		}
	}

	return false
}

// primaryProbeRunes decide which face of a multi-face family becomes the
// family's primary face. The space probe matters most: inline layout paints
// whitespace with the primary face without checking its cmap, so a face
// without a space glyph can never head a family.
const primaryProbeRunes = " Aa0,.-"

// rangeProbeScore counts the probes the declared unicode-range covers.
func (s FaceSpec) rangeProbeScore() int {
	score := 0

	for _, r := range primaryProbeRunes {
		if s.covers(r) {
			score++
		}
	}

	return score
}

// glyphProbeScore counts the probes the face actually maps. It breaks ties
// between faces whose declarations look equal (for example two faces without a
// unicode-range descriptor, the Google Fonts latin and latin-ext files when a
// page forgets the descriptor).
func glyphProbeScore(fnt *Font) int {
	if fnt == nil {
		return 0
	}

	score := 0

	for _, r := range primaryProbeRunes {
		if fnt.GlyphID(r) != 0 {
			score++
		}
	}

	return score
}
