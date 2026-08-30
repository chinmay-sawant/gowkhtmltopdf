package layout

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

const (
	unitCh    = "ch"
	digitZero = '0' // kept for style_ch_test.go; ch layout now uses 0.5em
)

// lengthToPt converts a CSS length to points. ch is 0.5em (same fallback
// already in css.LengthToPt); other units go through css.LengthToPt.
func lengthToPt(val float64, unit string, fsize float64) (float64, bool) {
	if strings.EqualFold(unit, unitCh) {
		if pt, ok := chLengthPt(val, fsize); ok {
			return pt, true
		}
	}

	return css.LengthToPt(val, unit, fsize)
}

func chLengthPt(val, fsize float64) (float64, bool) {
	return val * fsize * 0.5, true
}
