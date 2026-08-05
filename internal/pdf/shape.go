package pdf

import (
	"unicode"
	"unicode/utf8"
)

// ShapeText applies best-effort text shaping before PDF emission when no
// font face is available for OpenType. Prefer ShapeTextFont when a *Font is
// known (TextShow does).
//
// Fallback pipeline (no GSUB / face unavailable):
//  1. Unicode NFC normalize (helps some Indic sequences)
//  2. Arabic presentation-form joining (initial/medial/final/isolated)
//  3. RTL run reverse for Arabic/Hebrew ranges
//
// When the active face has OpenType GSUB, ShapeTextFont uses
// go-text/typesetting and reverse-cmaps glyphs to Unicode CIDs for Type0.
// Indic remains Partial — matra reordering is only as good as the OT face
// and reverse-cmap coverage.
func ShapeText(s string) string {
	if s == "" {
		return s
	}
	s = string([]rune(s)) // ensure valid
	s = nfcNormalize(s)
	s = shapeArabicJoining(s)
	return reverseRTLRuns(s)
}

func nfcNormalize(s string) string {
	// Prefer stdlib when available via unicode iteration; full NFC requires
	// golang.org/x/text which is forbidden — do a minimal pass for common
	// Hangul/Arabic compatibility by returning s unchanged when no combining
	// marks, else strip orphaned combining marks after base (best-effort).
	if !hasCombining(s) {
		return s
	}
	runes := []rune(s)
	out := make([]rune, 0, len(runes))
	for _, r := range runes {
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) || unicode.Is(unicode.Me, r) {
			if len(out) == 0 {
				continue // drop leading marks
			}
			// Keep marks after a base (no reordering).
			out = append(out, r)
			continue
		}
		out = append(out, r)
	}
	return string(out)
}

func hasCombining(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Mc, r) || unicode.Is(unicode.Me, r) {
			return true
		}
	}
	return false
}

func reverseRTLRuns(s string) string {
	runes := []rune(s)
	out := make([]rune, 0, len(runes))
	i := 0
	for i < len(runes) {
		if isRTLRune(runes[i]) {
			j := i + 1
			for j < len(runes) && (isRTLRune(runes[j]) || isRTLNeutral(runes[j])) {
				j++
			}
			end := j
			for end > i && isRTLNeutral(runes[end-1]) {
				end--
			}
			for k := end - 1; k >= i; k-- {
				out = append(out, runes[k])
			}
			for k := end; k < j; k++ {
				out = append(out, runes[k])
			}
			i = j
			continue
		}
		out = append(out, runes[i])
		i++
	}
	return string(out)
}

func isRTLRune(r rune) bool {
	switch {
	case r >= 0x0590 && r <= 0x05FF:
		return true
	case r >= 0x0600 && r <= 0x06FF:
		return true
	case r >= 0x0700 && r <= 0x074F:
		return true
	case r >= 0x0750 && r <= 0x077F:
		return true
	case r >= 0x08A0 && r <= 0x08FF:
		return true
	case r >= 0xFB50 && r <= 0xFDFF:
		return true
	case r >= 0xFE70 && r <= 0xFEFF:
		return true
	}
	return unicode.Is(unicode.Hebrew, r) || unicode.Is(unicode.Arabic, r)
}

func isRTLNeutral(r rune) bool {
	switch r {
	case ' ', '\t', '-', '–', '—', '/', '\\', '.', ',', ':', ';', '!', '?', '\'', '"', '(', ')', '[', ']':
		return true
	}
	return false
}

// join types for Arabic letters in the U+0621..U+064A block.
const (
	joinNone  = iota
	joinRight // right-joining only (final/isolated)
	joinDual  // dual-joining (all four forms)
	joinTrans // transparent (harakat) — ignore for joining adjacency
)

