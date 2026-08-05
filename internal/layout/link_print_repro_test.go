package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// TestWikiPrintLinkUnderlineNotBlue: Wikipedia @media print sets
// color:inherit !important on anchors. Links must stay body-colored (black)
// and only gain underlines for discoverability — never a forced blue.
func TestWikiPrintLinkUnderlineNotBlue(t *testing.T) {
	s := sheet(t, `
body { color: #000000; }
@media print {
  a, a.external, a.new, a.stub { color: inherit !important; text-decoration: inherit !important }
}
a { text-decoration: none; color: #36c }
`)
	root, err := html.Parse(`<html><body><p>Hello <a href="/wiki/Cuba">Cuba</a> world</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	res, err := Layout(root, Options{
		Width: 500, Height: 800, Sheets: []*css.Stylesheet{s},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	foundText, foundLine := false, false
	for _, op := range res.Ops {
		if op.Kind == OpText {
			t.Logf("text %q rgb=(%.3f,%.3f,%.3f)", op.Text, op.R, op.G, op.B)
			if strings.Contains(op.Text, "Cuba") {
				foundText = true
				// Near black — not Vector blue (#0645ad / #36c).
				if op.R > 0.15 || op.G > 0.15 || op.B > 0.35 {
					t.Errorf("Cuba link rgb=(%.2f,%.2f,%.2f), want black (no forced blue)", op.R, op.G, op.B)
				}
			}
		}
		if op.Kind == OpLine {
			foundLine = true
		}
	}
	if !foundText {
		t.Fatalf("missing Cuba text; %d ops", len(res.Ops))
	}
	if !foundLine {
		t.Fatal("expected underline OpLine for print links")
	}
}

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
			if op.B > 0.4 && op.R < 0.3 {
				t.Errorf("unexpected blue link color (%v,%v,%v)", op.R, op.G, op.B)
			}
		}
		if op.Kind == OpLine {
			foundLine = true
		}
	}
	if !foundLine {
		t.Error("expected underline OpLine for print links")
	}
	if st.TextDecoration != "underline" {
		t.Errorf("decor=%q, want underline", st.TextDecoration)
	}
}
