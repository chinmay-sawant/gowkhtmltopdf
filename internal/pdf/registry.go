package pdf

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/line"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// Registry indexes discoverable TTF faces by CSS family name (lowercased).
// Liberation defaults stay available via FaceSet; this holds opt-in folder fonts.
type Registry struct {
	mu            sync.RWMutex
	byFamily      map[string][]*Font // family → faces (any weight/style)
	exactByFamily map[string][]*Font // exact family tokens only (bundled aliases)
	faces         []*Font            // stable registration order for fallback scans
	// specByFace carries the @font-face descriptors (weight, style, unicode
	// range) of webfont registrations. Faces without an entry fall back to
	// their own OS/2 and macStyle metadata.
	specByFace map[*Font]FaceSpec
}

// NewRegistry returns an empty font registry.
func NewRegistry() *Registry {
	return &Registry{ //nolint:exhaustruct // intentional zero-value mu field
		byFamily:   map[string][]*Font{},
		specByFace: map[*Font]FaceSpec{},
	}
}

// LogFontRegistryScan emits the shared font-path scan notice after a registry
// has been built by a PDF or image request.
func LogFontRegistryScan(global settings.PdfGlobal, log io.Writer) {
	if log == nil || log == io.Discard || global.Quiet {
		return
	}

	if len(global.FontPaths) == 0 && !global.UseSystemFonts {
		return
	}

	count := len(global.FontPaths)
	if global.UseSystemFonts {
		count += len(DefaultSystemFontDirs())
	}

	line.Emit(log, line.Info, "scanned %d font path(s)", count)
}

func (r *Registry) registerFaceLocked(fnt *Font) {
	for _, existing := range r.faces {
		if existing == fnt {
			return
		}
	}

	r.faces = append(r.faces, fnt)
}

// AddFont registers a parsed face under its family name (and PostScript name).
//
//nolint:wsl // lock initialization and registration must remain one critical section.
func (r *Registry) AddFont(fnt *Font) {
	if r == nil || fnt == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byFamily == nil {
		r.byFamily = map[string][]*Font{}
	}
	r.registerFaceLocked(fnt)

	names := fnt.LoadNames()
	if len(names) == 0 && fnt.PostScriptName != "" {
		names = []string{fnt.PostScriptName}
	}

	for _, n := range names {
		key := strings.ToLower(strings.TrimSpace(n))
		if key == "" {
			continue
		}

		r.byFamily[key] = append(r.byFamily[key], fnt)
	}
}

// normalizeFamilyKey lowercases and strips quotes from one CSS family name,
// matching the keys AddFamilyAlias and Lookup use.
func normalizeFamilyKey(family string) string {
	key := strings.ToLower(strings.TrimSpace(family))

	return strings.Trim(key, `"'`)
}

// AddFamilyAlias registers f under an explicit CSS family name. The face
// keeps its own weight/style/glyph metadata: use AddFamilyAliasSpec to attach
// @font-face descriptors.
func (r *Registry) AddFamilyAlias(family string, font *Font) {
	r.AddFamilyAliasSpec(family, font, FaceSpec{}) //nolint:exhaustruct // zero spec means file metadata
}

// AddFamilyAliasSpec registers font under an explicit CSS family name with the
// @font-face descriptors that select it. Selection looks the spec up per face,
// so a face registered earlier by AddFont picks it up too.
//
//nolint:wsl // lock initialization and registration must remain one critical section.
func (r *Registry) AddFamilyAliasSpec(family string, font *Font, spec FaceSpec) {
	if r == nil || font == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byFamily == nil {
		r.byFamily = map[string][]*Font{}
	}

	if r.specByFace == nil {
		r.specByFace = map[*Font]FaceSpec{}
	}

	r.specByFace[font] = spec

	r.registerFaceLocked(font)

	key := normalizeFamilyKey(family)
	if key == "" {
		return
	}

	r.byFamily[key] = append(r.byFamily[key], font)
}

// AddExactFamilyAlias registers font under an explicit CSS family name for
// exact family tokens only. Unlike AddFamilyAlias, the generic serif /
// sans-serif / monospace expansions do not see the alias. The bundled DejaVu
// fallback faces use it: an exact font-family:'DejaVu Sans' must resolve them,
// while the sans-serif expansion (which lists "dejavu sans" as a fallback
// candidate) must keep preferring Liberation. The face is not added to the
// fallback scan order because the FaceSet already covers it.
//
//nolint:wsl // lock initialization and registration must remain one critical section.
func (r *Registry) AddExactFamilyAlias(family string, font *Font) {
	if r == nil || font == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.exactByFamily == nil {
		r.exactByFamily = map[string][]*Font{}
	}

	key := normalizeFamilyKey(family)
	if key == "" {
		return
	}

	r.exactByFamily[key] = append(r.exactByFamily[key], font)
}

