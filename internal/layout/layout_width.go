package layout

import "github.com/chinmay-sawant/gowkhtmltopdf/internal/html"

// resolveDefiniteWidth applies the width/width% to *w. Returns false when the
// width resolves to auto (cyclic % honesty: indefinite containing block).
func resolveDefiniteWidth(
	eng *engine, node *html.Node, style *ResolvedStyle, availW float64, width *float64,
) (bool, bool) {
	if isIntrinsicWidth(style.Width) {
		*width = eng.flexIntrinsicWidth(node, *style, style.Width == widthMinContent)

		return true, true
	}

	definiteW := style.Width >= 0 || style.WidthPercent >= 0

	switch {
	case style.WidthPercent >= 0:
		// Cyclic % honesty: indefinite containing block → treat as auto.
		if availW > 0 && availW < 1e12 {
			*width = availW * style.WidthPercent / oneHundred
		} else {
			definiteW = false
		}
	case style.Width >= 0:
		*width = eng.scalePt(style.Width)
	}

	return definiteW, false
}
