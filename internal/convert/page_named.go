package convert

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/settings"
)

// applyPageMarginBoxes maps unnamed @page margin-box quoted strings onto the
// existing CLI header/footer chrome. Occupied Left/Center/Right slots and
// HTMLURL headers win. Named / :first / :left / :right boxes parse onto
// PageRule.Boxes but do not change per-page chrome (HF repeats every page).
func applyPageMarginBoxes(
	header, footer settings.HeaderFooter, sheets []*css.Stylesheet,
) (settings.HeaderFooter, settings.HeaderFooter) {
	boxes := collectUnnamedPageBoxes(sheets)
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

func collectUnnamedPageBoxes(sheets []*css.Stylesheet) css.PageMarginBoxes {
	var boxes css.PageMarginBoxes

	for _, sheet := range sheets {
		if sheet == nil {
			continue
		}

		for _, rule := range sheet.Pages {
			if strings.TrimSpace(rule.Sel) != "" {
				continue
			}

			boxes = mergePageMarginBoxes(boxes, rule.Boxes)
		}
	}

	return boxes
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
