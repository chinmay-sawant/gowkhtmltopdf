package imageout

import (
	"encoding/base64"
	"testing"
)

func TestDebugDataURIRegion(t *testing.T) {
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

	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y += 2 {
		row := make([]byte, 0, img.Bounds().Dx())
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x += 2 {
			c := asNRGBA(img.At(x, y))
			if c.R > 200 && c.G < 60 && c.B < 60 && c.A > 200 {
				row = append(row, 'R')
			} else if c.A == 0 {
				row = append(row, '.')
			} else {
				row = append(row, '-')
			}
		}
		t.Logf("y=%3d %s", y, row)
	}
}
