package pdf

import (
	"bytes"
	"sync"

	"gowkhtmltopdf/internal/pdf/assets"
)

// FaceSet holds the four Liberation Sans faces used for CSS
// font-weight / font-style selection.
//
// ponytail: Liberation faces bundled in-tree (assets/); system fonts opt-in only.
type FaceSet struct {
	Regular    *Font
	Bold       *Font
	Italic     *Font
	BoldItalic *Font
}

//nolint:gochecknoglobals // lazy init of the embedded-family cache
var (
	defaultFacesOnce sync.Once
	defaultFaces     *FaceSet
	errDefaultFaces  error
)

// LoadDefaultFaces returns the embedded Liberation Sans family
// (Regular, Bold, Italic, BoldItalic). The result is cached.
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
	if fs == nil {
		return nil
	}

	bold := weight >= fontWeightBoldMin

	if bold {
		if italic && fs.BoldItalic != nil {
			return fs.BoldItalic
		}

		if fs.Bold != nil {
			return fs.Bold
		}
	}

	if italic && fs.Italic != nil {
		return fs.Italic
	}

	if fs.Regular != nil {
		return fs.Regular
	}

	if fs.Bold != nil {
		return fs.Bold
	}

	return fs.Italic
}

// DefaultFont returns the embedded Liberation Sans regular face.
func DefaultFont() (*Font, error) {
	fs, err := LoadDefaultFaces()
	if err != nil {
		return nil, err
	}

	return fs.Regular, nil
}
