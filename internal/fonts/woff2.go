package fonts

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"github.com/andybalholm/brotli"
)

const (
	woff2Signature  = "wOF2"
	woff2HeaderSize = 48

	woff2TagMask      = 0x3f
	woff2TagEscape    = 0x3f
	woff2TagSize      = 4 // explicit tag length in the table directory
	woff2VersionShift = 6
	// For glyf and loca, transform version 3 is the null transform; for every
	// other table version 0 is null. hmtx defines transform version 1.
	woff2NullTransformVersion = 3
	woff2HmtxTransformVersion = 1
	woff2MaxBase128Bytes      = 5
	woff2Base128HighBits      = 0xfe000000
	woff2Base128Mask          = 0x7f
	woff2Base128Shift         = 7
)

var (
	errWOFF2TooShort      = errors.New("fonts: woff2 file too short")
	errWOFF2BadSignature  = errors.New("fonts: woff2 bad signature")
	errWOFF2TooManyTables = errors.New("fonts: woff2 too many tables")
	errWOFF2BadDirectory  = errors.New("fonts: woff2 bad table directory")
	errWOFF2SFNTTooLarge  = errors.New("fonts: woff2 reconstructed SFNT too large")
	errWOFF2BadTable      = errors.New("fonts: woff2 table empty or too large")
	errWOFF2Transform     = errors.New("fonts: woff2 transformed glyf/loca/hmtx unsupported (null transform only)")
	errWOFF2CFF           = errors.New("fonts: woff2 CFF/OTTO not supported (TrueType outlines only)")
	errWOFF2Length        = errors.New("fonts: woff2 brotli output length mismatch")
)

// woff2KnownTags maps the 6-bit tag index from the WOFF2 specification.
//
//nolint:gochecknoglobals // immutable WOFF2 vocabulary
var woff2KnownTags = [woff2TagMask]string{
	"cmap", "head", "hhea", "hmtx", "maxp", "name", "OS/2", "post",
	"cvt ", "fpgm", "glyf", "loca", "prep", "CFF ", "VORG", "EBDT",
	"EBLC", "gasp", "hdmx", "kern", "LTSH", "PCLT", "VDMX", "vhea",
	"vmtx", "BASE", "GDEF", "GPOS", "GSUB", "EBSC", "JSTF", "MATH",
	"CBDT", "CBLC", "COLR", "CPAL", "SVG ", "sbix", "acnt", "avar",
	"bdat", "bloc", "bsln", "cvar", "fdsc", "feat", "fmtx", "fvar",
	"gvar", "hsty", "just", "lcar", "mort", "morx", "opbd", "prop",
	"trak", "Zapf", "Silf", "Glat", "Gloc", "Feat", "Sill",
}

// woff2Table is one WOFF2 table directory entry. Only untransformed tables
// reach the assembler, so the stored length equals origLength.
type woff2Table struct {
	tag        string
	origLength uint32
}

// decodeWOFF2Header validates the fixed WOFF2 header and returns the flavor,
// table count, and compressed data size.
func decodeWOFF2Header(data []byte) (uint32, int, uint32, error) {
	if len(data) < woff2HeaderSize {
		return 0, 0, 0, errWOFF2TooShort
	}

	if string(data[0:4]) != woff2Signature {
		return 0, 0, 0, errWOFF2BadSignature
	}

	flavor := binary.BigEndian.Uint32(data[4:8])
	if flavor == ottoFlavor {
		return 0, 0, 0, errWOFF2CFF
	}

	numTables := int(binary.BigEndian.Uint16(data[12:14]))
	if numTables <= 0 || numTables > maxTables {
		return 0, 0, 0, errWOFF2TooManyTables
	}

	totalSFNT := binary.BigEndian.Uint32(data[16:20])
	if totalSFNT == 0 || totalSFNT > maxSFNTSize {
		return 0, 0, 0, errWOFF2SFNTTooLarge
	}

	return flavor, numTables, binary.BigEndian.Uint32(data[20:24]), nil
}

// DecodeWOFF2 reconstructs a WOFF2 file into SFNT bytes. The glyf/loca/hmtx
// transforms are not implemented: files whose glyf, loca, or hmtx table
// carries a transform are rejected with errWOFF2Transform, so only
// null-transform fonts decode. CFF/OTTO flavor is rejected like the WOFF1 path.
func DecodeWOFF2(data []byte) ([]byte, error) {
	flavor, numTables, compressedSize, err := decodeWOFF2Header(data)
	if err != nil {
		return nil, err
	}

	tables, dataOff, transformed, err := parseWOFF2Directory(data, numTables)
	if err != nil {
		return nil, err
	}

	if transformed {
		return nil, errWOFF2Transform
	}

	compressedLen := int(compressedSize)
	if compressedLen > len(data)-dataOff {
		return nil, errWOFF2TooShort
	}

	var want uint64
	for _, table := range tables {
		want += uint64(table.origLength)
	}

	compressed := data[dataOff : dataOff+compressedLen]

	plain, err := decompressWOFF2(compressed, want)
	if err != nil {
		return nil, err
	}

	parts := splitWOFF2Tables(plain, tables)

	return assembleWOFF2SFNT(flavor, tables, parts)
}

