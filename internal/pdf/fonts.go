package pdf

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sort"

	"gowkhtmltopdf/internal/pdf/assets"
)

// Font is a parsed TrueType/OpenType font (table-directory view). It exposes
// the metrics needed for text layout and the glyph data needed to embed and
// subset the font into the PDF.
type Font struct {
	data []byte

	unitsPerEm    int16
	indexToLocFmt int16
	numGlyphs     int
	ascender      int16 // font units
	descender     int16 // font units
	numHMetrics   int
	xMin, yMin    int16
	xMax, yMax    int16
	macStyle      uint16
	italicAngle   int16
	capHeight     int16

	tables  map[string][]byte // name -> raw table bytes (for rebuilding subset)
	cmap    map[uint32]uint16 // rune -> glyph id
	advance []int32           // advance width in font units per glyph
}

// DefaultFont returns the embedded Liberation Sans regular font.
func DefaultFont() (*Font, error) {
	return ParseTTF(bytes.Clone(assets.LiberationSansRegularTTF))
}

// ParseTTF parses a TrueType (or OpenType with TrueType outlines) font file.
// CFF-based fonts return an error - this writer targets TrueType outlines.
func ParseTTF(data []byte) (*Font, error) {
	if len(data) < 12 {
		return nil, errors.New("font: file too short")
	}
	if !bytes.Equal(data[0:4], []byte{0, 1, 0, 0}) &&
		!bytes.Equal(data[0:4], []byte("true")) {
		return nil, errors.New("font: not a TrueType font")
	}
	numTables := int(binary.BigEndian.Uint16(data[4:6]))
	if 12+16*numTables > len(data) {
		return nil, errors.New("font: truncated table directory")
	}
	f := &Font{data: data, tables: map[string][]byte{}}
	for i := 0; i < numTables; i++ {
		rec := data[12+16*i:]
		tag := string(rec[0:4])
		off := int(binary.BigEndian.Uint32(rec[8:12]))
		length := int(binary.BigEndian.Uint32(rec[12:16]))
		if off+length > len(data) {
			return nil, fmt.Errorf("font: table %q out of range", tag)
		}
		f.tables[tag] = data[off : off+length]
	}
	if err := f.parseHead(); err != nil {
		return nil, err
	}
	if err := f.parseMaxp(); err != nil {
		return nil, err
	}
	if err := f.parseHhea(); err != nil {
		return nil, err
	}
	if err := f.parseHmtx(); err != nil {
		return nil, err
	}
	if err := f.parseOS2(); err != nil {
		return nil, err
	}
	if err := f.parseCmap(); err != nil {
		return nil, err
	}
	return f, nil
}

func (f *Font) parseHead() error {
	t, ok := f.tables["head"]
	if !ok || len(t) < 54 {
		return errors.New("font: missing head table")
	}
	f.unitsPerEm = int16(binary.BigEndian.Uint16(t[18:20]))
	if f.unitsPerEm <= 0 {
		return errors.New("font: bad unitsPerEm")
	}
	f.xMin = int16(binary.BigEndian.Uint16(t[36:38]))
	f.yMin = int16(binary.BigEndian.Uint16(t[38:40]))
	f.xMax = int16(binary.BigEndian.Uint16(t[40:42]))
	f.yMax = int16(binary.BigEndian.Uint16(t[42:44]))
	f.macStyle = binary.BigEndian.Uint16(t[44:46])
	f.indexToLocFmt = int16(binary.BigEndian.Uint16(t[50:52]))
	return nil
}

func (f *Font) parseMaxp() error {
	t, ok := f.tables["maxp"]
	if !ok || len(t) < 6 {
		return errors.New("font: missing maxp table")
	}
	f.numGlyphs = int(binary.BigEndian.Uint16(t[4:6]))
	return nil
}

func (f *Font) parseHhea() error {
	t, ok := f.tables["hhea"]
	if !ok || len(t) < 36 {
		return errors.New("font: missing hhea table")
	}
	f.ascender = int16(binary.BigEndian.Uint16(t[4:6]))
	f.descender = int16(binary.BigEndian.Uint16(t[6:8]))
	f.numHMetrics = int(binary.BigEndian.Uint16(t[34:36]))
	if f.numHMetrics <= 0 {
		return errors.New("font: bad numberOfHMetrics")
	}
	return nil
}

func (f *Font) parseHmtx() error {
	t, ok := f.tables["hmtx"]
	if !ok {
		return errors.New("font: missing hmtx table")
	}
	need := f.numHMetrics * 4
	if len(t) < need {
		return errors.New("font: truncated hmtx table")
	}
	lastAdv := int32(binary.BigEndian.Uint16(t[need-4 : need-2]))
	f.advance = make([]int32, f.numGlyphs)
	for i := 0; i < f.numHMetrics; i++ {
		f.advance[i] = int32(binary.BigEndian.Uint16(t[i*4 : i*4+2]))
	}
	for i := f.numHMetrics; i < f.numGlyphs; i++ {
		f.advance[i] = lastAdv
	}
	return nil
}

