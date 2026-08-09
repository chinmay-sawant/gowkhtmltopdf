package pdf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sort"
)

var (
	errSubsetBadHead = errors.New("font: bad head in subset")
	errSubsetBadMaxp = errors.New("font: bad maxp in subset")
	errSubsetBadHhea = errors.New("font: bad hhea in subset")
	errSubsetNoMaps  = errors.New("font: empty cmap mappings")
)

// subsetResult is a minimal TrueType subset for PDF embedding.
type subsetResult struct {
	data     []byte
	runes    []rune          // sorted runes that got glyphs
	glyphIDs map[rune]uint16 // rune -> new glyph id (1..n)
	widths   []float64       // advance widths in font units, index = new glyph id
}

// subsetScope selects which runes enter the subset cmap / glyph map.
// Simple fonts map Latin-1 only; Type0 needs the full BMP for Identity-H CIDs.
type subsetScope int

const (
	subsetSimple  subsetScope = iota // Latin-1 single-byte PDF fonts
	subsetUnicode                    // Unicode BMP (Type0 / CIDToGIDMap)
)

// subsetFont builds a minimal TrueType font containing only the glyphs used
// by runes. The rebuilt cmap maps accepted runes to renumbered glyph ids.
// scope=subsetSimple keeps Latin-1 only; scope=subsetUnicode keeps BMP
// (codes above U+FFFF are skipped).
func subsetFont(fnt *Font, used []rune, scope subsetScope) (*subsetResult, error) {
	accept := func(r rune) bool {
		if scope == subsetSimple {
			return simpleFontRune(r)
		}

		return r <= maxBMPCode
	}

	glyphSet := collectUsedGlyphs(fnt, used, accept)
	glyphs := sortedGlyphs(glyphSet)
	oldToNew := make(map[uint16]uint16, len(glyphs))
	advances, lsbs, outlines := collectGlyphData(fnt, glyphs, oldToNew)
	outlines = cloneOutlines(outlines, oldToNew)

	res := buildSubsetResult(fnt, used, accept, oldToNew)

	sub := &subsetter{
		f:        fnt,
		glyphs:   glyphs,
		outlines: outlines,
		advances: advances,
		lsbs:     lsbs,
		mappings: glyphMappings(res.glyphIDs),
	}

	data, err := sub.build()
	if err != nil {
		return nil, err
	}

	res.data = data
	res.widths = make([]float64, len(advances))

	for i, a := range advances {
		res.widths[i] = float64(a)
	}

	return res, nil
}

// cloneOutlines strips hinting from cloned outlines and remaps composite
// components — source tables are never mutated.
func cloneOutlines(outlines [][]byte, oldToNew map[uint16]uint16) [][]byte {
	cloned := make([][]byte, len(outlines))
	for i, o := range outlines {
		cloned[i] = stripGlyphHints(bytes.Clone(o))
		remapComposite(cloned[i], oldToNew)
	}

	return cloned
}

// buildSubsetResult maps the accepted runes to their renumbered glyph ids.
func buildSubsetResult(fnt *Font, used []rune, accept func(rune) bool, oldToNew map[uint16]uint16) *subsetResult {
	res := &subsetResult{glyphIDs: map[rune]uint16{}} //nolint:exhaustruct // intentional zero-value fields

	for _, rVal := range used {
		if !accept(rVal) {
			continue
		}

		old := fnt.GlyphID(rVal)
		if old == 0 {
			continue
		}

		res.glyphIDs[rVal] = oldToNew[old]
	}

	res.runes = sortedRunes(res.glyphIDs)

	return res
}

// glyphMappings builds the sorted rune→glyph mapping table for the cmap.
func glyphMappings(glyphIDs map[rune]uint16) []codeGlyph {
	mappings := make([]codeGlyph, 0, len(glyphIDs))

	for r, g := range glyphIDs {
		mappings = append(mappings, codeGlyph{code: uint16(r), glyph: g})
	}

	sort.Slice(mappings, func(i, j int) bool { return mappings[i].code < mappings[j].code })

	return mappings
}

