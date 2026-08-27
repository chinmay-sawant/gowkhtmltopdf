//nolint:testpackage,cyclop,varnamelen // :is() cascade proofs
package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func TestIsPseudoStyle(t *testing.T) {
	t.Parallel()

	root := mustParse(t, `<html><body>
		<div>div</div>
		<p>p</p>
		<span>span</span>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{sheet(t, `
		:is(div, p) { color: #ff0000 }
	`)}, "print", testViewport, 800)

	var divN, pN, spanN *html.Node

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode {
			switch node.Name {
			case "div":
				divN = node
			case "p":
				pN = node
			case "span":
				spanN = node
			}
		}

		for _, c := range node.Children {
			walk(c)
		}
	}
	walk(root)

	if divN == nil || pN == nil || spanN == nil {
		t.Fatal("missing div, p, or span")
	}

	wantRed := [3]float64{1, 0, 0}

	if got := styles[divN].Color; got != wantRed {
		t.Fatalf("div via :is(div, p) Color = %v, want red", got)
	}

	if got := styles[pN].Color; got != wantRed {
		t.Fatalf("p via :is(div, p) Color = %v, want red", got)
	}

	if got := styles[spanN].Color; got == wantRed {
		t.Fatalf("span Color = %v, must not match :is(div, p)", got)
	}
}
