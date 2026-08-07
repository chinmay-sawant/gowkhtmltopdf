package settings

import (
	"fmt"
	"strings"
)

// pageSizes maps ISO/ANSI/other named page sizes to points (1 pt = 1/72 in),
// matching Qt QPrinter / wkhtmltopdf page size names.
var pageSizes = map[string][2]float64{
	"A0":        {2383.94, 3370.39},
	"A1":        {1683.78, 2383.94},
	"A2":        {1190.55, 1683.78},
	"A3":        {841.89, 1190.55},
	"A4":        {595.28, 841.89},
	"A5":        {419.53, 595.28},
	"A6":        {297.64, 419.53},
	"B0":        {2834.65, 4008.19},
	"B1":        {2004.09, 2834.65},
	"B2":        {1417.32, 2004.09},
	"B3":        {1000.63, 1417.32},
	"B4":        {708.66, 1000.63},
	"B5":        {498.90, 708.66},
	"B6":        {354.33, 498.90},
	"C5E":       {459.21, 649.13},
	"Comm10E":   {297.00, 684.00},
	"DLE":       {311.81, 623.62},
	"Executive": {521.86, 756.00},
	"Folio":     {612.00, 936.00},
	"Ledger":    {1224.00, 792.00},
	"Legal":     {612.00, 1008.00},
	"Letter":    {612.00, 792.00},
	"Tabloid":   {792.00, 1224.00},
}

// ParsePageSize resolves a page size name to width/height in points
// (portrait orientation; caller swaps for landscape). "Custom" and
// "A4" etc. are case-insensitive.
func ParsePageSize(name string) (w, h float64, err error) {
	if name == "" {
		name = "A4"
	}

	key := strings.ToLower(name)
	if sz, ok := pageSizes[canonical(key)]; ok {
		return sz[0], sz[1], nil
	}

	return 0, 0, fmt.Errorf("unknown page size %q", name)
}

func canonical(key string) string {
	// Qt names used by wkhtmltopdf are mixed-case ("Comm10E", "C5E").
	switch key {
	case "comm10e":
		return "Comm10E"
	case "c5e":
		return "C5E"
	case "dle":
		return "DLE"
	case "letter":
		return "Letter"
	case "legal":
		return "Legal"
	case "ledger":
		return "Ledger"
	case "tabloid":
		return "Tabloid"
	case "executive":
		return "Executive"
	case "folio":
		return "Folio"
	case "a0", "a1", "a2", "a3", "a4", "a5", "a6":
		return strings.ToUpper(key)
	case "b0", "b1", "b2", "b3", "b4", "b5", "b6":
		return strings.ToUpper(key)
	}

	if len(key) == 2 && (key[0] == 'a' || key[0] == 'b' || key[0] == 'c') {
		return strings.ToUpper(key)
	}

	return key
}
