package layout

import (
	"math"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// TestCSSVarFontSizeMedium: Vector-style custom properties resolve through
// var() chains (0.875rem → 10.5pt), and Georgia stays in the family list
// (paint aliases it to Liberation).
func TestCSSVarFontSizeMedium(t *testing.T) {
	s := sheet(t, `
:root {
  --font-size-small: 0.875rem;
  --font-size-medium: var(--font-size-small);
  --line-height-content: 1.6;
}
.vector-body {
  font-size: var(--font-size-medium);
  line-height: var(--line-height-content);
  font-family: Georgia, "Liberation Serif", serif;
}
`)
	root, err := html.Parse(`<html><body class="vector-body"><p id="t">Hello</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	styles := resolveStyles(root, []*css.Stylesheet{s}, "print", 800, 600)
	p := findEl(root, "p")
	if p == nil {
		t.Fatal("p not found")
	}
	st := styles[p]
	want := 10.5 // 0.875rem × 12pt root
	if math.Abs(st.FontSize-want) > 0.05 {
		t.Fatalf("font-size=%.2f want ~%.1f (custom prop chain)", st.FontSize, want)
	}
	if len(st.FontFamily) == 0 || st.FontFamily[0] != "Georgia" {
		t.Fatalf("font-family=%v want Georgia first (aliased to Liberation at paint)", st.FontFamily)
	}
}

// TestCSSVarFontSizeMediumDefault8: bare var(--font-size-medium) without a
// stylesheet definition falls back to 8pt (print density target).
func TestCSSVarFontSizeMediumDefault8(t *testing.T) {
	s := sheet(t, `
.vector-body { font-size: var(--font-size-medium); }
`)
	root, err := html.Parse(`<html><body class="vector-body"><p>Hello</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}
	styles := resolveStyles(root, []*css.Stylesheet{s}, "print", 800, 600)
	p := findEl(root, "p")
	if p == nil {
		t.Fatal("p not found")
	}
	st := styles[p]
	if math.Abs(st.FontSize-8) > 0.05 {
		t.Fatalf("font-size=%.2f want 8pt default for unresolved --font-size-medium", st.FontSize)
	}
}

func findEl(n *html.Node, name string) *html.Node {
	if n.Type == html.ElementNode && n.Name == name {
		return n
	}
	for _, c := range n.Children {
		if f := findEl(c, name); f != nil {
			return f
		}
	}
	return nil
}
