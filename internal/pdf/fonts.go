package pdf

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"unicode/utf16"

	//nolint:depguard // go-text/typesetting is the repo's fixed text-shaping dependency
	gtfont "github.com/go-text/typesetting/font"
)

var (
	errFontTooShort           = errors.New("font: file too short")
	errFontCFFNotSupported    = errors.New("font: CFF/OTTO OpenType not supported (TrueType outlines only)")
	errFontNotTrueType        = errors.New("font: not a TrueType font")
	errFontTruncatedDirectory = errors.New("font: truncated table directory")
	errFontMissingHead        = errors.New("font: missing head table")
	errFontBadUnitsPerEm      = errors.New("font: bad unitsPerEm")
	errFontMissingMaxp        = errors.New("font: missing maxp table")
	errFontMissingHhea        = errors.New("font: missing hhea table")
	errFontBadNumHMetrics     = errors.New("font: bad numberOfHMetrics")
	errFontMissingHmtx        = errors.New("font: missing hmtx table")
	errFontTruncatedHmtx      = errors.New("font: truncated hmtx table")
	errFontMissingCmap        = errors.New("font: missing cmap table")
	errFontEmptyCmap          = errors.New("font: empty cmap")
	errFontTruncatedCmap4     = errors.New("font: truncated cmap format 4")
	errFontTruncatedCmap4Segs = errors.New("font: truncated cmap format 4 segments")
	errFontTruncatedCmap12    = errors.New("font: truncated cmap format 12")
)

// Font is a parsed TrueType/OpenType font (table-directory view). It exposes
// the metrics needed for text layout and the glyph data needed to embed and
// subset the font into the PDF.
type Font struct {
	data []byte
	// fingerprint identifies the loaded face independently of its display
	// name. Registries may assign the same PostScriptName to different files;
	// subset caching must not merge those faces.
	fingerprint [32]byte

	// PostScriptName is the PDF /BaseFont label (e.g. LiberationSans-Bold).
	// Empty when the font was loaded without a registry name.
	PostScriptName string

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
	lsb     []int16           // left side bearing per glyph

	// Derived, immutable caches live on the Font next to the data they
	// derive from (locality) and disappear with it (bounds): the go-text
	// face and the reverse cmap are built at most once each.
	gotOnce  sync.Once
	gotFace  *gtfont.Face // parsed go-text face (nil on failure)
	revOnce  sync.Once
	rev      map[uint16]rune
	nameOnce sync.Once
	names    []string
}

// ParseTTF parses a TrueType (or OpenType with TrueType outlines) font file.
// CFF-based fonts return an error - this writer targets TrueType outlines.
func ParseTTF(data []byte) (*Font, error) {
	if len(data) < sfntOffsetTableSize {
		return nil, errFontTooShort
	}

	if !bytes.Equal(data[0:4], []byte{0, 1, 0, 0}) &&
		!bytes.Equal(data[0:4], []byte("true")) {
		if bytes.Equal(data[0:4], []byte("OTTO")) {
			return nil, errFontCFFNotSupported
		}

		return nil, errFontNotTrueType
	}

	tables, err := parseTableDirectory(data)
	if err != nil {
		return nil, err
	}

	font := &Font{ //nolint:exhaustruct // intentional zero-value fields
		data:        data,
		fingerprint: sha256.Sum256(data),
		tables:      tables,
	}

	if err := font.parseAll(); err != nil {
		return nil, err
	}

	return font, nil
}

// parseAll runs the table parsers in dependency order; the first failure
// aborts the parse.
func (f *Font) parseAll() error {
	if err := f.parseHead(); err != nil {
		return err
	}

	if err := f.parseMaxp(); err != nil {
		return err
	}

	if err := f.parseHhea(); err != nil {
		return err
	}

	if err := f.parseHmtx(); err != nil {
		return err
	}

	f.parseOS2()

	return f.parseCmap()
}

