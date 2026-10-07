package css

import icss "github.com/chinmay-sawant/gowkhtmltopdf/internal/css"

// sameSheets reports whether two collections hold the same parsed sheets in
// the same order, which lets a relayout keep the font registry it already
// merged. Pointer identity is the right test: the sheet cache returns the
// same pointers for sources it has already parsed.
func sameSheets(left, right []*icss.Stylesheet) bool {
	if len(left) != len(right) {
		return false
	}

	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}

	return true
}
