package pdf

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// pdfNameToken keeps only characters safe in a PDF name token.
func pdfNameToken(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "Font"
	}
	return b.String()
}

// widthsInEm is the single home of the font-units→PDF-1000-em conversion,
// feeding both the simple /Widths array and the Type0 /W array. The result
// is indexed by subset glyph id.
func widthsInEm(sub *subsetResult, unitsPerEm int16) []float64 {
	upm := float64(unitsPerEm)
	if upm <= 0 {
		upm = 1000
	}
	ws := make([]float64, len(sub.widths))
	for i, w := range sub.widths {
		// PDF glyph space: 1000 units = 1 em.
		ws[i] = w * 1000 / upm
	}
	return ws
}

// subsetWidths returns (firstCode, lastCode, widths) with widths indexed by
// char code in PDF 1000-unit em space; codes without a glyph get 0.
func subsetWidths(sub *subsetResult, unitsPerEm int16) (int, int, []float64) {
	if len(sub.glyphIDs) == 0 {
		return 0, 0, nil
	}
	ws := widthsInEm(sub, unitsPerEm)
	first, last := 0xFF, 0
	for r := range sub.glyphIDs {
		c := int(r)
		if c < first {
			first = c
		}
		if c > last {
			last = c
		}
	}
	widths := make([]float64, last-first+1)
	for r, g := range sub.glyphIDs {
		if int(g) < len(ws) {
			widths[int(r)-first] = ws[g]
		}
	}
	return first, last, widths
}

// ensureToUnicode emits the ToUnicode CMap for a subset and returns its ref.
// codeBytes is 1 for simple (WinAnsi single-byte) or 2 for Identity-H CIDs.
func (d *Document) ensureToUnicode(sub *subsetResult, codeBytes int) objRef {
	ref := d.newObject()
	var b strings.Builder
	b.WriteString("/CIDInit /ProcSet findresource begin\n")
	b.WriteString("12 dict begin\n")
	b.WriteString("begincmap\n")
	b.WriteString("/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n")
	b.WriteString("/CMapName /Adobe-Identity-UCS def\n")
	b.WriteString("/CMapType 2 def\n")
	b.WriteString("1 begincodespacerange\n")
	if codeBytes >= 2 {
		b.WriteString("<0000> <FFFF>\n")
	} else {
		fmt.Fprintf(&b, "<%02X> <%02X>\n", byte(0), byte(0xFF))
	}
	b.WriteString("endcodespacerange\n")
	// code → unicode (code == rune for both simple Latin-1 and Identity-H CIDs)
	type m struct{ code, r rune }
	var maps []m
	for r := range sub.glyphIDs {
		maps = append(maps, m{code: r, r: r})
	}
	sort.Slice(maps, func(a, b int) bool { return maps[a].code < maps[b].code })
	for start := 0; start < len(maps); start += 100 {
		end := start + 100
		if end > len(maps) {
			end = len(maps)
		}
		fmt.Fprintf(&b, "%d beginbfchar\n", end-start)
		for _, mm := range maps[start:end] {
			if codeBytes >= 2 {
				fmt.Fprintf(&b, "<%04X> <%04X>\n", mm.code, mm.r)
			} else {
				fmt.Fprintf(&b, "<%02X> <%04X>\n", mm.code, mm.r)
			}
		}
		b.WriteString("endbfchar\n")
	}
	b.WriteString("endcmap\n")
	b.WriteString("/CMapName currentdict /CMap defineresource pop\n")
	b.WriteString("end\nend\n")
	d.setDict(ref, fmt.Sprintf("<< /Length %d >>", b.Len()))
	d.setStream(ref, []byte(b.String()))
	return ref
}

// runesKey builds a stable cache key for a rune set.
func runesKey(used []rune) string {
	r := append([]rune(nil), used...)
	sort.Slice(r, func(i, j int) bool { return r[i] < r[j] })
	seen := map[rune]bool{}
	var sb bytes.Buffer
	for _, x := range r {
		if seen[x] {
			continue
		}
		seen[x] = true
		fmt.Fprintf(&sb, "%x,", x)
	}
	return sb.String()
}
