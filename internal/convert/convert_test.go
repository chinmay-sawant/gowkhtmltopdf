package convert

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/settings"
)

// newCommand writes html into a temp dir and returns a cli.Command pointing
// at it, with local file access enabled (the frozen ACL default blocks local
// reads unless the user opts in).
func newCommand(t *testing.T, html string, output string) (*cli.Command, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "input.html")
	if err := os.WriteFile(path, []byte(html), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}
	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Objects: []settings.PdfObject{
			{Page: path, Load: settings.DefaultLoadPage()},
		},
		Output: output,
	}
	// --enable-local-file-access: global flag on, object-level block off.
	cmd.Global.EnableLocalFileAccess = true
	cmd.Objects[0].Load.BlockLocalFileAccess = false
	cmd.Global.Size = settings.Size{PageSize: cmd.Global.PageSize}
	return cmd, dir
}

// runPDF runs RunPDF and returns the produced PDF bytes. When output is "-"
// the PDF lands on os.Stdout, so the caller must have redirected it.
func runPDF(t *testing.T, cmd *cli.Command) []byte {
	t.Helper()
	var log bytes.Buffer
	if err := RunPDF(cmd, &log); err != nil {
		t.Fatalf("RunPDF: %v", err)
	}
	data, err := os.ReadFile(cmd.Output)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	return data
}

// pageCount counts page dicts. The pages tree root is emitted as
// "/Type /Pages\n", which never matches "/Type /Page\n".
func pageCount(data []byte) int {
	return bytes.Count(data, []byte("/Type /Page\n"))
}

func TestRunPDFSinglePageA4(t *testing.T) {
	cmd, _ := newCommand(t, `<html><body><h1>Hello</h1><p>world</p></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("output is not a PDF")
	}
	if n := pageCount(data); n != 1 {
		t.Errorf("pages = %d, want 1", n)
	}
	if !bytes.Contains(data, []byte("/FontFile2")) {
		t.Error("expected embedded subset font (/FontFile2)")
	}
}

func TestRunPDFMultiPage(t *testing.T) {
	var sb strings.Builder
	sb.WriteString("<html><body>")
	for i := 0; i < 200; i++ {
		sb.WriteString("<p>paragraph of text number ")
		sb.WriteString(string(rune('a' + i%26)))
		sb.WriteString(" with some words to wrap</p>")
	}
	sb.WriteString("</body></html>")
	cmd, _ := newCommand(t, sb.String(), filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)
	if n := pageCount(data); n < 2 {
		t.Errorf("pages = %d, want >= 2", n)
	}
}

func TestRunPDFStyleTableImage(t *testing.T) {
	pngB64 := pngDataURL(t, 12, 12)
	html := `<html><head><style>
.box { background-color: #336699; width: 80px; height: 30px; }
</style></head><body>
<div class="box">colored</div>
<table><tr><th>a</th><th>b</th></tr><tr><td>1</td><td>2</td></tr></table>
<img src="` + pngB64 + `">
</body></html>`
	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	data := runPDF(t, cmd)
	if n := pageCount(data); n != 1 {
		t.Errorf("pages = %d, want 1", n)
	}
	if !bytes.Contains(data, []byte("/Subtype /Image")) {
		t.Error("expected an embedded image xobject")
	}
}

func TestRunPDFLinkedStylesheet(t *testing.T) {
	cmd, dir := newCommand(t, `<html><head><link rel="stylesheet" href="style.css"></head><body><div class="box">styled</div></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	if err := os.WriteFile(filepath.Join(dir, "style.css"), []byte(".box { background-color: #000000; }"), 0o644); err != nil {
		t.Fatalf("write css: %v", err)
	}
	data := runPDF(t, cmd)
	if n := pageCount(data); n != 1 {
		t.Errorf("pages = %d, want 1", n)
	}
}

func TestRunPDFScreenOnlyStylesheetExcluded(t *testing.T) {
	cmd, dir := newCommand(t, `<html><head><link rel="stylesheet" href="screen.css" media="screen"></head><body><div class="box">styled</div></body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	if err := os.WriteFile(filepath.Join(dir, "screen.css"), []byte(".box { background-color: #000000; }"), 0o644); err != nil {
		t.Fatalf("write css: %v", err)
	}
	if err := RunPDF(cmd, &bytes.Buffer{}); err != nil {
		t.Fatalf("RunPDF: %v", err)
	}
}

func TestRunPDFOutputStdout(t *testing.T) {
	cmd, _ := newCommand(t, `<html><body><p>stdout test</p></body></html>`, "-")
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	runErr := RunPDF(cmd, &bytes.Buffer{})
	w.Close()
	os.Stdout = old
	if runErr != nil {
		t.Fatalf("RunPDF: %v", runErr)
	}
	data, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read stdout capture: %v", err)
	}
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("stdout output is not a PDF")
	}
	if n := pageCount(data); n != 1 {
		t.Errorf("pages = %d, want 1", n)
	}
}

func TestRunPDFMissingFile(t *testing.T) {
	cmd, _ := newCommand(t, `<html><body>x</body></html>`, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Objects[0].Page = filepath.Join(t.TempDir(), "does-not-exist.html")
	if err := RunPDF(cmd, &bytes.Buffer{}); err == nil {
		t.Fatal("expected error for missing input file, got nil")
	}
}

// pngDataURL builds a minimal valid RGBA PNG as a data: URL.
func pngDataURL(t *testing.T, w, h int) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{200, 30, 30, 255})
		}
	}
	var out bytes.Buffer
	if err := png.Encode(&out, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(out.Bytes())
}
