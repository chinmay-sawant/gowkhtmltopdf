package convert

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// hasBaseFont reports whether data references a /BaseFont named Custom, the
// font-family this file's fixtures register. The writer prefixes embedded fonts
// with a PDF subset tag (six uppercase letters plus "+"), so both
// "/BaseFont /Custom" and "/BaseFont /ABCDEF+Custom" match.
func hasBaseFont(data []byte) bool {
	re := regexp.MustCompile(`/BaseFont /(?:[A-Z]{6}\+)?Custom[^A-Za-z]`)

	return re.Match(data)
}

// copyTestdataTTF copies a known TTF into dir as Custom.ttf for @font-face fixtures.
// Prefer Liberation (full Latin cmap) so ASCII body text actually uses the face.
func copyTestdataTTF(t *testing.T, dir string) string {
	t.Helper()

	src := filepath.Join("..", "..", "internal", "pdf", "assets", "LiberationSans-Regular.ttf")

	data, err := os.ReadFile(src)
	if err != nil {
		src = filepath.Join("..", "..", "testdata", "fonts", "NotoSansKR-HangulSubset.ttf")

		data, err = os.ReadFile(src)
		if err != nil {
			t.Fatalf("read testdata ttf: %v", err)
		}
	}

	dst := filepath.Join(dir, "Custom.ttf")
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		t.Fatalf("write Custom.ttf: %v", err)
	}

	return dst
}

func fontFaceHTML(srcURL string) string {
	return `<html><head><style>
@font-face { font-family: Custom; src: url(` + srcURL + `); }
body { font-family: Custom, sans-serif; font-size: 14pt; }
</style></head><body><p>Hello CustomFace</p></body></html>`
}

func TestFontFaceLocalEmbed(t *testing.T) {
	t.Parallel()
	cmd, dir := newCommand(t, fontFaceHTML("Custom.ttf"), filepath.Join(t.TempDir(), "out.pdf"))
	copyTestdataTTF(t, dir)

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}

	if !bytes.Contains(data, []byte("/FontFile2")) {
		t.Error("expected embedded subset font (/FontFile2)")
	}
	// MergeFontFaces sets PostScriptName from font-family; the writer may add
	// a six-letter subset tag in front of it.
	if !hasBaseFont(data) {
		t.Errorf("expected /BaseFont /Custom from @font-face; log=%q", log.String())
	}
}

func TestFontFaceACLDeny(t *testing.T) {
	t.Parallel()
	// Primary page needs a readable path; deny the font by allowing only the
	// page directory (sibling fonts/ is outside --allow).
	root := t.TempDir()
	pageDir := filepath.Join(root, "page")
	fontDir := filepath.Join(root, "fonts")

	if err := os.MkdirAll(pageDir, 0o755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(fontDir, 0o755); err != nil {
		t.Fatal(err)
	}

	copyTestdataTTF(t, fontDir)

	htmlPath := filepath.Join(pageDir, "input.html")
	if err := os.WriteFile(htmlPath, []byte(fontFaceHTML("../fonts/Custom.ttf")), 0o600); err != nil {
		t.Fatalf("write html: %v", err)
	}

	obj := settings.DefaultPdfObject()
	obj.Page = htmlPath
	// ACL test: do not open local file access; only Allow pageDir for the HTML.
	global := settings.DefaultPdfGlobal()
	global.Load.EnableLocalFileAccess = false
	global.Load.Allow = []string{pageDir}
	cmd := NewPDFRequest(global, []settings.PdfObject{obj}, nil, nil)

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}

	warn := log.String()
	if !strings.Contains(warn, "@font-face") {
		t.Errorf("expected @font-face ACL warning; log=%q", warn)
	}
	// Face must not register under Custom when FetchSub is denied.
	if hasBaseFont(data) {
		t.Error("ACL deny must not embed /BaseFont /Custom")
	}

	if !bytes.Contains(data, []byte("/FontFile2")) {
		t.Error("expected Liberation fallback embed (/FontFile2)")
	}
}

func TestFontFaceWOFFEmbed(t *testing.T) {
	t.Parallel()
	cmd, dir := newCommand(t, fontFaceHTML("Custom.woff"), filepath.Join(t.TempDir(), "out.pdf"))
	ttfPath := copyTestdataTTF(t, dir)

	ttf, err := os.ReadFile(ttfPath)
	if err != nil {
		t.Fatalf("read ttf: %v", err)
	}

	woff := encodeWOFF1Test(t, ttf)
	if err := os.WriteFile(filepath.Join(dir, "Custom.woff"), woff, 0o600); err != nil {
		t.Fatalf("write woff: %v", err)
	}

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if !hasBaseFont(data) {
		t.Errorf("expected WOFF1 @font-face embed /BaseFont /Custom; log=%q", log.String())
	}
}

