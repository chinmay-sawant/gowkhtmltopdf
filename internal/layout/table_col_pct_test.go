//nolint:testpackage // tests exercise unexported package internals via shared helpers
package layout

import (
	"strings"
	"testing"

	"gowkhtmltopdf/internal/css"
	"gowkhtmltopdf/internal/html"
)

func TestTableColumnWidthPercent(t *testing.T) { //nolint:cyclop
	t.Parallel()

	cssSheet := sheet(t, `
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

	res, err := Layout(root, Options{ //nolint:exhaustruct // intentional zero fields
		Width: 500, Height: 400, Sheets: []*css.Stylesheet{cssSheet},
		Media: "print", Background: true, DebugBoxes: true,
	})
	if err != nil {
		t.Fatal(err)
	}

	var shortX, markerX float64

	var cells [][4]float64 // x,y,w,h of stroke rects that look like cells

	for _, paintOp := range res.Ops {
		if paintOp.Kind == OpText {
			if strings.Contains(paintOp.Text, "short") {
				shortX = paintOp.X
			}

			if strings.Contains(paintOp.Text, "MARKER") {
				markerX = paintOp.X
			}
		}

		if paintOp.Kind == OpStrokeRect && paintOp.H > 5 && paintOp.W > 20 {
			cells = append(cells, [4]float64{paintOp.X, paintOp.Y, paintOp.W, paintOp.H})
		}
	}

	t.Logf("short=%.0f MARKER=%.0f cells=%v", shortX, markerX, cells)
	gap := markerX - shortX
	// 25% of 400 = 100pt column; MARKER should start near x≈100.
	if gap < 80 || gap > 130 {
		t.Fatalf("column gap short→MARKER = %.0f, want ~100pt (25/75 of 400pt)", gap)
	}
}
