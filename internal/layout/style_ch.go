package layout

import (
	"strings"
	"sync"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

const (
	unitCh    = "ch"
	digitZero = '0' // U+0030 DIGIT ZERO; CSS ch is this glyph's advance
)

// lengthToPt converts a CSS length to points. ch uses the default Liberation
// face's U+0030 advance at fsize; other units go through css.LengthToPt
// (ex stays 0.5em). Falls back to 0.5em when the face or advance is missing.
func lengthToPt(val float64, unit string, fsize float64) (float64, bool) {
	if strings.EqualFold(unit, unitCh) {
		if pt, ok := chLengthPt(val, fsize); ok {
			return pt, true
		}
	}

	return css.LengthToPt(val, unit, fsize)
}

func chLengthPt(val, fsize float64) (float64, bool) {
	face := defaultChFace()
	if face == nil {
		return 0, false
	}

	adv := face.GlyphAdvancePoints(digitZero, fsize)
	if adv <= 0 {
		return 0, false
	}

	return val * adv, true
}

var (
	chFaceOnce sync.Once //nolint:gochecknoglobals // one immutable default-face cache
	chFace     *pdf.Font //nolint:gochecknoglobals // one immutable default-face cache
)

func defaultChFace() *pdf.Font {
	chFaceOnce.Do(func() {
		faces, err := pdf.LoadDefaultFaces()
		if err != nil || faces == nil {
			return
		}

		chFace = faces.Regular
	})

	return chFace
}
