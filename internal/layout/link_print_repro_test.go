package layout

import (
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestWikiPrintLinkDecorationRepro(t *testing.T) {
	s := sheet(t, `
@media print {
  a, a.external, a.new, a.stub { color: inherit !important; text-decoration: inherit !important }
}
a { text-decoration: none; color: #36c }
a:visited { color: #6a60b0 }
`)
	root, err := html.Parse(`<html><body><p>Hello <a href="/wiki/Cuba">Cuba</a> world</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	styles := resolveStyles(root, []*css.Stylesheet{s}, "print", 500, 800)
	var a *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "a" {
			a = n
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	st := styles[a]
	t.Logf("color=(%v,%v,%v) decor=%q", st.Color[0], st.Color[1], st.Color[2], st.TextDecoration)
	res, err := Layout(root, Options{
		Width: 500, Height: 800, Sheets: []*css.Stylesheet{s}, Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	foundLine := false
	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text == "Cuba" {
			t.Logf("Cuba text rgb=(%v,%v,%v)", op.R, op.G, op.B)
			if op.B < 0.5 {
				t.Errorf("print inherit links should be blue-ish, got (%v,%v,%v)", op.R, op.G, op.B)
			}
		}
		if op.Kind == OpLine {
			foundLine = true
		}
	}
	// Chrome print PDF underlines links so they remain visible as links.
	if !foundLine {
		t.Error("expected underline OpLine for print links (Chrome-visible links)")
	}
	if st.TextDecoration != "underline" {
		t.Errorf("decor=%q, want underline", st.TextDecoration)
	}
}
