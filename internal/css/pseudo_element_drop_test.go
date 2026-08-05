package css

import (
	"testing"

	"gowkhtmltopdf/internal/html"
)

// TestPseudoElementSelectorDoesNotApplyToHost: stripping ::before from
// selectors used to leave the host (`p`) matching Vector print's
// `p::before { width: 120pt }`, crushing wiki body columns to 120pt.
func TestPseudoElementSelectorDoesNotApplyToHost(t *testing.T) {
	s, err := Parse(`p::before { width: 120pt; content: ''; display: block }
p:before { width: 99pt }
p { color: #000 }`)
	if err != nil {
		t.Fatal(err)
	}
	root, err := html.Parse(`<html><body><p>Hi</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	var p *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "p" {
			p = n
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	nRules := 0
	for _, r := range s.Rules {
		for _, sel := range r.Selectors {
			nRules++
			t.Logf("selector=%v decls=%v", sel, r.Decls)
			for _, d := range r.Decls {
				if d.Prop == "width" {
					t.Fatalf("pseudo-element width rule leaked as host selector %v", sel)
				}
			}
		}
	}
	if nRules != 1 {
		t.Fatalf("expected only p{color} rule to survive, got %d selectors", nRules)
	}
	_ = p
}
