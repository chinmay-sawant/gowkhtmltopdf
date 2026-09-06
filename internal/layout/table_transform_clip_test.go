package layout

import (
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
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
	res, err := Layout(root, Options{ //nolint:exhaustruct
		Width: 400, Height: 200, Background: true, Sheets: []*css.Stylesheet{sheet},
	})
	if err != nil {
		t.Fatal(err)
	}

	var rightCellLeft float64
	var sawRight bool
	for _, op := range res.Ops {
		// right cell background (#eef ≈ high B)
		if op.Kind == OpFillRect && op.B > 0.9 && op.W > 80 && op.X > 50 {
			if !sawRight || op.X > rightCellLeft {
				rightCellLeft = op.X
				sawRight = true
			}
		}
	}
	if !sawRight {
		t.Fatal("right cell background fill not found")
	}

	sawXform := false
	for _, op := range res.Ops {
		if !op.XformSet {
			continue
		}
		sawXform = true
		if op.X+0.05 < rightCellLeft {
			t.Fatalf("transformed op X=%.2f spills left of cell X=%.2f", op.X, rightCellLeft)
		}
	}
	if !sawXform {
		t.Fatal("expected transformed ops")
	}
}
