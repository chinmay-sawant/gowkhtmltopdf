package pdf

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf16"
)

// OpenType CPAL palette-type flags (version 1).
const (
	PaletteUsableWithLightBackground uint32 = 0x0001
	PaletteUsableWithDarkBackground  uint32 = 0x0002
)

const (
	cpalHeaderSize       = 12
	cpalVersion1Extra    = 12
	colrHeaderSize       = 14
	paletteDemoLightB    = 200
	paletteDemoLightG    = 80
	paletteDemoLightR    = 0
	paletteDemoDarkB     = 0
	paletteDemoDarkG     = 90
	paletteDemoDarkR     = 230
	paletteDemoAlpha     = 255
	namePlatformWin      = 3
	nameEncodingWinUni   = 1
	nameLangWinUS        = 0x0409
	nameIDFamily         = 1
	nameIDSubfamily      = 2
	nameIDFull           = 4
	nameIDPostScript     = 6
	nameIDTypoFamily     = 16
	paletteDemoSubfamily = "Regular"
)

var errNoColorPalettes = errors.New("font: no color palettes")

// PaletteDemoFamily is the CSS family name written into PaletteDemoTTF.
const PaletteDemoFamily = "Palette Demo"

// CPALColor is one OpenType CPAL color record (BGRA bytes).
type CPALColor struct {
	B, G, R, A uint8
}

// RGB01 converts the record to 0..1 sRGB used by layout paint ops.
func (c CPALColor) RGB01() (float64, float64, float64) {
	scale := float64(maxUint8)

	return float64(c.R) / scale, float64(c.G) / scale, float64(c.B) / scale
}

// ColorPalette is one CPAL palette: a row of colors plus optional type flags.
type ColorPalette struct {
	Colors []CPALColor
	Type   uint32
}

// ColorPalettes returns parsed CPAL palettes, or nil when the table is missing
// or truncated. COLR presence is not required for the parse.
func (f *Font) ColorPalettes() []ColorPalette {
	if f == nil {
		return nil
	}

	f.ensureParsed()

	return parseCPAL(f.tables["CPAL"])
}

// PaletteSolidFill is the documented lite font-palette consumer: the first
// color of the selected CPAL palette as a solid fill. ok is false for
// "normal", missing tables, or an unknown ident. Layered COLR paint is out of
// scope; PDF still embeds glyf outlines.
func (f *Font) PaletteSolidFill(name string) (float64, float64, float64, bool) {
	if f == nil || !f.HasColorPalette() {
		return 0, 0, 0, false
	}

	palettes := f.ColorPalettes()
	idx := selectPaletteIndex(palettes, name)
	if idx < 0 || idx >= len(palettes) || len(palettes[idx].Colors) == 0 {
		return 0, 0, 0, false
	}

	r, g, b := palettes[idx].Colors[0].RGB01()

	return r, g, b, true
}

func selectPaletteIndex(palettes []ColorPalette, name string) int {
	if len(palettes) == 0 {
		return -1
	}

	switch strings.ToLower(strings.TrimSpace(name)) {
	case "", "normal":
		return -1
	case "light":
		for i, palette := range palettes {
			if palette.Type&PaletteUsableWithLightBackground != 0 {
				return i
			}
		}

		return 0
	case "dark":
		for i, palette := range palettes {
			if palette.Type&PaletteUsableWithDarkBackground != 0 {
				return i
			}
		}

		if len(palettes) > 1 {
			return len(palettes) - 1
		}

		return 0
	}

	n, err := strconv.Atoi(name)
	if err != nil || n < 0 || n >= len(palettes) {
		return -1
	}

	return n
}