// arabicForms maps base letter → [isolated, final, initial, medial].
// 0 means that form is unavailable (use isolated).
var arabicForms = map[rune][4]rune{
	0x0621: {0xFE80, 0, 0, 0},                // hamza
	0x0622: {0xFE81, 0xFE82, 0, 0},           // alef madda
	0x0623: {0xFE83, 0xFE84, 0, 0},           // alef hamza above
	0x0624: {0xFE85, 0xFE86, 0, 0},           // waw hamza
	0x0625: {0xFE87, 0xFE88, 0, 0},           // alef hamza below
	0x0626: {0xFE89, 0xFE8A, 0xFE8B, 0xFE8C}, // yeh hamza
	0x0627: {0xFE8D, 0xFE8E, 0, 0},           // alef
	0x0628: {0xFE8F, 0xFE90, 0xFE91, 0xFE92}, // beh
	0x0629: {0xFE93, 0xFE94, 0, 0},           // teh marbuta
	0x062A: {0xFE95, 0xFE96, 0xFE97, 0xFE98}, // teh
	0x062B: {0xFE99, 0xFE9A, 0xFE9B, 0xFE9C}, // theh
	0x062C: {0xFE9D, 0xFE9E, 0xFE9F, 0xFEA0}, // jeem
	0x062D: {0xFEA1, 0xFEA2, 0xFEA3, 0xFEA4}, // hah
	0x062E: {0xFEA5, 0xFEA6, 0xFEA7, 0xFEA8}, // khah
	0x062F: {0xFEA9, 0xFEAA, 0, 0},           // dal
	0x0630: {0xFEAB, 0xFEAC, 0, 0},           // thal
	0x0631: {0xFEAD, 0xFEAE, 0, 0},           // reh
	0x0632: {0xFEAF, 0xFEB0, 0, 0},           // zain
	0x0633: {0xFEB1, 0xFEB2, 0xFEB3, 0xFEB4}, // seen
	0x0634: {0xFEB5, 0xFEB6, 0xFEB7, 0xFEB8}, // sheen
	0x0635: {0xFEB9, 0xFEBA, 0xFEBB, 0xFEBC}, // sad
	0x0636: {0xFEBD, 0xFEBE, 0xFEBF, 0xFEC0}, // dad
	0x0637: {0xFEC1, 0xFEC2, 0xFEC3, 0xFEC4}, // tah
	0x0638: {0xFEC5, 0xFEC6, 0xFEC7, 0xFEC8}, // zah
	0x0639: {0xFEC9, 0xFECA, 0xFECB, 0xFECC}, // ain
	0x063A: {0xFECD, 0xFECE, 0xFECF, 0xFED0}, // ghain
	0x0641: {0xFED1, 0xFED2, 0xFED3, 0xFED4}, // feh
	0x0642: {0xFED5, 0xFED6, 0xFED7, 0xFED8}, // qaf
	0x0643: {0xFED9, 0xFEDA, 0xFEDB, 0xFEDC}, // kaf
	0x0644: {0xFEDD, 0xFEDE, 0xFEDF, 0xFEE0}, // lam
	0x0645: {0xFEE1, 0xFEE2, 0xFEE3, 0xFEE4}, // meem
	0x0646: {0xFEE5, 0xFEE6, 0xFEE7, 0xFEE8}, // noon
	0x0647: {0xFEE9, 0xFEEA, 0xFEEB, 0xFEEC}, // heh
	0x0648: {0xFEED, 0xFEEE, 0, 0},           // waw
	0x0649: {0xFEEF, 0xFEF0, 0, 0},           // alef maksura
	0x064A: {0xFEF1, 0xFEF2, 0xFEF3, 0xFEF4}, // yeh
}

// Lam-Alef ligatures: lam + alef variants → presentation ligature.
var lamAlef = map[rune][2]rune{ // [isolated, final]
	0x0622: {0xFEF5, 0xFEF6},
	0x0623: {0xFEF7, 0xFEF8},
	0x0625: {0xFEF9, 0xFEFA},
	0x0627: {0xFEFB, 0xFEFC},
}

func arabicJoinType(r rune) int {
	if r >= 0x064B && r <= 0x065F { // harakat / superscript alef etc.
		return joinTrans
	}
	if r == 0x0670 || r == 0x0640 { // superscript alef / tatweel
		if r == 0x0640 {
			return joinDual
		}
		return joinTrans
	}
	forms, ok := arabicForms[r]
	if !ok {
		return joinNone
	}
	if forms[2] != 0 { // has initial → dual
		return joinDual
	}
	if forms[0] != 0 {
		return joinRight
	}
	return joinNone
}

func shapeArabicJoining(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}
	// First pass: Lam-Alef ligatures (logical order).
	tmp := make([]rune, 0, len(runes))
	for i := 0; i < len(runes); i++ {
		if runes[i] == 0x0644 && i+1 < len(runes) {
			if lig, ok := lamAlef[runes[i+1]]; ok {
				// Decide isol vs fina from left neighbor later — store as isol for now
				// and tag via private use: use isol; joining pass upgrades.
				tmp = append(tmp, lig[0])
				i++
				continue
			}
		}
		tmp = append(tmp, runes[i])
	}
	runes = tmp
	out := make([]rune, len(runes))
	copy(out, runes)

	for i, r := range runes {
		jt := arabicJoinType(r)
		if jt == joinNone || jt == joinTrans {
			continue
		}
		// Lam-Alef ligature codepoints already presentation forms.
		if r >= 0xFEF5 && r <= 0xFEFC {
			prev := prevJoinCause(runes, i)
			if prev {
				out[i] = r + 1 // final form is isol+1 in this block
			}
			continue
		}
		forms, ok := arabicForms[r]
		if !ok {
			continue
		}
		prev := prevJoinCause(runes, i)
		next := nextJoinCause(runes, i)
		var form int // 0 isol 1 fina 2 init 3 medi
		switch jt {
		case joinDual:
			switch {
			case prev && next:
				form = 3
			case !prev && next:
				form = 2
			case prev && !next:
				form = 1
			default:
				form = 0
			}
		case joinRight:
			if prev {
				form = 1
			} else {
				form = 0
			}
		}
		if forms[form] != 0 {
			out[i] = forms[form]
		} else if forms[0] != 0 {
			out[i] = forms[0]
		}
	}
	return string(out)
}

func prevJoinCause(runes []rune, i int) bool {
	for j := i - 1; j >= 0; j-- {
		jt := arabicJoinType(runes[j])
		if jt == joinTrans {
			continue
		}
		return jt == joinDual // only dual-joining letters cause a join to the right
	}
	return false
}

func nextJoinCause(runes []rune, i int) bool {
	for j := i + 1; j < len(runes); j++ {
		jt := arabicJoinType(runes[j])
		if jt == joinTrans {
			continue
		}
		return jt == joinDual || jt == joinRight
	}
	return false
}

// ShapeNeeded reports whether s contains scripts that benefit from ShapeText.
func ShapeNeeded(s string) bool {
	for i := 0; i < len(s); {
		r, size := utf8.DecodeRuneInString(s[i:])
		i += size
		if isRTLRune(r) || unicode.Is(unicode.Mn, r) {
			return true
		}
	}
	return false
}