func TestFontFaceBadWOFF2Skipped(t *testing.T) {
	t.Parallel()

	html := `<html><head><style>
@font-face { font-family: Custom; src: url(Custom.woff2); }
body { font-family: Custom, sans-serif; }
</style></head><body><p>WOFF2 bad</p></body></html>`

	cmd, dir := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	if err := os.WriteFile(filepath.Join(dir, "Custom.woff2"), []byte("wOF2not-real"), 0o600); err != nil {
		t.Fatalf("write woff2: %v", err)
	}

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if hasBaseFont(data) {
		t.Error("undecodable WOFF2 src must not register Custom")
	}

	if !strings.Contains(log.String(), "woff2: invalid font data") {
		t.Errorf("expected a decode failure warning; log=%q", log.String())
	}
}

// TestFontFaceWOFF2Embed proves a real WOFF2 source registers in the full
// convert pipeline (decode + face lookup), not just in the prepare unit test.
func TestFontFaceWOFF2Embed(t *testing.T) {
	t.Parallel()

	cmd, dir := newCommand(t, fontFaceHTML("Custom.woff2"), filepath.Join(t.TempDir(), "out.pdf"))

	woff2Path := filepath.Join("..", "..", "testdata", "fonts", "woff2", "LiberationSans-Regular-latin.woff2")

	fixture, err := os.ReadFile(woff2Path)
	if err != nil {
		t.Fatalf("read woff2 fixture: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, "Custom.woff2"), fixture, 0o600); err != nil {
		t.Fatalf("write woff2: %v", err)
	}

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if !hasBaseFont(data) {
		t.Errorf("expected WOFF2 @font-face embed /BaseFont /Custom; log=%q", log.String())
	}
}

func TestFontFaceBadWOFFSkipped(t *testing.T) {
	t.Parallel()

	html := `<html><head><style>
@font-face { font-family: Custom; src: url(Custom.woff); }
body { font-family: Custom, sans-serif; }
</style></head><body><p>bad WOFF</p></body></html>`

	cmd, dir := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	if err := os.WriteFile(filepath.Join(dir, "Custom.woff"), []byte("not-a-real-woff"), 0o600); err != nil {
		t.Fatalf("write woff: %v", err)
	}

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if hasBaseFont(data) {
		t.Error("bad WOFF must not register Custom")
	}
}

func TestFontFaceHTTPSFetchAttempted(t *testing.T) {
	t.Parallel()
	// Remote https @font-face is allowed through FetchSub; a dead host must
	// warn and not register the face (no silent skip-by-policy).
	html := `<html><head><style>
@font-face { font-family: Custom; src: url(https://127.0.0.1:1/fonts/Custom.ttf); }
body { font-family: Custom, sans-serif; }
</style></head><body><p>HTTPS fetch</p></body></html>`
	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if hasBaseFont(data) {
		t.Error("failed https @font-face must not register Custom")
	}
}

func TestFontFaceUndecodableDataSkipped(t *testing.T) {
	t.Parallel()

	html := `<html><head><style>
@font-face { font-family: Custom; src: url(data:font/ttf;base64,AAAA); }
body { font-family: Custom, sans-serif; }
</style></head><body><p>data skip</p></body></html>`
	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if hasBaseFont(data) {
		t.Error("undecodable data: @font-face src must not register Custom")
	}

	if !strings.Contains(log.String(), "src skipped (data:font/ttf;base64,") {
		t.Errorf("expected a data: skip warning naming the media type; log=%q", log.String())
	}
}

// TestFontFaceMissingWOFF2WithTTFFallback proves a missing WOFF2 source falls
// through to a later TTF source in the same src (fetch failure, then parse the
// next candidate). Format policy no longer skips .woff2 outright.
func TestFontFaceMissingWOFF2WithTTFFallback(t *testing.T) {
	t.Parallel()

	srcs := map[string]string{
		"plain": `url(Custom.woff2) format("woff2"), url(Custom.ttf)`,
		"query": `url(Custom.woff2?v=9) format("woff2"), url(Custom.ttf)`,
	}

	for name, src := range srcs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			html := `<html><head><style>
@font-face { font-family: Custom; src: ` + src + `; }
body { font-family: Custom, sans-serif; font-size: 14pt; }
</style></head><body><p>woff2 fallback</p></body></html>`
			cmd, dir := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
			// No Custom.woff2 on disk: the fetch fails and the later TTF must
			// still register.
			copyTestdataTTF(t, dir)

			var log bytes.Buffer
			data := runPDFWithLog(t, cmd, &log)

			if !hasBaseFont(data) {
				t.Errorf("TTF fallback after a missing WOFF2 did not register; log=%q", log.String())
			}

			if !strings.Contains(log.String(), "Custom.woff2") {
				t.Errorf("expected a WOFF2 failure warning naming the source; log=%q", log.String())
			}
		})
	}
}

