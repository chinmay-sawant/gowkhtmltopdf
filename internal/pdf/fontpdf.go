package pdf

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// pdfNameToken keeps only characters safe in a PDF name token.
func pdfNameToken(s string) string {
	var buf strings.Builder

	for _, r := range s {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			buf.WriteRune(r)
		}
	}

	if buf.Len() == 0 {
		return "Font"
	}

	return buf.String()
}

// widthsInEm is the single home of the font-units→PDF-1000-em conversion,
// feeding both the simple /Widths array and the Type0 /W array. The result
// is indexed by subset glyph id.
func widthsInEm(sub *subsetResult, unitsPerEm int16) []float64 {
	upm := float64(unitsPerEm)
	if upm <= 0 {
		upm = 1000
	}

	wspace := make([]float64, len(sub.widths))
	for i, w := range sub.widths {
		// PDF glyph space: 1000 units = 1 em.
		wspace[i] = w * pdfUnitsPerEm / upm
	}

	return wspace
}

// subsetWidths returns (firstCode, lastCode, widths) with widths indexed by
// char code in PDF 1000-unit em space; codes without a glyph get 0.
func subsetWidths(sub *subsetResult, unitsPerEm int16) (int, int, []float64) {
	if len(sub.glyphIDs) == 0 {
		return 0, 0, nil
	}

	wspace := widthsInEm(sub, unitsPerEm)
	first, last := 0xFF, 0

	for r := range sub.glyphIDs {
		cur := int(r)
		if cur < first {
			first = cur
		}

		if cur > last {
			last = cur
		}
	}

	widths := make([]float64, last-first+1)

	for r, g := range sub.glyphIDs {
		if int(g) < len(wspace) {
			widths[int(r)-first] = wspace[g]
		}
	}

	return first, last, widths
}

// ensureToUnicode emits the ToUnicode CMap for a subset and returns its ref.
// codeBytes is 1 for simple (WinAnsi single-byte) or 2 for Identity-H CIDs.
func (d *Document) ensureToUnicode(sub *subsetResult, codeBytes int) objRef {
	ref := d.newObject()

	var buf strings.Builder

	buf.WriteString("/CIDInit /ProcSet findresource begin\n")
	buf.WriteString("12 dict begin\n")
	buf.WriteString("begincmap\n")
	buf.WriteString("/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n")
	buf.WriteString("/CMapName /Adobe-Identity-UCS def\n")
	buf.WriteString("/CMapType 2 def\n")
	buf.WriteString("1 begincodespacerange\n")

	if codeBytes >= codeBytesTwo {
		buf.WriteString("<0000> <FFFF>\n")
	} else {
		fmt.Fprintf(&buf, "<%02X> <%02X>\n", byte(0), byte(maxLatin1Code))
	}

	buf.WriteString("endcodespacerange\n")
	// code → unicode (code == rune for both simple Latin-1 and Identity-H CIDs)
	type m struct{ code, r rune }

	var maps []m
	for r := range sub.glyphIDs {
		maps = append(maps, m{code: r, r: r})
	}

	sort.Slice(maps, func(a, b int) bool { return maps[a].code < maps[b].code })

	for start := 0; start < len(maps); start += 100 {
		end := start + cidToGIDChunk
		if end > len(maps) {
			end = len(maps)
		}

		fmt.Fprintf(&buf, "%d beginbfchar\n", end-start)

		for _, mm := range maps[start:end] {
			if codeBytes >= codeBytesTwo {
				fmt.Fprintf(&buf, "<%04X> <%04X>\n", mm.code, mm.r)
			} else {
				fmt.Fprintf(&buf, "<%02X> <%04X>\n", mm.code, mm.r)
			}
		}

		buf.WriteString("endbfchar\n")
	}

	buf.WriteString("endcmap\n")
	buf.WriteString("/CMapName currentdict /CMap defineresource pop\n")
	buf.WriteString("end\nend\n")
	d.setDict(ref, fmt.Sprintf("<< /Length %d >>", buf.Len()))
	d.setStream(ref, []byte(buf.String()))

	return ref
}

// runesKey builds a stable cache key for a rune set.
func runesKey(used []rune) string {
	runVal := append([]rune(nil), used...)
	sort.Slice(runVal, func(i, j int) bool { return runVal[i] < runVal[j] })

	seen := map[rune]bool{}

	var strB bytes.Buffer

	for _, posX := range runVal {
		if seen[posX] {
			continue
		}

		seen[posX] = true

		fmt.Fprintf(&strB, "%x,", posX)
	}

	return strB.String()
}
