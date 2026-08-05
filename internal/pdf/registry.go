package pdf

import (
	"os"
	"path/filepath"
	"strings"
)

// Registry indexes discoverable TTF faces by CSS family name (lowercased).
// Liberation defaults stay available via FaceSet; this holds opt-in folder fonts.
type Registry struct {
	byFamily map[string][]*Font // family → faces (any weight/style)
}

// NewRegistry returns an empty font registry.
func NewRegistry() *Registry {
	return &Registry{byFamily: map[string][]*Font{}}
}

// AddFont registers a parsed face under its family name (and PostScript name).
func (r *Registry) AddFont(f *Font) {
	if r == nil || f == nil {
		return
	}
	names := f.FamilyNames()
	if len(names) == 0 && f.PostScriptName != "" {
		names = []string{f.PostScriptName}
	}
	for _, n := range names {
		key := strings.ToLower(strings.TrimSpace(n))
		if key == "" {
			continue
		}
		r.byFamily[key] = append(r.byFamily[key], f)
	}
}

// AddFamilyAlias registers f under an explicit CSS family name.
func (r *Registry) AddFamilyAlias(family string, f *Font) {
	if r == nil || f == nil {
		return
	}
	key := strings.ToLower(strings.TrimSpace(family))
	key = strings.Trim(key, `"'`)
	if key == "" {
		return
	}
	r.byFamily[key] = append(r.byFamily[key], f)
}

// Lookup returns a face matching family list + weight/italic, or nil.
// CSS generic families and common proprietary names (Georgia, Arial, …)
// map to metrically compatible Liberation faces so we never require
// licensed system fonts — only Liberation (bundled) or opt-in free fonts.
func (r *Registry) Lookup(families []string, weight int, italic bool) *Font {
	if r == nil {
		return nil
	}
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
// token. Generics and proprietary names resolve to Liberation only — never
// Georgia/Arial/Times TTFs. Bundled FaceSet covers Liberation Sans when the
// registry has no Liberation Serif/Mono from --use-system-fonts.
func fontFamilyKeys(fam string) []string {
	key := strings.ToLower(strings.TrimSpace(fam))
	key = strings.Trim(key, `"'`)
	if key == "" {
		return nil
	}
	switch key {
	case "serif", "georgia", "times", "times new roman", "times new roman ps",
		"linux libertine", "source serif 4", "source serif pro", "cambria":
		return []string{"liberation serif"}
	case "sans-serif", "arial", "helvetica", "helvetica neue", "verdana",
		"tahoma", "segoe ui", "system-ui", "ui-sans-serif":
		return []string{"liberation sans"}
	case "monospace", "courier", "courier new", "consolas", "menlo", "monaco",
		"ui-monospace":
		return []string{"liberation mono"}
	default:
		return []string{key}
	}
}

// FindWithGlyph returns any registered face that has a glyph for ch, preferring
// weight/italic match. Used as a last-resort Unicode fallback when CSS
// font-family faces (and Liberation) lack the codepoint (e.g. IPA ˈ/ɾ).
func (reg *Registry) FindWithGlyph(ch rune, weight int, italic bool) *Font {
	if reg == nil {
		return nil
	}
	bold := weight >= 700
	var best *Font
	bestScore := -1
	seen := map[*Font]bool{}
	for _, faces := range reg.byFamily {
		for _, f := range faces {
			if f == nil || seen[f] || f.GlyphID(ch) == 0 {
				continue
			}
			seen[f] = true
			score := 1
			if f.Bold() == bold {
				score += 2
			}
			if f.Italic() == italic {
				score += 2
			}
			// Prefer known Unicode-capable families when several match.
			for _, n := range f.FamilyNames() {
				low := strings.ToLower(n)
				if strings.Contains(low, "dejavu") || strings.Contains(low, "noto") || strings.Contains(low, "freesans") {
					score += 3
					break
				}
			}
			if score > bestScore {
				bestScore = score
				best = f
			}
		}
	}
	return best
}

func pickFace(faces []*Font, weight int, italic bool) *Font {
	bold := weight >= 700
	var best *Font
	bestScore := -1
	for _, f := range faces {
		score := 0
		if f.Bold() == bold {
			score += 2
		}
		if f.Italic() == italic {
			score += 2
		}
		if score > bestScore {
			bestScore = score
			best = f
		}
	}
	return best
}

// ScanFontDir loads .ttf files from dir (non-recursive). Errors on individual
// files are skipped; the directory itself must be readable.
func ScanFontDir(dir string) (*Registry, error) {
	r := NewRegistry()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		low := strings.ToLower(name)
		if !strings.HasSuffix(low, ".ttf") && !strings.HasSuffix(low, ".otf") {
			continue
		}
		path := filepath.Join(dir, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		f, err := ParseTTF(data)
		if err != nil {
			continue
		}
		if f.PostScriptName == "" {
			f.PostScriptName = strings.TrimSuffix(name, filepath.Ext(name))
		}
		r.AddFont(f)
	}
	return r, nil
}

// MergeRegistries combines multiple registries (later faces append).
func MergeRegistries(regs ...*Registry) *Registry {
	out := NewRegistry()
	for _, r := range regs {
		if r == nil {
			continue
		}
		for fam, faces := range r.byFamily {
			out.byFamily[fam] = append(out.byFamily[fam], faces...)
		}
	}
	return out
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
		for _, e := range entries {
			path := filepath.Join(dir, e.Name())
			if e.IsDir() {
				if depth > 0 {
					scan(path, depth-1)
				}
				continue
			}
			low := strings.ToLower(e.Name())
			if !strings.HasSuffix(low, ".ttf") && !strings.HasSuffix(low, ".otf") {
				continue
			}
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			f, err := ParseTTF(data)
			if err != nil {
				continue
			}
			if f.PostScriptName == "" {
				f.PostScriptName = strings.TrimSuffix(e.Name(), filepath.Ext(e.Name()))
			}
			out.AddFont(f)
		}
	}
	for _, d := range dirs {
		scan(d, 2)
	}
	return out
}