func parseCPAL(tbl []byte) []ColorPalette {
	if len(tbl) < cpalHeaderSize {
		return nil
	}

	version := binary.BigEndian.Uint16(tbl[0:2])
	numEntries := int(binary.BigEndian.Uint16(tbl[2:4]))
	numPalettes := int(binary.BigEndian.Uint16(tbl[4:6]))
	numColors := int(binary.BigEndian.Uint16(tbl[6:8]))
	colorOff := int(binary.BigEndian.Uint32(tbl[8:12]))

	if numEntries <= 0 || numPalettes <= 0 || numColors < numEntries {
		return nil
	}

	indexOff := cpalHeaderSize
	indexEnd := indexOff + uint16Bytes*numPalettes
	if indexEnd > len(tbl) {
		return nil
	}

	types := make([]uint32, numPalettes)
	if version >= 1 {
		typesOffPos := indexEnd
		if typesOffPos+uint32Bytes > len(tbl) {
			return nil
		}

		typesOff := int(binary.BigEndian.Uint32(tbl[typesOffPos : typesOffPos+uint32Bytes]))
		if typesOff > 0 {
			for i := range numPalettes {
				pos := typesOff + i*uint32Bytes
				if pos+uint32Bytes > len(tbl) {
					break
				}

				types[i] = binary.BigEndian.Uint32(tbl[pos : pos+uint32Bytes])
			}
		}
	}

	palettes := make([]ColorPalette, 0, numPalettes)

	for i := range numPalettes {
		start := int(binary.BigEndian.Uint16(tbl[indexOff+i*uint16Bytes:]))
		end := start + numEntries
		if start < 0 || end > numColors {
			return nil
		}

		colors := make([]CPALColor, 0, numEntries)

		for n := start; n < end; n++ {
			pos := colorOff + n*uint32Bytes
			if pos+uint32Bytes > len(tbl) {
				return nil
			}

			colors = append(colors, CPALColor{
				B: tbl[pos],
				G: tbl[pos+1],
				R: tbl[pos+2],
				A: tbl[pos+3],
			})
		}

		palettes = append(palettes, ColorPalette{Colors: colors, Type: types[i]})
	}

	return palettes
}

// WithColorPalettes rebuilds ttf with CPAL (and a minimal COLR so
// HasColorPalette is true) plus an optional family name rewrite.
func WithColorPalettes(ttf []byte, family string, palettes []ColorPalette) ([]byte, error) {
	if len(palettes) == 0 {
		return nil, errNoColorPalettes
	}

	tables, err := parseTableDirectory(ttf)
	if err != nil {
		return nil, err
	}

	built := make([]struct {
		tag  string
		data []byte
	}, 0, len(tables)+3)

	for tag, data := range tables {
		if tag == "COLR" || tag == "CPAL" {
			continue
		}

		if family != "" && tag == "name" {
			continue
		}

		built = append(built, struct {
			tag  string
			data []byte
		}{tag: tag, data: data})
	}

	if family != "" {
		ps := strings.ReplaceAll(family, " ", "") + "-" + paletteDemoSubfamily
		built = append(built, struct {
			tag  string
			data []byte
		}{tag: "name", data: buildNameTable(family, paletteDemoSubfamily, family+" "+paletteDemoSubfamily, ps)})
	}

	cpal, err := buildCPAL(palettes)
	if err != nil {
		return nil, err
	}

	built = append(built, struct {
		tag  string
		data []byte
	}{tag: "CPAL", data: cpal})
	built = append(built, struct {
		tag  string
		data []byte
	}{tag: "COLR", data: buildEmptyCOLR()})

	return buildFontFile(built)
}

// PaletteDemoTTF injects a two-palette CPAL (light blue, dark orange) and a
// "Palette Demo" name table into a TrueType face, typically Liberation Sans.
func PaletteDemoTTF(base []byte) ([]byte, error) {
	return WithColorPalettes(base, PaletteDemoFamily, []ColorPalette{
		{
			Colors: []CPALColor{{
				B: paletteDemoLightB, G: paletteDemoLightG, R: paletteDemoLightR, A: paletteDemoAlpha,
			}},
			Type: PaletteUsableWithLightBackground,
		},
		{
			Colors: []CPALColor{{
				B: paletteDemoDarkB, G: paletteDemoDarkG, R: paletteDemoDarkR, A: paletteDemoAlpha,
			}},
			Type: PaletteUsableWithDarkBackground,
		},
	})
}

