//nolint:all
package layout

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/chinmay-sawant/gowkhtmltopdf/internal/css"
)

func TestMastheadFlexImagesPaint(t *testing.T) {
	t.Parallel()
	logo, err := os.ReadFile(filepath.Join("..", "..", "testdata", "golden", "logo.png"))
	if err != nil {
		t.Fatal(err)
	}
	root := mustParse(t, `<html><body>
<header style="display:flex;align-items:center;gap:12px;">
  <img class="logo" src="logo.png" alt="logo" style="height:36px">
</header>
</body></html>`)
	res, err := Layout(root, Options{ //nolint:exhaustruct
		Width: 500, Height: 200, Sheets: []*css.Stylesheet{sheet(t, `body{margin:0}`)}, Media: "print",
		Images: func(src string) ([]byte, error) {
			if src == "logo.png" {
				return logo, nil
			}
			return nil, os.ErrNotExist
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	var imgs int
	for _, op := range res.Ops {
		if op.Kind == OpImage && len(op.Image) > 0 {
			imgs++
			t.Logf("OpImage at (%.1f,%.1f) %.1fx%.1f intrinsic %dx%d bytes=%d", op.X, op.Y, op.W, op.H, op.ImgW, op.ImgH, len(op.Image))
		}
	}
	if imgs < 1 {
		t.Fatalf("expected logo OpImage, got %d; total ops=%d", imgs, len(res.Ops))
	}
}
