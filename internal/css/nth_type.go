package css

import (
	"strings"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func isNthArgPseudo(name string) bool {
	switch name {
	case nthChildPseudo, nthOfTypePseudo, nthLastOfTypePseudo:
		return true
	default:
		return false
	}
}

// matchOfTypePseudo handles :first-of-type, :last-of-type, :nth-of-type(),
// and :nth-last-of-type(). Index is 1-based among element siblings with the
// same tag (HTML tag names compare case-insensitively).
func matchOfTypePseudo(pseudo PseudoClass, node *html.Node) bool {
	switch pseudo.Name {
	case firstOfTypePseudo:
		return matchNth(nthForm{kind: nthInt, a: 1}, ofTypeIndex(node))
	case lastOfTypePseudo:
		return matchNth(nthForm{kind: nthInt, a: 1}, ofTypeLastIndex(node))
	case nthOfTypePseudo:
		return matchNth(pseudo.nth, ofTypeIndex(node))
	case nthLastOfTypePseudo:
		return matchNth(pseudo.nth, ofTypeLastIndex(node))
	default:
		return false
	}
}

// ofTypeIndex is 1-based among element siblings with the same tag.
func ofTypeIndex(node *html.Node) int {
	if node == nil || node.Type != html.ElementNode {
		return 0
	}

	if node.Parent == nil {
		return 1
	}

	idx := 0

	for _, cur := range node.Parent.Children {
		if !sameTypeElement(cur, node) {
			continue
		}

		idx++
		if cur == node {
			return idx
		}
	}

	return 0
}

// ofTypeLastIndex is the reverse 1-based index among same-tag siblings.
func ofTypeLastIndex(node *html.Node) int {
	if node == nil || node.Type != html.ElementNode {
		return 0
	}

	if node.Parent == nil {
		return 1
	}

	total := 0

	for _, cur := range node.Parent.Children {
		if sameTypeElement(cur, node) {
			total++
		}
	}

	index := ofTypeIndex(node)
	if index == 0 {
		return 0
	}

	return total - index + 1
}

func sameTypeElement(cur, node *html.Node) bool {
	return cur.Type == html.ElementNode && strings.EqualFold(cur.Name, node.Name)
}
