package html

import (
	"fmt"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pubstate"
)

func registerRoot() {
	pubstate.RegisterRoot(func(doc any) *html.Node {
		parsed, ok := doc.(*Document)
		if !ok || parsed == nil {
			return nil
		}

		return parsed.root
	})
}

// Document is one parsed HTML tree.
// The CSS package reads it. Callers do not see the engine node.
type Document struct {
	root *html.Node
}

// Element is one element found by id.
type Element struct {
	Tag  string
	ID   string
	Text string
}

// Parse parses UTF-8 HTML into a document the CSS package can style.
// A nil source is an empty document.
func Parse(source []byte) (*Document, error) {
	registerRoot()

	root, err := html.ParseDocument(source)
	if err != nil {
		return nil, fmt.Errorf("html: parse: %w", err)
	}

	if root == nil {
		return nil, errEmpty
	}

	return &Document{root: root}, nil
}

// Find returns the element with the given id.
// The text is the element's descendant text.
func (d *Document) Find(elementID string) (Element, bool) {
	if d == nil || d.root == nil || elementID == "" {
		return Element{}, false //nolint:exhaustruct // no element was found
	}

	node := d.root.FindFirst(func(candidate *html.Node) bool {
		return candidate.Type == html.ElementNode && candidate.Attribute("id") == elementID
	})
	if node == nil {
		return Element{}, false //nolint:exhaustruct // no element was found
	}

	return Element{
		Tag:  node.Name,
		ID:   elementID,
		Text: node.TextContent(),
	}, true
}
