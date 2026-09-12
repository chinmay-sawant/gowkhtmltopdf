//nolint:all // targeted unit tests for the font variant face-resolution consumer
package layout

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf/assets"
)

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

// TestResolveFontVariantsCapableFaceDoesNotFakeInstancing proves the
// capability probe sees a variable/palette-capable face and that the consumer
// still resolves to the default instance instead of pretending.
//
// Variable instancing is not reachable in this package version: go-text
// v0.3.4 exposes font.Face.SetVariations, but pdf.Font embeds default-instance
// glyf outlines, its advances come from the raw hmtx table, and nothing in
// internal/pdf applies variation coordinates while shaping or subsetting.
// Shaping with coordinates the emitted PDF does not embed would desync
// glyphs from advances, so the default instance is the deliberate outcome and
// the gap is reported instead.
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
		t.Fatalf("capable face lookup = %p, want default instance %p", got, capable)
	}
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
