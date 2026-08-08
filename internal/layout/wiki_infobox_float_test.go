//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"math"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestInfoboxResolvedWidth22em(t *testing.T) {
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
@media (min-width: 640px) {
  .infobox { margin-left: 1em; float: right; clear: right; width: 22em; font-size: 88%; }
}
@media (max-width: 640px) {
  .infobox { width: 100%; }
}
`)

	root, err := html.Parse(`<html><body><table class="infobox"><tr><td>x</td></tr></table></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	styles := resolveStyles(root, []*css.Stylesheet{cssSheet}, "print", 538, 800)

	var table *html.Node

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Name == displayTable {
			table = n
		}

		for _, c := range n.Children {
			walk(c)
		}
	}
	walk(root)

	sty := styles[table]
	t.Logf("float=%q width=%.2f width%%=%.2f fontSize=%.2f display=%q",
		sty.Float, sty.Width, sty.WidthPercent, sty.FontSize, sty.Display)

	if sty.Float != floatRight {
		t.Fatalf("float=%q, want right (min-width:640px rule)", sty.Float)
	}

	if sty.WidthPercent >= 0 {
		t.Fatalf("width%%=%v set — max-width:640px rule wrongly won", sty.WidthPercent)
	}

	if sty.Width < 150 || sty.Width > 280 {
		t.Fatalf("width=%.2fpt, want ~22em (~200-240pt)", sty.Width)
	}
}

// TestWikiInfoboxFloatLeavesReadableTextColumn reproduces the Ana/wiki print
// layout bug: a floated .infobox { width:22em } must leave a usable text
// column (Chrome ~360pt on A4), not a ~120pt ribbon.
func TestWikiInfoboxFloatLeavesReadableTextColumn(t *testing.T) { //nolint:gocognit,cyclop,funlen
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
@media (min-width: 640px) {
  .infobox { margin-left: 1em; float: right; clear: right; width: 22em; font-size: 88%; }
}
@media (max-width: 640px) {
  .infobox { width: 100%; }
}
p { font-size: 12pt; line-height: 16pt; text-align: left; }
`)

	const contentW = 538.0 // ~A4 content width in pt

	htmlSrc := `<html><body>
<table class="infobox"><tr><td>Portrait</td></tr>
<tr><th>Born</th><td>30 April 1988 Havana</td></tr>
<tr><th>Citizenship</th><td>Cuba Spain United States</td></tr>
</table>
<p>Ana Celia de Armas Caso is a Cuban-born actress holding Cuban Spanish and American citizenship in ` +
		`Hollywood films.</p>
<p>She began her career in Cuba with a leading role in the romantic drama ` +
		`Mona and later moved to Spain and Hollywood studios.</p>
</body></html>`

	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: contentW, Height: 800, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Background: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var floatRight float64

	for _, op := range res.Ops {
		if op.Kind == OpText && (op.Text == "Portrait" || op.Text == "Born") {
			if op.X > floatRight {
				floatRight = op.X
			}
		}
	}

	if floatRight < contentW*0.4 {
		t.Fatalf("infobox text starts at x=%.0f, expected toward right (>= %.0f)", floatRight, contentW*0.4)
	}

	var bodyWidths []float64

	seenY := map[int]bool{}

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || paintOp.X > contentW*0.45 || len(paintOp.Text) < 4 {
			continue
		}

		yi := int(paintOp.Y + 0.5)
		if seenY[yi] {
			continue
		}

		seenY[yi] = true
		xStart, xEnd := paintOp.X, paintOp.X+paintOp.W

		for _, otherOp := range res.Ops {
			if otherOp.Kind != OpText || math.Abs(otherOp.Y-paintOp.Y) > 0.5 || otherOp.X > contentW*0.45 {
				continue
			}

			if otherOp.X < xStart {
				xStart = otherOp.X
			}

			if otherOp.X+otherOp.W > xEnd {
				xEnd = otherOp.X + otherOp.W
			}
		}

		w := xEnd - xStart
		if w > 40 {
			bodyWidths = append(bodyWidths, w)
		}
	}

	if len(bodyWidths) == 0 {
		t.Fatal("no body text lines found left of infobox")
	}

	var sum, best float64
	for _, w := range bodyWidths {
		sum += w

		if w > best {
			best = w
		}
	}

	avg := sum / float64(len(bodyWidths))
	t.Logf("body line widths: best=%.0f avg=%.0f n=%d floatTextX=%.0f", best, avg, len(bodyWidths), floatRight)

	if best < 250 {
		t.Fatalf("widest body line = %.0fpt, want >= 250pt beside 22em float", best)
	}

	if avg < 180 {
		t.Fatalf("average body line = %.0fpt, want >= 180pt (premature wraps)", avg)
	}
}

func TestWikiInfoboxFloatWithWideImage(t *testing.T) { //nolint:cyclop,funlen
	t.Parallel()

	cssSheet := sheet(t, `
body { margin: 0; font-size: 12pt; }
.infobox { float: right; clear: right; width: 22em; font-size: 88%; }
p { font-size: 12pt; line-height: 16pt; text-align: left; }
img { max-width: 100%; height: auto; }
`)

	const contentW = 538.0

	png := tinyPNG(400, 500) // wider than 22em in px terms
	htmlSrc := `<html><body>
<table class="infobox"><tr><td><img src="photo.png" width="400" height="500"></td></tr>
<tr><th>Born</th><td>30 April 1988</td></tr>
</table>
<p>Ana Celia de Armas Caso is a Cuban-born actress holding Cuban Spanish and American citizenship in ` +
		`Hollywood films today.</p>
</body></html>`

	root, err := html.Parse(htmlSrc)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: contentW, Height: 800, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Background: true,
		Images: func(_ string) ([]byte, error) { return png, nil },
	})
	if err != nil {
		t.Fatal(err)
	}

	var imgW float64

	for _, op := range res.Ops {
		if op.Kind == OpImage {
			imgW = op.W
			t.Logf("image op x=%.0f w=%.0f", op.X, op.W)
		}
	}
	// 22em at 10.56pt ≈ 232pt — image must not force float wider than that.
	if imgW > 250 {
		t.Fatalf("floated infobox image width=%.0fpt, want capped near 22em (~232pt)", imgW)
	}

	var best float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || paintOp.X > 250 {
			continue
		}

		if paintOp.W > best {
			best = paintOp.W
		}
	}
	// Also check line spans
	seen := map[int]float64{}

	for _, paintOp := range res.Ops {
		if paintOp.Kind != OpText || paintOp.X > contentW*0.45 {
			continue
		}

		yi := int(paintOp.Y + 0.5)
		r := paintOp.X + paintOp.W

		if r > seen[yi] {
			seen[yi] = r
		}
	}

	for _, r := range seen {
		if r > best {
			best = r
		}
	}

	t.Logf("best body extent=%.0f", best)

	if best < 250 {
		t.Fatalf("body text extent=%.0f, squeezed by oversized float image", best)
	}
}
