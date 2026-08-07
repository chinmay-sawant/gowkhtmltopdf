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

var (
	defaultFacesOnce sync.Once
	defaultFaces     *FaceSet
	defaultFacesErr  error
)

// LoadDefaultFaces returns the embedded Liberation Sans family
// (Regular, Bold, Italic, BoldItalic). The result is cached.
func LoadDefaultFaces() (*FaceSet, error) {
	defaultFacesOnce.Do(func() {
		faces := &FaceSet{} //nolint:exhaustruct // intentional zero-value fields

		var err error
		if faces.Regular, err = parseNamed("LiberationSans", assets.LiberationSansRegularTTF); err != nil {
			defaultFacesErr = err

			return
		}

		if faces.Bold, err = parseNamed("LiberationSans-Bold", assets.LiberationSansBoldTTF); err != nil {
			defaultFacesErr = err

			return
		}

		if faces.Italic, err = parseNamed("LiberationSans-Italic", assets.LiberationSansItalicTTF); err != nil {
			defaultFacesErr = err

			return
		}

		if faces.BoldItalic, err = parseNamed("LiberationSans-BoldItalic", assets.LiberationSansBoldItalicTTF); err != nil {
			defaultFacesErr = err

			return
		}

		defaultFaces = faces
	})

	return defaultFaces, defaultFacesErr
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

	switch {
	case bold && italic && fs.BoldItalic != nil:
		return fs.BoldItalic
	case bold && fs.Bold != nil:
		return fs.Bold
	case italic && fs.Italic != nil:
		return fs.Italic
	case fs.Regular != nil:
		return fs.Regular
	case fs.Bold != nil:
		return fs.Bold
	default:
		return fs.Italic
	}
}

// DefaultFont returns the embedded Liberation Sans regular face.
func DefaultFont() (*Font, error) {
	fs, err := LoadDefaultFaces()
	if err != nil {
		return nil, err
	}

	return fs.Regular, nil
}
