package convert

import (
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// applyPageMarginBoxes maps unnamed @page margin-box strings onto the existing
// CLI header/footer chrome. Occupied slots and HTMLURL headers win.
func applyPageMarginBoxes(
	header, footer settings.HeaderFooter, boxes css.PageMarginBoxes,
) (settings.HeaderFooter, settings.HeaderFooter) {
	if header.HTMLURL == "" {
		header.Left = firstNonEmpty(header.Left, boxes.TopLeft)
		header.Center = firstNonEmpty(header.Center, boxes.TopCenter)
		header.Right = firstNonEmpty(header.Right, boxes.TopRight)
	}

	if footer.HTMLURL == "" {
		footer.Left = firstNonEmpty(footer.Left, boxes.BottomLeft)
		footer.Center = firstNonEmpty(footer.Center, boxes.BottomCenter)
		footer.Right = firstNonEmpty(footer.Right, boxes.BottomRight)
	}

	return header, footer
}

func mergePageMarginBoxes(dst, src css.PageMarginBoxes) css.PageMarginBoxes {
	dst.TopLeft = firstNonEmpty(src.TopLeft, dst.TopLeft)
	dst.TopCenter = firstNonEmpty(src.TopCenter, dst.TopCenter)
	dst.TopRight = firstNonEmpty(src.TopRight, dst.TopRight)
	dst.BottomLeft = firstNonEmpty(src.BottomLeft, dst.BottomLeft)
	dst.BottomCenter = firstNonEmpty(src.BottomCenter, dst.BottomCenter)
	dst.BottomRight = firstNonEmpty(src.BottomRight, dst.BottomRight)

	return dst
}

func firstNonEmpty(primary, fallback string) string {
	if primary != "" {
		return primary
	}

	return fallback
}
