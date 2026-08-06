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
	s := sheet(t, `
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
	styles := resolveStyles(root, []*css.Stylesheet{s}, "print", contentW, 800)
	var p *html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == "p" && p == nil {
			p = n
		}
		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)
	if st := styles[p]; st.Width >= 0 && st.Width < 200 {
		t.Fatalf("p width=%.1f leaked from p::before", st.Width)
	}

	res, err := Layout(root, Options{
		Width: contentW, Height: 800, Sheets: []*css.Stylesheet{s},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	lines := map[int][2]float64{}
	for _, op := range res.Ops {
		if op.Kind != OpText || op.X > contentW*0.55 {
			continue
		}
		yi := int(op.Y + 0.5)
		ln := lines[yi]
		if ln[1] == 0 && ln[0] == 0 {
			ln = [2]float64{op.X, op.X + op.W}
		} else {
			if op.X < ln[0] {
				ln[0] = op.X
			}
			if op.X+op.W > ln[1] {
				ln[1] = op.X + op.W
			}
		}
		lines[yi] = ln
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
