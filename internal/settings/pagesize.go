package settings

import (
	"errors"
	"fmt"
	"strings"
)

// pageSizeEntry stores one ISO/ANSI/other named page size in points (1 pt =
// 1/72 in), matching Qt QPrinter / wkhtmltopdf page size names. The fixed
// table avoids exposing a mutable map to package-local callers.
type pageSizeEntry struct {
	name   string
	width  float64
	height float64
}

//nolint:gochecknoglobals,mnd // static page size lookup table
var pageSizes = [...]pageSizeEntry{
	{name: "a0", width: 2383.94, height: 3370.39},
	{name: "a1", width: 1683.78, height: 2383.94},
	{name: "a2", width: 1190.55, height: 1683.78},
	{name: "a3", width: 841.89, height: 1190.55},
	{name: "a4", width: 595.28, height: 841.89},
	{name: "a5", width: 419.53, height: 595.28},
	{name: "a6", width: 297.64, height: 419.53},
	{name: "b0", width: 2834.65, height: 4008.19},
	{name: "b1", width: 2004.09, height: 2834.65},
	{name: "b2", width: 1417.32, height: 2004.09},
	{name: "b3", width: 1000.63, height: 1417.32},
	{name: "b4", width: 708.66, height: 1000.63},
	{name: "b5", width: 498.90, height: 708.66},
	{name: "b6", width: 354.33, height: 498.90},
	{name: "c5e", width: 459.21, height: 649.13},
	{name: "comm10e", width: 297.00, height: 684.00},
	{name: "dle", width: 311.81, height: 623.62},
	{name: "executive", width: 521.86, height: 756.00},
	{name: "folio", width: 612.00, height: 936.00},
	{name: "ledger", width: 1224.00, height: 792.00},
	{name: "legal", width: 612.00, height: 1008.00},
	{name: "letter", width: 612.00, height: 792.00},
	{name: "tabloid", width: 792.00, height: 1224.00},
}

// errUnknownPageSize is returned by ParsePageSize for unrecognized names.
var errUnknownPageSize = errors.New("unknown page size")

// ParsePageSize resolves a page size name to width/height in points
// (portrait orientation; caller swaps for landscape). "Custom" and
// "A4" etc. are case-insensitive.
func ParsePageSize(name string) (float64, float64, error) {
	if name == "" {
		name = "A4"
	}

	key := strings.ToLower(name)
	for _, sz := range pageSizes {
		if sz.name == key {
			return sz.width, sz.height, nil
		}
	}

	return 0, 0, fmt.Errorf("%w %q", errUnknownPageSize, name)
}
