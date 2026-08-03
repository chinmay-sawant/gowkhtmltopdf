package pdf

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
)

// fontKey identifies a subset in the document font cache.
type fontKey struct {
	runes string // joined runes the subset covers
}

// embeddedFont is one subset embedded in the document.
type embeddedFont struct {
	subset  *subsetResult
	ref     string // /FontFile2 object ref
	descRef string // /FontDescriptor object ref
	fontRef string // font dict object ref
}

// ensureFont subsets f for runes and emits the font objects once per subset.
// Returns the font dict ref to use in page resources.
func (d *Document) ensureFont(f *Font, used []rune) (string, error) {
	if len(used) == 0 {
		used = []rune{' '}
	}
	key := runesKey(used)
	if ref, ok := d.fontCache[key]; ok {
		return ref, nil
	}
	sub, err := subsetFont(f, used)
	if err != nil {
		return "", err
	}
	ef := &embeddedFont{subset: sub}
	ef.fontRef = d.newObject()
	ef.descRef = d.newObject()
	ef.ref = d.newObject()

	// FontFile2 stream (flate-compressed subset TTF)
	raw := flateBytes(sub.data)
	d.setDict(ef.ref, fmt.Sprintf("<< /Length %d /Filter /FlateDecode /Length1 %d >>",
		len(raw), len(sub.data)))
	d.setStream(ef.ref, raw)

	// FontDescriptor
	xMin, yMin, xMax, yMax := f.BBox()
	flags := 32 // nonsymbolic
	italicAngle := 0
	if f.Italic() {
		italicAngle = -12
		flags |= 64
	}
	if f.Bold() {
		flags |= 4
	}
	d.setDict(ef.descRef, fmt.Sprintf(
		"<< /Type /FontDescriptor /FontName /LiberationSans /Flags %d /FontBBox [%d %d %d %d] /ItalicAngle %d /Ascent %d /Descent %d /CapHeight %d /StemV 80 /FontFile2 %s >>",
		flags, xMin, yMin, xMax, yMax, italicAngle,
		f.Ascent(), f.Descent(), f.CapHeight(), ef.ref))

	// font dict: simple TrueType; char codes are Latin-1 rune values resolved
	// by the subset cmap
	first, last, widths := subsetWidths(sub)
	ws := make([]string, len(widths))
	for i, w := range widths {
		ws[i] = num(w)
	}
	d.setDict(ef.fontRef, fmt.Sprintf(
		"<< /Type /Font /Subtype /TrueType /BaseFont /LiberationSans /FirstChar %d /LastChar %d /Widths [%s] /FontDescriptor %s /Encoding /WinAnsiEncoding /ToUnicode %s >>",
		first, last, strings.Join(ws, " "), ef.descRef, d.ensureToUnicode(sub)))

	d.fontCache[key] = ef.fontRef
	return ef.fontRef, nil
}

// subsetWidths returns (firstCode, lastCode, widths) with widths indexed by
// char code; codes without a glyph get 0.
func subsetWidths(sub *subsetResult) (int, int, []float64) {
	if len(sub.glyphIDs) == 0 {
		return 0, 0, nil
	}
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
		if int(g) < len(sub.widths) {
			widths[int(r)-first] = sub.widths[g]
		}
	}
	return first, last, widths
}

// ensureToUnicode emits the ToUnicode CMap for a subset and returns its ref.
func (d *Document) ensureToUnicode(sub *subsetResult) string {
	ref := d.newObject()
	var b strings.Builder
	b.WriteString("/CIDInit /ProcSet findresource begin\n")
	b.WriteString("12 dict begin\n")
	b.WriteString("begincmap\n")
	b.WriteString("/CIDSystemInfo << /Registry (Adobe) /Ordering (UCS) /Supplement 0 >> def\n")
	b.WriteString("/CMapName /Adobe-Identity-UCS def\n")
	b.WriteString("/CMapType 2 def\n")
	b.WriteString("1 begincodespacerange\n")
	fmt.Fprintf(&b, "<%02X> <%02X>\n", byte(0), byte(0xFF))
	b.WriteString("endcodespacerange\n")
	// group mappings: code → unicode (code == rune value for Latin-1)
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
			fmt.Fprintf(&b, "<%02X> <%04X>\n", mm.code, mm.r)
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
