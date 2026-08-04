package pdf

import "unicode"

// ShapeText applies best-effort text shaping before PDF emission.
// Without HarfBuzz we only reverse contiguous RTL (Arabic/Hebrew) runs so
// visual order is closer to expected; we do NOT perform joining, ligation,
// mark positioning, or Indic reordering. Callers must treat Arabic/Indic
// as unsupported for production claims.
func ShapeText(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	out := make([]rune, 0, len(runes))
	i := 0
	for i < len(runes) {
		if isRTLRune(runes[i]) {
			j := i + 1
			for j < len(runes) && (isRTLRune(runes[j]) || isRTLNeutral(runes[j])) {
				j++
			}
			// trim trailing neutrals back to LTR side
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
	case r >= 0x0590 && r <= 0x05FF: // Hebrew
		return true
	case r >= 0x0600 && r <= 0x06FF: // Arabic
		return true
	case r >= 0x0700 && r <= 0x074F: // Syriac
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