func buildCPAL(palettes []ColorPalette) ([]byte, error) {
	numEntries := len(palettes[0].Colors)
	if numEntries == 0 {
		return nil, fmt.Errorf("%w: empty first palette", errNoColorPalettes)
	}

	numPalettes := len(palettes)
	records := make([]CPALColor, 0, numPalettes*numEntries)
	indices := make([]uint16, numPalettes)
	types := make([]uint32, numPalettes)

	for i, palette := range palettes {
		if len(palette.Colors) != numEntries {
			return nil, fmt.Errorf("%w: palette %d length %d, want %d", errNoColorPalettes, i, len(palette.Colors), numEntries)
		}

		//nolint:gosec // palette count is tiny and well below uint16
		indices[i] = uint16(len(records))
		records = append(records, palette.Colors...)
		types[i] = palette.Type
	}

	indexBytes := uint16Bytes * numPalettes
	headerEnd := cpalHeaderSize + indexBytes + cpalVersion1Extra
	colorOff := headerEnd
	typesOff := colorOff + uint32Bytes*len(records)
	buf := make([]byte, typesOff+uint32Bytes*numPalettes)

	binary.BigEndian.PutUint16(buf[0:2], 1)
	//nolint:gosec // palette entry counts are tiny
	binary.BigEndian.PutUint16(buf[2:4], uint16(numEntries))
	//nolint:gosec // palette counts are tiny
	binary.BigEndian.PutUint16(buf[4:6], uint16(numPalettes))
	//nolint:gosec // color record counts are tiny
	binary.BigEndian.PutUint16(buf[6:8], uint16(len(records)))
	//nolint:gosec // table is far below 4GiB
	binary.BigEndian.PutUint32(buf[8:12], uint32(colorOff))

	for i, idx := range indices {
		binary.BigEndian.PutUint16(buf[cpalHeaderSize+i*uint16Bytes:], idx)
	}

	extra := cpalHeaderSize + indexBytes
	//nolint:gosec // table is far below 4GiB
	binary.BigEndian.PutUint32(buf[extra:], uint32(typesOff))
	binary.BigEndian.PutUint32(buf[extra+uint32Bytes:], 0)
	binary.BigEndian.PutUint32(buf[extra+2*uint32Bytes:], 0)

	for i, rec := range records {
		pos := colorOff + i*uint32Bytes
		buf[pos] = rec.B
		buf[pos+1] = rec.G
		buf[pos+2] = rec.R
		buf[pos+3] = rec.A
	}

	for i, flag := range types {
		binary.BigEndian.PutUint32(buf[typesOff+i*uint32Bytes:], flag)
	}

	return buf, nil
}

func buildEmptyCOLR() []byte {
	buf := make([]byte, colrHeaderSize)
	binary.BigEndian.PutUint32(buf[4:8], colrHeaderSize)
	binary.BigEndian.PutUint32(buf[8:12], colrHeaderSize)

	return buf
}

func buildNameTable(family, subfamily, full, ps string) []byte {
	type nameRec struct {
		id  uint16
		txt string
	}

	recs := []nameRec{
		{nameIDFamily, family},
		{nameIDSubfamily, subfamily},
		{nameIDFull, full},
		{nameIDPostScript, ps},
		{nameIDTypoFamily, family},
	}

	encoded := make([][]byte, len(recs))
	strBytes := 0

	for i, rec := range recs {
		u := utf16.Encode([]rune(rec.txt))
		raw := make([]byte, len(u)*uint16Bytes)

		for j, unit := range u {
			binary.BigEndian.PutUint16(raw[j*uint16Bytes:], unit)
		}

		encoded[i] = raw
		strBytes += len(raw)
	}

	header := sfntNameHeaderSize + sfntNameRecordSize*len(recs)
	buf := make([]byte, header+strBytes)
	binary.BigEndian.PutUint16(buf[0:2], 0)
	//nolint:gosec // name record count is tiny
	binary.BigEndian.PutUint16(buf[2:4], uint16(len(recs)))
	//nolint:gosec // header size is tiny
	binary.BigEndian.PutUint16(buf[4:6], uint16(header))

	off := 0

	for i, rec := range recs {
		pos := sfntNameHeaderSize + i*sfntNameRecordSize
		binary.BigEndian.PutUint16(buf[pos:], namePlatformWin)
		binary.BigEndian.PutUint16(buf[pos+2:], nameEncodingWinUni)
		binary.BigEndian.PutUint16(buf[pos+4:], nameLangWinUS)
		binary.BigEndian.PutUint16(buf[pos+6:], rec.id)
		//nolint:gosec // name string lengths are tiny
		binary.BigEndian.PutUint16(buf[pos+8:], uint16(len(encoded[i])))
		//nolint:gosec // name string offsets are tiny
		binary.BigEndian.PutUint16(buf[pos+10:], uint16(off))
		copy(buf[header+off:], encoded[i])
		off += len(encoded[i])
	}

	return buf
}