func parseTableDirectory(data []byte) (map[string][]byte, error) {
	numTables := int(binary.BigEndian.Uint16(data[4:6]))
	if sfntOffsetTableSize+sfntTableRecordSize*numTables > len(data) {
		return nil, errFontTruncatedDirectory
	}

	tables := make(map[string][]byte, numTables)

	for i := range numTables {
		rec := data[sfntOffsetTableSize+sfntTableRecordSize*i:]
		tag := string(rec[0:4])
		off := int(binary.BigEndian.Uint32(rec[8:12]))
		length := int(binary.BigEndian.Uint32(rec[12:16]))

		if off+length > len(data) {
			return nil, fmt.Errorf("font: table %q out of range", tag) //nolint:err113 // dynamic tag in message
		}

		tables[tag] = data[off : off+length]
	}

	return tables, nil
}

func (f *Font) parseHead() error {
	tbl, ok := f.tables["head"]
	if !ok || len(tbl) < 54 {
		return errFontMissingHead
	}

	//nolint:gosec // font units are verified <= 0 after conversion; int16 range is the format's
	f.unitsPerEm = int16(binary.BigEndian.Uint16(tbl[18:20]))
	if f.unitsPerEm <= 0 {
		return errFontBadUnitsPerEm
	}

	//nolint:gosec // raw table int16 fields; wrapping is not possible at these fixed offsets
	f.xMin = int16(binary.BigEndian.Uint16(tbl[36:38]))
	//nolint:gosec // raw table int16 fields; wrapping is not possible at these fixed offsets
	f.yMin = int16(binary.BigEndian.Uint16(tbl[38:40]))
	//nolint:gosec // raw table int16 fields; wrapping is not possible at these fixed offsets
	f.xMax = int16(binary.BigEndian.Uint16(tbl[40:42]))
	//nolint:gosec // raw table int16 fields; wrapping is not possible at these fixed offsets
	f.yMax = int16(binary.BigEndian.Uint16(tbl[42:44]))
	f.macStyle = binary.BigEndian.Uint16(tbl[44:46])
	//nolint:gosec // indexToLocFmt is 0 or 1 per spec
	f.indexToLocFmt = int16(binary.BigEndian.Uint16(tbl[50:52]))

	return nil
}

func (f *Font) parseMaxp() error {
	t, ok := f.tables["maxp"]
	if !ok || len(t) < 6 {
		return errFontMissingMaxp
	}

	f.numGlyphs = int(binary.BigEndian.Uint16(t[4:6]))

	return nil
}

func (f *Font) parseHhea() error {
	tbl, ok := f.tables["hhea"]
	if !ok || len(tbl) < 36 {
		return errFontMissingHhea
	}

	//nolint:gosec // raw table int16 fields; the TrueType format stores signed values here
	f.ascender = int16(binary.BigEndian.Uint16(tbl[4:6]))
	//nolint:gosec // raw table int16 fields; the TrueType format stores signed values here
	f.descender = int16(binary.BigEndian.Uint16(tbl[6:8]))
	f.numHMetrics = int(binary.BigEndian.Uint16(tbl[34:36]))

	if f.numHMetrics <= 0 {
		return errFontBadNumHMetrics
	}

	return nil
}

func (f *Font) parseHmtx() error {
	tbl, ok := f.tables["hmtx"]
	if !ok {
		return errFontMissingHmtx
	}

	need := f.numHMetrics * bytesPerHMetric
	if len(tbl) < need {
		return errFontTruncatedHmtx
	}

	lastAdv := int32(binary.BigEndian.Uint16(tbl[need-4 : need-2]))
	f.advance = make([]int32, f.numGlyphs)
	f.lsb = make([]int16, f.numGlyphs)

	for i := range f.numHMetrics {
		f.advance[i] = int32(binary.BigEndian.Uint16(tbl[i*4 : i*4+2]))
		f.lsb[i] = int16(binary.BigEndian.Uint16(tbl[i*4+2 : i*4+4])) //nolint:gosec // lsb is int16 per hmtx spec
	}

	sideBearings := tbl[need:]

	for i := f.numHMetrics; i < f.numGlyphs; i++ {
		f.advance[i] = lastAdv
		off := (i - f.numHMetrics) * bytesPerLongHorMetricSide

		if off+2 <= len(sideBearings) {
			f.lsb[i] = int16(binary.BigEndian.Uint16(sideBearings[off : off+2])) //nolint:gosec // lsb is int16 per hmtx spec
		}
	}

	return nil
}