func collectUsedGlyphs(fnt *Font, used []rune, accept func(rune) bool) map[uint16]bool {
	// .notdef always included
	glyphSet := map[uint16]bool{0: true}

	for _, r := range used {
		if !accept(r) {
			continue
		}

		g := fnt.GlyphID(r)
		if g == 0 {
			continue
		}

		collectGlyph(fnt, g, glyphSet)
	}

	return glyphSet
}

func sortedGlyphs(set map[uint16]bool) []uint16 {
	// sort glyphs by original id for deterministic output
	glyphs := make([]uint16, 0, len(set))
	for g := range set {
		glyphs = append(glyphs, g)
	}

	sort.Slice(glyphs, func(i, j int) bool { return glyphs[i] < glyphs[j] })

	return glyphs
}

func sortedRunes(m map[rune]uint16) []rune {
	out := make([]rune, 0, len(m))
	for r := range m {
		out = append(out, r)
	}

	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })

	return out
}

func collectGlyphData(fnt *Font, glyphs []uint16, oldToNew map[uint16]uint16) ([]int32, []int16, [][]byte) {
	advances := make([]int32, 0, len(glyphs))
	lsbs := make([]int16, 0, len(glyphs))
	outlines := make([][]byte, len(glyphs))

	for newID, old := range glyphs {
		oldToNew[old] = uint16(newID) //nolint:gosec // newID < len(glyphs) <= numGlyphs

		if int(old) < len(fnt.advance) {
			advances = append(advances, fnt.advance[old])
		} else {
			advances = append(advances, 0)
		}

		if int(old) < len(fnt.lsb) {
			lsbs = append(lsbs, fnt.lsb[old])
		} else {
			lsbs = append(lsbs, 0)
		}

		outlines[newID] = fnt.glyphOutline(old)
	}

	return advances, lsbs, outlines
}

// collectGlyph adds g and (for composites) all referenced children.
func collectGlyph(fnt *Font, glob uint16, set map[uint16]bool) {
	if set[glob] {
		return
	}

	set[glob] = true
	for _, c := range fnt.compositeGlyphIDs(glob) {
		collectGlyph(fnt, c, set)
	}
}

// stripGlyphHints removes TrueType hinting bytecode from a glyf outline.
// Subsets omit fpgm/prep/cvt, so leftover instructions can garble CJK
// composites in PDF viewers (broken 東京都 etc.).
func stripGlyphHints(buf []byte) []byte {
	if len(buf) < glyfHeaderSize {
		return buf
	}

	numContours := int16(binary.BigEndian.Uint16(buf[0:2])) //nolint:gosec // numContours is int16 per glyf spec
	if numContours < 0 {
		return stripCompositeHints(buf)
	}

	if numContours == 0 {
		return buf
	}

	n := int(numContours)
	if len(buf) < 10+2*n+2 {
		return buf
	}

	insPos := glyfHeaderSize + uint16Bytes*n
	insLen := int(binary.BigEndian.Uint16(buf[insPos:]))

	after := insPos + uint16Bytes + insLen
	if after > len(buf) {
		return buf
	}

	if insLen == 0 {
		return buf
	}

	out := make([]byte, 0, len(buf)-insLen)
	out = append(out, buf[:insPos]...)
	out = append(out, 0, 0) // instructionLength = 0
	out = append(out, buf[after:]...)

	return out
}

func stripCompositeHints(buf []byte) []byte {
	pos := 10

	var lastFlagsAt int

	for {
		if pos+4 > len(buf) {
			return buf
		}

		lastFlagsAt = pos
		flags := binary.BigEndian.Uint16(buf[pos : pos+2])
		pos += 4

		if flags&0x0001 != 0 {
			pos += 4
		} else {
			pos += 2
		}

		switch {
		case flags&0x0008 != 0:
			pos += 2
		case flags&0x0040 != 0:
			pos += 4
		case flags&0x0080 != 0:
			pos += 8
		}

		if flags&0x0020 == 0 {
			break
		}
	}

	if pos > len(buf) {
		return buf
	}

	flags := binary.BigEndian.Uint16(buf[lastFlagsAt : lastFlagsAt+2])
	if flags&glyfHaveInstructions == 0 { // WE_HAVE_INSTRUCTIONS
		// Drop any accidental trailing bytes past the component list.
		return bytes.Clone(buf[:pos])
	}

	out := bytes.Clone(buf[:pos])
	binary.BigEndian.PutUint16(out[lastFlagsAt:lastFlagsAt+2], flags&^glyfHaveInstructions)

	return out
}

