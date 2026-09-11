package layout

import (
	"math"
	"unsafe"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

// advanceKey is the per-Layout glyph-advance cache key: face identity, size
// bit pattern, and rune. Header cells repeat "Line"/"SKU" at one size; body
// strings are unique so the cache is per glyph, not per row.
type advanceKey struct {
	face uintptr
	size uint64
	r    rune
}

func (e *engine) glyphAdvance(face *pdf.Font, size float64, runic rune) float64 {
	if face == nil {
		return 0
	}

	key := advanceKey{
		face: uintptr(unsafe.Pointer(face)),
		size: math.Float64bits(size),
		r:    runic,
	}

	if e.advanceCache != nil {
		if width, ok := e.advanceCache[key]; ok {
			e.advanceHits++

			return width
		}
	} else {
		e.advanceCache = make(map[advanceKey]float64)
	}

	width := face.AdvanceInPoints(runic, size)
	e.advanceCache[key] = width

	return width
}