func (f *Font) parseOS2() {
	tbl, ok := f.tables["OS/2"]
	if !ok || len(tbl) < 90 {
		// OS/2 optional; use hhea for cap height fallback
		f.capHeight = f.ascender

		return
	}

	if v := int16(binary.BigEndian.Uint16(tbl[88:90])); v != 0 { //nolint:gosec // capHeight is int16 per OS/2 spec
		f.capHeight = v
	} else {
		f.capHeight = f.ascender
	}

	if f.italicAngle == 0 {
		// italic angle comes from 'post' when present
		if post, ok := f.tables["post"]; ok && len(post) >= 4 {
			//nolint:gosec // italicAngle is int16 per post spec
			f.italicAngle = int16(binary.BigEndian.Uint16(post[4:6]) / fixed14Divisor)
		}
	}
}

func (f *Font) parseCmap() error {
	tbl, ok := f.tables["cmap"]
	if !ok || len(tbl) < 4 {
		return errFontMissingCmap
	}

	f.cmap = map[uint32]uint16{}
	subs := parseCmapSubtables(tbl)
	mergeCmapSubtables(f, tbl, subs)

	if len(f.cmap) == 0 {
		return errFontEmptyCmap
	}

	return nil
}

// cmapSubtable is one cmap encoding record.
type cmapSubtable struct {
	platform, encoding int
	offset             int
}

func parseCmapSubtables(tbl []byte) []cmapSubtable {
	num := int(binary.BigEndian.Uint16(tbl[2:4]))
	subs := make([]cmapSubtable, 0, num)

	for i := range num {
		rec := tbl[4+i*8:]
		subs = append(subs, cmapSubtable{
			platform: int(binary.BigEndian.Uint16(rec[0:2])),
			encoding: int(binary.BigEndian.Uint16(rec[2:4])),
			offset:   int(binary.BigEndian.Uint32(rec[4:8])),
		})
	}

	return subs
}

// mergeCmapSubtables walks the preferred Unicode encodings in priority order
// so CJK + Hangul + Latin are all covered (DroidSansFallback puts Hangul in
// a format-12 table while format-4 (3,1) alone is incomplete).
func mergeCmapSubtables(fnt *Font, tbl []byte, subs []cmapSubtable) {
	order := [][2]int{{3, 10}, {3, 1}, {0, 4}, {0, 3}, {0, 5}, {0, 2}, {0, 1}, {0, 0}}

	for _, o := range order {
		for idx := range subs {
			if subs[idx].platform != o[0] || subs[idx].encoding != o[1] {
				continue
			}

			if subs[idx].offset < 0 || subs[idx].offset+2 > len(tbl) {
				continue
			}

			state := tbl[subs[idx].offset:]
			format := int(binary.BigEndian.Uint16(state[0:2]))

			fnt.parseCmapSubtable(state, format)
		}
	}
}

func (f *Font) parseCmapSubtable(state []byte, format int) {
	switch format {
	case cmapFormat4:
		_ = f.parseCmap4(state)
	case cmapFormat12:
		_ = f.parseCmap12(state)
	case cmapFormat6:
		f.parseCmap6(state)
	case 0:
		f.parseCmap0(state)
	}
}

