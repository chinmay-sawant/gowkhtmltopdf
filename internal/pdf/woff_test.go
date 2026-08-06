package pdf

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func readLiberationTTF(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("assets", "LiberationSans-Regular.ttf"))
	if err != nil {
		t.Fatalf("read liberation: %v", err)
	}
	return data
}

// encodeWOFF1ForTest builds a minimal valid WOFF1 from SFNT tables (zlib).
func encodeWOFF1ForTest(t *testing.T, sfnt []byte) []byte {
	t.Helper()
	if len(sfnt) < 12 {
		t.Fatal("sfnt too short")
	}
	flavor := binary.BigEndian.Uint32(sfnt[0:4])
	numTables := int(binary.BigEndian.Uint16(sfnt[4:6]))
	type tab struct {
		tag            [4]byte
		offset, length uint32
		checksum       uint32
	}
	tabs := make([]tab, numTables)
	for i := 0; i < numTables; i++ {
		rec := sfnt[12+16*i:]
		copy(tabs[i].tag[:], rec[0:4])
		tabs[i].checksum = binary.BigEndian.Uint32(rec[4:8])
		tabs[i].offset = binary.BigEndian.Uint32(rec[8:12])
		tabs[i].length = binary.BigEndian.Uint32(rec[12:16])
	}

	var compressed [][]byte
	var origLens []uint32
	var compLens []uint32
	for _, tb := range tabs {
		raw := sfnt[tb.offset : tb.offset+tb.length]
		var buf bytes.Buffer
		zw := zlib.NewWriter(&buf)
		if _, err := zw.Write(raw); err != nil {
			t.Fatalf("zlib write: %v", err)
		}
		if err := zw.Close(); err != nil {
			t.Fatalf("zlib close: %v", err)
		}
		comp := buf.Bytes()
		if len(comp) >= len(raw) {
			comp = bytes.Clone(raw)
		}
		compressed = append(compressed, comp)
		origLens = append(origLens, tb.length)
		compLens = append(compLens, uint32(len(comp)))
	}

	header := make([]byte, woffHeaderSize)
	copy(header[0:4], []byte(woffSignature))
	binary.BigEndian.PutUint32(header[4:8], flavor)
	binary.BigEndian.PutUint16(header[12:14], uint16(numTables))
	binary.BigEndian.PutUint32(header[16:20], uint32(len(sfnt)))

	dir := make([]byte, numTables*woffEntrySize)
	payloadOff := uint32(woffHeaderSize + numTables*woffEntrySize)
	var body bytes.Buffer
	for i, tb := range tabs {
		// Align each table start to 4 bytes (WOFF).
		for payloadOff%4 != 0 {
			body.WriteByte(0)
			payloadOff++
		}
		rec := dir[i*woffEntrySize : (i+1)*woffEntrySize]
		copy(rec[0:4], tb.tag[:])
		binary.BigEndian.PutUint32(rec[4:8], payloadOff)
		binary.BigEndian.PutUint32(rec[8:12], compLens[i])
		binary.BigEndian.PutUint32(rec[12:16], origLens[i])
		binary.BigEndian.PutUint32(rec[16:20], tb.checksum)
		body.Write(compressed[i])
		payloadOff += compLens[i]
	}

	out := append(append(header, dir...), body.Bytes()...)
	binary.BigEndian.PutUint32(out[8:12], uint32(len(out)))
	return out
}

func TestDecodeWOFFRoundTripParseTTF(t *testing.T) {
	sfnt := readLiberationTTF(t)
	woff := encodeWOFF1ForTest(t, sfnt)
	got, err := DecodeWOFF(woff)
	if err != nil {
		t.Fatalf("DecodeWOFF: %v", err)
	}
	f, err := ParseTTF(got)
	if err != nil {
		t.Fatalf("ParseTTF after WOFF: %v", err)
	}
	if f.GlyphID('A') == 0 {
		t.Fatal("expected glyph for 'A'")
	}
	f2, err := ParseFontBytes(woff)
	if err != nil {
		t.Fatalf("ParseFontBytes WOFF: %v", err)
	}
	if f2.GlyphID('A') == 0 {
		t.Fatal("ParseFontBytes: expected glyph for 'A'")
	}
}

func TestDecodeWOFFRejectsOTTO(t *testing.T) {
	// Minimal fake WOFF with OTTO flavor.
	buf := make([]byte, woffHeaderSize)
	copy(buf[0:4], []byte(woffSignature))
	copy(buf[4:8], []byte("OTTO"))
	binary.BigEndian.PutUint16(buf[12:14], 1)
	binary.BigEndian.PutUint32(buf[16:20], 100)
	_, err := DecodeWOFF(buf)
	if !errors.Is(err, errWOFFFlavorCFF) {
		t.Fatalf("got %v, want errWOFFFlavorCFF", err)
	}
}

func TestDecodeWOFFRejectsOverlap(t *testing.T) {
	sfnt := readLiberationTTF(t)
	woff := encodeWOFF1ForTest(t, sfnt)
	// Corrupt second table offset to overlap the first compressed span.
	numTables := int(binary.BigEndian.Uint16(woff[12:14]))
	if numTables < 2 {
		t.Skip("need ≥2 tables")
	}
	firstOff := binary.BigEndian.Uint32(woff[woffHeaderSize+4 : woffHeaderSize+8])
	rec2 := woffHeaderSize + woffEntrySize
	binary.BigEndian.PutUint32(woff[rec2+4:rec2+8], firstOff) // same offset → overlap
	_, err := DecodeWOFF(woff)
	if !errors.Is(err, errWOFFOverlap) {
		t.Fatalf("got %v, want errWOFFOverlap", err)
	}
}

func TestDecodeWOFF2Gap(t *testing.T) {
	// Concrete gap: WOFF2 needs Brotli; typesetting has no WOFF2 reader and we
	// do not add direct modules. ParseFontBytes rejects wOF2 with a clear error.
	buf := []byte("wOF2....fake...")
	_, err := ParseFontBytes(buf)
	if !errors.Is(err, errWOFF2Unsupported) {
		t.Fatalf("ParseFontBytes WOFF2: got %v, want errWOFF2Unsupported", err)
	}
}

func TestParseFontBytesTTFUnchanged(t *testing.T) {
	sfnt := readLiberationTTF(t)
	f, err := ParseFontBytes(sfnt)
	if err != nil {
		t.Fatal(err)
	}
	if f.GlyphID('B') == 0 {
		t.Fatal("expected glyph B")
	}
}
