package prepare

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"io"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/andybalholm/brotli"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// testResources builds the same resource seam Document uses, with a real
// loader so data: URLs decode through the load policy.
func testResources(t *testing.T) ResourceContext {
	t.Helper()

	loader, err := load.NewLoaderWithError(settings.LoadGlobal{})
	if err != nil {
		t.Fatalf("new loader: %v", err)
	}

	return NewResourceContext(loader, "https://example.test/root.html", settings.DefaultLoadPage())
}

func readFontAsset(t *testing.T) []byte {
	t.Helper()

	data, err := os.ReadFile(filepath.Join("..", "..", "pdf", "assets", "LiberationSans-Regular.ttf"))
	if err != nil {
		t.Fatalf("read font asset: %v", err)
	}

	return data
}

func dataURL(mediaType string, body []byte) string {
	return "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(body)
}

// TestMergeFontFacesDataURLTTF proves a base64 data: TTF registers through
// the whole merge path.
func TestMergeFontFacesDataURLTTF(t *testing.T) {
	t.Parallel()

	uri := dataURL("font/ttf", readFontAsset(t))
	sheet := &css.Stylesheet{ // font-face-only sheet
		FontFaces: []css.FontFace{{Family: "InlineFace", Src: "url(" + uri + ")"}},
	}

	registry := testResources(t).MergeFontFaces(t.Context(), pdf.NewRegistry(), []*css.Stylesheet{sheet}, 1, io.Discard)
	if font := registry.Lookup([]string{"InlineFace"}, 400, false); font == nil {
		t.Fatal("data: TTF font-face was not registered")
	}
}

// TestMergeFontFacesDataURLWOFF2 proves a null-transform WOFF2 data: URL is
// Brotli-decoded and registered.
func TestMergeFontFacesDataURLWOFF2(t *testing.T) {
	t.Parallel()

	woff2 := buildWOFF2(t, readFontAsset(t), false)
	uri := dataURL("font/woff2", woff2)
	sheet := &css.Stylesheet{ // font-face-only sheet
		FontFaces: []css.FontFace{{Family: "InlineWoff2", Src: "url(" + uri + ")"}},
	}

	registry := testResources(t).MergeFontFaces(t.Context(), pdf.NewRegistry(), []*css.Stylesheet{sheet}, 1, io.Discard)
	if font := registry.Lookup([]string{"InlineWoff2"}, 400, false); font == nil {
		t.Fatal("data: WOFF2 font-face was not registered")
	}
}

// TestFetchFontFaceMalformedPayloads covers the reject path for bad base64,
// non-font bytes, and a truncated WOFF2 header.
func TestFetchFontFaceMalformedPayloads(t *testing.T) {
	t.Parallel()

	truncated := append([]byte("wOF2"), make([]byte, 40)...)
	cases := map[string]string{
		"bad base64":      "data:font/ttf;base64,!!!!",
		"garbage bytes":   dataURL("font/ttf", []byte("not a font at all")),
		"truncated woff2": dataURL("font/woff2", truncated),
	}

	for name, uri := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if _, ok := fetchFontFace(t.Context(), testResources(t).Bound(), uri, 1, io.Discard); ok {
				t.Fatalf("%s: fetchFontFace accepted a malformed payload", name)
			}
		})
	}
}

// TestFetchFontFaceTransformedWOFF2Rejected pins the documented limit: a
// transformed glyf table is rejected, not silently mis-reconstructed.
func TestFetchFontFaceTransformedWOFF2Rejected(t *testing.T) {
	t.Parallel()

	woff2 := buildWOFF2(t, readFontAsset(t), true)
	uri := dataURL("font/woff2", woff2)

	if _, ok := fetchFontFace(t.Context(), testResources(t).Bound(), uri, 1, io.Discard); ok {
		t.Fatal("transformed glyf WOFF2 must be rejected")
	}
}

// knownTagIndex covers a few WOFF2 known-tag indices so the builder exercises
// both directory encodings. The remaining tables use the escape tag.
var knownTagIndex = map[string]byte{ //nolint:gochecknoglobals // test vocabulary
	"head": 1, "hhea": 2, "hmtx": 3, "glyf": 10, "loca": 11,
}