// nextComponentPos returns the position just past the current composite
// component (after its argument words and transformation values).
func nextComponentPos(b []byte, pos int) int {
	flags := binary.BigEndian.Uint16(b[pos : pos+2])
	pos += 4

	if flags&0x0001 != 0 { // ARG_1_AND_2_ARE_WORDS
		pos += 4
	} else {
		pos += 2
	}

	switch {
	case flags&0x0008 != 0: // WE_HAVE_A_SCALE
		pos += 2
	case flags&0x0040 != 0: // WE_HAVE_AN_X_AND_Y_SCALE
		pos += 4
	case flags&0x0080 != 0: // WE_HAVE_A_TWO_BY_TWO
		pos += 8
	}

	return pos
}

// remapComposite rewrites composite component glyph ids in a glyf outline.
func remapComposite(buf []byte, oldToNew map[uint16]uint16) {
	if len(buf) < glyfHeaderSize {
		return
	}

	numContours := int16(binary.BigEndian.Uint16(buf[0:2])) //nolint:gosec // numContours is int16 per glyf spec
	if numContours >= 0 {
		return
	}

	pos := 10

	for {
		if pos+4 > len(buf) {
			break
		}

		flags := binary.BigEndian.Uint16(buf[pos : pos+2])

		old := binary.BigEndian.Uint16(buf[pos+2 : pos+4])
		if n, ok := oldToNew[old]; ok {
			binary.BigEndian.PutUint16(buf[pos+2:pos+4], n)
		}

		pos = nextComponentPos(buf, pos)

		if flags&0x0020 == 0 { // MORE_COMPONENTS
			break
		}
	}
}

// subsetter assembles the subset font file.
type codeGlyph struct {
	code  uint16
	glyph uint16
}

type subsetter struct {
	f        *Font
	glyphs   []uint16
	outlines [][]byte
	advances []int32
	lsbs     []int16
	mappings []codeGlyph // sorted by code
}

//nolint:funlen // one sequential sfnt assembly: tables, glyph data, loca, checksums
func (s *subsetter) build() ([]byte, error) {
	numGlyphs := len(s.glyphs)
	// long loca always (format 1). TrueType requires each glyph to start on a
	// 4-byte boundary; odd/unaligned glyf offsets corrupt CJK composites in
	// PDFium and other viewers (garbled 東京都 etc.).
	loca := make([]uint32, numGlyphs+1)
	padded := padOutlines(s.outlines, loca)

	// hmtx: advance (2) + lsb (2) per glyph
	hmtx := make([]byte, len(s.advances)*bytesPerHMetric)
	for idx, a := range s.advances {
		binary.BigEndian.PutUint16(
			hmtx[idx*bytesPerHMetric:],
			uint16(a), //nolint:gosec // advance is int32 stored as uint16 per hmtx spec
		)

		lsb := int16(0)
		if idx < len(s.lsbs) {
			lsb = s.lsbs[idx]
		}

		binary.BigEndian.PutUint16(
			hmtx[idx*bytesPerHMetric+uint16Bytes:],
			uint16(lsb), //nolint:gosec // lsb is int16 stored as uint16 per hmtx spec
		)
	}

	// cmap: rune codes → renumbered glyph ids
	cmap, err := unicodeCmap4(s.mappings)
	if err != nil {
		return nil, err
	}

	// head: copy original, patch indexToLocFormat=1
	head := bytes.Clone(s.f.tables["head"])
	if len(head) < headMinSize {
		return nil, errSubsetBadHead
	}

	binary.BigEndian.PutUint16(head[50:52], 1)

	// maxp: copy original, patch numGlyphs
	maxp := bytes.Clone(s.f.tables["maxp"])
	if len(maxp) < maxpMinSize {
		return nil, errSubsetBadMaxp
	}

	binary.BigEndian.PutUint16(maxp[4:6], uint16(numGlyphs)) //nolint:gosec // subset size is bounded by source numGlyphs

	// hhea: copy, patch numberOfHMetrics
	hhea := bytes.Clone(s.f.tables["hhea"])
	if len(hhea) < hheaMinSize {
		return nil, errSubsetBadHhea
	}

	binary.BigEndian.PutUint16(hhea[34:36], uint16(numGlyphs)) //nolint:gosec // subset size is bounded by source numGlyphs

	glyf := new(bytes.Buffer)
	for _, o := range padded {
		glyf.Write(o)
	}

	tables := []struct {
		tag  string
		data []byte
	}{
		{"head", head},
		{"hhea", hhea},
		{"maxp", maxp},
		{"hmtx", hmtx},
		{"cmap", cmap},
		{"loca", encodeUint32Slice(loca)},
		{"glyf", glyf.Bytes()},
		{"OS/2", cloneTable(s.f, "OS/2")},
		{"post", cloneTable(s.f, "post")},
	}

	return buildFontFile(tables)
}

