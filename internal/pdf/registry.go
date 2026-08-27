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
	mu       sync.RWMutex
	byFamily map[string][]*Font // family → faces (any weight/style)
	faces    []*Font            // stable registration order for fallback scans
}

// NewRegistry returns an empty font registry.
func NewRegistry() *Registry {
	return &Registry{ //nolint:exhaustruct // intentional zero-value mu field
		byFamily: map[string][]*Font{},
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

// AddFamilyAlias registers f under an explicit CSS family name.
//
//nolint:wsl // lock initialization and registration must remain one critical section.
func (r *Registry) AddFamilyAlias(family string, font *Font) {
	if r == nil || font == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.byFamily == nil {
		r.byFamily = map[string][]*Font{}
	}
	r.registerFaceLocked(font)

	key := strings.ToLower(strings.TrimSpace(family))
	key = strings.Trim(key, `"'`)

	if key == "" {
		return
	}

	r.byFamily[key] = append(r.byFamily[key], font)
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
		for _, key := range fontFamilyKeys(fam) {
			faces := r.byFamily[key]
			if len(faces) == 0 {
				continue
			}

			if f := pickFace(faces, weight, italic); f != nil {
				return f
			}
		}
	}

	return nil
}

// fontFamilyKeys returns lowercase registry keys to try for one CSS family
// token. Named families stay as-is; only CSS generics expand to Liberation.
func fontFamilyKeys(fam string) []string {
	key := strings.ToLower(strings.TrimSpace(fam))
	key = strings.Trim(key, `"'`)

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
		score := glyphFaceScore(fnt, codePoint, bold, italic)
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
// weight/italic match bonuses and a premium for known Unicode-capable
// families (DejaVu/Noto/FreeSans).
//
//nolint:cyclop // glyph scoring logic
func glyphFaceScore(fnt *Font, codePoint rune, bold, italic bool) int {
	if fnt == nil || fnt.GlyphID(codePoint) == 0 {
		return -1
	}

	score := 1
	if fnt.Bold() == bold {
		score += 2
	}

	if fnt.Italic() == italic {
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

func pickFace(faces []*Font, weight int, italic bool) *Font {
	bold := weight >= fontWeightBoldMin

	var best *Font

	bestScore := -1

	for _, fnt := range faces {
		score := 0
		if fnt.Bold() == bold {
			score += 2
		}

		if fnt.Italic() == italic {
			score += 2
		}

		if score > bestScore {
			bestScore = score
			best = fnt
		}
	}

	return best
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

// RegistryFromGlobal builds an opt-in font registry from PdfGlobal font
// settings. Returns nil when nothing was configured. Callers own logging.
func RegistryFromGlobal(global settings.PdfGlobal) *Registry {
	return RegistryFromPaths(global.FontPaths, global.UseSystemFonts)
}

// scanFontFile parses a font file into the registry, skipping anything that
// is not a TTF/OTF or fails to parse.
func scanFontFile(out *Registry, path string, entry os.DirEntry) {
	low := strings.ToLower(entry.Name())
	if !strings.HasSuffix(low, ".ttf") && !strings.HasSuffix(low, ".otf") {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return
	}

	fnt, err := ParseTTF(data)
	if err != nil {
		return
	}

	if fnt.PostScriptName == "" {
		fnt.PostScriptName = strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
	}

	out.AddFont(fnt)
}
