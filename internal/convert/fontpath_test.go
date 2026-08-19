//nolint:testpackage,wsl,funlen,cyclop // font-path / @font-face integration proofs
package convert

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

func TestFontPathSelectedFamilyVisibleInPDF(t *testing.T) {
	t.Parallel()

	fontDir := t.TempDir()
	copyAssetTTF(t, fontDir, "LiberationSans-Regular.ttf", "Face.ttf")

	html := `<html><body style="font-family: 'Liberation Sans', sans-serif; font-size: 14pt;">
<p>FontPathVisible</p></body></html>`
	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.FontPaths = []string{fontDir}

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if !bytes.Contains(data, []byte("/FontFile2")) {
		t.Fatal("expected embedded font")
	}

	// Discovery prefers the name-table PostScript name (LiberationSans).
	hasBase := bytes.Contains(data, []byte("/BaseFont"))
	hasLib := bytes.Contains(data, []byte("LiberationSans")) || bytes.Contains(data, []byte("Liberation"))
	if !hasBase || !hasLib {
		t.Fatalf("expected Liberation family in PDF BaseFont; log=%q", log.String())
	}

	if !strings.Contains(log.String(), "font discovery:") {
		t.Fatalf("expected discovery diagnostics; log=%q", log.String())
	}
}

func TestFontPathGeorgiaSelectsGeorgia(t *testing.T) {
	t.Parallel()

	fontDir := t.TempDir()
	writePatchedSerifFamily(t, filepath.Join(fontDir, "Georgia.ttf"), "Georgia")

	html := `<html><body style="font-family: Georgia, serif; font-size: 14pt;">
<p>GeorgiaPath</p></body></html>`
	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.FontPaths = []string{fontDir}

	data := runPDF(t, cmd)
	// Patched face keeps LiberationSerif PostScript-ish bytes but family alias
	// for layout comes from the name table; BaseFont often strips spaces.
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("not a PDF")
	}

	reg := pdf.ScanFontDirs([]string{fontDir})
	if reg.Lookup([]string{"Georgia"}, 400, false) == nil {
		t.Fatal("Georgia must resolve from font-path directory")
	}
}

func TestFontPathGelasioDoesNotSatisfyGeorgia(t *testing.T) {
	t.Parallel()

	fontDir := t.TempDir()
	writePatchedSerifFamily(t, filepath.Join(fontDir, "Gelasio.ttf"), "Gelasio")

	html := `<html><body style="font-family: Georgia, serif; font-size: 14pt;">
<p>NotGelasio</p></body></html>`
	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	cmd.Global.FontPaths = []string{fontDir}

	data := runPDF(t, cmd)
	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("not a PDF")
	}

	reg := pdf.ScanFontDirs([]string{fontDir})
	if reg.Lookup([]string{"Georgia"}, 400, false) != nil {
		t.Fatal("Gelasio must not register as Georgia")
	}

	// Fallback still embeds a serif/sans Liberation face.
	if !bytes.Contains(data, []byte("/FontFile2")) {
		t.Fatal("expected fallback embed")
	}
}

func TestFontPathFileNotSilentEmptyDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	copyAssetTTF(t, dir, "LiberationSans-Regular.ttf", "Only.ttf")
	filePath := filepath.Join(dir, "Only.ttf")

	var log bytes.Buffer

	reg := pdf.RegistryFromPathsLog([]string{filePath}, false, &log)
	if reg == nil || reg.Lookup([]string{"Liberation Sans"}, 400, false) == nil {
		t.Fatalf("file font-path should load face; log=%q", log.String())
	}

	junk := filepath.Join(dir, "readme.md")
	if err := os.WriteFile(junk, []byte("nope"), 0o600); err != nil {
		t.Fatal(err)
	}

	log.Reset()
	reg = pdf.RegistryFromPathsLog([]string{junk}, false, &log)
	if reg.Lookup([]string{"Liberation Sans"}, 400, false) != nil {
		t.Fatal("non-font file must not load faces")
	}

	if !strings.Contains(log.String(), "expects a directory") && !strings.Contains(log.String(), "skip") {
		t.Fatalf("expected clear skip diagnostic; log=%q", log.String())
	}
}

