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

func TestBackgroundRepeatXTiles(t *testing.T) {
	t.Parallel()

	sheet, err := css.Parse(`
.box {
  width: 200pt; height: 40pt;
  background-color: #eef;
  background-image: url("logo.png");
  background-repeat: repeat-x;
  background-size: 20pt 20pt;
}
`)
	if err != nil {
		t.Fatal(err)
	}

	root, err := html.Parse(`<html><body><div class="box"></div></body></html>`)
	if err != nil {
		t.Fatal(err)
	}

	rootDir, _ := filepath.Abs("../..")
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

	tileCount := 0

	for _, paintOp := range res.Ops {
		if paintOp.Kind == layout.OpImage && paintOp.IsBackground {
			tileCount++

			t.Logf("tile#%d x=%.1f y=%.1f w=%.1f h=%.1f", tileCount, paintOp.X, paintOp.Y, paintOp.W, paintOp.H)
		}
	}

	// 200pt / 20pt = 10 tiles expected
	if tileCount < 5 {
		t.Fatalf("repeat-x tiles=%d, want >= 5 (200pt box / 20pt tile)", tileCount)
	}
}