// Lookup returns a face matching family list + weight/italic, or nil.
// Each CSS family token is tried as its exact registry key first. Only the
// CSS generics serif / sans-serif / monospace expand to Liberation (and
// similar libre) faces — named families like Georgia are never rewritten.
func (r *Registry) Lookup(families []string, weight int, italic bool) *Font {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, fam := range families {
		if faces := r.familyFaces(fam); len(faces) > 0 {
			if f := r.pickFace(faces, weight, italic); f != nil {
				return f
			}
		}
	}

	return nil
}

// LookupRune returns the face that covers codePoint for the first matching CSS
// family, or nil when no face of any family declares coverage. Faces whose
// declared unicode-range excludes the code point are not candidates; among the
// rest, a face that actually maps the code point beats one that only declares
// it, then weight/style match decides, then registration order. Per-code-point
// lookup is what lets one family split into unicode-range partitions (Google
// Fonts latin and latin-ext) paint each rune with its declared face.
func (r *Registry) LookupRune(families []string, weight int, italic bool, codePoint rune) *Font {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, fam := range families {
		if faces := r.familyFaces(fam); len(faces) > 0 {
			if f := r.pickFaceRune(faces, weight, italic, codePoint); f != nil {
				return f
			}
		}
	}

	return nil
}

// familyFaces returns the candidate faces for one CSS family token: the exact
// registry key first, then the generic expansions. Bundled exact aliases are
// visible only for named tokens, never for a generic expansion.
func (r *Registry) familyFaces(fam string) []*Font {
	keys := fontFamilyKeys(fam)

	for _, key := range keys {
		faces := r.byFamily[key]
		if len(faces) == 0 && len(keys) == 1 {
			faces = r.exactByFamily[key]
		}

		if len(faces) > 0 {
			return faces
		}
	}

	return nil
}

// fontFamilyKeys returns lowercase registry keys to try for one CSS family
// token. Named families stay as-is; only CSS generics expand to Liberation.
func fontFamilyKeys(fam string) []string {
	key := normalizeFamilyKey(fam)
	if key == "" {
		return nil
	}

	switch key {
	case "serif":
		return []string{"liberation serif", "dejavu serif", "noto serif"}
	case "sans-serif":
		return []string{"liberation sans", "dejavu sans", "noto sans"}
	case "monospace":
		return []string{"liberation mono", "dejavu sans mono", "noto sans mono"}
	default:
		return []string{key}
	}
}

// FindWithGlyph returns any registered face that has a glyph for ch, preferring
// weight/italic match. Used as a last-resort Unicode fallback when CSS
// font-family faces (and Liberation) lack the codepoint (e.g. IPA ˈ/ɾ).
func (r *Registry) FindWithGlyph(codePoint rune, weight int, italic bool) *Font {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	bold := weight >= fontWeightBoldMin

	var best *Font

	bestScore := -1

	for _, fnt := range r.faces {
		score := r.glyphFaceScore(fnt, codePoint, bold, italic)
		if score < 0 {
			continue
		}

		if score > bestScore || (score == bestScore && fontIdentityLess(fnt, best)) {
			bestScore = score
			best = fnt
		}
	}

	return best
}

// fontIdentityLess provides a stable tie-breaker independent of map iteration
// or alias registration order. The parsed fingerprint distinguishes different
// files that happen to share a PostScript name; the name is a readable
// fallback for synthetic/test faces without a fingerprint.
//
//nolint:wsl // tie-break fields are intentionally checked in priority order.
func fontIdentityLess(left, right *Font) bool {
	if left == nil {
		return false
	}
	if right == nil {
		return true
	}
	if cmp := bytes.Compare(left.fingerprint[:], right.fingerprint[:]); cmp != 0 {
		return cmp < 0
	}

	return strings.ToLower(left.PostScriptName) < strings.ToLower(right.PostScriptName)
}

