package pdf

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
)

// pdfNameSafe reports whether r is allowed inside a PDF name token.
func pdfNameSafe(r rune) bool {
	return (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
}

// pdfNameToken keeps only characters safe in a PDF name token.
func pdfNameToken(s string) string {
	var buf strings.Builder

	for _, r := range s {
		if pdfNameSafe(r) {
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
		var rangeLine [16]byte

		line := append(rangeLine[:0], '<')
		line = appendHex2(line, 0)
		line = append(line, '>', ' ', '<')
		line = appendHex2(line, byte(maxLatin1Code))
		line = append(line, '>', '\n')
		buf.Write(line)
	}

	buf.WriteString("endcodespacerange\n")

	// code → unicode (code == rune for both simple Latin-1 and Identity-H CIDs)
	maps := make([]unicodeMapEntry, 0, len(sub.glyphIDs))
	for r := range sub.glyphIDs {
		maps = append(maps, unicodeMapEntry{code: r, r: r})
	}

	sort.Slice(maps, func(a, b int) bool { return maps[a].code < maps[b].code })
	buf.Write(appendBfcharSection(nil, maps, codeBytes))

	buf.WriteString("endcmap\n")
	buf.WriteString("/CMapName currentdict /CMap defineresource pop\n")
	buf.WriteString("end\nend\n")
	d.setDict(ref, fmt.Sprintf("<< /Length %d >>", buf.Len()))
	d.setStream(ref, []byte(buf.String()))

	return ref
}

// unicodeMapEntry is one code-to-Unicode mapping in the ToUnicode CMap.
type unicodeMapEntry struct{ code, r rune }

// bfcharLineMax bounds one "<0000> <0000>\n" mapping line; the chunk overhead
// covers the "N beginbfchar\n" header and the "endbfchar\n" footer.
const (
	bfcharLineMax       = 14
	bfcharChunkOverhead = 32
)

// appendBfcharSection appends the bfchar chunks for maps in code order.
func appendBfcharSection(dst []byte, maps []unicodeMapEntry, codeBytes int) []byte {
	for start := 0; start < len(maps); start += cidToGIDChunk {
		end := start + cidToGIDChunk
		if end > len(maps) {
			end = len(maps)
		}

		chunk := make([]byte, 0, (end-start)*bfcharLineMax+bfcharChunkOverhead)
		chunk = strconv.AppendInt(chunk, int64(end-start), pdfNumBase)
		chunk = append(chunk, " beginbfchar\n"...)

		for _, entry := range maps[start:end] {
			chunk = append(chunk, '<')
			if codeBytes >= codeBytesTwo {
				chunk = appendHex4(chunk, entry.code)
			} else {
				chunk = appendHex2(chunk, byte(entry.code))
			}

			chunk = append(chunk, '>', ' ', '<')
			chunk = appendHex4(chunk, entry.r)
			chunk = append(chunk, '>', '\n')
		}

		chunk = append(chunk, "endbfchar\n"...)
		dst = append(dst, chunk...)
	}

	return dst
}

// runesKey builds a stable cache key for a rune set. It sorts used in place;
// callers pass the content-owned set only after all text emission is complete.
func runesKey(used []rune) string {
	if !slices.IsSorted(used) {
		slices.Sort(used)
	}

	unique := 0

	var previous rune

	for _, posX := range used {
		if unique > 0 && posX == previous {
			continue
		}

		previous = posX
		unique++
	}

	const (
		hexDigitsPerRune = 4
		hexBase          = 16
	)

	out := make([]byte, 0, unique*hexDigitsPerRune)

	previous = 0
	havePrevious := false

	for _, posX := range used {
		if havePrevious && posX == previous {
			continue
		}

		out = strconv.AppendUint(out, uint64(posX), hexBase)
		out = append(out, ',')
		previous = posX
		havePrevious = true
	}

	return string(out)
}
