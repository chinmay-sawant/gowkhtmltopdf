package css //nolint:testpackage // exercises unexported parseSelector

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// :target must not degrade to the host selector. Wikipedia paints
// ol.references li:target with a progressive-subtle blue; if :target is
// stripped, every reference gets that background in print.
func TestTargetPseudoDoesNotMatchBareHost(t *testing.T) {
	t.Parallel()

	doc, err := html.Parse(`<html><body><ol class="references"><li id="cite_note-1">one</li></ol></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	var liVal *html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "li" {
			liVal = n
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(doc)

	if liVal == nil {
		t.Fatal("no li")
	}

	sel, found := parseSelector("ol.references li:target")
	if !found {
		t.Fatal("parseSelector failed")
	}

	if Match(sel, liVal) {
		t.Fatal("li:target matched in print; want no match (static PDF has no :target)")
	}

	bare, found := parseSelector("ol.references li")
	if !found {
		t.Fatal("parse bare")
	}

	if !Match(bare, liVal) {
		t.Fatal("bare ol.references li should still match")
	}
}