// padOutlines aligns each outline to 4 bytes and records the running loca.
func padOutlines(outlines [][]byte, loca []uint32) [][]byte {
	cur := 0
	padded := make([][]byte, len(outlines))

	for idx, o := range outlines {
		loca[idx] = uint32(cur) //nolint:gosec // cumulative glyf size stays far below uint32

		page := bytes.Clone(o)
		for len(page)%4 != 0 {
			page = append(page, 0)
		}

		padded[idx] = page
		cur += len(page)
	}

	loca[len(outlines)] = uint32(cur) //nolint:gosec // cumulative glyf size stays far below uint32

	return padded
}

func cloneTable(f *Font, tag string) []byte {
	if t, ok := f.tables[tag]; ok {
		return bytes.Clone(t)
	}

	return nil
}

func encodeUint32Slice(v []uint32) []byte {
	b := make([]byte, len(v)*uint32Bytes)
	for i, x := range v {
		binary.BigEndian.PutUint32(b[i*uint32Bytes:], x)
	}

	return b
}

// unicodeCmap4 builds a cmap with one format-4 subtable mapping codes to
// renumbered glyph ids. Consecutive codes mapping to consecutive glyphs are
// coalesced into segments (constant delta).
func unicodeCmap4(mappings []codeGlyph) ([]byte, error) {
	if len(mappings) == 0 {
		return nil, errSubsetNoMaps
	}

	segs := buildCmap4Segs(mappings)
	segCount := len(segs)
	length := cmapFormat4LenBase + cmapFormat4SegStride*segCount // 14-byte header + reservedPad + 4 arrays
	buf := make([]byte, length)
	binary.BigEndian.PutUint16(buf[0:2], cmapFormat4)                  // format
	binary.BigEndian.PutUint16(buf[2:4], uint16(length))               //nolint:gosec // segCount bounded by BMP codes
	binary.BigEndian.PutUint16(buf[6:8], uint16(segCount*uint16Bytes)) //nolint:gosec // segCount bounded by BMP codes

	maxPow := 1
	for maxPow*2 <= segCount {
		maxPow *= 2
	}

	//nolint:gosec // segCount bounded by BMP codes // searchRange
	binary.BigEndian.PutUint16(buf[8:10], uint16(maxPow*uint16Bytes))
	//nolint:gosec // segCount bounded by BMP codes // entrySelector
	binary.BigEndian.PutUint16(buf[10:12], uint16(maxPow))
	//nolint:gosec // segCount bounded by BMP codes // rangeShift
	binary.BigEndian.PutUint16(buf[12:14], uint16(segCount*2-maxPow*2))

	writeCmap4Arrays(buf, segs)
	// wrap in cmap table: version, numTables, (3,1) subtable record
	out := make([]byte, 0, sfntOffsetTableSize+length)
	out = append(out, 0, 0, 0, 1)
	out = append(out, 0, cmapPlatformWin, 0, cmapWinUnicodeBMP)

	var rec [4]byte

	binary.BigEndian.PutUint32(rec[:], sfntOffsetTableSize)
	out = append(out, rec[:]...)
	out = append(out, buf...)

	return out, nil
}

