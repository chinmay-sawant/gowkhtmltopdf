package convert

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/settings"
)

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
	if err := os.WriteFile(dst, data, 0o644); err != nil {
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
	cmd, dir := newCommand(t, fontFaceHTML("Custom.ttf"), filepath.Join(t.TempDir(), "out.pdf"))
	copyTestdataTTF(t, dir)

	var log bytes.Buffer
	if err := RunPDF(cmd, &log); err != nil {
		t.Fatalf("RunPDF: %v\nlog: %s", err, log.String())
	}

	data, err := os.ReadFile(cmd.Output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}

	if !bytes.Contains(data, []byte("/FontFile2")) {
		t.Error("expected embedded subset font (/FontFile2)")
	}
	// MergeFontFaces sets PostScriptName from font-family → /BaseFont /Custom
	if !bytes.Contains(data, []byte("/BaseFont /Custom")) {
		t.Errorf("expected /BaseFont /Custom from @font-face; log=%q", log.String())
	}
}

func TestFontFaceACLDeny(t *testing.T) {
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
	if err := os.WriteFile(htmlPath, []byte(fontFaceHTML("../fonts/Custom.ttf")), 0o644); err != nil {
		t.Fatalf("write html: %v", err)
	}

	obj := settings.DefaultPdfObject()
	obj.Page = htmlPath
	// ACL test: do not open local file access; only Allow pageDir for the HTML.
	cmd := &cli.Command{
		Global:  settings.DefaultPdfGlobal(),
		Objects: []settings.PdfObject{obj},
		Output:  filepath.Join(t.TempDir(), "out.pdf"),
	}
	cmd.Global.Load.EnableLocalFileAccess = false
	cmd.Global.Load.Allow = []string{pageDir}
	cmd.Global.Size = settings.Size{PageSize: cmd.Global.PageSize}

	var log bytes.Buffer
	if err := RunPDF(cmd, &log); err != nil {
		t.Fatalf("RunPDF: %v\nlog: %s", err, log.String())
	}

	data, err := os.ReadFile(cmd.Output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}

	warn := log.String()
	if !strings.Contains(warn, "@font-face") {
		t.Errorf("expected @font-face ACL warning; log=%q", warn)
	}
	// Face must not register under Custom when FetchSub is denied.
	if bytes.Contains(data, []byte("/BaseFont /Custom")) {
		t.Error("ACL deny must not embed /BaseFont /Custom")
	}

	if !bytes.Contains(data, []byte("/FontFile2")) {
		t.Error("expected Liberation fallback embed (/FontFile2)")
	}
}

