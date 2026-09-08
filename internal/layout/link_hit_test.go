//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
)

func TestLinkHrefClassification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		href string
		want bool
	}{
		{href: "docs/item.html", want: true},
		{href: "../item.html", want: true},
		{href: "?page=2", want: true},
		{href: "//cdn.example/item", want: true},
		{href: "#section", want: true},
		{href: "https://example.test/item", want: true},
		{href: "mailto:person@example.test", want: true},
		{href: "javascript:void(0)", want: false},
		{href: "data:text/plain,hello", want: false},
		{href: "ftp://example.test/item", want: false},
		{href: "", want: false},
	}

	for _, test := range tests {
		t.Run(test.href, func(t *testing.T) {
			t.Parallel()

			if got := isLinkHref(test.href); got != test.want {
				t.Errorf("isLinkHref(%q) = %v, want %v", test.href, got, test.want)
			}
		})
	}
}

func TestRelativeAnchorEmitsURIAndUnsupportedSchemesDoNot(t *testing.T) {
	t.Parallel()

	res := layoutHTML(t, `<html><body>`+
		`<a href="docs/item.html">relative</a>`+
		`<a href="javascript:void(0)">script</a>`+
		`<a href="ftp://example.test/item">ftp</a>`+
		`</body></html>`)

	links := opsOfKind(res, OpLinkURI)
	if len(links) != 1 || links[0].URI != "docs/item.html" {
		t.Fatalf("links = %+v, want one raw relative URI", links)
	}
}

// TestLinkAnnotationHasHitHeight: URI link ops must cover the glyph box so
// PDF viewers give a usable hover/click target (not a zero-height line).
func TestLinkAnnotationHasHitHeight(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
a { color: inherit; text-decoration: underline; }
`)

	root, err := html.Parse(`<html><body><p>See ` +
		`<a href="https://example.com/academy">Academy Award</a> here.</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 400, Height: 200, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var linkH, textSize float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpLinkURI && paintOp.URI != "" {
			if paintOp.H > linkH {
				linkH = paintOp.H
			}
		}

		if paintOp.Kind == OpText && paintOp.Size > textSize {
			textSize = paintOp.Size
		}
	}

	if linkH < textSize*0.5 {
		t.Fatalf("link H=%.2f too small for font size %.2f (zero-height hit target)", linkH, textSize)
	}

	t.Logf("linkH=%.2f fontSize=%.2f", linkH, textSize)
}

// TestUnderlineSitsBelowDescenders: underline Y must be below the text baseline.
func TestUnderlineSitsBelowDescenders(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `a { text-decoration: underline; font-size: 14pt; }`)

	root, err := html.Parse(`<html><body><a href="https://example.com">gyp</a></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 200, Height: 100, Sheets: []*css.Stylesheet{cssSheet}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	var baseline, underY float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpText {
			baseline = paintOp.Y
		}

		if paintOp.Kind == OpLine && paintOp.W > 0 && paintOp.H == 0 {
			underY = paintOp.Y
		}
	}

	if underY <= baseline {
		t.Fatalf("underline y=%.2f should be below baseline %.2f", underY, baseline)
	}

	gap := underY - baseline
	if gap < 1.5 {
		t.Fatalf("underline gap %.2fpt too tight (want >= 1.5pt below baseline)", gap)
	}

	t.Logf("baseline=%.2f underline=%.2f gap=%.2f", baseline, underY, gap)
}
