package pdf

import (
	"fmt"
	"sort"
	"strings"
)

// needsType0 reports whether the rune set requires a CID/Type0 font
// (any code point outside the Latin-1 simple-font range).
func needsType0(used []rune) bool {
	for _, r := range used {
		if r > 0xFF {
			return true
		}
	}
	return false
}

// ensureFont subsets f for runes and emits the font objects once per subset.
// Latin-1-only subsets use a simple TrueType font; any higher Unicode uses
// Type0 / CIDFontType2 with Identity-H and Unicode CIDs.
//
// ponytail: Type0+simple dual embed — both product-real (Latin-1 vs CJK/BMP).
func (d *Document) ensureFont(f *Font, used []rune) (string, error) {
	if len(used) == 0 {
		used = []rune{' '}
	}
	if needsType0(used) {
		return d.ensureFontType0(f, used)
	}
	return d.ensureFontSimple(f, used)
}

func (d *Document) ensureFontSimple(f *Font, used []rune) (string, error) {
	baseName := f.PostScriptName
	if baseName == "" {
		baseName = "LiberationSans"
	}
	key := "simple|" + baseName + "|" + runesKey(used)
	if ref, ok := d.fontCache[key]; ok {
		return ref, nil
	}
	sub, err := subsetFont(f, used, subsetSimple)
	if err != nil {
		return "", err
	}
	ef := &embeddedFont{subset: sub}
	ef.fontRef = d.newObject()
	ef.descRef = d.newObject()
	ef.ref = d.newObject()

	raw := flateBytes(sub.data)
	d.setDict(ef.ref, fmt.Sprintf("<< /Length %d /Filter /FlateDecode /Length1 %d >>",
		len(raw), len(sub.data)))
	d.setStream(ef.ref, raw)

	xMin, yMin, xMax, yMax := f.PDFBBox()
	flags := 32
	italicAngle := 0
	if f.Italic() {
		italicAngle = -12
		flags |= 64
	}
	if f.Bold() {
		flags |= 4
	}
	pdfName := pdfNameToken(baseName)
	d.setDict(ef.descRef, fmt.Sprintf(
		"<< /Type /FontDescriptor /FontName /%s /Flags %d /FontBBox [%d %d %d %d] /ItalicAngle %d /Ascent %d /Descent %d /CapHeight %d /StemV 80 /FontFile2 %s >>",
		pdfName, flags, xMin, yMin, xMax, yMax, italicAngle,
		f.PDFAscent(), f.PDFDescent(), f.PDFCapHeight(), ef.ref))

	first, last, widths := subsetWidths(sub, f.UnitsPerEm())
	ws := make([]string, len(widths))
	for i, w := range widths {
		ws[i] = num(w)
	}
	d.setDict(ef.fontRef, fmt.Sprintf(
		"<< /Type /Font /Subtype /TrueType /BaseFont /%s /FirstChar %d /LastChar %d /Widths [%s] /FontDescriptor %s /Encoding /WinAnsiEncoding /ToUnicode %s >>",
		pdfName, first, last, strings.Join(ws, " "), ef.descRef, d.ensureToUnicode(sub, 1)))

	d.fontCache[key] = ef.fontRef
	return ef.fontRef, nil
}

// ensureFontType0 embeds a CID-keyed Type0 font. Content streams must show
// text as Identity-H CIDs equal to Unicode code points (see pdfHexCIDs).
func (d *Document) ensureFontType0(f *Font, used []rune) (string, error) {
	baseName := f.PostScriptName
	if baseName == "" {
		baseName = "LiberationSans"
	}
	key := "type0|" + baseName + "|" + runesKey(used)
	if ref, ok := d.fontCache[key]; ok {
		return ref, nil
	}
	sub, err := subsetFont(f, used, subsetUnicode)
	if err != nil {
		return "", err
	}

	fileRef := d.newObject()
	descRef := d.newObject()
	cidRef := d.newObject()
	type0Ref := d.newObject()
	mapRef := d.newObject()

	raw := flateBytes(sub.data)
	d.setDict(fileRef, fmt.Sprintf("<< /Length %d /Filter /FlateDecode /Length1 %d >>",
		len(raw), len(sub.data)))
	d.setStream(fileRef, raw)

	xMin, yMin, xMax, yMax := f.PDFBBox()
	flags := 32
	italicAngle := 0
	if f.Italic() {
		italicAngle = -12
		flags |= 64
	}
	if f.Bold() {
		flags |= 4
	}
	pdfName := pdfNameToken(baseName) + "Identity"
	d.setDict(descRef, fmt.Sprintf(
		"<< /Type /FontDescriptor /FontName /%s /Flags %d /FontBBox [%d %d %d %d] /ItalicAngle %d /Ascent %d /Descent %d /CapHeight %d /StemV 80 /FontFile2 %s >>",
		pdfName, flags, xMin, yMin, xMax, yMax, italicAngle,
		f.PDFAscent(), f.PDFDescent(), f.PDFCapHeight(), fileRef))

	// CIDToGIDMap: 2 bytes per CID from 0..maxCID.
	maxCID := 0
	for r := range sub.glyphIDs {
		if int(r) > maxCID {
			maxCID = int(r)
		}
	}
	cidMap := make([]byte, (maxCID+1)*2)
	upm := float64(f.UnitsPerEm())
	if upm <= 0 {
		upm = 1000
	}
	var wParts []string
	type rw struct {
		r rune
		w float64
	}
	var rows []rw
	for r, g := range sub.glyphIDs {
		cidMap[int(r)*2] = byte(g >> 8)
		cidMap[int(r)*2+1] = byte(g)
		adv := 0.0
		if int(g) < len(sub.widths) {
			adv = sub.widths[g] * 1000 / upm
		}
		rows = append(rows, rw{r, adv})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].r < rows[j].r })
	for _, row := range rows {
		wParts = append(wParts, fmt.Sprintf("%d [%s]", row.r, num(row.w)))
	}
	mapRaw := cidMap // keep uncompressed; some viewers mishandle Flate CIDToGIDMap
	d.setDict(mapRef, fmt.Sprintf("<< /Length %d >>", len(mapRaw)))
	d.setStream(mapRef, mapRaw)

	d.setDict(cidRef, fmt.Sprintf(
		"<< /Type /Font /Subtype /CIDFontType2 /BaseFont /%s /CIDSystemInfo << /Registry (Adobe) /Ordering (Identity) /Supplement 0 >> /FontDescriptor %s /DW 500 /W [%s] /CIDToGIDMap %s >>",
		pdfName, descRef, strings.Join(wParts, " "), mapRef))

	d.setDict(type0Ref, fmt.Sprintf(
		"<< /Type /Font /Subtype /Type0 /BaseFont /%s /Encoding /Identity-H /DescendantFonts [%s] /ToUnicode %s >>",
		pdfName, cidRef, d.ensureToUnicode(sub, 2)))

	d.fontCache[key] = type0Ref
	return type0Ref, nil
}

// pdfHexCIDs encodes s as an Identity-H hex string of Unicode CIDs.
func pdfHexCIDs(s string) string {
	var b strings.Builder
	b.WriteByte('<')
	for _, r := range s {
		if r > 0xFFFF {
			r = '?'
		}
		fmt.Fprintf(&b, "%04X", r)
	}
	b.WriteByte('>')
	return b.String()
}
