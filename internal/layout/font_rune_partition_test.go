package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf/assets"
)

// TestRuneFaceHonorsUnicodeRangePartition pins the learncpp-1 shape at the
// layout layer: one CSS family is split into unicode-range partitions, the
// latin-ext face registers first, and the latin face is the family primary.
// The latin face also maps U+0100, so glyph coverage alone cannot tell the
// two apart: the @font-face unicode-range descriptor must decide, and the
// rune painted with the latin-ext face even though the primary face covers
// the codepoint.
//
// The second half pins that the descriptor lookup does not swallow glyph
// fallback: a rune that no partition declares still resolves through the
// existing family/default scan to a face that maps it.
func TestRuneFaceHonorsUnicodeRangePartition(t *testing.T) { //nolint:cyclop // partition + fallback face gates
	t.Parallel()

	latin, err := pdf.ParseTTF(assets.LiberationSansRegular())
	if err != nil {
		t.Fatalf("ParseTTF latin face: %v", err)
	}

	ext, err := pdf.ParseTTF(assets.UnicodeFallbackRegular())
	if err != nil {
		t.Fatalf("ParseTTF latin-ext face: %v", err)
	}

	// Both faces map U+0100, so the descriptor must decide.
	if latin.GlyphID(0x0100) == 0 || ext.GlyphID(0x0100) == 0 {
		t.Fatalf("fixture faces must both map U+0100: latin=%v ext=%v",
			latin.GlyphID(0x0100) != 0, ext.GlyphID(0x0100) != 0)
	}

	reg := pdf.NewRegistry()
	// learncpp registration order: latin-ext first, latin second.
	reg.AddFamilyAliasSpec("PartFace", ext, pdf.FaceSpec{
		Ranges: []pdf.UnicodeRange{{Lo: 0x0100, Hi: 0x02AF}},
	})
	reg.AddFamilyAliasSpec("PartFace", latin, pdf.FaceSpec{
		Ranges: []pdf.UnicodeRange{{Lo: 0x0000, Hi: 0x00FF}},
	})

	if got := reg.Lookup([]string{"PartFace"}, 400, false); got != latin {
		t.Fatalf("family primary = %v, want the latin face", got)
	}

	root := mustParse(t, `<html><body><p>AB Ā★</p></body></html>`)
	cssSheet := sheet(t, `body { margin: 0; font-family: PartFace; font-size: 20px }`)

	res, err := Layout(root, Options{
		Width: testViewport, Height: 800,
		Sheets: []*css.Stylesheet{cssSheet}, Registry: reg,
	})
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	fonts := paintedFontsByRune(res)

	if fonts['A'] != latin {
		t.Fatalf("U+0041 painted with %v, want the declared latin face", faceName(fonts['A']))
	}

	if fonts[0x0100] != ext {
		t.Fatalf("U+0100 painted with %v, want the declared latin-ext face", faceName(fonts[0x0100]))
	}

	if got := fonts[0x2605]; got == nil || got.GlyphID(0x2605) == 0 {
		t.Fatalf("U+2605 (declared by no partition) painted with %v, want a fallback face that maps it",
			faceName(got))
	}
}

// paintedFontsByRune maps every painted rune to the face of the text run that
// carries it.
func paintedFontsByRune(res *Result) map[rune]*pdf.Font {
	out := map[rune]*pdf.Font{}

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || paintOp.Font == nil {
			continue
		}

		for _, r := range paintOp.Text {
			out[r] = paintOp.Font
		}
	}

	return out
}

func faceName(f *pdf.Font) string {
	if f == nil {
		return "<nil>"
	}

	return f.PostScriptName
}
