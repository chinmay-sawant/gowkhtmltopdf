package imageout

import (
	"bytes"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/cli"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/convert/prepare"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/load"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// copyTestdataTTF copies a known TTF into dir as Custom.ttf for @font-face fixtures.
func copyTestdataTTF(t *testing.T, dir string) {
	t.Helper()

	candidates := []string{
		filepath.Join("..", "pdf", "assets", "LiberationSans-Regular.ttf"),
		filepath.Join("..", "..", "testdata", "fonts", "NotoSansKR-HangulSubset.ttf"),
	}

	var data []byte

	var readErr error

	for _, src := range candidates {
		data, readErr = os.ReadFile(src)
		if readErr == nil {
			break
		}
	}

	if readErr != nil {
		t.Fatalf("read testdata ttf: %v", readErr)
	}

	dst := filepath.Join(dir, "Custom.ttf")
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		t.Fatalf("write Custom.ttf: %v", err)
	}
}

func fontFaceHTML(srcURL string) string {
	return `<html><head><style>
@font-face { font-family: Custom; src: url(` + srcURL + `); }
body { font-family: Custom, sans-serif; font-size: 14pt; }
</style></head><body><p>Hello CustomFace</p></body></html>`
}

// collectFontLayout runs the same load + sheet collection + font-face merge
// path as RunRequest, then lays out the document (same MergeFontFaces path as
// RunRequest).
func collectFontLayout(
	t *testing.T,
	cmd *cli.Command,
	htmlPath string,
	fontLog io.Writer,
) (*layout.Result, *pdf.Registry) {
	t.Helper()

	loader, err := load.NewLoaderWithError(imageLoadGlobal(cmd.Global, cmd.Image))
	if err != nil {
		t.Fatalf("new loader: %v", err)
	}

	res, err := loader.Load(t.Context(), htmlPath, cmd.Objects[0].Load)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	root, err := html.ParseDocument(res.Body)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	resources := prepare.NewResourceContext(loader, res.Base, cmd.Objects[0].Load)
	sheets := resources.CollectSheets(
		t.Context(),
		root,
		prepare.SheetOptions{
			ViewportW: 768, ViewportH: 576, MediaType: "screen",
		},
		io.Discard,
	)

	reg := resources.MergeFontFaces(t.Context(), nil, sheets, 1, fontLog)

	def, err := pdf.DefaultFont()
	if err != nil {
		t.Fatalf("default font: %v", err)
	}

	lay, err := layout.Layout(root, layout.Options{
		Width: 200 * 0.75, Height: 200 * 0.75,
		Font: def, Registry: reg, Sheets: sheets, Background: true,
	})
	if err != nil {
		t.Fatalf("layout: %v", err)
	}

	return lay, reg
}

// TestFontFaceLocalUsesCustom proves ACL-allowed local @font-face registers
// Custom and layout attaches that face (same MergeFontFaces path as
// RunRequest).
func TestFontFaceLocalUsesCustom(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	copyTestdataTTF(t, dir)

	htmlPath := filepath.Join(dir, "input.html")
	if err := os.WriteFile(htmlPath, []byte(fontFaceHTML("Custom.ttf")), 0o600); err != nil {
		t.Fatalf("write html: %v", err)
	}

	pngOut := filepath.Join(dir, "out.png")
	cmd := runCommand(t, "--width", "200", "--format", "png", "-o", pngOut, htmlPath)

	var log bytes.Buffer

	var out bytes.Buffer

	req := NewRequest(cmd.Global, cmd.Image, cmd.Objects, &out)
	if err := RunRequest(t.Context(), req, &log); err != nil {
		t.Fatalf("RunRequest: %v\nlog: %s", err, log.String())
	}

	if _, err := png.Decode(bytes.NewReader(out.Bytes())); err != nil {
		t.Fatalf("decode png: %v", err)
	}

	// Open-box: same merge + layout as RunRequest must attach Custom (not Liberation fallback).
	lay, reg := collectFontLayout(t, cmd, htmlPath, io.Discard)
	if reg == nil || reg.Lookup([]string{"Custom"}, 400, false) == nil {
		t.Fatal("expected Custom face in registry after MergeFontFaces")
	}

	assertCustomFontUsed(t, lay)
}

// assertCustomFontUsed fails unless a text op uses the Custom face.
func assertCustomFontUsed(t *testing.T, lay *layout.Result) {
	t.Helper()

	for i := range lay.Ops {
		op := &lay.Ops[i]
		if op.Kind == layout.OpText && op.Font != nil && op.Font.PostScriptName == "Custom" {
			return
		}
	}

	t.Error("expected layout text ops to use @font-face Custom")
}

// TestFontFaceACLDeny ensures a denied @font-face src falls back without panic.
func TestFontFaceACLDeny(t *testing.T) {
	t.Parallel()
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

	pngOut := filepath.Join(t.TempDir(), "out.png")
	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Image:  settings.DefaultImageGlobal(),
		Objects: []settings.PdfObject{
			{Page: htmlPath, Load: settings.DefaultLoadPage()},
		},
		Output: pngOut,
	}
	cmd.Global.Load.EnableLocalFileAccess = false
	cmd.Global.Load.Allow = []string{pageDir}
	cmd.Image.Width = 200
	cmd.Image.Format = "png"

	var log bytes.Buffer

	var out bytes.Buffer

	req := NewRequest(cmd.Global, cmd.Image, cmd.Objects, &out)
	if err := RunRequest(t.Context(), req, &log); err != nil {
		t.Fatalf("RunRequest: %v\nlog: %s", err, log.String())
	}

	warn := log.String()
	if !strings.Contains(warn, "@font-face") {
		t.Errorf("expected @font-face ACL warning; log=%q", warn)
	}

	if _, err := png.Decode(bytes.NewReader(out.Bytes())); err != nil {
		t.Fatalf("decode png: %v", err)
	}

	// Face must not register under Custom when FetchSub is denied.
	var denyLog bytes.Buffer

	_, reg := collectFontLayout(t, cmd, htmlPath, &denyLog)
	if reg != nil && reg.Lookup([]string{"Custom"}, 400, false) != nil {
		t.Error("ACL deny must not register Custom")
	}
}
