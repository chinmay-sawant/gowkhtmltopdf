//nolint:testpackage // white-box test shares unexported helpers (redPNG, renderHTMLOpts) with imageout_test.go
package imageout

import (
	"encoding/base64"
	"testing"
)

func TestDebugDataURIRegion(t *testing.T) {
	t.Parallel()
	raw := redPNG(t, 16, 16)
	src := `<html><body><img src="data:image/png;base64,` +
		base64.StdEncoding.EncodeToString(raw) + `"></body></html>`

	img, err := renderHTMLOpts(src, RenderOptions{ //nolint:exhaustruct // intentional zero/partial fields
		Images: dataURIImages,
	})
	if err != nil {
		t.Fatalf("Render: %v", err)
	}

	t.Logf("image bounds: %v", img.Bounds())

	for row := img.Bounds().Min.Y; row < img.Bounds().Max.Y; row += 2 {
		line := make([]byte, 0, img.Bounds().Dx())

		for col := img.Bounds().Min.X; col < img.Bounds().Max.X; col += 2 {
			c := asNRGBA(img.At(col, row))

			switch {
			case c.R > 200 && c.G < 60 && c.B < 60 && c.A > 200:
				line = append(line, 'R')
			case c.A == 0:
				line = append(line, '.')
			default:
				line = append(line, '-')
			}
		}

		t.Logf("y=%3d %s", row, line)
	}
}
