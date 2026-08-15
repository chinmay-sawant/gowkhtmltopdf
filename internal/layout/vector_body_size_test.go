//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

// TestVectorBodyPrintSize: clientpref-1 circular token falls back to 1rem (12pt).
func TestVectorBodyPrintSize(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
html.vector-feature-custom-font-size-clientpref-1 {
  --font-size-medium: var(--font-size-medium, 1rem);
}
.vector-body {
  font-size: var(--font-size-medium);
  font-family: Georgia, serif;
}
p { margin: 0; }
`)

	root, err := html.Parse(`<html class="vector-feature-custom-font-size-clientpref-1">` +
		`<body><div class="vector-body"><p>Hello world article text</p></div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", 400, 400)

	p := findEl(root, "p")
	if p == nil {
		t.Fatal("no p")
	}

	sty := styles[p]
	if sty.FontSize < 11.5 || sty.FontSize > 12.5 {
		t.Fatalf("want ~12pt (1rem fallback), got %.2f", sty.FontSize)
	}

	if len(sty.FontFamily) == 0 || sty.FontFamily[0] != "Georgia" {
		t.Fatalf("family=%v", sty.FontFamily)
	}
}

// TestPrintZoomDensifies12ptTo8: author CSS p{font-size:12pt} + operator
// --zoom 2/3 yields ~8pt paint size (recipe policy, not cascade invention).
func TestPrintZoomDensifies12ptTo8(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
p { font-size: 12pt; margin: 0; font-family: Georgia, serif; font-weight: normal; }
`)

	root, err := html.Parse(`<html><body><p>Hello world article text beside the frame</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	const zoom = 8.0 / 12.0

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 400, Height: 200, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Zoom: zoom,
	})
	if err != nil {
		t.Fatal(err)
	}

	var size float64

	for _, op := range res.Ops {
		if op.Kind == OpText && op.Text != "" {
			size = op.Size

			break
		}
	}

	if size < 7.5 || size > 8.5 {
		t.Fatalf("painted size=%.2f want ~8pt with zoom 2/3", size)
	}
}
