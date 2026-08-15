//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// TestPrintLinkInheritHonorsCascade: text-decoration:inherit on links must
// not invent underlines (CSS-faithful default).
func TestPrintLinkInheritHonorsCascade(t *testing.T) { //nolint:cyclop
	t.Parallel()

	cssSheet := sheet(t, `
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

	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", 500, 800)

	var acc *html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "a" {
			acc = n
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)

	if acc == nil {
		t.Fatal("no anchor")
	}

	st := styles[acc]
	if st.TextDecoration == "underline" {
		t.Fatalf("decoration=%q: inherit must not force underline without --print-link-underline", st.TextDecoration)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 500, Height: 800, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	// color:inherit still paints black; underlines remain for PDF link affordance
	// (emitLine underlines a[href] even when cascade decoration is none).
	for _, op := range res.Ops {
		if op.Kind == OpText && strings.Contains(op.Text, "Cuba") {
			if op.R > 0.15 || op.G > 0.15 || op.B > 0.35 {
				t.Errorf("Cuba link rgb=(%.2f,%.2f,%.2f), want black (color:inherit)", op.R, op.G, op.B)
			}
		}
	}
}

// TestPrintLinkUnderlineOptIn: --print-link-underline forces underlines after cascade.
func TestPrintLinkUnderlineOptIn(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { color: #000000; }
@media print {
  a { color: inherit !important; text-decoration: inherit !important }
}
a { text-decoration: none; color: #36c }
`)

	root, err := html.Parse(`<html><body><p>Hello <a href="/wiki/Cuba">Cuba</a> world</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 500, Height: 800, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Background: true, PrintLinkUnderline: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	foundLine := false

	for _, op := range res.Ops {
		if op.Kind == OpLine {
			foundLine = true
		}
	}

	if !foundLine {
		t.Fatal("expected underline OpLine with PrintLinkUnderline")
	}
}