// glyphFaceScore scores a face for ch: -1 when it lacks the glyph, plus
// weight/italic match bonuses from the @font-face spec (falling back to the
// file's own metadata) and a premium for known Unicode-capable families
// (DejaVu/Noto/FreeSans).
//
//nolint:cyclop // glyph scoring logic
func (r *Registry) glyphFaceScore(fnt *Font, codePoint rune, bold, italic bool) int {
	if fnt == nil || fnt.GlyphID(codePoint) == 0 {
		return -1
	}

	spec := r.specByFace[fnt]

	score := 1
	if r.faceBold(fnt, spec) == bold {
		score += 2
	}

	if r.faceItalic(fnt, spec) == italic {
		score += 2
	}

	psLow := strings.ToLower(fnt.PostScriptName)
	if strings.Contains(psLow, "dejavu") || strings.Contains(psLow, "noto") || strings.Contains(psLow, "freesans") {
		score += 3
	} else {
		for _, n := range fnt.FamilyNames() {
			low := strings.ToLower(n)
			if strings.Contains(low, "dejavu") || strings.Contains(low, "noto") || strings.Contains(low, "freesans") {
				score += 3

				break
			}
		}
	}

	return score
}

// italicMatchScore outweighs one weight step so an italic request prefers the
// family's italic face over a same-weight upright, and vice versa.
const italicMatchScore = 4

// pickFace selects the face closest to the requested CSS weight and style.
// Weight comes from the @font-face font-weight descriptor when declared, then
// from the OS/2 usWeightClass, so a family that ships separate 400/500/700
// files is not forced onto whichever face registered first; a face without
// usable OS/2 data reports the CSS default 400. Equal scores are broken by
// unicode-range coverage of the primary probe runes, then by the face's own
// glyph coverage, then by registration order, which keeps the result
// deterministic.
func (r *Registry) pickFace(faces []*Font, weight int, italic bool) *Font {
	var best *Font

	bestScore, bestRange, bestGlyph := -1, -1, -1

	for _, fnt := range faces {
		spec := r.specByFace[fnt]

		score := r.styleMatchScore(fnt, spec, weight, italic)
		ranges := spec.rangeProbeScore()
		glyphs := glyphProbeScore(fnt)

		if score > bestScore ||
			(score == bestScore && ranges > bestRange) ||
			(score == bestScore && ranges == bestRange && glyphs > bestGlyph) {
			best = fnt
			bestScore, bestRange, bestGlyph = score, ranges, glyphs
		}
	}

	return best
}

// pickFaceRune selects the face for one code point: faces whose declared
// unicode-range excludes it are not candidates, and among the rest a face that
// actually maps the code point outranks one that only declares it. Returns nil
// when no face of the family covers the code point.
func (r *Registry) pickFaceRune(faces []*Font, weight int, italic bool, codePoint rune) *Font {
	var best *Font

	bestGlyph, bestScore := -1, -1

	for _, fnt := range faces {
		spec := r.specByFace[fnt]
		if !spec.covers(codePoint) {
			continue
		}

		glyph := 0
		if fnt.GlyphID(codePoint) != 0 {
			glyph = 1
		}

		score := r.styleMatchScore(fnt, spec, weight, italic)

		if glyph > bestGlyph || (glyph == bestGlyph && score > bestScore) {
			best = fnt
			bestGlyph, bestScore = glyph, score
		}
	}

	return best
}

// styleMatchScore ranks one face against a weight/style request using the
// @font-face descriptors when declared and the file's own metadata otherwise.
func (r *Registry) styleMatchScore(fnt *Font, spec FaceSpec, weight int, italic bool) int {
	score := weightMatchScore(r.faceWeight(fnt, spec), weight)

	if r.faceItalic(fnt, spec) == italic {
		score += italicMatchScore
	}

	return score
}

// faceWeight returns the face's declared CSS weight: the @font-face
// font-weight descriptor when present, else the OS/2/macStyle weight class.
func (r *Registry) faceWeight(fnt *Font, spec FaceSpec) int {
	if spec.Weight > 0 {
		return spec.Weight
	}

	return fnt.WeightClass()
}

// faceBold reports whether the face matches a bold request.
func (r *Registry) faceBold(fnt *Font, spec FaceSpec) bool {
	if spec.Weight > 0 {
		return spec.Weight >= fontWeightBoldMin
	}

	return fnt.Bold()
}

// faceItalic returns the face's declared style: the @font-face font-style
// descriptor when present, else the macStyle italic bit.
func (r *Registry) faceItalic(fnt *Font, spec FaceSpec) bool {
	if spec.StyleSet {
		return spec.Italic
	}

	return fnt.Italic()
}

// weightMatchScore ranks a declared weight against a CSS weight request: an
// exact match scores highest, then the closest 100-unit step. Distances of
// four steps or more score zero, so the italic match decides those pairs.
func weightMatchScore(faceWeight, requested int) int {
	const (
		exactScore = 4
		step       = 100
	)

	distance := faceWeight - requested
	if distance < 0 {
		distance = -distance
	}

	steps := distance / step
	if steps >= exactScore {
		return 0
	}

	return exactScore - steps
}