type cmap4Seg struct {
	start, end, delta uint16
}

// buildCmap4Segs coalesces consecutive (code -> code+1, glyph -> glyph+1)
// mappings into segments with a constant delta, plus the 0xFFFF sentinel.
func buildCmap4Segs(mappings []codeGlyph) []cmap4Seg {
	segs := make([]cmap4Seg, 0, len(mappings)/2+1)

	for idx := 0; idx < len(mappings); {
		jdx := idx
		for jdx+1 < len(mappings) &&
			mappings[jdx+1].code == mappings[jdx].code+1 &&
			mappings[jdx+1].glyph == mappings[jdx].glyph+1 {
			jdx++
		}

		delta := (int(mappings[idx].glyph) - int(mappings[idx].code)) & maxUint16Val
		//nolint:gosec // masked to uint16 above
		segs = append(segs, cmap4Seg{mappings[idx].code, mappings[jdx].code, uint16(delta)})
		idx = jdx + 1
	}

	return append(segs, cmap4Seg{0xFFFF, 0xFFFF, 1}) // sentinel
}

func writeCmap4Arrays(buf []byte, segs []cmap4Seg) {
	segCount := len(segs)
	endOff := 14
	startOff := endOff + uint16Bytes*segCount + uint16Bytes
	deltaOff := startOff + uint16Bytes*segCount
	rangeOff := deltaOff + uint16Bytes*segCount

	for i, s := range segs {
		binary.BigEndian.PutUint16(buf[endOff+i*2:], s.end)
		binary.BigEndian.PutUint16(buf[startOff+i*2:], s.start)
		binary.BigEndian.PutUint16(buf[deltaOff+i*2:], s.delta)
		binary.BigEndian.PutUint16(buf[rangeOff+i*2:], 0)
	}
}

// simpleFontRune reports whether r can be encoded as a single-byte char code
// in a simple PDF font (Latin-1 range; Type0/CID is deferred).
func simpleFontRune(r rune) bool { return r >= 0 && r <= 0xFF }

// buildFontFile assembles an sfnt with the given tables, sorted by tag, with
// proper checksumAdjustment in head.
func buildFontFile(tables []struct {
	tag  string
	data []byte
},
) ([]byte, error) {
	// drop nil tables
	tmp := make([]struct {
		tag  string
		data []byte
	}, 0, len(tables))

	for _, x := range tables {
		if x.data != nil {
			tmp = append(tmp, x)
		}
	}

	sort.Slice(tmp, func(i, j int) bool { return tmp[i].tag < tmp[j].tag })
	num := len(tmp)
	// compute head checksum adjustment: total file length must be 0 mod 2^32
	dirLen := sfntOffsetTableSize + sfntTableRecordSize*num
	// align each table to 4 bytes
	total := dirLen
	aligned := make([]int, num)

	for i, x := range tmp {
		pad := (sfntTableAlign - total%sfntTableAlign) % sfntTableAlign
		total += pad
		aligned[i] = pad
		total += len(x.data)
	}

	headIdx := -1

	for i, x := range tmp {
		if x.tag == "head" {
			headIdx = i
		}
	}

	if headIdx >= 0 {
		return patchHeadChecksum(tmp, aligned, headIdx)
	}

	return assembleFile(tmp, aligned), nil
}