// parseWOFF2Directory reads every table directory entry and returns the
// directory, the compressed-data offset, and whether any table is transformed.
func parseWOFF2Directory(data []byte, numTables int) ([]woff2Table, int, bool, error) {
	tables := make([]woff2Table, 0, numTables)
	transformed := false
	off := woff2HeaderSize

	var total uint64

	for range numTables {
		table, next, entryTransformed, err := parseWOFF2Entry(data, off)
		if err != nil {
			return nil, 0, false, err
		}

		tables = append(tables, table)
		transformed = transformed || entryTransformed
		off = next

		total += uint64(table.origLength)
		if total > maxSFNTSize {
			return nil, 0, false, errWOFF2SFNTTooLarge
		}
	}

	return tables, off, transformed, nil
}

// parseWOFF2Entry reads one flags byte, the tag, origLength, and the
// transformLength when the table carries a transform.
func parseWOFF2Entry(data []byte, off int) (woff2Table, int, bool, error) {
	if off >= len(data) {
		return woff2Table{}, 0, false, errWOFF2TooShort
	}

	flags := data[off]
	off++

	tag, off, err := woff2Tag(data, flags, off)
	if err != nil {
		return woff2Table{}, 0, false, err
	}

	orig, off, err := readUIntBase128(data, off)
	if err != nil {
		return woff2Table{}, 0, false, err
	}

	if orig == 0 || orig > maxTableLen {
		return woff2Table{}, 0, false, errWOFF2BadTable
	}

	transformed, err := woff2TableTransform(tag, flags>>woff2VersionShift)
	if err != nil {
		return woff2Table{}, 0, false, err
	}

	if transformed {
		// The length is read so the directory stays aligned; the transform
		// itself is rejected after the directory pass.
		_, off, err = readUIntBase128(data, off)
		if err != nil {
			return woff2Table{}, 0, false, err
		}
	}

	return woff2Table{tag: tag, origLength: orig}, off, transformed, nil
}

// woff2TableTransform reports whether the table's version selects a transform.
// glyf and loca use version 3 as the null transform, hmtx uses version 1, and
// every other table must use version 0.
func woff2TableTransform(tag string, version byte) (bool, error) {
	switch tag {
	case "glyf", "loca":
		return version != woff2NullTransformVersion, nil
	case "hmtx":
		if version != 0 && version != woff2HmtxTransformVersion {
			return false, errWOFF2BadDirectory
		}

		return version != 0, nil
	default:
		if version != 0 {
			return false, errWOFF2BadDirectory
		}

		return false, nil
	}
}

// woff2Tag resolves the known-tag index or reads the explicit 4-byte tag.
func woff2Tag(data []byte, flags byte, off int) (string, int, error) {
	idx := flags & woff2TagMask
	if idx != woff2TagEscape {
		return woff2KnownTags[idx], off, nil
	}

	if off+woff2TagSize > len(data) {
		return "", 0, errWOFF2TooShort
	}

	return string(data[off : off+woff2TagSize]), off + woff2TagSize, nil
}

// readUIntBase128 decodes the WOFF2 variable-length unsigned integer.
func readUIntBase128(data []byte, off int) (uint32, int, error) {
	var value uint32

	for idx := range woff2MaxBase128Bytes {
		if off+idx >= len(data) {
			return 0, 0, errWOFF2TooShort
		}

		current := data[off+idx]
		if idx == 0 && current == 0x80 {
			return 0, 0, errWOFF2BadDirectory
		}

		if value&woff2Base128HighBits != 0 {
			return 0, 0, errWOFF2BadDirectory
		}

		value = value<<woff2Base128Shift | uint32(current&woff2Base128Mask)

		if current&0x80 == 0 {
			return value, off + idx + 1, nil
		}
	}

	return 0, 0, errWOFF2BadDirectory
}

// decompressWOFF2 Brotli-decodes the table block, capping the output at the
// declared original table sizes.
func decompressWOFF2(compressed []byte, want uint64) ([]byte, error) {
	reader := brotli.NewReader(bytes.NewReader(compressed))
	limited := io.LimitReader(reader, int64(want)+1) //nolint:gosec // want is capped by maxSFNTSize

	plain, err := io.ReadAll(limited)
	if err != nil {
		return nil, fmt.Errorf("fonts: woff2 brotli: %w", err)
	}

	if uint64(len(plain)) != want {
		return nil, errWOFF2Length
	}

	return plain, nil
}

// splitWOFF2Tables cuts the decompressed block into tables in directory order.
// Untransformed tables store origLength bytes each.
func splitWOFF2Tables(plain []byte, tables []woff2Table) [][]byte {
	parts := make([][]byte, len(tables))
	off := 0

	for i, table := range tables {
		end := off + int(table.origLength)
		parts[i] = plain[off:end]
		off = end
	}

	return parts
}
