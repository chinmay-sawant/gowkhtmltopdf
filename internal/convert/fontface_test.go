package convert

import (
	"bytes"
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
	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Objects: []settings.PdfObject{
			{Page: htmlPath, Load: settings.DefaultLoadPage()},
		},
		Output: filepath.Join(t.TempDir(), "out.pdf"),
	}
	cmd.Global.EnableLocalFileAccess = false
	cmd.Global.Allow = []string{pageDir}
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

func TestFontFaceWOFFSkipped(t *testing.T) {
	html := `<html><head><style>
@font-face { font-family: Custom; src: url(Custom.woff); }
body { font-family: Custom, sans-serif; }
</style></head><body><p>WOFF skip</p></body></html>`
	cmd, dir := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	if err := os.WriteFile(filepath.Join(dir, "Custom.woff"), []byte("not-a-real-woff"), 0o644); err != nil {
		t.Fatalf("write woff: %v", err)
	}

	var log bytes.Buffer
	if err := RunPDF(cmd, &log); err != nil {
		t.Fatalf("RunPDF: %v\nlog: %s", err, log.String())
	}
	warn := log.String()
	if !strings.Contains(warn, "TTF/OTF only") {
		t.Errorf("expected WOFF skip warning; log=%q", warn)
	}
	data, err := os.ReadFile(cmd.Output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if bytes.Contains(data, []byte("/BaseFont /Custom")) {
		t.Error("WOFF src must not register Custom")
	}
}

func TestFontFaceHTTPSSkipped(t *testing.T) {
	html := `<html><head><style>
@font-face { font-family: Custom; src: url(https://example.com/fonts/Custom.ttf); }
body { font-family: Custom, sans-serif; }
</style></head><body><p>HTTPS skip</p></body></html>`
	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))

	var log bytes.Buffer
	if err := RunPDF(cmd, &log); err != nil {
		t.Fatalf("RunPDF: %v\nlog: %s", err, log.String())
	}
	warn := log.String()
	if !strings.Contains(warn, "network src") {
		t.Errorf("expected network skip warning; log=%q", warn)
	}
	data, err := os.ReadFile(cmd.Output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if bytes.Contains(data, []byte("/BaseFont /Custom")) {
		t.Error("https @font-face src must not register Custom")
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
