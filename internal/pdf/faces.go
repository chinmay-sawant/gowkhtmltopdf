package pdf

import (
	"strings"
	"sync"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf/assets"
)

// dejaVuSansFamily is the lowercase CSS family key of the bundled DejaVu Sans
// fallback faces. ResolveFamily lowercases its input before comparing, and the
// registry normalizes aliases to the same key.
const dejaVuSansFamily = "dejavu sans"

// FaceSet holds the bundled Liberation CSS families and Unicode fallback faces.
//
// ponytail: Liberation faces bundled in-tree (assets/); system fonts opt-in only.
type FaceSet struct {
	Regular             *Font
	Bold                *Font
	Italic              *Font
	BoldItalic          *Font
	Serif               *Font
	SerifBold           *Font
	SerifItalic         *Font
	SerifBoldItalic     *Font
	Mono                *Font
	MonoBold            *Font
	MonoItalic          *Font
	MonoBoldItalic      *Font
	UnicodeFallback     *Font
	UnicodeFallbackBold *Font
}

//nolint:gochecknoglobals // lazy init of the embedded-family cache
var (
	defaultFacesOnce sync.Once
	defaultFaces     *FaceSet
)

// LoadDefaultFaces returns the embedded Liberation families and Unicode
// fallback faces. The result is cached, and each face loads (clone + parse) on
// first use, so a document only pays for the faces it touches.
func LoadDefaultFaces() (*FaceSet, error) {
	defaultFacesOnce.Do(func() {
		defaultFaces = &FaceSet{
			Regular:             parseNamed(fallbackFontName, assets.LiberationSansRegular),
			Bold:                parseNamed("LiberationSans-Bold", assets.LiberationSansBold),
			Italic:              parseNamed("LiberationSans-Italic", assets.LiberationSansItalic),
			BoldItalic:          parseNamed("LiberationSans-BoldItalic", assets.LiberationSansBoldItalic),
			Serif:               parseNamed("LiberationSerif", assets.LiberationSerifRegular),
			SerifBold:           parseNamed("LiberationSerif-Bold", assets.LiberationSerifBold),
			SerifItalic:         parseNamed("LiberationSerif-Italic", assets.LiberationSerifItalic),
			SerifBoldItalic:     parseNamed("LiberationSerif-BoldItalic", assets.LiberationSerifBoldItalic),
			Mono:                parseNamed("LiberationMono", assets.LiberationMonoRegular),
			MonoBold:            parseNamed("LiberationMono-Bold", assets.LiberationMonoBold),
			MonoItalic:          parseNamed("LiberationMono-Italic", assets.LiberationMonoItalic),
			MonoBoldItalic:      parseNamed("LiberationMono-BoldItalic", assets.LiberationMonoBoldItalic),
			UnicodeFallback:     parseNamed("DejaVuSans-UnicodeFallback", assets.UnicodeFallbackRegular),
			UnicodeFallbackBold: parseNamed("DejaVuSans-UnicodeFallback-Bold", assets.UnicodeFallbackBold),
		}
	})

	return defaultFaces, nil
}

// parseNamed builds one bundled face. The accessor runs on first use, so the
// single asset clone (for example assets.LiberationSansRegular copies the
// embedded bytes once) and the parse are deferred until a document asks for
// the face. A second clone on this path doubled the cold font charge, 3.15 MB
// measured.
func parseNamed(name string, load func() []byte) *Font {
	fnt := newLazyFont(load)
	fnt.PostScriptName = name

	return fnt
}

// Resolve picks a face for the given CSS weight and italic flag.
// Falls back toward Regular when a style is missing.
func (fs *FaceSet) Resolve(weight int, italic bool) *Font {
	return resolveFamilyFaces(fs.Regular, fs.Bold, fs.Italic, fs.BoldItalic, weight, italic)
}

// ResolveFamily selects the bundled family corresponding to CSS named and
// generic families while preserving the first supported family in the list.
func (fs *FaceSet) ResolveFamily(families []string, weight int, italic bool) *Font {
	for _, family := range families {
		switch strings.ToLower(strings.Trim(strings.TrimSpace(family), `"'`)) {
		case "serif", "georgia", "times", "times new roman", "liberation serif":
			return resolveFamilyFaces(fs.Serif, fs.SerifBold, fs.SerifItalic, fs.SerifBoldItalic, weight, italic)
		case "monospace", "courier", "courier new", "consolas", "monaco", "liberation mono":
			return resolveFamilyFaces(fs.Mono, fs.MonoBold, fs.MonoItalic, fs.MonoBoldItalic, weight, italic)
		case "sans-serif", "arial", "helvetica", "tahoma", "verdana", "calibri", "liberation sans":
			return fs.Resolve(weight, italic)
		case "system-ui", dejaVuSansFamily:
			// The DejaVu faces are the Unicode fallback family; an explicit
			// font-family:'DejaVu Sans' (the font-language-override demo)
			// resolves them instead of falling through to Liberation.
			return resolveFamilyFaces(fs.UnicodeFallback, fs.UnicodeFallbackBold, nil, fs.UnicodeFallbackBold, weight, italic)
		}
	}

	return nil
}

//nolint:cyclop // fallback precedence is intentionally explicit
func resolveFamilyFaces(regular, boldFace, italicFace, boldItalic *Font, weight int, italic bool) *Font {
	if regular == nil && boldFace == nil && italicFace == nil && boldItalic == nil {
		return nil
	}

	bold := weight >= fontWeightBoldMin

	if bold {
		if italic && boldItalic != nil {
			return boldItalic
		}

		if boldFace != nil {
			return boldFace
		}
	}

	if italic && italicFace != nil {
		return italicFace
	}

	if regular != nil {
		return regular
	}

	if boldFace != nil {
		return boldFace
	}

	return italicFace
}

// DefaultFont returns the embedded Liberation Sans regular face.
func DefaultFont() (*Font, error) {
	fs, err := LoadDefaultFaces()
	if err != nil {
		return nil, err
	}

	return fs.Regular, nil
}
