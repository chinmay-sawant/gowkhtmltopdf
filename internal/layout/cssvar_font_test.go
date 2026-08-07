package layout

import (
	"math"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// TestCSSVarFontSizeMedium: Vector-style custom properties resolve through
// var() chains (0.875rem → 10.5pt) when defined; Georgia stays in the family
// list (paint tries the author's stack order).
func TestCSSVarFontSizeMedium(t *testing.T) {
	t.Parallel()
	cssSheet := sheet(t, `
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

	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", 800, 600)

	p := findEl(root, "p")
	if p == nil {
		t.Fatal("p not found")
	}

	sty := styles[p]
	want := 10.5 // 0.875rem × 12pt root

	if math.Abs(sty.FontSize-want) > 0.05 {
		t.Fatalf("font-size=%.2f want ~%.1f (custom prop chain)", sty.FontSize, want)
	}

	if len(sty.FontFamily) == 0 || sty.FontFamily[0] != "Georgia" {
		t.Fatalf("font-family=%v want Georgia first", sty.FontFamily)
	}
}

// TestCSSVarFontSizeMediumUnresolved: bare var(--font-size-medium) without a
// definition or CSS fallback does not invent 8pt — font-size stays UA/inherited.
func TestCSSVarFontSizeMediumUnresolved(t *testing.T) {
	t.Parallel()
	cssSheet := sheet(t, `
.vector-body { font-size: var(--font-size-medium); }
`)

	root, err := html.Parse(`<html><body class="vector-body"><p>Hello</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", 800, 600)

	p := findEl(root, "p")
	if p == nil {
		t.Fatal("p not found")
	}

	st := styles[p]
	// Unresolved var → empty → fontSize("", parent) keeps inherited/UA 12pt.
	if math.Abs(st.FontSize-12) > 0.05 {
		t.Fatalf("font-size=%.2f want UA 12pt for unresolved var (no invented 8pt)", st.FontSize)
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
