package layout //nolint:testpackage // benchmark exercises unexported layout and paint stages.

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/pdf"
)

//nolint:wsl // fixture assembly groups source fragments for readability.
func scalabilityFixture(tb testing.TB, count int) (*html.Node, *css.Stylesheet) {
	tb.Helper()

	source := "<html><head></head><body><main>"
	for i := range count {
		source += fmt.Sprintf(
			"<section class=\"scale-item\"><div class=\"sticky-box\">item %d</div><p>content</p></section>",
			i,
		)
	}
	source += "</main></body></html>"

	root, err := html.Parse(source)
	if err != nil {
		tb.Fatalf("html.Parse: %v", err)
	}

	sheet, err := css.Parse(`
body { margin: 0; }
.scale-item { page-break-before: always; transform: translate(1pt, 0); padding: 4pt; }
.scale-item:first-child { page-break-before: auto; }
.sticky-box { position: sticky; top: 0; height: 12pt; background: #eeeeee; }
`)
	if err != nil {
		tb.Fatalf("css.Parse: %v", err)
	}

	return root, sheet
}

//nolint:wsl // benchmark timing boundaries intentionally surround the loop.
func BenchmarkDeepChromeAndForcedBreaks(b *testing.B) {
	for _, count := range []int{10, 100, 1000} {
		b.Run(fmt.Sprintf("%d-items", count), func(b *testing.B) {
			root, sheet := scalabilityFixture(b, count)
			opts := Options{Width: 500, Height: 700, Sheets: []*css.Stylesheet{sheet}, Background: true} //nolint:exhaustruct
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				res, err := LayoutContext(b.Context(), root, opts)
				if err != nil {
					b.Fatal(err)
				}
				if res == nil || len(res.Ops) == 0 {
					b.Fatal("layout produced no operations")
				}
			}
		})
	}
}

//nolint:wsl // repeated render and byte assertions are intentionally adjacent.
func TestDeepChromeOutputStable(t *testing.T) {
	t.Parallel()
	for _, count := range []int{10, 100} {
		t.Run(fmt.Sprintf("%d-items", count), func(t *testing.T) {
			t.Parallel()
			root, sheet := scalabilityFixture(t, count)
			opts := Options{Width: 500, Height: 700, Sheets: []*css.Stylesheet{sheet}, Background: true} //nolint:exhaustruct
			first := renderScalabilityPDF(t, root, opts)
			second := renderScalabilityPDF(t, root, opts)
			if !bytes.Equal(first, second) {
				t.Fatalf("repeated render changed PDF bytes for %d items", count)
			}
		})
	}
}

func renderScalabilityPDF(t *testing.T, root *html.Node, opts Options) []byte {
	t.Helper()

	res, err := LayoutContext(t.Context(), root, opts)
	if err != nil {
		t.Fatalf("Layout: %v", err)
	}

	doc := pdf.NewDocument()
	if err := Paint(doc, res, PaintOptions{PageWidth: 500, PageHeight: 700}); err != nil { //nolint:exhaustruct
		t.Fatalf("Paint: %v", err)
	}

	var out bytes.Buffer
	if err := doc.Write(&out); err != nil {
		t.Fatalf("Write: %v", err)
	}

	return out.Bytes()
}
