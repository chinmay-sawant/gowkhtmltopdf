package layout

import (
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

// Vector print CSS uses:
//
//	p::before { content:''; display:block; width:120pt; overflow:hidden; page-break-after:avoid }
//
// Pseudo-element rules must not apply to the host. A prior tokenizer bug
// stripped ::before and left `p { width:120pt }`, squeezing every wiki
// paragraph to a 120pt column beside the infobox.
func TestVectorPrintPBeforeDoesNotSqueezeLines(t *testing.T) {
	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
.infobox { float: right; width: 22em; font-size: 88%; }
p { font-size: 12pt; line-height: 16pt; text-align: justify; margin: 0.5em 0; }
p::before { content: ''; display: block; width: 120pt; overflow: hidden; page-break-after: avoid; }
`)

	const contentW = 538.0

	htmlSrc := `<html><body>
<table class="infobox"><tr><td>Portrait photo here</td></tr>
<tr><th>Born</th><td>30 April 1988</td></tr></table>
<p>Ana Celia de Armas Caso is a Cuban-born actress holding Cuban Spanish and American citizenship in Hollywood films today and tomorrow.</p>
<p>She began her career in Cuba with a leading role in the romantic drama and later moved to Spain.</p>
</body></html>`

	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", contentW, 800)

	var page *html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "p" && page == nil {
			page = n
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)

	if st := styles[page]; st.Width >= 0 && st.Width < 200 {
		t.Fatalf("p width=%.1f leaked from p::before", st.Width)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: contentW, Height: 800, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	lines := map[int][2]float64{}

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || paintOp.X > contentW*0.55 {
			continue
		}

		yIdx := int(paintOp.Y + 0.5)

		line := lines[yIdx]
		if line[1] == 0 && line[0] == 0 {
			line = [2]float64{paintOp.X, paintOp.X + paintOp.W}
		} else {
			if paintOp.X < line[0] {
				line[0] = paintOp.X
			}

			if paintOp.X+paintOp.W > line[1] {
				line[1] = paintOp.X + paintOp.W
			}
		}

		lines[yIdx] = line
	}

	best := 0.0

	for _, ln := range lines {
		w := ln[1] - ln[0]
		if w > best {
			best = w
		}
	}

	t.Logf("best line extent=%.0f", best)

	if best < 250 {
		t.Fatalf("p::before squeezed lines to best=%.0fpt; want >= 250pt beside 22em float", best)
	}
}
