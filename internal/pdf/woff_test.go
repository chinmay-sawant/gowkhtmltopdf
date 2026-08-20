//nolint:testpackage // tests reach into unexported state
package pdf

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// woffTestTable mirrors one SFNT table directory record.
type woffTestTable struct {
	tag            [4]byte
	offset, length uint32
	checksum       uint32
}

// readSFNTDirectory extracts the SFNT table directory.
func readSFNTDirectory(t *testing.T, sfnt []byte, numTables int) []woffTestTable {
	t.Helper()

	tabs := make([]woffTestTable, numTables)

	for i := range numTables {
		rec := sfnt[12+16*i:]
		copy(tabs[i].tag[:], rec[0:4])
		tabs[i].checksum = binary.BigEndian.Uint32(rec[4:8])
		tabs[i].offset = binary.BigEndian.Uint32(rec[8:12])
		tabs[i].length = binary.BigEndian.Uint32(rec[12:16])
	}

	return tabs
}

// compressSFNTTables zlib-compresses each table, keeping incompressible
// tables verbatim.
func compressSFNTTables(t *testing.T, tabs []woffTestTable, sfnt []byte) ([][]byte, []uint32, []uint32) {
	t.Helper()

	compressed := make([][]byte, 0, len(tabs))
	origLens := make([]uint32, 0, len(tabs))
	compLens := make([]uint32, 0, len(tabs))

	for _, table := range tabs {
		raw := sfnt[table.offset : table.offset+table.length]

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
		origLens = append(origLens, table.length)
		compLens = append(compLens, uint32(len(comp))) //nolint:gosec // test fixture sizes are small
	}

	return compressed, origLens, compLens
}

// buildWOFFPayload lays out the aligned table directory and body.
func buildWOFFPayload(
	t *testing.T,
	tabs []woffTestTable,
	compressed [][]byte,
	compLens, origLens []uint32,
) ([]byte, []byte) {
	t.Helper()

	dir := make([]byte, len(tabs)*woffEntrySize)
	payloadOff := uint32(woffHeaderSize + len(tabs)*woffEntrySize) //nolint:gosec // test fixture

	var body bytes.Buffer

	for idx, table := range tabs {
		// Align each table start to 4 bytes (WOFF).
		for payloadOff%4 != 0 {
			body.WriteByte(0)

			payloadOff++
		}

		rec := dir[idx*woffEntrySize : (idx+1)*woffEntrySize]
		copy(rec[0:4], table.tag[:])
		binary.BigEndian.PutUint32(rec[4:8], payloadOff)
		binary.BigEndian.PutUint32(rec[8:12], compLens[idx])
		binary.BigEndian.PutUint32(rec[12:16], origLens[idx])
		binary.BigEndian.PutUint32(rec[16:20], table.checksum)
		body.Write(compressed[idx])
		payloadOff += compLens[idx]
	}

	return dir, body.Bytes()
}

// encodeWOFF1ForTest builds a minimal valid WOFF1 from SFNT tables (zlib).
func encodeWOFF1ForTest(t *testing.T, sfnt []byte) []byte {
	t.Helper()

	if len(sfnt) < 12 {
		t.Fatal("sfnt too short")
	}

	flavor := binary.BigEndian.Uint32(sfnt[0:4])
	numTables := int(binary.BigEndian.Uint16(sfnt[4:6]))
	tabs := readSFNTDirectory(t, sfnt, numTables)
	compressed, origLens, compLens := compressSFNTTables(t, tabs, sfnt)

	header := make([]byte, woffHeaderSize)
	copy(header[0:4], []byte(woffSignature))
	binary.BigEndian.PutUint32(header[4:8], flavor)
	binary.BigEndian.PutUint16(header[12:14], uint16(numTables)) //nolint:gosec // test fixture
	binary.BigEndian.PutUint32(header[16:20], uint32(len(sfnt))) //nolint:gosec // test fixture

	dir, body := buildWOFFPayload(t, tabs, compressed, compLens, origLens)

	out := bytes.Join([][]byte{header, dir, body}, nil)
	binary.BigEndian.PutUint32(out[8:12], uint32(len(out))) //nolint:gosec // test fixture

	return out
}

func TestDecodeWOFFRoundTripParseTTF(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestDecodeWOFF2RoundTrip(t *testing.T) {
	t.Parallel()

	woff2, err := os.ReadFile(filepath.Join("..", "..", "testdata", "fonts", "LiberationSans-Regular.woff2"))
	if err != nil {
		t.Fatalf("read woff2 fixture: %v", err)
	}

	f, err := ParseFontBytes(woff2)
	if err != nil {
		t.Fatalf("ParseFontBytes WOFF2: %v", err)
	}

	if f.GlyphID('B') == 0 {
		t.Fatal("expected glyph B after WOFF2 decode")
	}
}

func TestDecodeWOFF2RejectsGarbage(t *testing.T) {
	t.Parallel()

	buf := []byte("wOF2....fake...")

	_, err := ParseFontBytes(buf)
	if err == nil {
		t.Fatal("expected error for garbage WOFF2")
	}

	if !errors.Is(err, errWOFF2Invalid) && !errors.Is(err, errWOFFBadSignature) &&
		!errors.Is(err, errWOFF2Collection) {
		// Accept any wrapped woff2 invalid error.
		if !strings.Contains(strings.ToLower(err.Error()), "woff2") {
			t.Fatalf("ParseFontBytes garbage WOFF2: %v", err)
		}
	}
}

func TestParseFontBytesTTFUnchanged(t *testing.T) {
	t.Parallel()
	sfnt := readLiberationTTF(t)

	f, err := ParseFontBytes(sfnt)
	if err != nil {
		t.Fatal(err)
	}

	if f.GlyphID('B') == 0 {
		t.Fatal("expected glyph B")
	}
}