// buildWOFF2 wraps an SFNT in a WOFF2 container. transformGlyf flips glyf to
// transform version 0 (with a placeholder transformLength) and loca to version
// 0 with transformLength 0 to exercise the rejection path; otherwise glyf and
// loca use the null transform version 3, which this slice can reconstruct.
func buildWOFF2(t *testing.T, sfnt []byte, transformGlyf bool) []byte {
	t.Helper()

	numTables := binary.BigEndian.Uint16(sfnt[4:6])
	flavor := binary.BigEndian.Uint32(sfnt[0:4])

	var raw bytes.Buffer

	var dir bytes.Buffer

	for i := range int(numTables) {
		rec := sfnt[12+16*i:]
		tag := string(rec[0:4])
		off := int(binary.BigEndian.Uint32(rec[8:12]))
		length := binary.BigEndian.Uint32(rec[12:16])

		raw.Write(sfnt[off : off+int(length)])

		version, transformLen, transformed := woff2TransformVersion(tag, length, transformGlyf)

		if idx, known := knownTagIndex[tag]; known {
			dir.WriteByte(idx | version<<6)
		} else {
			dir.WriteByte(0x3f | version<<6)
			dir.WriteString(tag)
		}

		dir.Write(uIntBase128(length))

		if transformed {
			dir.Write(uIntBase128(transformLen))
		}
	}

	var compressed bytes.Buffer

	writer := brotli.NewWriterLevel(&compressed, brotli.BestSpeed)
	if _, err := writer.Write(raw.Bytes()); err != nil {
		t.Fatalf("brotli write: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("brotli close: %v", err)
	}

	dirLen := dir.Len()
	compressedLen := compressed.Len()
	header := woff2Header(t, flavor, numTables, len(sfnt), dirLen, compressedLen)

	out := make([]byte, 0, len(header)+dirLen+compressedLen)
	out = append(out, header...)
	out = append(out, dir.Bytes()...)
	out = append(out, compressed.Bytes()...)

	return out
}

// woff2TransformVersion returns the transform version byte, transformLength,
// and transformed flag for one table tag. glyf and loca under transformGlyf
// take version 0 so the decoder exercises its rejection path; otherwise they
// take the null transform version 3.
func woff2TransformVersion(tag string, length uint32, transformGlyf bool) (byte, uint32, bool) {
	switch {
	case tag == "glyf" && transformGlyf:
		return 0, length, true
	case tag == "loca" && transformGlyf:
		return 0, 0, true
	case tag == "glyf", tag == "loca":
		return 3, 0, false
	}

	return 0, 0, false
}

// woff2Header builds the 48-byte WOFF2 header. Every size field is bounds
// checked against its encoded width before the conversion, so an oversized
// value fails the test instead of silently truncating the field.
func woff2Header(t *testing.T, flavor uint32, numTables uint16, sfntLen, dirLen, compressedLen int) []byte {
	t.Helper()

	total := 48 + dirLen + compressedLen
	if total < 0 || total > math.MaxUint32 {
		t.Fatalf("woff2 total size out of range: %d", total)
	}

	if sfntLen < 0 || sfntLen > math.MaxUint32 {
		t.Fatalf("sfnt size out of range: %d", sfntLen)
	}

	if compressedLen < 0 || compressedLen > math.MaxUint32 {
		t.Fatalf("compressed size out of range: %d", compressedLen)
	}

	header := make([]byte, 48)
	copy(header[0:4], "wOF2")
	binary.BigEndian.PutUint32(header[4:8], flavor)
	binary.BigEndian.PutUint32(header[8:12], uint32(total)) //nolint:gosec // bounded by the math.MaxUint32 check above
	binary.BigEndian.PutUint16(header[12:14], numTables)
	binary.BigEndian.PutUint32(header[16:20], uint32(sfntLen)) //nolint:gosec // bounded by the math.MaxUint32 check above
	binary.BigEndian.PutUint32(header[20:24], uint32(compressedLen))
	binary.BigEndian.PutUint16(header[24:26], 1)

	return header
}

// uIntBase128 encodes a WOFF2 variable-length unsigned integer.
func uIntBase128(value uint32) []byte {
	var buf [5]byte

	pos := len(buf) - 1
	buf[pos] = byte(value & 0x7f)

	for value >>= 7; value > 0; value >>= 7 {
		pos--
		buf[pos] = byte(value&0x7f) | 0x80
	}

	return buf[pos:]
}
