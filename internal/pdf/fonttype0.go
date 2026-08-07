package pdf

import (
	"fmt"
	"sort"
	"strconv"
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
// Type0 / CIDFontType2 with Identity-H and Unicode CIDs. One embed pipeline
// with a mode switch: the shared FontFile2+FontDescriptor constructor
// (embedFontFile) feeds both modes; only the rune→char-code mapping and the
// dict tails differ.
//
// ponytail: Type0+simple dual embed — both product-real (Latin-1 vs CJK/BMP).
func (d *Document) ensureFont(f *Font, used []rune) (objRef, error) {
	if len(used) == 0 {
		used = []rune{' '}
	}
	type0 := needsType0(used)
	baseName := f.PostScriptName
	if baseName == "" {
		baseName = "LiberationSans"
	}
	mode := 0
	if type0 {
		mode = 1
	}
	key := fmt.Sprintf("v%d|%x|%s|%s", mode, f.fingerprint, baseName, runesKey(used))
	if ref, ok := d.fontCache[key]; ok {
		return ref, nil
	}
	scope := subsetSimple
	if type0 {
		scope = subsetUnicode
	}
	sub, err := subsetFont(f, used, scope)
	if err != nil {
		return 0, err
	}
	pdfName := pdfNameToken(baseName)
	fileRef, descRef, err := d.embedFontFile(f, sub, pdfName)
	if err != nil {
		return 0, err
	}
	var ref objRef
	if type0 {
		ref, err = d.emitType0(f, sub, pdfName+"Identity", fileRef, descRef)
	} else {
		ref, err = d.emitSimple(f, sub, pdfName, descRef)
	}
	if err != nil {
		return 0, err
	}
	d.fontCache[key] = ref
	return ref, nil
}

// embedFontFile emits the FontFile2 stream and its FontDescriptor, shared by
// the simple and Type0 tails. fontName is the /FontName token without /.
func (d *Document) embedFontFile(f *Font, sub *subsetResult, fontName string) (fileRef, descRef objRef, err error) {
	fileRef = d.newObject()
	descRef = d.newObject()

	raw := flateBytes(sub.data)
	d.setDict(fileRef, dict{}.add("/Length", strconv.Itoa(len(raw))).
		add("/Filter", "/FlateDecode").
		add("/Length1", strconv.Itoa(len(sub.data))).String())
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
	d.setDict(descRef, fmt.Sprintf(
		"<< /Type /FontDescriptor /FontName /%s /Flags %d /FontBBox [%d %d %d %d] /ItalicAngle %d /Ascent %d /Descent %d /CapHeight %d /StemV 80 /FontFile2 %s >>",
		fontName, flags, xMin, yMin, xMax, yMax, italicAngle,
		f.PDFAscent(), f.PDFDescent(), f.PDFCapHeight(), fileRef))
	return fileRef, descRef, nil
}

// emitSimple writes the simple TrueType font dict (WinAnsi single-byte).
func (d *Document) emitSimple(f *Font, sub *subsetResult, pdfName string, descRef objRef) (objRef, error) {
	first, last, widths := subsetWidths(sub, f.UnitsPerEm())
	ws := make([]string, len(widths))
	for i, w := range widths {
		ws[i] = num(w)
	}
	fontRef := d.newObject()
	d.setDict(fontRef, dict{}.add("/Type", "/Font").
		add("/Subtype", "/TrueType").
		add("/BaseFont", "/"+pdfName).
		add("/FirstChar", strconv.Itoa(first)).
		add("/LastChar", strconv.Itoa(last)).
		add("/Widths", "["+strings.Join(ws, " ")+"]").
		add("/FontDescriptor", descRef.String()).
		add("/Encoding", "/WinAnsiEncoding").
		add("/ToUnicode", d.ensureToUnicode(sub, 1).String()).String())
	return fontRef, nil
}

// emitType0 writes the CID-keyed Type0 font objects. Content streams must
// show text as Identity-H CIDs equal to Unicode code points (see pdfHexCIDs).
func (d *Document) emitType0(f *Font, sub *subsetResult, pdfName string, fileRef, descRef objRef) (objRef, error) {
	cidRef := d.newObject()
	type0Ref := d.newObject()
	mapRef := d.newObject()

	// CIDToGIDMap: 2 bytes per CID from 0..maxCID.
	maxCID := 0
	for r := range sub.glyphIDs {
		if int(r) > maxCID {
			maxCID = int(r)
		}
	}
	cidMap := make([]byte, (maxCID+1)*2)
	ws := widthsInEm(sub, f.UnitsPerEm())
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
		if int(g) < len(ws) {
			adv = ws[g]
		}
		rows = append(rows, rw{r, adv})
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].r < rows[j].r })
	for _, row := range rows {
		wParts = append(wParts, fmt.Sprintf("%d [%s]", row.r, num(row.w)))
	}
	mapRaw := cidMap // keep uncompressed; some viewers mishandle Flate CIDToGIDMap
	d.setDict(mapRef, dict{}.add("/Length", strconv.Itoa(len(mapRaw))).String())
	d.setStream(mapRef, mapRaw)

	d.setDict(cidRef, dict{}.add("/Type", "/Font").
		add("/Subtype", "/CIDFontType2").
		add("/BaseFont", "/"+pdfName).
		add("/CIDSystemInfo", "<< /Registry (Adobe) /Ordering (Identity) /Supplement 0 >>").
		add("/FontDescriptor", descRef.String()).
		add("/DW", "500").
		add("/W", "["+strings.Join(wParts, " ")+"]").
		add("/CIDToGIDMap", mapRef.String()).String())

	d.setDict(type0Ref, dict{}.add("/Type", "/Font").
		add("/Subtype", "/Type0").
		add("/BaseFont", "/"+pdfName).
		add("/Encoding", "/Identity-H").
		add("/DescendantFonts", "["+cidRef.String()+"]").
		add("/ToUnicode", d.ensureToUnicode(sub, 2).String()).String())

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