func TestFontFaceStyleVariantsSelect(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	copyAssetTTF(t, dir, "LiberationSans-Regular.ttf", "R.ttf")
	copyAssetTTF(t, dir, "LiberationSans-Bold.ttf", "B.ttf")
	copyAssetTTF(t, dir, "LiberationSans-Italic.ttf", "I.ttf")
	copyAssetTTF(t, dir, "LiberationSans-BoldItalic.ttf", "BI.ttf")

	html := `<html><head><style>
@font-face { font-family: Custom; src: url(R.ttf); font-weight: 400; font-style: normal; }
@font-face { font-family: Custom; src: url(B.ttf); font-weight: 700; font-style: normal; }
@font-face { font-family: Custom; src: url(I.ttf); font-weight: 400; font-style: italic; }
@font-face { font-family: Custom; src: url(BI.ttf); font-weight: 700; font-style: italic; }
body { font-family: Custom, sans-serif; font-size: 12pt; }
.b { font-weight: 700; }
.i { font-style: italic; }
.bi { font-weight: 700; font-style: italic; }
</style></head><body>
<p>Regular</p>
<p class="b">Bold</p>
<p class="i">Italic</p>
<p class="bi">BoldItalic</p>
</body></html>`

	cmd, pageDir := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))
	for _, name := range []string{"R.ttf", "B.ttf", "I.ttf", "BI.ttf"} {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}

		if err := os.WriteFile(filepath.Join(pageDir, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)
	if !bytes.Contains(data, []byte("/BaseFont /Custom")) {
		t.Fatalf("expected Custom @font-face embed; log=%q", log.String())
	}

	reg := pdf.NewRegistry()
	faces := []struct {
		file   string
		weight int
		italic bool
	}{
		{"R.ttf", 400, false},
		{"B.ttf", 700, false},
		{"I.ttf", 400, true},
		{"BI.ttf", 700, true},
	}
	for _, face := range faces {
		raw, err := os.ReadFile(filepath.Join(pageDir, face.file))
		if err != nil {
			t.Fatal(err)
		}

		fnt, err := pdf.ParseTTF(raw)
		if err != nil {
			t.Fatal(err)
		}

		fnt.PostScriptName = "Custom"
		fnt.SetStyleOverride(face.weight, face.italic)
		reg.AddFamilyAlias("Custom", fnt)
	}

	for _, face := range faces {
		got := reg.Lookup([]string{"Custom"}, face.weight, face.italic)
		if got == nil || got.Bold() != (face.weight >= 700) || got.Italic() != face.italic {
			t.Fatalf("style lookup failed for weight=%d italic=%v", face.weight, face.italic)
		}
	}
}

func TestInvalidFontFaceFallsBack(t *testing.T) {
	t.Parallel()

	html := `<html><head><style>
@font-face { font-family: Broken; src: url(missing-face.ttf); }
body { font-family: Broken, sans-serif; font-size: 14pt; }
</style></head><body><p>FallbackOK</p></body></html>`

	cmd, _ := newCommand(t, html, filepath.Join(t.TempDir(), "out.pdf"))

	var log bytes.Buffer
	data := runPDFWithLog(t, cmd, &log)

	if !bytes.HasPrefix(data, []byte("%PDF-")) {
		t.Fatal("expected PDF despite invalid @font-face")
	}

	if bytes.Contains(data, []byte("/BaseFont /Broken")) {
		t.Fatal("broken face must not embed as Broken")
	}

	if !bytes.Contains(data, []byte("/FontFile2")) {
		t.Fatal("expected Liberation fallback embed")
	}

	if !strings.Contains(log.String(), "@font-face") {
		t.Fatalf("expected @font-face warning; log=%q", log.String())
	}
}

func TestCLIFontPathSettingsReachRegistry(t *testing.T) {
	t.Parallel()
	// Mirrors CLI --font-path assignment: settings land on PdfGlobal.FontPaths
	// and loadFontRegistry builds the same DiscoverFonts registry as Document.
	fontDir := t.TempDir()
	copyAssetTTF(t, fontDir, "LiberationMono-Regular.ttf", "Mono.ttf")

	global := settings.DefaultPdfGlobal()
	global.FontPaths = []string{fontDir}
	global.Load.EnableLocalFileAccess = true

	reg := loadFontRegistry(global, nil)
	if reg == nil || reg.Lookup([]string{"Liberation Mono"}, 400, false) == nil {
		t.Fatal("CLI FontPaths must reach registry")
	}
}

func copyAssetTTF(t *testing.T, dir, asset, name string) {
	t.Helper()

	src := filepath.Join("..", "pdf", "assets", asset)

	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read %s: %v", asset, err)
	}

	if err := os.WriteFile(filepath.Join(dir, name), data, 0o600); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func writePatchedSerifFamily(t *testing.T, dst, family string) {
	t.Helper()

	src := filepath.Join("..", "pdf", "assets", "LiberationSerif-Regular.ttf")

	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read serif: %v", err)
	}

	old := "Liberation Serif"
	if len(family) > len(old) {
		t.Fatalf("family too long: %q", family)
	}

	padded := family + strings.Repeat(" ", len(old)-len(family))
	out := append([]byte(nil), data...)
	out = bytes.ReplaceAll(out, []byte(old), []byte(padded))

	oldUTF := utf16BEString(old)
	newUTF := utf16BEString(padded)
	out = bytes.ReplaceAll(out, oldUTF, newUTF)

	if err := os.WriteFile(dst, out, 0o600); err != nil {
		t.Fatal(err)
	}
}

func utf16BEString(s string) []byte {
	out := make([]byte, 0, len(s)*2)
	for _, r := range s {
		out = append(out, byte(r>>8), byte(r))
	}

	return out
}
