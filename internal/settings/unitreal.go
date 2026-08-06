package settings

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// UnitReal is a scalar with an optional unit suffix, mirroring wkhtmltopdf's
// UnitReal (pdfsettings.cc). Values without a suffix are interpreted in the
// unit given to ParseUnitReal.
type UnitReal struct {
	Value float64
	Unit  string // "" | mm | cm | m | in | pt | px | em | rem | ex | ch | %
}

// ErrInvalidUnitReal is returned by ParseUnitReal for unparseable input.
var ErrInvalidUnitReal = errors.New("invalid unit real")

// ParseUnitReal parses a number with an optional unit suffix, e.g. "10mm",
// "1.5in", "12pt", "100%". A bare number takes the implied unit.
func ParseUnitReal(s string, impliedUnit string) (UnitReal, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return UnitReal{}, fmt.Errorf("%w: empty", ErrInvalidUnitReal)
	}
	unit := impliedUnit
	for _, u := range []string{"rem", "em", "ex", "ch", "mm", "cm", "in", "pt", "px", "m", "%"} {
		if strings.HasSuffix(s, u) {
			unit = u
			s = s[:len(s)-len(u)]
			break
		}
	}
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return UnitReal{}, fmt.Errorf("%w: %q", ErrInvalidUnitReal, s)
	}
	return UnitReal{Value: v, Unit: unit}, nil
}

// Points converts to PDF points (1/72 inch) using the CSS reference ratio of
// 96 px per inch. % is not convertible and returns 0 with ok=false.
func (u UnitReal) Points() (float64, bool) {
	var perInch float64
	switch u.Unit {
	case "mm":
		perInch = 25.4
	case "cm":
		perInch = 2.54
	case "m":
		perInch = 0.0254
	case "in":
		perInch = 1
	case "pt":
		return u.Value, true
	case "px":
		return u.Value * 72 / 96, true
	case "em", "rem", "ex", "ch":
		return 0, false // font-relative; resolved by layout
	case "%":
		return 0, false
	}
	if perInch == 0 {
		return 0, false
	}
	return u.Value / perInch * 72, true
}

// Mm returns the value converted to millimetres (1 mm = 72/25.4 pt).
func (u UnitReal) Mm() (float64, bool) {
	pt, ok := u.Points()
	if !ok {
		return 0, false
	}
	return pt * 25.4 / 72, true
}
