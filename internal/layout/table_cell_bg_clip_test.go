package layout_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/layout"
)

// TestTableCellBackgroundImageClipped: a cover-sized background inside a td
// must not paint into the neighboring cell (fixture-60 page-3 spill).
//
//nolint:cyclop,funlen // cell background clip assertion keeps full table layout check
func TestTableCellBackgroundImageClipped(t *testing.T) {
	t.Parallel()

	sheet, err := css.Parse(`
table { width: 300pt; table-layout: fixed; border-collapse: collapse; }
td { padding: 4pt; border: 1px solid #000; vertical-align: top; }
td.a { width: 50%; }
td.b { width: 50%; }
.demo {
  height: 40pt;
  background-image: url("logo.png");
  background-size: cover;
  background-repeat: no-repeat;
}
`)
	if err != nil {
		t.Fatal(err)
	}

	root, err := html.Parse(`<html><body><table><tr>
<td class="a"><div class="demo"></div></td>
<td class="b">RIGHT</td>
</tr></table></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	rootDir, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}

	logoPath := filepath.Join(rootDir, "testdata/golden/logo.png")

	res, err := layout.Layout(root, layout.Options{ //nolint:exhaustruct
		Width: 400, Height: 200, Background: true, Media: "print",
		Sheets: []*css.Stylesheet{sheet},
		Images: func(src string) ([]byte, error) {
			src = strings.TrimPrefix(src, "file://")
			if !filepath.IsAbs(src) {
				src = filepath.Join(filepath.Dir(logoPath), filepath.Base(src))
			}

			return os.ReadFile(src)
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	var rightX float64

	for _, paintOp := range res.Ops {
		if paintOp.Kind == layout.OpText && strings.Contains(paintOp.Text, "RIGHT") {
			rightX = paintOp.X

			break
		}
	}

	if rightX < 100 {
		t.Fatalf("RIGHT text x=%.1f, want in second column", rightX)
	}

	for _, paintOp := range res.Ops {
		if paintOp.Kind != layout.OpImage || !paintOp.IsBackground {
			continue
		}

		if paintOp.X+paintOp.W > rightX-1 {
			t.Fatalf("background image spills into RIGHT column: img=[%.1f..%.1f] rightTextX=%.1f",
				paintOp.X, paintOp.X+paintOp.W, rightX)
		}

		if paintOp.X+paintOp.W > 160 { // mid table ~150
			// soft check: image should stay in first half
			t.Logf("img x=%.1f w=%.1f rightX=%.1f", paintOp.X, paintOp.W, rightX)
		}
	}
}
