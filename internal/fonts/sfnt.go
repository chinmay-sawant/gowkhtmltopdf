package fonts

import "encoding/binary"

const (
	sfntHeaderSize = 12
	sfntRecordSize = 16
	sfntSearchMul  = 16
	sfntWordSize   = 4
	sfntPadMask    = 3
	ottoFlavor     = 0x4F54544F // 'OTTO'
)

// assembleWOFF2SFNT builds an SFNT file from the decompressed, untransformed
// WOFF2 tables. Table checksums are recomputed; head.checkSumAdjustment is
// left to the PDF subsetter, matching the WOFF1 path.
func assembleWOFF2SFNT(flavor uint32, tables []woff2Table, data [][]byte) ([]byte, error) {
	numTables := len(tables)
	headerSize := sfntHeaderSize + sfntRecordSize*numTables

	total := headerSize

	for _, table := range data {
		total += paddedLen(len(table))
	}

	if total > maxSFNTSize {
		return nil, errWOFF2SFNTTooLarge
	}

	out := make([]byte, total)
	binary.BigEndian.PutUint32(out[0:4], flavor)
	binary.BigEndian.PutUint16(out[4:6], uint16(numTables)) //nolint:gosec // bounded by maxTables

	searchRange := 1
	entrySelector := 0

	for searchRange*2 <= numTables {
		searchRange *= 2
		entrySelector++
	}

	rangeShift := numTables*sfntSearchMul - searchRange*sfntSearchMul

	binary.BigEndian.PutUint16(out[6:8], uint16(searchRange*sfntSearchMul)) //nolint:gosec // bounded by maxTables
	binary.BigEndian.PutUint16(out[8:10], uint16(entrySelector))            //nolint:gosec // bounded by maxTables
	binary.BigEndian.PutUint16(out[10:12], uint16(rangeShift))              //nolint:gosec // bounded by maxTables

	offset := uint32(headerSize) //nolint:gosec // bounded by maxSFNTSize

	for idx, table := range tables {
		rec := out[sfntHeaderSize+sfntRecordSize*idx:]
		copy(rec[0:4], table.tag)
		binary.BigEndian.PutUint32(rec[4:8], sfntChecksum(data[idx]))
		binary.BigEndian.PutUint32(rec[8:12], offset)
		binary.BigEndian.PutUint32(rec[12:16], uint32(len(data[idx]))) //nolint:gosec // bounded by maxTableLen
		copy(out[offset:], data[idx])
		offset += uint32(paddedLen(len(data[idx]))) //nolint:gosec // bounded by maxTableLen
	}

	return out, nil
}

// paddedLen rounds a table length up to the SFNT 4-byte alignment.
func paddedLen(n int) int {
	return (n + sfntPadMask) &^ sfntPadMask
}

// sfntChecksum sums the table as big-endian uint32 words, zero-padding the
// final partial word.
func sfntChecksum(table []byte) uint32 {
	var sum uint32

	for pos := 0; pos < len(table); pos += sfntWordSize {
		var word [sfntWordSize]byte

		end := pos + sfntWordSize
		if end > len(table) {
			end = len(table)
		}

		copy(word[:], table[pos:end])
		sum += binary.BigEndian.Uint32(word[:])
	}

	return sum
}
