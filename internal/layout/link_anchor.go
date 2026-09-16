package layout

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// anchorHref returns an <a> element's link target when it can become a PDF URI
// or fragment link. It is the single href filter shared by inline anchor
// flattening and enclosing-anchor lookups, so an anchor that author CSS gives
// display:block / display:inline-block (or that is a flex/grid item) keeps the
// same target set as an inline one.
func anchorHref(node *html.Node) string {
	if node == nil || node.Type != html.ElementNode || node.Name != cssTagA {
		return ""
	}

	href := strings.TrimSpace(node.Attribute("href"))
	if !isLinkHref(href) {
		return ""
	}

	return href
}

// enclosingAnchorHref returns the nearest ancestor anchor's link target.
// Inline anchors are flattened by collectInlineSpan, which stamps href on the
// items it collects; anchors whose box is built (display:block or
// inline-block, flex/grid item, float) never reach that path, so their
// descendants recover the target from the DOM parent chain instead. The walk
// stops at the first valid href, so a nested anchor wins over its ancestor.
func enclosingAnchorHref(node *html.Node) string {
	if node == nil {
		return ""
	}

	for parent := node.Parent; parent != nil; parent = parent.Parent {
		if href := anchorHref(parent); href != "" {
			return href
		}
	}

	return ""
}
