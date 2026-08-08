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
func ShapeText(str string) string {
	if str == "" {
		return str
	}

	str = string([]rune(str)) // ensure valid
	str = stripOrphanMark(str)
	str = shapeArabicJoining(str)

	return reverseRTLRuns(str)
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

	for _, rVal := range runes {
		if unicode.Is(unicode.Mn, rVal) || unicode.Is(unicode.Mc, rVal) || unicode.Is(unicode.Me, rVal) {
			if len(out) == 0 {
				continue // drop leading marks
			}
			// Keep marks after a base (no reordering).
			out = append(out, rVal)

			continue
		}

		out = append(out, rVal)
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

	idx := 0
	for idx < len(runes) {
		if isRTLRune(runes[idx]) {
			jdx := idx + 1
			for jdx < len(runes) && (isRTLRune(runes[jdx]) || isRTLNeutral(runes[jdx])) {
				jdx++
			}

			end := jdx
			for end > idx && isRTLNeutral(runes[end-1]) {
				end--
			}

			for k := end - 1; k >= idx; k-- {
				out = append(out, runes[k])
			}

			for k := end; k < jdx; k++ {
				out = append(out, runes[k])
			}

			idx = jdx

			continue
		}

		out = append(out, runes[idx])
		idx++
	}

	return string(out)
}

// rtlRanges are the Unicode blocks treated as right-to-left by the fallback
// pipeline (Hebrew, Arabic, Syriac, Arabic Supplement/Extended-A and the
// Arabic presentation forms).
var rtlRanges = [][2]rune{ //nolint:gochecknoglobals // immutable lookup table
	{0x0590, 0x05FF}, // Hebrew
	{0x0600, 0x06FF}, // Arabic
	{0x0700, 0x074F}, // Syriac
	{0x0750, 0x077F}, // Arabic Supplement
	{0x08A0, 0x08FF}, // Arabic Extended-A
	{0xFB50, 0xFDFF}, // Arabic Presentation Forms-A
	{0xFE70, 0xFEFF}, // Arabic Presentation Forms-B
}

func isRTLRune(rVal rune) bool {
	for _, rg := range rtlRanges {
		if rVal >= rg[0] && rVal <= rg[1] {
			return true
		}
	}

	return unicode.Is(unicode.Hebrew, rVal) || unicode.Is(unicode.Arabic, rVal)
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

// Arabic presentation form indices (0 isol 1 fina 2 init 3 medi).
const (
	formIsol = iota
	formFina
	formInit
	formMedi
)

// arabicForms: base → [isolated, final, initial, medial]; 0 = use isolated.
// Covers U+0621..U+064A presentation forms only — enough for common Arabic
// without OT; expand only if a no-GSUB face is a product requirement.
//
//nolint:gochecknoglobals // immutable Arabic presentation-form lookup tables
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
var lamAlef = map[rune][2]rune{ //nolint:gochecknoglobals // immutable lookup table; [isolated, final]
	0x0622: {0xFEF5, 0xFEF6},
	0x0623: {0xFEF7, 0xFEF8},
	0x0625: {0xFEF9, 0xFEFA},
	0x0627: {0xFEFB, 0xFEFC},
}

func arabicJoinType(rVal rune) int {
	if rVal >= 0x064B && rVal <= 0x065F {
		return joinTrans
	}

	if rVal == 0x0670 || rVal == 0x0640 {
		if rVal == arabicTatweel {
			return joinDual
		}

		return joinTrans
	}

	forms, ok := arabicForms[rVal]
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

// applyLamAlef folds lam + alef pairs into their presentation ligatures
// (logical order).
func applyLamAlef(runes []rune) []rune {
	tmp := make([]rune, 0, len(runes))

	for idx := 0; idx < len(runes); idx++ {
		if runes[idx] == 0x0644 && idx+1 < len(runes) {
			if lig, ok := lamAlef[runes[idx+1]]; ok {
				tmp = append(tmp, lig[0])
				idx++

				continue
			}
		}

		tmp = append(tmp, runes[idx])
	}

	return tmp
}

// dualForm selects the presentation form for a dual-joining letter from its
// joining context.
func dualForm(prev, next bool) int {
	switch {
	case prev && next:
		return formMedi
	case !prev && next:
		return formInit
	case prev && !next:
		return formFina
	default:
		return formIsol
	}
}

// selectArabicForm picks the presentation form index (isolated/final/
// initial/medial) from the joining type and the joining context.
func selectArabicForm(jt int, prev, next bool) int {
	if jt == joinDual {
		return dualForm(prev, next)
	}

	if jt == joinRight && prev {
		return formFina
	}

	return formIsol
}

// isLamAlefLigature reports whether run is a lam-alef presentation ligature.
func isLamAlefLigature(run rune) bool {
	return run >= 0xFEF5 && run <= 0xFEFC
}

// shapeArabicRune writes the shaped form of runes[idx] into out.
func shapeArabicRune(out []rune, runes []rune, idx int) {
	run := runes[idx]
	jtVal := arabicJoinType(run)

	if jtVal == joinNone || jtVal == joinTrans {
		return
	}

	if isLamAlefLigature(run) {
		if prevJoinCause(runes, idx) {
			out[idx] = run + 1 // final = isol+1 in this block
		}

		return
	}

	forms, ok := arabicForms[run]
	if !ok {
		return
	}

	form := selectArabicForm(jtVal, prevJoinCause(runes, idx), nextJoinCause(runes, idx))

	if forms[form] != 0 {
		out[idx] = forms[form]
	} else if forms[0] != 0 {
		out[idx] = forms[0]
	}
}

func shapeArabicJoining(s string) string {
	runes := []rune(s)
	if len(runes) == 0 {
		return s
	}

	runes = applyLamAlef(runes)
	out := make([]rune, len(runes))
	copy(out, runes)

	for idx := range runes {
		shapeArabicRune(out, runes, idx)
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
