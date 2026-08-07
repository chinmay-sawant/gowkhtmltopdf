package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// TestNowrapSpanWrapsBesideFloat: white-space:nowrap must not glue a long span
// onto a shortened line and paint over a right float (wiki .IPA beside infobox).
func TestNowrapSpanWrapsBesideFloat(t *testing.T) {
	cssSheet := sheet(t, `
body { margin: 0; font-size: 11pt; }
.infobox { float: right; clear: right; width: 160pt; background: #eee; }
.IPA { white-space: nowrap; }
p { margin: 0; text-align: left; }
`)
	htmlSrc := `<html><body>
<table class="infobox"><tr><td>Portrait photo box that must stay clear of lead text</td></tr>
<tr><td>more infobox rows to make it tall enough</td></tr>
<tr><td>row three</td></tr><tr><td>row four</td></tr><tr><td>row five</td></tr>
</table>
<p>Ana Celia de Armas Caso <span class="IPA">[ˈana ˈselja ðe ˈaɾmas ˈkaso]</span>
is a Cuban-born actress holding citizenship.</p>
</body></html>`

	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}

	const pageW = 500.0

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: pageW, Height: 700, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var floatLeft float64 = pageW

	for _, op := range res.Ops {
		if op.Kind == OpText && strings.HasPrefix(strings.TrimSpace(op.Text), "Portrait") {
			if op.X < floatLeft {
				floatLeft = op.X
			}
		}
	}

	if floatLeft > pageW-50 {
		t.Fatal("infobox not found on the right")
	}

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText {
			continue
		}
		// Only assert on lead-paragraph content, not infobox cells.
		lead := strings.Contains(paintOp.Text, "Ana") || strings.Contains(paintOp.Text, "Cuban") ||
			strings.Contains(paintOp.Text, "actress") || strings.Contains(paintOp.Text, "selja") ||
			strings.Contains(paintOp.Text, "aɾmas") || strings.Contains(paintOp.Text, "kaso") ||
			strings.Contains(paintOp.Text, "ˈana") || strings.Contains(paintOp.Text, "[")
		if !lead {
			continue
		}

		right := paintOp.X + paintOp.W
		if right > floatLeft+2 {
			t.Fatalf("lead text %q ends at x=%.1f, overlaps float starting ~%.1f",
				paintOp.Text, right, floatLeft)
		}
	}
}
