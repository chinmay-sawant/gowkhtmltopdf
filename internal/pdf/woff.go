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
func ParseFontBytes(data []byte) (*Font, error) {
	if len(data) >= 4 {
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
	if flavor == 0x4F54544F { // 'OTTO'
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

	for i := 0; i < numTables; i++ {
		off := woffHeaderSize + i*woffEntrySize
		rec := data[off : off+woffEntrySize]
		copy(entries[i].tag[:], rec[0:4])
		entries[i].offset = binary.BigEndian.Uint32(rec[4:8])
		entries[i].comp = binary.BigEndian.Uint32(rec[8:12])
		entries[i].orig = binary.BigEndian.Uint32(rec[12:16])
		entries[i].checksum = binary.BigEndian.Uint32(rec[16:20])

		e := entries[i]
		if e.orig == 0 || e.orig > woffMaxTableLen {
			return nil, errWOFFBadOffset
		}
		if e.comp == 0 || uint64(e.offset)+uint64(e.comp) > uint64(len(data)) {
			return nil, errWOFFBadOffset
		}
		if e.comp > e.orig {
			return nil, errWOFFBadOffset
		}
		end := e.offset + e.comp
		for _, s := range spans {
			if e.offset < s.end && end > s.start {
				return nil, errWOFFOverlap
			}
		}
		spans = append(spans, span{start: e.offset, end: end})
	}

	tables := make([][]byte, numTables)
	var sumOrig uint64
	for i, e := range entries {
		raw := data[e.offset : e.offset+e.comp]
		var plain []byte
		if e.comp < e.orig {
			zr, err := zlib.NewReader(bytes.NewReader(raw))
			if err != nil {
				return nil, fmt.Errorf("woff: zlib: %w", err)
			}
			// Cap read at origLength; reject trailing junk / under-read.
			limited := io.LimitReader(zr, int64(e.orig)+1)
			plain, err = io.ReadAll(limited)
			zr.Close()
			if err != nil {
				return nil, fmt.Errorf("woff: decompress: %w", err)
			}
			if uint32(len(plain)) != e.orig {
				return nil, fmt.Errorf("woff: table %q decompressed length %d != %d", e.tag, len(plain), e.orig)
			}
		} else {
			plain = bytes.Clone(raw)
		}
		tables[i] = plain
		sumOrig += uint64(e.orig)
		if sumOrig > woffMaxSFNTSize {
			return nil, errWOFFSFNTTooLarge
		}
	}

	// SFNT header + directory + 4-byte-padded tables.
	headerSize := 12 + 16*numTables
	var dataSize uint64
	for _, t := range tables {
		dataSize += uint64((len(t) + 3) &^ 3)
	}
	total := uint64(headerSize) + dataSize
	if total > woffMaxSFNTSize || total > math.MaxInt32 {
		return nil, errWOFFSFNTTooLarge
	}

	out := make([]byte, total)
	binary.BigEndian.PutUint32(out[0:4], flavor)
	binary.BigEndian.PutUint16(out[4:6], uint16(numTables))
	// searchRange / entrySelector / rangeShift (OpenType)
	sr := uint16(1)
	es := uint16(0)
	for sr*2 <= uint16(numTables) {
		sr *= 2
		es++
	}
	binary.BigEndian.PutUint16(out[6:8], sr*16)
	binary.BigEndian.PutUint16(out[8:10], es)
	binary.BigEndian.PutUint16(out[10:12], uint16(numTables)*16-sr*16)

	tableOffset := uint32(headerSize)
	for i, e := range entries {
		rec := out[12+16*i:]
		copy(rec[0:4], e.tag[:])
		binary.BigEndian.PutUint32(rec[4:8], e.checksum)
		binary.BigEndian.PutUint32(rec[8:12], tableOffset)
		binary.BigEndian.PutUint32(rec[12:16], uint32(len(tables[i])))
		copy(out[tableOffset:], tables[i])
		padded := uint32((len(tables[i]) + 3) &^ 3)
		tableOffset += padded
	}
	return out, nil
}

// DecodeWOFF2 documents the WOFF2 gap: Brotli is not in stdlib and not an
// allowlisted direct module; go-text/typesetting only reads WOFF1.
func DecodeWOFF2(data []byte) ([]byte, error) {
	if len(data) < 4 || string(data[0:4]) != woff2Signature {
		return nil, errWOFFBadSignature
	}
	return nil, errWOFF2Unsupported
}
