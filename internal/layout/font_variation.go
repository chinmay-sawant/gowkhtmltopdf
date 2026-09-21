package layout

import (
	"math"
	"strconv"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

const fontVariationInitialCapacity = 4

// fontVariantCapability records the tables the CSS font variation family
// needs from a resolved face.
type fontVariantCapability struct {
	variationAxes bool // fvar present: variable font
	colorPalette  bool // COLR and CPAL present: color-palette font
}

// resolveFontVariants is the face-resolution consumer for font-optical-sizing,
// font-variation-settings, and font-palette. Static faces (bundled Liberation
// and DejaVu) have no fvar and no COLR/CPAL, so CSS makes all three no-ops.
//
// When the face has fvar, wght/opsz/wdth from font-variation-settings and
// font-optical-sizing:auto (opsz = used font-size in pt) are instanced into
// glyf outlines and hmtx advances. font-palette stays a documented gap:
// there is still no COLR/CPAL paint path.
func resolveFontVariants(sty *ResolvedStyle, face *pdf.Font) *pdf.Font {
	if sty == nil || face == nil {
		return face
	}

	wantsAxes := sty.FontVariationSettings != fontVariantNormal || sty.FontOpticalSizing == fontOpticalAuto
	wantsPalette := sty.FontPalette != fontVariantNormal

	if !wantsAxes && !wantsPalette {
		return face
	}

	capability := faceFontVariantCapability(face)
	if wantsAxes && capability.variationAxes {
		if vars := variationSettingsFor(sty, face); len(vars) > 0 {
			face = face.Instance(vars)
		}
	}

	return face
}

// faceFontVariantCapability probes the resolved face for variation and palette
// tables. Both are false for the static bundled faces.
func faceFontVariantCapability(face *pdf.Font) fontVariantCapability {
	if face == nil {
		return fontVariantCapability{} //nolint:exhaustruct // zero value means neither capability
	}

	return fontVariantCapability{
		variationAxes: face.HasVariationAxes(),
		colorPalette:  face.HasColorPalette(),
	}
}

// variationSettingsFor builds the design-space coordinates to instance.
// font-optical-sizing:auto supplies opsz from used font-size (pt) unless
// font-variation-settings already set that axis. Variation-settings values
// win, matching CSS Fonts 4 low-level override order.
func variationSettingsFor(sty *ResolvedStyle, face *pdf.Font) []pdf.Variation {
	if sty == nil || face == nil {
		return nil
	}

	vars := make([]pdf.Variation, 0, fontVariationInitialCapacity)
	seen := map[string]int{}

	if sty.FontOpticalSizing == fontOpticalAuto {
		if axis, ok := findVariationAxis(face, "opsz"); ok {
			vars = append(vars, pdf.Variation{Tag: axis.Tag, Value: float32(sty.FontSize)})
			seen[axis.Tag] = 0
		}
	}

	if sty.FontVariationSettings != fontVariantNormal {
		for _, part := range splitFontVariationList(sty.FontVariationSettings) {
			tag, number, ok := parseFontVariationPair(part)
			if !ok {
				continue
			}

			value, err := strconv.ParseFloat(number, 32)
			if err != nil {
				continue
			}

			item := pdf.Variation{Tag: tag, Value: float32(value)}
			if idx, exists := seen[tag]; exists {
				vars[idx] = item

				continue
			}

			seen[tag] = len(vars)
			vars = append(vars, item)
		}
	}

	return vars
}

func findVariationAxis(face *pdf.Font, tag string) (pdf.VariationAxis, bool) {
	for _, axis := range face.VariationAxes() {
		if axis.Tag == tag {
			return axis, true
		}
	}

	return pdf.VariationAxis{}, false //nolint:exhaustruct // not found
}

func variationCacheBits(sty *ResolvedStyle) (string, string, uint64) {
	if sty == nil {
		return "", "", 0
	}

	optical := sty.FontOpticalSizing
	variations := sty.FontVariationSettings

	if optical == fontOpticalAuto {
		return optical, variations, math.Float64bits(sty.FontSize)
	}

	return optical, variations, 0
}
