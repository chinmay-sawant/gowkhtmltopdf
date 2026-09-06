package layout_test

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

func TestTableCellClipsTransformedContent(t *testing.T) {
	t.Parallel()

	sheet, err := css.Parse(`
table { border-collapse: collapse; table-layout: fixed; width: 300pt; }
td { border: 1px solid #000; padding: 4pt; vertical-align: top; }
td.left { width: 150pt; }
td.right { width: 150pt; background: #eef; }
.rot { -webkit-transform: rotate(8deg); background: #fd8; padding: 6px; display: inline-block; }
`)
	if err != nil {
		t.Fatal(err)
	}

	root, err := html.Parse(`<html><body><table><tr>
<td class="left">desc</td>
<td class="right"><div class="rot">T</div></td>
</tr></table></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	res, err := layout.Layout(root, layout.Options{ //nolint:exhaustruct
		Width: 400, Height: 200, Background: true, Sheets: []*css.Stylesheet{sheet},
	})
	if err != nil {
		t.Fatal(err)
	}

	rightCellLeft, ok := findRightCellLeft(res)
	if !ok {
		t.Fatal("right cell background fill not found")
	}

	assertTransformedOpsClipped(t, res, rightCellLeft)
}

func findRightCellLeft(res *layout.Result) (float64, bool) {
	var (
		rightCellLeft float64
		sawRight      bool
	)

	for _, paintOp := range res.Ops {
		// right cell background (#eef ≈ high B)
		if !isRightCellFill(paintOp) {
			continue
		}

		if !sawRight || paintOp.X > rightCellLeft {
			rightCellLeft = paintOp.X
			sawRight = true
		}
	}

	return rightCellLeft, sawRight
}

func isRightCellFill(paintOp layout.Op) bool {
	return paintOp.Kind == layout.OpFillRect && paintOp.B > 0.9 && paintOp.W > 80 && paintOp.X > 50
}

func assertTransformedOpsClipped(t *testing.T, res *layout.Result, rightCellLeft float64) {
	t.Helper()

	sawXform := false

	for _, paintOp := range res.Ops {
		if !paintOp.XformSet {
			continue
		}

		sawXform = true

		if paintOp.X+0.05 < rightCellLeft {
			t.Fatalf("transformed op X=%.2f spills left of cell X=%.2f", paintOp.X, rightCellLeft)
		}
	}

	if !sawXform {
		t.Fatal("expected transformed ops")
	}
}