// HasVariationAxes reports whether the face carries an fvar table, that is,
// whether it is a variable font. The bundled Liberation and DejaVu faces are
// static, so this is false for every default face.
//
// The font-variation consumer in internal/layout uses the probe to tell a
// spec-correct no-op (static face, CSS variations have no effect) from a known
// gap (variable face the writer cannot instance).
func (f *Font) HasVariationAxes() bool {
	if f == nil {
		return false
	}

	f.ensureParsed()

	_, ok := f.tables["fvar"]

	return ok
}

// HasColorPalette reports whether the face carries both COLR and CPAL, the
// tables font-palette needs to select a color palette. The PDF writer embeds
// glyf outlines only and has no CPAL/COLR painting path, so a true result
// identifies a known gap rather than supported palette painting.
func (f *Font) HasColorPalette() bool {
	if f == nil {
		return false
	}

	f.ensureParsed()

	_, hasCOLR := f.tables["COLR"]
	_, hasCPAL := f.tables["CPAL"]

	return hasCOLR && hasCPAL
}

// DefaultSystemFontDirs returns common system font directories for the current OS.
// Callers must opt in via --use-system-fonts; nothing is scanned by default.
// Proprietary Windows/corefont trees are omitted — use Liberation (bundled)
// plus libre faces under /usr/share/fonts (DejaVu/Noto for IPA fallback).
func DefaultSystemFontDirs() []string {
	dirs := []string{
		"/usr/share/fonts",
		"/usr/local/share/fonts",
		"/usr/share/fonts/truetype",
		"/usr/share/fonts/truetype/droid",
		"/usr/share/fonts/opentype",
	}

	if home, err := os.UserHomeDir(); err == nil {
		for _, rel := range []string{".fonts", ".local/share/fonts"} {
			d := filepath.Join(home, rel)
			if st, err := os.Stat(d); err == nil && st.IsDir() {
				dirs = append(dirs, d)
			}
		}
	}

	return dirs
}

// ScanFontDirs walks each directory non-recursively (and one level of
// subdirectories under /usr/share/fonts style trees) collecting .ttf faces.
func ScanFontDirs(dirs []string) *Registry {
	out := NewRegistry()
	seen := map[string]bool{}

	var scan func(string, int)

	scan = func(dir string, depth int) {
		if dir == "" || seen[dir] {
			return
		}

		seen[dir] = true

		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}

		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())

			if entry.IsDir() {
				if depth > 0 {
					scan(path, depth-1)
				}

				continue
			}

			scanFontFile(out, path, entry)
		}
	}
	for _, d := range dirs {
		scan(d, fontScanMaxDepth)
	}

	return out
}

// RegistryFromPaths builds an opt-in font registry from explicit font paths
// and optional system font directories. Returns nil when nothing was configured.
func RegistryFromPaths(fontPaths []string, useSystemFonts bool) *Registry {
	var dirs []string

	dirs = append(dirs, fontPaths...)

	if useSystemFonts {
		dirs = append(dirs, DefaultSystemFontDirs()...)
	}

	if len(dirs) == 0 {
		return nil
	}

	return ScanFontDirs(dirs)
}

// RegistryFromGlobal builds the font registry for one conversion from
// PdfGlobal font settings. It always returns a registry: even when no font
// paths are configured, the bundled DejaVu Sans fallback faces are registered
// as exact family aliases so font-family:'DejaVu Sans' resolves without opt-in
// discovery. Callers own logging.
func RegistryFromGlobal(global settings.PdfGlobal) *Registry {
	registry := RegistryFromPaths(global.FontPaths, global.UseSystemFonts)
	if registry == nil {
		registry = NewRegistry()
	}

	// Exact aliases only: the generic sans-serif expansion lists "dejavu sans"
	// as a fallback candidate, and registering the bundled faces there would
	// switch every generic sans-serif run from Liberation to DejaVu.
	if faces, err := LoadDefaultFaces(); err == nil {
		registry.AddExactFamilyAlias(dejaVuSansFamily, faces.UnicodeFallback)
		registry.AddExactFamilyAlias(dejaVuSansFamily, faces.UnicodeFallbackBold)
	}

	return registry
}

// scanFontFile parses a font file into the registry, skipping anything that
// is not a TTF/OTF or fails to parse. Parses are memoized across conversions
// by loadFontFile (see font_file_cache.go).
func scanFontFile(out *Registry, path string, entry os.DirEntry) {
	low := strings.ToLower(entry.Name())
	if !strings.HasSuffix(low, ".ttf") && !strings.HasSuffix(low, ".otf") {
		return
	}

	fnt := loadFontFile(path, entry)
	if fnt != nil {
		out.AddFont(fnt)
	}
}
