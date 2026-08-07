package pdf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"sort"
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
func subsetFont(f *Font, used []rune, scope subsetScope) (*subsetResult, error) {
	accept := func(r rune) bool {
		if scope == subsetSimple {
			return simpleFontRune(r)
		}

		return r <= 0xFFFF
	}

	// .notdef always included
	glyphSet := map[uint16]bool{0: true}

	for _, r := range used {
		if !accept(r) {
			continue
		}

		g := f.GlyphID(r)
		if g == 0 {
			continue
		}

		collectGlyph(f, g, glyphSet)
	}
	// sort glyphs by original id for deterministic output
	glyphs := make([]uint16, 0, len(glyphSet))
	for g := range glyphSet {
		glyphs = append(glyphs, g)
	}

	sort.Slice(glyphs, func(i, j int) bool { return glyphs[i] < glyphs[j] })

	oldToNew := map[uint16]uint16{}
	advances := make([]int32, 0, len(glyphs))
	lsbs := make([]int16, 0, len(glyphs))
	outlines := make([][]byte, len(glyphs))

	for newID, old := range glyphs {
		oldToNew[old] = uint16(newID)

		if int(old) < len(f.advance) {
			advances = append(advances, f.advance[old])
		} else {
			advances = append(advances, 0)
		}

		if int(old) < len(f.lsb) {
			lsbs = append(lsbs, f.lsb[old])
		} else {
			lsbs = append(lsbs, 0)
		}

		outlines[newID] = f.glyphOutline(old)
	}
	// remap composite components (on clones - never mutate source tables)
	cloned := make([][]byte, len(outlines))
	for i, o := range outlines {
		cloned[i] = stripGlyphHints(bytes.Clone(o))
		remapComposite(cloned[i], oldToNew)
	}

	outlines = cloned

	res := &subsetResult{glyphIDs: map[rune]uint16{}}

	for _, r := range used {
		if !accept(r) {
			continue
		}

		old := f.GlyphID(r)
		if old == 0 {
			continue
		}

		res.glyphIDs[r] = oldToNew[old]
	}

	for r := range res.glyphIDs {
		res.runes = append(res.runes, r)
	}

	sort.Slice(res.runes, func(i, j int) bool { return res.runes[i] < res.runes[j] })

	sub := &subsetter{f: f, glyphs: glyphs, outlines: outlines, advances: advances, lsbs: lsbs}
	sub.mappings = make([]codeGlyph, 0, len(res.glyphIDs))

	for r, g := range res.glyphIDs {
		sub.mappings = append(sub.mappings, codeGlyph{code: uint16(r), glyph: g})
	}

	sort.Slice(sub.mappings, func(i, j int) bool { return sub.mappings[i].code < sub.mappings[j].code })

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

// collectGlyph adds g and (for composites) all referenced children.
func collectGlyph(f *Font, g uint16, set map[uint16]bool) {
	if set[g] {
		return
	}

	set[g] = true
	for _, c := range f.compositeGlyphIDs(g) {
		collectGlyph(f, c, set)
	}
}

// stripGlyphHints removes TrueType hinting bytecode from a glyf outline.
// Subsets omit fpgm/prep/cvt, so leftover instructions can garble CJK
// composites in PDF viewers (broken 東京都 etc.).
func stripGlyphHints(b []byte) []byte {
	if len(b) < 10 {
		return b
	}

	numContours := int16(binary.BigEndian.Uint16(b[0:2]))
	if numContours < 0 {
		return stripCompositeHints(b)
	}

	if numContours == 0 {
		return b
	}

	n := int(numContours)
	if len(b) < 10+2*n+2 {
		return b
	}

	insPos := 10 + 2*n
	insLen := int(binary.BigEndian.Uint16(b[insPos:]))

	after := insPos + 2 + insLen
	if after > len(b) {
		return b
	}

	if insLen == 0 {
		return b
	}

	out := make([]byte, 0, len(b)-insLen)
	out = append(out, b[:insPos]...)
	out = append(out, 0, 0) // instructionLength = 0
	out = append(out, b[after:]...)

	return out
}

func stripCompositeHints(b []byte) []byte {
	pos := 10
	lastFlagsAt := -1

	for {
		if pos+4 > len(b) {
			return b
		}

		lastFlagsAt = pos
		flags := binary.BigEndian.Uint16(b[pos : pos+2])
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

	if lastFlagsAt < 0 || pos > len(b) {
		return b
	}

	flags := binary.BigEndian.Uint16(b[lastFlagsAt : lastFlagsAt+2])
	if flags&0x0100 == 0 { // WE_HAVE_INSTRUCTIONS
		// Drop any accidental trailing bytes past the component list.
		return bytes.Clone(b[:pos])
	}

	out := bytes.Clone(b[:pos])
	binary.BigEndian.PutUint16(out[lastFlagsAt:lastFlagsAt+2], flags&^0x0100)

	return out
}

// remapComposite rewrites composite component glyph ids in a glyf outline.
func remapComposite(b []byte, oldToNew map[uint16]uint16) {
	if len(b) < 10 {
		return
	}

	numContours := int16(binary.BigEndian.Uint16(b[0:2]))
	if numContours >= 0 {
		return
	}

	pos := 10

	for {
		if pos+4 > len(b) {
			break
		}

		flags := binary.BigEndian.Uint16(b[pos : pos+2])

		old := binary.BigEndian.Uint16(b[pos+2 : pos+4])
		if n, ok := oldToNew[old]; ok {
			binary.BigEndian.PutUint16(b[pos+2:pos+4], n)
		}

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

func (s *subsetter) build() ([]byte, error) {
	numGlyphs := len(s.glyphs)
	// long loca always (format 1). TrueType requires each glyph to start on a
	// 4-byte boundary; odd/unaligned glyf offsets corrupt CJK composites in
	// PDFium and other viewers (garbled 東京都 etc.).
	loca := make([]uint32, numGlyphs+1)
	cur := 0
	padded := make([][]byte, numGlyphs)

	for i, o := range s.outlines {
		loca[i] = uint32(cur)

		p := bytes.Clone(o)
		for len(p)%4 != 0 {
			p = append(p, 0)
		}

		padded[i] = p
		cur += len(p)
	}

	loca[numGlyphs] = uint32(cur)

	// hmtx: advance (2) + lsb (2) per glyph
	hmtx := new(bytes.Buffer)
	for i, a := range s.advances {
		binary.Write(hmtx, binary.BigEndian, uint16(a))

		lsb := int16(0)
		if i < len(s.lsbs) {
			lsb = s.lsbs[i]
		}

		binary.Write(hmtx, binary.BigEndian, uint16(lsb))
	}

	// cmap: rune codes → renumbered glyph ids
	cmap, err := unicodeCmap4(s.mappings)
	if err != nil {
		return nil, err
	}

	// head: copy original, patch indexToLocFormat=1
	head := bytes.Clone(s.f.tables["head"])
	if len(head) < 52 {
		return nil, errors.New("font: bad head in subset")
	}

	binary.BigEndian.PutUint16(head[50:52], 1)

	// maxp: copy original, patch numGlyphs
	maxp := bytes.Clone(s.f.tables["maxp"])
	if len(maxp) < 6 {
		return nil, errors.New("font: bad maxp in subset")
	}

	binary.BigEndian.PutUint16(maxp[4:6], uint16(numGlyphs))

	// hhea: copy, patch numberOfHMetrics
	hhea := bytes.Clone(s.f.tables["hhea"])
	if len(hhea) < 36 {
		return nil, errors.New("font: bad hhea in subset")
	}

	binary.BigEndian.PutUint16(hhea[34:36], uint16(numGlyphs))

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
		{"hmtx", hmtx.Bytes()},
		{"cmap", cmap},
		{"loca", uint32Bytes(loca)},
		{"glyf", glyf.Bytes()},
		{"OS/2", cloneTable(s.f, "OS/2")},
		{"post", cloneTable(s.f, "post")},
	}

	return buildFontFile(tables)
}

func cloneTable(f *Font, tag string) []byte {
	if t, ok := f.tables[tag]; ok {
		return bytes.Clone(t)
	}

	return nil
}

func uint32Bytes(v []uint32) []byte {
	b := make([]byte, len(v)*4)
	for i, x := range v {
		binary.BigEndian.PutUint32(b[i*4:], x)
	}

	return b
}

// unicodeCmap4 builds a cmap with one format-4 subtable mapping codes to
// renumbered glyph ids. Consecutive codes mapping to consecutive glyphs are
// coalesced into segments (constant delta).
func unicodeCmap4(mappings []codeGlyph) ([]byte, error) {
	if len(mappings) == 0 {
		return nil, errors.New("font: empty cmap mappings")
	}

	type seg struct{ start, end, delta uint16 }

	var segs []seg

	for i := 0; i < len(mappings); {
		j := i
		for j+1 < len(mappings) &&
			mappings[j+1].code == mappings[j].code+1 &&
			mappings[j+1].glyph == mappings[j].glyph+1 {
			j++
		}

		delta := (int(mappings[i].glyph) - int(mappings[i].code)) & 0xFFFF
		segs = append(segs, seg{mappings[i].code, mappings[j].code, uint16(delta)})
		i = j + 1
	}

	segs = append(segs, seg{0xFFFF, 0xFFFF, 1}) // sentinel
	segCount := len(segs)
	length := 16 + 8*segCount // 14-byte header + reservedPad + 4 arrays
	b := make([]byte, length)
	binary.BigEndian.PutUint16(b[0:2], 4) // format
	binary.BigEndian.PutUint16(b[2:4], uint16(length))
	binary.BigEndian.PutUint16(b[6:8], uint16(segCount*2))

	maxPow := 1
	for maxPow*2 <= segCount {
		maxPow *= 2
	}

	binary.BigEndian.PutUint16(b[8:10], uint16(maxPow*2))             // searchRange
	binary.BigEndian.PutUint16(b[10:12], uint16(maxPow))              // entrySelector
	binary.BigEndian.PutUint16(b[12:14], uint16(segCount*2-maxPow*2)) // rangeShift

	endOff := 14
	startOff := endOff + 2*segCount + 2
	deltaOff := startOff + 2*segCount
	rangeOff := deltaOff + 2*segCount

	for i, s := range segs {
		binary.BigEndian.PutUint16(b[endOff+i*2:], s.end)
		binary.BigEndian.PutUint16(b[startOff+i*2:], s.start)
		binary.BigEndian.PutUint16(b[deltaOff+i*2:], s.delta)
		binary.BigEndian.PutUint16(b[rangeOff+i*2:], 0)
	}
	// wrap in cmap table: version, numTables, (3,1) subtable record
	out := make([]byte, 0, 12+length)
	out = append(out, 0, 0, 0, 1)
	out = append(out, 0, 3, 0, 1)

	var rec [4]byte

	binary.BigEndian.PutUint32(rec[:], 12)
	out = append(out, rec[:]...)
	out = append(out, b...)

	return out, nil
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
	var t []struct {
		tag  string
		data []byte
	}

	for _, x := range tables {
		if x.data != nil {
			t = append(t, x)
		}
	}

	sort.Slice(t, func(i, j int) bool { return t[i].tag < t[j].tag })
	num := len(t)
	// compute head checksum adjustment: total file length must be 0 mod 2^32
	dirLen := 12 + 16*num
	// align each table to 4 bytes
	total := dirLen
	aligned := make([]int, num)

	for i, x := range t {
		pad := (4 - total%4) % 4
		total += pad
		aligned[i] = pad
		total += len(x.data)
	}
	// file checksum must be 0x1B0BADB0D via head.checksumAdjustment
	headIdx := -1

	for i, x := range t {
		if x.tag == "head" {
			headIdx = i
		}
	}

	if headIdx >= 0 {
		// checksum of the file with checksumAdjustment zeroed
		zeroed := make([]byte, len(t[headIdx].data))
		copy(zeroed, t[headIdx].data)
		copy(zeroed[8:12], []byte{0, 0, 0, 0})
		t[headIdx].data = zeroed
		zeroedSum := checksum(zeroed)

		// layout the whole file to compute checksum
		full := assembleFile(t, aligned)
		sum := checksum(full)
		// place adjustment such that the final file sums to 0xB1B0AFBA.
		// The head checksum in the directory is kept at the zeroed-head
		// value, so the adjustment only shifts the sum once.
		adj := 0xB1B0AFBA - sum
		adjusted := bytes.Clone(zeroed)
		binary.BigEndian.PutUint32(adjusted[8:12], adj)
		t[headIdx].data = adjusted
		full = assembleFile(t, aligned)
		// freeze the directory entry for head to the zeroed-head checksum
		for i := range len(t) {
			if t[i].tag == "head" {
				rec := 12 + 16*i + 4
				binary.BigEndian.PutUint32(full[rec:rec+4], zeroedSum)

				break
			}
		}

		return full, nil
	}

	return assembleFile(t, aligned), nil
}

func assembleFile(t []struct {
	tag  string
	data []byte
}, aligned []int,
) []byte {
	num := len(t)
	buf := new(bytes.Buffer)
	buf.Write([]byte{0, 1, 0, 0})

	var numT [2]byte

	binary.BigEndian.PutUint16(numT[:], uint16(num))
	buf.Write(numT[:])
	// searchRange, entrySelector, rangeShift
	maxPow := 1
	sel := 0

	for maxPow*2 <= num {
		maxPow *= 2
		sel++
	}

	var sr [2]byte

	binary.BigEndian.PutUint16(sr[:], uint16(maxPow*16))
	buf.Write(sr[:])

	var es [2]byte

	binary.BigEndian.PutUint16(es[:], uint16(sel))
	buf.Write(es[:])

	var rs [2]byte

	binary.BigEndian.PutUint16(rs[:], uint16(num*16-maxPow*16))
	buf.Write(rs[:])

	// directory
	offset := 12 + 16*num
	for i, x := range t {
		offset += aligned[i] // padding before table i

		buf.WriteString(x.tag)

		var cs [4]byte

		binary.BigEndian.PutUint32(cs[:], checksum(x.data))
		buf.Write(cs[:])

		var off [4]byte

		binary.BigEndian.PutUint32(off[:], uint32(offset))
		buf.Write(off[:])

		var tlen [4]byte

		binary.BigEndian.PutUint32(tlen[:], uint32(len(x.data)))
		buf.Write(tlen[:])

		offset += len(x.data)
	}
	// table data with alignment
	for i, x := range t {
		for range aligned[i] {
			buf.WriteByte(0)
		}

		buf.Write(x.data)
	}

	return buf.Bytes()
}

func checksum(b []byte) uint32 {
	sum := uint32(0)
	for i := 0; i+4 <= len(b); i += 4 {
		sum += binary.BigEndian.Uint32(b[i : i+4])
	}

	if rem := len(b) % 4; rem != 0 {
		pad := make([]byte, 4-rem)
		sum += binary.BigEndian.Uint32(append(pad, b[len(b)-rem:]...))
	}

	return sum
}
