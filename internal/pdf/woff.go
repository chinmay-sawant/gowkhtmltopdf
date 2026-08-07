package pdf

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
)

const (
	woffSignature  = "wOFF"
	woff2Signature = "wOF2"
	woffHeaderSize = 44
	woffEntrySize  = 20

	// Caps for untrusted @font-face payloads (decompress-bomb / DoS).
	woffMaxTables   = 1024
	woffMaxSFNTSize = 32 << 20 // 32 MiB reconstructed SFNT
	woffMaxTableLen = 16 << 20 // 16 MiB per table
)

var (
	errWOFFTooShort      = errors.New("woff: file too short")
	errWOFFBadSignature  = errors.New("woff: bad signature")
	errWOFFTooManyTables = errors.New("woff: too many tables")
	errWOFFSFNTTooLarge  = errors.New("woff: reconstructed SFNT too large")
	errWOFFBadOffset     = errors.New("woff: bad table offset or length")
	errWOFFOverlap       = errors.New("woff: overlapping compressed tables")
	errWOFF2Unsupported  = errors.New("woff2: decode requires Brotli (not available; typesetting has WOFF1 only; no new direct modules)")
	errWOFFFlavorCFF     = errors.New("woff: CFF/OTTO OpenType not supported (TrueType outlines only)")
)

// ParseFontBytes parses TTF/OTF (TrueType outlines) or WOFF1-wrapped SFNT.
// WOFF2 is rejected with a clear error (Brotli not allowlisted).
//
// ponytail: WOFF1 in-tree only; no Brotli / WOFF2 direct dep.
func ParseFontBytes(data []byte) (*Font, error) {
	if len(data) >= tagSize {
		sig := string(data[0:4])
		switch sig {
		case woffSignature:
			sfnt, err := DecodeWOFF(data)
			if err != nil {
				return nil, err
			}

			data = sfnt
		case woff2Signature:
			return nil, errWOFF2Unsupported
		}
	}

	return ParseTTF(data)
}

// DecodeWOFF decompresses a WOFF1 file into SFNT (TrueType/OpenType) bytes
// using stdlib compress/zlib. CFF/OTTO flavor is rejected.
func DecodeWOFF(data []byte) ([]byte, error) {
	if len(data) < woffHeaderSize {
		return nil, errWOFFTooShort
	}

	if string(data[0:4]) != woffSignature {
		return nil, errWOFFBadSignature
	}

	flavor := binary.BigEndian.Uint32(data[4:8])
	if flavor == ottoFlavor { // 'OTTO'
		return nil, errWOFFFlavorCFF
	}

	numTables := int(binary.BigEndian.Uint16(data[12:14]))
	if numTables <= 0 || numTables > woffMaxTables {
		return nil, errWOFFTooManyTables
	}

	totalSfntSize := binary.BigEndian.Uint32(data[16:20])
	if totalSfntSize == 0 || totalSfntSize > woffMaxSFNTSize {
		return nil, errWOFFSFNTTooLarge
	}

	dirEnd := woffHeaderSize + numTables*woffEntrySize
	if dirEnd > len(data) {
		return nil, errWOFFTooShort
	}

	type entry struct {
		tag                [4]byte
		offset, comp, orig uint32
		checksum           uint32
	}

	entries := make([]entry, numTables)

	type span struct{ start, end uint32 }

	spans := make([]span, 0, numTables)

	for idx := range numTables {
		off := woffHeaderSize + idx*woffEntrySize
		rec := data[off : off+woffEntrySize]
		copy(entries[idx].tag[:], rec[0:4])
		entries[idx].offset = binary.BigEndian.Uint32(rec[4:8])
		entries[idx].comp = binary.BigEndian.Uint32(rec[8:12])
		entries[idx].orig = binary.BigEndian.Uint32(rec[12:16])
		entries[idx].checksum = binary.BigEndian.Uint32(rec[16:20])

		entry := entries[idx]
		if entry.orig == 0 || entry.orig > woffMaxTableLen {
			return nil, errWOFFBadOffset
		}

		if entry.comp == 0 || uint64(entry.offset)+uint64(entry.comp) > uint64(len(data)) {
			return nil, errWOFFBadOffset
		}

		if entry.comp > entry.orig {
			return nil, errWOFFBadOffset
		}

		end := entry.offset + entry.comp
		for _, s := range spans {
			if entry.offset < s.end && end > s.start {
				return nil, errWOFFOverlap
			}
		}

		spans = append(spans, span{start: entry.offset, end: end})
	}

	tables := make([][]byte, numTables)

	var sumOrig uint64

	for idx, entry := range entries {
		raw := data[entry.offset : entry.offset+entry.comp]

		var plain []byte

		if entry.comp < entry.orig {
			zreader, err := zlib.NewReader(bytes.NewReader(raw))
			if err != nil {
				return nil, fmt.Errorf("woff: zlib: %w", err)
			}
			// Cap read at origLength; reject trailing junk / under-read.
			limited := io.LimitReader(zreader, int64(entry.orig)+1)
			plain, err = io.ReadAll(limited)

			zreader.Close()

			if err != nil {
				return nil, fmt.Errorf("woff: decompress: %w", err)
			}

			if uint32(len(plain)) != entry.orig {
				return nil, fmt.Errorf("woff: table %q decompressed length %d != %d", entry.tag, len(plain), entry.orig)
			}
		} else {
			plain = bytes.Clone(raw)
		}

		tables[idx] = plain

		sumOrig += uint64(entry.orig)
		if sumOrig > woffMaxSFNTSize {
			return nil, errWOFFSFNTTooLarge
		}
	}

	// SFNT header + directory + 4-byte-padded tables.
	headerSize := sfntOffsetTableSize + sfntTableRecordSize*numTables

	var dataSize uint64
	for _, t := range tables {
		dataSize += uint64((len(t) + padMask3) &^ padMask3)
	}

	total := uint64(headerSize) + dataSize
	if total > woffMaxSFNTSize || total > math.MaxInt32 {
		return nil, errWOFFSFNTTooLarge
	}

	out := make([]byte, total)
	binary.BigEndian.PutUint32(out[0:4], flavor)
	binary.BigEndian.PutUint16(out[4:6], uint16(numTables))
	// searchRange / entrySelector / rangeShift (OpenType)
	searchR := uint16(1)
	entrySel := uint16(0)

	for searchR*2 <= uint16(numTables) {
		searchR *= 2
		entrySel++
	}

	binary.BigEndian.PutUint16(out[6:8], searchR*sfntSearchRangeMul)
	binary.BigEndian.PutUint16(out[8:10], entrySel)
	binary.BigEndian.PutUint16(out[10:12], uint16(numTables)*16-searchR*16)

	tableOffset := uint32(headerSize)

	for idx, e := range entries {
		rec := out[12+16*idx:]
		copy(rec[0:4], e.tag[:])
		binary.BigEndian.PutUint32(rec[4:8], e.checksum)
		binary.BigEndian.PutUint32(rec[8:12], tableOffset)
		binary.BigEndian.PutUint32(rec[12:16], uint32(len(tables[idx])))
		copy(out[tableOffset:], tables[idx])
		padded := uint32((len(tables[idx]) + padMask3) &^ padMask3)
		tableOffset += padded
	}

	return out, nil
}