func (f *Font) parseCmap6(state []byte) {
	if len(state) < cmapFormat6Header {
		return
	}

	first := binary.BigEndian.Uint16(state[6:8])
	count := int(binary.BigEndian.Uint16(state[8:10]))

	if 10+2*count > len(state) {
		return
	}

	for index := range count {
		glyph := binary.BigEndian.Uint16(state[10+index*2:])
		if glyph == 0 {
			continue
		}

		cp := uint32(first) + uint32(index) //nolint:gosec // first+count cannot exceed uint32
		if _, ok := f.cmap[cp]; !ok {
			f.cmap[cp] = glyph
		}
	}
}

func (f *Font) parseCmap0(state []byte) {
	if len(state) < cmapFormat0Size {
		return
	}

	for j := range 256 {
		if g := state[6+j]; g != 0 {
			cp := uint32(j) //nolint:gosec // j is bounded by 256
			if _, ok := f.cmap[cp]; !ok {
				f.cmap[cp] = uint16(g)
			}
		}
	}
}

func (f *Font) parseCmap4(state []byte) error {
	if len(state) < cmapFormat4Header {
		return errFontTruncatedCmap4
	}

	segCount := int(binary.BigEndian.Uint16(state[6:8])) / uint16Bytes
	endOff := 14
	startOff := endOff + uint16Bytes*segCount + uint16Bytes // + reservedPad
	deltaOff := startOff + uint16Bytes*segCount

	rangeOff := deltaOff + uint16Bytes*segCount
	if rangeOff+2*segCount > len(state) {
		return errFontTruncatedCmap4Segs
	}

	for idx := range segCount {
		end := uint32(binary.BigEndian.Uint16(state[endOff+idx*2 : endOff+idx*2+2]))
		start := uint32(binary.BigEndian.Uint16(state[startOff+idx*2 : startOff+idx*2+2]))

		if end == 0xFFFF && start == 0xFFFF {
			continue // final sentinel segment
		}

		//nolint:gosec // idDelta is signed int16 per cmap-4 spec
		idDelta := int32(int16(binary.BigEndian.Uint16(state[deltaOff+idx*2 : deltaOff+idx*2+2])))
		idRange := int32(binary.BigEndian.Uint16(state[rangeOff+idx*2 : rangeOff+idx*2+2]))

		for codepoint := start; codepoint <= end; codepoint++ {
			if _, exists := f.cmap[codepoint]; exists {
				continue
			}

			glyph := cmap4GlyphID(state, idx, codepoint, start, idDelta, idRange, rangeOff)
			if glyph > 0 && glyph < int32(f.numGlyphs) { //nolint:gosec // numGlyphs is small relative to int32
				f.cmap[codepoint] = uint16(glyph) //nolint:gosec // range-checked above
			}
		}
	}

	return nil
}

func cmap4GlyphID(state []byte, idx int, codepoint, start uint32, idDelta, idRange int32, rangeOff int) int32 {
	if idRange == 0 {
		return (int32(codepoint) + idDelta) & maxUint16Val //nolint:gosec // codepoint is BMP-scoped in format 4
	}

	gAddr := rangeOff + idx*uint16Bytes + int(idRange) + int(codepoint-start)*uint16Bytes
	if gAddr+2 > len(state) {
		return 0
	}

	g := binary.BigEndian.Uint16(state[gAddr:])
	if g == 0 {
		return 0
	}

	return (int32(g) + idDelta) & maxUint16Val
}

