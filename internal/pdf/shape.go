package pdf

import (
	"unicode"
	"unicode/utf8"
)

// ShapeText is the no-face fallback pipeline used when ShapeTextFont cannot
// run OpenType (no *Font, no GSUB, reverse-cmap miss). Prefer ShapeTextFont
// whenever a face is known (TextShow does).
//
// Fallback only:
//  1. light combining-mark pass (stdlib; full NFC needs x/text)
//  2. Arabic presentation-form joining (tables below)
//  3. RTL run reverse for Arabic/Hebrew ranges
//
// Indic remains Partial without OT.
func ShapeText(s string) string {
	if s == "" {
		return s
	}
	s = string([]rune(s)) // ensure valid
	s = stripOrphanMark(s)
	s = shapeArabicJoining(s)
	return reverseRTLRuns(s)
}

// stripOrphanMark drops leading combining marks (best-effort stand-in for
// full NFC, which needs golang.org/x/text and is forbidden here). Marks
// after a base character are kept as-is, without reordering.
func stripOrphanMark(s string) string {
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

// --- manual Arabic joining (fallback when no GSUB) ---
// ponytail: manual Arabic joining when no GSUB, OT via go-text when available

const (
	joinNone  = iota
	joinRight // right-joining only (final/isolated)
	joinDual  // dual-joining (all four forms)
	joinTrans // transparent (harakat) — ignore for joining adjacency
)

// arabicForms: base → [isolated, final, initial, medial]; 0 = use isolated.
// Covers U+0621..U+064A presentation forms only — enough for common Arabic
// without OT; expand only if a no-GSUB face is a product requirement.
var arabicForms = map[rune][4]rune{
	0x0621: {0xFE80, 0, 0, 0},
	0x0622: {0xFE81, 0xFE82, 0, 0},
	0x0623: {0xFE83, 0xFE84, 0, 0},
	0x0624: {0xFE85, 0xFE86, 0, 0},
	0x0625: {0xFE87, 0xFE88, 0, 0},
	0x0626: {0xFE89, 0xFE8A, 0xFE8B, 0xFE8C},
	0x0627: {0xFE8D, 0xFE8E, 0, 0},
	0x0628: {0xFE8F, 0xFE90, 0xFE91, 0xFE92},
	0x0629: {0xFE93, 0xFE94, 0, 0},
	0x062A: {0xFE95, 0xFE96, 0xFE97, 0xFE98},
	0x062B: {0xFE99, 0xFE9A, 0xFE9B, 0xFE9C},
	0x062C: {0xFE9D, 0xFE9E, 0xFE9F, 0xFEA0},
	0x062D: {0xFEA1, 0xFEA2, 0xFEA3, 0xFEA4},
	0x062E: {0xFEA5, 0xFEA6, 0xFEA7, 0xFEA8},
	0x062F: {0xFEA9, 0xFEAA, 0, 0},
	0x0630: {0xFEAB, 0xFEAC, 0, 0},
	0x0631: {0xFEAD, 0xFEAE, 0, 0},
	0x0632: {0xFEAF, 0xFEB0, 0, 0},
	0x0633: {0xFEB1, 0xFEB2, 0xFEB3, 0xFEB4},
	0x0634: {0xFEB5, 0xFEB6, 0xFEB7, 0xFEB8},
	0x0635: {0xFEB9, 0xFEBA, 0xFEBB, 0xFEBC},
	0x0636: {0xFEBD, 0xFEBE, 0xFEBF, 0xFEC0},
	0x0637: {0xFEC1, 0xFEC2, 0xFEC3, 0xFEC4},
	0x0638: {0xFEC5, 0xFEC6, 0xFEC7, 0xFEC8},
	0x0639: {0xFEC9, 0xFECA, 0xFECB, 0xFECC},
	0x063A: {0xFECD, 0xFECE, 0xFECF, 0xFED0},
	0x0641: {0xFED1, 0xFED2, 0xFED3, 0xFED4},
	0x0642: {0xFED5, 0xFED6, 0xFED7, 0xFED8},
	0x0643: {0xFED9, 0xFEDA, 0xFEDB, 0xFEDC},
	0x0644: {0xFEDD, 0xFEDE, 0xFEDF, 0xFEE0},
	0x0645: {0xFEE1, 0xFEE2, 0xFEE3, 0xFEE4},
	0x0646: {0xFEE5, 0xFEE6, 0xFEE7, 0xFEE8},
	0x0647: {0xFEE9, 0xFEEA, 0xFEEB, 0xFEEC},
	0x0648: {0xFEED, 0xFEEE, 0, 0},
	0x0649: {0xFEEF, 0xFEF0, 0, 0},
	0x064A: {0xFEF1, 0xFEF2, 0xFEF3, 0xFEF4},
}

// Lam-Alef ligatures: lam + alef variants → presentation ligature.
var lamAlef = map[rune][2]rune{ // [isolated, final]
	0x0622: {0xFEF5, 0xFEF6},
	0x0623: {0xFEF7, 0xFEF8},
	0x0625: {0xFEF9, 0xFEFA},
	0x0627: {0xFEFB, 0xFEFC},
}

func arabicJoinType(r rune) int {
	if r >= 0x064B && r <= 0x065F {
		return joinTrans
	}
	if r == 0x0670 || r == 0x0640 {
		if r == 0x0640 {
			return joinDual
		}
		return joinTrans
	}
	forms, ok := arabicForms[r]
	if !ok {
		return joinNone
	}
	if forms[2] != 0 {
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
	// Lam-Alef ligatures (logical order).
	tmp := make([]rune, 0, len(runes))
	for i := 0; i < len(runes); i++ {
		if runes[i] == 0x0644 && i+1 < len(runes) {
			if lig, ok := lamAlef[runes[i+1]]; ok {
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
		if r >= 0xFEF5 && r <= 0xFEFC {
			if prevJoinCause(runes, i) {
				out[i] = r + 1 // final = isol+1 in this block
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
		return jt == joinDual
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

// ShapeNeeded reports whether s contains scripts that benefit from shaping.
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
