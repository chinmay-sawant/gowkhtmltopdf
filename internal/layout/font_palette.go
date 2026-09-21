package layout

import "github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"

// fontPaletteFill is the lite font-palette paint consumer: when the face has
// COLR+CPAL and the CSS value is light, dark, or a palette index, the first
// CPAL color of that palette replaces CSS color. normal and unknown idents
// keep CSS color. Layered COLR glyphs are not painted.
func fontPaletteFill(sty *ResolvedStyle, face *pdf.Font) (float64, float64, float64, bool) {
	if sty == nil || face == nil {
		return 0, 0, 0, false
	}

	if sty.FontPalette == "" || sty.FontPalette == fontVariantNormal {
		return 0, 0, 0, false
	}

	return face.PaletteSolidFill(sty.FontPalette)
}
