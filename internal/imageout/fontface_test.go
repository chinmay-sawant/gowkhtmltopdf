package imageout

import (
	"bytes"
	"context"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gowkhtmltopdf/internal/cli"
	"gowkhtmltopdf/internal/convert"
	"gowkhtmltopdf/internal/html"
	"gowkhtmltopdf/internal/layout"
	"gowkhtmltopdf/internal/load"
	"gowkhtmltopdf/internal/pdf"
	"gowkhtmltopdf/internal/settings"
)

// copyTestdataTTF copies a known TTF into dir as Custom.ttf for @font-face fixtures.
func copyTestdataTTF(t *testing.T, dir string) string {
	t.Helper()
	src := filepath.Join("..", "pdf", "assets", "LiberationSans-Regular.ttf")
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

// TestFontFaceLocalUsesCustom proves ACL-allowed local @font-face registers
// Custom and layout attaches that face (same MergeFontFaces path as Run).
func TestFontFaceLocalUsesCustom(t *testing.T) {
	dir := t.TempDir()
	copyTestdataTTF(t, dir)
	htmlPath := filepath.Join(dir, "input.html")
	if err := os.WriteFile(htmlPath, []byte(fontFaceHTML("Custom.ttf")), 0o644); err != nil {
		t.Fatalf("write html: %v", err)
	}

	pngOut := filepath.Join(dir, "out.png")
	cmd := runCommand(t, "--width", "200", "--format", "png", htmlPath, pngOut)
	var log bytes.Buffer
	if err := Run(context.Background(), cmd, &log); err != nil {
		t.Fatalf("Run: %v\nlog: %s", err, log.String())
	}
	f, err := os.Open(pngOut)
	if err != nil {
		t.Fatalf("open png: %v", err)
	}
	defer f.Close()
	if _, err := png.Decode(f); err != nil {
		t.Fatalf("decode png: %v", err)
	}

	// Open-box: same merge + layout as Run must attach Custom (not Liberation fallback).
	loader := load.NewLoader(cmd.Image.Load)
	loader.Allow = cmd.Global.Allow
	loader.EnableLocalFileAccess = cmd.Global.EnableLocalFileAccess
	res, err := loader.Load(context.Background(), htmlPath, cmd.Objects[0].Load)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	root, err := html.Parse(string(res.Body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	sheets := collectSheets(context.Background(), loader, root, res.Base, cmd.Objects[0].Load, io.Discard)
	reg := convert.MergeFontFaces(context.Background(), loader, nil, sheets, res.Base, cmd.Objects[0].Load, 1, io.Discard)
	if reg == nil || reg.Lookup([]string{"Custom"}, 400, false) == nil {
		t.Fatal("expected Custom face in registry after MergeFontFaces")
	}
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
	sawCustom := false
	for i := range lay.Ops {
		op := &lay.Ops[i]
		if op.Kind != layout.OpText || op.Font == nil {
			continue
		}
		if op.Font.PostScriptName == "Custom" {
			sawCustom = true
			break
		}
	}
	if !sawCustom {
		t.Error("expected layout text ops to use @font-face Custom")
	}
}

// TestFontFaceACLDeny ensures a denied @font-face src falls back without panic.
func TestFontFaceACLDeny(t *testing.T) {
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

	pngOut := filepath.Join(t.TempDir(), "out.png")
	cmd := &cli.Command{
		Global: settings.DefaultPdfGlobal(),
		Image:  settings.DefaultImageGlobal(),
		Objects: []settings.PdfObject{
			{Page: htmlPath, Load: settings.DefaultLoadPage()},
		},
		Output: pngOut,
	}
	cmd.Global.EnableLocalFileAccess = false
	cmd.Global.Allow = []string{pageDir}
	cmd.Image.Width = 200
	cmd.Image.Format = "png"

	var log bytes.Buffer
	if err := Run(context.Background(), cmd, &log); err != nil {
		t.Fatalf("Run: %v\nlog: %s", err, log.String())
	}
	warn := log.String()
	if !strings.Contains(warn, "@font-face") {
		t.Errorf("expected @font-face ACL warning; log=%q", warn)
	}
	f, err := os.Open(pngOut)
	if err != nil {
		t.Fatalf("open png: %v", err)
	}
	defer f.Close()
	if _, err := png.Decode(f); err != nil {
		t.Fatalf("decode png: %v", err)
	}

	// Face must not register under Custom when FetchSub is denied.
	loader := load.NewLoader(cmd.Image.Load)
	loader.Allow = cmd.Global.Allow
	loader.EnableLocalFileAccess = cmd.Global.EnableLocalFileAccess
	res, err := loader.Load(context.Background(), htmlPath, cmd.Objects[0].Load)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	rootHTML, err := html.Parse(string(res.Body))
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	sheets := collectSheets(context.Background(), loader, rootHTML, res.Base, cmd.Objects[0].Load, io.Discard)
	var denyLog bytes.Buffer
	reg := convert.MergeFontFaces(context.Background(), loader, nil, sheets, res.Base, cmd.Objects[0].Load, 1, &denyLog)
	if reg != nil && reg.Lookup([]string{"Custom"}, 400, false) != nil {
		t.Error("ACL deny must not register Custom")
	}
}
