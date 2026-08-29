package layout

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
	"github.com/chinmay-sawant/gowkhtmltopdf/internal/html"
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
	res, err := Layout(root, Options{ //nolint:exhaustruct
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
	n := 0
	var xs []float64
	for _, op := range res.Ops {
		if op.Kind == OpImage && op.IsBackground {
			n++
			xs = append(xs, op.X)
			t.Logf("tile#%d x=%.1f y=%.1f w=%.1f h=%.1f", n, op.X, op.Y, op.W, op.H)
		}
	}
	// 200pt / 20pt = 10 tiles expected
	if n < 5 {
		t.Fatalf("repeat-x tiles=%d, want >= 5 (200pt box / 20pt tile)", n)
	}
}
