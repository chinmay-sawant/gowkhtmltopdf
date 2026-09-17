package layout

import (
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// faceFor selects the TrueType face for a resolved style (bold/italic),
// preferring CSS font-family matches from the opt-in registry, then the
// bundled Liberation FaceSet.
func (e *engine) faceFor(sty *ResolvedStyle) *pdf.Font {
	if sty == nil {
		return e.font
	}

	key := faceStyleKey{
		famHash: sty.famHash,
		weight:  sty.FontWeight,
		italic:  sty.FontItalic,
	}

	if e.faceByStyle != nil {
		if f, ok := e.faceByStyle[key]; ok {
			return f
		}
	}

	face := e.lookupFaceFor(sty)

	if e.faceByStyle == nil {
		e.faceByStyle = make(map[faceStyleKey]*pdf.Font)
	}

	e.faceByStyle[key] = face

	return face
}

// lookupFaceFor is the uncached faceFor path.
func (e *engine) lookupFaceFor(sty *ResolvedStyle) *pdf.Font {
	if sty == nil {
		return e.font
	}

	return resolveFontVariants(sty, e.lookupBaseFaceFor(sty))
}

// lookupBaseFaceFor resolves the CSS family/weight/italic face without the
// font-variation family consumer.
func (e *engine) lookupBaseFaceFor(sty *ResolvedStyle) *pdf.Font {
	if e.registry != nil {
		if f := e.registry.Lookup(sty.FontFamily, sty.FontWeight, sty.FontItalic); f != nil {
			return f
		}
	}

	if e.faces != nil {
		if f := e.faces.ResolveFamily(sty.FontFamily, sty.FontWeight, sty.FontItalic); f != nil {
			return f
		}

		if f := e.faces.Resolve(sty.FontWeight, sty.FontItalic); f != nil {
			return f
		}
	}

	return e.font
}

// fontVariantCapability records the tables the CSS font variation family
// needs from a resolved face.
type fontVariantCapability struct {
	variationAxes bool // fvar present: variable font
	colorPalette  bool // COLR and CPAL present: color-palette font
}

// resolveFontVariants is the face-resolution consumer for font-optical-sizing,
// font-variation-settings, and font-palette. It reads the three fields and the
// resolved face's OpenType tables, then returns the face the writer will use.
//
// Static faces (every bundled Liberation and DejaVu face) have no fvar and no
// COLR/CPAL; CSS makes all three properties no-ops there, so returning the
// default face is spec-correct.
//
// A registry face loaded with --font-path can expose fvar and/or COLR+CPAL.
// This writer cannot apply either: pdf.Font embeds default-instance glyf
// outlines and has no CPAL/COLR painting path, and go-text v0.3.4 variable
// instancing (font.Face.SetVariations) only affects the shaping/raster face,
// not the embedded outlines. Such a face still resolves to its default
// instance. That is a known gap, recorded rather than faked by shaping with
// variation coordinates the PDF would not embed.
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
	if (wantsAxes && !capability.variationAxes) || (wantsPalette && !capability.colorPalette) {
		return face
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

// faceForRune picks the face for one rune: the @font-face descriptor-aware
// registry lookup first (unicode-range partitions paint each rune with its
// declared face), then browser-like family/default fallback for runes the
// chosen face does not map, so Hangul/Latin/CJK can come from different faces
// in one run.
func (e *engine) faceForRune(sty *ResolvedStyle, runeValue rune) *pdf.Font {
	if sty == nil {
		return e.font
	}

	return e.runeFace(sty, e.faceFor(sty), runeValue)
}

// runeFace resolves one rune against a known style primary face.
//
// Whitespace always uses the primary face: layout trims it and inline paint
// never checks its cmap. Without a registry the primary fast path stands
// (common Latin/report text). With a registry the decision is per codepoint
// because a rune declared to a non-primary unicode-range partition must paint
// with its declared face even when the primary maps the same codepoint; only a
// declared face that actually maps the rune wins, so glyph-missing runes still
// walk the existing family/default scan.
func (e *engine) runeFace(sty *ResolvedStyle, primary *pdf.Font, runeValue rune) *pdf.Font {
	if sty == nil {
		return primary
	}

	if isRuneWhitespace(runeValue) {
		return primary
	}

	if e.registry == nil && primary != nil && primary.GlyphID(runeValue) != 0 {
		return primary
	}

	face := e.faceForRuneCached(sty, runeValue, primary)
	if face == nil {
		face = e.font
	}

	return face
}

// faceForRuneCached memoizes the per-rune face for one style identity. The
// key hashes the family tokens, so the lookup does not allocate a joined
// family string.
func (e *engine) faceForRuneCached(sty *ResolvedStyle, runeValue rune, primary *pdf.Font) *pdf.Font {
	if sty == nil {
		return primary
	}

	key := faceRuneKey{
		famHash: sty.famHash,
		weight:  sty.FontWeight,
		italic:  sty.FontItalic,
		r:       runeValue,
	}

	if e.faceByRune != nil {
		if f, ok := e.faceByRune[key]; ok {
			return f
		}
	}

	face := e.resolveRuneFace(sty, runeValue, primary)

	if e.faceByRune == nil {
		e.faceByRune = make(map[faceRuneKey]*pdf.Font)
	}

	e.faceByRune[key] = face

	return face
}

// resolveRuneFace is the uncached per-rune decision: the declared partition
// face wins when it maps the rune; otherwise the style primary keeps the rune
// when it has the glyph, then the family/default glyph scan runs.
func (e *engine) resolveRuneFace(sty *ResolvedStyle, runeValue rune, primary *pdf.Font) *pdf.Font {
	if e.registry != nil {
		if f := e.registry.LookupRune(sty.FontFamily, sty.FontWeight, sty.FontItalic, runeValue); f != nil &&
			f.GlyphID(runeValue) != 0 {
			return f
		}
	}

	if primary != nil && primary.GlyphID(runeValue) != 0 {
		return primary
	}

	face := e.lookupFaceForRune(sty, runeValue)
	if face == nil {
		face = primary
	}

	return face
}

// lookupFaceForRune is the uncached face resolution path.
func (e *engine) lookupFaceForRune(sty *ResolvedStyle, runeValue rune) *pdf.Font {
	if sty == nil {
		return e.font
	}

	if f := e.registryFamilyWithGlyph(sty, runeValue); f != nil {
		return f
	}

	if f := e.facesWithGlyph(sty, runeValue); f != nil {
		return f
	}

	if e.font != nil && e.font.GlyphID(runeValue) != 0 {
		return e.font
	}

	// Last resort: any opt-in registry face that covers this codepoint
	// (DejaVu/Noto when --font-path / --use-system-fonts scanned them).
	if f := e.registryGlyphFallback(sty, runeValue); f != nil {
		return f
	}

	return e.faceFor(sty)
}

// isRuneWhitespace reports whether r is a rune that inline layout trims.
func isRuneWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}
