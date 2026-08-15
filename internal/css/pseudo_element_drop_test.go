package css_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// TestPseudoElementSelectorDoesNotApplyToHost: p::before { width:120pt } must
// not match the host <p> (that crushed wiki body columns to 120pt when
// ::before was stripped). Pseudo rules are kept for generated content.
func TestPseudoElementSelectorDoesNotApplyToHost(t *testing.T) { //nolint:cyclop // host-vs-pseudo match bookkeeping
	t.Parallel()

	str, err := css.Parse(`p::before { width: 120pt; content: ''; display: block }
p:before { width: 99pt }
p { color: #000 }`)
	if err != nil {
		t.Fatal(err)
	}

	root, err := html.Parse(`<html><body><p>Hi</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	var page *html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "p" {
			page = n
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)

	if page == nil {
		t.Fatal("no p")
	}

	hostMatches := 0
	pseudoMatches := 0

	for _, r := range str.Rules {
		for _, sel := range r.Selectors {
			if css.Match(sel, page) {
				hostMatches++

				for _, d := range r.Decls {
					if d.Prop == "width" {
						t.Fatalf("pseudo width matched host via Match: %v", sel)
					}
				}
			}

			if css.MatchPseudo(sel, page, "before") {
				pseudoMatches++
			}
		}
	}

	if hostMatches != 1 {
		t.Fatalf("host matches=%d, want 1 (p{color})", hostMatches)
	}

	if pseudoMatches < 1 {
		t.Fatalf("pseudo matches=%d, want ≥1 for ::before/ :before", pseudoMatches)
	}
}
