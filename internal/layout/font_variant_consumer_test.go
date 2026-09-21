//nolint:all // targeted unit tests for the font variant face-resolution consumer
package layout

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf/assets"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// TestDejaVuSansFamilyResolvesFallback proves font-family:'DejaVu Sans'
// resolves the bundled fallback faces, both through the default registry
// (convert and imageout build it from settings) and through the FaceSet when
// no registry is supplied. Without this the language-override demo falls back
// to Liberation and the Serbian SRB locl substitution cannot show.
func TestDejaVuSansFamilyResolvesFallback(t *testing.T) {
	t.Parallel()

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatalf("LoadDefaultFaces: %v", err)
	}

	style := initialStyle()
	style.FontFamily = []string{"DejaVu Sans"}
	style.FontWeight = 700

	withRegistry := &engine{
		font:     faces.Regular,
		faces:    faces,
		registry: pdf.RegistryFromGlobal(settings.DefaultPdfGlobal()),
		scale:    1,
	}

	if got := withRegistry.lookupFaceFor(&style); got != faces.UnicodeFallbackBold {
		t.Fatalf("registry lookup = %p, want fallback bold %p", got, faces.UnicodeFallbackBold)
	}

	noRegistry := &engine{
		font:  faces.Regular,
		faces: faces,
		scale: 1,
	}

	if got := noRegistry.lookupFaceFor(&style); got != faces.UnicodeFallbackBold {
		t.Fatalf("faceset lookup = %p, want fallback bold %p", got, faces.UnicodeFallbackBold)
	}
}

// TestResolveFontVariantsStaticBundledFaces pins the spec-correct no-op path.
// Every bundled Liberation and DejaVu face is static: no fvar table and no
// COLR/CPAL tables. CSS makes font-optical-sizing, font-variation-settings,
// and font-palette no-ops on such faces, so face resolution must return the
// same face it would use without the declarations.
func TestResolveFontVariantsStaticBundledFaces(t *testing.T) {
	t.Parallel()

	faces, err := pdf.LoadDefaultFaces()
	if err != nil {
		t.Fatalf("LoadDefaultFaces: %v", err)
	}

	for _, testCase := range []struct {
		name string
		face *pdf.Font
	}{
		{"LiberationSans-Regular", faces.Regular},
		{"DejaVuSans-UnicodeFallback", faces.UnicodeFallback},
	} {
		if testCase.face.HasVariationAxes() || testCase.face.HasColorPalette() {
			t.Fatalf("%s: bundled face advertises fvar or COLR+CPAL, want a static face", testCase.name)
		}
	}

	eng := &engine{font: faces.Regular, faces: faces, scale: 1}

	style := initialStyle()
	style.FontVariationSettings = `"wght" 700`
	style.FontOpticalSizing = fontOpticalAuto
	style.FontPalette = fontPaletteDark

	if got := eng.lookupFaceFor(&style); got != faces.Regular {
		t.Fatalf("static face lookup = %p, want bundled regular %p", got, faces.Regular)
	}
}

// TestResolveFontVariantsCapableFaceDoesNotFakeInstancing proves a directory
// that only *looks* like fvar/COLR (renamed tags, no real payloads) still
// resolves to the default face. Real instancing requires a parseable fvar.
func TestResolveFontVariantsCapableFaceDoesNotFakeInstancing(t *testing.T) {
	t.Parallel()

	// Renaming table directory tags gives the probe real fvar, COLR, and
	// CPAL entries without changing any payload. The probe checks presence
	// only, and pdf.ParseTTF validates directory offsets and lengths, not the
	// renamed tables' contents.
	capable, err := pdf.ParseTTF(renameSFNTTables(
		assets.LiberationSansRegular(),
		map[string]string{"post": "fvar", "name": "COLR", "FFTM": "CPAL"},
	))
	if err != nil {
		t.Fatalf("ParseTTF patched face: %v", err)
	}

	if !capable.HasVariationAxes() || !capable.HasColorPalette() {
		t.Fatalf(
			"capability probe = axes %v palette %v, want both true",
			capable.HasVariationAxes(), capable.HasColorPalette(),
		)
	}

	registry := pdf.NewRegistry()
	registry.AddFamilyAlias("VarFace", capable)

	eng := &engine{font: capable, registry: registry, scale: 1}

	style := initialStyle()
	style.FontFamily = []string{"VarFace"}
	style.FontVariationSettings = `"wght" 700`
	style.FontOpticalSizing = fontOpticalNone
	style.FontPalette = fontPaletteDark

	if got := eng.lookupFaceFor(&style); got != capable {
		t.Fatalf("renamed-tag face lookup = %p, want default instance %p", got, capable)
	}
}

