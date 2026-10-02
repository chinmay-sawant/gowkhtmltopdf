package fonts

import "encoding/binary"

const (
	sfntHeaderSize = 12
	sfntRecordSize = 16
	sfntSearchMul  = 16
	sfntPadMask    = 3
	ottoFlavor     = 0x4F54544F // 'OTTO'
)

// assembleWOFF2SFNT builds an SFNT file from the decompressed, untransformed
// WOFF2 tables. Table checksums are recomputed; head.checkSumAdjustment is
// left to the PDF subsetter, matching the WOFF1 path.
func assembleWOFF2SFNT(flavor uint32, tables []woff2Table, data [][]byte) ([]byte, error) {
	numTables := len(tables)
	headerSize := sfntHeaderSize + sfntRecordSize*numTables

	total := uint64(headerSize)

	for _, table := range data {
		total += uint64(paddedLen(len(table)))
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

	binary.BigEndian.PutUint16(out[6:8], uint16(searchRange*sfntSearchMul))                           //nolint:gosec // bounded by maxTables
	binary.BigEndian.PutUint16(out[8:10], uint16(entrySelector))                                      //nolint:gosec // bounded by maxTables
	binary.BigEndian.PutUint16(out[10:12], uint16(numTables*sfntSearchMul-searchRange*sfntSearchMul)) //nolint:gosec // bounded by maxTables

	offset := uint32(headerSize) //nolint:gosec // bounded by maxSFNTSize

	for i, table := range tables {
		rec := out[sfntHeaderSize+sfntRecordSize*i:]
		copy(rec[0:4], table.tag)
		binary.BigEndian.PutUint32(rec[4:8], sfntChecksum(data[i]))
		binary.BigEndian.PutUint32(rec[8:12], offset)
		binary.BigEndian.PutUint32(rec[12:16], uint32(len(data[i]))) //nolint:gosec // bounded by maxTableLen
		copy(out[offset:], data[i])
		offset += uint32(paddedLen(len(data[i]))) //nolint:gosec // bounded by maxTableLen
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

	for i := 0; i < len(table); i += 4 {
		var word [4]byte

		end := i + 4
		if end > len(table) {
			end = len(table)
		}

		copy(word[:], table[i:end])
		sum += binary.BigEndian.Uint32(word[:])
	}

	return sum
}