func (f *Font) parseOS2() error {
	t, ok := f.tables["OS/2"]
	if !ok || len(t) < 90 {
		// OS/2 optional; use hhea for cap height fallback
		f.capHeight = f.ascender
		return nil
	}
	if v := int16(binary.BigEndian.Uint16(t[88:90])); v != 0 {
		f.capHeight = v
	} else {
		f.capHeight = f.ascender
	}
	if f.italicAngle == 0 {
		// italic angle comes from 'post' when present
		if post, ok := f.tables["post"]; ok && len(post) >= 4 {
			f.italicAngle = int16(binary.BigEndian.Uint16(post[4:6]) / 64)
		}
	}
	return nil
}

func (f *Font) parseCmap() error {
	t, ok := f.tables["cmap"]
	if !ok || len(t) < 4 {
		return errors.New("font: missing cmap table")
	}
	num := int(binary.BigEndian.Uint16(t[2:4]))
	type sub struct {
		platform, encoding int
		offset             int
	}
	var subs []sub
	for i := 0; i < num; i++ {
		rec := t[4+i*8:]
		subs = append(subs, sub{
			platform: int(binary.BigEndian.Uint16(rec[0:2])),
			encoding: int(binary.BigEndian.Uint16(rec[2:4])),
			offset:   int(binary.BigEndian.Uint32(rec[4:8])),
		})
	}
	// preference: MS Unicode (3,1) then (3,10) then Unicode (0,x)
	order := [][2]int{{3, 1}, {3, 10}, {0, 0}, {0, 1}, {0, 2}, {0, 3}, {0, 4}, {0, 5}}
	var best *sub
	for _, o := range order {
		for i := range subs {
			if subs[i].platform == o[0] && subs[i].encoding == o[1] {
				best = &subs[i]
				break
			}
		}
		if best != nil {
			break
		}
	}
	if best == nil {
		return errors.New("font: no Unicode cmap subtable")
	}
	if best.offset < 0 || best.offset+2 > len(t) {
		return errors.New("font: cmap subtable out of range")
	}
	format := int(binary.BigEndian.Uint16(t[best.offset : best.offset+2]))
	f.cmap = map[uint32]uint16{}
	st := t[best.offset:]
	switch format {
	case 0:
		if len(st) < 262 {
			return errors.New("font: truncated cmap format 0")
		}
		for i := 0; i < 256; i++ {
			if g := st[6+i]; g != 0 {
				f.cmap[uint32(i)] = uint16(g)
			}
		}
	case 4:
		if err := f.parseCmap4(st); err != nil {
			return err
		}
	case 6:
		if len(st) < 10 {
			return errors.New("font: truncated cmap format 6")
		}
		first := binary.BigEndian.Uint16(st[6:8])
		count := int(binary.BigEndian.Uint16(st[8:10]))
		if 10+2*count > len(st) {
			return errors.New("font: truncated cmap format 6")
		}
		for i := 0; i < count; i++ {
			g := binary.BigEndian.Uint16(st[10+i*2:])
			if g != 0 {
				f.cmap[uint32(first)+uint32(i)] = g
			}
		}
	case 12:
		if err := f.parseCmap12(st); err != nil {
			return err
		}
	default:
		return fmt.Errorf("font: unsupported cmap format %d", format)
	}
	if len(f.cmap) == 0 {
		return errors.New("font: empty cmap")
	}
	return nil
}

func (f *Font) parseCmap4(st []byte) error {
	if len(st) < 14 {
		return errors.New("font: truncated cmap format 4")
	}
	segCount := int(binary.BigEndian.Uint16(st[6:8])) / 2
	endOff := 14
	startOff := endOff + 2*segCount + 2 // + reservedPad
	deltaOff := startOff + 2*segCount
	rangeOff := deltaOff + 2*segCount
	if rangeOff+2*segCount > len(st) {
		return errors.New("font: truncated cmap format 4 segments")
	}
	for i := 0; i < segCount; i++ {
		end := uint32(binary.BigEndian.Uint16(st[endOff+i*2 : endOff+i*2+2]))
		start := uint32(binary.BigEndian.Uint16(st[startOff+i*2 : startOff+i*2+2]))
		if end == 0xFFFF && start == 0xFFFF {
			continue // final sentinel segment
		}
		idDelta := int32(int16(binary.BigEndian.Uint16(st[deltaOff+i*2 : deltaOff+i*2+2])))
		idRange := int32(binary.BigEndian.Uint16(st[rangeOff+i*2 : rangeOff+i*2+2]))
		for cp := start; cp <= end; cp++ {
			var glyph int32
			if idRange == 0 {
				glyph = (int32(cp) + idDelta) & 0xFFFF
			} else {
				gAddr := rangeOff + i*2 + int(idRange) + int(cp-start)*2
				if gAddr+2 > len(st) {
					continue
				}
				g := binary.BigEndian.Uint16(st[gAddr:])
				if g == 0 {
					continue
				}
				glyph = (int32(g) + idDelta) & 0xFFFF
			}
			if glyph > 0 && glyph < int32(f.numGlyphs) {
				f.cmap[cp] = uint16(glyph)
			}
		}
	}
	return nil
}

