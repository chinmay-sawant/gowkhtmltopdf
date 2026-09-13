package settings

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const (
	// CSS/PDF unit conversion constants.
	pointsPerInch    = 72
	cssPixelsPerInch = 96
	mmPerInch        = 25.4
)

// UnitReal is a scalar with an optional unit suffix, mirroring wkhtmltopdf's
// UnitReal (pdfsettings.cc). Values without a suffix are interpreted in the
// unit given to ParseUnitReal.
type UnitReal struct {
	Value float64
	Unit  string // "" | mm | cm | m | in | pt | px | em | rem | ex | ch | %
}

// ErrInvalidUnitReal is returned by ParseUnitReal for unparseable input.
var ErrInvalidUnitReal = errors.New("settings: invalid unit real")

// ParseUnitReal parses a number with an optional unit suffix, e.g. "10mm",
// "1.5in", "12pt", "100%". A bare number takes the implied unit.
func ParseUnitReal(raw string, impliedUnit string) (UnitReal, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return UnitReal{}, fmt.Errorf("%w: empty", ErrInvalidUnitReal)
	}

	unit := impliedUnit

	// Match the suffix case-insensitively (like the sibling parsers) but slice
	// the original: the unit constants are ASCII, so byte offsets are stable.
	lower := strings.ToLower(raw)

	for _, u := range []string{"rem", "em", "ex", "ch", "mm", "cm", "in", "pt", "px", "m", "%"} {
		if strings.HasSuffix(lower, u) {
			unit = u
			raw = raw[:len(raw)-len(u)]

			break
		}
	}

	v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
	if err != nil || !finite(v) {
		return UnitReal{}, fmt.Errorf("%w: %q", ErrInvalidUnitReal, raw)
	}

	return UnitReal{Value: v, Unit: unit}, nil
}

// finite reports whether value is neither NaN nor an infinity. It mirrors the
// root API predicate (document_validate.go finitePositive) so CLI and library
// reject the same numeric inputs.
func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
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
		return u.Value * pointsPerInch / cssPixelsPerInch, true
	case "em", "rem", "ex", "ch":
		return 0, false // font-relative; resolved by layout
	case "%":
		return 0, false
	}

	if perInch == 0 {
		return 0, false
	}

	return u.Value / perInch * pointsPerInch, true
}

// Mm returns the value converted to millimetres (1 mm = 72/25.4 pt).
func (u UnitReal) Mm() (float64, bool) {
	pt, ok := u.Points()
	if !ok {
		return 0, false
	}

	return pt * mmPerInch / pointsPerInch, true
}
