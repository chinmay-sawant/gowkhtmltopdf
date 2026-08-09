package settings

import (
	"errors"
	"fmt"
	"strings"
)

// pageSizes maps ISO/ANSI/other named page sizes to points (1 pt = 1/72 in),
// matching Qt QPrinter / wkhtmltopdf page size names. Keys are lowercase;
// ParsePageSize lowercases its input before lookup. It is immutable after
// package initialization and reused by every parse operation.
var pageSizes = map[string][2]float64{
	"a0":        {2383.94, 3370.39},
	"a1":        {1683.78, 2383.94},
	"a2":        {1190.55, 1683.78},
	"a3":        {841.89, 1190.55},
	"a4":        {595.28, 841.89},
	"a5":        {419.53, 595.28},
	"a6":        {297.64, 419.53},
	"b0":        {2834.65, 4008.19},
	"b1":        {2004.09, 2834.65},
	"b2":        {1417.32, 2004.09},
	"b3":        {1000.63, 1417.32},
	"b4":        {708.66, 1000.63},
	"b5":        {498.90, 708.66},
	"b6":        {354.33, 498.90},
	"c5e":       {459.21, 649.13},
	"comm10e":   {297.00, 684.00},
	"dle":       {311.81, 623.62},
	"executive": {521.86, 756.00},
	"folio":     {612.00, 936.00},
	"ledger":    {1224.00, 792.00},
	"legal":     {612.00, 1008.00},
	"letter":    {612.00, 792.00},
	"tabloid":   {792.00, 1224.00},
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

	if sz, ok := pageSizes[strings.ToLower(name)]; ok {
		return sz[0], sz[1], nil
	}

	return 0, 0, fmt.Errorf("%w %q", errUnknownPageSize, name)
}
