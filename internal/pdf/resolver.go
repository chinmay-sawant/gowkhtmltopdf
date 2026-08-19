package pdf

import (
	"fmt"
	"strings"
)

// FontResolver is the single family/glyph selection seam for layout and HF.
// It walks the author CSS stack with exact registry matches first, then
// generics / real Liberation names / system-ui, then Liberation Sans as the
// terminal default. Legacy display-name aliases (Georgia, Arial, …) are never
// rewritten.
type FontResolver struct {
	Faces    *FaceSet
	Registry *Registry
	// Warn is an optional diagnostic sink. It is not on the hot layout path
	// unless a caller marks faces unavailable.
	Warn func(string)

	unavailable map[[32]byte]struct{}
}

// NewFontResolver builds a resolver over bundled faces and an optional registry.
func NewFontResolver(faces *FaceSet, reg *Registry) *FontResolver {
	return &FontResolver{ //nolint:exhaustruct // intentional zero Warn / unavailable
		Faces:    faces,
		Registry: reg,
	}
}

// MarkUnavailable excludes a face from later selection (embed-preflight miss).
// Convert owns re-layout (Phase 5); the resolver only skips the candidate.
func (r *FontResolver) MarkUnavailable(fnt *Font, reason string) {
	if r == nil || fnt == nil {
		return
	}

	if r.unavailable == nil {
		r.unavailable = make(map[[32]byte]struct{})
	}

	r.unavailable[fnt.fingerprint] = struct{}{}
	r.warnf("font %s unavailable: %s", fnt.PostScriptName, reason)
}

func (r *FontResolver) isUnavailable(fnt *Font) bool {
	if r == nil || fnt == nil || len(r.unavailable) == 0 {
		return false
	}

	_, ok := r.unavailable[fnt.fingerprint]

	return ok
}

func (r *FontResolver) warnf(format string, args ...any) {
	if r == nil || r.Warn == nil {
		return
	}

	r.Warn(fmt.Sprintf(format, args...))
}

// ResolveFamilyStyle walks CSS families per the Phase 1 resolution contract.
func (r *FontResolver) ResolveFamilyStyle(families []string, weight int, italic bool) *Font {
	if r == nil {
		return nil
	}

	for _, fam := range families {
		if f := r.resolveToken(fam, weight, italic); f != nil {
			return f
		}
	}

	if r.Faces != nil {
		if f := r.usable(r.Faces.Resolve(weight, italic)); f != nil {
			return f
		}
	}

	return nil
}

// ResolveRune picks a face covering codePoint without replacing the primary
// face for runes it already covers (whitespace and present glyphs stay on
// primary).
//
//nolint:cyclop // ordered family/glyph fallback is intentionally explicit
func (r *FontResolver) ResolveRune(families []string, weight int, italic bool, codePoint rune, primary *Font) *Font {
	if r == nil {
		return primary
	}

	if isResolverWhitespace(codePoint) {
		return primary
	}

	if primary != nil && primary.GlyphID(codePoint) != 0 && !r.isUnavailable(primary) {
		return primary
	}

	for _, fam := range families {
		if f := r.resolveToken(fam, weight, italic); f != nil && f.GlyphID(codePoint) != 0 {
			return f
		}
	}

	if r.Faces != nil {
		if f := r.usable(r.Faces.Resolve(weight, italic)); f != nil && f.GlyphID(codePoint) != 0 {
			return f
		}

		if weight >= fontWeightBoldMin &&
			r.Faces.UnicodeFallbackBold != nil &&
			r.Faces.UnicodeFallbackBold.GlyphID(codePoint) != 0 &&
			!r.isUnavailable(r.Faces.UnicodeFallbackBold) {
			return r.Faces.UnicodeFallbackBold
		}

		if r.Faces.UnicodeFallback != nil &&
			r.Faces.UnicodeFallback.GlyphID(codePoint) != 0 &&
			!r.isUnavailable(r.Faces.UnicodeFallback) {
			return r.Faces.UnicodeFallback
		}
	}

	if r.Registry != nil {
		if f := r.usable(r.Registry.FindWithGlyph(codePoint, weight, italic)); f != nil {
			return f
		}
	}

	return primary
}

func (r *FontResolver) resolveToken(fam string, weight int, italic bool) *Font {
	key := normalizeFamilyToken(fam)
	if key == "" {
		return nil
	}

	if r.Registry != nil {
		// Lookup expands only CSS generics; named tokens stay exact.
		if f := r.usable(r.Registry.Lookup([]string{fam}, weight, italic)); f != nil {
			return f
		}
	}

	return r.resolveBundledToken(key, weight, italic)
}

func (r *FontResolver) resolveBundledToken(key string, weight int, italic bool) *Font {
	if r.Faces == nil {
		return nil
	}

	switch key {
	case "serif", "liberation serif":
		return r.usable(resolveFamilyFaces(
			r.Faces.Serif, r.Faces.SerifBold, r.Faces.SerifItalic, r.Faces.SerifBoldItalic,
			weight, italic,
		))
	case "monospace", "liberation mono":
		return r.usable(resolveFamilyFaces(
			r.Faces.Mono, r.Faces.MonoBold, r.Faces.MonoItalic, r.Faces.MonoBoldItalic,
			weight, italic,
		))
	case "sans-serif", "liberation sans":
		return r.usable(r.Faces.Resolve(weight, italic))
	case "system-ui":
		return r.usable(resolveFamilyFaces(
			r.Faces.UnicodeFallback, r.Faces.UnicodeFallbackBold, nil, r.Faces.UnicodeFallbackBold,
			weight, italic,
		))
	default:
		return nil
	}
}

func (r *FontResolver) usable(fnt *Font) *Font {
	if fnt == nil || r.isUnavailable(fnt) {
		return nil
	}

	return fnt
}

func normalizeFamilyToken(fam string) string {
	return strings.ToLower(strings.Trim(strings.TrimSpace(fam), `"'`))
}

func isResolverWhitespace(r rune) bool {
	return r == ' ' || r == '\t' || r == '\n' || r == '\r'
}
