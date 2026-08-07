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

	names := f.LoadNames()
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
// Each CSS family token is tried as its exact registry key first. Only the
// CSS generics serif / sans-serif / monospace expand to Liberation (and
// similar libre) faces — named families like Georgia are never rewritten.
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
