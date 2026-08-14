package pdf

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
)

// fallbackFontName is the PostScript name substituted when a loaded face has
// none (the embedded default face uses the same name).
const fallbackFontName = "LiberationSans"

// needsType0 reports whether the rune set requires a CID/Type0 font
// (any code point outside the Latin-1 simple-font range).
func needsType0(used []rune) bool {
	for _, r := range used {
		if r > maxLatin1Code {
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
//
//nolint:cyclop,funlen // ensureFont switches between simple and Type0 embedding with cache-key precompute
func (d *Document) ensureFont(fnt *Font, name string, used []rune) (objRef, error) {
	if fnt == nil {
		return 0, errNilFont
	}

	if len(used) == 0 {
		used = []rune{' '}
	}

	baseName := fnt.PostScriptName
	if baseName == "" {
		baseName = fallbackFontName
	}

	// The finalize-time rune union makes the cache key and the Type0
	// decision identical for every page, so both are precomputed once per
	// document (unionFontRunes) and reused on the hot path.
	var key string

	var type0 bool

	if pre, ok := d.fontKeys[name]; ok && d.fontKeyFonts[name] == fnt {
		key = pre
		type0 = d.fontType0[name]
	} else {
		type0 = needsType0(used)
		mode := 0

		if type0 {
			mode = 1
		}

		key = fmt.Sprintf("v%d|%x|%s|%s", mode, fnt.fingerprint, baseName, runesKey(used))
	}

	if ref, ok := d.fontCache[key]; ok {
		return ref, nil
	}

	scope := subsetSimple
	if type0 {
		scope = subsetUnicode
	}

	sub, err := subsetFont(fnt, used, scope)
	if err != nil {
		return 0, err
	}

	// Arlington / ISO 32000: FontDescriptor /FontName must equal the
	// owning font's /BaseFont (CIDFontType2 parent for Type0 descendants).
	pdfName := pdfNameToken(baseName)
	if type0 {
		pdfName += "Identity"
	}

	fileRef, descRef := d.embedFontFile(fnt, sub, pdfName)

	var ref objRef
	if type0 {
		ref = d.emitType0(fnt, sub, pdfName, fileRef, descRef)
	} else {
		ref = d.emitSimple(fnt, sub, pdfName, descRef)
	}

	d.fontCache[key] = ref

	return ref, nil
}

// embedFontFile emits the FontFile2 stream and its FontDescriptor, shared by
// the simple and Type0 tails. fontName is the /FontName token without /.
func (d *Document) embedFontFile(fnt *Font, sub *subsetResult, fontName string) (objRef, objRef) {
	fileRef := d.newObject()
	descRef := d.newObject()

	raw := flateBytes(sub.data)
	d.setDict(fileRef, dict{}.add("/Length", strconv.Itoa(len(raw))).
		add("/Filter", "/FlateDecode").
		add("/Length1", strconv.Itoa(len(sub.data))).String())
	d.setStream(fileRef, raw)

	xMin, yMin, xMax, yMax := fnt.PDFBBox()
	// ISO 19005-3 §6.2.11.6: Nonsymbolic TrueType fonts must set bit 6 (32) and clear bit 3 (4).
	flags := 32

	italicAngle := int(fnt.italicAngle)
	if italicAngle == 0 && fnt.Italic() {
		italicAngle = defaultItalicAngle
	}

	if fnt.Italic() {
		flags |= 64
	}

	d.setDict(descRef, fmt.Sprintf(
		"<< /Type /FontDescriptor /FontName /%s /Flags %d /FontBBox [%d %d %d %d] "+
			"/ItalicAngle %d /Ascent %d /Descent %d /CapHeight %d /StemV 80 /FontFile2 %s >>",
		fontName, flags, xMin, yMin, xMax, yMax, italicAngle,
		fnt.PDFAscent(), fnt.PDFDescent(), fnt.PDFCapHeight(), fileRef))

	return fileRef, descRef
}

// emitSimple writes the simple TrueType font dict (WinAnsi single-byte).
func (d *Document) emitSimple(f *Font, sub *subsetResult, pdfName string, descRef objRef) objRef {
	first, last, widths := subsetWidths(sub, f.UnitsPerEm())
	wspace := make([]string, len(widths))

	for i, w := range widths {
		wspace[i] = num(w)
	}

	fontRef := d.newObject()
	d.setDict(fontRef, dict{}.add("/Type", "/Font").
		add("/Subtype", "/TrueType").
		add("/BaseFont", "/"+pdfName).
		add("/FirstChar", strconv.Itoa(first)).
		add("/LastChar", strconv.Itoa(last)).
		add("/Widths", "["+strings.Join(wspace, " ")+"]").
		add("/FontDescriptor", descRef.String()).
		add("/Encoding", "/WinAnsiEncoding").
		add("/ToUnicode", d.ensureToUnicode(sub, 1).String()).String())

	return fontRef
}

// emitType0 writes the CID-keyed Type0 font objects. Content streams must
// show text as Identity-H CIDs equal to Unicode code points (see pdfHexCIDs).
func (d *Document) emitType0(fnt *Font, sub *subsetResult, pdfName string, _ objRef, descRef objRef) objRef {
	cidRef := d.newObject()
	type0Ref := d.newObject()
	mapRef := d.newObject()

	cidMap, wParts := buildCIDMap(sub, fnt.UnitsPerEm())

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
		add("/ToUnicode", d.ensureToUnicode(sub, toUnicodeTwoByte).String()).String())

	return type0Ref
}

// buildCIDMap renders the CIDToGIDMap bytes and the sorted /W width runs.
func buildCIDMap(sub *subsetResult, unitsPerEm int16) ([]byte, []string) {
	// CIDToGIDMap: 2 bytes per CID from 0..maxCID.
	maxCID := 0
	for r := range sub.glyphIDs {
		if int(r) > maxCID {
			maxCID = int(r)
		}
	}

	cidMap := make([]byte, (maxCID+1)*cidBytesPerEntry)
	wspace := widthsInEm(sub, unitsPerEm)

	wParts := make([]string, 0, len(sub.glyphIDs))

	type rw struct {
		r rune
		w float64
	}

	rows := make([]rw, 0, len(sub.glyphIDs))

	for rVal, glob := range sub.glyphIDs {
		cidMap[int(rVal)*cidBytesPerEntry] = byte(glob >> cidGlyphHighShift)
		cidMap[int(rVal)*cidBytesPerEntry+1] = byte(glob)

		adv := 0.0
		if int(glob) < len(wspace) {
			adv = wspace[glob]
		}

		rows = append(rows, rw{rVal, adv})
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].r < rows[j].r })

	for _, row := range rows {
		wParts = append(wParts, fmt.Sprintf("%d [%s]", row.r, num(row.w)))
	}

	return cidMap, wParts
}