func (f *Font) parseCmap12(st []byte) error {
	if len(st) < 16 {
		return errors.New("font: truncated cmap format 12")
	}
	groups := int(binary.BigEndian.Uint32(st[12:16]))
	for i := 0; i < groups; i++ {
		rec := st[16+i*12:]
		start := binary.BigEndian.Uint32(rec[0:4])
		end := binary.BigEndian.Uint32(rec[4:8])
		startGlyph := binary.BigEndian.Uint32(rec[8:12])
		for cp := start; cp <= end; cp++ {
			g := startGlyph + (cp - start)
			if g < uint32(f.numGlyphs) {
				f.cmap[cp] = uint16(g)
			}
		}
	}
	return nil
}

// UnitsPerEm returns the font's design size.
func (f *Font) UnitsPerEm() int16 { return f.unitsPerEm }

// Ascent returns the typographic ascent in font units.
func (f *Font) Ascent() int16 { return f.ascender }

// Descent returns the typographic descent (negative) in font units.
func (f *Font) Descent() int16 { return f.descender }

// CapHeight returns the cap height in font units.
func (f *Font) CapHeight() int16 { return f.capHeight }

// BBox returns the font bounding box in font units.
func (f *Font) BBox() (int16, int16, int16, int16) { return f.xMin, f.yMin, f.xMax, f.yMax }

// Bold reports whether the font declares a bold macStyle.
func (f *Font) Bold() bool { return f.macStyle&1 != 0 }

// Italic reports whether the font declares an italic macStyle.
func (f *Font) Italic() bool { return f.macStyle&2 != 0 }

// GlyphID maps a rune to its glyph id (0 = .notdef when missing).
func (f *Font) GlyphID(r rune) uint16 {
	if g, ok := f.cmap[uint32(r)]; ok {
		return g
	}
	return 0
}

// Advance returns the horizontal advance width in font units for a rune.
func (f *Font) Advance(r rune) float64 {
	g := f.GlyphID(r)
	if int(g) >= len(f.advance) {
		return float64(f.advance[0])
	}
	return float64(f.advance[g])
}

// AdvanceInPoints converts an advance from font units to points for a size.
func (f *Font) AdvanceInPoints(r rune, size float64) float64 {
	return f.Advance(r) / float64(f.unitsPerEm) * size
}

// glyphOutline returns the glyf bytes for a glyph id (raw, incl. header).
func (f *Font) glyphOutline(g uint16) []byte {
	glyf, ok := f.tables["glyf"]
	if !ok {
		return nil
	}
	loca, ok := f.tables["loca"]
	if !ok {
		return nil
	}
	n := int(g)
	var off, next int
	if f.indexToLocFmt == 0 {
		off = int(binary.BigEndian.Uint16(loca[n*2:])) * 2
		next = int(binary.BigEndian.Uint16(loca[(n+1)*2:])) * 2
	} else {
		off = int(binary.BigEndian.Uint32(loca[n*4:]))
		next = int(binary.BigEndian.Uint32(loca[(n+1)*4:]))
	}
	if off < 0 || next < off || next > len(glyf) {
		return nil
	}
	return glyf[off:next]
}

// compositeGlyphIDs returns the glyph ids referenced by a composite glyph.
func (f *Font) compositeGlyphIDs(g uint16) []uint16 {
	out := []uint16{}
	b := f.glyphOutline(g)
	if len(b) < 10 {
		return out
	}
	numContours := int16(binary.BigEndian.Uint16(b[0:2]))
	if numContours >= 0 {
		return out // simple glyph
	}
	// composite: flags at 10, glyph index at 12, args, then more components
	pos := 10
	for {
		if pos+4 > len(b) {
			break
		}
		flags := binary.BigEndian.Uint16(b[pos : pos+2])
		child := binary.BigEndian.Uint16(b[pos+2 : pos+4])
		out = append(out, child)
		argBytes := 2
		if flags&0x0001 != 0 {
			argBytes = 4 // ARG_1_AND_2_ARE_WORDS
		}
		pos += 4 + argBytes
		if flags&0x0008 == 0 { // MORE_COMPONENTS
			break
		}
	}
	return out
}

// Runes returns all runes mapped by the font, sorted.
func (f *Font) Runes() []rune {
	out := make([]rune, 0, len(f.cmap))
	for cp := range f.cmap {
		out = append(out, rune(cp))
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
