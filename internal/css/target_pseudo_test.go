package css

import (
	"testing"

	"gowkhtmltopdf/internal/html"
)

// :target must not degrade to the host selector. Wikipedia paints
// ol.references li:target with a progressive-subtle blue; if :target is
// stripped, every reference gets that background in print.
func TestTargetPseudoDoesNotMatchBareHost(t *testing.T) {
	doc, err := html.Parse(`<html><body><ol class="references"><li id="cite_note-1">one</li></ol></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	var li *html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "li" {
			li = n
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(doc)

	if li == nil {
		t.Fatal("no li")
	}

	sel, ok := parseSelector("ol.references li:target")
	if !ok {
		t.Fatal("parseSelector failed")
	}

	if Match(sel, li) {
		t.Fatal("li:target matched in print; want no match (static PDF has no :target)")
	}

	bare, ok := parseSelector("ol.references li")
	if !ok {
		t.Fatal("parse bare")
	}

	if !Match(bare, li) {
		t.Fatal("bare ol.references li should still match")
	}
}
