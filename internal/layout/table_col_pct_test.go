package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestTableColumnWidthPercent(t *testing.T) {
	s := sheet(t, `
table { width: 400pt; border-collapse: collapse; }
td, th { border: 1px solid #000; padding: 2pt; font-size: 10pt; text-align: left; }
`)

	root, err := html.Parse(`<html><body><table>
<tr><th style="width:25%">A</th><th style="width:75%">B</th></tr>
<tr><td style="width:25%">short</td><td style="width:75%">MARKER</td></tr>
</table></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := Layout(root, Options{
		Width: 500, Height: 400, Sheets: []*css.Stylesheet{s},
		Media: "print", Background: true, DebugBoxes: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var shortX, markerX float64

	var cells [][4]float64 // x,y,w,h of stroke rects that look like cells

	for _, op := range res.Ops {
		if op.Kind == OpText {
			if strings.Contains(op.Text, "short") {
				shortX = op.X
			}

			if strings.Contains(op.Text, "MARKER") {
				markerX = op.X
			}
		}

		if op.Kind == OpStrokeRect && op.H > 5 && op.W > 20 {
			cells = append(cells, [4]float64{op.X, op.Y, op.W, op.H})
		}
	}

	t.Logf("short=%.0f MARKER=%.0f cells=%v", shortX, markerX, cells)
	gap := markerX - shortX
	// 25% of 400 = 100pt column; MARKER should start near x≈100.
	if gap < 80 || gap > 130 {
		t.Fatalf("column gap short→MARKER = %.0f, want ~100pt (25/75 of 400pt)", gap)
	}
}