func TestFontFaceWOFFEmbed(t *testing.T) {
	cmd, dir := newCommand(t, fontFaceHTML("Custom.woff"), filepath.Join(t.TempDir(), "out.pdf"))
	ttfPath := copyTestdataTTF(t, dir)

	ttf, err := os.ReadFile(ttfPath)
	if err != nil {
		t.Fatalf("read ttf: %v", err)
	}

	woff := encodeWOFF1Test(t, ttf)
	if err := os.WriteFile(filepath.Join(dir, "Custom.woff"), woff, 0o644); err != nil {
		t.Fatalf("write woff: %v", err)
	}

	var log bytes.Buffer
	if err := RunPDF(cmd, &log); err != nil {
		t.Fatalf("RunPDF: %v\nlog: %s", err, log.String())
	}

	data, err := os.ReadFile(cmd.Output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if !bytes.Contains(data, []byte("/BaseFont /Custom")) {
		t.Errorf("expected WOFF1 @font-face embed /BaseFont /Custom; log=%q", log.String())
	}
}

func TestFontFaceWOFF2Skipped(t *testing.T) {
	html := `<html><head><style>
@font-face { font-family: Custom; src: url(Custom.woff2); }
body { font-family: Custom, sans-serif; }
</style></head><body><p>WOFF2 skip</p></body></html>`

	cmd, dir := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	if err := os.WriteFile(filepath.Join(dir, "Custom.woff2"), []byte("wOF2not-real"), 0o644); err != nil {
		t.Fatalf("write woff2: %v", err)
	}

	var log bytes.Buffer
	if err := RunPDF(cmd, &log); err != nil {
		t.Fatalf("RunPDF: %v\nlog: %s", err, log.String())
	}

	warn := log.String()
	if !strings.Contains(warn, "WOFF2") {
		t.Errorf("expected WOFF2 skip warning; log=%q", warn)
	}

	data, err := os.ReadFile(cmd.Output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if bytes.Contains(data, []byte("/BaseFont /Custom")) {
		t.Error("WOFF2 src must not register Custom")
	}
}

func TestFontFaceBadWOFFSkipped(t *testing.T) {
	html := `<html><head><style>
@font-face { font-family: Custom; src: url(Custom.woff); }
body { font-family: Custom, sans-serif; }
</style></head><body><p>bad WOFF</p></body></html>`

	cmd, dir := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	if err := os.WriteFile(filepath.Join(dir, "Custom.woff"), []byte("not-a-real-woff"), 0o644); err != nil {
		t.Fatalf("write woff: %v", err)
	}

	var log bytes.Buffer
	if err := RunPDF(cmd, &log); err != nil {
		t.Fatalf("RunPDF: %v\nlog: %s", err, log.String())
	}

	warn := log.String()
	if !strings.Contains(warn, "@font-face") {
		t.Errorf("expected @font-face parse warning; log=%q", warn)
	}

	data, err := os.ReadFile(cmd.Output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if bytes.Contains(data, []byte("/BaseFont /Custom")) {
		t.Error("bad WOFF must not register Custom")
	}
}

func TestFontFaceHTTPSFetchAttempted(t *testing.T) {
	// Remote https @font-face is allowed through FetchSub; a dead host must
	// warn and not register the face (no silent skip-by-policy).
	html := `<html><head><style>
@font-face { font-family: Custom; src: url(https://127.0.0.1:1/fonts/Custom.ttf); }
body { font-family: Custom, sans-serif; }
</style></head><body><p>HTTPS fetch</p></body></html>`
	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))

	var log bytes.Buffer
	if err := RunPDF(cmd, &log); err != nil {
		t.Fatalf("RunPDF: %v\nlog: %s", err, log.String())
	}

	warn := log.String()
	if !strings.Contains(warn, "@font-face src") {
		t.Errorf("expected @font-face fetch warning; log=%q", warn)
	}

	if strings.Contains(warn, "network src") && strings.Contains(warn, "skipped") {
		t.Errorf("https fonts must no longer be policy-skipped; log=%q", warn)
	}

	data, err := os.ReadFile(cmd.Output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if bytes.Contains(data, []byte("/BaseFont /Custom")) {
		t.Error("failed https @font-face must not register Custom")
	}
}

func TestFontFaceDataSkipped(t *testing.T) {
	html := `<html><head><style>
@font-face { font-family: Custom; src: url(data:font/ttf;base64,AAAA); }
body { font-family: Custom, sans-serif; }
</style></head><body><p>data skip</p></body></html>`
	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))

	var log bytes.Buffer
	if err := RunPDF(cmd, &log); err != nil {
		t.Fatalf("RunPDF: %v\nlog: %s", err, log.String())
	}

	warn := log.String()
	if !strings.Contains(warn, "data:") {
		t.Errorf("expected data: skip warning; log=%q", warn)
	}

	data, err := os.ReadFile(cmd.Output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}

	if bytes.Contains(data, []byte("/BaseFont /Custom")) {
		t.Error("data: @font-face src must not register Custom")
	}
}

// encodeWOFF1Test builds a minimal WOFF1 from SFNT for @font-face fixtures.
func encodeWOFF1Test(t *testing.T, sfnt []byte) []byte {
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

	for i, tb := range tabs {
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
			comp = append([]byte(nil), raw...)
		}

		compressed[i] = comp
		origLens[i] = tb.length
		compLens[i] = uint32(len(comp))
	}

	header := make([]byte, woffHeaderSize)
	copy(header[0:4], []byte("wOFF"))
	binary.BigEndian.PutUint32(header[4:8], flavor)
	binary.BigEndian.PutUint16(header[12:14], uint16(numTables))
	binary.BigEndian.PutUint32(header[16:20], uint32(len(sfnt)))

	dir := make([]byte, numTables*woffEntrySize)
	payloadOff := uint32(woffHeaderSize + numTables*woffEntrySize)

	var body bytes.Buffer

	for i, tb := range tabs {
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
