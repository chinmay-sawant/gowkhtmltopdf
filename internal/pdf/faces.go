package pdf

import (
	"bytes"
	"strings"
	"sync"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf/assets"
)

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
	errDefaultFaces  error
)

// LoadDefaultFaces returns the embedded Liberation families and Unicode
// fallback faces. The result is cached.
//
//nolint:cyclop,lll,funlen // face loading is an explicit fail-fast initialization sequence
func LoadDefaultFaces() (*FaceSet, error) {
	defaultFacesOnce.Do(func() {
		faces := &FaceSet{} //nolint:exhaustruct // intentional zero-value fields

		var err error
		if faces.Regular, err = parseNamed(fallbackFontName, assets.LiberationSansRegular()); err != nil {
			errDefaultFaces = err

			return
		}

		if faces.Bold, err = parseNamed("LiberationSans-Bold", assets.LiberationSansBold()); err != nil {
			errDefaultFaces = err

			return
		}

		if faces.Italic, err = parseNamed("LiberationSans-Italic", assets.LiberationSansItalic()); err != nil {
			errDefaultFaces = err

			return
		}

		if faces.BoldItalic, err = parseNamed("LiberationSans-BoldItalic", assets.LiberationSansBoldItalic()); err != nil {
			errDefaultFaces = err

			return
		}

		if faces.Serif, err = parseNamed("LiberationSerif", assets.LiberationSerifRegular()); err != nil {
			errDefaultFaces = err

			return
		}

		if faces.SerifBold, err = parseNamed("LiberationSerif-Bold", assets.LiberationSerifBold()); err != nil {
			errDefaultFaces = err

			return
		}

		if faces.SerifItalic, err = parseNamed("LiberationSerif-Italic", assets.LiberationSerifItalic()); err != nil {
			errDefaultFaces = err

			return
		}

		if faces.SerifBoldItalic, err = parseNamed("LiberationSerif-BoldItalic", assets.LiberationSerifBoldItalic()); err != nil {
			errDefaultFaces = err

			return
		}

		if faces.Mono, err = parseNamed("LiberationMono", assets.LiberationMonoRegular()); err != nil {
			errDefaultFaces = err

			return
		}

		if faces.MonoBold, err = parseNamed("LiberationMono-Bold", assets.LiberationMonoBold()); err != nil {
			errDefaultFaces = err

			return
		}

		if faces.MonoItalic, err = parseNamed("LiberationMono-Italic", assets.LiberationMonoItalic()); err != nil {
			errDefaultFaces = err

			return
		}

		if faces.MonoBoldItalic, err = parseNamed("LiberationMono-BoldItalic", assets.LiberationMonoBoldItalic()); err != nil {
			errDefaultFaces = err

			return
		}

		if faces.UnicodeFallback, err = parseNamed("DejaVuSans-UnicodeFallback", assets.UnicodeFallbackRegular()); err != nil {
			errDefaultFaces = err

			return
		}

		if faces.UnicodeFallbackBold, err = parseNamed("DejaVuSans-UnicodeFallback-Bold", assets.UnicodeFallbackBold()); err != nil {
			errDefaultFaces = err

			return
		}

		defaultFaces = faces
	})

	return defaultFaces, errDefaultFaces
}

func parseNamed(name string, data []byte) (*Font, error) {
	fnt, err := ParseTTF(bytes.Clone(data))
	if err != nil {
		return nil, err
	}

	fnt.PostScriptName = name

	return fnt, nil
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
		case "system-ui":
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
