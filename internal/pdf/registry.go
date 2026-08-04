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

// Lookup returns a face matching family list + weight/italic, or nil.
func (r *Registry) Lookup(families []string, weight int, italic bool) *Font {
	if r == nil {
		return nil
	}
	for _, fam := range families {
		key := strings.ToLower(strings.TrimSpace(fam))
		key = strings.Trim(key, `"'`)
		faces := r.byFamily[key]
		if len(faces) == 0 {
			continue
		}
		if f := pickFace(faces, weight, italic); f != nil {
			return f
		}
	}
	return nil
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
		if !strings.HasSuffix(low, ".ttf") {
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
func DefaultSystemFontDirs() []string {
	return []string{
		"/usr/share/fonts",
		"/usr/local/share/fonts",
		"/usr/share/fonts/truetype",
		"/usr/share/fonts/truetype/droid",
		"/usr/share/fonts/opentype",
	}
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
			if !strings.HasSuffix(low, ".ttf") {
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