// patchHeadChecksum lays out the file, then rewrites head.checksumAdjustment
// so the whole file checksum equals sfntHeadCheckAdj.
func patchHeadChecksum(tmp []struct {
	tag  string
	data []byte
},
	aligned []int,
	headIdx int,
) ([]byte, error) {
	// checksum of the file with checksumAdjustment zeroed
	zeroed := make([]byte, len(tmp[headIdx].data))
	copy(zeroed, tmp[headIdx].data)
	copy(zeroed[8:12], []byte{0, 0, 0, 0})
	tmp[headIdx].data = zeroed
	zeroedSum := checksum(zeroed)

	// layout the whole file to compute checksum
	full := assembleFile(tmp, aligned)
	sum := checksum(full)
	// place adjustment such that the final file sums to 0xB1B0AFBA.
	// The head checksum in the directory is kept at the zeroed-head
	// value, so the adjustment only shifts the sum once.
	adj := sfntHeadCheckAdj - sum
	adjusted := bytes.Clone(zeroed)
	binary.BigEndian.PutUint32(adjusted[8:12], adj)
	tmp[headIdx].data = adjusted
	full = assembleFile(tmp, aligned)
	// freeze the directory entry for head to the zeroed-head checksum
	for i := range tmp {
		if tmp[i].tag == "head" {
			rec := sfntOffsetTableSize + sfntTableRecordSize*i + uint32Bytes
			binary.BigEndian.PutUint32(full[rec:rec+4], zeroedSum)

			break
		}
	}

	return full, nil
}

func assembleFile(tables []struct {
	tag  string
	data []byte
}, aligned []int,
) []byte {
	num := len(tables)
	buf := new(bytes.Buffer)
	buf.Write([]byte{0, 1, 0, 0})
	writeSFNTHeader(buf, num)

	// directory
	offset := sfntOffsetTableSize + sfntTableRecordSize*num
	for i, posX := range tables {
		offset += aligned[i] // padding before table i

		buf.WriteString(posX.tag)

		var cs [4]byte

		binary.BigEndian.PutUint32(cs[:], checksum(posX.data))
		buf.Write(cs[:])

		var off [4]byte

		binary.BigEndian.PutUint32(off[:], uint32(offset)) //nolint:gosec // sfnt offsets stay far below uint32
		buf.Write(off[:])

		var tlen [4]byte

		binary.BigEndian.PutUint32(tlen[:], uint32(len(posX.data))) //nolint:gosec // table sizes stay far below uint32
		buf.Write(tlen[:])

		offset += len(posX.data)
	}
	// table data with alignment
	for i, x := range tables {
		for range aligned[i] {
			buf.WriteByte(0)
		}

		buf.Write(x.data)
	}

	return buf.Bytes()
}

func writeSFNTHeader(buf *bytes.Buffer, num int) {
	var numT [2]byte

	binary.BigEndian.PutUint16(numT[:], uint16(num)) //nolint:gosec // sfnt table count is small
	buf.Write(numT[:])
	// searchRange, entrySelector, rangeShift
	maxPow := 1
	sel := 0

	for maxPow*2 <= num {
		maxPow *= 2
		sel++
	}

	var sr [2]byte

	binary.BigEndian.PutUint16(sr[:], uint16(maxPow*sfntSearchRangeMul)) //nolint:gosec // table count is small
	buf.Write(sr[:])

	var es [2]byte

	binary.BigEndian.PutUint16(es[:], uint16(sel)) //nolint:gosec // table count is small
	buf.Write(es[:])

	var rs [2]byte

	//nolint:gosec // table count is small
	binary.BigEndian.PutUint16(rs[:], uint16(num*sfntSearchRangeMul-maxPow*sfntSearchRangeMul))
	buf.Write(rs[:])
}

func checksum(buf []byte) uint32 {
	sum := uint32(0)
	for i := 0; i+4 <= len(buf); i += 4 {
		sum += binary.BigEndian.Uint32(buf[i : i+4])
	}

	if rem := len(buf) % sfntTableAlign; rem != 0 {
		tail := make([]byte, sfntTableAlign)
		copy(tail[sfntTableAlign-rem:], buf[len(buf)-rem:])
		sum += binary.BigEndian.Uint32(tail)
	}

	return sum
}