func TestFontFaceDataURIRegistersSupportedPayload(t *testing.T) {
	t.Parallel()

	ttfPath := copyTestdataTTF(t, t.TempDir())

	ttf, err := os.ReadFile(ttfPath)
	if err != nil {
		t.Fatalf("read ttf: %v", err)
	}

	tests := []struct {
		name string
		uri  string
	}{
		{"woff1", "data:font/woff;base64," + base64.StdEncoding.EncodeToString(encodeWOFF1Test(t, ttf))},
		{"ttf", "data:font/ttf;base64," + base64.StdEncoding.EncodeToString(ttf)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			cmd, _ := newCommand(t, fontFaceHTML(test.uri), filepath.Join(t.TempDir(), "out.pdf"))

			var log bytes.Buffer
			data := runPDFWithLog(t, cmd, &log)

			if !hasBaseFont(data) {
				t.Errorf("data: %s payload did not register /BaseFont /Custom; log=%q", test.name, log.String())
			}
		})
	}
}

// TestFontFaceDataURILogHygiene pins the log contract for data: font srcs: no
// printf format leftovers and no line carrying the base64 payload (the old
// skip warning formatted a 42KB data URI into the next warning line).
func TestFontFaceDataURILogHygiene(t *testing.T) {
	t.Parallel()

	payload := bytes.Repeat([]byte{0xAB}, 32*1024)
	encoded := base64.StdEncoding.EncodeToString(payload)
	uri := "data:font/woff2;base64," + encoded

	cmd, _ := newCommand(t, fontFaceHTML(uri), filepath.Join(t.TempDir(), "out.pdf"))

	var log bytes.Buffer

	runPDFWithLog(t, cmd, &log)

	output := log.String()
	if strings.Contains(output, "%!(EXTRA") {
		t.Errorf("format/arg mismatch leaked into the log; log=%q", output)
	}

	if strings.Contains(output, encoded[:64]) {
		t.Error("base64 payload leaked into the log")
	}

	for _, line := range strings.Split(output, "\n") {
		if len(line) > 1024 {
			t.Errorf("log line is %d bytes, want <= 1024: %.80q...", len(line), line)
		}
	}

	if !strings.Contains(output, "src skipped (data:font/woff2;base64,") {
		t.Errorf("expected a data: skip warning naming scheme and media type; log=%q", output)
	}
}

// encodeWOFF1Test builds a minimal WOFF1 from SFNT for @font-face fixtures.
func encodeWOFF1Test(t *testing.T, sfnt []byte) []byte { //nolint:funlen // WOFF1 container has many fixed header fields
	t.Helper()

	if len(sfnt) < 12 {
		t.Fatal("sfnt too short")
	}

	const (
		woffHeaderSize = 44
		woffEntrySize  = 20
	)

	flavor := binary.BigEndian.Uint32(sfnt[0:4])
	numTables := int(binary.BigEndian.Uint16(sfnt[4:6]))

	type tab struct {
		tag            [4]byte
		offset, length uint32
		checksum       uint32
	}

	tabs := make([]tab, numTables)

	for i := range numTables {
		rec := sfnt[12+16*i:]
		copy(tabs[i].tag[:], rec[0:4])
		tabs[i].checksum = binary.BigEndian.Uint32(rec[4:8])
		tabs[i].offset = binary.BigEndian.Uint32(rec[8:12])
		tabs[i].length = binary.BigEndian.Uint32(rec[12:16])
	}

	compressed := make([][]byte, numTables)
	origLens := make([]uint32, numTables)
	compLens := make([]uint32, numTables)

	for idx, table := range tabs {
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
			comp = append([]byte(nil), raw...)
		}

		compressed[idx] = comp
		origLens[idx] = table.length
		compLens[idx] = uint32(len(comp)) //nolint:gosec // bounded test fixture size
	}

	header := make([]byte, woffHeaderSize)
	copy(header[0:4], []byte("wOFF"))
	binary.BigEndian.PutUint32(header[4:8], flavor)
	binary.BigEndian.PutUint16(header[12:14], uint16(numTables)) //nolint:gosec // real font table counts
	binary.BigEndian.PutUint32(header[16:20], uint32(len(sfnt))) //nolint:gosec // bounded test fixture size

	dir := make([]byte, numTables*woffEntrySize)
	payloadOff := uint32(woffHeaderSize + numTables*woffEntrySize) //nolint:gosec // bounded test fixture size

	var body bytes.Buffer

	for table, tbl := range tabs {
		for payloadOff%4 != 0 {
			body.WriteByte(0)

			payloadOff++
		}

		rec := dir[table*woffEntrySize : (table+1)*woffEntrySize]
		copy(rec[0:4], tbl.tag[:])
		binary.BigEndian.PutUint32(rec[4:8], payloadOff)
		binary.BigEndian.PutUint32(rec[8:12], compLens[table])
		binary.BigEndian.PutUint32(rec[12:16], origLens[table])
		binary.BigEndian.PutUint32(rec[16:20], tbl.checksum)
		body.Write(compressed[table])
		payloadOff += compLens[table]
	}

	out := make([]byte, 0, len(header)+len(dir)+body.Len())
	out = append(out, header...)
	out = append(out, dir...)
	out = append(out, body.Bytes()...)
	binary.BigEndian.PutUint32(out[8:12], uint32(len(out))) //nolint:gosec // bounded test fixture size

	return out
}
