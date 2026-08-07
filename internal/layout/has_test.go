package layout

import (
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestHasSelectorArticleBorder(t *testing.T) {
	s := sheet(t, `article:has(.footnote) { border-left: 3pt solid #00f }`)
	root := mustParse(t, `<html><body>
		<article id="with"><p>body</p><span class="footnote">fn</span></article>
		<article id="without"><p>body</p></article>
	</body></html>`)
	styles := resolveStyles(root, []*css.Stylesheet{s}, "print", testViewport, 800)

	var with, without *html.Node

	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Name == "article" {
			switch node.Attribute("id") {
			case "with":
				with = node
			case "without":
				without = node
			}
		}

		for _, c := range node.Children {
			walk(c)
		}
	}
	walk(root)

	if with == nil || without == nil {
		t.Fatal("missing articles")
	}

	stWith := styles[with]
	stWithout := styles[without]

	if stWith.BorderLeft.Width < 2.9 || stWith.BorderLeft.Width > 3.1 {
		t.Errorf("article:has(.footnote) border-left width = %v, want 3pt", stWith.BorderLeft.Width)
	}

	if stWith.BorderLeft.Color[2] < 0.9 {
		t.Errorf("article:has(.footnote) border color = %v, want blue", stWith.BorderLeft.Color)
	}

	if stWithout.BorderLeft.Width > 0.01 {
		t.Errorf("plain article should have no left border, got width %v", stWithout.BorderLeft.Width)
	}
}

func TestHasSelectorTableRowHighlight(t *testing.T) {
	s := sheet(t, `tr:has(td.neg) td { color: red }`)
	res := layoutHTML(t, `<html><body><table>
		<tr><td class="neg">-1</td><td>loss</td></tr>
		<tr><td>2</td><td>gain</td></tr>
	</table></body></html>`, s)

	texts := opsOfKind(res, OpText)
	if len(texts) < 4 {
		t.Fatalf("texts = %+v", texts)
	}
	// first row cells should be red; second row default black
	var red, black int

	for _, op := range texts {
		if op.R > 0.9 && op.G < 0.1 && op.B < 0.1 {
			red++
		}

		if op.R < 0.1 && op.G < 0.1 && op.B < 0.1 {
			black++
		}
	}

	if red < 2 {
		t.Errorf("expected >=2 red texts from tr:has(td.neg), got red=%d texts=%+v", red, texts)
	}

	if black < 2 {
		t.Errorf("expected >=2 black texts from plain row, got black=%d texts=%+v", black, texts)
	}
}
