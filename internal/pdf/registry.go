package pdf

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// DiscoverySkip records one skipped path during opt-in font discovery.
type DiscoverySkip struct {
	Path   string
	Reason string
}

// Discovery is the result of scanning --font-path / system font directories.
// Skips never include font file bytes.
type Discovery struct {
	Registry     *Registry
	ScannedPaths []string
	Loaded       int
	Skipped      int
	Skips        []DiscoverySkip
}

// Log writes a compact discovery summary to writer (no font bytes). Quiet
// callers pass io.Discard. Loaded/skipped counts and a few skip reasons are
// included.
func (d *Discovery) Log(writer io.Writer) {
	if writer == nil || writer == io.Discard {
		return
	}

	empty := d.Loaded == 0 && d.Skipped == 0 && len(d.ScannedPaths) == 0
	if empty {
		return
	}

	fmt.Fprintf(writer, "info: font discovery: scanned %d path(s), loaded %d face(s), skipped %d\n",
		len(d.ScannedPaths), d.Loaded, d.Skipped)

	const maxReasons = 8

	for i, skip := range d.Skips {
		if i >= maxReasons {
			fmt.Fprintf(writer, "warning: font discovery: … %d more skip(s)\n", len(d.Skips)-maxReasons)

			break
		}

		fmt.Fprintf(writer, "warning: font discovery: skip %s: %s\n", skip.Path, skip.Reason)
	}
}

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

		if score > bestScore || (score == bestScore && fontIdentityLess(fnt, best)) {
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

// ScanFontDirs walks each directory (depth fontScanMaxDepth) collecting
// .ttf/.otf faces. Bare file paths that are .ttf/.otf are loaded as a
// convenience; other non-directories are skipped with a reason (never treated
// as an empty directory). Prefer DiscoverFonts when diagnostics are needed.
func ScanFontDirs(dirs []string) *Registry {
	return DiscoverFonts(dirs).Registry
}

// DiscoverFonts scans dirs in order and returns loaded faces plus skip
// diagnostics. Depth is fontScanMaxDepth for directories; extensions are
// .ttf/.otf only. CFF/OTTO, variable (fvar), and parse failures are skipped.
//
//nolint:cyclop,funlen // directory vs file path branching is intentional and local
func DiscoverFonts(dirs []string) Discovery {
	out := NewRegistry()
	seen := map[string]bool{}
	report := Discovery{Registry: out} //nolint:exhaustruct // counters filled below

	var scanDir func(string, int)

	scanDir = func(dir string, depth int) {
		if dir == "" || seen[dir] {
			return
		}

		seen[dir] = true

		report.ScannedPaths = append(report.ScannedPaths, dir)

		entries, err := os.ReadDir(dir)
		if err != nil {
			report.addSkip(dir, "read dir: "+err.Error())

			return
		}

		for _, entry := range entries {
			path := filepath.Join(dir, entry.Name())

			if entry.IsDir() {
				if depth > 0 {
					scanDir(path, depth-1)
				}

				continue
			}

			scanFontFile(&report, out, path)
		}
	}

	for _, path := range dirs {
		if path == "" {
			continue
		}

		info, err := os.Stat(path)
		if err != nil {
			report.addSkip(path, "stat: "+err.Error())

			continue
		}

		if info.IsDir() {
			scanDir(path, fontScanMaxDepth)

			continue
		}

		// Explicit file path: accept .ttf/.otf; never silently empty-dir.
		report.ScannedPaths = append(report.ScannedPaths, path)
		if !isTTFOrOTFPath(path) {
			report.addSkip(path, "font-path expects a directory or .ttf/.otf file")

			continue
		}

		scanFontFile(&report, out, path)
	}

	return report
}

// RegistryFromPaths builds an opt-in font registry from explicit font paths
// and optional system font directories. Returns nil when nothing was configured.
func RegistryFromPaths(fontPaths []string, useSystemFonts bool) *Registry {
	return RegistryFromPathsLog(fontPaths, useSystemFonts, nil)
}

// RegistryFromPathsLog is RegistryFromPaths with discovery diagnostics on log.
func RegistryFromPathsLog(fontPaths []string, useSystemFonts bool, log io.Writer) *Registry {
	var dirs []string

	dirs = append(dirs, fontPaths...)

	if useSystemFonts {
		dirs = append(dirs, DefaultSystemFontDirs()...)
	}

	if len(dirs) == 0 {
		return nil
	}

	report := DiscoverFonts(dirs)
	(&report).Log(log)

	return report.Registry
}

func (d *Discovery) addSkip(path, reason string) {
	d.Skipped++
	d.Skips = append(d.Skips, DiscoverySkip{Path: path, Reason: reason})
}

func isTTFOrOTFPath(path string) bool {
	low := strings.ToLower(path)

	return strings.HasSuffix(low, ".ttf") || strings.HasSuffix(low, ".otf")
}

// scanFontFile parses a font file into the registry, recording skip reasons
// for unsupported extensions, CFF/OTTO, variable fonts, and parse failures.
func scanFontFile(report *Discovery, out *Registry, path string) {
	if !isTTFOrOTFPath(path) {
		return
	}

	data, err := os.ReadFile(path)
	if err != nil {
		report.addSkip(path, "read: "+err.Error())

		return
	}

	fnt, err := ParseTTF(data)
	if err != nil {
		report.addSkip(path, discoverySkipReason(err))

		return
	}

	// Prefer name-table PostScript name; fall back to the file stem only when
	// the name table has none (LoadNames is a no-op when already set).
	_ = fnt.LoadNames()

	if fnt.PostScriptName == "" {
		fnt.PostScriptName = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}

	out.AddFont(fnt)

	report.Loaded++
}

func discoverySkipReason(err error) string {
	switch {
	case errors.Is(err, errFontCFFNotSupported):
		return "CFF/OTTO OpenType not supported (TrueType outlines only)"
	case errors.Is(err, errFontVariableRejected):
		return "variable font (fvar) rejected; use a static face"
	case err != nil:
		return "parse: " + err.Error()
	default:
		return "parse failed"
	}
}