func (f *Font) parseCmap12(state []byte) error {
	if len(state) < cmapFormat12Header {
		return errFontTruncatedCmap12
	}

	groups := int(binary.BigEndian.Uint32(state[12:16]))
	for i := range groups {
		rec := state[16+i*12:]
		start := binary.BigEndian.Uint32(rec[0:4])
		end := binary.BigEndian.Uint32(rec[4:8])
		startGlyph := binary.BigEndian.Uint32(rec[8:12])

		for codepoint := start; codepoint <= end; codepoint++ {
			if _, exists := f.cmap[codepoint]; exists {
				continue
			}

			g := startGlyph + (codepoint - start)
			if g < uint32(f.numGlyphs) { //nolint:gosec // numGlyphs is small relative to uint32
				f.cmap[codepoint] = uint16(g) //nolint:gosec // range-checked above
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

// pdfEmScale returns the factor that converts font design units to the PDF
// FontDescriptor 1000-unit em. Writing raw 2048-upm values made viewers treat
// Ascent 1825 as 1.825em (Liberation) and inflate text selection boxes.
func (f *Font) pdfEmScale() float64 {
	upm := float64(f.unitsPerEm)
	if upm <= 0 {
		return 1
	}

	return pdfUnitsPerEm / upm
}

func (f *Font) scaleToPDFEm(v int16) int {
	return int(math.Round(float64(v) * f.pdfEmScale()))
}

// PDFAscent is Ascent in 1000-em PDF glyph space.
func (f *Font) PDFAscent() int { return f.scaleToPDFEm(f.ascender) }

// PDFDescent is Descent in 1000-em PDF glyph space.
func (f *Font) PDFDescent() int { return f.scaleToPDFEm(f.descender) }

// PDFCapHeight is CapHeight in 1000-em PDF glyph space.
func (f *Font) PDFCapHeight() int { return f.scaleToPDFEm(f.capHeight) }

// PDFBBox is the font bbox in 1000-em PDF glyph space.
func (f *Font) PDFBBox() (int, int, int, int) {
	return f.scaleToPDFEm(f.xMin), f.scaleToPDFEm(f.yMin), f.scaleToPDFEm(f.xMax), f.scaleToPDFEm(f.yMax)
}

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

// FamilyNames returns CSS-friendly family names from the font's name table
// (NameIDs 1 and 16 when present). Empty when the name table is missing.
// See LoadNames for the deliberate PostScriptName fill that backs this.
func (f *Font) FamilyNames() []string {
	return f.LoadNames()
}

// LoadNames reads the name table and makes the PostScriptName fill explicit:
// when f.PostScriptName is still empty, the first NameID 6/4 record becomes
// it (the PDF /BaseFont label). Callers that need the names without the
// mutation can call this once up front. Returns family names (NameIDs 1 and
// 16), or nil when the name table is missing.
func (f *Font) LoadNames() []string {
	if f == nil {
		return nil
	}

	f.nameOnce.Do(func() {
		f.names = f.loadNames()
	})

	return f.names
}

func (f *Font) loadNames() []string {
	tbl, ok := f.tables["name"]
	if !ok || len(tbl) < 6 {
		return nil
	}

	count := int(binary.BigEndian.Uint16(tbl[2:4]))
	strOff := int(binary.BigEndian.Uint16(tbl[4:6]))
	seen := map[string]bool{}

	var out []string

	for i := range count {
		rec := sfntNameHeaderSize + i*sfntNameRecordSize
		if rec+12 > len(tbl) {
			break
		}

		nameID := binary.BigEndian.Uint16(tbl[rec+6 : rec+8])
		if !isNameRecordID(nameID) {
			continue
		}

		str, decodable := decodeNameRecord(tbl, strOff, rec)
		if !decodable {
			continue
		}

		if isFamilyNameID(nameID) {
			out = appendUnique(out, seen, str)
		}

		if f.PostScriptName == "" && isPSNameID(nameID) {
			f.PostScriptName = strings.ReplaceAll(str, " ", "")
		}
	}

	return out
}

// isNameRecordID reports whether nameID is a name we consume (family,
// PostScript or both).
func isNameRecordID(nameID uint16) bool {
	return nameID == 1 || nameID == 16 || nameID == 4 || nameID == 6
}

// isFamilyNameID reports whether nameID selects a family name (1 or 16).
func isFamilyNameID(nameID uint16) bool {
	return nameID == 1 || nameID == 16
}

// isPSNameID reports whether nameID selects a PostScript name (4 or 6).
func isPSNameID(nameID uint16) bool {
	return nameID == 6 || nameID == 4
}

// decodeNameRecord decodes one name-table record into a string when its
// platform encoding is supported.
func decodeNameRecord(tbl []byte, strOff, rec int) (string, bool) {
	platform := binary.BigEndian.Uint16(tbl[rec : rec+2])
	encoding := binary.BigEndian.Uint16(tbl[rec+2 : rec+4])
	length := int(binary.BigEndian.Uint16(tbl[rec+8 : rec+10]))
	offset := int(binary.BigEndian.Uint16(tbl[rec+10 : rec+12]))

	start := strOff + offset
	if start < 0 || start+length > len(tbl) {
		return "", false
	}

	raw := tbl[start : start+length]

	switch {
	case platform == 3 && (encoding == 1 || encoding == 10):
		return decodeUTF16BE(raw), true
	case platform == 0:
		return decodeUTF16BE(raw), true
	case platform == 1:
		return string(raw), true
	default:
		return "", false
	}
}

// appendUnique appends str to out unless it is empty or already seen.
func appendUnique(out []string, seen map[string]bool, str string) []string {
	str = strings.TrimSpace(str)
	if str == "" || seen[str] {
		return out
	}

	seen[str] = true

	return append(out, str)
}

func decodeUTF16BE(buf []byte) string {
	if len(buf)%2 != 0 {
		return string(buf)
	}

	u := make([]uint16, len(buf)/uint16Bytes)
	for i := range u {
		u[i] = binary.BigEndian.Uint16(buf[i*2:])
	}

	return string(utf16.Decode(u))
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

// GlyphAdvancePoints returns the advance width in points for r at size with
// a single cmap lookup: the same glyph table and out-of-range fallback that
// Advance uses, so the result equals AdvanceInPoints(r, size).
func (f *Font) GlyphAdvancePoints(r rune, size float64) float64 {
	g := f.GlyphID(r)

	adv := float64(f.advance[0])
	if int(g) < len(f.advance) {
		adv = float64(f.advance[g])
	}

	return adv / float64(f.unitsPerEm) * size
}

// glyphOutline returns the glyf bytes for a glyph id (raw, incl. header).
func (f *Font) glyphOutline(glob uint16) []byte {
	found, okVal := f.tables["glyf"]
	if !okVal {
		return nil
	}

	loca, okVal := f.tables["loca"]
	if !okVal {
		return nil
	}

	count := int(glob)

	var off, next int

	if f.indexToLocFmt == 0 {
		off = int(binary.BigEndian.Uint16(loca[count*uint16Bytes:])) * uint16Bytes
		next = int(binary.BigEndian.Uint16(loca[(count+1)*uint16Bytes:])) * uint16Bytes
	} else {
		off = int(binary.BigEndian.Uint32(loca[count*4:]))
		next = int(binary.BigEndian.Uint32(loca[(count+1)*4:]))
	}

	if off < 0 || next < off || next > len(found) {
		return nil
	}

	return found[off:next]
}

// compositeGlyphIDs returns the glyph ids referenced by a composite glyph.
func (f *Font) compositeGlyphIDs(g uint16) []uint16 {
	out := []uint16{}
	buf := f.glyphOutline(g)

	if len(buf) < glyfHeaderSize {
		return out
	}

	numContours := int16(binary.BigEndian.Uint16(buf[0:2])) //nolint:gosec // numContours is int16 per glyf spec
	if numContours >= 0 {
		return out // simple glyph
	}

	pos := 10

	for {
		if pos+4 > len(buf) {
			break
		}

		flags := binary.BigEndian.Uint16(buf[pos : pos+2])
		child := binary.BigEndian.Uint16(buf[pos+2 : pos+4])
		out = append(out, child)

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
