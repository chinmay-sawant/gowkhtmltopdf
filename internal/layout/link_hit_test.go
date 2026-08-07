package layout

import (
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// TestLinkAnnotationHasHitHeight: URI link ops must cover the glyph box so
// PDF viewers give a usable hover/click target (not a zero-height line).
func TestLinkAnnotationHasHitHeight(t *testing.T) {
	s := sheet(t, `
body { margin: 0; font-size: 12pt; }
a { color: inherit; text-decoration: underline; }
`)

	root, err := html.Parse(`<html><body><p>See <a href="https://example.com/academy">Academy Award</a> here.</p></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{
		Width: 400, Height: 200, Sheets: []*css.Stylesheet{s},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var linkH, textSize float64

	for _, op := range res.Ops {
		if op.Kind == OpLinkURI && op.URI != "" {
			if op.H > linkH {
				linkH = op.H
			}
		}

		if op.Kind == OpText && op.Size > textSize {
			textSize = op.Size
		}
	}

	if linkH < textSize*0.5 {
		t.Fatalf("link H=%.2f too small for font size %.2f (zero-height hit target)", linkH, textSize)
	}

	t.Logf("linkH=%.2f fontSize=%.2f", linkH, textSize)
}

// TestUnderlineSitsBelowDescenders: underline Y must be below the text baseline.
func TestUnderlineSitsBelowDescenders(t *testing.T) {
	s := sheet(t, `a { text-decoration: underline; font-size: 14pt; }`)

	root, err := html.Parse(`<html><body><a href="https://example.com">gyp</a></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{
		Width: 200, Height: 100, Sheets: []*css.Stylesheet{s}, Media: "print",
	})
	if err != nil {
		t.Fatal(err)
	}

	var baseline, underY float64

	for _, op := range res.Ops {
		if op.Kind == OpText {
			baseline = op.Y
		}

		if op.Kind == OpLine && op.W > 0 && op.H == 0 {
			underY = op.Y
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