func loadGowkVar(t *testing.T) *pdf.Font {
	t.Helper()

	path := filepath.Join("..", "..", "testdata", "fonts", "implemented-audit", "GowkVar-VF.ttf")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read GowkVar-VF: %v", err)
	}

	face, err := pdf.ParseTTF(data)
	if err != nil {
		t.Fatalf("ParseTTF GowkVar: %v", err)
	}

	return face
}

func gowkVarEngine(t *testing.T) (*engine, *pdf.Font) {
	t.Helper()

	face := loadGowkVar(t)
	registry := pdf.NewRegistry()
	registry.AddFamilyAlias("Gowk Var", face)

	return &engine{font: face, registry: registry, scale: 1}, face
}

func TestVariationSettingsChangesAdvance(t *testing.T) {
	t.Parallel()

	eng, base := gowkVarEngine(t)
	defAdv := base.Advance('A')

	style := initialStyle()
	style.FontFamily = []string{"Gowk Var"}
	style.FontVariationSettings = `"wght" 900`
	style.FontOpticalSizing = fontOpticalNone

	got := eng.lookupFaceFor(&style)
	if got == nil || got == base {
		t.Fatal("wght 900 resolved to the default instance")
	}

	if adv := got.Advance('A'); adv <= defAdv {
		t.Fatalf("wght 900 advance = %g, want > default %g", adv, defAdv)
	}

	if outlineWidth(got.GlyphContours('A')) <= outlineWidth(base.GlyphContours('A')) {
		t.Fatal("wght 900 outline is not wider than the default instance")
	}
}

func TestOpticalSizingAutoSetsOpsz(t *testing.T) {
	t.Parallel()

	eng, base := gowkVarEngine(t)
	defAdv := base.Advance('A')

	style := initialStyle()
	style.FontFamily = []string{"Gowk Var"}
	style.FontOpticalSizing = fontOpticalAuto
	style.FontSize = 72
	style.FontVariationSettings = fontVariantNormal

	got := eng.lookupFaceFor(&style)
	if got == nil || got == base {
		t.Fatal("optical-sizing:auto at 72pt resolved to the default instance")
	}

	if adv := got.Advance('A'); adv <= defAdv {
		t.Fatalf("opsz 72 advance = %g, want > default %g", adv, defAdv)
	}
}

func TestOpticalSizingNoneKeepsDefaultOpsz(t *testing.T) {
	t.Parallel()

	eng, base := gowkVarEngine(t)

	style := initialStyle()
	style.FontFamily = []string{"Gowk Var"}
	style.FontOpticalSizing = fontOpticalNone
	style.FontSize = 72
	style.FontVariationSettings = fontVariantNormal

	if got := eng.lookupFaceFor(&style); got != base {
		t.Fatalf("optical-sizing:none at 72pt = %p, want default %p", got, base)
	}
}

func TestResolveFontVariantsInstancesAxes(t *testing.T) {
	t.Parallel()

	eng, base := gowkVarEngine(t)

	style := initialStyle()
	style.FontFamily = []string{"Gowk Var"}
	style.FontVariationSettings = `"wght" 700, "wdth" 200`
	style.FontOpticalSizing = fontOpticalNone

	got := resolveFontVariants(&style, base)
	if got == base {
		t.Fatal("resolveFontVariants kept the default instance")
	}

	if got.Advance('A') <= base.Advance('A') {
		t.Fatalf("instanced advance %g, want > default %g", got.Advance('A'), base.Advance('A'))
	}

	same := eng.lookupFaceFor(&style)
	if same.Advance('A') != got.Advance('A') {
		t.Fatalf("lookupFaceFor advance %g, want instanced %g", same.Advance('A'), got.Advance('A'))
	}
}

func outlineWidth(contours [][]pdf.GlyphPoint) float64 {
	if len(contours) == 0 || len(contours[0]) == 0 {
		return 0
	}

	minX, maxX := contours[0][0].X, contours[0][0].X
	for _, contour := range contours {
		for _, p := range contour {
			if p.X < minX {
				minX = p.X
			}

			if p.X > maxX {
				maxX = p.X
			}
		}
	}

	return maxX - minX
}

// renameSFNTTables rewrites matching table directory tags to the requested
// replacements. Offsets, lengths, and payloads are untouched.
func renameSFNTTables(ttf []byte, renames map[string]string) []byte {
	out := bytes.Clone(ttf)
	numTables := int(binary.BigEndian.Uint16(out[4:6]))

	for i := range numTables {
		record := out[12+16*i:]
		if to, ok := renames[string(record[0:4])]; ok {
			copy(record[0:4], to)
		}
	}

	return out
}
